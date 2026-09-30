//go:build !darwin && !linux

package main

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
