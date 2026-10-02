//go:build !darwin && !linux

package main

import (
	"os"
	"os/exec"
)

func configure(cmd *exec.Cmd)             {}
func maxRSS(state *os.ProcessState) int64 { return 0 }
