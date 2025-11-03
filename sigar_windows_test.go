package sigar

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SigarWindows", func() {
	Describe("Uptime", func() {
		It("returns the uptime", func() {
			var u Uptime
			Expect(u.Get()).To(Succeed())
			Expect(u.Length).To(BeNumerically(">", 0))
		})
	})

	Describe("Memory", func() {
		It("gets the total memory", func() {
			var mem Mem
			Expect(mem.Get()).To(Succeed())
			Expect(mem.Total).To(BeNumerically(">", 0))
			Expect(mem.Free).To(BeNumerically(">", 0))
			Expect(mem.ActualFree).To(BeNumerically(">", 0))
			Expect(mem.Used).To(BeNumerically(">", 0))
		})
	})

	Describe("Swap", func() {
		It("gets the total memory", func() {
			var swap Swap
			Expect(swap.Get()).To(Succeed())
			Expect(swap.Total).To(BeNumerically(">", 0))
			Expect(swap.Free).To(BeNumerically(">", 0))
			Expect(swap.Used).To(BeNumerically(">", 0))
		})
	})

	Describe("Disk", func() {
		It("gets the total disk space", func() {
			var usage FileSystemUsage
			Expect(usage.Get(os.TempDir())).To(Succeed())
			Expect(usage.Total).To(BeNumerically(">", 0))
			Expect(usage.Free).To(BeNumerically(">", 0))
			Expect(usage.Used).To(BeNumerically(">", 0))
		})
	})

	Describe("CPU", func() {
		It("gets the cumulative number of cpu ticks", func() {
			var old Cpu
			Expect(old.Get()).To(Succeed())

			var cpu Cpu
			Eventually(func() uint64 {
				cpu.Get() //nolint:errcheck
				return cpu.Idle
			}, time.Second*10).Should(BeNumerically(">", old.Idle))

			Eventually(func() uint64 {
				cpu.Get() //nolint:errcheck
				return cpu.User
			}, time.Second*20).Should(BeNumerically(">", old.User))

			Eventually(func() uint64 {
				cpu.Get() //nolint:errcheck
				return cpu.Sys
			}, time.Second*10).Should(BeNumerically(">", old.Sys))
		})
	})
})
