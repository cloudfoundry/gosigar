// Copyright (c) 2012 VMware, Inc.

package sigar

func (self *LoadAverage) Get() error {
	return ErrNotImplemented
}

func (u *Uptime) Get() error {
	return ErrNotImplemented
}

func (m *Mem) Get() error {
	return ErrNotImplemented
}

func (s *Swap) Get() error {
	return ErrNotImplemented
}

func (c *Cpu) Get() error {
	return ErrNotImplemented
}

func (self *CpuList) Get() error {
	return ErrNotImplemented
}

func (self *FileSystemList) Get() error {
	return ErrNotImplemented
}

func (self *ProcList) Get() error {
	return ErrNotImplemented
}

func (self *ProcState) Get(pid int) error {
	return ErrNotImplemented
}

func (self *ProcMem) Get(pid int) error {
	return ErrNotImplemented
}

func (self *ProcTime) Get(pid int) error {
	return ErrNotImplemented
}

func (self *ProcArgs) Get(pid int) error {
	return ErrNotImplemented
}

func (self *ProcExe) Get(pid int) error {
	return ErrNotImplemented
}

func (fs *FileSystemUsage) Get(path string) error {
	return ErrNotImplemented
}

func checkErrno(r1 uintptr, e1 error) error {
	return ErrNotImplemented
}
