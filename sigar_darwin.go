// Copyright (c) 2012 VMware, Inc.

package sigar

/*
#include <stdlib.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <mach/mach_init.h>
#include <mach/mach_host.h>
#include <mach/host_info.h>
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

func (self *LoadAverage) Get() error {
	avg := []C.double{0, 0, 0}

	C.getloadavg(&avg[0], C.int(len(avg)))

	self.One = float64(avg[0])
	self.Five = float64(avg[1])
	self.Fifteen = float64(avg[2])

	return nil
}

func (self *Uptime) Get() error {
	tv := syscall.Timeval{}

	if err := sysctlbyname("kern.boottime", &tv); err != nil {
		return err
	}

	self.Length = time.Since(time.Unix(tv.Unix())).Seconds()

	return nil
}

func (self *Mem) Get() error {
	var vmstat C.vm_statistics_data_t

	if err := sysctlbyname("hw.memsize", &self.Total); err != nil {
		return err
	}

	if err := vm_info(&vmstat); err != nil {
		return err
	}

	kern := uint64(vmstat.inactive_count) << 12
	self.Free = uint64(vmstat.free_count) << 12

	self.Used = self.Total - self.Free
	self.ActualFree = self.Free + kern
	self.ActualUsed = self.Used - kern

	return nil
}

type xsw_usage struct {
	Total, Avail, Used uint64
}

func (self *Swap) Get() error {
	sw_usage := xsw_usage{}

	if err := sysctlbyname("vm.swapusage", &sw_usage); err != nil {
		return err
	}

	self.Total = sw_usage.Total
	self.Used = sw_usage.Used
	self.Free = sw_usage.Avail

	return nil
}

func (self *FileSystemList) Get() error {
	num, err := getfsstat(nil, C.MNT_NOWAIT)
	if num < 0 {
		return err
	}

	buf := make([]syscall.Statfs_t, num)

	num, err = getfsstat(buf, C.MNT_NOWAIT)
	if err != nil {
		return err
	}

	fslist := make([]FileSystem, 0, num)

	for i := 0; i < num; i++ {
		fs := FileSystem{}

		fs.DirName = bytePtrToString(&buf[i].Mntonname[0])
		fs.DevName = bytePtrToString(&buf[i].Mntfromname[0])
		fs.SysTypeName = bytePtrToString(&buf[i].Fstypename[0])

		fslist = append(fslist, fs)
	}

	self.List = fslist

	return err
}

func vm_info(vmstat *C.vm_statistics_data_t) error {
	var count C.mach_msg_type_number_t = C.HOST_VM_INFO_COUNT

	status := C.host_statistics(
		C.host_t(C.mach_host_self()),
		C.HOST_VM_INFO,
		C.host_info_t(unsafe.Pointer(vmstat)),
		&count)

	if status != C.KERN_SUCCESS {
		return fmt.Errorf("host_statistics=%d", status)
	}

	return nil
}

//generic Sysctl buffer unmarshalling
func sysctlbyname(name string, data interface{}) (err error) {
	val, err := syscall.Sysctl(name)
	if err != nil {
		return err
	}

	buf := []byte(val)

	switch v := data.(type) {
	case *uint64:
		*v = *(*uint64)(unsafe.Pointer(&buf[0]))
		return
	}

	bbuf := bytes.NewBuffer([]byte(val))
	return binary.Read(bbuf, binary.LittleEndian, data)
}

//syscall.Getfsstat() wrapper is broken, roll our own to workaround.
func getfsstat(buf []syscall.Statfs_t, flags int) (n int, err error) {
	var ptr uintptr
	var size uintptr

	if len(buf) > 0 {
		ptr = uintptr(unsafe.Pointer(&buf[0]))
		size = unsafe.Sizeof(buf[0]) * uintptr(len(buf))
	} else {
		ptr = uintptr(0)
		size = uintptr(0)
	}

	trap := uintptr(syscall.SYS_GETFSSTAT64)
	ret, _, errno := syscall.Syscall(trap, ptr, size, uintptr(flags))

	n = int(ret)
	if errno != 0 {
		err = errno
	}

	return
}
