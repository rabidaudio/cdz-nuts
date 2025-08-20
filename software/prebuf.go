package main

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rabidaudio/audiocd"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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

func (pb *PreBuffer) emptyChanToBuf(limit int) error {
	// take everything in the queue and load it into the buffer
	i := 0
	start := GetCPU()
FOR:
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
				i += 1
				if i >= limit {
					break FOR
				}
			}
		default:
			break FOR
		}
	}
	end := GetCPU()
	pb.showWithState("read: loaded % 4d sectors (of % 4d) to buf in %v us", i, i+len(pb.cbuf), float32(end-start)/1000)
	return nil
}

// Block until high water mark is reached, at which point we can begin reading
func (pb *PreBuffer) AwaitHighWaterMark() {
	for {
		if len(pb.cbuf) >= cap(pb.cbuf) {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	pb.emptyChanToBuf(-1)
}

func (pb *PreBuffer) Read(p []byte) (n int, err error) {
	// fill the buffer
	pb.emptyChanToBuf(len(p) / audiocd.BytesPerSector)
	if pb.buf.Len() == 0 {
		return 0, nil
	}
	return pb.buf.Read(p) // read from the buffer
}

func (pb *PreBuffer) Seek(offset int64, whence int) (int64, error) {
	// pause pipe, clear buffer, and seek
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	pb.emptyChanToBuf(-1)
	pb.buf.Truncate(0)

	return pb.src.Seek(offset, whence)
}

func (pb *PreBuffer) Pipe() error {
	if pb.pipeRunning {
		return fmt.Errorf("pipe already running")
	}
	pb.pipeRunning = true
	defer func() { pb.pipeRunning = false }()

	fmt.Printf("filling buffer\n")

	i := 0
	for {
		p := make([]byte, pb.chunkSize)

		start := GetCPU()
		pb.mtx.Lock()
		if pb.closed {
			pb.mtx.Unlock()
			return nil
		}
		n, err := pb.src.Read(p)
		pb.mtx.Unlock()
		end := GetCPU()
		if err != nil {
			return err
		}

		pb.cbuf <- p[:n] // load after unlock in case channel fills
		i += 1
		if i%1000 == 0 {
			pb.showWithState("pipe: read %d sectors in %v us", n, float32(end-start)/1000)
		}
	}
}

func (pb *PreBuffer) showWithState(format string, args ...any) {
	p := message.NewPrinter(language.English)
	p.Printf("[chan: % 3d\tbuf: % 8d]\t%v\n", len(pb.cbuf), pb.buf.Len()/audiocd.BytesPerSector, p.Sprintf(format, args...))
}

func GetCPU() int64 {
	// usage := new(syscall.Rusage)
	// syscall.Getrusage(syscall.RUSAGE_SELF, usage)
	// return usage.Utime.Nano() + usage.Stime.Nano()
	return time.Now().UnixNano()
}

func (pb *PreBuffer) Close() error {
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	pb.closed = true // interrupt fill
	return nil
}
