//go:build !darwin && !linux

package main

func cpuNS() int64   { return 0 }
func peakRSS() int64 { return 0 }
