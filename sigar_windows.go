package sigar

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/cloudfoundry/gosigar/sys/windows"
)

var (
	kernel32DLL = syscall.MustLoadDLL("kernel32")

	procGetDiskFreeSpace     = kernel32DLL.MustFindProc("GetDiskFreeSpaceW")
	procGetSystemTimes       = kernel32DLL.MustFindProc("GetSystemTimes")
	procGetTickCount64       = kernel32DLL.MustFindProc("GetTickCount64")
	procGlobalMemoryStatusEx = kernel32DLL.MustFindProc("GlobalMemoryStatusEx")

	// processQueryLimitedInfoAccess is set to PROCESS_QUERY_INFORMATION for Windows
	// 2003 and XP where PROCESS_QUERY_LIMITED_INFORMATION is unknown. For all newer
	// OS versions it is set to PROCESS_QUERY_LIMITED_INFORMATION.
	processQueryLimitedInfoAccess = windows.PROCESS_QUERY_LIMITED_INFORMATION
)

func (la *LoadAverage) Get() error { //nolint:staticcheck
	return ErrNotImplemented
}

func (u *Uptime) Get() error { //nolint:staticcheck
	r1, _, e1 := syscall.SyscallN(procGetTickCount64.Addr())
	if e1 != 0 {
		return error(e1)
	}
	u.Length = (time.Duration(r1) * time.Millisecond).Seconds()
	return nil
}

type memorystatusex struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func (m *Mem) Get() error { //nolint:staticcheck
	var x memorystatusex
	x.Length = uint32(unsafe.Sizeof(x))
	r1, _, e1 := syscall.SyscallN(procGlobalMemoryStatusEx.Addr(),
		uintptr(unsafe.Pointer(&x)),
	)
	if err := checkErrno(r1, e1); err != nil {
		return fmt.Errorf("GlobalMemoryStatusEx: %s", err)
	}
	m.Total = x.TotalPhys
	m.Free = x.AvailPhys
	m.ActualFree = m.Free
	m.Used = m.Total - m.Free
	m.ActualUsed = m.Used
	return nil
}

func (m *Mem) GetIgnoringCGroups() error { //nolint:staticcheck
	return m.Get()
}

func (s *Swap) Get() error { //nolint:staticcheck
	memoryStatusEx, err := windows.GlobalMemoryStatusEx()
	if err != nil {
		return fmt.Errorf("GlobalMemoryStatusEx: %w", err)
	}

	s.Total = memoryStatusEx.TotalPageFile
	s.Free = memoryStatusEx.AvailPageFile
	s.Used = s.Total - s.Free
	return nil
}

func (c *Cpu) Get() error { //nolint:staticcheck
	var (
		idleTime   syscall.Filetime
		kernelTime syscall.Filetime // Includes kernel and idle time.
		userTime   syscall.Filetime
	)
	r1, _, e1 := syscall.SyscallN(procGetSystemTimes.Addr(),
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if err := checkErrno(r1, e1); err != nil {
		return fmt.Errorf("GetSystemTimes: %s", err)
	}

	c.Idle = uint64(idleTime.Nanoseconds())
	c.Sys = uint64(kernelTime.Nanoseconds()) - c.Idle
	c.User = uint64(userTime.Nanoseconds())
	return nil
}

func (cl *CpuList) Get() error { //nolint:staticcheck
	return ErrNotImplemented
}

func (fsl *FileSystemList) Get() error { //nolint:staticcheck
	return ErrNotImplemented
}

func (pl *ProcList) Get() error { //nolint:staticcheck
	return ErrNotImplemented
}

func (ps *ProcState) Get(pid int) error { //nolint:staticcheck
	return ErrNotImplemented
}

func (pm *ProcMem) Get(pid int) error { //nolint:staticcheck
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("OpenProcess failed for pid=%v %w", pid, err)
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck

	counters, err := windows.GetProcessMemoryInfo(handle)
	if err != nil {
		return fmt.Errorf("GetProcessMemoryInfo failed for pid=%v %w", pid, err)
	}

	pm.Resident = uint64(counters.WorkingSetSize)
	pm.Size = uint64(counters.PrivateUsage)
	return nil
}

func (pt *ProcTime) Get(pid int) error { //nolint:staticcheck
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("OpenProcess failed for pid=%v %w", pid, err)
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck

	var CPU syscall.Rusage
	if err := syscall.GetProcessTimes(handle, &CPU.CreationTime, &CPU.ExitTime, &CPU.KernelTime, &CPU.UserTime); err != nil {
		return fmt.Errorf("GetProcessTimes failed for pid=%v %w", pid, err)
	}

	// Windows epoch times are expressed as time elapsed since midnight on
	// January 1, 1601, at Greenwich, England. This converts the Filetime to
	// unix epoch in milliseconds.
	pt.StartTime = uint64(CPU.CreationTime.Nanoseconds() / 1e6)

	// Convert to millis.
	pt.User = uint64(windows.FiletimeToDuration(&CPU.UserTime).Nanoseconds() / 1e6)
	pt.Sys = uint64(windows.FiletimeToDuration(&CPU.KernelTime).Nanoseconds() / 1e6)
	pt.Total = pt.User + pt.Sys

	return nil
}

func (pa *ProcArgs) Get(pid int) error { //nolint:staticcheck
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("OpenProcess failed for pid=%v %w", pid, err)
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck
	pbi, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return fmt.Errorf("NtQueryProcessBasicInformation failed for pid=%v %w", pid, err)
	}
	userProcParams, err := windows.GetUserProcessParams(handle, pbi)
	if err != nil {
		return nil
	}
	argsW, err := windows.ReadProcessUnicodeString(handle, &userProcParams.CommandLine)
	if err == nil {
		pa.List, err = windows.ByteSliceToStringSlice(argsW)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pe *ProcExe) Get(pid int) error { //nolint:staticcheck
	return ErrNotImplemented
}

func (fs *FileSystemUsage) Get(path string) error { //nolint:staticcheck
	root, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("FileSystemUsage (%s): %s", path, err)
	}

	var (
		SectorsPerCluster uint32
		BytesPerSector    uint32
		// NumberOfFreeClusters available to the user associated with the calling thread.
		NumberOfFreeClusters uint32
		// TotalNumberOfClusters available to the user associated with the calling thread.
		TotalNumberOfClusters uint32
	)
	r1, _, e1 := syscall.SyscallN(procGetDiskFreeSpace.Addr(),
		uintptr(unsafe.Pointer(root)),
		uintptr(unsafe.Pointer(&SectorsPerCluster)),
		uintptr(unsafe.Pointer(&BytesPerSector)),
		uintptr(unsafe.Pointer(&NumberOfFreeClusters)),
		uintptr(unsafe.Pointer(&TotalNumberOfClusters)),
	)
	if err := checkErrno(r1, e1); err != nil {
		return fmt.Errorf("FileSystemUsage (%s): %s", path, err)
	}

	m := uint64(SectorsPerCluster * BytesPerSector / 1024)
	fs.Total = uint64(TotalNumberOfClusters) * m
	fs.Free = uint64(NumberOfFreeClusters) * m
	fs.Avail = fs.Free
	fs.Used = fs.Total - fs.Free

	return nil
}

func checkErrno(r1 uintptr, e1 error) error {
	if r1 == 0 {
		var e syscall.Errno
		if errors.As(e1, &e) && e != 0 {
			return e1
		}
		return syscall.EINVAL
	}
	return nil
}
