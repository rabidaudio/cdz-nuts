package main

import (
	"fmt"
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
	// keys := make(chan term.Key)

	// err = term.Init()
	// if err != nil {
	// 	panic(err)
	// }
	// defer term.Close()
	// go func() {
	// 	for {
	// 		ev := term.PollEvent()
	// 		switch ev.Type {
	// 		case term.EventKey:
	// 			term.Sync()
	// 			key := ev.Key
	// 			keys <- key
	// 			if key == term.KeyEsc {
	// 				close(keys)
	// 				return
	// 			}
	// 		case term.EventError:
	// 			panic(ev.Err)
	// 		}
	// 	}
	// }()

	// ctrl := &beep.Ctrl{Streamer: beep.Seq(s, beep.Callback(func() {
	// 	done <- true
	// })), Paused: false}

	// speaker.Play(ctrl)

	seq := beep.Seq(s, beep.Callback(func() {
		done <- true
	}))
	speaker.Play(seq)
	fmt.Printf("playing\n")

	<-done
	// for {
	// 	select {
	// 	case key, ok := <-keys:
	// 		if !ok {
	// 			return
	// 		}
	// 		switch key {
	// 		case term.KeyEsc:
	// 			return
	// 		case term.KeySpace:
	// 			fmt.Printf("playpause\n")
	// 			ctrl.Paused = !ctrl.Paused
	// 		case term.KeyArrowLeft:
	// 			fmt.Printf("previous\n")
	// 			s.Prev()
	// 		case term.KeyArrowRight:
	// 			fmt.Printf("next\n")
	// 			s.Next()
	// 		}
	// 	case <-done:
	// 		return
	// 	}
	// }

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
