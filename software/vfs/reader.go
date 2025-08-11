package vfs

import (
	"io"
	"log"
	"math"
	"os"

	"github.com/diskfs/go-diskfs/filesystem/fat32"
)

type vfsReader struct {
	f           *Filesystem
	imgFile     *os.File
	offset      int64
	trackRanges []TrackRange
}

// Create an io.Reader that reads from the image for filesystem data
// and from track wav data when in a track boundary
func (f *Filesystem) Reader() (io.ReadSeeker, error) {
	trackRanges, err := f.TrackRanges()
	if err != nil {
		return nil, err
	}
	imgf, err := os.Open(f.Path)
	if err != nil {
		return nil, err
	}
	return &vfsReader{f: f, imgFile: imgf, trackRanges: trackRanges, offset: 0}, nil
}

func (r *vfsReader) Read(p []byte) (int, error) {
	// if we are within a track, we should read from there instead
	// otherwise we read from the file
	// since Read is allowed to read less than requested, we always stop
	// reading at a track block boundary
	withinTrack := -1
	var trackRange fat32.DiskRange
	var nextTrackBounds uint64 = math.MaxInt64
	// ...[======]...[=====].[=========]..
	// ^^          ^--^   ^--------^
	end := uint64(r.offset) + uint64(len(p))
	for i, tr := range r.trackRanges {
		for _, dr := range tr.DiskRanges {
			if r.offset >= int64(dr.Offset) && r.offset < int64(dr.Offset+dr.Length) {
				withinTrack = i
				trackRange = dr
			}
			if dr.Offset > uint64(r.offset) && dr.Offset < end {
				nextTrackBounds = dr.Offset
			}
		}
	}

	size := len(p)
	if withinTrack == -1 {
		// read directly from file, up to len(p) or nextTrackBounds, whichever is smaller
		distanceToNextTrack := int(nextTrackBounds - uint64(r.offset))
		if size > distanceToNextTrack {
			size = distanceToNextTrack
		}

		n, err := r.imgFile.Read(p[:size])
		log.Printf("%08d\t%05d\tflash\t%v\n", r.offset, n, err)
		r.offset += int64(n)
		return n, err
	} else {
		// read from the specified track up to len(p) or trackRange, whichever is smaller

		distanceToEndOfRange := int(trackRange.Offset+trackRange.Length) - int(r.offset)
		if size > distanceToEndOfRange {
			size = distanceToEndOfRange
		}

		track := r.f.cd.Tracks[withinTrack]

		// figure out where in the track we are
		var tOffset int64 = 0
		for _, dr := range r.trackRanges[withinTrack].DiskRanges {
			if dr.Offset == trackRange.Offset {
				break
			}
			tOffset += int64(dr.Length)
		}
		tOffset += r.offset - int64(trackRange.Offset)
		_, err := track.Seek(int64(tOffset), io.SeekStart)
		if err != nil {
			return 0, err
		}
		// size should also not go beyond the actual track size
		trackLength := r.trackRanges[withinTrack].FileInfo.Size()
		distanceToEndOfTrack := int(trackLength - int64(tOffset))
		if size > distanceToEndOfTrack {
			// if it does, read to end of track and then read from filesystem
			n, err := track.Read(p[:distanceToEndOfTrack])
			log.Printf("%08d\t%05d\twav %v:end\t%v\n", r.offset, n, withinTrack, err)
			r.offset += int64(n)
			if err != nil {
				return n, err
			}

			n2, err := r.imgFile.Read(p[distanceToEndOfTrack:size])
			log.Printf("%08d\t%05d\tflash wavend\t%v\n", r.offset, n2, err)
			r.offset += int64(n2)
			return n + n2, err
		}

		n, err := track.Read(p[:size])
		log.Printf("%08d\t%05d\twav %v\t%v\n", r.offset, n, withinTrack, err)
		r.offset += int64(n)
		return n, err
	}
}

func (r *vfsReader) Seek(offset int64, whence int) (int64, error) {
	// seek position should match file
	newOffset, err := r.imgFile.Seek(offset, whence)
	r.offset = newOffset
	return newOffset, err
}

// ensure interface conformation
var _ io.ReadSeeker = (*vfsReader)(nil)
