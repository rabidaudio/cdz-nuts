package cdparanoia

// cgo wrapper for libcdparanoia

// #cgo LDFLAGS: -lcdda_interface -lcdda_paranoia
// #include <stdint.h>
// #include <stdlib.h>
// #include <cdda_interface.h>
// #include <cdda_paranoia.h>
//
// /* Calling C function pointers from Go is not supported,
//    but this is a workaround. See https://pkg.go.dev/cmd/cgo */
// typedef int (*read_toc_fn) (struct cdrom_drive *d);
// int bridge_read_toc(read_toc_fn f, struct cdrom_drive *d) {
//   return f(d);
// }
import "C"
import (
	"unsafe"
)

// Set this to true to enable debugging
var EnableLogs = false

type CDRom struct {
	drive    unsafe.Pointer // *C.cdrom_drive
	paranoia unsafe.Pointer // *C.cdrom_paranoia
	opened   bool
}

// Version returns the libcdparanoia version string
func Version() string {
	return C.GoString(C.paranoia_version())
}

func OpenDevice(dev string) (*CDRom, error) {
	str := C.CString(dev)
	defer C.free(unsafe.Pointer(str))
	return initDrive(C.cdda_identify(str, logLevel(), nil))
}

func Open() (*CDRom, error) {
	return initDrive(C.cdda_find_a_cdrom(logLevel(), nil))
}

func initDrive(drive *C.cdrom_drive) (*CDRom, error) {
	if drive == nil {
		return nil, ErrNoDrive
	}
	paranoia := C.paranoia_init(drive)
	if err, ok := parseError(C.cdda_open(drive)); !ok {
		return nil, err
	}
	return &CDRom{
		drive:    unsafe.Pointer(drive),
		paranoia: unsafe.Pointer(paranoia),
		opened:   false,
	}, nil
}

func (cdr *CDRom) Model() string {
	return C.GoString((*C.cdrom_drive)(cdr.drive).drive_model)
}

func (cdr *CDRom) DriveType() DriveType {
	return DriveType(int((*C.cdrom_drive)(cdr.drive).drive_type))
}

func (cdr *CDRom) InterfaceType() InterfaceType {
	return InterfaceType(int((*C.cdrom_drive)(cdr.drive)._interface))
}

func (cdr *CDRom) SectorCount() int {
	return int((*C.cdrom_drive)(cdr.drive).nsectors)
}

func (cdr *CDRom) TrackCount() int {
	return int((*C.cdrom_drive)(cdr.drive).tracks)
}

func (cdr *CDRom) FirstAudioSector() int64 {
	return int64((*C.cdrom_drive)(cdr.drive).audio_first_sector)
}

func (cdr *CDRom) LastAudioSector() int64 {
	return int64((*C.cdrom_drive)(cdr.drive).audio_last_sector)
}

func (cdr *CDRom) TOC() ([]TOC, error) {
	drive := (*C.cdrom_drive)(cdr.drive)
	ctoc := drive.disc_toc

	toc := make([]TOC, cdr.TrackCount())
	for i := range toc {
		toc[i].Flags = Flag(ctoc[i].bFlags)
		toc[i].TrackNum = uint8(ctoc[i].bTrack)
		toc[i].StartSector = int32(ctoc[i].dwStartSector)
	}
	return toc, nil
}

// ReadAudio
// SetSpeed
// cd_extra ??

// TODO: any paranoia methods....

func (cdr *CDRom) Close() error {
	if cdr.opened {
		C.cdda_close((*C.cdrom_drive)(cdr.drive))
	}
	C.paranoia_free(cdr.paranoia)
	C.free(cdr.drive)

	return nil
}

func parseError(retval C.int) (err error, ok bool) {
	if retval == 0 {
		return nil, true
	}
	i := int(retval)
	if i < 0 {
		i = -1 * i
	}
	return ParanoiaError(i), false
}

func logLevel() C.int {
	if EnableLogs {
		return C.CDDA_MESSAGE_PRINTIT
	}
	return C.CDDA_MESSAGE_FORGETIT
}
