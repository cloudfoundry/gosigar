package sigar

import (
	"os"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
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
				Expect(cpu.Get()).To(Succeed())
				return cpu.Idle
			}, time.Second*10).Should(BeNumerically(">", old.Idle))

			Eventually(func() uint64 {
				Expect(cpu.Get()).To(Succeed())
				return cpu.User
			}, time.Second*10).Should(BeNumerically(">", old.User))

			Eventually(func() uint64 {
				Expect(cpu.Get()).To(Succeed())
				return cpu.Sys
			}, time.Second*10).Should(BeNumerically(">", old.Sys))
		})
	})

	Describe("When DLL procs cannot be loaded", func() {
		nilProc := func(pp **syscall.Proc) (restore func()) {
			op := *pp
			*pp = nil
			return func() { *pp = op }
		}

		It("returns ErrNotSupported when GetDiskFreeSpace cannot be loaded", func() {
			defer nilProc(&procGetDiskFreeSpace)()
			Expect(new(FileSystemUsage).Get(os.TempDir())).To(MatchError(ErrNotSupported))
		})

		It("returns ErrNotSupported when GetSystemTimes cannot be loaded", func() {
			defer nilProc(&procGetSystemTimes)()
			Expect(new(Cpu).Get()).To(MatchError(ErrNotSupported))
		})

		It("returns ErrNotSupported when GetTickCount64 cannot be loaded", func() {
			defer nilProc(&procGetTickCount64)()
			Expect(new(Uptime).Get()).To(MatchError(ErrNotSupported))
		})

		It("returns ErrNotSupported when GlobalMemoryStatusEx cannot be loaded", func() {
			defer nilProc(&procGlobalMemoryStatusEx)()
			Expect(new(Mem).Get()).To(MatchError(ErrNotSupported))
		})

		// test the test
		It("restores procs nilled in the previous test to their original value", func() {
			Expect(procGetDiskFreeSpace).ToNot(BeNil())
			Expect(procGetSystemTimes).ToNot(BeNil())
			Expect(procGetTickCount64).ToNot(BeNil())
			Expect(procGlobalMemoryStatusEx).ToNot(BeNil())
		})
	})

	Context("when parsing wmic output", func() {
		It("should parse the output", func() {
			res := strings.Join([]string{
				`AllocatedBaseSize=4791`,
				`CurrentUsage=393`,
				`Description=C:\pagefile.sys`,
				`InstallDate=20151221103329.285091-480`,
				`Name=C:\pagefile.sys`,
				`PeakUsage=2916`,
				`Status=`,
				`TempPageFile=FALSE`,
			}, "\r\n")

			out := []byte(res)
			num, err := parseWmicOutput(out, []byte("CurrentUsage"))
			Expect(err).To(BeNil())
			Expect(num).To(Equal(uint64(393)))

			num, err = parseWmicOutput(out, []byte("AllocatedBaseSize"))
			Expect(err).To(BeNil())
			Expect(num).To(Equal(uint64(4791)))

			num, err = parseWmicOutput(out, []byte("Status"))
			Expect(err).To(HaveOccurred())
			Expect(num).To(Equal(uint64(0)))

			num, err = parseWmicOutput(out, []byte("Current"))
			Expect(err).To(HaveOccurred())
			Expect(num).To(Equal(uint64(0)))
		})

	})
})
