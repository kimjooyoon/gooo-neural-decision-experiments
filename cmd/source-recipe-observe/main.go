// Paired source-recipe/full-document observations with an unchanged own model.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"syscall"
	"time"
)

const model = "publication/full-input-separate-arithmetic-20261003/models/compact/bag-original/fp32/model.json"

type process struct {
	WallMS            float64 `json:"wall_ms"`
	CPUMS             float64 `json:"cpu_ms"`
	CPUOneCorePercent float64 `json:"cpu_one_core_percent"`
	PeakRSSBytes      int64   `json:"peak_rss_bytes"`
}
type row struct {
	ID          string  `json:"id"`
	Input       string  `json:"input"`
	Model       bool    `json:"model"`
	Resolve     bool    `json:"resolve"`
	Generation  process `json:"generation"`
	Runtime     process `json:"runtime"`
	ModelCalls  int     `json:"model_calls"`
	Passed      int     `json:"passed"`
	Total       int     `json:"total"`
	DocumentSHA string  `json:"document_sha256"`
	SourceSHA   string  `json:"emitted_sha256"`
}
type generation struct {
	Source string `json:"source"`
	Report struct {
		ActivityID string `json:"activity_id"`
		Paths      struct {
			DocumentSHA string `json:"document_sha256"`
			Search      struct {
				Selection struct {
					Calls int `json:"local_model_predictions"`
				} `json:"selection"`
			} `json:"search"`
		} `json:"body_paths"`
	} `json:"report"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(path string) []byte { b, err := os.ReadFile(path); must(err); return b }
func hash(b []byte) string    { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func write(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	must(err)
	must(os.WriteFile(path, append(b, '\n'), 0644))
}
func compactSize(b []byte) int { var out bytes.Buffer; must(json.Compact(&out, b)); return out.Len() }

func run(binary, output string, args ...string) process {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	start := time.Now()
	err := c.Run()
	wall := time.Since(start)
	must(os.WriteFile(output, stdout.Bytes(), 0644))
	if err != nil {
		must(os.WriteFile(output+".stderr", stderr.Bytes(), 0644))
		panic(fmt.Sprintf("%s: %v: %s", output, err, stderr.String()))
	}
	p := process{WallMS: float64(wall) / 1e6, CPUMS: float64(c.ProcessState.UserTime()+c.ProcessState.SystemTime()) / 1e6}
	p.CPUOneCorePercent = 100 * p.CPUMS / p.WallMS
	if u, ok := c.ProcessState.SysUsage().(*syscall.Rusage); ok {
		p.PeakRSSBytes = u.Maxrss
		if runtime.GOOS == "linux" {
			p.PeakRSSBytes *= 1024
		}
	}
	return p
}

func fixtures(compiler, sourceRoot, out string) {
	for target, origin := range map[string]string{"source.gooo": "path-recipe.gooo.fixture", "recipe.json": "path-recipe.json", "resolve.json": "path-recipe-observation.json"} {
		must(os.WriteFile(filepath.Join(out, target), read(filepath.Join(sourceRoot, "examples/body-codegen", origin)), 0644))
	}
	preserve := filepath.Join(out, "preserve-generation.json")
	run(compiler, preserve, "body-codegen", "--json", "--activity", "Compose", filepath.Join(out, "source.gooo"))
	var original generation
	must(json.Unmarshal(read(preserve), &original))
	var full, recipe map[string]any
	must(json.Unmarshal(read("publication/path-observation-resolution-20261003/plan-ko.json"), &full))
	must(json.Unmarshal(read(filepath.Join(out, "recipe.json")), &recipe))
	base := full["path_plan"].(map[string]any)["base"].(map[string]any)
	base["id"], base["name"] = original.Report.ActivityID, "Compose"
	var decisions []any
	for _, raw := range recipe["choices"].([]any) {
		c := raw.(map[string]any)
		decisions = append(decisions, map[string]any{"id": c["id"], "kind": c["kind"], "intent": c["intent"], "target": 2 + 3*int(c["occurrence"].(float64)), "fallback": "layout_forward",
			"options": []any{map[string]any{"label": "layout_forward"}, map[string]any{"label": "layout_reverse", "reverse": true}}})
	}
	full["path_plan"].(map[string]any)["decisions"] = decisions
	full["test_cases"], full["max_attempts"] = recipe["test_cases"], recipe["max_attempts"]
	write(filepath.Join(out, "full.json"), full)
	var cases []any
	for _, n := range []int64{-8, -1, 0, 3, 10, 21} {
		cases = append(cases, map[string]int64{"input": n, "expected": 7 - n})
	}
	write(filepath.Join(out, "cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": cases})
}

func observe(compiler, goBin, out, kind string, useModel, resolve bool, repeat int) row {
	id := fmt.Sprintf("%s-model%t-resolve%t-r%d", kind, useModel, resolve, repeat)
	plan, source := filepath.Join(out, kind+".json"), filepath.Join(out, "source.gooo")
	gen, execution := filepath.Join(out, id+"-generation.json"), filepath.Join(out, id+"-runtime.json")
	args := []string{"body-codegen", "--json", "--activity", "Compose", "--path-plan", plan}
	if useModel {
		args = append(args, "--path-model", model)
	}
	if resolve {
		args = append(args, "--path-observation", filepath.Join(out, "resolve.json"))
	}
	args = append(args, source)
	r := row{ID: id, Input: kind, Model: useModel, Resolve: resolve, Generation: run(compiler, gen, args...)}
	var g generation
	must(json.Unmarshal(read(gen), &g))
	r.DocumentSHA, r.SourceSHA, r.ModelCalls = g.Report.Paths.DocumentSHA, hash([]byte(g.Source)), g.Report.Paths.Search.Selection.Calls
	r.Runtime = run(compiler, execution, "body-execute", "--source", source, "--path-plan", plan, "--generation", gen, "--cases", filepath.Join(out, "cases.json"), "--go-bin", goBin)
	var result struct {
		Observation struct {
			Stage string `json:"stage"`
			Cases []struct {
				Passed bool `json:"passed"`
			} `json:"cases"`
			Runs []json.RawMessage `json:"runs"`
		} `json:"observation"`
	}
	must(json.Unmarshal(read(execution), &result))
	if result.Observation.Stage != "COMPLETE" || len(result.Observation.Runs) != 2 {
		panic("native execution incomplete")
	}
	for _, c := range result.Observation.Cases {
		r.Total++
		if c.Passed {
			r.Passed++
		}
	}
	if r.Total != 6 || resolve && (r.ModelCalls != 0 || r.Passed != 6) {
		panic("finite resolution contract failed")
	}
	if useModel && !resolve && r.ModelCalls != 3 {
		panic("own-model predictions not observed")
	}
	return r
}

func main() {
	compiler := flag.String("compiler", "", "clean compiler executable")
	compilerSHA := flag.String("compiler-sha", "", "exact compiler revision")
	sourceRoot := flag.String("compiler-source", "", "compiler source for fixtures")
	goBin := flag.String("go-bin", "go", "Go 1.27.1 executable")
	out := flag.String("out", "publication/source-recipes-20261003", "new output directory")
	flag.Parse()
	if *compiler == "" || len(*compilerSHA) != 40 || *sourceRoot == "" {
		panic("explicit source and clean compiler required")
	}
	build, err := exec.Command(*goBin, "version", "-m", *compiler).Output()
	must(err)
	if !bytes.Contains(build, []byte("vcs.revision="+*compilerSHA)) || !bytes.Contains(build, []byte("vcs.modified=false")) {
		panic("compiler build identity mismatch")
	}
	if _, err := os.Stat(*out); !os.IsNotExist(err) {
		panic("output already exists")
	}
	must(os.MkdirAll(*out, 0755))
	fixtures(*compiler, *sourceRoot, *out)
	var rows []row
	pairs, calls, passed, total := 0, 0, 0, 0
	for repeat := range 3 {
		for _, useModel := range []bool{false, true} {
			for _, resolve := range []bool{false, true} {
				kinds := []string{"recipe", "full"}
				if repeat%2 == 1 {
					slices.Reverse(kinds)
				}
				var pair []row
				for _, kind := range kinds {
					r := observe(*compiler, *goBin, *out, kind, useModel, resolve, repeat)
					rows = append(rows, r)
					pair = append(pair, r)
					calls += r.ModelCalls
					passed += r.Passed
					total += r.Total
					write(filepath.Join(*out, "processes.json"), rows)
					fmt.Println(r.ID)
				}
				a, b := pair[0], pair[1]
				if a.DocumentSHA != b.DocumentSHA || a.SourceSHA != b.SourceSHA || a.Passed != b.Passed || a.ModelCalls != b.ModelCalls {
					panic("recipe/full observation diverged")
				}
				pairs++
			}
		}
	}
	write(filepath.Join(*out, "manifest.json"), map[string]any{"schema": "gooo/source-recipe-pilot/v1", "compiler_sha": *compilerSHA,
		"compiler_binary_sha256": hash(read(*compiler)), "source_sha256": hash(read(filepath.Join(*out, "source.gooo"))),
		"model_metadata_sha256": hash(read(model)), "model_weights_sha256": hash(read(filepath.Join(filepath.Dir(model), "weights.bin"))),
		"recipe_compact_bytes": compactSize(read(filepath.Join(*out, "recipe.json"))), "full_compact_bytes": compactSize(read(filepath.Join(*out, "full.json"))),
		"paired_equal_documents_and_emissions": pairs, "generations": len(rows), "compiled_runs": 2 * len(rows), "model_predictions": calls, "passed": passed, "total": total,
		"os": runtime.GOOS, "arch": runtime.GOARCH, "go_version": runtime.Version(), "training_updates": 0,
		"scope": "One authored three-subtraction task, one bilingual intent view, three repetitions, paired recipe/full documents, model requested/disconnected and search/direct resolution. Process measurements include recipe expansion. CPU percent is one-core process time divided by elapsed time; whole-host CPU unmeasured. Finite mechanism observation; no task-generalization or causal speed claim."})
}
