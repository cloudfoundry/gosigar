//go:build freebsd
// +build freebsd

package sigar

import (
	"errors"
	"golang.org/x/sys/unix"
	"syscall"
	"time"
	"unsafe"
)

type loadStruct struct {
	Ldavg  [3]uint32
	Fscale uint64
}

func (self *Uptime) Get() error {
	var tv syscall.Timeval
	boottimeRaw, err := unix.SysctlRaw("kern.boottime")
	if err != nil {
		return err
	}
	tv = *(*syscall.Timeval)(unsafe.Pointer(&boottimeRaw[0]))
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

func (self *FileSystemUsage) Get(path string) error {
	return errors.New("not implemented")
}

func (self *Cpu) Get() error {
	// Use kern.cp_time sysctl ?
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

func (self *FileSystemList) Get() error {
	return errors.New("not implemented")
}

func (self *ProcList) Get() error {
	return errors.New("not implemented")
}

func (self *ProcTime) Get(pid int) error {
	return errors.New("not implemented")
}

func (self *ProcState) Get(pid int) error {
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
