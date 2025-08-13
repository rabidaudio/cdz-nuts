package cdda

// #include <linux/major.h>
// #include <cdda_interface.h>
import "C"

type InterfaceType int

const (
	GENERIC_SCSI     InterfaceType = C.GENERIC_SCSI
	COOKED_IOCTL     InterfaceType = C.COOKED_IOCTL
	TEST_INTERFACE   InterfaceType = C.TEST_INTERFACE
	SGIO_SCSI        InterfaceType = C.SGIO_SCSI
	SGIO_SCSI_BUGGY1 InterfaceType = C.SGIO_SCSI_BUGGY1
)

type DriveType int

// from linux/major.h, with descriptions from scan_devices.c

const (
	IDE0_MAJOR DriveType = C.IDE0_MAJOR
	IDE1_MAJOR DriveType = C.IDE1_MAJOR
	IDE2_MAJOR DriveType = C.IDE2_MAJOR
	IDE3_MAJOR DriveType = C.IDE3_MAJOR
	IDE4_MAJOR DriveType = C.IDE4_MAJOR
	IDE5_MAJOR DriveType = C.IDE5_MAJOR
	IDE6_MAJOR DriveType = C.IDE6_MAJOR
	IDE7_MAJOR DriveType = C.IDE7_MAJOR
	IDE8_MAJOR DriveType = C.IDE8_MAJOR
	IDE9_MAJOR DriveType = C.IDE9_MAJOR

	CDU31A_CDROM_MAJOR DriveType = C.CDU31A_CDROM_MAJOR

	CDU535_CDROM_MAJOR DriveType = C.CDU535_CDROM_MAJOR

	MATSUSHITA_CDROM_MAJOR  DriveType = C.MATSUSHITA_CDROM_MAJOR
	MATSUSHITA_CDROM2_MAJOR DriveType = C.MATSUSHITA_CDROM2_MAJOR
	MATSUSHITA_CDROM3_MAJOR DriveType = C.MATSUSHITA_CDROM3_MAJOR
	MATSUSHITA_CDROM4_MAJOR DriveType = C.MATSUSHITA_CDROM4_MAJOR

	SANYO_CDROM_MAJOR DriveType = C.SANYO_CDROM_MAJOR

	MITSUMI_CDROM_MAJOR   DriveType = C.MITSUMI_CDROM_MAJOR
	MITSUMI_X_CDROM_MAJOR DriveType = C.MITSUMI_X_CDROM_MAJOR

	OPTICS_CDROM_MAJOR DriveType = C.OPTICS_CDROM_MAJOR

	AZTECH_CDROM_MAJOR DriveType = C.AZTECH_CDROM_MAJOR

	GOLDSTAR_CDROM_MAJOR DriveType = C.GOLDSTAR_CDROM_MAJOR

	CM206_CDROM_MAJOR DriveType = C.CM206_CDROM_MAJOR

	SCSI_CDROM_MAJOR   DriveType = C.SCSI_CDROM_MAJOR
	SCSI_GENERIC_MAJOR DriveType = C.SCSI_GENERIC_MAJOR
)

func (dt DriveType) String() string {
	switch dt {
	case IDE0_MAJOR, IDE1_MAJOR, IDE2_MAJOR, IDE3_MAJOR, IDE4_MAJOR, IDE5_MAJOR, IDE6_MAJOR, IDE7_MAJOR, IDE8_MAJOR, IDE9_MAJOR:
		return "ATAPI"

	case CDU31A_CDROM_MAJOR:
		return "Sony CDU31A or compatible"

	case CDU535_CDROM_MAJOR:
		return "Sony CDU535 or compatible"

	case MATSUSHITA_CDROM_MAJOR, MATSUSHITA_CDROM2_MAJOR, MATSUSHITA_CDROM3_MAJOR, MATSUSHITA_CDROM4_MAJOR:
		return "non-ATAPI IDE-style Matsushita/Panasonic CR-5xx or compatible"

	case SANYO_CDROM_MAJOR:
		return "Sanyo proprietary or compatible: NOT CDDA CAPABLE"

	case MITSUMI_CDROM_MAJOR, MITSUMI_X_CDROM_MAJOR:
		return "Mitsumi proprietary or compatible: NOT CDDA CAPABLE"

	case OPTICS_CDROM_MAJOR:
		return "Optics Dolphin or compatible: NOT CDDA CAPABLE"

	case AZTECH_CDROM_MAJOR:
		return "Aztech proprietary or compatible: NOT CDDA CAPABLE"

	case GOLDSTAR_CDROM_MAJOR:
		return "Goldstar proprietary: NOT CDDA CAPABLE"

	case CM206_CDROM_MAJOR:
		return "Philips/LMS CM206 proprietary: NOT CDDA CAPABLE"

	case SCSI_CDROM_MAJOR, SCSI_GENERIC_MAJOR:
		return "SCSI CDROM"

	default:
		return "unknown"
	}
}
