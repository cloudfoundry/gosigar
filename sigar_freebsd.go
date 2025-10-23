//go:build freebsd

package sigar

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	kernProc    = 14 // KERN_PROC in sysctl((3)
	kernProcPid = 1  // KERN_PROC_PID in sysctl(3)
)

// kinfo_proc as returned by kinfo_getproc() in libutil
// see also /usr/include/sys/user.h
type procInfo struct {
	Structsize   int32
	Layout       int32
	Args         int64
	Paddr        int64
	Addr         int64
	Tracep       int64
	Textvp       int64
	Fd           int64
	Vmspace      int64
	Wchan        int64
	Pid          int32
	Ppid         int32
	Pgid         int32
	Tpgid        int32
	Sid          int32
	Tsid         int32
	Jobc         [2]byte
	Spare_short1 [2]byte
	Tdev         int32
	Siglist      [16]byte
	Sigmask      [16]byte
	Sigignore    [16]byte
	Sigcatch     [16]byte
	Uid          int32
	Ruid         int32
	Svuid        int32
	Rgid         int32
	Svgid        int32
	Ngroups      [2]byte
	Spare_short2 [2]byte
	Groups       [64]byte
	Size         int64
	Rssize       int64
	Swrss        int64
	Tsize        int64
	Dsize        int64
	Ssize        int64
	Xstat        [2]byte
	Acflag       [2]byte
	Pctcpu       int32
	Estcpu       int32
	Slptime      int32
	Swtime       int32
	Cow          int32
	Runtime      int64
	Start        [16]byte
	Childtime    [16]byte
	Flag         int64
	Kiflag       int64
	Traceflag    int32
	Stat         [1]byte
	Nice         [1]byte
	Lock         [1]byte
	Rqindex      [1]byte
	Oncpu        [1]byte
	Lastcpu      [1]byte
	Ocomm        [17]byte
	Wmesg        [9]byte
	Login        [18]byte
	Lockname     [9]byte
	Comm         [20]byte
	Emul         [17]byte
	Sparestrings [68]byte
	Spareints    [36]byte
	Cr_flags     int32
	Jid          int32
	Numthreads   int32
	Tid          int32
	Pri          int32
	Rusage       [144]byte
	Rusage_ch    [144]byte
	Pcb          int64
	Kstack       int64
	Udata        int64
	Tdaddr       int64
	Spareptrs    [48]byte
	Spareint64s  [96]byte
	Sflag        int64
	Tdflags      int64
}

// See /usr/include/sys/proc.h
var (
	procStateIdle   = [1]uint8{0x1} // SIDL
	procStateRun    = [1]uint8{0x2} // SRUN
	procStateSleep  = [1]uint8{0x3} // SSLEEP
	procStateStop   = [1]uint8{0x4} // SSTOP
	procStateZombie = [1]uint8{0x5} // SZOMB
	procStateWait   = [1]uint8{0x6} // SWAIT
	procStateLock   = [1]uint8{0x7} // SLOCK
)

var (
	Procd string = "/proc"
)

type loadStruct struct {
	Ldavg  [3]uint32
	Fscale uint64
}

func getProcInfo(pid int) (procInfo, error) {
	var pinfo procInfo

	pinfoRaw, err := unix.SysctlRaw("kern", kernProc, kernProcPid, pid)
	if err != nil {
		return procInfo{}, err
	}
	pinfoReader := bytes.NewReader(pinfoRaw)
	err = binary.Read(pinfoReader, binary.LittleEndian, &pinfo)
	if err != nil {
		return procInfo{}, err
	}

	return pinfo, nil
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

func (u *Uptime) Get() error {
	var tv unix.Timeval
	boottimeRaw, err := unix.SysctlRaw("kern.boottime")
	if err != nil {
		return err
	}
	tv = *(*unix.Timeval)(unsafe.Pointer(&boottimeRaw[0]))
	u.Length = time.Since(time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)).Seconds()
	return nil
}

func (la *LoadAverage) Get() error {
	avgRaw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil {
		return err
	}
	avg := *(*loadStruct)(unsafe.Pointer(&avgRaw[0]))
	fscale := float64(avg.Fscale)

	la.One = float64(avg.Ldavg[0]) / fscale
	la.Five = float64(avg.Ldavg[1]) / fscale
	la.Fifteen = float64(avg.Ldavg[2]) / fscale

	return nil
}

func (pl *ProcList) Get() error {
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

	pl.List = list

	return nil
}

func (ps *ProcState) Get(pid int) error {
	pinfo, err := getProcInfo(pid)
	if err != nil {
		return err
	}
	ps.Name = string(bytes.Trim(pinfo.Comm[:], "\x00"))
	ps.Ppid = int(pinfo.Ppid)
	switch pinfo.Stat {
	case procStateRun:
		ps.State = RunStateRun
	case procStateIdle:
		ps.State = RunStateIdle
	case procStateSleep:
		ps.State = RunStateSleep
	case procStateStop:
		ps.State = RunStateStop
	case procStateZombie:
		ps.State = RunStateZombie
	default:
		ps.State = RunStateUnknown
	}

	return nil
}

