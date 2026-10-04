package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var checkout string

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func require(v bool, s string) {
	if !v {
		panic(s)
	}
}
func object(b []byte) map[string]any { var v map[string]any; must(json.Unmarshal(b, &v)); return v }
func read(p string) []byte           { b, e := os.ReadFile(p); must(e); return b }
func save(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(p, append(b, '\n'), 0600))
}
func command(bin string, args ...string) ([]byte, map[string]any) {
	c := exec.Command(bin, args...)
	c.Dir = checkout
	c.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	started := time.Now()
	b, e := c.Output()
	wall := time.Since(started)
	if e != nil {
		panic(fmt.Sprintf("command failed: %s: %v", b, e))
	}
	usage := c.ProcessState.SysUsage().(*syscall.Rusage)
	cpu := c.ProcessState.UserTime() + c.ProcessState.SystemTime()
	rss := usage.Maxrss
	if runtime.GOOS != "darwin" {
		rss *= 1024
	}
	return b, map[string]any{"wall_ns": wall.Nanoseconds(), "cpu_ns": cpu.Nanoseconds(), "cpu_one_core_percent": float64(cpu) / float64(wall) * 100, "max_single_process_rss_bytes": rss}
}
func main() {

	flag.StringVar(&checkout, "checkout", "", "compiler checkout with the public example")
	compiler := flag.String("compiler", "", "clean Go1.27.1 compiler executable")
	modelFlag := flag.String("model", "", "frozen public QAT model.json")
	output := flag.String("out", "", "new observation directory")
	flag.Parse()
	require(checkout != "" && *compiler != "" && *modelFlag != "" && *output != "" && flag.NArg() == 0, "--checkout --compiler --model --out required")
	var err error
	checkout, err = filepath.Abs(checkout)
	must(err)
	cli, err := filepath.Abs(*compiler)
	must(err)
	model, err := filepath.Abs(*modelFlag)
	must(err)
	dir, err := filepath.Abs(*output)
	must(err)
	must(os.Mkdir(dir, 0700))
	weights := read(filepath.Join(filepath.Dir(model), "weights.bin"))
	weightSHA := fmt.Sprintf("%x", sha256.Sum256(weights))
	require(weightSHA == "049081cc6dc62f3c4230f72e601c1fd421c514fce9cfdf7bc1eb412ca4c5514b", "frozen model weight differs")
	version, _ := command(cli, "version", "--build", "--json")
	build := object(version)
	require(build["source_status"] == "CLEAN_VCS" && build["go_version"] == "go1.27.1", "clean build required")
	save(filepath.Join(dir, "compiler-build.json"), build)
	base := read(filepath.Join(checkout, "examples/body-codegen/record-candidate-continuation.gooo.fixture"))
	var rows []any
	for _, profile := range []string{"deterministic", "qat_ternary"} {
		for _, budget := range []int{4, 8} {
			stem := fmt.Sprintf("%s-b%d", profile, budget)
			source := []byte(strings.Replace(string(base), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1))
			path := filepath.Join(dir, stem+".gooo.fixture")
			must(os.WriteFile(path, source, 0600))
			out := filepath.Join(dir, stem)
			args := []string{"body-compose", "--source", path, "--cases", "examples/body-codegen/record-field-updates-cases.json", "--out", out}
			if profile != "deterministic" {
				args = append(args, "--model", model)
			}
			raw, cost := command(cli, args...)
			v := object(raw)
			must(os.WriteFile(filepath.Join(dir, stem+"-native.json"), raw, 0600))
			a := v["composition"].(map[string]any)["steps"].([]any)[0].(map[string]any)["generation"].(map[string]any)["report"].(map[string]any)["record_assembly"].(map[string]any)
			rejected := 0
			for _, trial := range a["attempts"].([]any) {
				x := trial.(map[string]any)
				if x["status"] == "TYPECHECK_FAILED" {
					rejected++
					require(x["passed"].(float64) == 0 && x["total"].(float64) == 0 && x["fields_total"].(float64) == 0, "rejected candidate acquired test scores")
				}
			}
			calls := 0.
			if profile != "deterministic" {
				calls = 1
			}
			r := v["runtime"].(map[string]any)
			require(v["generated_now"] == true && a["model_calls"].(float64) == calls && r["model_calls"].(float64) == 0 && r["runtime_replayed"] == true, "prediction/native execution differs")
			if budget == 8 {
				require(a["fields_passed"].(float64) == 15 && r["finite_passed"].(float64) == 14, "complete candidate missing")
			}
			replayRaw, replayCost := command(cli, "body-compose", "--source", filepath.Join(out, "original.gooo"), "--cases", filepath.Join(out, "cases.json"), "--composition", filepath.Join(out, "composition.json"))
			must(os.WriteFile(filepath.Join(dir, stem+"-replay.json"), replayRaw, 0600))
			replay := object(replayRaw)
			replayedRuntime := replay["runtime"].(map[string]any)
			require(replay["generated_now"] == false && replayedRuntime["model_calls"].(float64) == 0 &&
				replayedRuntime["finite_passed"] == r["finite_passed"] && replayedRuntime["finite_total"] == r["finite_total"], "saved replay differs")
			rows = append(rows, map[string]any{"profile": profile, "budget": budget, "selected_mask": a["selected_mask"], "status": a["status"],
				"attempted_candidates": len(a["attempts"].([]any)), "type_rejected_candidates": rejected,
				"evaluated_candidates": len(a["attempts"].([]any)) - rejected, "selection_fields_passed": a["fields_passed"], "selection_fields_total": a["fields_total"],
				"native_outputs_passed": r["finite_passed"], "native_outputs_total": r["finite_total"], "generation_model_calls": calls,
				"runtime_model_calls": 0, "replay_generated_now": false, "replay_model_calls": 0, "prediction_ns": a["predict_ns"], "generation_cost": cost, "replay_cost": replayCost})
		}
	}
	save(filepath.Join(dir, "summary.json"), map[string]any{"schema": "gooo/record-candidate-continuation-observation/v1", "compiler_source": build["compiler_source_sha"], "model_weight_sha256": weightSHA,
		"training_updates": 0, "trials": rows, "scope": "one authored unused-local combination; four ordered generation and four saved replay commands; each native response repeats its execution; CPU includes waited descendants, RSS is a maximum single-process observation; host CPU delta and GPU utilization unmeasured"})
	fmt.Println("Four generation and four saved replay commands completed with the unchanged public model.")
}
