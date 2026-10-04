//go:build unix

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}
func raw(path string) []byte { b, e := os.ReadFile(path); must(e); return b }
func save(path string, value any) {
	b, e := json.MarshalIndent(value, "", "  ")
	must(e)
	must(os.WriteFile(path, append(b, '\n'), 0600))
}

type result struct {
	Sequence int
	Status   string
	Response struct {
		Source string
		Report struct {
			Compiler  string `json:"compiler_source_sha"`
			BodyPaths struct {
				Search struct {
					Selection struct {
						Local    *int `json:"local_model_predictions"`
						External *int `json:"external_provider_calls"`
					}
				}
			} `json:"body_paths"`
		}
	}
	Execution struct {
		Observation struct {
			Declared   int  `json:"declared_cases"`
			Replayed   bool `json:"runtime_replayed"`
			Projection bool `json:"projection_replayed"`
			Cases      []struct {
				Input, Expected, Actual int64
				Passed                  bool
			}
			Runs []struct{ Started, Completed bool }
		}
	}
}

func main() {
	check(len(os.Args) == 7, "usage worker model-json saved-native frozen-Go fresh-out source")
	worker, model, fixture, expected, out, source := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6]
	must(os.Mkdir(out, 0700))
	goBin := os.Getenv("GOOO_LOCAL_GO")
	check(goBin != "", "missing Go")
	var input bytes.Buffer
	for _, lang := range []string{"ko", "en"} {
		var request map[string]json.RawMessage
		must(json.Unmarshal(raw(filepath.Join(fixture, lang+"-model", "run-1-request.json")), &request))
		request["correlation_id"], _ = json.Marshal("worker-" + lang)
		must(json.NewEncoder(&input).Encode(request))
	}
	must(os.WriteFile(filepath.Join(out, "requests.jsonl"), input.Bytes(), 0600))
	var rows []map[string]any
	for _, mode := range []string{"model", "deterministic"} {
		args := []string{"--workers", "2", "--execute", "--go-bin", goBin}
		if mode == "model" {
			args = append(args, "--model", model)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		c := exec.CommandContext(ctx, worker, args...)
		c.Stdin = bytes.NewReader(input.Bytes())
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		c.Cancel = func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
		c.WaitDelay = 3 * time.Second
		var stdout, stderr bytes.Buffer
		c.Stdout, c.Stderr = &stdout, &stderr
		start := time.Now()
		err := c.Run()
		elapsed := time.Since(start).Nanoseconds()
		timedOut := ctx.Err() != nil
		cancel()
		must(os.WriteFile(filepath.Join(out, mode+".stdout.jsonl"), stdout.Bytes(), 0600))
		must(os.WriteFile(filepath.Join(out, mode+".stderr"), stderr.Bytes(), 0600))
		must(err)
		check(!timedOut, "worker timeout")
		decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
		seen := map[int]bool{}
		predictions, native, passed := 0, 0, 0
		for {
			var r result
			e := decoder.Decode(&r)
			if e == io.EOF {
				break
			}
			must(e)
			check((r.Sequence == 1 || r.Sequence == 2) && !seen[r.Sequence] && r.Status == "completed", "worker result scope")
			seen[r.Sequence] = true
			lang := []string{"ko", "en"}[r.Sequence-1]
			check(r.Response.Report.Compiler == source, "worker source mismatch")
			check(bytes.Equal([]byte(r.Response.Source), raw(filepath.Join(expected, lang+"-"+mode, "run-1-generated.go"))), "worker generated Go changed")
			s := r.Response.Report.BodyPaths.Search.Selection
			want := 0
			if mode == "model" {
				want = 1
			}
			check(s.Local != nil && *s.Local == want && s.External != nil && *s.External == 0, "worker prediction accounting")
			predictions += *s.Local
			o := r.Execution.Observation
			check(o.Declared == 128 && len(o.Cases) == 128 && o.Replayed && o.Projection && len(o.Runs) == 2, "worker execution scope")
			var cases struct {
				Cases []struct{ Input, Expected int64 }
			}
			must(json.Unmarshal(raw(filepath.Join(fixture, lang+"-model", "cases.json")), &cases))
			check(len(cases.Cases) == 128, "original cases changed")
			for i, v := range o.Cases {
				check(v.Passed && v.Input == cases.Cases[i].Input && v.Expected == cases.Cases[i].Expected && v.Actual == v.Expected, "worker finite output changed")
				passed++
			}
			for _, run := range o.Runs {
				check(run.Started && run.Completed, "worker native run incomplete")
				native++
			}
		}
		check(len(seen) == 2 && passed == 256 && native == 4, "worker denominator changed")
		userCPU, systemCPU := c.ProcessState.UserTime().Nanoseconds(), c.ProcessState.SystemTime().Nanoseconds()
		rows = append(rows, map[string]any{"mode": mode, "requests": 2, "workers": 2, "actual_model_predictions": predictions, "native_runs": native, "finite_passed": passed, "finite_total": 256, "process_joined": true, "timeout": false, "total_process_ns": elapsed,
			"cpu_user_ns": userCPU, "cpu_system_ns": systemCPU, "average_cpu_percent_one_core_basis": 100 * float64(userCPU+systemCPU) / float64(elapsed), "cpu_scope": "Go ProcessState exited worker and children; whole setup/construction/execution/EOF interval", "host_cpu_utilization": "UNOBSERVED", "model_only_ram": "UNOBSERVED"})
	}
	save(filepath.Join(out, "summary.json"), map[string]any{"status": "PASS", "source": source, "rows": rows, "requests": 4, "actual_model_predictions": 2, "native_runs": 8, "finite_passed": 512, "finite_total": 512, "new_intent_tasks": 0, "training_updates": 0, "scope": "standalone clean worker, two concurrent authored KO/EN requests per mode; total process time includes setup and EOF joining"})
	fmt.Println("Standalone worker: model/deterministic concurrent construction and native replay PASS.")
}
