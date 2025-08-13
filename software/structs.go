package main

import "io"

type Track struct {
	io.ReadSeeker
	Filename     string
	LengthFrames uint
}

type CD struct {
	Name   string
	Tracks []Track
}
