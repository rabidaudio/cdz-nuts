package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rabidaudio/cdz-nuts/spi"
)

func main() {
	// s, err := spi.Open()
	// if err != nil {
	// 	panic(err)
	// }
	// defer s.Close()

	f, err := os.Open("vfs/testdata/chronictown.img")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	sdev, err := spi.Open()
	if err != nil {
		panic(err)
	}
	defer sdev.Close()

	done := make(chan struct{})
	go func() {
		err = PollTransfer(sdev, f, done)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	fmt.Printf("running, ctrl-c to stop\n")
	<-sigs
	done <- struct{}{}
}
