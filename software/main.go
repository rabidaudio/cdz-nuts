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
	drive, err := cdparanoia.Init()
	if err != nil {
		panic(err)
	}
	defer drive.Close()

	fmt.Printf("drive: %v\n", drive)
}
