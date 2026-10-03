package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

type cost struct {
	WallMS, CPUms, OneCoreCPUPercent float64
	PeakRSSBytes                     int64
}

func execute(binary, dir, output string, args ...string) cost {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, binary, args...)
	c.Dir, c.WaitDelay = dir, 5*time.Second
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		err := syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	started := time.Now()
	err := c.Run()
	elapsed := time.Since(started)
	must(os.WriteFile(filepath.Join(dir, output), stdout.Bytes(), 0644))
	if err != nil {
		must(os.WriteFile(filepath.Join(dir, output+".stderr"), stderr.Bytes(), 0644))
		panic(fmt.Sprintf("%s: %v: %s", output, err, stderr.String()))
	}
	r := cost{WallMS: float64(elapsed) / 1e6, CPUms: float64(c.ProcessState.UserTime()+c.ProcessState.SystemTime()) / 1e6}
	r.OneCoreCPUPercent = 100 * r.CPUms / r.WallMS
	if u, ok := c.ProcessState.SysUsage().(*syscall.Rusage); ok {
		r.PeakRSSBytes = u.Maxrss
		if runtime.GOOS == "linux" {
			r.PeakRSSBytes *= 1024
		}
	}
	return r
}
