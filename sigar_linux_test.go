package sigar

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Helpers. Create various system information files
var procd string
var etcd string

func setupFile(path, contents string) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	err := ioutil.WriteFile(path, []byte(contents), 0444)
	Expect(err).ToNot(HaveOccurred())
}

func cgroupSetup(contents string) {
	setupFile(procd+"/self/cgroup", contents+"\n")
}

func memLimitSetup1(cg, contents string) {
	setupFile(procd+"/memory"+cg+"/memory.limit_in_bytes", contents+"\n")
}

func memLimitSetup2(cg, contents string) {
	setupFile(procd+cg+"/memory.high", contents+"\n")
}

func memStatSetup(cg, contents string) {
	setupFile(procd+"/memory"+cg+"/memory.stat", contents+"\n")
}

func memStatWithSwap(cg string) {
	memStatSetup(cg, `total_rss 14108536832
swap 290564089`)
}

func memStatWithoutSwap(cg string) {
	memStatSetup(cg, `total_rss 14108536832`)
}

func memUsageSetup2(cg, contents string) {
	setupFile(procd+cg+"/memory.current", contents+"\n")
}

func swapUsageSetup1(cg, contents string) {
	setupFile(procd+"/memory"+cg+"/memory.stat", contents+"\n")
}

func swapUsageSetup2(cg, contents string) {
	setupFile(procd+cg+"/memory.swap.current", contents+"\n")
}

func memUsageWithSwap(cg string) {
	memUsageSetup2(cg, `14108536832`)
	swapUsageSetup2(cg, `290564089`)
}

func memUsageWithoutSwap(cg string) {
	memUsageSetup2(cg, `14108536832`)
}

func memInfoSetup(contents string) {
	setupFile(procd+"/meminfo", contents+"\n")
}

func memInfoWithMemAvailable() {
	memInfoSetup(`
MemTotal:       35008180 kB
MemFree:          487816 kB
MemAvailable:   20913400 kB
Buffers:          249244 kB
Cached:          5064684 kB
`)
}

func memInfoWithoutMemAvailable() {
	memInfoSetup(`
MemTotal:       35008180 kB
MemFree:          487816 kB
Buffers:          249244 kB
Cached:          5064684 kB
`)
}

func setupEtcMtab() {
	setupMountsFile(etcd + "/mtab")
}

func setupProcMounts() {
	setupMountsFile(procd + "/mounts")
}

func setupMountsFile(filePath string) {
	setupFile(filePath, `sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
	proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
	udev /dev devtmpfs rw,nosuid,relatime,size=949852k,nr_inodes=185646,mode=755 0 0
	devpts /dev/pts devpts rw,nosuid,noexec,relatime,gid=5,mode=620,ptmxmode=000 0 0
	tmpfs /run tmpfs rw,nosuid,noexec,relatime,size=204020k,mode=755 0 0
	/dev/sda1 / ext4 rw,noatime,errors=remount-ro,stripe=32753 0 0
	/dev/sdb1 /home ext4 rw,noatime,errors=remount-ro 0 0`)
}

