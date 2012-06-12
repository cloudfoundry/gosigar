// Copyright (c) 2012 VMware, Inc.

package sigar

/*
#include <stdlib.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <mach/mach_init.h>
#include <mach/mach_host.h>
#include <mach/host_info.h>
#include <libproc.h>
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

func (self *ProcList) Get() error {
	n := C.proc_listpids(C.PROC_ALL_PIDS, 0, nil, 0)
	if n <= 0 {
		return syscall.EINVAL
	}
	buf := make([]byte, n)
	n = C.proc_listpids(C.PROC_ALL_PIDS, 0, unsafe.Pointer(&buf[0]), n)
	if n <= 0 {
		return syscall.ENOMEM
	}

	var pid int32
	num := int(n) / binary.Size(pid)
	list := make([]int, 0, num)
	bbuf := bytes.NewBuffer(buf)

	for i := 0; i < num; i++ {
		if err := binary.Read(bbuf, binary.LittleEndian, &pid); err != nil {
			return err
		}
		if pid == 0 {
			continue
		}

		list = append(list, int(pid))
	}

	self.List = list

	return nil
}

func (self *ProcState) Get(pid int) error {
	info := C.struct_proc_taskallinfo{}

	if err := task_info(pid, &info); err != nil {
		return err
	}

	self.Name = C.GoString(&info.pbsd.pbi_comm[0])

	switch info.pbsd.pbi_status {
	case C.SIDL:
		self.State = RunStateIdle
	case C.SRUN:
		self.State = RunStateRun
	case C.SSLEEP:
		self.State = RunStateSleep
	case C.SSTOP:
		self.State = RunStateStop
	case C.SZOMB:
		self.State = RunStateZombie
	default:
		self.State = RunStateUnknown
	}

	self.Ppid = int(info.pbsd.pbi_ppid)

	self.Tty = int(info.pbsd.e_tdev)

	self.Priority = int(info.ptinfo.pti_priority)

	self.Nice = int(info.pbsd.pbi_nice)

	return nil
}

func (self *ProcMem) Get(pid int) error {
	info := C.struct_proc_taskallinfo{}

	if err := task_info(pid, &info); err != nil {
		return err
	}

	self.Size = uint64(info.ptinfo.pti_virtual_size)
	self.Resident = uint64(info.ptinfo.pti_resident_size)
	self.PageFaults = uint64(info.ptinfo.pti_faults)

	return nil
}

func (self *ProcTime) Get(pid int) error {
	info := C.struct_proc_taskallinfo{}

	if err := task_info(pid, &info); err != nil {
		return err
	}

	self.User =
		uint64(info.ptinfo.pti_total_user) / uint64(time.Millisecond)

	self.Sys =
		uint64(info.ptinfo.pti_total_system) / uint64(time.Millisecond)

	self.Total = self.User + self.Sys

	self.StartTime = (uint64(info.pbsd.pbi_start_tvsec) * 1000) +
		(uint64(info.pbsd.pbi_start_tvusec) / 1000)

	return nil
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

// generic Sysctl buffer unmarshalling
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

// syscall.Getfsstat() wrapper is broken, roll our own to workaround.
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

func task_info(pid int, info *C.struct_proc_taskallinfo) error {
	size := C.int(unsafe.Sizeof(*info))
	ptr := unsafe.Pointer(info)

	n := C.proc_pidinfo(C.int(pid), C.PROC_PIDTASKALLINFO, 0, ptr, size)
	if n != size {
		return syscall.ENOMEM
	}

	return nil
}
