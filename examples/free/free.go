package main

import (
	"fmt"

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

	fmt.Printf("%18s %10s %10s\n", "total", "used", "free")
	fmt.Printf("Mem:    %10d %10d %10d\n", format(mem.Total), format(mem.Used), format(mem.Free))
	fmt.Printf("-/+ buffers/cache: %10d %10d\n", format(mem.ActualUsed), format(mem.ActualFree))
	fmt.Printf("Swap:   %10d %10d %10d\n", format(swap.Total), format(swap.Used), format(swap.Free))
}
