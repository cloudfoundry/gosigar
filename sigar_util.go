package sigar

import (
	"unsafe"
)

func bytePtrToString(ptr *int8) string { //nolint:unused
	bytes := (*[10000]byte)(unsafe.Pointer(ptr))

	n := 0
	for bytes[n] != 0 {
		n++
	}

	return string(bytes[0:n])
}

func chop(buf []byte) []byte { //nolint:unused
	return buf[0 : len(buf)-1]
}
