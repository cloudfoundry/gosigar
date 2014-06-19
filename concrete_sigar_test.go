package sigar_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	sigar "github.com/cloudfoundry/gosigar"
)

var _ = Describe("ConcreteSigar", func() {
	var concreteSigar *sigar.ConcreteSigar

	BeforeEach(func() {
		concreteSigar = &sigar.ConcreteSigar{}
	})

	Describe("CollectCpuStats", func() {
		It("immediately makes first CPU usage available even though it's not very accurate", func() {
			samplesCh, stop := concreteSigar.CollectCpuStats(500 * time.Millisecond)

			firstValue := <-samplesCh
			Expect(firstValue.User).To(BeNumerically(">", 0))

			stop <- struct{}{}
		})

		It("makes CPU usage delta values available", func() {
			samplesCh, stop := concreteSigar.CollectCpuStats(500 * time.Millisecond)

			firstValue := <-samplesCh

			secondValue := <-samplesCh
			Expect(secondValue.User).To(BeNumerically("<", firstValue.User))

			thirdValue := <-samplesCh
			Expect(thirdValue).ToNot(Equal(secondValue))

			stop <- struct{}{}
		})

		It("does not block", func() {
			_, stop := concreteSigar.CollectCpuStats(10 * time.Millisecond)

			// Sleep long enough for samplesCh to fill at least 2 values
			time.Sleep(20 * time.Millisecond)

			stop <- struct{}{}

			// If CollectCpuStats blocks it will never get here
			Expect(true).To(BeTrue())
		})
	})
})
