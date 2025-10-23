package main

import (
	"flag"
	"fmt"

	sigar "github.com/cloudfoundry/gosigar"
)

const outputFormat = "%-15s %4s %4s %5s %4s %-15s\n"

func formatSize(size uint64) string {
	return sigar.FormatSize(size * 1024)
}

func main() {
	var fileSystemFilter string
	flag.StringVar(&fileSystemFilter, "t", "", "Filesystem to filter on")
	flag.Parse()
	fslist := sigar.FileSystemList{}
	fslist.Get() //nolint:errcheck

	fmt.Printf(outputFormat, "Filesystem", "Size", "Used", "Avail", "Use%", "Mounted on")

	for _, fs := range fslist.List {
		dirName := fs.DirName
		usage := sigar.FileSystemUsage{}
		usage.Get(dirName) //nolint:errcheck

		if fileSystemFilter != "" {
			if fs.SysTypeName != fileSystemFilter {
				continue
			}
		}

		fmt.Printf(outputFormat,
			fs.DevName,
			formatSize(usage.Total),
			formatSize(usage.Used),
			formatSize(usage.Avail),
			sigar.FormatPercent(usage.UsePercent()),
			dirName)
	}
}
