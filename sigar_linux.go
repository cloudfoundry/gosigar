// Copyright (c) 2012 VMware, Inc.

package sigar

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"syscall"
)

const procd = "/proc/"

func (self *LoadAverage) Get() error {
	line, err := ioutil.ReadFile(procd + "loadavg")
	if err != nil {
		return nil
	}

	fields := strings.Fields(string(line))

	self.One, _ = strconv.ParseFloat(fields[0], 64)
	self.Five, _ = strconv.ParseFloat(fields[1], 64)
	self.Fifteen, _ = strconv.ParseFloat(fields[2], 64)

	return nil
}

func (self *Uptime) Get() error {
	sysinfo := syscall.Sysinfo_t{}

	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return err
	}

	self.Length = float64(sysinfo.Uptime)

	return nil
}

func (self *Mem) Get() error {
	var buffers, cached uint64
	table := map[string]*uint64{
		"MemTotal": &self.Total,
		"MemFree":  &self.Free,
		"Buffers":  &buffers,
		"Cached":   &cached,
	}

	if err := parseMeminfo(table); err != nil {
		return err
	}

	self.Used = self.Total - self.Free
	kern := buffers + cached
	self.ActualFree = self.Free + kern
	self.ActualUsed = self.Used - kern

	return nil
}

func (self *Swap) Get() error {
	sysinfo := syscall.Sysinfo_t{}

	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return err
	}

	self.Total = sysinfo.Totalswap
	self.Free = sysinfo.Freeswap
	self.Used = self.Total - self.Free

	return nil
}

func (self *FileSystemList) Get() error {
	capacity := len(self.List)
	if capacity == 0 {
		capacity = 10
	}
	fslist := make([]FileSystem, 0, capacity)

	err := readFile("/etc/mtab", func(line string) bool {
		fields := strings.Fields(line)

		fs := FileSystem{}
		fs.DevName = fields[0]
		fs.DirName = fields[1]
		fs.SysTypeName = fields[2]
		fs.Options = fields[3]

		fslist = append(fslist, fs)

		return true
	})

	self.List = fslist

	return err
}

func parseMeminfo(table map[string]*uint64) error {
	return readFile(procd+"meminfo", func(line string) bool {
		fields := strings.Split(line, ":")

		if ptr := table[fields[0]]; ptr != nil {
			num := strings.TrimLeft(fields[1], " ")
			val, err := strtoull(strings.Fields(num)[0])
			if err == nil {
				*ptr = val * 1024
			}
		}

		return true
	})
}

func readFile(file string, handler func(string) bool) error {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(bytes.NewBuffer(contents))

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if !handler(string(line)) {
			break
		}
	}

	return nil
}

func strtoull(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}
