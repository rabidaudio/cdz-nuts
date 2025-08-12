package cdparanoia

// cgo wrapper for libcdparanoia

// #cgo LDFLAGS: -lcdda_interface -lcdda_paranoia
// #include <stdint.h>
// #include <stdlib.h>
// #include <cdda_interface.h>
// #include <cdda_paranoia.h>
import "C"
import (
	"fmt"
	"math"
	"unsafe"
)

var EnableLogs = false

var ErrNoDrive = fmt.Errorf("cdparanoia: no cd drive detected")

type ParanoiaError int

const (
	ErrSetReadAudioMode      ParanoiaError = 1
	ErrReadTOCLeadOut        ParanoiaError = 2
	ErrIllegalNumberOfTracks ParanoiaError = 3
	ErrReadTOCHeader         ParanoiaError = 4
	ErrReadTOCEntry          ParanoiaError = 5
	ErrNoData                ParanoiaError = 6
	ErrUnknownReadError      ParanoiaError = 7
	ErrUnableToIdentifyModel ParanoiaError = 8
	ErrIllegalTOC            ParanoiaError = 9

	ErrInterfaceNotSupported ParanoiaError = 100
	ErrPermissionDenied      ParanoiaError = 102

	ErrKernelMemory ParanoiaError = 300

	ErrNotOpen               ParanoiaError = 400
	ErrInvalidTrackNumber    ParanoiaError = 401
	ErrNoAudioTracks         ParanoiaError = 403
	ErrNoMediumPresent       ParanoiaError = 404
	ErrOperationNotSupported ParanoiaError = 405
)

func (pe ParanoiaError) name() string {
	switch pe {
	case ErrSetReadAudioMode:
		return "unable to set CDROM to read audio mode"
	case ErrReadTOCLeadOut:
		return "unable to read table of contents lead-out"
	case ErrIllegalNumberOfTracks:
		return "cdrom reporting illegal number of tracks"
	case ErrReadTOCHeader:
		return "unable to read table of contents header"
	case ErrReadTOCEntry:
		return "unable to read table of contents entry"
	case ErrNoData:
		return "could not read any data from drive"
	case ErrUnknownReadError:
		return "unknown, unrecoverable error reading data"
	case ErrUnableToIdentifyModel:
		return "unable to identify CDROM model"
	case ErrIllegalTOC:
		return "cdrom reporting illegal table of contents"

	case ErrInterfaceNotSupported:
		return "interface not supported"
	case ErrPermissionDenied:
		return "permision denied on cdrom (ioctl) device"

	case ErrKernelMemory:
		return "kernel memory error"

	case ErrNotOpen:
		return "device not open"
	case ErrInvalidTrackNumber:
		return "invalid track number"
	case ErrNoAudioTracks:
		return "no audio tracks on disc"
	case ErrNoMediumPresent:
		return "no medium present"
	case ErrOperationNotSupported:
		return "option not supported by drive"
	default:
		return fmt.Sprintf("unknown error code: %v", int(pe))
	}
}

func (ParanoiaError) Error() string {
	return "cdparanoia: "
}

// Version returns the libcdparanoia version string
func Version() string {
	return C.GoString(C.paranoia_version())
}

type CDRom struct {
	drive    unsafe.Pointer // *C.cdrom_drive
	paranoia unsafe.Pointer // *C.cdrom_paranoia
	opened   bool
}

func logLevel() C.int {
	if EnableLogs {
		return C.CDDA_MESSAGE_PRINTIT
	}
	return C.CDDA_MESSAGE_FORGETIT
}

func parseError(retval int) (err error, ok bool) {
	if retval == 0 {
		return nil, true
	}
	return ParanoiaError(math.Abs(retval)), false
}

func Init() (*CDRom, error) {
	drive := C.cdda_find_a_cdrom(logLevel(), nil)
	if drive == nil {
		return nil, ErrNoDrive
	}
	paranoia := C.paranoia_init(drive)
	if err, ok := parseError(C.cdda_open(drive)); !ok {
		return err
	}
	return &CDRom{
		drive:    unsafe.Pointer(drive),
		paranoia: unsafe.Pointer(paranoia),
		opened:   false,
	}, nil
}

func (cdr *CDRom) Close() error {
	if cdr.opened {
		C.cdda_close((*C.cdrom_drive)(cdr.drive))
	}
	C.paranoia_free((*C.cdrom_paranoia)(cdr.paranoia))
	C.free(cdr.drive)

	return nil
}
