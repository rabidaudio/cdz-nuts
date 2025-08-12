package spi

import (
	"encoding/binary"
	"fmt"

	rpio "github.com/stianeikeland/go-rpio/v4"
)

// On loop:
// Start transaction
// Pi asks if any blocks to request [COMMAND_QUERY],
// Mcu responds [ACK | NAK [1 byte], sector address [uint32_t], sector count [uint8_t]]
// Pi [ACK], closes transaction
// if data was requested, Pi opens new transaction, writes all requested bytes, closes

const (
	NAK   = iota // 0x00
	ACK          // 0x01
	QUERY        // 0x02
)

const SPI_SPEED = 10_000_000 // 10 MHz

var ErrNoResponse = fmt.Errorf("spi: no response from mcu")

type Spi struct {
	dev        rpio.SpiDev
	chipSelect uint8
}

type DataRequest struct {
	Requested   bool
	Address     uint32
	SectorCount uint8
}

func Open() (*Spi, error) {
	return OpenDevice(rpio.Spi0, 0)
}

func OpenDevice(dev rpio.SpiDev, chipSelect uint8) (spi *Spi, err error) {
	err = rpio.Open()
	if err != nil {
		return
	}
	err = rpio.SpiBegin(dev)
	if err != nil {
		return
	}
	rpio.SpiChipSelect(chipSelect)
	rpio.SpiSpeed(SPI_SPEED)
	spi = &Spi{dev: dev, chipSelect: chipSelect}
	return
}

func (*Spi) Query() (dr DataRequest, err error) {
	bytes := make([]byte, 7)
	bytes[0] = QUERY
	rpio.SpiExchange(bytes)
	switch bytes[1] {
	case ACK:
		dr.Requested = true
	case NAK:
		dr.Requested = false
	default:
		for _, b := range bytes {
			if b != 0 {
				return dr, fmt.Errorf("spi: invalid response from mcu: %v", bytes)
			}
		}
		err = ErrNoResponse
		return
	}
	dr.Address = binary.LittleEndian.Uint32(bytes[2:])
	dr.SectorCount = bytes[6]
	return
}

// NOTE: only use writer interface if data has been requested!
func Write(p []byte) (n int, err error) {
	rpio.SpiTransmit(p...)
	return len(p), nil
}

func (s *Spi) Close() error {
	rpio.SpiEnd(s.dev)
	return nil
}
