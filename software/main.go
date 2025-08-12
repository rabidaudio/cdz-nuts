package main

import (
	"fmt"

	"github.com/rabidaudio/cdz-nuts/spi"
)

func main() {
	s, err := spi.Open()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	dr, err := s.Query()
	if err != nil {
		panic(fmt.Errorf("query: %w", err))
	}
	fmt.Printf("request: %v\n", dr)
}
