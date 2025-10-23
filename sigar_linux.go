package sigar

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	MaxUint64 = ^uint64(0)
	// UnlimitedMemorySize defines the bytes size when memory limit is not set (2 ^ 63 - 4096)
	UnlimitedMemorySize = "9223372036854771712"
)

var system struct {
	ticks uint64
	btime uint64
}

var Procd string
var Etcd string
var Sysd1 string
var Sysd2 string

// Files in system directories used here
//   - Procd
//       - /stat
//       - /meminfo
//       - /self/cgroup | 'grep :memory:' | split ':' | last => cgroup
//       - /self/cgroup | 'grep ::'       | split ':' | last => cgroup/fallback
//       - /self/mounts
//   - Sysd1 (cgroup v1)
//       - memory/<cgroup>/memory.limit_in_bytes
//       - memory/<cgroup>/memory.stat
//   - Sysd2 (cgroup v2)
//	 - <cgroup>/memory.high
//	 - <cgroup>/memory.current
//	 - <cgroup>/memory.swap.current
//
// While Procd is fixed `/proc` the `Sysd*` directories are
// dynamic. I.e. while there are semi-standard mount points for the
// cgroup controllers, this is just convention. They can be mounted
// anywhere. The file `/proc/self/mounts` contains the information we
// need.

func init() {
	system.ticks = 100 // C.sysconf(C._SC_CLK_TCK)

	Procd = "/proc"
	Etcd = "/etc"
	Sysd1 = ""
	Sysd2 = ""

	determineControllerMounts(&Sysd1, &Sysd2)

	// Fallbacks for cgroup controller mount points if nothing was
	// found in /proc/self/mounts
	if Sysd1 == "" {
		Sysd1 = "/sys/fs/cgroup/memory"
	}
	if Sysd2 == "" {
		Sysd2 = "/sys/fs/cgroup/unified"
	}

	// grab system boot time
	readFile(Procd+"/stat", func(line string) bool { //nolint:errcheck
		if strings.HasPrefix(line, "btime") {
			system.btime, _ = strtoull(line[6:]) //nolint:errcheck
			return false                         // stop reading
		}
		return true
	})
}

func (la *LoadAverage) Get() error { //nolint:staticcheck
	line, err := os.ReadFile(Procd + "/loadavg")
	if err != nil {
		return nil
	}

	fields := strings.Fields(string(line))

	la.One, _ = strconv.ParseFloat(fields[0], 64)     //nolint:errcheck
	la.Five, _ = strconv.ParseFloat(fields[1], 64)    //nolint:errcheck
	la.Fifteen, _ = strconv.ParseFloat(fields[2], 64) //nolint:errcheck

	return nil
}

func (u *Uptime) Get() error { //nolint:staticcheck
	sysinfo := syscall.Sysinfo_t{}

	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return err
	}

	u.Length = float64(sysinfo.Uptime)

	return nil
}

func (m *Mem) Get() error { //nolint:staticcheck
	return m.get(false)
}

func (m *Mem) GetIgnoringCGroups() error { //nolint:staticcheck
	return m.get(true)
}

