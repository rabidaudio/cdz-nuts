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
	"fmt"
	"io"
	"unsafe"
)

// Set this to true to enable debugging
var EnableLogs = false

// const SectorSizeData = int32(C.CD_FRAMESIZE)
const SectorSizeRaw = int32(C.CD_FRAMESIZE_RAW)

type ParanoiaFlags int

const (
	PARANOIA_MODE_FULL      ParanoiaFlags = C.PARANOIA_MODE_FULL
	PARANOIA_MODE_DISABLE   ParanoiaFlags = C.PARANOIA_MODE_DISABLE
	PARANOIA_MODE_VERIFY    ParanoiaFlags = C.PARANOIA_MODE_VERIFY
	PARANOIA_MODE_FRAGMENT  ParanoiaFlags = C.PARANOIA_MODE_FRAGMENT
	PARANOIA_MODE_OVERLAP   ParanoiaFlags = C.PARANOIA_MODE_OVERLAP
	PARANOIA_MODE_SCRATCH   ParanoiaFlags = C.PARANOIA_MODE_SCRATCH
	PARANOIA_MODE_REPAIR    ParanoiaFlags = C.PARANOIA_MODE_REPAIR
	PARANOIA_MODE_NEVERSKIP ParanoiaFlags = C.PARANOIA_MODE_NEVERSKIP
)

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

func (cdr *CDRom) FirstAudioSector() int32 {
	return int32((*C.cdrom_drive)(cdr.drive).audio_first_sector)
}

func (cdr *CDRom) LastAudioSector() int32 {
	return int32((*C.cdrom_drive)(cdr.drive).audio_last_sector)
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

func (cdr *CDRom) SetParanoiaFlags(flags ParanoiaFlags) {
	C.paranoia_modeset(cdr.paranoia, C.int(flags))
}

func (cdr *CDRom) ForceSearchOverlap(sectors int32) error {
	if sectors < 0 || sectors > 75 {
		return fmt.Errorf("cdparanoia: search overlap sectors must be 0 <= n <= 75")
	}
	C.paranoia_overlapset(cdr.paranoia, C.long(sectors))
	return nil
}

// paranoia_seek

// ReadAudio
// SetSpeed
// cd_extra ??

// TODO: any paranoia methods....

func (cdr *CDRom) Seek(offset int64, whence int) (int64, error) {
	res := int64(C.paranoia_seek(cdr.paranoia, C.long(offset), C.int(whence)))
	if res < 0 {
		err := ParanoiaError(-1 * res)
		return res, err
	}
	return res, nil
}

func (cdr *CDRom) Read(p []byte) (n int, err error) {
	if int32(len(p))%SectorSizeRaw != 0 {
		return 0, fmt.Errorf("cdparanoia: must read complete sectors")
	}
	if int32(len(p)) > SectorSizeRaw {
		return cdr.Read(p[:SectorSizeRaw])
	}
	buf := unsafe.Pointer(C.paranoia_read(cdr.paranoia, nil))
	if buf == nil {
		/// error from errno field
	}
	res := C.GoBytes(buf, C.int(SectorSizeRaw))
	// copy data into provided buffer, since paranoia will reclaim buffer
	copy(p, res)
	return int(SectorSizeRaw), nil
}

func (cdr *CDRom) Close() error {
	if cdr.opened {
		C.cdda_close((*C.cdrom_drive)(cdr.drive))
	}
	C.paranoia_free(cdr.paranoia)
	C.free(cdr.drive)

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
	return ParanoiaError(i), false
}

func logLevel() C.int {
	if EnableLogs {
		return C.CDDA_MESSAGE_PRINTIT
	}
	return C.CDDA_MESSAGE_FORGETIT
}
