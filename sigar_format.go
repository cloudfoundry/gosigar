// Copyright (c) 2012 VMware, Inc.

package sigar

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
)

//apr_strfsize
func FormatSize(size uint64) string {
	ord := []string{"K", "M", "G", "T", "P", "E"}
	o := 0
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)

	if size < 973 {
		fmt.Fprintf(w, "%3d ", size)
		w.Flush()
		return buf.String()
	}

	for {
		remain := size & 1023
		size >>= 10

		if size >= 973 {
			o++
			continue
		}

		if size < 9 || (size == 9 && remain < 973) {
			remain = ((remain * 5) + 256) / 512
			if remain >= 10 {
				size++
				remain = 0
			}

			fmt.Fprintf(w, "%d.%d%s", size, remain, ord[o])
			break
		}

		if remain >= 512 {
			size++
		}

		fmt.Fprintf(w, "%3d%s", size, ord[o])
		break
	}

	w.Flush()
	return buf.String()
}

func FormatPercent(percent float64) string {
	return strconv.FormatFloat(percent, 'f', -1, 64) + "%"
}

func (self *FileSystemUsage) UsePercent() float64 {
	b_used := (self.Total - self.Free) / 1024
	b_avail := self.Avail / 1024
	utotal := b_used + b_avail
	used := b_used

	if utotal != 0 {
		u100 := used * 100
		pct := u100 / utotal
		if u100%utotal != 0 {
			pct += 1
		}
		return (float64(pct) / float64(100)) * 100.0
	}

	return 0.0
}

func (self *Uptime) Format() string {
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	uptime := uint64(self.Length)

	days := uptime / (60 * 60 * 24)

	if days != 0 {
		s := ""
		if days > 1 {
			s = "s"
		}
		fmt.Fprintf(w, "%d day%s, ", days, s)
	}

	minutes := uptime / 60
	hours := minutes / 60
	hours %= 24
	minutes %= 60

	fmt.Fprintf(w, "%2d:%02d", hours, minutes)

	w.Flush()
	return buf.String()
}
