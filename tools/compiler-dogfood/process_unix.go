//go:build darwin || linux

package main

import (
	"os/exec"
	"runtime"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func peakRSS(cmd *exec.Cmd) int64 {
	if cmd.ProcessState == nil {
		return 0
	}
	usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	value := usage.Maxrss
	if runtime.GOOS == "linux" {
		value *= 1024
	}
	return value
}
