package cd

type Track struct {
	Filename     string
	LengthFrames uint
}

type CD struct {
	Name   string
	Tracks []Track
}
