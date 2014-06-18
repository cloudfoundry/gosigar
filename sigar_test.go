package sigar_test

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/gosigar"
)

var _ = Describe("Sigar", func() {
	var invalidPid = 666666

	It("load average", func() {
		avg := LoadAverage{}
		err := avg.Get()
		Expect(err).ToNot(HaveOccured())
	})

	It("uptime", func() {
		uptime := Uptime{}
		err := uptime.Get()
		Expect(err).ToNot(HaveOccured())
		Expect(uptime.Length).To(BeNumerically(">", 0))
	})

	It("mem", func() {
		mem := Mem{}
		err := mem.Get()
		Expect(err).ToNot(HaveOccured())

		Expect(mem.Total).To(BeNumerically(">", 0))
		Expect(mem.Used + mem.Free).To(BeNumerically("<=", mem.Total))
	})

	It("swap", func() {
		swap := Swap{}
		err := swap.Get()
		Expect(err).ToNot(HaveOccured())
		Expect(swap.Used + swap.Free).To(BeNumerically("<=", swap.Total))
	})

	It("cpu", func() {
		cpu := Cpu{}
		err := cpu.Get()
		Expect(err).ToNot(HaveOccured())
	})

	It("CollectCpuStats", func() {
		cpuUsages, stop := CollectCpuStats(500 * time.Millisecond)
		firstValue := <-cpuUsages
		secondValue := <-cpuUsages

		Expect(firstValue).ToNot(Equal(secondValue))

		stop <- true
	})

	It("cpu list", func() {
		cpulist := CpuList{}
		err := cpulist.Get()
		Expect(err).ToNot(HaveOccured())

		nsigar := len(cpulist.List)
		numcpu := runtime.NumCPU()
		Expect(nsigar).To(Equal(numcpu))
	})

	It("file system list", func() {
		fslist := FileSystemList{}
		err := fslist.Get()
		Expect(err).ToNot(HaveOccured())

		Expect(len(fslist.List)).To(BeNumerically(">", 0))
	})

	It("file system usage", func() {
		fsusage := FileSystemUsage{}
		err := fsusage.Get("/")
		Expect(err).ToNot(HaveOccured())

		err = fsusage.Get("T O T A L L Y B O G U S")
		Expect(err).To(HaveOccured())
	})

	It("proc list", func() {
		pids := ProcList{}
		err := pids.Get()
		Expect(err).ToNot(HaveOccured())

		Expect(len(pids.List)).To(BeNumerically(">", 2))

		err = pids.Get()
		Expect(err).ToNot(HaveOccured())
	})

	It("proc state", func() {
		state := ProcState{}
		err := state.Get(os.Getppid())
		Expect(err).ToNot(HaveOccured())

		Expect([]RunState{RunStateRun, RunStateSleep}).To(ContainElement(state.State))
		Expect([]string{"go", "ginkgo"}).To(ContainElement(state.Name))

		err = state.Get(invalidPid)
		Expect(err).To(HaveOccured())
	})

	It("proc mem", func() {
		mem := ProcMem{}
		err := mem.Get(os.Getppid())
		Expect(err).ToNot(HaveOccured())

		err = mem.Get(invalidPid)
		Expect(err).To(HaveOccured())
	})

	It("proc time", func() {
		time := ProcTime{}
		err := time.Get(os.Getppid())
		Expect(err).ToNot(HaveOccured())

		err = time.Get(invalidPid)
		Expect(err).To(HaveOccured())
	})

	It("proc args", func() {
		args := ProcArgs{}
		err := args.Get(os.Getppid())
		Expect(err).ToNot(HaveOccured())

		Expect(len(args.List)).To(BeNumerically(">=", 2))
	})

	It("proc exe", func() {
		exe := ProcExe{}
		err := exe.Get(os.Getppid())
		Expect(err).ToNot(HaveOccured())

		Expect([]string{"go", "ginkgo"}).To(ContainElement(filepath.Base(exe.Name)))
	})
})
