//go:build linux
// +build linux

package memory

import (
	"runtime/debug"
	"syscall"
)

const (
	minMemoryLimit int64 = 30 * 1024 * 1024
)

func GetAvailableMemoryMB() (uint64, error) {
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		return 0, err
	}

	//log.Println("free", info.Freeram/1024/1024, "swap", info.Freeswap/1024/1024, "unit", info.Unit)
	return uint64(info.Freeram+info.Freeswap) * uint64(info.Unit), nil
}

func SetDynamicMemoryLimit(percentage float64) {
	totalBytes, err := GetAvailableMemoryMB()

	if err == nil {
		// Calculate limit (e.g., 75% of total system RAM)
		limit := int64(float64(totalBytes) * percentage)
		if limit < minMemoryLimit {
			limit = minMemoryLimit
		}
		//log.Println("memory limit:", limit)
		debug.SetMemoryLimit(limit)
	}
}
