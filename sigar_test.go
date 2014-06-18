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
		Expect(err).ToNot(HaveOccurred())
	})

	It("uptime", func() {
		uptime := Uptime{}
		err := uptime.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(uptime.Length).To(BeNumerically(">", 0))
	})

	It("mem", func() {
		mem := Mem{}
		err := mem.Get()
		Expect(err).ToNot(HaveOccurred())

		Expect(mem.Total).To(BeNumerically(">", 0))
		Expect(mem.Used + mem.Free).To(BeNumerically("<=", mem.Total))
	})

	It("swap", func() {
		swap := Swap{}
		err := swap.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(swap.Used + swap.Free).To(BeNumerically("<=", swap.Total))
	})

	It("cpu", func() {
		cpu := Cpu{}
		err := cpu.Get()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("CollectCpuStats", func() {
		It("immediately makes first CPU usage available even though it's not very accurate", func() {
			samplesCh, stop := CollectCpuStats(500 * time.Millisecond)

			firstValue := <-samplesCh
			Expect(firstValue.User).To(BeNumerically(">", 0))

			stop <- struct{}{}
		})

		It("makes CPU usage delta values available", func() {
			samplesCh, stop := CollectCpuStats(500 * time.Millisecond)

			firstValue := <-samplesCh

			secondValue := <-samplesCh
			Expect(secondValue.User).To(BeNumerically("<", firstValue.User))

			thirdValue := <-samplesCh
			Expect(thirdValue).ToNot(Equal(secondValue))

			stop <- struct{}{}
		})

		It("does not block", func() {
			_, stop := CollectCpuStats(10 * time.Millisecond)

			// Sleep long enough for samplesCh to fill at least 2 values
			time.Sleep(20 * time.Millisecond)

			stop <- struct{}{}

			// If CollectCpuStats blocks it will never get here
			Expect(true).To(BeTrue())
		})
	})

	It("cpu list", func() {
		cpulist := CpuList{}
		err := cpulist.Get()
		Expect(err).ToNot(HaveOccurred())

		nsigar := len(cpulist.List)
		numcpu := runtime.NumCPU()
		Expect(nsigar).To(Equal(numcpu))
	})

	It("file system list", func() {
		fslist := FileSystemList{}
		err := fslist.Get()
		Expect(err).ToNot(HaveOccurred())

		Expect(len(fslist.List)).To(BeNumerically(">", 0))
	})

	It("file system usage", func() {
		fsusage := FileSystemUsage{}
		err := fsusage.Get("/")
		Expect(err).ToNot(HaveOccurred())

		err = fsusage.Get("T O T A L L Y B O G U S")
		Expect(err).To(HaveOccurred())
	})

	It("proc list", func() {
		pids := ProcList{}
		err := pids.Get()
		Expect(err).ToNot(HaveOccurred())

		Expect(len(pids.List)).To(BeNumerically(">", 2))

		err = pids.Get()
		Expect(err).ToNot(HaveOccurred())
	})

	It("proc state", func() {
		state := ProcState{}
		err := state.Get(os.Getppid())
		Expect(err).ToNot(HaveOccurred())

		Expect([]RunState{RunStateRun, RunStateSleep}).To(ContainElement(state.State))
		Expect([]string{"go", "ginkgo"}).To(ContainElement(state.Name))

		err = state.Get(invalidPid)
		Expect(err).To(HaveOccurred())
	})

	It("proc mem", func() {
		mem := ProcMem{}
		err := mem.Get(os.Getppid())
		Expect(err).ToNot(HaveOccurred())

		err = mem.Get(invalidPid)
		Expect(err).To(HaveOccurred())
	})

	It("proc time", func() {
		time := ProcTime{}
		err := time.Get(os.Getppid())
		Expect(err).ToNot(HaveOccurred())

		err = time.Get(invalidPid)
		Expect(err).To(HaveOccurred())
	})

	It("proc args", func() {
		args := ProcArgs{}
		err := args.Get(os.Getppid())
		Expect(err).ToNot(HaveOccurred())

		Expect(len(args.List)).To(BeNumerically(">=", 2))
	})

	It("proc exe", func() {
		exe := ProcExe{}
		err := exe.Get(os.Getppid())
		Expect(err).ToNot(HaveOccurred())

		Expect([]string{"go", "ginkgo"}).To(ContainElement(filepath.Base(exe.Name)))
	})
})
