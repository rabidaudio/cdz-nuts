package main

import (
	"io"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/rabidaudio/cdz-nuts/audiocd"
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
}

func NewStreamer(cd audiocd.AudioCD) (*cdStreamer, error) {
	err := cd.Open()
	if err != nil {
		return nil, err
	}
	err = cd.SetSpeed(1) // read at realtime speed
	if err != nil {
		return nil, err
	}
	return &cdStreamer{AudioCD: &cd}, nil
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

	extractFrame := func(p []byte) (l, r float64) {
		li := int16(p[0]) + int16(p[1])*(1<<8)
		ri := int16(p[2]) + int16(p[3])*(1<<8)
		return float64(li) / (1<<16 - 1), float64(ri) / (1<<16 - 1)
	}
	for i := range len(samples) {
		samples[i][0], samples[i][1] = extractFrame(buf[i*f : (i+1)*f])
	}

	return n / f, true
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

func (s *cdStreamer) Close() error {
	return s.AudioCD.Close()
}

var _ beep.StreamSeekCloser = (*cdStreamer)(nil)

func main() {
	err := speaker.Init(AudioCDFormat.SampleRate, AudioCDFormat.SampleRate.N(time.Second/10))
	if err != nil {
		panic(err)
	}

	cd, err := NewStreamer(audiocd.AudioCD{Device: "/dev/sr1", LogMode: audiocd.LogModeStdErr})
	if err != nil {
		panic(err)
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(cd, beep.Callback(func() {
		done <- true
	})))

	<-done
}
