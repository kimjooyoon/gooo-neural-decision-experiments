package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func check(v bool, s string) {
	if !v {
		panic(s)
	}
}
func decode(p string) map[string]any {
	b, e := os.ReadFile(p)
	must(e)
	var o map[string]any
	must(json.Unmarshal(b, &o))
	return o
}
func main() {
	check(len(os.Args) == 5, "usage probe compiler inputs output source-sha")
	compiler, inputs, out, source := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	must(os.Mkdir(out, 0700))
	root, e := os.MkdirTemp("", "gooo-fifo-fixed-")
	must(e)
	defer os.RemoveAll(root)
	fifo := filepath.Join(root, "go-fifo")
	must(syscall.Mkfifo(fifo, 0700))
	result := filepath.Join(out, "results")
	args := []string{"body-path-run", "--source", filepath.Join(inputs, "ko-source.gooo"), "--activity", "AssembleKorean", "--path-plan", filepath.Join(inputs, "ko-recipe.json"), "--cases", filepath.Join(inputs, "cases-128.json"), "--go-bin", fifo, "--out", result, "--repeat", "1", "--timing"}
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	c := exec.CommandContext(ctx, compiler, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
	c.WaitDelay = 2 * time.Second
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	start := time.Now()
	e = c.Run()
	elapsed := time.Since(start).Nanoseconds()
	must(os.WriteFile(filepath.Join(out, "stdout.raw"), stdout.Bytes(), 0600))
	must(os.WriteFile(filepath.Join(out, "stderr.raw"), stderr.Bytes(), 0600))
	text := ""
	if e != nil {
		text = e.Error()
	}
	b, e := json.MarshalIndent(map[string]any{"exit_code": c.ProcessState.ExitCode(), "external_timeout": ctx.Err() != nil, "elapsed_ns": elapsed, "process_error": text, "process_joined": true}, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(out, "process.json"), append(b, '\n'), 0600))
	check(c.ProcessState.ExitCode() == 1 && ctx.Err() == nil, "wrong failure exit or timeout")
	check(strings.Contains(stderr.String(), "regular executable file") && strings.Contains(stderr.String(), "native unobserved"), "failure diagnostics")
	timing := decode(filepath.Join(result, "run-1-timing.json"))
	check(timing["producer_source_sha"] == source && timing["producer_modified"] == "false", "clean producer")
	check(timing["status"] == "execution_failed", "failure status changed")
	rows := decode(filepath.Join(result, "summary.json"))["rows"].([]any)
	check(len(rows) == 1 && rows[0].(map[string]any)["native_runs"] == float64(0), "native accounting")
	generation := decode(filepath.Join(result, "run-1-generation.json"))
	report := generation["report"].(map[string]any)
	selection := report["body_paths"].(map[string]any)["search"].(map[string]any)["selection"].(map[string]any)
	check(selection["local_model_predictions"] == float64(0) && selection["external_provider_calls"] == float64(0), "model accounting")
	verify := exec.CommandContext(ctx, compiler, "body-path-run", "--verify-timing", "--out", result)
	readback, e := verify.CombinedOutput()
	must(os.WriteFile(filepath.Join(out, "readback.raw"), readback, 0600))
	must(e)
	binary, e := os.ReadFile(compiler)
	must(e)
	record := map[string]any{"status": "EXPECTED_FAILURE_RETAINED", "compiler_source": source, "compiler_sha256": fmt.Sprintf("%x", sha256.Sum256(binary)), "elapsed_ns": elapsed, "actual_model_predictions": 0, "native_runs": 0, "finite_expectations": "128 unobserved", "new_intent_tasks": 0, "training_updates": 0, "read_only_timing_checks": 1, "process_joined": true, "external_timeout": false}
	b, e = json.MarshalIndent(record, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(out, "summary.json"), append(b, '\n'), 0600))
}
