package sigar

import (
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("sigarLinux", func() {
	var procd string

	BeforeEach(func() {
		var err error
		procd, err = ioutil.TempDir("", "sigarTests")
		Expect(err).ToNot(HaveOccurred())
		Procd = procd
	})

	AfterEach(func() {
		Procd = "/proc"
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

				concreteSigar := &ConcreteSigar{}
				cpuUsages, stop := concreteSigar.CollectCpuStats(500 * time.Millisecond)

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

	Describe("Mem", func() {
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
			mem := Mem{}
			err := mem.Get()
			Expect(err).ToNot(HaveOccurred())

			Expect(mem.Total).To(BeNumerically("==", 374256*1024))
			Expect(mem.Free).To(BeNumerically("==", 274460*1024))
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

			vmstatFile := procd + "/vmstat"
			vmstatContents := `
nr_free_pages 31466
nr_alloc_batch 1465
nr_inactive_anon 536472
nr_active_anon 687249
nr_inactive_file 298927
nr_active_file 394956
nr_unevictable 0
nr_mlock 0
nr_anon_pages 1223424
nr_mapped 65499
nr_file_pages 694556
nr_dirty 79
nr_writeback 0
nr_slab_reclaimable 65066
nr_slab_unreclaimable 8930
nr_page_table_pages 5449
nr_kernel_stack 603
nr_unstable 0
nr_bounce 0
nr_vmscan_write 82260
nr_vmscan_immediate_reclaim 813854
nr_writeback_temp 0
nr_isolated_anon 0
nr_isolated_file 0
nr_shmem 35
nr_dirtied 12929751
nr_written 8381406
nr_pages_scanned 0
numa_hit 320441967
numa_miss 0
numa_foreign 0
numa_interleave 12968
numa_local 320441967
numa_other 0
workingset_refault 82976
workingset_activate 7538
workingset_nodereclaim 0
nr_anon_transparent_hugepages 1787
nr_free_cma 0
nr_dirty_threshold 138776
nr_dirty_background_threshold 69388
pgpgin 463395
pgpgout 38650530
pswpin 3192
pswpout 82186
pgalloc_dma 24
pgalloc_dma32 132894366
pgalloc_normal 222090800
pgalloc_movable 0
pgfree 355752894
pgactivate 4748817
pgdeactivate 2825899
pgfault 428620784
pgmajfault 4249
pgrefill_dma 0
pgrefill_dma32 311514
pgrefill_normal 728856
pgrefill_movable 0
pgsteal_kswapd_dma 0
pgsteal_kswapd_dma32 241904
pgsteal_kswapd_normal 386047
pgsteal_kswapd_movable 0
pgsteal_direct_dma 0
pgsteal_direct_dma32 63934
pgsteal_direct_normal 228160
pgsteal_direct_movable 0
pgscan_kswapd_dma 0
pgscan_kswapd_dma32 272276
pgscan_kswapd_normal 428781
pgscan_kswapd_movable 0
pgscan_direct_dma 0
pgscan_direct_dma32 66132
pgscan_direct_normal 247084
pgscan_direct_movable 0
pgscan_direct_throttle 0
zone_reclaim_failed 0
pginodesteal 2645
slabs_scanned 487296
kswapd_inodesteal 26051
kswapd_low_wmark_hit_quickly 3
kswapd_high_wmark_hit_quickly 82
pageoutrun 147
allocstall 579
pgrotated 145698
drop_pagecache 0
drop_slab 0
numa_pte_updates 0
numa_huge_pte_updates 0
numa_hint_faults 0
numa_hint_faults_local 0
numa_pages_migrated 0
pgmigrate_success 533657
pgmigrate_fail 0
compact_migrate_scanned 781620
compact_free_scanned 16336029
compact_isolated 1302183
compact_stall 1781
compact_fail 839
compact_success 942
htlb_buddy_alloc_success 0
htlb_buddy_alloc_fail 0
unevictable_pgs_culled 0
unevictable_pgs_scanned 0
unevictable_pgs_rescued 0
unevictable_pgs_mlocked 0
unevictable_pgs_munlocked 0
unevictable_pgs_cleared 0
unevictable_pgs_stranded 0
thp_fault_alloc 8771
thp_fault_fallback 350
thp_collapse_alloc 2230
thp_collapse_alloc_failed 89
thp_split 981
thp_zero_page_alloc 0
thp_zero_page_alloc_failed 0
balloon_inflate 0
balloon_deflate 0
balloon_migrate 0
`
			err = ioutil.WriteFile(vmstatFile, []byte(vmstatContents), 0444)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns correct swap info", func() {
			swap := Swap{}
			err := swap.Get()
			Expect(err).ToNot(HaveOccurred())

			Expect(swap.Total).To(BeNumerically("==", 786428*1024))
			Expect(swap.Free).To(BeNumerically("==", 786428*1024))
			Expect(swap.PageIn).To(BeNumerically("==", 3192))
			Expect(swap.PageOut).To(BeNumerically("==", 82186))
		})
	})
})
