// A deliberately small paired observation of the compiler's oracle loop.
// Run from the research repository root with an explicitly built clean compiler.
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
	ID         string  `json:"id"`
	Language   string  `json:"language"`
	Model      bool    `json:"model"`
	Mode       string  `json:"mode"`
	Generation process `json:"generation"`
	Runtime    process `json:"runtime"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func writeJSON(name string, value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	must(err)
	must(os.WriteFile(name, append(b, '\n'), 0644))
}
func sha(name string) string {
	b, err := os.ReadFile(name)
	must(err)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func run(binary, out string, args ...string) process {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	start := time.Now()
	err := c.Run()
	wall := time.Since(start)
	must(os.WriteFile(out, stdout.Bytes(), 0644))
	if err != nil {
		must(os.WriteFile(out+".stderr", stderr.Bytes(), 0644))
		panic(fmt.Sprintf("%s: %v: %s", out, err, stderr.String()))
	}
	p := process{WallMS: float64(wall) / 1e6, CPUMS: float64(c.ProcessState.UserTime()+c.ProcessState.SystemTime()) / 1e6}
	p.CPUOneCorePercent = 100 * p.CPUMS / p.WallMS
	if usage, ok := c.ProcessState.SysUsage().(*syscall.Rusage); ok {
		p.PeakRSSBytes = usage.Maxrss
		if runtime.GOOS == "linux" {
			p.PeakRSSBytes *= 1024
		}
	}
	return p
}

func fixtures(dir string) {
	source := "package observation\nnamespace observation\nentity Integer id \"observation://integer\"\n" +
		"activity Assemble(Integer) -> Integer computes \"let a = (input - 1); let b = (a - 3); return (b - 9)\"\n" +
		"activity Expected(Integer) -> Integer computes \"return (7 - input)\"\n" +
		"activity Unrelated(Integer) -> Integer computes \"return input\"\n"
	must(os.WriteFile(filepath.Join(dir, "source.gooo"), []byte(source), 0644))
	base := `{"schema":"gooo/body-codegen-typed-path-plan/v1","path_plan":{"schema":"gooo/typed-body-path-plan/v1","base":{"schema":"gooo/typed-body-plan/v1","id":"observation://assembly","name":"Assemble","result_type":"Int","expressions":[{"kind":"input","name":"input"},{"kind":"int","int":1},{"kind":"binary","operation":"subtract","left":0,"right":1},{"kind":"local","name":"a"},{"kind":"int","int":3},{"kind":"binary","operation":"subtract","left":3,"right":4},{"kind":"local","name":"b"},{"kind":"int","int":9},{"kind":"binary","operation":"subtract","left":6,"right":7}],"statements":[{"kind":"let","name":"a","expr":2},{"kind":"let","name":"b","expr":5},{"kind":"return","expr":8}],"root":[0,1,2]},"decisions":[]},"test_cases":[{"input":10,"expected":-3}],"max_attempts":8}`
	for _, lang := range []string{"ko", "en"} {
		var doc map[string]any
		must(json.Unmarshal([]byte(base), &doc))
		choices := make([]any, 3)
		for i := 0; i < 3; i++ {
			intent := fmt.Sprintf("Reverse the operand order in step %d.", i+1)
			if lang == "ko" {
				intent = fmt.Sprintf("%d번째 단계의 피연산자 순서를 반대로 배치한다.", i+1)
			}
			choices[i] = map[string]any{"id": fmt.Sprintf("order_%d", i), "kind": "operand_order", "target": 2 + 3*i, "intent": intent,
				"fallback": "layout_forward", "options": []any{map[string]any{"label": "layout_forward"}, map[string]any{"label": "layout_reverse", "reverse": true}}}
		}
		doc["path_plan"].(map[string]any)["decisions"] = choices
		writeJSON(filepath.Join(dir, "plan-"+lang+".json"), doc)
	}
	for _, mode := range []string{"rank_only", "oracle"} {
		request := map[string]any{"schema": "gooo/path-observation-request/v1", "inputs": []int{0, 3, -1}, "max_candidates": 8, "max_rounds": 2}
		if mode == "oracle" {
			request["oracle_activity"] = "Expected"
		}
		writeJSON(filepath.Join(dir, mode+".json"), request)
	}
	var cases []any
	for _, input := range []int{-8, -1, 0, 3, 10, 21} {
		cases = append(cases, map[string]int{"input": input, "expected": 7 - input})
	}
	writeJSON(filepath.Join(dir, "runtime-cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": cases})
}

func main() {
	compiler := flag.String("compiler", "", "absolute path to a clean compiler binary")
	compilerSHA := flag.String("compiler-sha", "", "full compiler commit")
	goBin := flag.String("go-bin", "go", "Go 1.27.1 executable")
	out := flag.String("out", "publication/path-observation-loop-20261003", "new result directory")
	flag.Parse()
	if *compiler == "" || len(*compilerSHA) != 40 {
		panic("explicit compiler and commit required")
	}
	if _, err := os.Stat(*out); !os.IsNotExist(err) {
		panic("output must be a new directory; previous observations are immutable")
	}
	must(os.MkdirAll(*out, 0755))
	fixtures(*out)
	var rows []row
	started := time.Now()
	for repeat := 0; repeat < 2; repeat++ {
		for _, lang := range []string{"ko", "en"} {
			modes := []string{"none", "rank_only", "oracle"}
			if repeat == 1 {
				modes = []string{"oracle", "rank_only", "none"}
			}
			for _, mode := range modes {
				models := []bool{false, true}
				if repeat == 1 {
					models = []bool{true, false}
				}
				for _, useModel := range models {
					id := fmt.Sprintf("%s-%s-model%t-r%d", lang, mode, useModel, repeat)
					plan := filepath.Join(*out, "plan-"+lang+".json")
					source := filepath.Join(*out, "source.gooo")
					gen := filepath.Join(*out, id+"-generation.json")
					native := filepath.Join(*out, id+"-runtime.json")
					args := []string{"body-codegen", "--json", "--activity", "Assemble", "--path-plan", plan}
					if mode != "none" {
						args = append(args, "--path-observation", filepath.Join(*out, mode+".json"))
					}
					if useModel {
						args = append(args, "--path-model", model, "--path-step-attempts", "1", "--path-feedback-rounds", "7")
					}
					args = append(args, source)
					r := row{ID: id, Language: lang, Model: useModel, Mode: mode, Generation: run(*compiler, gen, args...)}
					r.Runtime = run(*compiler, native, "body-execute", "--source", source, "--path-plan", plan, "--generation", gen,
						"--cases", filepath.Join(*out, "runtime-cases.json"), "--go-bin", *goBin)
					rows = append(rows, r)
					writeJSON(filepath.Join(*out, "processes.json"), rows)
					fmt.Println(id)
				}
			}
		}
	}
	writeJSON(filepath.Join(*out, "manifest.json"), map[string]any{"schema": "gooo/path-observation-pilot/v1", "compiler_sha": *compilerSHA,
		"compiler_binary_sha256": sha(*compiler), "model_metadata_sha256": sha(model), "model_weights_sha256": sha(filepath.Join(filepath.Dir(model), "weights.bin")),
		"os": runtime.GOOS, "arch": runtime.GOARCH, "go_version": runtime.Version(), "elapsed_ms": float64(time.Since(started)) / 1e6,
		"generations": len(rows), "compiled_runs": 2 * len(rows), "training_updates": 0,
		"scope": "One authored three-choice task, two language views, two repetitions, model on/off and no probes/rank-only/oracle. Finite mechanism pilot, not unseen-task accuracy or speedup proof."})
}
