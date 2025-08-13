package audiocd

// SampleRate is the number of samples per second. All Redbook audio
// CDs use at 44.1KHz.
const SampleRate = 44100

// BytesPerSample is 2 bytes, representing signed 16-bit samples.
const BytesPerSample = 2

// Channels is the number of audio channels in the data. All Redbook
// audio CDs are stereo.
//
// [Wikipedia] notes that four-channel audio support was planned but never
// implemented and no known drives support it.
//
// [Wikipedia]: https://en.wikipedia.org/wiki/Compact_Disc_Digital_Audio#Audio_format
const Channels = 2

// FramesPerSecond is the number of audio frames in one second of audio.
// An audio frame is the smallest valid unit of length for a track, defined
// as 1/75th of a second. Redbook track offsets are specified in MM:SS:FF.
//
// Note that this definition of frame is interchangable with sector.
// It is distinct from a 33-byte channel data frame, which this package does
// not concern itself with.
//
// For more information, see [Wikipdia].
//
// [Wikipdia]: https://en.wikipedia.org/wiki/Compact_Disc_Digital_Audio#Frames_and_timecode_frames
const FramesPerSecond = 75

// SamplesPerFrame is the number of 16-bit audio samples per channel
// that appear within one frame of data (294).
const SamplesPerFrame = SampleRate / FramesPerSecond / Channels

// BytesPerSector is the number of bytes of audio contained in one sector of
// CD data (and equivalently in one frame of samples), 2352 bytes.
//
// Sectors are the unit of interest when reading data from CDs. AudioCD reads
// data in units of sectors.
const BytesPerSector = SampleRate * Channels * BytesPerSample / FramesPerSecond
