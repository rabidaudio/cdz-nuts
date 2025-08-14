package mock

import "github.com/rabidaudio/cdz-nuts/spi"

type spiI interface {
	Query() (dr spi.DataRequest, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

var _ spiI = (*spi.Spi)(nil)

type Spi struct {
	RequestedBlock int32
	BytesWritten   int
}

var _ spiI = (*Spi)(nil)

func (m *Spi) Query() (dr spi.DataRequest, err error) {
	if m.RequestedBlock < 0 {
		dr.Requested = false
	} else {
		dr.Requested = true
		dr.Address = uint32(m.RequestedBlock)
		dr.SectorCount = 1
		m.RequestedBlock = -1
	}
	return
}

func (m *Spi) Write(p []byte) (n int, err error) {
	m.BytesWritten += len(p)
	return len(p), nil
}

func (m *Spi) Close() error {
	return nil
}
