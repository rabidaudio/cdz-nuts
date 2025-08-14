// Package audiocd allows reading PCM audio data from a CD-DA disk
// in the cd drive.
//
// It's a cgo wrapper for [CDParanoia], which means it only runs on Linux
// and requires libcdparanoia and headers to be installed, for example:
//
//	sudo apt install cdparanoia libcdparanoia-dev
//
// It also means it has really powerful error correction capabilities.
//
// [CDParanoia]: https://xiph.org/paranoia/index.html
package audiocd

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
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"unsafe"
)

// LogMode configures the destination for debug logs.
type LogMode int

const (
	LogModeSilent LogMode = 0 // disable logs
	LogModeStdErr LogMode = 1 // log to stderr
	LogModeLogger LogMode = 2 // log to the supplied log.Logger instance
)

// ParanoiaFlags enable specific error checking features.
type ParanoiaFlags int

const (
	ParanoiaModeFull    ParanoiaFlags = C.PARANOIA_MODE_FULL    // enable all error checking features
	ParanoiaModeDisable ParanoiaFlags = C.PARANOIA_MODE_DISABLE // disable all error checking features

	ParanoiaVerify    ParanoiaFlags = C.PARANOIA_MODE_VERIFY
	ParanoiaFragment  ParanoiaFlags = C.PARANOIA_MODE_FRAGMENT
	ParanoiaOverlap   ParanoiaFlags = C.PARANOIA_MODE_OVERLAP
	ParanoiaScratch   ParanoiaFlags = C.PARANOIA_MODE_SCRATCH
	ParanoiaRepair    ParanoiaFlags = C.PARANOIA_MODE_REPAIR
	ParanoiaNeverSkip ParanoiaFlags = C.PARANOIA_MODE_NEVERSKIP
)

// AudioCD reads data from a CD-DR format cd in the disk drive.
// If Device is specified, AudioCD will read from the specified block device.
// Otherwise it will try to read from the first detected disk drive device.
// An AudioCD must be [Open]ed before use. The zero value for AudioCD is ready to be opened.
//
// AudioCD implements [io.ReadSeekCloser].
//
// Debug logging can be enabled by specifying LogMode. For [LogModeLogger],
// supply a [log.Logger] instance to Logger.
type AudioCD struct {
	Device     string      // the path to the cdrom device, e.g. /dev/cdrom
	MaxRetries int         // number of repeated reads on failed sectors. Set to -1 to disable retries. If 0, the default of 20 will be used
	LogMode    LogMode     // direct the library logs
	Logger     *log.Logger // if LogMode == LogModeLogger, the log.Logger to use

	buf            bytes.Buffer
	bufferedOffset int64
	trueOffset     int64

	drive    unsafe.Pointer // *C.cdrom_drive
	paranoia unsafe.Pointer // *C.cdrom_paranoia
}

// ensure interface conformation
var _ io.ReadSeekCloser = (*AudioCD)(nil)

// FullSpeed can be passed to [SetSpeed] to run the drive at its fastest speed.
const FullSpeed = -1

// Open determines the properties of the drive and detects
// the audio cd. This method must be called before information
// about the drive and cd can be accessed and before data can
// be read from the disk.
//
// If one of [ErrReadTOCLeadOut], [ErrIllegalNumberOfTracks],
// [ErrReadTOCHeader], or [ErrReadTOCEntry] is returned,
// it's likely that no cd is in the drive or the cd is not
// an audio cd.
//
// Open this does not refer to controlling the drive tray.
func (cd *AudioCD) Open() error {
	if cd.IsOpen() {
		return nil
	}

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

	err := cd.SetSpeed(FullSpeed)
	if err != nil {
		return err
	}
	err = cd.seekSector(0)
	cd.bufferedOffset = 0
	cd.trueOffset = 0
	if err != nil {
		return err
	}
	cd.SetParanoiaMode(ParanoiaModeFull)
	cd.buf.Truncate(0)
	cd.buf.Grow(BytesPerSector)
	return nil
}

// TODO: cdda_track_copyp,cdda_track_preemp,cdda_track_channels

// func (cd *AudioCD) cddaSectorGetTrack(i int) int {
// 	return int(C.cdda_sector_gettrack((*C.cdrom_drive)(cd.drive), C.long(i)))
// }

// Model returns information about the cd drive's manufacturer and model number.
func (cd *AudioCD) Model() string {
	if !cd.IsOpen() {
		return ""
	}
	return C.GoString((*C.cdrom_drive)(cd.drive).drive_model)
}

func (cd *AudioCD) DriveType() DriveType {
	if !cd.IsOpen() {
		return -1
	}
	return DriveType(int((*C.cdrom_drive)(cd.drive).drive_type))
}

