package audiocd

// TODO: should we link statically instead??

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

const (
	pFull      = C.PARANOIA_MODE_FULL
	pDisable   = C.PARANOIA_MODE_DISABLE
	pVerify    = C.PARANOIA_MODE_VERIFY
	pFragment  = C.PARANOIA_MODE_FRAGMENT
	pOverlap   = C.PARANOIA_MODE_OVERLAP
	pScratch   = C.PARANOIA_MODE_SCRATCH
	pRepair    = C.PARANOIA_MODE_REPAIR
	pNeverSkip = C.PARANOIA_MODE_NEVERSKIP
)

func openDrive(cd *AudioCD) error {
	logLevel, logFlush := prepareLogs(cd.LogMode, cd.Logger)
	var p *C.char
	defer logFlush(unsafe.Pointer(p))

	var drive *C.cdrom_drive
	if cd.Device == "" {
		drive = C.cdda_find_a_cdrom(logLevel, &p)
	} else {
		str := C.CString(cd.Device)
		defer C.free(unsafe.Pointer(str))
		drive = C.cdda_identify(str, logLevel, &p)
	}

	if drive == nil {
		return ErrNoDrive
	}

	if err, ok := parseError(C.cdda_open(drive)); !ok {
		return err
	}
	cd.drive = unsafe.Pointer(drive)
	cd.paranoia = C.paranoia_init(drive)
	return nil
}

func model(drive unsafe.Pointer) string {
	return C.GoString((*C.cdrom_drive)(drive).drive_model)
}

func driveType(drive unsafe.Pointer) DriveType {
	return DriveType(int((*C.cdrom_drive)(drive).drive_type))
}

func interfaceType(drive unsafe.Pointer) InterfaceType {
	return InterfaceType(int((*C.cdrom_drive)(drive)._interface))
}

func trackCount(d unsafe.Pointer) int {
	return int((*C.cdrom_drive)(d).tracks)
}

func firstAudioSector(d unsafe.Pointer) int32 {
	return int32((*C.cdrom_drive)(d).audio_first_sector)
}

func toc(d unsafe.Pointer, ntracks int) []TrackPosition {
	drive := (*C.cdrom_drive)(d)
	ctoc := drive.disc_toc

	// NOTE: the end of the last track is the first sector
	// of the imaginary track after
	toc := make([]TrackPosition, ntracks+1)
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
	return toc[:ntracks]
}

func lengthSectors(d unsafe.Pointer) int32 {
	drive := (*C.cdrom_drive)(d)
	return int32(drive.disc_toc[int(drive.tracks)].dwStartSector)
}

func opened(d unsafe.Pointer) bool {
	return int((*C.cdrom_drive)(d).opened) != 0
}

func setParanoia(cd *AudioCD, flags ParanoiaFlags) {
	defer flushLogs(cd)
	C.paranoia_modeset(cd.paranoia, C.int(flags))
}

func overlapSet(cd *AudioCD, sectors int32) {
	defer flushLogs(cd)
	C.paranoia_overlapset(cd.paranoia, C.long(sectors))
}

func setSpeed(cd *AudioCD, x int) error {
	defer flushLogs(cd)
	drive := (*C.cdrom_drive)(cd.drive)
	err, _ := parseError(C.bridge_set_speed(drive.set_speed, drive, C.int(x)))
	return err
}

func seekSector(cd *AudioCD, sector int32) error {
	defer flushLogs(cd)

	res := int64(C.paranoia_seek(cd.paranoia, C.long(sector), C.int(io.SeekStart)))
	if res < 0 {
		return AudioCDError(-1 * res)
	}
	return nil
}

func readLimited(cd *AudioCD, p []byte, retries int) error {
	// TODO: expose callback? may not be possible
	buf := unsafe.Pointer(C.paranoia_read_limited(cd.paranoia, nil, C.int(retries)))
	// run logs and check for errors
	err := flushLogs(cd)
	if err != nil {
		return err
	}
	if buf == nil {
		return fmt.Errorf("audiocd: unknown error")
	}

	res := C.GoBytes(buf, C.int(BytesPerSector))
	// copy data into provided buffer, since paranoia will reclaim buffer
	copy(p, res)
	return nil
}

func closeDrive(d unsafe.Pointer) {
	C.cdda_close((*C.cdrom_drive)(d))
}

func paranoiaFree(p unsafe.Pointer) {
	C.paranoia_free(p)
}

func version() string {
	return C.GoString(C.paranoia_version())
}

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

func flushLogs(cd *AudioCD) (err error) {
	drive := (*C.cdrom_drive)(cd.drive)

	errstring := C.cdda_errors(drive)
	if errstring != nil {
		err = fmt.Errorf("audiocd: %v", C.GoString(errstring))
	}

	logger := cd.Logger
	if cd.LogMode != LogModeLogger || logger == nil {
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
