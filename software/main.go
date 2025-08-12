package main

import (
	// "go.uploadedlobster.com/discid"
	"fmt"

	"github.com/rabidaudio/cdz-nuts/cd"
)

func main() {
	// disc := discid.Read("") // Read from default device
	// defer disc.Close()

	fmt.Printf("value: %v\n", cd.TestIntegration())
}
