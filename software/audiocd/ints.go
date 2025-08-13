package audiocd

// type SectorIndex int

// func toSectorIndex(byteOffset int32) SectorIndex {
// 	return SectorIndex(byteOffset / SectorSizeRaw)
// }

// func (si SectorIndex) ByteOffset() int32 {
// 	return int32(si) * SectorSizeRaw
// }

// (samples/second)*(bytes/sample)*(channels)/(bytes/sector) = 75 sectors/sec
const SectorsPerSecond = (44100 * 2 * 2) / 2352
