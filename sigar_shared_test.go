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
		var cpuGenerator *exec.Cmd
		var noCpuGenerator *exec.Cmd

		BeforeEach(func() {
			pathToStress, err := gexec.Build("github.com/cloudfoundry/gosigar/stress")
			Expect(err).NotTo(HaveOccurred())
			cpuGenerator = exec.Command(pathToStress)
			if err := cpuGenerator.Start(); err != nil {
				panic("failed to start cpu generator")
			}

			noCPUCommand := "cat"
			if runtime.GOOS == "windows" {
				noCPUCommand = "notepad.exe"
			}
			noCpuGenerator = exec.Command(noCPUCommand)
			if err := noCpuGenerator.Start(); err != nil {
				panic("failed to start no cpu generator")
			}
		})

		AfterEach(func() {
			cpuGenerator.Process.Signal(os.Kill)
			noCpuGenerator.Process.Signal(os.Kill)
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

			err = pCpu.Get(noCpuGenerator.Process.Pid)
			Expect(err).ToNot(HaveOccurred())

			Expect(pCpu.Percent).To(BeNumerically("~", 0.0, 0.02))
		})
	})
})
