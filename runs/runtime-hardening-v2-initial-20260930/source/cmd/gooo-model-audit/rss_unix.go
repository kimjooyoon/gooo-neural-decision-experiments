//go:build darwin || linux

package main

import (
	"os"
	"runtime"
	"syscall"
)

func processPeakRSS(state *os.ProcessState) (*int64, string) {
	if state == nil {
		return nil, "unavailable"
	}
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage.Maxrss < 0 {
		return nil, "unavailable"
	}
	value := int64(usage.Maxrss)
	if runtime.GOOS == "linux" {
		value *= 1024 // getrusage reports KiB on Linux and bytes on Darwin.
	}
	return &value, "getrusage ru_maxrss normalized to bytes"
}

func processRSSConvention() string {
	return "getrusage ru_maxrss normalized to bytes"
}
