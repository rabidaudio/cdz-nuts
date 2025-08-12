package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rabidaudio/cdz-nuts/spi"
	"github.com/rabidaudio/cdz-nuts/vfs"
)

func PollTransfer(s *spi.Spi, f *os.File, close chan struct{}) error {
	for {
		select {
		case <-time.After(time.Millisecond):
			dr, err := s.Query()
			if err != nil {
				return err
			}
			if dr.Requested {
				fmt.Printf("received request %v\n", dr)
				addr := dr.Address * vfs.SECTOR_SIZE
				f.Seek(int64(addr), io.SeekStart)
				count := int64(vfs.SECTOR_SIZE) * int64(dr.SectorCount)
				_, err = io.CopyN(s, f, count)
				if err != nil {
					return err
				}
			}
		case <-close:
			fmt.Printf("closing")
			return nil
		}
	}
}
