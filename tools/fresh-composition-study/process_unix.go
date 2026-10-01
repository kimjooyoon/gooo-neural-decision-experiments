//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func configure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
func maxRSS(state *os.ProcessState) int64 {
	rusage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	value := rusage.Maxrss
	if runtime.GOOS == "linux" {
		value *= 1024
	}
	return value
}
