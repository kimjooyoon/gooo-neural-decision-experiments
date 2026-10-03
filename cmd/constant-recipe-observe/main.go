// Observe declared Gooo recipes with and without the frozen own model.
// The original constant-body profile remains the default.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const model = "publication/full-input-separate-arithmetic-20261003/models/compact/bag-original/fp32/model.json"

type cost struct {
	WallMS     float64 `json:"wall_ms"`
	CPUMS      float64 `json:"cpu_ms"`
	CPUPercent float64 `json:"cpu_one_core_percent"`
	RSS        int64   `json:"peak_rss_bytes"`
}
type record struct {
	ID         string `json:"id"`
	Model      bool   `json:"model_requested"`
	Generation cost   `json:"generation"`
	Runtime    cost   `json:"runtime"`
	Calls      int    `json:"model_predictions"`
	Passed     int    `json:"passed"`
	Total      int    `json:"total"`
	EmittedSHA string `json:"emitted_sha256"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); must(e); return b }
func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func save(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(p, append(b, '\n'), 0644))
}
func run(bin, out string, args ...string) cost {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	start := time.Now()
	err := c.Run()
	w := time.Since(start)
	must(os.WriteFile(out, stdout.Bytes(), 0644))
	if err != nil {
		must(os.WriteFile(out+".stderr", stderr.Bytes(), 0644))
		panic(fmt.Sprintf("child failed: %v: %s", err, stderr.String()))
	}
	r := cost{WallMS: float64(w) / 1e6, CPUMS: float64(c.ProcessState.UserTime()+c.ProcessState.SystemTime()) / 1e6}
	r.CPUPercent = 100 * r.CPUMS / r.WallMS
	if u, ok := c.ProcessState.SysUsage().(*syscall.Rusage); ok {
		r.RSS = u.Maxrss
		if runtime.GOOS == "linux" {
			r.RSS *= 1024
		}
	}
	return r
}
func fixtures(out string) {
	source := "package constants\nnamespace constants\nentity Integer id \"constants://integer\"\nactivity Compose(Integer) -> Integer computes \"let first = 2 - 3; let second = first - 5; return second - 7\"\n"
	must(os.WriteFile(filepath.Join(out, "source.gooo"), []byte(source), 0644))
	intents := []string{"첫 계산에서 3에서 2를 뺀다. First subtract two from three.", "다음으로 5에서 첫 결과를 뺀다. Next subtract the first result from five.", "마지막으로 둘째 결과에서 7을 뺀다. Finally subtract seven from the second result."}
	var choices []any
	for i, intent := range intents {
		choices = append(choices, map[string]any{"id": fmt.Sprintf("order-%d", i), "kind": "operand_order", "occurrence": i, "intent": intent})
	}
	save(filepath.Join(out, "recipe.json"), map[string]any{"schema": "gooo/source-typed-path-recipe/v1", "choices": choices, "test_cases": []any{map[string]int64{"input": 0, "expected": -3}}, "max_attempts": 8})
	var cases []any
	for _, n := range []int64{math.MinInt64, -7, -1, 0, 17, math.MaxInt64} {
		cases = append(cases, map[string]int64{"input": n, "expected": -3})
	}
	save(filepath.Join(out, "cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": cases})
}
func observe(compiler, goBin, out, activity string, expectedCases int, useModel bool, repeat int) record {
	id := fmt.Sprintf("model%t-r%d", useModel, repeat)
	source, plan := filepath.Join(out, "source.gooo"), filepath.Join(out, "recipe.json")
	gen, execution := filepath.Join(out, id+"-generation.json"), filepath.Join(out, id+"-runtime.json")
	args := []string{"body-codegen", "--json", "--activity", activity, "--path-plan", plan}
	if useModel {
		args = append(args, "--path-model", model)
	}
	r := record{ID: id, Model: useModel, Generation: run(compiler, gen, append(args, source)...)}
	var g struct {
		Source string `json:"source"`
		Report struct {
			Paths struct {
				Search struct {
					Selection struct {
						Calls int `json:"local_model_predictions"`
					} `json:"selection"`
				} `json:"search"`
			} `json:"body_paths"`
		} `json:"report"`
	}
	must(json.Unmarshal(read(gen), &g))
	r.EmittedSHA, r.Calls = hash([]byte(g.Source)), g.Report.Paths.Search.Selection.Calls
	if useModel && r.Calls != 1 || !useModel && r.Calls != 0 {
		panic("prediction count differs")
	}
	r.Runtime = run(compiler, execution, "body-execute", "--source", source, "--path-plan", plan, "--generation", gen, "--cases", filepath.Join(out, "cases.json"), "--go-bin", goBin)
	var result struct {
		Observation struct {
			Stage string            `json:"stage"`
			Runs  []json.RawMessage `json:"runs"`
			Cases []struct {
				Passed bool `json:"passed"`
			} `json:"cases"`
		} `json:"observation"`
	}
	must(json.Unmarshal(read(execution), &result))
	for _, c := range result.Observation.Cases {
		r.Total++
		if c.Passed {
			r.Passed++
		}
	}
	if result.Observation.Stage != "COMPLETE" || len(result.Observation.Runs) != 2 || r.Total != expectedCases || r.Passed != expectedCases {
		panic("native recipe body did not pass")
	}
	return r
}
func main() {
	compiler := flag.String("compiler", "", "clean compiler executable")
	revision := flag.String("compiler-sha", "", "exact compiler source")
	goBin := flag.String("go-bin", "go", "Go 1.27.1 executable")
	out := flag.String("out", "publication/constant-recipes-20261003", "fresh output directory")
	profile := flag.String("profile", "constant", "constant or condition-chain")
	sourceRoot := flag.String("compiler-source", "", "compiler repository for pinned condition-chain fixtures")
	flag.Parse()
	if flag.NArg() != 0 || *compiler == "" || len(*revision) != 40 || (*profile != "constant" && *profile != "condition-chain") || (*profile == "condition-chain" && *sourceRoot == "") {
		panic("compiler and revision required")
	}
	build, err := exec.Command(*goBin, "version", "-m", *compiler).Output()
	must(err)
	if !bytes.Contains(build, []byte("vcs.revision="+*revision)) || !bytes.Contains(build, []byte("vcs.modified=false")) || !bytes.Contains(build, []byte("v0.2.18-experimental")) {
		panic("compiler identity differs")
	}
	if _, err := os.Stat(*out); !os.IsNotExist(err) {
		panic("fresh output required")
	}
	must(os.MkdirAll(*out, 0755))
	activity, expectedCases := "Compose", 6
	schema := "gooo/constant-recipe-pilot/v1"
	scope := "One authored constant three-choice body, one bilingual intention, three repeats per arm. Finite native cases include both int64 endpoints. Process CPU is one-core-relative, not host CPU increase. No generalization or speed claim."
	if *profile == "condition-chain" {
		conditionFixtures(*sourceRoot, *revision, *out)
		activity, expectedCases, schema = "Clamp", 7, "gooo/condition-chain-recipe-pilot/v1"
		scope = "One authored three-region clamp, one bilingual intention, three repeats per arm. Seven finite native cases include int64 endpoints and range boundaries. Process CPU is one-core-relative, not host CPU increase. No generalization or speed claim."
	} else {
		fixtures(*out)
	}
	var rows []record
	calls, passed, total := 0, 0, 0
	for repeat := range 3 {
		order := []bool{false, true}
		if repeat%2 == 1 {
			order = []bool{true, false}
		}
		for _, use := range order {
			r := observe(*compiler, *goBin, *out, activity, expectedCases, use, repeat)
			rows = append(rows, r)
			calls += r.Calls
			passed += r.Passed
			total += r.Total
			if r.EmittedSHA != rows[0].EmittedSHA {
				panic("emitted recipe body differs")
			}
			save(filepath.Join(*out, "processes.json"), rows)
		}
	}
	save(filepath.Join(*out, "manifest.json"), map[string]any{"schema": schema, "profile": *profile, "compiler_sha": *revision, "compiler_binary_sha256": hash(read(*compiler)), "collector_sha256": hash(read("cmd/constant-recipe-observe/main.go")), "collector_files_sha256": collectorFiles(), "model_metadata_sha256": hash(read(model)), "model_weights_sha256": hash(read(filepath.Join(filepath.Dir(model), "weights.bin"))), "generations": len(rows), "compiled_runs": 2 * len(rows), "model_predictions": calls, "passed": passed, "total": total, "training_updates": 0, "os": runtime.GOOS, "arch": runtime.GOARCH, "go_version": runtime.Version(), "scope": scope})
	fmt.Printf("%d generations, %d compiled runs, %d model predictions, %d/%d finite expectations\n", len(rows), 2*len(rows), calls, passed, total)
}