func (m *Mem) get(ignoreCGroups bool) error { //nolint:staticcheck
	var available = MaxUint64
	var buffers, cached uint64
	table := map[string]*uint64{
		"MemTotal":     &m.Total,
		"MemFree":      &m.Free,
		"MemAvailable": &available,
		"Buffers":      &buffers,
		"Cached":       &cached,
	}

	if err := parseMeminfo(table); err != nil {
		return err
	}

	if available == MaxUint64 {
		m.ActualFree = m.Free + buffers + cached
	} else {
		m.ActualFree = available
	}

	m.Used = m.Total - m.Free
	m.ActualUsed = m.Total - m.ActualFree

	if ignoreCGroups {
		return nil
	}

	// Instead of detecting if this code is run within a container
	// or not (*), we simply attempt to retrieve the cgroup
	// information about memory limits and usage and if present
	// incorporate them into the results.
	//
	// 0. If we are unable to determine the Cgroup for the process
	//    we ignore it and stay with the host data.
	//
	// 1. If the cgroup limit is not available we ignore it and
	//    stay with the host data.
	//
	// 2. Note that we are taking the smaller of host total and
	//    cgroup limit, as the safer value for the total. The
	//    reason here is that there are Linux systems which report
	//    something like 8 EiB (Exa!) (**) as the cgroup limit, on
	//    systems which have only 64 GiB (Giga) of physical RAM.
	//
	// (*) There does not seem to be a truly reliable and portable
	//     means of detecting execution inside a container vs
	//     outside. Between all the platforms (macOS, linux,
	//     windows), and container runtimes (docker, lxc, oci, ...).
	//
	// (**) The exact value actually is 2^63 - 4096, i.e
	//	8 EiB - 4 KiB.  This is, as far as is known, the
	//	maximum limit of the Linux virtual memory system.

	var cgroup string
	if err := determineSelfCgroup(&cgroup); err != nil {
		// Unable to determine process' Cgroup
		return nil
	}

	cgroupLimit, err := determineMemoryLimit(cgroup)
	// (x) If the limit is not available or bogus we keep the host data as limit.

	if err == nil && cgroupLimit < m.Total {
		// See (2) above why only a cgroup limit less than the
		// host total is accepted as the new total available
		// memory in the cgroup.
		m.Total = cgroupLimit
	}

	rss, err := determineMemoryUsage(cgroup)

	if err != nil {
		return nil
	}

	swap, err := determineSwapUsage(cgroup)
	if err != nil {
		// Swap information is optional. I.e. the kernel may
		// have swap accounting disabled.  Because of this any
		// kind of trouble determining the swap usage is
		// mapped to `no swap used`. This allows us to limp
		// on with some inaccuracies, instead of aborting.
		swap = 0
	}

	m.Used = rss + swap
	m.Free = m.Total - m.Used

	m.ActualUsed = m.Used
	m.ActualFree = m.Free

	return nil
}

func (s *Swap) Get() error { //nolint:staticcheck
	table := map[string]*uint64{
		"SwapTotal": &s.Total,
		"SwapFree":  &s.Free,
	}

	if err := parseMeminfo(table); err != nil {
		return err
	}

	s.Used = s.Total - s.Free
	return nil
}

func (c *Cpu) Get() error { //nolint:staticcheck
	return readFile(Procd+"/stat", func(line string) bool {
		if len(line) > 4 && line[0:4] == "cpu " {
			parseCpuStat(c, line) //nolint:errcheck
			return false
		}
		return true
	})
}

func (cl *CpuList) Get() error { //nolint:staticcheck
	capacity := len(cl.List)
	if capacity == 0 {
		capacity = 4
	}
	list := make([]Cpu, 0, capacity)

	err := readFile(Procd+"/stat", func(line string) bool {
		if len(line) > 3 && line[0:3] == "cpu" && line[3] != ' ' {
			cpu := Cpu{}
			parseCpuStat(&cpu, line) //nolint:errcheck
			list = append(list, cpu)
		}
		return true
	})

	cl.List = list

	return err
}

func (fsl *FileSystemList) Get() error { //nolint:staticcheck
	src := Etcd + "/mtab"
	if _, err := os.Stat(src); err != nil {
		src = Procd + "/mounts"
	}
	capacity := len(fsl.List)
	if capacity == 0 {
		capacity = 10
	}
	fslist := make([]FileSystem, 0, capacity)

	err := readFile(src, func(line string) bool {
		fields := strings.Fields(line)

		fs := FileSystem{}
		fs.DevName = fields[0]
		fs.DirName = fields[1]
		fs.SysTypeName = fields[2]
		fs.Options = fields[3]

		fslist = append(fslist, fs)

		return true
	})

	fsl.List = fslist

	return err
}

