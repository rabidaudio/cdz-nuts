package cdparanoia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParanoiaVersion(t *testing.T) {
	assert.Equal(t, "10.2", Version())
}

func failIfErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

// NOTE: this test is setup for a specific test device
// with a specific CD plugged in and is not portable
func TestOldDriveInfo(t *testing.T) {
	drive, err := OpenDevice("/dev/sr1")
	failIfErr(t, err)
	defer drive.Close()

	assert.Equal(t, "MATSHITA UJDA775 DVD/CDRW 1.00 ", drive.Model())
	assert.Equal(t, SCSI_CDROM_MAJOR, drive.DriveType())
	assert.Equal(t, SGIO_SCSI, drive.InterfaceType())
	assert.Equal(t, 27, drive.SectorCount())
	assert.Equal(t, 5, drive.TrackCount())
	assert.Equal(t, SectorIndex(0), drive.FirstAudioSector())
	assert.Equal(t, SectorIndex(27), drive.LastAudioSector())

	toc, err := drive.TOC()
	failIfErr(t, err)

	assert.Equal(t, drive.TrackCount(), len(toc))

	assert.Equal(t, uint8(1), toc[0].TrackNum, "tracks are 1 indexed")
	assert.Equal(t, uint8(2), toc[1].TrackNum)
	assert.Equal(t, uint8(3), toc[2].TrackNum)
	assert.Equal(t, uint8(4), toc[3].TrackNum)
	assert.Equal(t, uint8(5), toc[4].TrackNum)

	for i := range toc {
		assert.True(t, toc[i].Flags.IsAudio())
	}

	assert.Equal(t, int32(0), toc[0].StartSector)
	assert.Equal(t, int32(44988), toc[4].StartSector)
}
