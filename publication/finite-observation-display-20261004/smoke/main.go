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

type obj = map[string]any

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func check(b bool, s string) {
	if !b {
		panic(s)
	}
}
func raw(p string) []byte        { b, e := os.ReadFile(p); must(e); return b }
func decode(p string) obj        { var o obj; must(json.Unmarshal(raw(p), &o)); return o }
func object(o obj, k string) obj { return o[k].(map[string]any) }
func run(binary string, args []string, out, name string) {
	ctx, stop := context.WithTimeout(context.Background(), 90*time.Second)
	defer stop()
	c := exec.CommandContext(ctx, binary, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
	c.WaitDelay = 3 * time.Second
	stdout, e := os.Create(filepath.Join(out, name+".stdout"))
	must(e)
	defer stdout.Close()
	stderr, e := os.Create(filepath.Join(out, name+".stderr"))
	must(e)
	defer stderr.Close()
	c.Stdout = stdout
	c.Stderr = stderr
	must(c.Run())
}

func missingTool(compiler, inputs, expected, out, source string) {
	dest := filepath.Join(out, "missing-tool")
	args := []string{"body-path-run", "--source", filepath.Join(inputs, "ko-source.gooo"),
		"--activity", "AssembleKorean", "--path-plan", filepath.Join(inputs, "ko-recipe.json"),
		"--cases", filepath.Join(inputs, "cases-128.json"), "--go-bin", filepath.Join(out, "absent-go"),
		"--out", dest, "--repeat", "1", "--timing"}
	ctx, stop := context.WithTimeout(context.Background(), 90*time.Second)
	defer stop()
	c := exec.CommandContext(ctx, compiler, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	e := c.Run()
	must(os.WriteFile(filepath.Join(out, "missing-tool.stdout"), stdout.Bytes(), 0600))
	must(os.WriteFile(filepath.Join(out, "missing-tool.stderr"), stderr.Bytes(), 0600))
	check(e != nil && c.ProcessState.ExitCode() == 1, "missing tool exit")
	check(strings.Contains(stderr.String(), "native unobserved"), "missing phase displayed as zero")
	check(strings.Contains(stderr.String(), "finite expectations unobserved (128 declared)"), "missing expectations displayed as observed failures")
	timing := decode(filepath.Join(dest, "run-1-timing.json"))
	check(timing["producer_source_sha"] == source && timing["producer_modified"] == "false", "failed clean binding")
	check(timing["status"] == "execution_failed", "original failed status")
	rows := decode(filepath.Join(dest, "summary.json"))["rows"].([]any)
	check(len(rows) == 1 && rows[0].(obj)["native_runs"] == float64(0), "failed native accounting")
	generation := decode(filepath.Join(dest, "run-1-generation.json"))
	selection := object(object(object(object(generation, "report"), "body_paths"), "search"), "selection")
	check(selection["local_model_predictions"] == float64(0) && selection["external_provider_calls"] == float64(0), "failed model accounting")
	for _, phase := range object(timing, "wall")["phases"].([]any) {
		check(!strings.HasPrefix(phase.(obj)["name"].(string), "native_run_"), "native work invented")
	}
	check(bytes.Equal(raw(filepath.Join(dest, "run-1-generated.go")), raw(filepath.Join(expected, "ko-deterministic", "run-1-generated.go"))), "failed lookup changed generation")
	run(compiler, []string{"body-path-run", "--verify-timing", "--out", dest}, out, "missing-tool-verify")
	b, e := json.MarshalIndent(obj{"status": "EXPECTED_FAILURE_RETAINED", "requests": 1,
		"actual_model_predictions": 0, "native_runs": 0, "runtime_expectations": "128 unobserved",
		"exit_code": 1, "native_display": "unobserved", "finite_expectation_display": "unobserved (128 declared)",
		"read_only_timing_checks": 1}, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(out, "missing-tool-summary.json"), append(b, '\n'), 0600))
}
func main() {
	check(len(os.Args) == 7, "usage smoke compiler inputs model expected out source-sha")
	compiler, inputs, model, expected, out, source := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6]
	must(os.Mkdir(out, 0700))
	goTool := os.Getenv("GOOO_LOCAL_GO")
	check(goTool != "", "set GOOO_LOCAL_GO")
	records := []obj{}
	predictions, passed, native := 0, 0, 0
	for _, lang := range []string{"ko", "en"} {
		for _, mode := range []string{"model", "deterministic"} {
			condition := lang + "-" + mode
			activity := "AssembleKorean"
			if lang == "en" {
				activity = "AssembleEnglish"
			}
			dest := filepath.Join(out, condition)
			args := []string{"body-path-run", "--source", filepath.Join(inputs, lang+"-source.gooo"), "--activity", activity, "--path-plan", filepath.Join(inputs, lang+"-recipe.json"), "--cases", filepath.Join(inputs, "cases-128.json"), "--go-bin", goTool, "--out", dest, "--repeat", "2", "--timing"}
			if mode == "model" {
				args = append(args, "--model", model)
			}
			run(compiler, args, out, condition)
			run(compiler, []string{"body-path-run", "--verify-timing", "--out", dest}, out, condition+"-verify")
			for i := 1; i <= 2; i++ {
				stem := fmt.Sprintf("run-%d", i)
				generation := decode(filepath.Join(dest, stem+"-generation.json"))
				runtime := decode(filepath.Join(dest, stem+"-runtime.json"))
				timing := decode(filepath.Join(dest, stem+"-timing.json"))
				obs := object(runtime, "observation")
				check(timing["producer_source_sha"] == source && timing["producer_modified"] == "false", "clean producer binding")
				check(object(timing, "wall")["status"] == "OBSERVED", "incomplete phase record")
				check(obs["projection_replayed"] == true && obs["runtime_replayed"] == true, "missing source/native replay")
				check(object(obs, "artifact")["reused"] == (i == 2), "artifact reuse")
				check(object(runtime, "completeness_receipt")["profile_id"] == "gooo/typed-path-runtime-v3", "profile changed")
				check(bytes.Equal(raw(filepath.Join(dest, stem+"-generated.go")), raw(filepath.Join(expected, condition, "run-1-generated.go"))), "generated bytes changed")
				selection := object(object(object(generation, "report"), "body_paths"), "search")
				n := object(selection, "selection")["local_model_predictions"].(float64)
				want := float64(0)
				if mode == "model" {
					want = 1
				}
				check(n == want, "model accounting")
				predictions += int(n)
				for _, value := range obs["cases"].([]any) {
					check(value.(obj)["passed"] == true, "unmet finite expectation")
					passed++
				}
				for _, value := range obs["runs"].([]any) {
					p := value.(obj)
					check(p["started"] == true && p["completed"] == true, "native failure")
					native++
				}
				records = append(records, obj{"condition": condition, "sequence": i, "response_ms": timing["response_ns"].(float64) / 1e6, "source": source})
			}
		}
	}
	check(predictions == 4 && passed == 1024 && native == 16, "unexpected denominators")
	b, e := json.MarshalIndent(obj{"status": "PASS", "requests": 8, "actual_model_predictions": predictions, "native_runs": native, "finite_passed": passed, "finite_total": 1024, "new_intent_tasks": 0, "training_updates": 0, "compiler_sha256": fmt.Sprintf("%x", sha256.Sum256(raw(compiler))), "records": records}, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(out, "smoke-summary.json"), append(b, '\n'), 0600))
	missingTool(compiler, inputs, expected, out, source)
}
