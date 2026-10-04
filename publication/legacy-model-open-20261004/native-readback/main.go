// Recalculate saved native observations without compilation or inference.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type testCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
	Passed   bool  `json:"passed"`
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func check(ok bool, s string) {
	if !ok {
		panic(s)
	}
}
func raw(p string) []byte    { b, e := os.ReadFile(p); must(e); return b }
func decode(p string, v any) { must(json.Unmarshal(raw(p), v)) }
func main() {
	check(len(os.Args) == 2, "usage native-readback saved-root")
	root := os.Args[1]
	var input struct {
		Cases []testCase `json:"cases"`
	}
	decode(filepath.Join(root, "inputs", "cases-128.json"), &input)
	check(len(input.Cases) == 128, "case count")
	var rows []map[string]any
	for _, arm := range []string{"candidate-native", "candidate-native-isolated", "installed-native-same-window"} {
		source := "2420ad198480ca4592306df7db630a82de6d9ed4"
		if arm == "installed-native-same-window" {
			source = "d1bfd273ab4e21d0191548b066a27bcb77d7ed86"
		}
		predictions, native, passed := 0, 0, 0
		for _, language := range []string{"ko", "en"} {
			for _, mode := range []string{"model", "deterministic"} {
				condition := language + "-" + mode
				dir := filepath.Join(root, arm, condition)
				for sequence := 1; sequence <= 2; sequence++ {
					stem := fmt.Sprintf("run-%d", sequence)
					var g struct {
						Report struct {
							BodyPaths struct {
								Search struct {
									Selection struct {
										Predictions int  `json:"local_model_predictions"`
										External    int  `json:"external_provider_calls"`
										Known       bool `json:"external_provider_calls_known"`
									} `json:"selection"`
								} `json:"search"`
							} `json:"body_paths"`
						} `json:"report"`
					}
					decode(filepath.Join(dir, stem+"-generation.json"), &g)
					want := 0
					if mode == "model" {
						want = 1
					}
					s := g.Report.BodyPaths.Search.Selection
					check(s.Predictions == want && s.External == 0 && s.Known, "model accounting")
					predictions += s.Predictions
					var r struct {
						Observation struct {
							Source     string     `json:"producer_source_sha"`
							Projection bool       `json:"projection_replayed"`
							Runtime    bool       `json:"runtime_replayed"`
							Cases      []testCase `json:"cases"`
							Runs       []struct {
								Started   bool `json:"started"`
								Completed bool `json:"completed"`
								Exit      int  `json:"exit_code"`
							} `json:"runs"`
						} `json:"observation"`
					}
					decode(filepath.Join(dir, stem+"-runtime.json"), &r)
					check(r.Observation.Source == source && r.Observation.Projection && r.Observation.Runtime, "source/replay")
					check(len(r.Observation.Cases) == 128 && len(r.Observation.Runs) == 2, "native extent")
					for i, c := range r.Observation.Cases {
						check(c.Input == input.Cases[i].Input && c.Expected == input.Cases[i].Expected && c.Actual == c.Expected && c.Passed, "case comparison")
						passed++
					}
					for _, run := range r.Observation.Runs {
						check(run.Started && run.Completed && run.Exit == 0, "native completion")
						native++
					}
					check(bytes.Equal(raw(filepath.Join(dir, stem+"-generated.go")), raw(filepath.Join(root, "expected", condition, "run-1-generated.go"))), "frozen generated Go")
					var timing struct {
						Source   string `json:"producer_source_sha"`
						Modified string `json:"producer_modified"`
						Status   string `json:"status"`
						Response int64  `json:"response_ns"`
						Wall     struct {
							Status string `json:"status"`
						} `json:"wall"`
					}
					decode(filepath.Join(dir, stem+"-timing.json"), &timing)
					check(timing.Source == source && timing.Modified == "false" && timing.Wall.Status == "OBSERVED" && timing.Status == "completed", "timing source/status")
					rows = append(rows, map[string]any{"arm": arm, "condition": condition, "sequence": sequence, "response_ns": timing.Response})
				}
			}
		}
		check(predictions == 4 && native == 16 && passed == 1024, "arm aggregate")
		missing := filepath.Join(root, arm, "missing-tool")
		var t struct {
			Source   string `json:"producer_source_sha"`
			Modified string `json:"producer_modified"`
			Status   string `json:"status"`
		}
		decode(filepath.Join(missing, "run-1-timing.json"), &t)
		check(t.Source == source && t.Modified == "false" && t.Status == "execution_failed", "missing tool retained")
		check(strings.Contains(string(raw(filepath.Join(root, arm, "missing-tool.stderr"))), "native unobserved"), "missing tool display")
	}
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "historical_constructions": 24, "historical_model_predictions": 12, "historical_native_runs": 48, "historical_finite_passed": 3072, "historical_finite_total": 3072, "original_missing_tool_controls": 3, "records": rows, "reader_model_predictions": 0, "reader_native_runs": 0, "new_intent_tasks": 0, "training_updates": 0}))
}
