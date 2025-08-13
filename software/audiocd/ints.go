package audiocd

func RealtimeSectorsPerSecond(sampleRate, numchannels int) int {
	return sampleRate * numchannels * 2 /*sizeof(uint16_t)*/ / int(BytesPerSector)
}
