package audiocd

// Flag is a set of bit flags attached to a track in the CD's
// table of contents.
//
// TODO(jdk): unable find a definition of these flags
type Flag uint8

// IsAudio reports wheither the track is an audio track.
// Mixed-mode disks can have data tracks in addition to audio tracks.
func (t TrackPosition) IsAudio() bool {
	return (uint8(t.Flags) & 0x04) == 0
}

// TrackPosition reports the offset information for tracks
// from the table of contents.
type TrackPosition struct {
	Flags         Flag
	TrackNum      uint8 // index of the track, starting at 1
	StartSector   int32 // address of the sector where the data starts
	LengthSectors int32 // total number of sectors the track covers
}

// TODO: handle pregap?
