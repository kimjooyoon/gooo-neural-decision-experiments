//go:build darwin || linux

package main

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestCanceledChildDoesNotHoldPipes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "sleep 5")
	configure(cmd)
	cmd.WaitDelay = time.Second
	var output bounded
	cmd.Stdout, cmd.Stderr = &output, &output
	start := time.Now()
	if err := cmd.Run(); err == nil {
		t.Fatal("canceled child succeeded")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatal("child/pipes retained past bound", elapsed)
	}
}

func TestChildBufferRejectsAtomicOverflow(t *testing.T) {
	var b bounded
	if n, err := b.Write(make([]byte, 1<<20)); err != nil || n != 1<<20 {
		t.Fatal(n, err)
	}
	if n, err := b.Write([]byte("x")); err == nil || n != 0 || b.Len() != 1<<20 {
		t.Fatal("partial overflow write", n, err)
	}
}
