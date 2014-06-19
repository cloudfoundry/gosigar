package sigar

import (
	"time"
)

type ConcreteSigar struct{}

func (c *ConcreteSigar) CollectCpuStats(collectionInterval time.Duration) (<-chan Cpu, chan<- struct{}) {
	// samplesCh is buffered to 1 value to immediately return first CPU sample
	samplesCh := make(chan Cpu, 1)

	stopCh := make(chan struct{})

	go func() {
		var cpuUsage Cpu

		// Immediately provide non-delta value.
		// samplesCh is buffered to 1 value, so it will not block.
		cpuUsage.Get()
		samplesCh <- cpuUsage

		ticker := time.NewTicker(collectionInterval)

		for {
			select {
			case <-ticker.C:
				previousCpuUsage := cpuUsage

				cpuUsage.Get()

				select {
				case samplesCh <- cpuUsage.Delta(previousCpuUsage):
				default:
					// Include default to avoid channel blocking
				}

			case <-stopCh:
				return
			}
		}
	}()

	return samplesCh, stopCh
}
