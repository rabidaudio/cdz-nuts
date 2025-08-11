package vfs

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/rabidaudio/carcd-adapter/cd"
	"github.com/stretchr/testify/assert"
)

var WAVECD = cd.CD{
	Name:   "The Tones",
	Tracks: []cd.Track{},
}

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func init() {
	entries := must(os.ReadDir("testdata"))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "wav") {
			continue
		}
		file := must(os.Open("testdata/" + entry.Name()))
		sizeBytesRaw := make([]byte, 4)
		must(file.ReadAt(sizeBytesRaw, 40))
		must(file.Seek(0, io.SeekStart))
		sizeBytes := binary.LittleEndian.Uint32(sizeBytesRaw)
		WAVECD.Tracks = append(WAVECD.Tracks, cd.Track{
			ReadSeeker:   file,
			Filename:     entry.Name(),
			LengthFrames: uint(sizeBytes) / 2 / 6,
		})
	}
}

func TestReader(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
		t.Log(buf.String())
	}()

	fsys, err := Create()
	assert.NoError(t, err)
	defer fsys.Close()

	err = fsys.LoadCD(WAVECD)
	assert.NoError(t, err)

	reader, err := fsys.Reader()
	assert.NoError(t, err)

	// Read the full length into a local file and inspect
	out, err := os.Create("testdata/out.img")
	assert.NoError(t, err)
	defer out.Close()

	n, err := io.Copy(out, reader)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, n, DISK_SIZE)
}

func TestReaderCompare(t *testing.T) {
	fsys, err := Create()
	assert.NoError(t, err)
	defer fsys.Close()

	err = fsys.LoadCD(WAVECD)
	assert.NoError(t, err)

	for i, tr := range WAVECD.Tracks {
		name, _ := trackPath(WAVECD, i)
		tf, err := fsys.fs.OpenFile(name, os.O_RDWR)
		assert.NoError(t, err)

		tf.Seek(0, io.SeekStart)
		_, err = io.Copy(tf, tr)
		assert.NoError(t, err)
	}

	out, err := os.Create("testdata/compare.img")
	assert.NoError(t, err)
	in, err := os.Open(fsys.Path)
	assert.NoError(t, err)
	_, err = io.Copy(out, in)
	assert.NoError(t, err)
}
