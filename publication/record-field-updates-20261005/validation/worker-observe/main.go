package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func require(v bool) {
	if !v {
		panic("sequential worker observation mismatch")
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); must(e); return b }
func sha(b []byte) string  { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func save(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(p, append(b, '\n'), 0644))
}
func main() {
	require(len(os.Args) == 6)
	worker, sourceDir, model, out, source := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]
	info, e := buildinfo.ReadFile(worker)
	must(e)
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	require(info.GoVersion == "go1.27.1" && settings["vcs.revision"] == source && settings["vcs.modified"] == "false")
	found := false
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			require(d.Version == "v0.2.23-experimental" && d.Replace == nil)
			found = true
		}
	}
	require(found)
	_, e = os.Stat(out)
	require(os.IsNotExist(e))
	must(os.MkdirAll(out, 0755))
	entries, e := os.ReadDir(sourceDir)
	must(e)
	var sources []string
	var identities []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "-qat_ternary-b1.gooo.fixture") {
			b := read(filepath.Join(sourceDir, entry.Name()))
			sources = append(sources, string(b))
			identities = append(identities, sha(b))
		}
	}
	require(len(sources) == 12)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, worker, "--workers", "2", "--model", model)
	stdin, e := cmd.StdinPipe()
	must(e)
	stdout, e := cmd.StdoutPipe()
	must(e)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	started := time.Now()
	must(cmd.Start())
	first := make(chan struct{})
	writes := make(chan error, 1)
	go func() {
		defer stdin.Close()
		for i := range 32 {
			b, e := json.Marshal(map[string]any{"schema": "gooo/native-body-stream-request/v1", "correlation_id": fmt.Sprintf("sequential-%d", i), "source": sources[i%len(sources)], "activity": "Select", "options": map[string]any{}})
			if e == nil {
				_, e = stdin.Write(append(b, '\n'))
			}
			if e != nil {
				writes <- e
				return
			}
			if i == 0 {
				select {
				case <-first:
				case <-ctx.Done():
					writes <- ctx.Err()
					return
				}
			}
		}
		writes <- nil
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var responses bytes.Buffer
	seen := map[string]bool{}
	fields := 0
	firstNS := int64(0)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var r struct {
			Status   string `json:"status"`
			ID       string `json:"correlation_id"`
			Response struct {
				Report struct {
					Assembly struct {
						Calls  int `json:"model_calls"`
						Fields int `json:"fields_passed"`
						Total  int `json:"fields_total"`
					} `json:"record_assembly"`
				} `json:"report"`
			} `json:"response"`
		}
		must(json.Unmarshal(line, &r))
		a := r.Response.Report.Assembly
		require(r.Status == "completed" && !seen[r.ID] && a.Calls == 1 && a.Fields == 12 && a.Total == 12)
		if len(seen) == 0 {
			require(r.ID == "sequential-0")
			firstNS = time.Since(started).Nanoseconds()
			close(first)
		}
		seen[r.ID] = true
		fields += a.Fields
		responses.Write(line)
		responses.WriteByte('\n')
	}
	must(scanner.Err())
	must(<-writes)
	must(cmd.Wait())
	require(len(seen) == 32 && fields == 384)
	rss := cmd.ProcessState.SysUsage().(*syscall.Rusage).Maxrss
	if runtime.GOOS != "darwin" {
		rss *= 1024
	}
	must(os.WriteFile(filepath.Join(out, "responses.jsonl"), responses.Bytes(), 0644))
	must(os.WriteFile(filepath.Join(out, "setup.json"), stderr.Bytes(), 0644))
	save(filepath.Join(out, "summary.json"), map[string]any{"schema": "gooo/sequential-record-worker-observation/v1", "compiler_source": source, "worker_sha256": sha(read(worker)), "metadata_sha256": sha(read(model)), "weights_sha256": sha(read(filepath.Join(filepath.Dir(model), "weights.bin"))), "source_sha256": identities, "workers": 2, "requests_completed": 32, "selection_fields_passed": fields, "selection_fields_total": 384, "model_calls": 32, "first_response_before_input_close": true, "first_response_wall_ns": firstNS, "whole_wall_ns": time.Since(started).Nanoseconds(), "process_cpu_ns": (cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()).Nanoseconds(), "peak_rss_bytes": rss, "responses_sha256": sha(responses.Bytes()), "platform": runtime.GOOS, "deadline_seconds": 60, "scope": "six shapes times two languages, all goal mask7; worker selection only, native execution measured separately; bounded observation"})
	fmt.Println("32 requests, 384/384 selection fields; first response before input closed.")
}
