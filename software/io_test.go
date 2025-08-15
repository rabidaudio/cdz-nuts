package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func failIfErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

const blockSize = 100

var fakeBlock []byte

func init() {
	fakeBlock = make([]byte, blockSize)
	for i := range fakeBlock {
		fakeBlock[i] = byte(i)
	}
	fakeBlock[0] = 0xDE
	fakeBlock[1] = 0xAD
	fakeBlock[2] = 0xBE
	fakeBlock[3] = 0xEF
}

type blockedReader struct {
	offset int64
}

func (r *blockedReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(p)%blockSize != 0 {
		return 0, fmt.Errorf("must read in blocks")
	}
	if len(p) > blockSize {
		n, err = r.Read(p[:blockSize])
		if err != nil {
			return
		}
		nn, err := r.Read(p[blockSize:])
		n += nn
		return n, err
	}
	r.offset += blockSize
	copy(p[:blockSize], fakeBlock)
	return blockSize, nil
}

func (r *blockedReader) Seek(offset int64, whence int) (int64, error) {
	if offset%blockSize != 0 {
		return 0, fmt.Errorf("must seek in blocks")
	}
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = 1000 + offset
	}
	return r.offset, nil
}

type BufferedReader struct {
	BlockReader io.ReadSeeker
	trueOffset  int64
	blockOffset int64
	buf         bytes.Buffer
}

func (b *BufferedReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	// if there's data available in the buffer, return just that
	if b.buf.Len() > 0 {
		n = len(p)
		if n > b.buf.Len() {
			n = b.buf.Len()
		}
		copy(p[:n], b.buf.Next(n))
		b.trueOffset += int64(n)
		nn, err := b.Read(p[n:])
		return n + nn, err
	}
	// otherwise load data into the buffer
	nblocks := (len(p) / blockSize) + 1
	err = b.loadNextBlocks(nblocks)
	if err != nil {
		return 0, err
	}
	// recurse to load said data from buffer
	return b.Read(p)
}

func (b *BufferedReader) loadNextBlocks(nblocks int) error {
	pp := make([]byte, nblocks*blockSize)
	n, err := b.BlockReader.Read(pp)
	b.blockOffset += int64(n)
	b.buf.Write(pp[:n])
	return err
}

func (b *BufferedReader) Seek(offset int64, whence int) (newoffset int64, err error) {
	switch whence {
	case io.SeekCurrent:
		newoffset = b.trueOffset + offset
	case io.SeekEnd:
		newoffset = 1000 + offset
	default:
		newoffset = offset
	}

	// nothing to do
	if newoffset == b.trueOffset {
		return
	}

	// can use data already in buffer
	if newoffset > b.trueOffset && newoffset < b.blockOffset {
		// empty the buffer up to current point
		_ = b.buf.Next(int(newoffset - b.trueOffset))
		b.trueOffset = newoffset
		return
	}

	// otherwise we're going to need to wipe and seek
	newblockoffset := newoffset - (newoffset % blockSize)
	b.blockOffset, err = b.BlockReader.Seek(newblockoffset, io.SeekStart)
	if err != nil {
		b.trueOffset = b.blockOffset
		return b.trueOffset, err
	}
	b.buf.Truncate(0)
	err = b.loadNextBlocks(1)
	b.trueOffset = b.blockOffset
	if err != nil {
		return b.trueOffset, err
	}
	// empty the buffer
	_ = b.buf.Next(int(newoffset - newblockoffset))
	b.trueOffset = newoffset
	return
}

