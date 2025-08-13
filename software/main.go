package main

import (
	// "go.uploadedlobster.com/discid"
	"fmt"
	"io"
	"os"

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

	fmt.Printf("drive: %+v | model: %v sectors/read: %v type: %v (%d) iface: %v\n", drive, drive.Model(), drive.SectorsPerRead(), drive.DriveType(), int(drive.DriveType()), drive.InterfaceType())

	toc := drive.TOC()
	fmt.Printf("TOC: %+v", toc)

	start := toc[4].StartSector

	_, err = drive.Seek(int64(start), io.SeekStart)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, toc[4].LengthSectors*cdparanoia.SectorSizeRaw)
	read := 0
	for read < len(buf) {
		n, err := drive.Read(buf[read:])
		if err != nil {
			panic(err)
		}
		read += n
	}

	err = os.WriteFile("track5.cdda", buf, 0777)
	if err != nil {
		panic(err)
	}

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
