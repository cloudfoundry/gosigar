package sigar

import (
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("gosigar: not implemented")

type Sigar interface {
	CollectCpuStats(collectionInterval time.Duration) (<-chan Cpu, chan<- struct{})
	GetLoadAverage() (LoadAverage, error)
	GetMem() (Mem, error)
	GetMemIgnoringCGroups() (Mem, error)
	GetSwap() (Swap, error)
	GetFileSystemUsage(string) (FileSystemUsage, error)
}

type Cpu struct {
	User    uint64
	Nice    uint64
	Sys     uint64
	Idle    uint64
	Wait    uint64
	Irq     uint64
	SoftIrq uint64
	Stolen  uint64
}

func (c *Cpu) Total() uint64 {
	return c.User + c.Nice + c.Sys + c.Idle +
		c.Wait + c.Irq + c.SoftIrq + c.Stolen
}

func (c *Cpu) Delta(other Cpu) Cpu {
	return Cpu{
		User:    c.User - other.User,
		Nice:    c.Nice - other.Nice,
		Sys:     c.Sys - other.Sys,
		Idle:    c.Idle - other.Idle,
		Wait:    c.Wait - other.Wait,
		Irq:     c.Irq - other.Irq,
		SoftIrq: c.SoftIrq - other.SoftIrq,
		Stolen:  c.Stolen - other.Stolen,
	}
}

type LoadAverage struct {
	One, Five, Fifteen float64
}

type Uptime struct {
	Length float64
}

type Mem struct {
	Total      uint64
	Used       uint64
	Free       uint64
	ActualFree uint64
	ActualUsed uint64
}

type Swap struct {
	Total uint64
	Used  uint64
	Free  uint64
}

type CpuList struct {
	List []Cpu
}

type FileSystem struct {
	DirName     string
	DevName     string
	TypeName    string
	SysTypeName string
	Options     string
	Flags       uint32
}

type FileSystemList struct {
	List []FileSystem
}

type FileSystemUsage struct {
	Total     uint64
	Used      uint64
	Free      uint64
	Avail     uint64
	Files     uint64
	FreeFiles uint64
}

type ProcList struct {
	List []int
}

type RunState byte

const (
	RunStateSleep   = 'S'
	RunStateRun     = 'R'
	RunStateStop    = 'T'
	RunStateZombie  = 'Z'
	RunStateIdle    = 'D'
	RunStateUnknown = '?'
)

type ProcState struct {
	Name      string
	State     RunState
	Ppid      int
	Tty       int
	Priority  int
	Nice      int
	Processor int
}

type ProcMem struct {
	Size        uint64
	Resident    uint64
	Share       uint64
	MinorFaults uint64
	MajorFaults uint64
	PageFaults  uint64
}

type ProcTime struct {
	StartTime uint64
	User      uint64
	Sys       uint64
	Total     uint64
}

type ProcCpu struct {
	ProcTime
	LastTime uint64
	Percent  float64
	cache    map[int]ProcCpu
}

type ProcArgs struct {
	List []string
}

type ProcExe struct {
	Name string
	Cwd  string
	Root string
}
