// Copyright (c) 2012 VMware, Inc.

package sigar

import (
	"time"
)

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

func (self *Cpu) Total() uint64 {
	return self.User + self.Nice + self.Sys + self.Idle +
		self.Wait + self.Irq + self.SoftIrq + self.Stolen
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

type ProcArgs struct {
	List []string
}

type ProcExe struct {
	Name string
	Cwd  string
	Root string
}

func CollectCpuStats(duration time.Duration) (<-chan Cpu, chan<- struct{}) {
	values := make(chan Cpu)
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(duration)

		var oldCpuUsage, cpuUsage Cpu

		for {
			select {
			case <-ticker.C:
				oldCpuUsage = cpuUsage
				cpuUsage.Get()

				// Make sure we don't block if consumer does not read from values channel
				// Without default the process will fail with goroutines deadlock
				select {
				case values <- cpuUsage.delta(oldCpuUsage):
				default:
				}

			case <-stop:
				return
			}
		}
	}()

	return values, stop
}

func (self Cpu) delta(other Cpu) Cpu {
	return Cpu{
		User:    self.User - other.User,
		Nice:    self.Nice - other.Nice,
		Sys:     self.Sys - other.Sys,
		Idle:    self.Idle - other.Idle,
		Wait:    self.Wait - other.Wait,
		Irq:     self.Irq - other.Irq,
		SoftIrq: self.SoftIrq - other.SoftIrq,
		Stolen:  self.Stolen - other.Stolen,
	}
}
