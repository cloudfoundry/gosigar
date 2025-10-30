//go:build darwin || linux || netbsd || openbsd

package sigar

import (
	"syscall"
)

func (fs *FileSystemUsage) Get(path string) error { //nolint:staticcheck
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return err
	}

	bsize := stat.Bsize / 512

	fs.Total = (stat.Blocks * uint64(bsize)) >> 1
	fs.Free = (stat.Bfree * uint64(bsize)) >> 1
	fs.Avail = (stat.Bavail * uint64(bsize)) >> 1
	fs.Used = fs.Total - fs.Free
	fs.Files = stat.Files
	fs.FreeFiles = stat.Ffree

	return nil
}
