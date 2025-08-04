package cd

type Track struct {
	Filename    string
	LengthBytes int64
}

type CD struct {
	Name   string
	Tracks []Track
}
