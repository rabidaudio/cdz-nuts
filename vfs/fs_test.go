package vfs

import (
	"io"
	"os"
	"testing"

	"github.com/rabidaudio/carcd-adapter/cd"
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

func guessLengthBytes(min, sec int) int64 {
	// seconds times 44.1KHz times 2 bytes per sample times 2 channels
	// return int64(((min * 60) + sec) * 44100 * 2 * 2)
	return 1337 // STOPSHIP
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

func TestLoadCD(t *testing.T) {
	cd := cd.CD{
		Name: "R.E.M. - Chronic Town",
		Tracks: []cd.Track{
			{
				Filename:    "Wolves, Lower",
				LengthBytes: guessLengthBytes(4, 15),
			},
			{
				Filename:    "Gardening at Night",
				LengthBytes: guessLengthBytes(3, 30),
			},
			{
				Filename:    "Carnival of Sorts (Box Cars)",
				LengthBytes: guessLengthBytes(3, 52),
			},
			{
				Filename:    "1,000,000",
				LengthBytes: guessLengthBytes(3, 6),
			},
			{
				Filename:    "Stumble",
				LengthBytes: guessLengthBytes(5, 40),
			},
		},
	}

	fsys, _ := Create()
	err := fsys.LoadCD(cd)
	assert.NoError(t, err)

	assert.NoError(t, copyFile(fsys.Path, "/Users/personal/projects/carcd-adapter/testresult.img")) // for inspection
}

func TestFileSize(t *testing.T) {
	cd := cd.CD{
		Tracks: []cd.Track{
			{
				Filename:    "Track 1",
				LengthBytes: 1337,
			},
		},
	}

	fsys, err := Create()
	assert.NoError(t, err)
	defer fsys.Close()

	err = fsys.LoadCD(cd)
	assert.NoError(t, err)

	t0, err := fsys.OpenFile("/TRACK00.WAV", os.O_RDONLY)
	assert.NoError(t, err)
	defer t0.Close()

	fileInfo, err := fsys.ReadDir("/")
	assert.NoError(t, err)

	found := false
	for _, fi := range fileInfo {
		if fi.Name() == "TRACK00.WAV" {
			found = true
			assert.Equal(t, cd.Tracks[0].LengthBytes, fi.Size())
		}
	}
	assert.True(t, found)
}