func (pl *ProcList) Get() error { //nolint:staticcheck
	dir, err := os.Open(Procd)
	if err != nil {
		return err
	}
	defer dir.Close() //nolint:errcheck

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

func (ps *ProcState) Get(pid int) error { //nolint:staticcheck
	contents, err := readProcFile(pid, "stat")
	if err != nil {
		return err
	}

	fields := strings.Fields(string(contents))

	ps.Name = fields[1][1 : len(fields[1])-1] // strip ()'s

	ps.State = RunState(fields[2][0])

	ps.Ppid, _ = strconv.Atoi(fields[3]) //nolint:errcheck

	ps.Tty, _ = strconv.Atoi(fields[6]) //nolint:errcheck

	ps.Priority, _ = strconv.Atoi(fields[17]) //nolint:errcheck

	ps.Nice, _ = strconv.Atoi(fields[18]) //nolint:errcheck

	ps.Processor, _ = strconv.Atoi(fields[38]) //nolint:errcheck

	return nil
}

func (pm *ProcMem) Get(pid int) error { //nolint:staticcheck
	contents, err := readProcFile(pid, "statm")
	if err != nil {
		return err
	}

	fields := strings.Fields(string(contents))

	size, _ := strtoull(fields[0]) //nolint:errcheck
	pm.Size = size << 12

	rss, _ := strtoull(fields[1]) //nolint:errcheck
	pm.Resident = rss << 12

	share, _ := strtoull(fields[2]) //nolint:errcheck
	pm.Share = share << 12

	contents, err = readProcFile(pid, "stat")
	if err != nil {
		return err
	}

	fields = strings.Fields(string(contents))

	pm.MinorFaults, _ = strtoull(fields[10]) //nolint:errcheck
	pm.MajorFaults, _ = strtoull(fields[12]) //nolint:errcheck
	pm.PageFaults = pm.MinorFaults + pm.MajorFaults

	return nil
}

func (pt *ProcTime) Get(pid int) error { //nolint:staticcheck
	contents, err := readProcFile(pid, "stat")
	if err != nil {
		return err
	}

	fields := strings.Fields(string(contents))

	user, _ := strtoull(fields[13]) //nolint:errcheck
	sys, _ := strtoull(fields[14])  //nolint:errcheck
	// convert to millis
	pt.User = user * (1000 / system.ticks)
	pt.Sys = sys * (1000 / system.ticks)
	pt.Total = pt.User + pt.Sys

	// convert to millis
	pt.StartTime, _ = strtoull(fields[21]) //nolint:errcheck
	pt.StartTime /= system.ticks
	pt.StartTime += system.btime
	pt.StartTime *= 1000

	return nil
}

func (pa *ProcArgs) Get(pid int) error { //nolint:staticcheck
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

func (pe *ProcExe) Get(pid int) error { //nolint:staticcheck
	fields := map[string]*string{
		"exe":  &pe.Name,
		"cwd":  &pe.Cwd,
		"root": &pe.Root,
	}

	for name, field := range fields {
		val, err := os.Readlink(procFileName(pid, name))

		if err != nil {
			return err
		}

		*field = val
	}

	return nil
}

func determineSwapUsage(cgroup string) (uint64, error) {
	// Check v2 over v1
	usageAsString, err := os.ReadFile(Sysd2 + cgroup + "/memory.swap.current")
	if err == nil {
		return strtoull(strings.Split(string(usageAsString), "\n")[0])
	}

	var swap uint64
	table := map[string]*uint64{
		"swap": &swap,
	}

	err, found := parseCgroupMeminfo(Sysd1+cgroup, table)
	if err == nil {
		if !found {
			// If no data was found, simply claim `zero swap used`.
			return 0, errors.New("no data found")
		}
		return swap, nil
	}

	return 0, err
}

func determineMemoryUsage(cgroup string) (uint64, error) {
	// Check v2 over v1
	usageAsString, err := os.ReadFile(Sysd2 + cgroup + "/memory.current")
	if err == nil {
		return strtoull(strings.Split(string(usageAsString), "\n")[0])
	}

	var rss uint64
	table := map[string]*uint64{
		"total_rss": &rss,
	}

	err, found := parseCgroupMeminfo(Sysd1+cgroup, table)
	if err == nil {
		if !found {
			return 0, errors.New("no data found")
		}
		return rss, nil
	}

	return 0, err
}

func determineMemoryLimit(cgroup string) (uint64, error) {
	// Check v2 over v1
	limitAsString, err := os.ReadFile(Sysd2 + cgroup + "/memory.high")
	if err == nil {
		val := strings.Split(string(limitAsString), "\n")[0]
		if val == "max" {
			return 0, errors.New("no limit")
			// See (x) in the caller where this keeps the host's self.Total.
		}
		return strtoull(val)
	}

	limitAsString, err = os.ReadFile(Sysd1 + cgroup + "/memory.limit_in_bytes")
	if string(limitAsString) != UnlimitedMemorySize && err == nil {
		return strtoull(strings.Split(string(limitAsString), "\n")[0])
	}

	var limit uint64
	table := map[string]*uint64{
		"hierarchical_memory_limit": &limit,
	}

	err, found := parseCgroupMeminfo(Sysd1+cgroup, table)
	if err == nil {
		if !found {
			// If no data was found, simply claim `zero limit set`.
			return 0, errors.New("no hierarchical memory limit found")
		}
		return limit, nil
	}

	return 0, err
}

func determineSelfCgroup(cgroup *string) error {
	// - /proc/self/cgroup
	//   Expected line syntax - id:tag:path
	//   Three fields required in each line.

	// Look for a cgroup v1 memory controller first
	err := readFile(Procd+"/self/cgroup", func(line string) bool {
		fields := strings.Split(line, ":")
		// Match: `*:memory:/path`
		if len(fields) < 3 {
			return true
		}
		if fields[1] == "memory" {
			*cgroup = strings.Trim(fields[len(fields)-1], " ")
		}
		return true
	})
	if err != nil {
		return err
	}
	if *cgroup != "" {
		return nil
	}

	// Fall back to a cgroup v2 memory controller
	err = readFile(Procd+"/self/cgroup", func(line string) bool {
		fields := strings.Split(line, ":")
		// Match: `0::/path`
		if len(fields) < 3 {
			return true
		}
		if (fields[0] == "0") && (fields[1] == "") {
			*cgroup = strings.Trim(fields[len(fields)-1], " ")
		}
		return true
	})
	if err != nil {
		return err
	}
	if *cgroup != "" {
		return nil
	}

	return errors.New("unable to determine control group")
}

func parseMeminfo(table map[string]*uint64) error {
	return readFile(Procd+"/meminfo", func(line string) bool {
		fields := strings.Split(line, ":")

		if ptr := table[fields[0]]; ptr != nil {
			num := strings.TrimLeft(fields[1], " ")
			val, err := strtoull(strings.Fields(num)[0])
			if err == nil {
				*ptr = val * 1024
			}
		}

		return true
	})
}

func parseCgroupMeminfo(cgroupDir string, table map[string]*uint64) (error, bool) {
	var found bool
	err := readFile(cgroupDir+"/memory.stat", func(line string) bool {
		fields := strings.Split(line, " ")
		if ptr := table[fields[0]]; ptr != nil {
			num := strings.TrimLeft(fields[1], " ")
			val, err := strtoull(strings.Fields(num)[0])
			if err == nil {
				*ptr = val
				found = true
			}
		}

		return true
	})
	if err != nil {
		return err, false
	}
	return nil, found
}

func parseCpuStat(self *Cpu, line string) error {
	fields := strings.Fields(line)

	self.User, _ = strtoull(fields[1])    //nolint:errcheck
	self.Nice, _ = strtoull(fields[2])    //nolint:errcheck
	self.Sys, _ = strtoull(fields[3])     //nolint:errcheck
	self.Idle, _ = strtoull(fields[4])    //nolint:errcheck
	self.Wait, _ = strtoull(fields[5])    //nolint:errcheck
	self.Irq, _ = strtoull(fields[6])     //nolint:errcheck
	self.SoftIrq, _ = strtoull(fields[7]) //nolint:errcheck
	self.Stolen, _ = strtoull(fields[8])  //nolint:errcheck

	return nil
}

func readFile(file string, handler func(string) bool) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(bytes.NewBuffer(contents))

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if !handler(string(line)) {
			break
		}
	}

	return nil
}

