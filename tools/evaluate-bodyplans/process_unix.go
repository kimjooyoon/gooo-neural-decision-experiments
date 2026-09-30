//go:build darwin || linux

package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Each bounded child owns a process group, so cancellation also kills compiler
// and test descendants. exec waits for its direct child and I/O goroutines.
func containCommand(command *exec.Cmd) error {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.WaitDelay = time.Second
	return nil
}
