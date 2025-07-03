package main

import (
	"fmt"
	"os"

	sigar "github.com/cloudfoundry/gosigar"
)

func format(val uint64) uint64 {
	return val / 1024
}

func main() {
	mem := sigar.Mem{}
	swap := sigar.Swap{}

	mem.Get()  //nolint:errcheck
	swap.Get() //nolint:errcheck

	fmt.Fprintf(os.Stdout, "%18s %10s %10s\n", //nolint:errcheck
		"total", "used", "free")

	fmt.Fprintf(os.Stdout, "Mem:    %10d %10d %10d\n", //nolint:errcheck
		format(mem.Total), format(mem.Used), format(mem.Free))

	fmt.Fprintf(os.Stdout, "-/+ buffers/cache: %10d %10d\n", //nolint:errcheck
		format(mem.ActualUsed), format(mem.ActualFree))

	fmt.Fprintf(os.Stdout, "Swap:   %10d %10d %10d\n", //nolint:errcheck
		format(swap.Total), format(swap.Used), format(swap.Free))
}
