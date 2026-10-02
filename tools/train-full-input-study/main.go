// Go launches exactly one offline optimizer and records its terminal resources.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func main() {
	python := flag.String("python", "", "existing local MPS environment executable")
	prepared := flag.String("prepared", "", "prepared Go feature bank")
	audit := flag.String("preparation-audit", "", "independent Go preparation replay")
	storage := flag.String("storage-preflight", "", "fresh Go storage inventory")
	output := flag.String("output", "", "fresh own-three-full-input phase")
	revision := flag.String("source-revision", "", "exact published source")
	flag.Parse()
	if err := run(*python, *prepared, *audit, *storage, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(python, prepared, audit, storage, output, revision string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision || runtime.Version() != "go1.27.1" {
		return errors.New("exact Go 1.27.1 source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean published source required")
	}
	absolute, err := filepath.Abs(output)
	if err != nil || filepath.Dir(absolute) != filepath.Join(root, "runs") || !strings.HasPrefix(filepath.Base(absolute), "own-three-full-input-") {
		return errors.New("fresh full-input phase under runs required")
	}
	if _, err = os.Lstat(absolute); !os.IsNotExist(err) {
		return errors.New("existing optimizer prefix must be retained")
	}
	protocol, err := os.ReadFile(fullinputstudy.Protocol)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(python) || prepared == "" || audit == "" || storage == "" {
		return errors.New("existing MPS executable and complete Go preparation required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-B", "training/train_full_input_judgment_v1.py",
		"--prepared", prepared, "--preparation-audit", audit, "--storage-preflight", storage,
		"--output", output, "--source-revision", revision, "--protocol-sha256", threecohort.SHA(protocol))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.WaitDelay = 10 * time.Second
	configureChild(cmd)
	started := time.Now()
	err = cmd.Run()
	wall := time.Since(started).Seconds()
	status, code, cpu, rss := "FAILED_PREFIX_RETAINED", -1, 0.0, int64(0)
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
		cpu = (cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()).Seconds()
		rss = childRSS(cmd.ProcessState)
	}
	if err == nil {
		status = "OPTIMIZER_PROCESS_COMPLETED"
	}
	report := map[string]any{"schema": "gooo/full-input-optimizer-process/v1", "status": status,
		"source_revision": revision, "protocol_sha256": threecohort.SHA(protocol),
		"started_at": started.UTC().Format(time.RFC3339Nano), "exit_code": code,
		"wall_seconds": wall, "process_cpu_seconds": cpu,
		"process_cpu_percent_one_core": 100 * cpu / wall, "process_peak_rss_bytes": rss,
		"deadline_exceeded": ctx.Err() != nil, "automatic_retries": 0,
		"gpu_utilization_measured": false, "host_cpu_causal_delta_measured": false,
		"scope": "One offline MPS optimizer process, including input validation and all four arms. Stage/update results are in its journals."}
	if e := os.MkdirAll(absolute, 0755); e != nil {
		return e
	}
	raw, e := json.MarshalIndent(report, "", "  ")
	if e != nil || len(raw) > 16<<10 {
		return errors.New("bounded terminal process report required")
	}
	f, e := os.OpenFile(filepath.Join(absolute, "optimizer-process.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return e
	}
	_, e = f.Write(append(raw, '\n'))
	closed := f.Close()
	if e != nil {
		return e
	}
	if closed != nil {
		return closed
	}
	fmt.Println(string(raw))
	return err
}
