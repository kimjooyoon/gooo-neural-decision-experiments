//go:build !darwin && !linux

package main

import (
	"errors"
	"os"
	"os/exec"
)

func cpuNS() int64   { return 0 }
func peakRSS() int64 { return 0 }
func freeBytes(string) (uint64, error) {
	return 0, errors.New("continuation disk accounting requires Darwin or Linux")
}
func configureChild(*exec.Cmd)        {}
func childRSS(*os.ProcessState) int64 { return 0 }
