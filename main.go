package main

import "github.com/rabidaudio/carcd-adapter/vfs"

func main() {
	f, err := vfs.Create()
	if err != nil {

	}
	defer f.Close()

}
