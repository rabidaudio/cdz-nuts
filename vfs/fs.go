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

const DISK_SIZE = 700 * fat32.MB
const SECTOR_SIZE = 512
const START = 2048

// Filesystem represents a virtual FAT32 filesystem containing WAV files
// corresponding to the tracks on the CD.
type Filesystem struct {
	filesystem.FileSystem
	Path    string
	cd      *cd.CD
	closefn func() error
}

// sanitizeName takes a file name and converts it to DOS format
// by uppercasing, limiting to ASCII letters, and triming to 8 chars
func sanitizeName(name string) string {
	// https://en.wikipedia.org/wiki/8.3_filename
	newName := make([]rune, 0, max(len(name), 8))
	for _, r := range []rune(strings.ToUpper(name)) {
		if len(newName) == 8 {
			break
		}
		if r >= 'A' && r <= 'Z' {
			newName = append(newName, r)
		}
	}
	return string(newName)
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
				// STOPSHIP
				Start: 0,
				Size:  uint32(10 * fat32.MB),
				// Start:    START,
				// Size:     uint32(DISK_SIZE) / SECTOR_SIZE,
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
		Path:       dskimg,
		FileSystem: fatfs,
		closefn:    closefn,
	}

	return &f, nil
}

// Create virtual files based on the CD configuration
func (f *Filesystem) LoadCD(cd cd.CD) (err error) {
	sDirName := sanitizeName(cd.Name)
	if sDirName != "" {
		sDirName = "/" + sDirName
		err = f.Mkdir(sDirName)
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
		fname := fmt.Sprintf("%v/TRACK%02d.WAV", sDirName, i)
		file, err := f.OpenFile(fname, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return fmt.Errorf("create track %v: %w", fname, err)
		}

		// NOTE: rather than copy from /dev/zero, a much faster way
		// to initialize the files is to seek the length and then write
		// zero bytes. This is proabably an implementation quirk though
		// so there's a test for it in case the behavior changes.

		// _, err = io.CopyN(file, zero, track.LengthBytes)
		// if err != nil {
		// 	return fmt.Errorf("write null %v: %w", fname, err)
		// }

		_, err = file.Seek(track.LengthBytes, io.SeekStart)
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

// Delete all files from the filesystem
func (f *Filesystem) Eject() {
	// TODO
}

func (f *Filesystem) Close() error {
	return f.closefn()
}
