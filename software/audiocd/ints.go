package audiocd

const SampleRate = 44100

const Channels = 2

const BytesPerSector = 2352

const SectorsPerSecond = 75 // (SampleRate * Channels * 2 bytes/sample)/BytesPerSector

const FullSpeed = -1

// The track positions are referenced by absolute timecode,
// relative to the start of the program area, in MSF format:
// minutes, seconds, and fractional seconds called frames.
// Each timecode frame is one seventy-fifth of a second, and
// corresponds to a block of 98 channel-data frames—ultimately,
// a block of 588 pairs of left and right audio samples.
