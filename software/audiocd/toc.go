package audiocd

// TODO: make constants for these. Couldn't find the definitions, should be in redbook
type Flag uint8

func (f Flag) IsAudio() bool {
	return (uint8(f) & 0x04) == 0
}

type TOC struct {
	Flags         Flag
	TrackNum      uint8 // 1-indexed
	StartSector   int32
	LengthSectors int32 // TODO: handle airgaps
}
