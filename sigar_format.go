package sigar

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"time"
)

// FormatSize Go version of apr_strfsize
func FormatSize(size uint64) string {
	ord := []string{"K", "M", "G", "T", "P", "E"}
	o := 0
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)

	if size < 973 {
		fmt.Fprintf(w, "%3d ", size) //nolint:errcheck
		w.Flush()                    //nolint:errcheck
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

			fmt.Fprintf(w, "%d.%d%s", size, remain, ord[o]) //nolint:errcheck
			break
		}

		if remain >= 512 {
			size++
		}

		fmt.Fprintf(w, "%3d%s", size, ord[o]) //nolint:errcheck
		break
	}

	w.Flush() //nolint:errcheck
	return buf.String()
}

func FormatPercent(percent float64) string {
	return strconv.FormatFloat(percent, 'f', -1, 64) + "%"
}

func (fs *FileSystemUsage) UsePercent() float64 { //nolint:staticcheck
	bUsed := (fs.Total - fs.Free) / 1024
	bAvail := fs.Avail / 1024
	uTotal := bUsed + bAvail
	used := bUsed

	if uTotal != 0 {
		u100 := used * 100
		pct := u100 / uTotal
		if u100%uTotal != 0 {
			pct += 1
		}
		return (float64(pct) / float64(100)) * 100.0
	}

	return 0.0
}

func (u *Uptime) Format() string { //nolint:staticcheck
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	uptime := uint64(u.Length)

	days := uptime / (60 * 60 * 24)

	if days != 0 {
		s := ""
		if days > 1 {
			s = "s"
		}
		fmt.Fprintf(w, "%d day%s, ", days, s) //nolint:errcheck
	}

	minutes := uptime / 60
	hours := minutes / 60
	hours %= 24
	minutes %= 60

	fmt.Fprintf(w, "%2d:%02d", hours, minutes) //nolint:errcheck

	w.Flush() //nolint:errcheck
	return buf.String()
}

func (pt *ProcTime) FormatStartTime() string { //nolint:staticcheck
	if pt.StartTime == 0 {
		return "00:00"
	}
	start := time.Unix(int64(pt.StartTime)/1000, 0)
	format := "Jan02"
	if time.Since(start).Seconds() < (60 * 60 * 24) {
		format = "15:04"
	}
	return start.Format(format)
}

func (pt *ProcTime) FormatTotal() string { //nolint:staticcheck
	t := pt.Total / 1000
	ss := t % 60
	t /= 60
	mm := t % 60
	t /= 60
	hh := t % 24
	return fmt.Sprintf("%02d:%02d:%02d", hh, mm, ss)
}
