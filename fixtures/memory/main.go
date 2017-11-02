package main

import (
	"flag"
	"time"
)

var count = flag.Int("count", 0, "how many integers to allocate")

func main() {
	flag.Parse()
	stuff := make([]int32, *count)
	for i := int32(0); i < int32(*count); i++ {
		stuff[i] = i
	}

	for {
		time.Sleep(time.Second)
	}
}
