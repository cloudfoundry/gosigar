//go:build freebsd
// +build freebsd

package sigar

import (
	"bytes"
	"errors"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var (
	Procd string = "/proc"
)

type loadStruct struct {
	Ldavg  [3]uint32
	Fscale uint64
}

func procFileName(pid int, name string) string {
	return Procd + "/" + strconv.Itoa(pid) + "/" + name
}

func readProcFile(pid int, name string) ([]byte, error) {
	path := procFileName(pid, name)
	contents, err := ioutil.ReadFile(path)

	if err != nil {
		if perr, ok := err.(*os.PathError); ok {
			if perr.Err == unix.ENOENT {
				return nil, unix.ESRCH
			}
		}
	}

	return contents, err
}

func (self *Uptime) Get() error {
	var tv unix.Timeval
	boottimeRaw, err := unix.SysctlRaw("kern.boottime")
	if err != nil {
		return err
	}
	tv = *(*unix.Timeval)(unsafe.Pointer(&boottimeRaw[0]))
	self.Length = time.Since(time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)).Seconds()
	return nil
}

func (self *LoadAverage) Get() error {
	avgRaw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil {
		return err
	}
	avg := *(*loadStruct)(unsafe.Pointer(&avgRaw[0]))
	fscale := float64(avg.Fscale)

	self.One = float64(avg.Ldavg[0]) / fscale
	self.Five = float64(avg.Ldavg[1]) / fscale
	self.Fifteen = float64(avg.Ldavg[2]) / fscale

	return nil
}

func (self *ProcList) Get() error {
	dir, err := os.Open(Procd)
	if err != nil {
		return err
	}
	defer dir.Close()

	const readAllDirnames = -1 // see os.File.Readdirnames doc

	names, err := dir.Readdirnames(readAllDirnames)
	if err != nil {
		return err
	}

	capacity := len(names)
	list := make([]int, 0, capacity)

	for _, name := range names {
		if name[0] < '0' || name[0] > '9' {
			continue
		}
		pid, err := strconv.Atoi(name)
		if err == nil {
			list = append(list, pid)
		}
	}

	self.List = list

	return nil
}

func (self *ProcState) Get(pid int) error {
	contents, err := readProcFile(pid, "status")
	if err != nil {
		return err
	}
	fields := strings.Fields(string(contents))

	self.Name = fields[0]
	self.Ppid, _ = strconv.Atoi(fields[2])

	return nil
}

func (self *FileSystemList) Get() error {
	n, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	if err != nil {
		return err
	}
	fslist := make([]FileSystem, 0, n)
	buf := make([]unix.Statfs_t, n)
	unix.Getfsstat(buf, unix.MNT_NOWAIT)
	for _, f := range buf {
		fs := FileSystem{}
		fs.DirName = string(bytes.Trim(f.Mntonname[:], "\x00"))
		fs.DevName = string(bytes.Trim(f.Mntfromname[:], "\x00"))
		fs.SysTypeName = string(bytes.Trim(f.Fstypename[:], "\x00"))
		fslist = append(fslist, fs)
	}
	self.List = fslist
	return nil
}

func (self *FileSystemUsage) Get(path string) error {
	stat := unix.Statfs_t{}
	err := unix.Statfs(path, &stat)
	if err != nil {
		return err
	}

	bsize := stat.Bsize / 512

	self.Total = (uint64(stat.Blocks) * uint64(bsize)) >> 1
	self.Free = (uint64(stat.Bfree) * uint64(bsize)) >> 1
	self.Avail = (uint64(stat.Bavail) * uint64(bsize)) >> 1
	self.Used = self.Total - self.Free
	self.Files = stat.Files
	self.FreeFiles = uint64(stat.Ffree)

	return nil
}

func (self *ProcTime) Get(pid int) error {
	rusage := unix.Rusage{}
	unix.Getrusage(pid, &rusage)
	self.User = uint64(rusage.Utime.Nano() / 1e6)
	self.Sys = uint64(rusage.Stime.Nano() / 1e6)
	self.Total = self.User + self.Sys
	return nil
}

func (self *Cpu) Get() error {
	// unix.SysctlRaw("kern.cp_time")
	return errors.New("not implemented: Cpu")
}

func (self *Mem) Get() error {
	return errors.New("not implemented: Mem")
}

func (self *Mem) GetIgnoringCGroups() error {
	return errors.New("not implemented: Mem")
}

func (self *Swap) Get() error {
	// Use vm.swap_total sysctl ?
	return errors.New("not implemented: Swap")
}

func (self *CpuList) Get() error {
	return errors.New("not implemented: CpuList")
}

func (self *ProcMem) Get(pid int) error {
	return errors.New("not implemented: ProcMem")
}

func (self *ProcArgs) Get(pid int) error {
	return errors.New("not implemented: ProcArgs")
}

func (self *ProcExe) Get(pid int) error {
	return errors.New("not implemented: ProcExe")
}
