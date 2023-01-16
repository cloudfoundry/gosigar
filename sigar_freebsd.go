//go:build freebsd
// +build freebsd

package sigar

import (
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
		fs.DirName = string(f.Mntonname[:])
		fs.DevName = string(f.Mntfromname[:])
		fs.SysTypeName = string(f.Fstypename[:])
		fslist = append(fslist, fs)
	}
	self.List = fslist
	return nil
}

func (self *FileSystemUsage) Get(path string) error {
	return errors.New("not implemented")
}

func (self *Cpu) Get() error {
	// unix.SysctlRaw("kern.cp_time")
	return errors.New("not implemented")
}

func (self *Mem) Get() error {
	return errors.New("not implemented")
}

func (self *Mem) GetIgnoringCGroups() error {
	return errors.New("not implemented")
}

func (self *Swap) Get() error {
	// Use vm.swap_total sysctl ?
	return errors.New("not implemented")
}

func (self *CpuList) Get() error {
	return errors.New("not implemented")
}

func (self *ProcTime) Get(pid int) error {
	return errors.New("not implemented")
}

func (self *ProcMem) Get(pid int) error {
	return errors.New("not implemented")
}

func (self *ProcArgs) Get(pid int) error {
	return errors.New("not implemented")
}

func (self *ProcExe) Get(pid int) error {
	return errors.New("not implemented")
}