func (cd *AudioCD) InterfaceType() InterfaceType {
	if !cd.IsOpen() {
		return -1
	}
	return InterfaceType(int((*C.cdrom_drive)(cd.drive)._interface))
}

// func (cd *AudioCD) SectorsPerRead() int {
// 	return int((*C.cdrom_drive)(cd.drive).nsectors)
// }

// func (cd *AudioCD) SetSectorsPerRead(sectors int) {
// 	(*C.cdrom_drive)(cd.drive).nsectors = C.int(sectors)
// }

// TrackCount returns number of audio tracks on the disk.
// The CD-DA format supports a maximum of 99 tracks.
func (cd *AudioCD) TrackCount() int {
	if !cd.IsOpen() {
		return -1
	}
	return int((*C.cdrom_drive)(cd.drive).tracks)
}

// FirstAudioSector returns the sector index of the first track.
func (cd *AudioCD) FirstAudioSector() int32 {
	if !cd.IsOpen() {
		return -1
	}
	return int32((*C.cdrom_drive)(cd.drive).audio_first_sector)
}

// TOC returns the table of contents from the disk.
//
// The table of contents lists the tracks on the disk
// and the sector offsets they can be found at.
// It will have length of [TrackCount].
func (cd *AudioCD) TOC() []TrackPosition {
	if !cd.IsOpen() {
		return nil
	}
	drive := (*C.cdrom_drive)(cd.drive)
	ctoc := drive.disc_toc

	// NOTE: the end of the last track is the first sector
	// of the imaginary track after
	toc := make([]TrackPosition, cd.TrackCount()+1)
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

	return toc[:cd.TrackCount()]
}

// LengthSectors returns the total number of sectors on the disk
// with audio data. This is the sector after the last track.
func (cd *AudioCD) LengthSectors() int32 {
	if !cd.IsOpen() {
		return -1
	}
	ctoc := (*C.cdrom_drive)(cd.drive).disc_toc
	return int32(ctoc[cd.TrackCount()].dwStartSector)
}

// IsOpen reports whether the instance has been initialized
// and checked for audio tracks.
//
// IsOpen does not refer to the state of the drive tray.
func (cd *AudioCD) IsOpen() bool {
	if cd.drive == nil {
		return false
	}
	return int((*C.cdrom_drive)(cd.drive).opened) != 0
}

// SetParanoiaMode sets how "paranoid" audiocd will be about error
// checking and correcting. [ParanoiaModeFull] (the default)
// enables all the correction features. [ParanoiaModeDisable] (0)
// disables all checks. Individual checks can be enabled, e.g.
// ParanoiaRepair|ParanoiaNeverSkip.
func (cd *AudioCD) SetParanoiaMode(flags ParanoiaFlags) {
	defer cd.flushLogs()

	C.paranoia_modeset(cd.paranoia, C.int(flags))
}

// ForceSearchOverlap sets the minimum number of sectors to search
// when detecting overlap issues during data verification.
func (cd *AudioCD) ForceSearchOverlap(sectors int32) error {
	if !cd.IsOpen() {
		return os.ErrClosed
	}
	if sectors < 0 || sectors > 75 {
		return fmt.Errorf("audiocd: search overlap sectors must be 0 <= n <= 75")
	}
	defer cd.flushLogs()

	C.paranoia_overlapset(cd.paranoia, C.long(sectors))
	return nil
}

// SetSpeed sets the data read speed multiplier.
// 1x reads at real-time audio speed, 75 sectors/second.
// Use [FullSpeed] (the default) to read as fast as possible.
func (cd *AudioCD) SetSpeed(x int) error {
	if !cd.IsOpen() {
		return os.ErrClosed
	}

	defer cd.flushLogs()
	drive := (*C.cdrom_drive)(cd.drive)
	err, _ := parseError(C.bridge_set_speed(drive.set_speed, drive, C.int(x)))
	return err
}

// TODO: separate C-calling methods from Go methods

