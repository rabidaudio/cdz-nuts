package audiocd

// cgo wrapper for libcdparanoia

// #cgo LDFLAGS: -lcdda_interface -lcdda_paranoia
// #include <stdint.h>
// #include <stdlib.h>
// #include <cdda_interface.h>
// #include <cdda_paranoia.h>
//
// /* Calling C function pointers from Go is not supported,
//    but this is a workaround. See https://pkg.go.dev/cmd/cgo */
// typedef int (*set_speed_fn) (struct cdrom_drive *d, int speed);
// int bridge_set_speed(set_speed_fn f, struct cdrom_drive *d, int speed) {
//   return f(d, speed);
// }
import "C"
import (
	"fmt"
	"io"
	"log"
	"strings"
	"unsafe"
)

type LogMode int

const (
	LogModeSilent LogMode = 0
	LogModeStdErr LogMode = 1
	LogModeLogger LogMode = 2
)

const BytesPerSector = int32(C.CD_FRAMESIZE_RAW)

const FullSpeed = -1

type ParanoiaFlags int

const (
	PARANOIA_MODE_FULL    ParanoiaFlags = C.PARANOIA_MODE_FULL
	PARANOIA_MODE_DISABLE ParanoiaFlags = C.PARANOIA_MODE_DISABLE

	PARANOIA_MODE_VERIFY    ParanoiaFlags = C.PARANOIA_MODE_VERIFY
	PARANOIA_MODE_FRAGMENT  ParanoiaFlags = C.PARANOIA_MODE_FRAGMENT
	PARANOIA_MODE_OVERLAP   ParanoiaFlags = C.PARANOIA_MODE_OVERLAP
	PARANOIA_MODE_SCRATCH   ParanoiaFlags = C.PARANOIA_MODE_SCRATCH
	PARANOIA_MODE_REPAIR    ParanoiaFlags = C.PARANOIA_MODE_REPAIR
	PARANOIA_MODE_NEVERSKIP ParanoiaFlags = C.PARANOIA_MODE_NEVERSKIP
)

type CDRom struct {
	drive      unsafe.Pointer // *C.cdrom_drive
	paranoia   unsafe.Pointer // *C.cdrom_paranoia
	MaxRetries int
	LogMode    LogMode
	Logger     *log.Logger
}

// TODO: cdda_track_copyp,cdda_track_preemp,cdda_track_channels

// Version returns the libcdparanoia version string
func Version() string {
	return C.GoString(C.paranoia_version())
}

func OpenDefault() (*CDRom, error) {
	return OpenDefaultL(LogModeSilent, nil)
}

func OpenDefaultL(lm LogMode, logger *log.Logger) (*CDRom, error) {
	logLevel, logFlush := prepareLogs(lm, logger)
	var p *C.char
	d := C.cdda_find_a_cdrom(logLevel, &p)
	logFlush(unsafe.Pointer(p))
	return initDrive(d, lm, logger)
}

func OpenDevice(dev string) (*CDRom, error) {
	return OpenDeviceL(dev, LogModeSilent, nil)
}

func OpenDeviceL(dev string, lm LogMode, logger *log.Logger) (*CDRom, error) {
	str := C.CString(dev)
	defer C.free(unsafe.Pointer(str))

	logLevel, logFlush := prepareLogs(lm, logger)
	var p *C.char
	d := C.cdda_identify(str, logLevel, &p)
	logFlush(unsafe.Pointer(p))
	return initDrive(d, lm, logger)
}

func initDrive(drive *C.cdrom_drive, lm LogMode, logger *log.Logger) (*CDRom, error) {
	if drive == nil {
		return nil, ErrNoDrive
	}

	if err, ok := parseError(C.cdda_open(drive)); !ok {
		return nil, err
	}

	paranoia := C.paranoia_init(drive)

	cdr := CDRom{
		drive:      unsafe.Pointer(drive),
		paranoia:   unsafe.Pointer(paranoia),
		MaxRetries: 20,
		LogMode:    lm,
		Logger:     logger,
	}

	err := cdr.SetSpeed(FullSpeed)
	if err != nil {
		return nil, err
	}
	_, err = cdr.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	cdr.SetParanoiaFlags(PARANOIA_MODE_FULL)

	return &cdr, nil
}

// func (cdr *CDRom) cddaSectorGetTrack(i int) int {
// 	return int(C.cdda_sector_gettrack((*C.cdrom_drive)(cdr.drive), C.long(i)))
// }

func (cdr *CDRom) Model() string {
	return C.GoString((*C.cdrom_drive)(cdr.drive).drive_model)
}

func (cdr *CDRom) DriveType() DriveType {
	return DriveType(int((*C.cdrom_drive)(cdr.drive).drive_type))
}

func (cdr *CDRom) InterfaceType() InterfaceType {
	return InterfaceType(int((*C.cdrom_drive)(cdr.drive)._interface))
}

func (cdr *CDRom) SectorsPerRead() int {
	return int((*C.cdrom_drive)(cdr.drive).nsectors)
}

func (cdr *CDRom) SetSectorsPerRead(sectors int) {
	(*C.cdrom_drive)(cdr.drive).nsectors = C.int(sectors)
}

func (cdr *CDRom) TrackCount() int {
	return int((*C.cdrom_drive)(cdr.drive).tracks)
}

func (cdr *CDRom) FirstAudioSector() int32 {
	return int32((*C.cdrom_drive)(cdr.drive).audio_first_sector)
}

