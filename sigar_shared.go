package sigar

import (
	"time"
)

func (pc *ProcCpu) Get(pid int) error { //nolint:staticcheck
	if pc.cache == nil {
		pc.cache = make(map[int]ProcCpu)
	}
	prevProcCpu := pc.cache[pid]

	procTime := &ProcTime{}
	if err := procTime.Get(pid); err != nil {
		return err
	}
	pc.StartTime = procTime.StartTime
	pc.User = procTime.User
	pc.Sys = procTime.Sys
	pc.Total = procTime.Total

	pc.LastTime = uint64(time.Now().UnixNano() / int64(time.Millisecond))
	pc.cache[pid] = *pc

	if prevProcCpu.LastTime == 0 {
		time.Sleep(100 * time.Millisecond)
		return pc.Get(pid)
	}

	pc.Percent = float64(pc.Total-prevProcCpu.Total) / float64(pc.LastTime-prevProcCpu.LastTime)
	return nil
}
