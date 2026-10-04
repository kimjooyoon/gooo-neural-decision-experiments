// Read only saved observations. Receipt counts are historical work, not calls
// made by this reader. Int64 comparisons preserve integer endpoint cases.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type totals struct {
	Predictions int `json:"actual_model_predictions"`
	Native      int `json:"native_runs"`
	Passed      int `json:"finite_passed"`
	Total       int `json:"finite_total"`
}
type row struct {
	Condition     string `json:"condition"`
	Attempt       int    `json:"attempt"`
	Joined        bool   `json:"process_joined"`
	SwapperJoined bool   `json:"swapper_joined"`
	Timeout       bool   `json:"timeout"`
	Released      bool   `json:"writer_released_fifo_reader"`
	Wait          bool   `json:"confirmed_wait_before_validation"`
	Absent        bool   `json:"output_directory_absent"`
	Exit          int    `json:"exit_code"`
	StdoutBytes   int    `json:"stdout_bytes"`
	Stderr        string `json:"stderr"`
	Elapsed       int64  `json:"elapsed_ns"`
	Observed      totals `json:"observed"`
}
type caseValue struct {
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
func observe(dir string, cases []caseValue, expectedGo []byte, source string) totals {
	var g struct {
		Report struct {
			BodyPaths struct {
				Search struct {
					Selection struct {
						Predictions int `json:"local_model_predictions"`
					} `json:"selection"`
				} `json:"search"`
			} `json:"body_paths"`
		} `json:"report"`
	}
	decode(filepath.Join(dir, "run-1-generation.json"), &g)
	var r struct {
		Observation struct {
			Source string      `json:"producer_source_sha"`
			Cases  []caseValue `json:"cases"`
			Runs   []struct {
				Started   bool `json:"started"`
				Completed bool `json:"completed"`
				Exit      int  `json:"exit_code"`
			} `json:"runs"`
			Projection bool `json:"projection_replayed"`
			Runtime    bool `json:"runtime_replayed"`
		} `json:"observation"`
	}
	decode(filepath.Join(dir, "run-1-runtime.json"), &r)
	check(r.Observation.Source == source, "runtime source binding")
	check(r.Observation.Projection && r.Observation.Runtime, "replay missing")
	check(len(r.Observation.Cases) == len(cases), "case extent")
	result := totals{Predictions: g.Report.BodyPaths.Search.Selection.Predictions, Total: len(cases)}
	for i, c := range r.Observation.Cases {
		check(c.Input == cases[i].Input && c.Expected == cases[i].Expected && c.Actual == cases[i].Expected && c.Passed, "finite expectation")
		result.Passed++
	}
	for _, run := range r.Observation.Runs {
		check(run.Started && run.Completed && run.Exit == 0, "native run")
		result.Native++
	}
	check(result.Predictions == 1 && result.Native == 2, "actual work counts")
	check(bytes.Equal(raw(filepath.Join(dir, "run-1-generated.go")), expectedGo), "frozen Go changed")
	return result
}
func main() {
	check(len(os.Args) == 2 || len(os.Args) == 3, "usage readback saved-root [baseline|candidate]")
	root := os.Args[1]
	variant := "baseline"
	if len(os.Args) == 3 {
		variant = os.Args[2]
	}
	check(variant == "baseline" || variant == "candidate", "variant")
	name, attempts, wantMetadata, wantRegular, wantWait := "installed-model-swap-v2", 104, 40, 36, 1
	source := "d1bfd273ab4e21d0191548b066a27bcb77d7ed86"
	if variant == "candidate" {
		name, attempts, wantMetadata, wantRegular, wantWait = "candidate-model-swap", 128, 64, 34, 0
		source = "2420ad198480ca4592306df7db630a82de6d9ed4"
	}
	var input struct {
		Schema string      `json:"schema"`
		Cases  []caseValue `json:"cases"`
	}
	decode(filepath.Join(root, "inputs", "cases-128.json"), &input)
	check(input.Schema == "gooo/body-runtime-cases/v1", "input schema")
	cases := input.Cases
	check(len(cases) == 128, "input cases")
	control := filepath.Join(root, "regular-control")
	expectedGo := raw(filepath.Join(control, "run-1-generated.go"))
	controlWork := observe(control, cases, expectedGo, "d1bfd273ab4e21d0191548b066a27bcb77d7ed86")
	var s struct {
		Attempts       int    `json:"attempts"`
		Confirmed      int    `json:"confirmed_fifo_waits"`
		Regular        totals `json:"regular_observations"`
		Rows           []row  `json:"records"`
		ModelRequested bool   `json:"model_requested"`
		NewIntents     int    `json:"new_intent_tasks"`
		Training       int    `json:"training_updates"`
	}
	swap := filepath.Join(root, name)
	decode(filepath.Join(swap, "summary.json"), &s)
	check(s.Attempts == attempts && len(s.Rows) == attempts && s.ModelRequested && s.NewIntents == 0 && s.Training == 0, "declared scope")
	total := totals{}
	waits, metadata, weights, regular := 0, 0, 0, 0
	for _, r := range s.Rows {
		check(r.Joined && r.SwapperJoined && !r.Timeout, "owned process not joined")
		if r.Condition == "metadata" {
			metadata++
		} else {
			check(r.Condition == "weights", "condition")
			weights++
		}
		label := fmt.Sprintf("%s-%02d", r.Condition, r.Attempt)
		var saved row
		decode(filepath.Join(swap, label+"-process.json"), &saved)
		check(saved == r, "process record differs")
		check(string(raw(filepath.Join(swap, label+".stderr"))) == r.Stderr, "stderr differs")
		check(len(raw(filepath.Join(swap, label+".stdout"))) == r.StdoutBytes, "stdout extent")
		if r.Wait {
			check(variant == "baseline" && r.Condition == "metadata" && r.Attempt == 40 && r.Released && r.Exit == 1 && r.Absent && r.StdoutBytes == 0 && r.Elapsed >= 500000000, "wait not supported")
			waits++
		} else {
			check(!r.Released, "unclassified writer release")
		}
		if r.Exit == 0 {
			check(!r.Absent, "successful output absent")
			w := observe(filepath.Join(swap, label+"-results"), cases, expectedGo, source)
			check(w == r.Observed, "observed work differs")
			total.Predictions += w.Predictions
			total.Native += w.Native
			total.Passed += w.Passed
			total.Total += w.Total
			regular++
		} else {
			check(r.Exit == 1 && r.Absent && r.StdoutBytes == 0 && r.Observed == totals{}, "setup failure produced work")
		}
	}
	check(metadata == wantMetadata && weights == 64 && waits == wantWait && s.Confirmed == waits && regular == wantRegular && total == s.Regular, "declared aggregate")
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "variant": variant, "historical_attempts": attempts, "confirmed_fifo_waits": waits, "regular_completions": regular, "historical_observed_work": total, "separate_regular_control": controlWork, "reader_model_predictions": 0, "reader_native_runs": 0, "new_intent_tasks": 0, "training_updates": 0}))
}