func (cdr *CDRom) TOC() []TOC {
	drive := (*C.cdrom_drive)(cdr.drive)
	ctoc := drive.disc_toc

	// NOTE: the end of the last track is the first sector
	// of the imaginary track after
	toc := make([]TOC, cdr.TrackCount()+1)
	audiolen := toc[len(toc)-1].StartSector
	for i := range toc {
		toc[i].Flags = Flag(ctoc[i].bFlags)
		toc[i].TrackNum = uint8(ctoc[i].bTrack)
		toc[i].StartSector = int32(ctoc[i].dwStartSector)
	}
	// compute lengths
	if len(toc) == 1 {
		toc[0].LengthSectors = audiolen
	} else {
		for i := range toc[1:] {
			toc[i].LengthSectors = toc[i+1].StartSector - toc[i].StartSector
		}
	}

	return toc[:cdr.TrackCount()]
}

func (cdr *CDRom) LengthSectors() int32 {
	ctoc := (*C.cdrom_drive)(cdr.drive).disc_toc
	return int32(ctoc[cdr.TrackCount()].dwStartSector)
}

func (cdr *CDRom) IsOpen() bool {
	return int((*C.cdrom_drive)(cdr.drive).opened) != 0
}

func (cdr *CDRom) SetParanoiaFlags(flags ParanoiaFlags) {
	defer cdr.flushLogs()

	C.paranoia_modeset(cdr.paranoia, C.int(flags))
}

func (cdr *CDRom) ForceSearchOverlap(sectors int32) error {
	if sectors < 0 || sectors > 75 {
		return fmt.Errorf("audiocd: search overlap sectors must be 0 <= n <= 75")
	}
	defer cdr.flushLogs()

	C.paranoia_overlapset(cdr.paranoia, C.long(sectors))
	return nil
}

func (cdr *CDRom) SetSpeed(kbps int) error {
	defer cdr.flushLogs()
	drive := (*C.cdrom_drive)(cdr.drive)
	err, _ := parseError(C.bridge_set_speed(drive.set_speed, drive, C.int(kbps)))
	return err
}

func (cdr *CDRom) Seek(offset int64, whence int) (int64, error) {
	defer cdr.flushLogs()
	res := int64(C.paranoia_seek(cdr.paranoia, C.long(offset), C.int(whence)))
	if res < 0 {
		err := AudioCDError(-1 * res)
		return res, err
	}
	return res, nil
}

func (cdr *CDRom) Read(p []byte) (n int, err error) {
	if !cdr.IsOpen() {
		return 0, ErrNotOpen
	}
	if len(p) == 0 {
		return 0, nil
	}

	if int32(len(p))%BytesPerSector != 0 {
		return 0, fmt.Errorf("audiocd: must read complete sectors")
	}
	if int32(len(p)) > BytesPerSector {
		return cdr.Read(p[:BytesPerSector])
	}
	buf := unsafe.Pointer(C.paranoia_read_limited(cdr.paranoia, nil, C.int(cdr.MaxRetries)))
	// run logs and check for errors
	err = cdr.flushLogs()
	if err != nil {
		return 0, err
	}
	if buf == nil {
		return 0, fmt.Errorf("audiocd: unknown error")
	}

	res := C.GoBytes(buf, C.int(BytesPerSector))
	// copy data into provided buffer, since paranoia will reclaim buffer
	copy(p, res)
	return int(BytesPerSector), nil
}

func (cdr *CDRom) Close() error {
	if cdr.IsOpen() {
		C.cdda_close((*C.cdrom_drive)(cdr.drive))
	}
	if cdr.paranoia != nil {
		C.paranoia_free(cdr.paranoia)
	}
	// this doesn't seem to be necessary, and can cause double-free's
	// if cdr.drive != nil {
	// 	C.free(cdr.drive)
	// }
	return nil
}

var _ io.ReadSeeker = (*CDRom)(nil)

func parseError(retval C.int) (err error, ok bool) {
	if retval == 0 {
		return nil, true
	}
	i := int(retval)
	if i < 0 {
		i = -1 * i
	}
	return AudioCDError(i), false
}

func prepareLogs(lm LogMode, logger *log.Logger) (C.int, func(unsafe.Pointer)) {
	nopLogFlush := func(p unsafe.Pointer) {}
	switch lm {
	case LogModeStdErr:
		return C.int(LogModeStdErr), nopLogFlush
	case LogModeLogger:
		if logger != nil {
			return C.int(LogModeLogger), func(p unsafe.Pointer) {
				if logger != nil && p != nil {
					str := C.GoString((*C.char)(p))
					for line := range strings.Lines(str) {
						logger.Print(line)
					}
					C.free(p)
				}
			}
		}
	}
	return C.int(LogModeSilent), nopLogFlush
}

func (cdr *CDRom) flushLogs() (err error) {
	drive := (*C.cdrom_drive)(cdr.drive)

	errstring := C.cdda_errors(drive)
	if errstring != nil {
		err = fmt.Errorf("audiocd: %v", C.GoString(errstring))
	}

	logger := cdr.Logger
	if cdr.LogMode != LogModeLogger || logger == nil {
		return
	}

	if errstring != nil {
		for line := range strings.Lines(C.GoString(errstring)) {
			logger.Print(line)
		}
	}

	msgstring := C.cdda_messages(drive)
	if msgstring != nil {
		for line := range strings.Lines(C.GoString(msgstring)) {
			logger.Print(line)
		}
	}
	return
}