// Seek provides access to the cursor position for reading audio data.
// It allows seeking to arbitrary sub-sector byte offsets.
func (cd *AudioCD) Seek(offset int64, whence int) (int64, error) {
	var newoffset int64
	switch whence {
	case io.SeekCurrent:
		newoffset = cd.trueOffset + offset
	case io.SeekEnd:
		end := int64(cd.LengthSectors() * BytesPerSector)
		newoffset = end + offset
	default:
		newoffset = offset
	}

	if newoffset == cd.trueOffset {
		// nothing to do
		return cd.trueOffset, nil
	}

	if newoffset > cd.trueOffset && newoffset < cd.bufferedOffset {
		// can use data already in buffer
		_ = cd.buf.Next(int(newoffset - cd.trueOffset)) // empty the buffer up to current point
		cd.trueOffset = newoffset
		return cd.trueOffset, nil
	}

	// otherwise we're going to need to wipe buffer and seek
	cd.buf.Truncate(0) // wipe buffered data
	cd.trueOffset = cd.bufferedOffset
	secoffset := newoffset - (newoffset % BytesPerSector)
	err := cd.seekSector(int32(secoffset / BytesPerSector))
	if err != nil {
		cd.trueOffset = cd.bufferedOffset
		return cd.trueOffset, err
	}
	err = cd.bufferSectors(1)
	cd.trueOffset = cd.bufferedOffset
	if err != nil {
		return cd.trueOffset, err
	}
	// seek buffer ahead to sub-sector offset
	_ = cd.buf.Next(int(newoffset - secoffset))
	cd.trueOffset = newoffset
	return cd.trueOffset, nil
}

// SeekToSector seeks the cd to the specfied sector index.
// This is useful for going to the start of a track.
func (cd *AudioCD) SeekToSector(sector int32) (int64, error) {
	return cd.Seek(int64(sector)*BytesPerSector, io.SeekStart)
}

func (cd *AudioCD) seekSector(sector int32) error {
	if !cd.IsOpen() {
		return os.ErrClosed
	}

	defer cd.flushLogs()
	res := int64(C.paranoia_seek(cd.paranoia, C.long(sector), C.int(io.SeekStart)))
	if res < 0 {
		return AudioCDError(-1 * res)
	}
	return res, nil
}

// Read reads PCM audio data from the disk.
//
// Read only supports reading complete sectors, and will error
// for partial reads.
//
// PCM data is signed 16-bit samples. Data will be in host byte order,
// regardless of drive endianness.
func (cd *AudioCD) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	// if there's data available in the buffer, return just that
	if cd.buf.Len() > 0 {
		n = len(p)
		if n > cd.buf.Len() {
			n = cd.buf.Len()
		}
		copy(p[:n], cd.buf.Next(n))
		cd.trueOffset += int64(n)

		// if more was requested, continue reading
		nn, err := cd.Read(p[n:])
		return n + nn, err
	}

	// otherwise load data into the buffer
	nsectors := (len(p) / BytesPerSector) + 1
	err = cd.bufferSectors(nsectors)
	if err != nil {
		return 0, err
	}
	// recurse to load said data from buffer
	return cd.Read(p)
}

func (cd *AudioCD) readSectors(p []byte) (int64, error) {
	if !cd.IsOpen() {
		return 0, os.ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}

	// TODO: maintain a read-ahead buffer to allow sub-sector reads
	if int32(len(p))%BytesPerSector != 0 {
		return 0, fmt.Errorf("audiocd: must read complete sectors")
	}

	if int32(len(p)) > BytesPerSector {
		// read one sector
		n, err := cd.readSectors(p[:BytesPerSector])
		if err != nil {
			return n, err
		}
		// read remaining sectors
		nn, err := cd.readSectors(p[BytesPerSector:])
		return n + nn, err
	}

	// TODO: expose callback
	retries := cd.MaxRetries
	if retries < 0 {
		retries = 0 // disable
	} else if retries == 0 {
		retries = 20 // default value
	}
	buf := unsafe.Pointer(C.paranoia_read_limited(cd.paranoia, nil, C.int(retries)))
	// run logs and check for errors
	err := cd.flushLogs()
	if err != nil {
		return 0, err
	}
	if buf == nil {
		return 0, fmt.Errorf("audiocd: unknown error")
	}

	res := C.GoBytes(buf, C.int(BytesPerSector))
	// copy data into provided buffer, since paranoia will reclaim buffer
	copy(p, res)
	return BytesPerSector, nil
}

func (cd *AudioCD) bufferSectors(nsectors int) error {
	p := make([]byte, nsectors*BytesPerSector)
	n, err := cd.readSectors(p)
	cd.bufferedOffset += n
	cd.buf.Write(p[:n])
	return err
}

// Close releases access to the cd drive. Data can no longer be accessed
// unless [Open]ed again.
//
// Close this does not refer to controlling the drive tray.
func (cd *AudioCD) Close() error {
	if cd.IsOpen() {
		C.cdda_close((*C.cdrom_drive)(cd.drive))
	}
	if cd.paranoia != nil {
		C.paranoia_free(cd.paranoia)
	}
	// this doesn't seem to be necessary, and can cause double-free's
	// if cd.drive != nil {
	// 	C.free(cd.drive)
	// }
	cd.paranoia = nil
	cd.drive = nil
	cd.buf.Truncate(0)
	return nil
}

// Version returns the libcdparanoia version string.
func Version() string {
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

func (cd *AudioCD) flushLogs() (err error) {
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
