package main

type spiI interface {
	Query() (dr DataRequest, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

var _ spiI = (*Spi)(nil)

type MSpi struct {
	RequestedBlock int32
	BytesWritten   int
}

var _ spiI = (*Spi)(nil)

func (m *MSpi) Query() (dr DataRequest, err error) {
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

func (m *MSpi) Write(p []byte) (n int, err error) {
	m.BytesWritten += len(p)
	return len(p), nil
}

func (m *MSpi) Close() error {
	return nil
}
