package main

import (
	"flag"
	"time"
)

var count = flag.Int("count", 0, "how many bytes to allocate")

func main() {
	flag.Parse()
	stuff := make([]int8, *count)
	for i := 0; i < *count; i++ {
		stuff[i] = int8(i % 10)
	}

	for {
		time.Sleep(time.Second * 60)
	}
}
