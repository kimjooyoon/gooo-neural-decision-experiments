//go:build !darwin && !linux

package main

import (
	"os"
	"os/exec"
)

func configureChild(cmd *exec.Cmd)          {}
func childRSS(state *os.ProcessState) int64 { return 0 }
