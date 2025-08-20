package main

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

// TODO: generalize to any io.Reader interface
type PreBuffer struct {
	src         io.ReadSeeker
	cbuf        chan []byte
	buf         bytes.Buffer
	pipeRunning bool
	closed      bool
	mtx         sync.Mutex
	hwm         int64
	chunkSize   int
}

var _ io.ReadSeekCloser = (*PreBuffer)(nil)

func NewPreBuffer(src io.ReadSeeker, chunkSize int, hwm int64) *PreBuffer {
	pb := PreBuffer{
		src:       src,
		cbuf:      make(chan []byte, hwm/int64(chunkSize)),
		hwm:       hwm,
		chunkSize: chunkSize,
	}
	return &pb
}

func (pb *PreBuffer) emptyChanToBuf() error {
	// take everything in the queue and load it into the buffer
	for {
		select {
		case p, ok := <-pb.cbuf:
			{
				if !ok {
					return nil
				}
				_, err := pb.buf.Write(p)
				if err != nil {
					return err
				}
			}
		default:
			return nil
		}
	}
}

// Block until high water mark is reached, at which point we can begin reading
func (pb *PreBuffer) AwaitHighWaterMark() {
	for {
		if len(pb.cbuf) >= cap(pb.cbuf) {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	pb.emptyChanToBuf()
}

func (pb *PreBuffer) Read(p []byte) (n int, err error) {
	// fill the buffer
	pb.emptyChanToBuf()
	if pb.buf.Len() == 0 {
		return 0, nil
	}
	return pb.buf.Read(p) // read from the buffer
}

func (pb *PreBuffer) Seek(offset int64, whence int) (int64, error) {
	// pause pipe, clear buffer, and seek
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	pb.emptyChanToBuf()
	pb.buf.Truncate(0)

	return pb.src.Seek(offset, whence)
}

func (pb *PreBuffer) Pipe() error {
	if pb.pipeRunning {
		return fmt.Errorf("fill already running")
	}
	pb.pipeRunning = true
	defer func() { pb.pipeRunning = false }()

	fmt.Printf("filling buffer\n")

	for {
		if pb.closed {
			return nil
		}

		p := make([]byte, pb.chunkSize)

		pb.mtx.Lock()
		n, err := pb.src.Read(p)
		pb.mtx.Unlock()
		if err != nil {
			return err
		}

		pb.cbuf <- p[:n] // load after unlock in case channel fills
	}
}

func (pb *PreBuffer) Close() error {
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	pb.closed = true // interrupt fill
	return nil
}
