package main

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/rabidaudio/audiocd"
)

type PreBuffer struct {
	cd          *audiocd.AudioCD
	buf         bytes.Buffer
	fillRunning bool
	mtx         sync.Mutex
	sb          []byte
	closed      bool
	seeked      bool
	// seekc       chan int64
}

func NewPreBuffer(cd *audiocd.AudioCD) *PreBuffer {
	pb := PreBuffer{
		cd: cd,
		// seekc: make(chan int64),
	}
	return &pb
}

func (pb *PreBuffer) Read(p []byte) (n int, err error) {
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	if pb.buf.Len() == 0 {
		return 0, nil
	}
	return pb.buf.Read(p) // read from the buffer
}

func (pb *PreBuffer) Seek(offset int64, whence int) (int64, error) {
	// if pb.fillRunning {
	// 	// wait and interrupt feed
	// 	pb.seekc <- offset
	// }
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	pb.seeked = true
	return pb.cd.Seek(offset, whence)
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

func (pb *PreBuffer) Fill(hiwm, lowwm time.Duration, ready chan<- bool) error {
	if pb.fillRunning {
		return fmt.Errorf("fill already running")
	}
	pb.fillRunning = true
	defer func() { pb.fillRunning = false }()

	hwmbytes := int64(hiwm.Seconds()*audiocd.SampleRate) * audiocd.Channels * audiocd.BytesPerSample
	lwmbytes := int64(lowwm.Seconds()*audiocd.SampleRate) * audiocd.Channels * audiocd.BytesPerSample

	filling := true
	sentReady := false
	i := 0
	fmt.Printf("filling buffer\n")
	for {
		pb.mtx.Lock()

		if pb.closed {
			// closed
			pb.buf.Truncate(0)
			pb.mtx.Unlock()
			return nil
		}

		if pb.seeked {
			pb.seeked = false
			// start reading again from new position on next loop
			pb.buf.Truncate(0)
			filling = true
			continue
		}

		// keep filling
		err := pb.fillBuffer()
		if err != nil {
			return err
		}
		pb.mtx.Unlock()

		// TODO: better way?
		if int64(pb.buf.Len()) > hwmbytes {
			time.Sleep(1 * time.Second)
		}

		if filling && int64(pb.buf.Len()) > hwmbytes {
			fmt.Printf("high water mark reached, slowing drive\n")
			filling = false
			// slow down
			pb.cd.SetSpeed(2)
			if !sentReady {
				ready <- true // report that we are ready to begin reading
				sentReady = true
			}
		} else if !filling && int64(pb.buf.Len()) < int64(lwmbytes) {
			fmt.Printf("low water mark reached, speeding up\n")
			// speed up
			filling = true
			pb.cd.SetSpeed(audiocd.FullSpeed)
		}
	}
}

func (pb *PreBuffer) Close() {
	pb.mtx.Lock()
	defer pb.mtx.Unlock()

	pb.closed = true
}

func (pb *PreBuffer) fillBuffer() error {
	if pb.sb == nil {
		pb.sb = make([]byte, audiocd.BytesPerSector)
	}
	n, err := pb.cd.Read(pb.sb)
	if err != nil {
		return err
	}
	pb.buf.Write(pb.sb[:n])
	return nil
}

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