func strtoull(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}

func procFileName(pid int, name string) string {
	return Procd + "/" + strconv.Itoa(pid) + "/" + name
}

func readProcFile(pid int, name string) ([]byte, error) {
	path := procFileName(pid, name)
	contents, err := os.ReadFile(path)

	if err != nil {
		if perr, ok := err.(*os.PathError); ok {
			if perr.Err == syscall.ENOENT {
				return nil, syscall.ESRCH
			}
		}
	}

	return contents, err
}

func determineControllerMounts(sysd1, sysd2 *string) {
	// grab cgroup controller mount points
	readFile(Procd+"/self/mounts", func(line string) bool { //nolint:errcheck

		// Entries have the form `device path type options`.
		// The elements are separated by single spaces.
		//
		// v2: `path` element of entry fulfilling `type == "cgroup2"`.
		// v1: `path` element of entry fulfilling `type == "cgroup" && options ~ "memory"`
		//
		// NOTE: The `device` column can be anything. It
		// cannot be used to pare down the set of entries
		// going into the full check.

		fields := strings.Split(line, " ")
		if len(fields) < 4 {
			return true
		}

		mpath := fields[1]
		mtype := fields[2]
		moptions := fields[3]

		if mtype == "cgroup2" {
			if *sysd2 != "" {
				return true
			}
			*sysd2 = mpath
			return true
		}
		if mtype == "cgroup" {
			options := strings.Split(moptions, ",")
			if stringSliceContains(options, "memory") {
				if *sysd1 != "" {
					return true
				}
				*sysd1 = mpath
				return true
			}
		}
		return true
	})
}

func stringSliceContains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
