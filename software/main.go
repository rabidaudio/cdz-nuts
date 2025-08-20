package main

import (
	"fmt"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/manifoldco/promptui"
	"github.com/rabidaudio/audiocd"
)

func main() {
	err := speaker.Init(AudioCDFormat.SampleRate, audiocd.SamplesPerFrame)
	if err != nil {
		panic(err)
	}

	cd := audiocd.AudioCD{LogMode: audiocd.LogModeStdErr}
	err = cd.Open()
	if err != nil {
		panic(err)
	}
	defer cd.Close()

	s, err := NewStreamer(&cd)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	fmt.Printf("setup complete\n")

	done := make(chan bool)

	ctrl := &beep.Ctrl{Streamer: beep.Seq(s, beep.Callback(func() {
		done <- true
	})), Paused: false}

	speaker.Play(ctrl)

	for {
		fmt.Printf("playing %v | track %d\n", !ctrl.Paused, s.CurrentTrack()+1)
		prompt := promptui.Prompt{
			Label: "n=next, p=previous, enter=play/pause, q=quit",
		}

		result, err := prompt.Run()
		if err != nil {
			panic(err)
		}

		switch result {
		case "":
			ctrl.Paused = !ctrl.Paused
		case "p":
			s.Prev()
		case "n":
			s.Next()
		case "q":
			return
		}
	}

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
