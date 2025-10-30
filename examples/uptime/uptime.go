package main

import (
	"fmt"
	"time"

	sigar "github.com/cloudfoundry/gosigar"
)

func main() {
	concreteSigar := sigar.ConcreteSigar{}

	uptime := sigar.Uptime{}
	uptime.Get() //nolint:errcheck
	avg, err := concreteSigar.GetLoadAverage()
	if err != nil {
		fmt.Printf("Failed to get load average\n")
		return
	}

	fmt.Printf(" %s up %s load average: %.2f, %.2f, %.2f\n",
		time.Now().Format("15:04:05"),
		uptime.Format(),
		avg.One, avg.Five, avg.Fifteen)
}
