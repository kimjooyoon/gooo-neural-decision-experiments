//go:build !darwin && !linux

package main

import "os"

func processPeakRSS(*os.ProcessState) (*int64, string) {
	return nil, "unavailable for this operating system"
}

func processRSSConvention() string { return "unavailable for this operating system" }
