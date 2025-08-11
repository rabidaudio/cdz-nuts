package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"syscall"
)

func main() {
	drv, err := os.OpenFile("/dev/disk8", os.O_RDONLY|syscall.O_NONBLOCK, fs.ModePerm)
	if err != nil {
		panic(err)
	}

	i := 0
	buf := new(bytes.Buffer)
	for i < 0x1000 {
		n, err := drv.Read(buf.Bytes())
		i += n
		if err != nil {
			panic(err)
		}
	}
	out, err := os.Create("dump.bin")
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(out, buf)
	if err != nil {
		panic(err)
	}
}
