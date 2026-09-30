//go:build !darwin && !linux

package main

import (
	"os"
	"os/exec"
)

func configureProcess(cmd *exec.Cmd)               {}
func peakRSS(state *os.ProcessState) (int64, bool) { return 0, false }
