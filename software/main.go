package main

import (
	// "go.uploadedlobster.com/discid"
	"fmt"

	"github.com/rabidaudio/cdz-nuts/cdparanoia"
)

func main() {
	// disc := discid.Read("") // Read from default device
	// defer disc.Close()

	cdparanoia.EnableLogs = true

	fmt.Printf("value: %v\n", cdparanoia.Version())
	drive, err := cdparanoia.OpenDevice("/dev/sr1")
	if err != nil {
		panic(err)
	}
	defer drive.Close()

	fmt.Printf("drive: %+v | model: %v sectors: %v type: %v (%d) iface: %v\n", drive, drive.Model(), drive.SectorCount(), drive.DriveType(), int(drive.DriveType()), drive.InterfaceType())

	toc, err := drive.TOC()
	if err != nil {
		panic(err)
	}
	fmt.Printf("TOC: %+v", toc)

	// 	s, err := spi.Open()
	// if err != nil {
	// 	panic(err)
	// }
	// defer s.Close()

	// dr, err := s.Query()
	// if err != nil {
	// 	panic(fmt.Errorf("query: %w", err))
	// }
	// fmt.Printf("request: %v\n", dr)
}