func TestIO(t *testing.T) {
	dest := make([]byte, 1000)

	zeroOut := func() {
		for i := range 1000 {
			dest[i] = 0
		}
	}

	r := blockedReader{}
	b := BufferedReader{BlockReader: &r}

	sn, err := b.Seek(0, io.SeekStart)
	failIfErr(t, err)
	assert.Equal(t, int64(0), sn)

	// Read one block
	n, err := b.Read(dest[:blockSize])
	failIfErr(t, err)
	assert.Equal(t, blockSize, n)
	assert.Equal(t, fakeBlock, dest[:blockSize])

	// Read remaining blocks
	n, err = b.Read(dest[blockSize:])
	failIfErr(t, err)
	assert.Equal(t, 9*blockSize, n)
	assert.Equal(t, fakeBlock, dest[blockSize:2*blockSize])
	assert.Equal(t, fakeBlock, dest[9*blockSize:])

	// seek to block 5 and read 2
	sn, err = b.Seek(500, io.SeekStart)
	failIfErr(t, err)
	assert.Equal(t, int64(500), sn)

	assert.Equal(t, int64(500), b.trueOffset)
	assert.Equal(t, int64(600), b.blockOffset)
	assert.Equal(t, int64(600), r.offset)
	assert.Equal(t, 100, b.buf.Len())

	zeroOut()
	n, err = b.Read(dest[500:700])
	failIfErr(t, err)
	assert.Equal(t, 200, n)
	assert.Equal(t, fakeBlock, dest[500:600])
	assert.Equal(t, fakeBlock, dest[600:700])

	assert.Equal(t, int64(700), b.trueOffset)
	assert.Equal(t, int64(800), b.blockOffset)
	assert.Equal(t, int64(800), r.offset)

	// read a partial block
	zeroOut()
	n, err = b.Read(dest[700:750])
	failIfErr(t, err)
	assert.Equal(t, 50, n)
	assert.Equal(t, fakeBlock[:50], dest[700:750])

	assert.Equal(t, int64(750), b.trueOffset)
	assert.Equal(t, int64(800), b.blockOffset)
	assert.Equal(t, int64(800), r.offset)

	// read across the next block
	zeroOut()
	n, err = b.Read(dest[750:850])
	failIfErr(t, err)
	assert.Equal(t, 100, n)
	assert.Equal(t, fakeBlock[50:], dest[750:800])
	assert.Equal(t, fakeBlock[:50], dest[800:850])

	assert.Equal(t, int64(900), b.blockOffset)
	assert.Equal(t, int64(850), b.trueOffset)
	assert.Equal(t, int64(900), r.offset)

	// seek within buffer
	sn, err = b.Seek(25, io.SeekCurrent)
	failIfErr(t, err)
	assert.Equal(t, int64(875), sn)

	assert.Equal(t, int64(875), b.trueOffset)
	assert.Equal(t, int64(900), b.blockOffset)
	assert.Equal(t, int64(900), r.offset)

	zeroOut()
	n, err = b.Read(dest[875:925])
	failIfErr(t, err)
	assert.Equal(t, fakeBlock[75:], dest[875:900])
	assert.Equal(t, fakeBlock[:25], dest[900:925])

	// seek after partial read
	sn, err = b.Seek(225, io.SeekStart)
	failIfErr(t, err)
	assert.Equal(t, int64(225), sn)
	assert.Equal(t, int64(225), b.trueOffset)
	assert.Equal(t, int64(300), b.blockOffset)
	assert.Equal(t, int64(300), r.offset)

	zeroOut()
	n, err = b.Read(dest[225:475])
	failIfErr(t, err)
	assert.Equal(t, 250, n)
	assert.Equal(t, fakeBlock[25:], dest[225:300])
	assert.Equal(t, fakeBlock, dest[300:400])
	assert.Equal(t, fakeBlock[:75], dest[400:475])

	assert.Equal(t, int64(475), b.trueOffset)
	assert.Equal(t, int64(500), b.blockOffset)
	assert.Equal(t, int64(500), r.offset)

	// read the whole thing again
	zeroOut()
	_, err = b.Seek(0, io.SeekStart)
	failIfErr(t, err)
	_, err = b.Read(dest)
	failIfErr(t, err)
	for i := range 10 {
		assert.Equal(t, fakeBlock, dest[i*100:(i+1)*100])
	}

	// read random amounts at random offsets
	for range 1000 {
		read, _ := rand.Int(rand.Reader, big.NewInt(200))
		start, _ := rand.Int(rand.Reader, big.NewInt(800))
		_, err = b.Seek(start.Int64(), io.SeekStart)
		failIfErr(t, err)
		_, err = b.Read(dest[start.Int64() : start.Int64()+read.Int64()])
		failIfErr(t, err)
	}
	for i := range 10 {
		assert.Equal(t, fakeBlock, dest[i*100:(i+1)*100])
	}
}
