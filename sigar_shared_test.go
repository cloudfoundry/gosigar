package sigar

import (
	"os"
	"os/exec"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("SigarShared", func() {
	Describe("ProcCpu", func() {
		var (
			cpuGenerator   *exec.Cmd
			noCPUGenerator *exec.Cmd
		)

		BeforeEach(func() {
			pathToStress, err := gexec.Build("github.com/cloudfoundry/gosigar/fixtures/stress")
			Expect(err).NotTo(HaveOccurred())
			cpuGenerator = exec.Command(pathToStress)
			if err := cpuGenerator.Start(); err != nil {
				panic("failed to start cpu generator")
			}

			noCPUCommand := "cat"
			if runtime.GOOS == "windows" {
				noCPUCommand = "notepad.exe"
			}
			noCPUGenerator = exec.Command(noCPUCommand)
			if err := noCPUGenerator.Start(); err != nil {
				panic("failed to start no cpu generator")
			}
		})

		AfterEach(func() {
			cpuGenerator.Process.Signal(os.Kill)
			noCPUGenerator.Process.Signal(os.Kill)
		})

		It("calculates percentage", func() {
			time.Sleep(time.Second) // High CPU process needs a second to spool up

			pCpu := &ProcCpu{}

			err := pCpu.Get(cpuGenerator.Process.Pid)
			Expect(err).ToNot(HaveOccurred())
			Expect(pCpu.Percent).To(BeNumerically("~", 1.0, 0.1))
		})

		It("does not conflate multiple processes", func() {
			time.Sleep(time.Second) // High CPU process needs a second to spool up

			pCpu := &ProcCpu{}

			err := pCpu.Get(cpuGenerator.Process.Pid)
			Expect(err).ToNot(HaveOccurred())

			err = pCpu.Get(noCPUGenerator.Process.Pid)
			Expect(err).ToNot(HaveOccurred())

			Expect(pCpu.Percent).To(BeNumerically("~", 0.0, 0.02))
		})
	})

	Describe("ProcMem", func() {
		var memGenerator *exec.Cmd
		var noMemGenerator *exec.Cmd

		BeforeEach(func() {
			pathToMemory, err := gexec.Build("github.com/cloudfoundry/gosigar/fixtures/memory")
			Expect(err).NotTo(HaveOccurred())
			memGenerator = exec.Command(pathToMemory, "-count", "16000000")
			if err := memGenerator.Start(); err != nil {
				panic("failed to start mem generator")
			}

			noMemGenerator = exec.Command(pathToMemory, "-count", "0")
			if err := noMemGenerator.Start(); err != nil {
				panic("failed to start no mem generator")
			}
		})

		AfterEach(func() {
			memGenerator.Process.Signal(os.Kill)
			noMemGenerator.Process.Signal(os.Kill)
		})

		It("calculates memory usage", func() {
			time.Sleep(time.Second) // High MEM process needs a second to spool up

			pMem := &ProcMem{}

			err := pMem.Get(memGenerator.Process.Pid)
			Expect(err).ToNot(HaveOccurred())
			Expect(pMem.Resident).To(BeNumerically("~", 18000000, 5*1024*1024))
			Expect(pMem.Size).To(BeNumerically(">=", pMem.Resident))

			pNoMem := &ProcMem{}
			err = pNoMem.Get(noMemGenerator.Process.Pid)
			Expect(err).ToNot(HaveOccurred())

			Expect(pNoMem.Resident).To(BeNumerically("~", 2000000, 5*1024*1024))
			Expect(pNoMem.Size).To(BeNumerically(">=", pNoMem.Resident))
		})
	})
})
