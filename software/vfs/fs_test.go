package vfs

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "UPCASE", sanitizeName("upcase"))
	assert.Equal(t, "MYFILE", sanitizeName("my file"))
	assert.Equal(t, "LIMITSLE", sanitizeName("limitslengthtoeight"))
	assert.Equal(t, "RMVNUMR", sanitizeName("r3m0v35 num83r5"))
	assert.Equal(t, "", sanitizeName(""))
	assert.Equal(t, "ILUV", sanitizeName("I luv ĀḞÍ♥︎✨ :3"))
}

func TestCreate(t *testing.T) {
	fsys, err := Create()
	defer assert.NoError(t, fsys.Close())

	assert.NoError(t, err)
}

func secondsToFrames(min, sec int) uint {
	// seconds times 44.1KHz = samples
	// 6 samples * 2 channels in each 33 byte frame (after CIC)
	samples := ((min * 60) + sec) * 44100
	return uint(samples / 6)
}

func copyFile(srcpath, dstpath string) (err error) {
	r, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer r.Close() // ignore error: file was opened read-only.

	w, err := os.Create(dstpath)
	if err != nil {
		return err
	}

	defer func() {
		if c := w.Close(); err == nil {
			err = c
		}
	}()

	_, err = io.Copy(w, r)
	return err
}

var CHRONIC_TOWN = CD{
	Name: "R.E.M. - Chronic Town",
	Tracks: []Track{
		{
			Filename:     "Wolves, Lower",
			LengthFrames: secondsToFrames(4, 15),
		},
		{
			Filename:     "Gardening at Night",
			LengthFrames: secondsToFrames(3, 30),
		},
		{
			Filename:     "Carnival of Sorts (Box Cars)",
			LengthFrames: secondsToFrames(3, 52),
		},
		{
			Filename:     "1,000,000",
			LengthFrames: secondsToFrames(3, 6),
		},
		{
			Filename:     "Stumble",
			LengthFrames: secondsToFrames(5, 40),
		},
	},
}

func TestLoadCD(t *testing.T) {
	fsys, err := Create()
	assert.NoError(t, err)
	defer fsys.Close()

	err = fsys.LoadCD(CHRONIC_TOWN)
	assert.NoError(t, err)
}

func TestFileSize(t *testing.T) {
	cd := CD{
		Tracks: []Track{
			{
				Filename:     "Track 1",
				LengthFrames: 1337,
			},
		},
	}

	fsys, err := Create()
	assert.NoError(t, err)
	defer fsys.Close()

	err = fsys.LoadCD(cd)
	assert.NoError(t, err)

	t0, err := fsys.fs.OpenFile("/TRACK00.WAV", os.O_RDONLY)
	assert.NoError(t, err)
	defer t0.Close()

	fileInfo, err := fsys.fs.ReadDir("/")
	assert.NoError(t, err)

	found := false
	for _, fi := range fileInfo {
		if fi.Name() == "TRACK00.WAV" {
			found = true
			assert.Equal(t, int64(1337*6*2), fi.Size())
		}
	}
	assert.True(t, found)
}

func TestTrackRanges(t *testing.T) {
	fsys, err := Create()
	assert.NoError(t, err)
	defer fsys.Close()

	err = fsys.LoadCD(CHRONIC_TOWN)
	assert.NoError(t, err)

	trackRanges, err := fsys.TrackRanges()
	assert.NoError(t, err)

	assert.Equal(t, len(CHRONIC_TOWN.Tracks), len(trackRanges))

	for i, tr := range CHRONIC_TOWN.Tracks {
		// assert.Equal(t, i, trackRanges[i].Index)
		// assert.Equal(t, tr, trackRanges[i].Track)

		totalSize := 0
		for _, r := range trackRanges[i].DiskRanges {
			// fmt.Printf("%v\t%v\t%v\n", i, r.Offset, r.Length)
			totalSize += int(r.Length)
		}
		// sum of bytes should be divisible by sector size
		assert.Equal(t, 0, totalSize%SECTOR_SIZE)
		// total sector size should be longer than file size
		assert.GreaterOrEqual(t, totalSize, int(tr.LengthFrames*6*2))
	}
}
