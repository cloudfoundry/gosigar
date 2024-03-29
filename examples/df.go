package main

import (
	"flag"
	"fmt"
	"os"

	sigar "github.com/cloudfoundry/gosigar"
)

const output_format = "%-15s %4s %4s %5s %4s %-15s\n"

func formatSize(size uint64) string {
	return sigar.FormatSize(size * 1024)
}

func main() {
	var fileSystemFilter string
	flag.StringVar(&fileSystemFilter, "t", "", "Filesystem to filter on")
	flag.Parse()
	fslist := sigar.FileSystemList{}
	fslist.Get()

	fmt.Fprintf(os.Stdout, output_format,
		"Filesystem", "Size", "Used", "Avail", "Use%", "Mounted on")

	for _, fs := range fslist.List {
		dir_name := fs.DirName
		usage := sigar.FileSystemUsage{}
		usage.Get(dir_name)

		if fileSystemFilter != "" {
			if fs.SysTypeName != fileSystemFilter {
				continue
			}
		}

		fmt.Fprintf(os.Stdout, output_format,
			fs.DevName,
			formatSize(usage.Total),
			formatSize(usage.Used),
			formatSize(usage.Avail),
			sigar.FormatPercent(usage.UsePercent()),
			dir_name)
	}
}
