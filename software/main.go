package main

import (
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/rabidaudio/audiocd"
)

func main() {
	err := speaker.Init(AudioCDFormat.SampleRate, AudioCDFormat.SampleRate.N(time.Second/10))
	if err != nil {
		panic(err)
	}

	cd := audiocd.AudioCD{LogMode: audiocd.LogModeStdErr}
	st, err := NewStreamer(&cd)
	if err != nil {
		panic(err)
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(st, beep.Callback(func() {
		done <- true
	})))

	<-done

	// f, err := os.Open("vfs/testdata/chronictown.img")
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()

	// sdev, err := spi.Open()
	// if err != nil {
	// 	panic(err)
	// }
	// defer sdev.Close()

	// done := make(chan struct{})
	// go func() {
	// 	err = PollTransfer(sdev, f, done)
	// }()

	// sigs := make(chan os.Signal, 1)
	// signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// fmt.Printf("running, ctrl-c to stop\n")
	// <-sigs
	// done <- struct{}{}
}
