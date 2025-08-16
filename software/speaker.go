package main

import (
	"io"
	"time"

	"github.com/faiface/beep"
	"github.com/rabidaudio/audiocd"
)

var AudioCDFormat = beep.Format{
	SampleRate:  audiocd.SampleRate,
	NumChannels: audiocd.Channels,
	Precision:   audiocd.BytesPerSample,
}

type cdStreamer struct {
	*audiocd.AudioCD
	err    error
	offset int
	pb     *PreBuffer
}

func NewStreamer(cd *audiocd.AudioCD) (*cdStreamer, error) {
	if !cd.IsOpen() {
		err := cd.Open()
		if err != nil {
			return nil, err
		}
	}

	pb := NewPreBuffer(cd)
	ready := make(chan bool, 1)
	go func() {
		err := pb.Fill(30*time.Second, 3*time.Second, ready)
		if err != nil {
			panic(err)
		}
	}()
	<-ready // wait until we're ready to play

	return &cdStreamer{AudioCD: cd, pb: pb}, nil
}

func (s *cdStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	f := audiocd.Channels * audiocd.BytesPerSample
	buf := make([]byte, len(samples)*f)
	for n < len(buf) {
		nn, err := s.AudioCD.Read(buf[n:])
		s.err = err
		n += nn
		if err != nil {
			return 0, false
		}
	}
	for i := range len(samples) {
		samples[i][0], samples[i][1] = extractFrame(buf[i*f : (i+1)*f])
	}
	return n / f, true
}

func extractFrame(p []byte) (l, r float64) {
	li := int16(p[0]) + int16(p[1])*(1<<8)
	ri := int16(p[2]) + int16(p[3])*(1<<8)
	return float64(li) / (1<<16 - 1), float64(ri) / (1<<16 - 1)
}

func (s *cdStreamer) Err() error {
	return s.err
}

func (s *cdStreamer) Len() int {
	return int(s.AudioCD.LengthSectors()) * audiocd.SamplesPerFrame * audiocd.Channels
}

func (s *cdStreamer) Position() int {
	return s.offset
}

func (s *cdStreamer) Seek(p int) error {
	// seek to the start of the sector
	_, err := s.AudioCD.Seek(int64(p*audiocd.BytesPerSample), io.SeekStart)
	return err
}

func (s *cdStreamer) CurrentTrack() int {
	sector := s.offset / audiocd.SamplesPerFrame / audiocd.Channels
	return s.AudioCD.TrackAtSector(sector)
}

func (s *cdStreamer) Prev() {
	t := s.CurrentTrack()
	if t > 1 {
		sec := s.AudioCD.TOC()[t-1].StartSector
		s.Seek(sec * audiocd.Channels * audiocd.SamplesPerFrame)
	}
}

func (s *cdStreamer) Next() {
	t := s.CurrentTrack()
	if t <= s.AudioCD.TrackCount() {
		sec := s.AudioCD.TOC()[t-1+1].StartSector
		s.Seek(sec * audiocd.Channels * audiocd.SamplesPerFrame)
	}
}

func (s *cdStreamer) Close() error {
	s.pb.Close()

	return s.AudioCD.Close()
}

var _ beep.StreamSeekCloser = (*cdStreamer)(nil)
