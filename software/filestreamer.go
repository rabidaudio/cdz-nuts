package main

import (
	"bytes"
	"io"
	"os"

	"github.com/faiface/beep"
	"github.com/rabidaudio/audiocd"
)

type WaveFileStreamer struct {
	f         *os.File
	sizeBytes int64
	offset    int64
	err       error
	buf       bytes.Buffer
}

const WavHeaderSize = 44

func NewWaveStreamer(path string) (*WaveFileStreamer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	offset, err := f.Seek(WavHeaderSize, io.SeekStart) // skip headers
	if err != nil {
		return nil, err
	}
	return &WaveFileStreamer{
		f:         f,
		sizeBytes: stat.Size(),
		offset:    offset,
	}, nil
}

func (s *WaveFileStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	b := len(samples) * audiocd.Channels * audiocd.BytesPerSample

	r := io.LimitReader(s.f, int64(b))
	_, err := s.buf.ReadFrom(r)
	if err != nil {
		s.err = err
		return 0, false
	}
	n = 0
	f := make([]byte, audiocd.Channels*audiocd.BytesPerSample)
	for i := range len(samples) {
		_, err = s.buf.Read(f)
		if err != nil {
			s.err = err
			return 0, false
		}
		samples[i][0], samples[i][1] = extractFrame(f)
		n += 1
	}
	return n, true
}

func (s *WaveFileStreamer) Err() error {
	return s.err
}

func (s *WaveFileStreamer) Len() int {
	return int((s.sizeBytes - WavHeaderSize) / audiocd.Channels * audiocd.BytesPerSample)
}

func (s *WaveFileStreamer) Position() int {
	return int((s.offset - WavHeaderSize) / audiocd.Channels * audiocd.BytesPerSample)
}

func (s *WaveFileStreamer) Seek(p int) error {
	bp := int64((p * audiocd.Channels * audiocd.BytesPerSample) + WavHeaderSize)
	n, err := s.f.Seek(bp, io.SeekStart)
	s.offset = int64(n)
	return err
}

func (s *WaveFileStreamer) Close() error {
	s.buf.Truncate(0)
	return s.f.Close()
}

var _ beep.StreamSeekCloser = (*WaveFileStreamer)(nil)