func (fsl *FileSystemList) Get() error {
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
	fsl.List = fslist
	return nil
}

func (fs *FileSystemUsage) Get(path string) error {
	stat := unix.Statfs_t{}
	err := unix.Statfs(path, &stat)
	if err != nil {
		return err
	}

	bsize := stat.Bsize / 512

	fs.Total = (uint64(stat.Blocks) * uint64(bsize)) >> 1
	fs.Free = (uint64(stat.Bfree) * uint64(bsize)) >> 1
	fs.Avail = (uint64(stat.Bavail) * uint64(bsize)) >> 1
	fs.Used = fs.Total - fs.Free
	fs.Files = stat.Files
	fs.FreeFiles = uint64(stat.Ffree)

	return nil
}

func (pt *ProcTime) Get(pid int) error {
	rusage := unix.Rusage{}
	unix.Getrusage(pid, &rusage)
	pt.User = uint64(rusage.Utime.Nano() / 1e6)
	pt.Sys = uint64(rusage.Stime.Nano() / 1e6)
	pt.Total = pt.User + pt.Sys

	var tv unix.Timeval
	pinfo, err := getProcInfo(pid)
	if err != nil {
		return err
	}
	tv = *(*unix.Timeval)(unsafe.Pointer(&pinfo.Start[0]))
	pt.StartTime = (uint64(tv.Sec) * 1000) + (uint64(tv.Usec) / 1000)

	return nil
}

func (c *Cpu) Get() error {
	cpTime, err := unix.SysctlRaw("kern.cp_time")
	if err != nil {
		return err
	}

	cpuRaw := *(*cpuStat)(unsafe.Pointer(&cpTime[0]))

	c.User = uint64(cpuRaw.user)
	c.Nice = uint64(cpuRaw.nice)
	c.Sys = uint64(cpuRaw.sys)
	c.Irq = uint64(cpuRaw.irq)
	c.Idle = uint64(cpuRaw.idle)

	return nil
}

func (m *Mem) Get() error {
	var err error

	m.Total, err = unix.SysctlUint64("hw.physmem")
	if err != nil {
		return err
	}

	pageSize, err := unix.SysctlUint32("hw.pagesize")
	if err != nil {
		return err
	}

	freePages, err := unix.SysctlUint32("vm.stats.vm.v_free_count")
	if err != nil {
		return err
	}
	m.Free = uint64(freePages) * uint64(pageSize)

	m.Used = m.Total - m.Free
	return nil
}

func (m *Mem) GetIgnoringCGroups() error {
	return m.Get()
}

func (s *Swap) Get() error {
	var err error
	s.Total, err = unix.SysctlUint64("vm.swap_total")
	if err != nil {
		return err
	}

	return nil
}

type cpuStat struct {
	user int64
	nice int64
	sys  int64
	irq  int64
	idle int64
}

func (cl *CpuList) Get() error {
	cpTimes, err := unix.SysctlRaw("kern.cp_times")
	if err != nil {
		return err
	}

	// 5 values of 8 bytes (int64) per CPU
	ncpu := len(cpTimes) / 8 / 5

	cpulist := make([]Cpu, 0, ncpu)

	for i := 0; i < ncpu; i++ {
		cpuRaw := *(*cpuStat)(unsafe.Pointer(&cpTimes[i*8*5]))

		cpu := Cpu{}
		cpu.User = uint64(cpuRaw.user)
		cpu.Nice = uint64(cpuRaw.nice)
		cpu.Sys = uint64(cpuRaw.sys)
		cpu.Irq = uint64(cpuRaw.irq)
		cpu.Idle = uint64(cpuRaw.idle)

		cpulist = append(cpulist, cpu)
	}
	cl.List = cpulist

	return nil
}

func (pm *ProcMem) Get(pid int) error {
	pageSize, err := unix.SysctlUint32("hw.pagesize")
	if err != nil {
		return err
	}

	rusage := unix.Rusage{}
	unix.Getrusage(pid, &rusage)

	pm.Resident = (uint64(rusage.Ixrss) + uint64(rusage.Idrss)) * uint64(pageSize)
	pm.Share = uint64(rusage.Isrss) * uint64(pageSize)
	pm.Size = pm.Resident + pm.Share
	pm.MinorFaults = uint64(rusage.Minflt)
	pm.MajorFaults = uint64(rusage.Majflt)
	pm.PageFaults = pm.MinorFaults + pm.MajorFaults

	return nil
}

func (pa *ProcArgs) Get(pid int) error {
	contents, err := readProcFile(pid, "cmdline")
	if err != nil {
		return err
	}

	bbuf := bytes.NewBuffer(contents)

	var args []string

	for {
		arg, err := bbuf.ReadBytes(0)
		if err == io.EOF {
			break
		}
		args = append(args, string(chop(arg)))
	}

	pa.List = args

	return nil
}

func (pe *ProcExe) Get(pid int) error {
	var err error
	pe.Name, err = os.Readlink(procFileName(pid, "file"))
	if err != nil {
		return err
	}
	return nil
}
