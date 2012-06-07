// Copyright (c) 2012 VMware, Inc.

package sigar

import (
	"testing"
)

func TestLoadAverage(t *testing.T) {
	avg := LoadAverage{}
	if err := avg.Get(); err != nil {
		t.Error(err)
	}
}

func TestUptime(t *testing.T) {
	uptime := Uptime{}
	if err := uptime.Get(); err != nil {
		t.Error(err)
	}
	if uptime.Length <= 0 {
		t.Errorf("Invalid uptime.Length=%d", uptime.Length)
	}
}

func TestMem(t *testing.T) {
	mem := Mem{}
	err := mem.Get()
	if err != nil {
		t.Error(err)
	}
	if mem.Total <= 0 {
		t.Errorf("Invalid mem.Total=%d", mem.Total)
	}

	if (mem.Used + mem.Free) > mem.Total {
		t.Errorf("Invalid mem.Used=%d or mem.Free=%d",
			mem.Used, mem.Free)
	}
}

func TestSwap(t *testing.T) {
	swap := Swap{}
	err := swap.Get()
	if err != nil {
		t.Error(err)
	}
	if (swap.Used + swap.Free) > swap.Total {
		t.Errorf("Invalid swap.Used=%d or swap.Free=%d",
			swap.Used, swap.Free)
	}
}

func TestFileSystemList(t *testing.T) {
	fslist := FileSystemList{}
	err := fslist.Get()
	if err != nil {
		t.Error(err)
	}

	if len(fslist.List) <= 0 {
		t.Error("Empty FileSystemList")
	}
}

func TestFileSystemUsage(t *testing.T) {
	fsusage := FileSystemUsage{}
	err := fsusage.Get("/")
	if err != nil {
		t.Error(err)
	}

	err = fsusage.Get("T O T A L L Y B O G U S")
	if err == nil {
		t.Error("FileSystemUsage.Get should have failed")
	}
}
