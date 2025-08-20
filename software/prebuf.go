package main

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/rabidaudio/audiocd"
)

// TODO: generalize to any io.Reader interface
type PreBuffer struct {
	cd          *audiocd.AudioCD
	cbuf        chan []byte
	buf         bytes.Buffer
	fillRunning bool
	closed      bool
	mtx         sync.Mutex
	hwm         time.Duration
	buffered    int64
	// seeked      bool
}

func NewPreBuffer(cd *audiocd.AudioCD, hwm time.Duration) *PreBuffer {
	hwmsectors := int(hwm.Seconds() * audiocd.SectorsPerSecond)

	pb := PreBuffer{
		cd:   cd,
		cbuf: make(chan []byte, hwmsectors),
		hwm:  hwm,
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
	hwmbytes := int64(pb.hwm.Seconds()*audiocd.SampleRate) * audiocd.Channels * audiocd.BytesPerSample
	for {
		pb.mtx.Lock()
		b := pb.buffered
		pb.mtx.Unlock()
		if b >= hwmbytes {
			return
		}
		pb.emptyChanToBuf()
		time.Sleep(1 * time.Millisecond)
	}
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
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	// clear buffer
	pb.emptyChanToBuf()
	pb.buf.Truncate(0)
	pb.buffered = 0

	// pb.seeked = true // notify the other thread that we've seeked
	return pb.cd.Seek(offset, whence)
}

func (pb *PreBuffer) Pipe() error {
	if pb.fillRunning {
		return fmt.Errorf("fill already running")
	}
	pb.fillRunning = true
	defer func() { pb.fillRunning = false }()

	// hwmbytes := int64(pb.hwm.Seconds()*audiocd.SampleRate) * audiocd.Channels * audiocd.BytesPerSample
	// lwmbytes := int64(lowwm.Seconds()*audiocd.SampleRate) * audiocd.Channels * audiocd.BytesPerSample

	// filling := true
	// sentReady := false
	fmt.Printf("filling buffer\n")

	// buffered := int64(0)
	for {
		pb.mtx.Lock()

		if pb.closed {
			pb.mtx.Unlock()
			return nil
		}

		// if pb.seeked {
		// 	pb.seeked = false
		// 	buffered = 0
		// 	// filling = true
		// 	continue
		// }

		p := make([]byte, audiocd.BytesPerSector)
		n, err := pb.cd.Read(p)
		if err != nil {
			pb.mtx.Unlock()
			return err
		}
		pb.cbuf <- p[n:]
		pb.buffered += int64(n)
		pb.mtx.Unlock()

		// if !sentReady && buffered >= hwmbytes {
		// 	ready <- true // report that we are ready to begin reading
		// 	sentReady = true
		// }

		// TODO: adjust disk speed as needed

		// if filling && buffered > hwmbytes {
		// 	fmt.Printf("high water mark reached, slowing drive\n")
		// 	filling = false
		// 	// slow down
		// 	// pb.cd.SetSpeed(2)
		// 	if !sentReady {
		// 		ready <- true // report that we are ready to begin reading
		// 		sentReady = true
		// 	}
		// } else if !filling && buffered < lwmbytes {
		// 	fmt.Printf("low water mark reached, speeding up\n")
		// 	// speed up
		// 	filling = true
		// 	// pb.cd.SetSpeed(audiocd.FullSpeed)
		// }
	}
}

// <-seek
// 		 wipe buffer
// <- close
// 		cleanup
// else
//		if fillingState
//			read fast
//			if bufferedSize > highWM  fillingState = false
//		else
//			read slow
//			if bufferedSize < lowWM 	fillingState = true

// func (pb *PreBuffer) Fill(hiwm, lowwm time.Duration, ready chan<- bool) error {
// 	if pb.fillRunning {
// 		return fmt.Errorf("fill already running")
// 	}
// 	pb.fillRunning = true
// 	defer func() { pb.fillRunning = false }()

// 	hwmbytes := int64(hiwm.Seconds()*audiocd.SampleRate) * audiocd.Channels * audiocd.BytesPerSample
// 	lwmbytes := int64(lowwm.Seconds()*audiocd.SampleRate) * audiocd.Channels * audiocd.BytesPerSample

// 	filling := true
// 	sentReady := false
// 	i := 0
// 	fmt.Printf("filling buffer\n")
// 	for {
// 		pb.mtx.Lock()

// 		if pb.closed {
// 			// closed
// 			pb.buf.Truncate(0)
// 			pb.mtx.Unlock()
// 			return nil
// 		}

// 		if pb.seeked {
// 			pb.seeked = false
// 			// start reading again from new position on next loop
// 			pb.buf.Truncate(0)
// 			filling = true
// 			continue
// 		}

// 		// keep filling
// 		err := pb.fillBuffer()
// 		if err != nil {
// 			return err
// 		}
// 		pb.mtx.Unlock()

// 		// TODO: better way?
// 		if int64(pb.buf.Len()) > hwmbytes {
// 			time.Sleep(1 * time.Second)
// 		}

// 		if filling && int64(pb.buf.Len()) > hwmbytes {
// 			fmt.Printf("high water mark reached, slowing drive\n")
// 			filling = false
// 			// slow down
// 			pb.cd.SetSpeed(2)
// 			if !sentReady {
// 				ready <- true // report that we are ready to begin reading
// 				sentReady = true
// 			}
// 		} else if !filling && int64(pb.buf.Len()) < int64(lwmbytes) {
// 			fmt.Printf("low water mark reached, speeding up\n")
// 			// speed up
// 			filling = true
// 			pb.cd.SetSpeed(audiocd.FullSpeed)
// 		}
// 	}
// }

func (pb *PreBuffer) Close() {
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	pb.closed = true // interrupt fill
}

// func (pb *PreBuffer) fillBuffer() error {
// 	if pb.sb == nil {
// 		pb.sb = make([]byte, audiocd.BytesPerSector)
// 	}
// 	n, err := pb.cd.Read(pb.sb)
// 	if err != nil {
// 		return err
// 	}
// 	pb.buf.Write(pb.sb[:n])
// 	return nil
// }

// func (s *cdStreamer) SeekTo(tracknum int, byteoffset int) error {
// 	track := s.cd.TOC()[tracknum-1]
// 	end := track.LengthSectors * audiocd.BytesPerSector
// 	if byteoffset < 0 || byteoffset >= end {
// 		return fmt.Errorf("seekto: %d out of bounds (track length %d bytes)", byteoffset, end)
// 	}
// 	dest := int64(track.StartSector*audiocd.BytesPerSector + byteoffset)
// 	_, err := s.Seek(dest, io.SeekStart)
// 	return err
// }
