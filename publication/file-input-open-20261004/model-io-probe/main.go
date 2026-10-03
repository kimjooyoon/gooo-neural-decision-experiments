//go:build unix

// Read-only follow-up: static model FIFOs, with owned writer release and joining.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	if len(os.Args) != 5 {
		panic("usage: compiler inputs public-model-dir fresh-output")
	}
	compiler, input, model, out := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	must(os.Mkdir(out, 0700))
	root, err := os.MkdirTemp("", "gooo-model-io-")
	must(err)
	defer os.RemoveAll(root)
	var rows []map[string]any
	for _, mode := range []string{"metadata", "weights"} {
		dir := filepath.Join(root, mode)
		must(os.Mkdir(dir, 0700))
		for _, name := range []string{"model.json", "weights.bin"} {
			path := filepath.Join(dir, name)
			if (mode == "metadata" && name == "model.json") || (mode == "weights" && name == "weights.bin") {
				must(syscall.Mkfifo(path, 0600))
			} else {
				b, e := os.ReadFile(filepath.Join(model, name))
				must(e)
				must(os.WriteFile(path, b, 0600))
			}
		}
		fifo := filepath.Join(dir, "model.json")
		if mode == "weights" {
			fifo = filepath.Join(dir, "weights.bin")
		}
		dest := filepath.Join(out, mode+"-results")
		args := []string{"body-path-run", "--source", filepath.Join(input, "ko-source.gooo"), "--activity", "AssembleKorean", "--path-plan", filepath.Join(input, "ko-recipe.json"), "--cases", filepath.Join(input, "cases-128.json"), "--model", filepath.Join(dir, "model.json"), "--go-bin", os.Getenv("GOOO_LOCAL_GO"), "--out", dest}
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		c := exec.CommandContext(ctx, compiler, args...)
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		c.Cancel = func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
		c.WaitDelay = time.Second
		var stdout, stderr bytes.Buffer
		c.Stdout, c.Stderr = &stdout, &stderr
		started := time.Now()
		must(c.Start())
		done := make(chan error, 1)
		go func() { done <- c.Wait() }()
		released := false
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			writer, e := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0)
			if e == nil {
				released = true
				must(writer.Close())
			}
			<-done
		}
		timedOut := ctx.Err() != nil
		cancel()
		must(os.WriteFile(filepath.Join(out, mode+".stdout"), stdout.Bytes(), 0600))
		must(os.WriteFile(filepath.Join(out, mode+".stderr"), stderr.Bytes(), 0600))
		_, e := os.Stat(dest)
		absent := os.IsNotExist(e)
		row := map[string]any{"condition": mode + "-fifo", "elapsed_ns": time.Since(started).Nanoseconds(), "pid": c.Process.Pid, "process_joined": true, "writer_released_fifo_reader": released, "timeout": timedOut, "exit_code": c.ProcessState.ExitCode(), "output_directory_absent": absent, "stdout_bytes": stdout.Len(), "stderr": stderr.String(), "args": args, "source_generation_observed": false}
		b, e := json.MarshalIndent(row, "", "  ")
		must(e)
		must(os.WriteFile(filepath.Join(out, mode+"-process.json"), append(b, '\n'), 0600))
		rows = append(rows, row)
		if timedOut {
			panic("timeout; inconclusive")
		}
	}
	b, err := json.MarshalIndent(map[string]any{"scope": "actual clean installed compiler; controlled release of owned model FIFO; no compiler/model mutation", "rows": rows, "new_intent_tasks": 0, "training_updates": 0}, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(out, "summary.json"), append(b, '\n'), 0600))
	fmt.Println(string(b))
}
