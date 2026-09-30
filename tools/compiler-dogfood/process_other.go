//go:build !darwin && !linux

package main

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
func peakRSS(cmd *exec.Cmd) int64    { return 0 }
