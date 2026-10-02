//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func configureChild(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func childRSS(state *os.ProcessState) int64 {
	r, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	n := r.Maxrss
	if runtime.GOOS == "linux" {
		n *= 1024
	}
	return n
}
