// Actual CLI controls: unobserved failures versus deliberately changed expectations.
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

type object = map[string]any

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func check(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func raw(path string) []byte          { b, err := os.ReadFile(path); must(err); return b }
func decode(path string) object       { var o object; must(json.Unmarshal(raw(path), &o)); return o }
func obj(o object, key string) object { return o[key].(map[string]any) }
func write(path string, value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	must(err)
	must(os.WriteFile(path, append(b, '\n'), 0600))
}
func hash(path string) string { return fmt.Sprintf("%x", sha256.Sum256(raw(path))) }
func run(binary string, args []string, out, label string) (int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, binary, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
	c.WaitDelay = 3 * time.Second
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	started := time.Now()
	err := c.Run()
	elapsed := time.Since(started).Nanoseconds()
	must(os.WriteFile(filepath.Join(out, label+".stdout"), stdout.Bytes(), 0600))
	must(os.WriteFile(filepath.Join(out, label+".stderr"), stderr.Bytes(), 0600))
	text := ""
	if err != nil {
		text = err.Error()
	}
	check(c.ProcessState != nil, "no process state")
	write(filepath.Join(out, label+"-process.json"), object{"elapsed_ns": elapsed, "exit_code": c.ProcessState.ExitCode(), "error": text, "timeout": ctx.Err() != nil, "process_joined": true})
	check(ctx.Err() == nil, "timeout")
	return c.ProcessState.ExitCode(), stderr.String()
}
func main() {
	check(len(os.Args) == 7, "usage probe compiler inputs model frozen-Go output source-sha")
	compiler, inputs, model, expected, out, source := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6]
	must(os.Mkdir(out, 0700))
	goTool := os.Getenv("GOOO_LOCAL_GO")
	check(goTool != "", "set GOOO_LOCAL_GO")
	root, err := os.MkdirTemp("", "gooo-expectation-controls-")
	must(err)
	defer os.RemoveAll(root)
	fifo := filepath.Join(root, "go-fifo")
	must(syscall.Mkfifo(fifo, 0700))
	var suite struct {
		Schema string `json:"schema"`
		Cases  []struct {
			Input    int64 `json:"input"`
			Expected int64 `json:"expected"`
		} `json:"cases"`
	}
	originalCases := filepath.Join(inputs, "cases-128.json")
	must(json.Unmarshal(raw(originalCases), &suite))
	check(len(suite.Cases) == 128, "case count")
	for i := range suite.Cases {
		suite.Cases[i].Expected++
	}
	changed := filepath.Join(out, "changed-expectations.json")
	write(changed, suite)
	invalid := filepath.Join(out, "invalid-expectations.json")
	must(os.WriteFile(invalid, []byte("{\"schema\":\"gooo/body-runtime-cases/v1\",\"cases\":[{\"input\":2}]}\n"), 0600))
	before := object{"compiler": hash(compiler), "weights": hash(filepath.Join(filepath.Dir(model), "weights.bin")), "original_cases": hash(originalCases), "source": hash(filepath.Join(inputs, "ko-source.gooo")), "recipe": hash(filepath.Join(inputs, "ko-recipe.json"))}
	write(filepath.Join(out, "bindings-before.json"), before)
	var rows []object
	generations, predictions, native, finite := 0, 0, 0, 0
	for _, tc := range []struct {
		name, cases, tool, mode, display string
		exit, native                     int
	}{
		{"missing-tool", originalCases, filepath.Join(root, "absent-go"), "deterministic", "unobserved (128 declared)", 1, 0},
		{"invalid-expectations", invalid, goTool, "deterministic", "unobserved", 1, 0},
		{"fifo-tool", originalCases, fifo, "deterministic", "unobserved (128 declared)", 1, 0},
		{"observed-zero-deterministic", changed, goTool, "deterministic", "0/128", 0, 2},
		{"observed-zero-model", changed, goTool, "model", "0/128", 0, 2},
	} {
		dest := filepath.Join(out, tc.name)
		args := []string{"body-path-run", "--source", filepath.Join(inputs, "ko-source.gooo"), "--activity", "AssembleKorean", "--path-plan", filepath.Join(inputs, "ko-recipe.json"), "--cases", tc.cases, "--go-bin", tc.tool, "--out", dest, "--repeat", "1", "--timing"}
		if tc.mode == "model" {
			args = append(args, "--model", model)
		}
		code, stderr := run(compiler, args, out, tc.name)
		check(code == tc.exit, "exit changed")
		check(strings.Contains(stderr, "finite expectations "+tc.display), "finite display")
		if tc.native == 0 {
			check(!strings.Contains(stderr, "finite expectations 0/"), "invented observed zero")
		} else {
			check(!strings.Contains(stderr, "unobserved"), "real zero hidden")
		}
		timing := decode(filepath.Join(dest, "run-1-timing.json"))
		check(timing["producer_source_sha"] == source && timing["producer_modified"] == "false", "clean producer")
		row := decode(filepath.Join(dest, "summary.json"))["rows"].([]any)[0].(object)
		check(row["native_runs"] == float64(tc.native) && row["passed"] == float64(0), "legacy row changed")
		wantTotal := float64(128)
		if tc.name == "invalid-expectations" {
			wantTotal = 0
		}
		check(row["total"] == wantTotal, "declared JSON denominator changed")
		prediction := 0
		if tc.name == "invalid-expectations" {
			check(row["status"] == "rejected" && row["generation_file"] == nil && row["runtime_file"] == nil, "rejection reached generation")
		} else {
			generations++
			generation := decode(filepath.Join(dest, "run-1-generation.json"))
			selection := obj(obj(obj(obj(generation, "report"), "body_paths"), "search"), "selection")
			prediction = int(selection["local_model_predictions"].(float64))
			want := 0
			if tc.mode == "model" {
				want = 1
			}
			check(prediction == want && selection["external_provider_calls"] == float64(0), "prediction accounting")
			check(bytes.Equal(raw(filepath.Join(dest, "run-1-generated.go")), raw(filepath.Join(expected, "ko-"+tc.mode, "run-1-generated.go"))), "generation changed")
			runtime := decode(filepath.Join(dest, "run-1-runtime.json"))
			observation := obj(runtime, "observation")
			cases := observation["cases"].([]any)
			check(len(cases) == tc.native*64, "observed case count")
			if tc.native > 0 {
				check(row["status"] == "completed" && observation["runtime_replayed"] == true, "actual zero incomplete")
				for _, value := range cases {
					c := value.(object)
					check(c["passed"] == false && c["expected"].(float64) == c["actual"].(float64)+1, "changed expectation not retained")
				}
				finite += len(cases)
			} else {
				check(row["status"] == "execution_failed", "failed status changed")
			}
		}
		verify, _ := run(compiler, []string{"body-path-run", "--verify-timing", "--out", dest}, out, tc.name+"-verify")
		check(verify == 0, "timing binding failure")
		predictions += prediction
		native += tc.native
		rows = append(rows, object{"condition": tc.name, "mode": tc.mode, "original_status": row["status"], "declared_total": row["total"], "passed": 0, "observed_cases": tc.native * 64, "native_runs": tc.native, "model_predictions": prediction, "finite_display": tc.display, "exit_code": code, "response_ms": timing["response_ns"].(float64) / 1e6})
	}
	check(generations == 4 && predictions == 1 && native == 4 && finite == 256, "scope accounting")
	check(before["compiler"] == hash(compiler) && before["weights"] == hash(filepath.Join(filepath.Dir(model), "weights.bin")) && before["original_cases"] == hash(originalCases), "bindings changed")
	write(filepath.Join(out, "summary.json"), object{"status": "PASS_WITH_ORIGINAL_FAILURES_AND_INTENTIONALLY_CHANGED_EXPECTATIONS", "compiler_source": source, "compiler_sha256": hash(compiler), "requests": 5, "generations": generations, "actual_model_predictions": predictions, "native_runs": native, "intentionally_changed_expectations_observed": finite, "intentionally_changed_expectations_passed": 0, "read_only_timing_checks": 5, "new_intent_tasks": 0, "training_updates": 0, "synthetic_partial_native_cases": 0, "rows": rows})
}
