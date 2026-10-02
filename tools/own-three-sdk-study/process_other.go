//go:build !darwin && !linux

package main

import "errors"

func cpuNS() int64   { return 0 }
func peakRSS() int64 { return 0 }
func freeBytes(string) (uint64, error) {
	return 0, errors.New("continuation disk accounting requires Darwin or Linux")
}