var _ = Describe("sigarLinux", func() {
	BeforeEach(func() {
		var err error
		procd, err = ioutil.TempDir("", "sigarTests")
		Expect(err).ToNot(HaveOccurred())
		etcd, err = ioutil.TempDir("", "sigarTestsEtc")
		Expect(err).ToNot(HaveOccurred())
		// Can share the directory, no overlap in files used
		Procd = procd
		Etcd = etcd
		Sysd1 = procd + "/memory"
		Sysd2 = procd
	})

	AfterEach(func() {
		Procd = "/proc"
		Etcd = "/etc"
		Sysd1 = "/sys/fs/cgroup/unified"
		Sysd2 = "/sys/fs/cgroup/memory"

		err := os.RemoveAll(procd)
		Expect(err).ToNot(HaveOccurred())

		err = os.RemoveAll(etcd)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Initialization subset: Cgroup Controller Mountpoints", func() {
		Describe("No mounts", func() {
			var sys1, sys2 string

			BeforeEach(func() {
				setupFile(procd+"/self/mounts", `
device path type options
alpha /fox cgroup options
beta /dog jumping memory
cgroup /cat sleeping irrelevant
delta /carp
`)
			})

			It("it is a no-op", func() {
				determineControllerMounts(&sys1, &sys2)
				Expect(sys1).To(Equal(""))
				Expect(sys2).To(Equal(""))
			})
		})

		Describe("Mounts", func() {
			var sys1, sys2 string

			BeforeEach(func() {
				setupFile(procd+"/self/mounts", `
device path type options
memory1 /somewhere/over/the/rainbow cgroup dummy,memory,and,other
memory2 /smart/fox/jumped/by/lazy/dog cgroup2 irrelevant,options
`)
			})

			It("it extracts the mounts", func() {
				determineControllerMounts(&sys1, &sys2)
				Expect(sys1).To(Equal("/somewhere/over/the/rainbow"))
				Expect(sys2).To(Equal("/smart/fox/jumped/by/lazy/dog"))
			})
		})

		Describe("Multiple Mounts", func() {
			var sys1, sys2 string

			BeforeEach(func() {
				setupFile(procd+"/self/mounts", `
device path type options
memory1 /somewhere/over/the/rainbow cgroup dummy,memory,and,other
memory2 /smart/fox/jumped/by/lazy/dog cgroup2 irrelevant,options
memory1 /somewhere/over/the/rainbow/duplicate cgroup dummy,memory,and,other
memory2 /smart/fox/jumped/by/lazy/dog/duplicate cgroup2 irrelevant,options
`)
			})

			It("it extracts the first matching mounts", func() {
				determineControllerMounts(&sys1, &sys2)
				Expect(sys1).To(Equal("/somewhere/over/the/rainbow"))
				Expect(sys2).To(Equal("/smart/fox/jumped/by/lazy/dog"))
			})
		})
	})

	Describe("CPU", func() {
		var (
			statFile string
			cpu      Cpu
		)

		BeforeEach(func() {
			statFile = procd + "/stat"
			cpu = Cpu{}
		})

		Describe("Get", func() {
			It("gets CPU usage", func() {
				statContents := []byte("cpu 25 1 2 3 4 5 6 7")
				err := ioutil.WriteFile(statFile, statContents, 0644)
				Expect(err).ToNot(HaveOccurred())

				err = cpu.Get()
				Expect(err).ToNot(HaveOccurred())
				Expect(cpu.User).To(Equal(uint64(25)))
			})

			It("ignores empty lines", func() {
				statContents := []byte("cpu ")
				err := ioutil.WriteFile(statFile, statContents, 0644)
				Expect(err).ToNot(HaveOccurred())

				err = cpu.Get()
				Expect(err).ToNot(HaveOccurred())
				Expect(cpu.User).To(Equal(uint64(0)))
			})
		})

		Describe("CollectCpuStats", func() {
			It("collects CPU usage over time", func() {
				statContents := []byte("cpu 25 1 2 3 4 5 6 7")
				err := ioutil.WriteFile(statFile, statContents, 0644)
				Expect(err).ToNot(HaveOccurred())

				concrete := &ConcreteSigar{}
				cpuUsages, stop := concrete.CollectCpuStats(500 * time.Millisecond)

				Expect(<-cpuUsages).To(Equal(Cpu{
					User:    uint64(25),
					Nice:    uint64(1),
					Sys:     uint64(2),
					Idle:    uint64(3),
					Wait:    uint64(4),
					Irq:     uint64(5),
					SoftIrq: uint64(6),
					Stolen:  uint64(7),
				}))

				statContents = []byte("cpu 30 3 7 10 25 55 36 65")
				err = ioutil.WriteFile(statFile, statContents, 0644)
				Expect(err).ToNot(HaveOccurred())

				Expect(<-cpuUsages).To(Equal(Cpu{
					User:    uint64(5),
					Nice:    uint64(2),
					Sys:     uint64(5),
					Idle:    uint64(7),
					Wait:    uint64(21),
					Irq:     uint64(50),
					SoftIrq: uint64(30),
					Stolen:  uint64(58),
				}))

				stop <- struct{}{}
			})
		})
	})

	Describe("Memory", func() {
		Describe("determineSelfCgroup", func() {
			It("fails for missing file", func() {
				var cg string
				err := determineSelfCgroup(&cg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("open " + procd + "/self/cgroup: no such file or directory"))
				Expect(cg).To(Equal(""))
			})
			It("fails for empty file", func() {
				cgroupSetup(``)
				var cg string
				err := determineSelfCgroup(&cg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to determine control group"))
				Expect(cg).To(Equal(""))
			})
			It("fails for missing data", func() {
				cgroupSetup(`12:freezer:/`)
				var cg string
				err := determineSelfCgroup(&cg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("unable to determine control group"))
				Expect(cg).To(Equal(""))
			})
			It("finds *:memory: over 0::", func() {
				cgroupSetup(`4:memory:/user
0::/bogus`)
				var cg string
				err := determineSelfCgroup(&cg)
				Expect(err).ToNot(HaveOccurred())
				Expect(cg).To(Equal("/user"))
			})
			It("find 0:: without *:memory:", func() {
				cgroupSetup(`0::/user`)
				var cg string
				err := determineSelfCgroup(&cg)
				Expect(err).ToNot(HaveOccurred())
				Expect(cg).To(Equal("/user"))
			})
		})

		Describe("determineMemoryLimit", func() {
			It("fails for missing files", func() {
				limit, err := determineMemoryLimit(``)
				Expect(err).To(HaveOccurred())
				// it will falls back to memory.stat when memory.limit_in_bytes not found
				Expect(err.Error()).To(Equal("open " + procd + "/memory/memory.stat: no such file or directory"))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for missing data in memory.stat file", func() {
				memStatSetup(``, ``)
				limit, err := determineMemoryLimit(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`no hierarchical memory limit found`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for missing data in v1 file", func() {
				memLimitSetup1(``, ``)
				limit, err := determineMemoryLimit(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for missing data in v2 file", func() {
				memLimitSetup2(``, ``)
				limit, err := determineMemoryLimit(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for bogus data in v1 file", func() {
				memLimitSetup1(``, `bogus`)
				limit, err := determineMemoryLimit(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "bogus": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for bogus data in v2 file", func() {
				memLimitSetup2(``, `bogus`)
				limit, err := determineMemoryLimit(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "bogus": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("returns v2 data over v1", func() {
				memLimitSetup1(``, `1111`)
				memLimitSetup2(``, `2222`)
				limit, err := determineMemoryLimit(``)
				Expect(err).ToNot(HaveOccurred())
				Expect(limit).To(BeNumerically("==", 2222))
			})
			It("returns v1 data when v2 not available", func() {
				memLimitSetup1(``, `1111`)
				limit, err := determineMemoryLimit(``)
				Expect(err).ToNot(HaveOccurred())
				Expect(limit).To(BeNumerically("==", 1111))
			})
			It("returns hierarchyMemoryLimit when limit_in_bytes is unlimited", func() {
				memStatSetup(``, `hierarchical_memory_limit 3333`)
				limit, err := determineMemoryLimit(``)
				Expect(err).ToNot(HaveOccurred())
				Expect(limit).To(BeNumerically("==", 3333))
			})
			It("signals v2 no limit with failure", func() {
				memLimitSetup2(``, `max`)
				limit, err := determineMemoryLimit(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`no limit`))
				Expect(limit).To(BeNumerically("==", 0))
			})
		})

		Describe("determineMemoryUsage", func() {
			It("fails for missing files", func() {
				limit, err := determineMemoryUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("open " + procd + "/memory/memory.stat: no such file or directory"))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for missing data in v1 file", func() {
				memStatSetup(``, ``)
				limit, err := determineMemoryUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`no data found`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for missing data in v2 file", func() {
				memUsageSetup2(``, ``)
				limit, err := determineMemoryUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for bogus data in v1 file", func() {
				memStatSetup(``, `total_rss bogus`)
				limit, err := determineMemoryUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`no data found`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for bogus data in v2 file", func() {
				memUsageSetup2(``, `bogus`)
				limit, err := determineMemoryUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "bogus": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("returns v2 data over v1", func() {
				memStatSetup(``, `total_rss 1111`)
				memUsageSetup2(``, `2222`)
				limit, err := determineMemoryUsage(``)
				Expect(err).ToNot(HaveOccurred())
				Expect(limit).To(BeNumerically("==", 2222))
			})
			It("returns v1 data when v2 not available", func() {
				memStatSetup(``, `total_rss 1111`)
				limit, err := determineMemoryUsage(``)
				Expect(err).ToNot(HaveOccurred())
				Expect(limit).To(BeNumerically("==", 1111))
			})
		})

		Describe("determineSwapUsage", func() {
			It("fails for missing files", func() {
				limit, err := determineSwapUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("open " + procd + "/memory/memory.stat: no such file or directory"))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for missing data in v1 file", func() {
				swapUsageSetup1(``, ``)
				limit, err := determineSwapUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`no data found`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for missing data in v2 file", func() {
				swapUsageSetup2(``, ``)
				limit, err := determineSwapUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for bogus data in v1 file", func() {
				swapUsageSetup1(``, `swap bogus`)
				limit, err := determineSwapUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`no data found`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("fails for bogus data in v2 file", func() {
				swapUsageSetup2(``, `bogus`)
				limit, err := determineSwapUsage(``)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`strconv.ParseUint: parsing "bogus": invalid syntax`))
				Expect(limit).To(BeNumerically("==", 0))
			})
			It("returns v2 data over v1", func() {
				swapUsageSetup1(``, `swap 1111`)
				swapUsageSetup2(``, `2222`)
				limit, err := determineSwapUsage(``)
				Expect(err).ToNot(HaveOccurred())
				Expect(limit).To(BeNumerically("==", 2222))
			})
			It("returns v1 data when v2 not available", func() {
				swapUsageSetup1(``, `swap 1111`)
				limit, err := determineSwapUsage(``)
				Expect(err).ToNot(HaveOccurred())
				Expect(limit).To(BeNumerically("==", 1111))
			})
		})

		Describe("Mem", func() {
			Describe("Without MemAvailable", func() {
				BeforeEach(func() {
					memInfoSetup(`
MemTotal:         374256 kB
MemFree:          274460 kB
Buffers:            9764 kB
Cached:            38648 kB
SwapCached:            0 kB
Active:            33772 kB
Inactive:          31184 kB
Active(anon):      16572 kB
Inactive(anon):      552 kB
Active(file):      17200 kB
Inactive(file):    30632 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:        786428 kB
SwapFree:         786428 kB
Dirty:                 0 kB
Writeback:             0 kB
AnonPages:         16564 kB
Mapped:             6612 kB
Shmem:               584 kB
Slab:              19092 kB
SReclaimable:       9128 kB
SUnreclaim:         9964 kB
KernelStack:         672 kB
PageTables:         1864 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:      973556 kB
Committed_AS:      55880 kB
VmallocTotal:   34359738367 kB
VmallocUsed:       21428 kB
VmallocChunk:   34359713596 kB
HardwareCorrupted:     0 kB
AnonHugePages:         0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:       59328 kB
DirectMap2M:      333824 kB`)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 374256*1024))
					Expect(mem.Free).To(BeNumerically("==", 274460*1024))
					Expect(mem.ActualFree).To(BeNumerically("==", 322872*1024))
					Expect(mem.ActualUsed).To(BeNumerically("==", 51384*1024))
				})
			})

			Describe("With MemAvailable", func() {
				BeforeEach(func() {
					memInfoSetup(`
MemTotal:       35008180 kB
MemFree:          487816 kB
MemAvailable:   20913400 kB
Buffers:          249244 kB
Cached:          5064684 kB
SwapCached:       158628 kB
Active:         10974348 kB
Inactive:        7441132 kB
Active(anon):    7921056 kB
Inactive(anon):  5192512 kB
Active(file):    3053292 kB
Inactive(file):  2248620 kB
Unevictable:           4 kB
Mlocked:               4 kB
SwapTotal:      35013660 kB
SwapFree:       33981728 kB
Dirty:               652 kB
Writeback:             0 kB
AnonPages:      12975584 kB
Mapped:           341188 kB
Shmem:             12280 kB
Slab:           15754916 kB
SReclaimable:   15534604 kB
SUnreclaim:       220312 kB
KernelStack:       42960 kB
PageTables:        52744 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:    52517748 kB
Committed_AS:   22939984 kB
VmallocTotal:   34359738367 kB
VmallocUsed:           0 kB
VmallocChunk:          0 kB
HardwareCorrupted:     0 kB
AnonHugePages:  11448320 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:      667520 kB
DirectMap2M:    34983936 kB`)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 487816*1024))
					Expect(mem.ActualFree).To(BeNumerically("==", 20913400*1024))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14094780*1024))
				})
			})

			// Three toggles:
			// - MemAvailable     present        yes/no
			// - Cgroup limit     valid&sensible yes/no
			// - Cgroup swap data present        yes/no
			// Times two, for cgroup v1 and v2/
			//
			// Total of 16 tests.
			//
			// Note that `MemAvailable present yes/no` does not matter in
			// the results, as the cgroup derived results will write over
			// them. Thus we have 2 groups a 4 tests, with identical
			// results for the equivalent tests of each group.

			Describe("With MemAvailable. With v1 cgroup limit. With v1 cgroup swap", func() {
				// cgroup = '/user'
				// The other tests use cgroup = '/'. This reduces the amount of changes needed
				BeforeEach(func() {
					cgroupSetup(`4:memory:/user`)
					memInfoWithMemAvailable()
					memLimitSetup1(`/user`, `21390950400`)
					memStatWithSwap(`/user`)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("With MemAvailable. With v1 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithMemAvailable()
					memLimitSetup1(``, `21390950400`)
					memStatWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("With MemAvailable. Overlarge v1 cgroup limit. With v1 cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithMemAvailable()
					memLimitSetup1(``, `213909504000000000000`)
					memStatWithSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("With MemAvailable. Overlarge v1 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithMemAvailable()
					memLimitSetup1(``, `213909504000000000000`)
					memStatWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("Without MemAvailable. With v1 cgroup limit. With v1 cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup1(``, `21390950400`)
					memStatWithSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("Without MemAvailable. With v1 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup1(``, `21390950400`)
					memStatWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("Without MemAvailable. Overlarge v1 cgroup limit. With v1 cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup1(``, `213909504000000000000`)
					memStatWithSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("Without MemAvailable. Overlarge v1 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup1(``, `213909504000000000000`)
					memStatWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("With MemAvailable. With v2 cgroup limit. With v2 cgroup swap", func() {
				// cgroup = '/user'
				// The other tests use cgroup = '/'. This reduces the amount of changes needed
				BeforeEach(func() {
					cgroupSetup(`4:memory:/user`)
					memInfoWithMemAvailable()
					memLimitSetup2(`/user`, `21390950400`)
					memUsageWithSwap(`/user`)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("With MemAvailable. With v2 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithMemAvailable()
					memLimitSetup2(``, `21390950400`)
					memUsageWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("With MemAvailable. Overlarge v2 cgroup limit. With v2 cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithMemAvailable()
					memLimitSetup2(``, `213909504000000000000`)
					memUsageWithSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("With MemAvailable. Overlarge v2 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithMemAvailable()
					memLimitSetup2(``, `213909504000000000000`)
					memUsageWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("Without MemAvailable. With v2 cgroup limit. With v2 cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup2(``, `21390950400`)
					memUsageWithSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("Without MemAvailable. With v2 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup2(``, `21390950400`)
					memUsageWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 21390950400))
					Expect(mem.Free).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 21390950400-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("Without MemAvailable. Overlarge v2 cgroup limit. With v2 cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup2(``, `213909504000000000000`)
					memUsageWithSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832-290564089))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832+290564089))
				})
			})

			Describe("Without MemAvailable. Overlarge v2 cgroup limit. Without cgroup swap", func() {
				BeforeEach(func() {
					cgroupSetup(`4:memory:/`)
					memInfoWithoutMemAvailable()
					memLimitSetup2(``, `213909504000000000000`)
					memUsageWithoutSwap(``)
				})

				It("returns correct memory info", func() {
					mem := Mem{}
					err := mem.Get()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualFree).To(BeNumerically("==", 35008180*1024-14108536832))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14108536832))
				})
			})

			Describe("When cgroups are enabled but the GetIgnoringCGroups method is called", func() {
				// cgroup = '/user'
				// The other tests use cgroup = '/'. This reduces the amount of changes needed
				BeforeEach(func() {
					cgroupSetup(`4:memory:/user`)
					memInfoWithMemAvailable()
					memLimitSetup2(`/user`, `21390950400`)
					memUsageWithSwap(`/user`)
				})

				It("returns the system memory info", func() {
					mem := Mem{}
					err := mem.GetIgnoringCGroups()
					Expect(err).ToNot(HaveOccurred())

					Expect(mem.Total).To(BeNumerically("==", 35008180*1024))
					Expect(mem.Free).To(BeNumerically("==", 487816*1024))
					Expect(mem.ActualFree).To(BeNumerically("==", 20913400*1024))
					Expect(mem.ActualUsed).To(BeNumerically("==", 14433054720))
				})
			})
		})

		Describe("Swap", func() {
			var meminfoFile string
			BeforeEach(func() {
				meminfoFile = procd + "/meminfo"

				meminfoContents := `
MemTotal:         374256 kB
MemFree:          274460 kB
Buffers:            9764 kB
Cached:            38648 kB
SwapCached:            0 kB
Active:            33772 kB
Inactive:          31184 kB
Active(anon):      16572 kB
Inactive(anon):      552 kB
Active(file):      17200 kB
Inactive(file):    30632 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:        786428 kB
SwapFree:         786428 kB
Dirty:                 0 kB
Writeback:             0 kB
AnonPages:         16564 kB
Mapped:             6612 kB
Shmem:               584 kB
Slab:              19092 kB
SReclaimable:       9128 kB
SUnreclaim:         9964 kB
KernelStack:         672 kB
PageTables:         1864 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:      973556 kB
Committed_AS:      55880 kB
VmallocTotal:   34359738367 kB
VmallocUsed:       21428 kB
VmallocChunk:   34359713596 kB
HardwareCorrupted:     0 kB
AnonHugePages:         0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:       59328 kB
DirectMap2M:      333824 kB
`
				err := ioutil.WriteFile(meminfoFile, []byte(meminfoContents), 0444)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns correct memory info", func() {
				swap := Swap{}
				err := swap.Get()
				Expect(err).ToNot(HaveOccurred())

				Expect(swap.Total).To(BeNumerically("==", 786428*1024))
				Expect(swap.Free).To(BeNumerically("==", 786428*1024))
			})
		})
		Describe("List filesystems in /etc/mtab", func() {
			BeforeEach(func() {
				setupEtcMtab()
			})

			It("returns correct list of filesystems", func() {
				fslist := FileSystemList{}
				err := fslist.Get()
				Expect(err).ToNot(HaveOccurred())

				Expect(len(fslist.List)).To(BeNumerically(">", 0))
			})
		})
		Describe("List filesystems in /proc/mounts", func() {
			BeforeEach(func() {
				setupProcMounts()
			})

			It("returns correct list of filesystems", func() {
				fslist := FileSystemList{}
				err := fslist.Get()
				Expect(err).ToNot(HaveOccurred())

				Expect(len(fslist.List)).To(BeNumerically(">", 0))
			})
		})
	})
})
