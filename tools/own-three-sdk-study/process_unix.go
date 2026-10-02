//go:build darwin || linux

package main

import (
	"runtime"
	"syscall"
)

func usage() syscall.Rusage {
	var r syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &r) != nil {
		return syscall.Rusage{}
	}
	return r
}
func cpuNS() int64 {
	r := usage()
	return (r.Utime.Sec+r.Stime.Sec)*1e9 + (int64(r.Utime.Usec)+int64(r.Stime.Usec))*1000
}
func peakRSS() int64 {
	r := usage()
	n := r.Maxrss
	if runtime.GOOS == "linux" {
		n *= 1024
	}
	return n
}
