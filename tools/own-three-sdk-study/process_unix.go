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

func freeBytes(path string) (uint64, error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return 0, err
	}
	return uint64(s.Bavail) * uint64(s.Bsize), nil
}
