package vfs

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/fat32"
	"github.com/diskfs/go-diskfs/partition/mbr"
	"github.com/rabidaudio/carcd-adapter/cd"
)

const DISK_SIZE = 50 * fat32.MB // STOPSHIP // 700 * fat32.MB
const SECTOR_SIZE = 512

// Filesystem represents a virtual FAT32 filesystem containing WAV files
// corresponding to the tracks on the CD.
type Filesystem struct {
	fs      filesystem.FileSystem
	Path    string
	cd      *cd.CD
	closefn func() error
}

// sanitizeName takes a file name and converts it to DOS format
// by uppercasing, limiting to ASCII letters, and triming to 8 chars
func sanitizeName(name string) string {
	// https://en.wikipedia.org/wiki/8.3_filename
	newName := make([]rune, 0, 8)
	for _, r := range strings.ToUpper(name) {
		if len(newName) == 8 {
			break
		}
		if r >= 'A' && r <= 'Z' {
			newName = append(newName, r)
		}
	}
	return string(newName)
}

func trackSizeBytes(t *cd.Track) int64 {
	// TODO: include metadata, artifical track predelay
	// 6 samples per channel per frame, 16 bits per sample plus 44 header
	return int64(t.LengthFrames*6*2) + 44
}

// Create a new filesystem instance. Data is backed by a temporary file.
// Be sure to Close() the Filesystem after use.
func Create() (*Filesystem, error) {
	// Setup a file in the tmp directory to be a virtual filesystem
	tmpdir, err := os.MkdirTemp("", "carcd")
	if err != nil {
		return nil, err
	}
	dskimg := tmpdir + "/disk.img"
	dsk, err := diskfs.Create(dskimg, DISK_SIZE, diskfs.SectorSizeDefault)
	if err != nil {
		return nil, err
	}

	// create an MBR with one partition
	table := &mbr.Table{
		LogicalSectorSize:  SECTOR_SIZE,
		PhysicalSectorSize: SECTOR_SIZE,
		Partitions: []*mbr.Partition{
			{
				Bootable: false,
				Type:     mbr.Linux,
				Start:    0,
				Size:     uint32(DISK_SIZE) / SECTOR_SIZE,
			},
		},
	}
	err = dsk.Partition(table)
	if err != nil {
		defer os.Remove(dskimg)
		return nil, err
	}
	// Create a FAT32 filesystem
	fatfs, err := dsk.CreateFilesystem(disk.FilesystemSpec{
		Partition:   1,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: "VIRTUALCD",
	})
	if err != nil {
		defer os.Remove(dskimg)
		return nil, err
	}

	closefn := func() (err error) {
		err = fatfs.Close()
		if err != nil {
			return err
		}
		err = os.Remove(dskimg)
		if err != nil {
			return err
		}
		err = os.Remove(tmpdir)
		if err != nil {
			return err
		}
		return nil
	}

	f := Filesystem{
		Path:    dskimg,
		fs:      fatfs,
		closefn: closefn,
	}

	return &f, nil
}

// Create virtual files based on the CD configuration
func (f *Filesystem) LoadCD(cd cd.CD) (err error) {
	if f.cd != nil {
		return fmt.Errorf("current CD not ejected")
	}

	sDirName := dirName(cd)
	if sDirName != "" {
		err = f.fs.Mkdir(sDirName)
		if err != nil {
			return err
		}
	}

	// Open /dev/zero for reading null bytes
	// zero, err := os.Open("/dev/zero")
	// if err != nil {
	// 	return fmt.Errorf("read /dev/zero: %w", err)
	// }
	// defer zero.Close()

	for i, track := range cd.Tracks {
		// TODO: try reading ID3 data for generating track names
		fname, _ := trackPath(cd, i)
		file, err := f.fs.OpenFile(fname, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return fmt.Errorf("create track %v: %w", fname, err)
		}

		// NOTE: rather than copy from /dev/zero, a much faster way
		// to initialize the files is to seek the length and then write
		// zero bytes. This is proabably an implementation quirk though
		// so there's a test for it in case the behavior changes.

		// _, err = io.CopyN(file, zero, trackSizeBytes(&track))
		// if err != nil {
		// 	return fmt.Errorf("write null %v: %w", fname, err)
		// }

		_, err = file.Seek(trackSizeBytes(&track), io.SeekStart)
		if err != nil {
			return err
		}
		_, err = file.Write([]byte{})
		if err != nil {
			return err
		}
	}
	f.cd = &cd
	return nil
}

func dirName(cd cd.CD) string {
	sDirName := sanitizeName(cd.Name)
	if sDirName != "" {
		sDirName = "/" + sDirName
	}
	return sDirName
}

func trackPath(cd cd.CD, i int) (string, bool) {
	if i < 0 || i >= len(cd.Tracks) {
		return "", false
	}
	// t := cd.Tracks[i]
	return fmt.Sprintf("%v/TRACK%02d.WAV", dirName(cd), i), true
}

type TrackRange struct {
	FileInfo   os.FileInfo
	DiskRanges []fat32.DiskRange
}

// Get the block bounds over which the track files are placed
func (f *Filesystem) TrackRanges() ([]TrackRange, error) {
	if f.cd == nil {
		return nil, fmt.Errorf("no CD loaded")
	}

	fileInfo, err := f.fs.ReadDir(dirName(*f.cd))
	if err != nil {
		return nil, err
	}

	trackRanges := make([]TrackRange, len(f.cd.Tracks))

	for i := range f.cd.Tracks {
		path, ok := trackPath(*f.cd, i)
		if !ok {
			return nil, fmt.Errorf("invalid track index")
		}

		tf, err := f.fs.OpenFile(path, os.O_RDONLY)
		if err != nil {
			return nil, err
		}
		defer tf.Close()

		fattf, ok := tf.(*fat32.File)
		if !ok {
			return nil, fmt.Errorf("not fat32 file")
		}
		diskRange, err := fattf.GetDiskRanges()
		if err != nil {
			return nil, err
		}
		trackRanges[i] = TrackRange{DiskRanges: diskRange}

		for _, fi := range fileInfo {
			if fi.IsDir() {
				continue
			}
			if name, ok := trackPath(*f.cd, i); ok && strings.HasSuffix(name, fi.Name()) {
				trackRanges[i].FileInfo = fi
			}
		}
	}

	return trackRanges, nil
}

// Delete all files from the filesystem
func (f *Filesystem) Eject() error {
	if f.cd == nil {
		return nil
	}
	for i := range f.cd.Tracks {
		// TODO: try reading ID3 data for generating track names
		fname, _ := trackPath(*f.cd, i)
		err := f.fs.Remove(fname)
		if err != nil {
			return err
		}
	}
	err := f.fs.Remove(dirName(*f.cd))
	if err != nil {
		return err
	}
	f.cd = nil
	return nil
}

func (f *Filesystem) Close() error {
	f.Eject()
	return f.closefn()
}
