package main

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
)

const source = "2420ad198480ca4592306df7db630a82de6d9ed4"

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}
func raw(path string) []byte { b, err := os.ReadFile(path); must(err); return b }

type request struct {
	Correlation string `json:"correlation_id"`
	Cases       struct {
		Cases []struct{ Input, Expected int64 }
	} `json:"execution_cases"`
}
type result struct {
	Sequence    int
	Status      string
	Correlation string `json:"correlation_id"`
	Response    struct {
		Source string
		Report struct {
			Source string `json:"compiler_source_sha"`
			Paths  struct {
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
type summary struct {
	Status, Source string
	Rows           []struct {
		Mode        string
		Wall        int64   `json:"total_process_ns"`
		User        int64   `json:"cpu_user_ns"`
		System      int64   `json:"cpu_system_ns"`
		CPU         float64 `json:"average_cpu_percent_one_core_basis"`
		Joined      bool    `json:"process_joined"`
		Timeout     bool
		Requests    int
		Workers     int
		Predictions int    `json:"actual_model_predictions"`
		Native      int    `json:"native_runs"`
		Passed      int    `json:"finite_passed"`
		Total       int    `json:"finite_total"`
		HostCPU     string `json:"host_cpu_utilization"`
		ModelRAM    string `json:"model_only_ram"`
	}
	Requests    int
	Predictions int `json:"actual_model_predictions"`
	Native      int `json:"native_runs"`
	Passed      int `json:"finite_passed"`
	Total       int `json:"finite_total"`
	NewTasks    int `json:"new_intent_tasks"`
	Updates     int `json:"training_updates"`
}

func main() {
	check(len(os.Args) == 3, "usage worker-saved-root candidate-saved-root")
	root := filepath.Join(os.Args[1], "candidate-shared-worker-observations")
	var requests []request
	d := json.NewDecoder(bytes.NewReader(raw(filepath.Join(root, "requests.jsonl"))))
	for {
		var r request
		err := d.Decode(&r)
		if err == io.EOF {
			break
		}
		must(err)
		check(len(r.Cases.Cases) == 128, "request case scope")
		requests = append(requests, r)
	}
	check(len(requests) == 2 && requests[0].Correlation == "worker-ko" && requests[1].Correlation == "worker-en", "request identity")
	var s summary
	must(json.Unmarshal(raw(filepath.Join(root, "summary.json")), &s))
	check(s.Status == "PASS" && s.Source == source && len(s.Rows) == 2 && s.NewTasks == 0 && s.Updates == 0, "summary scope")
	passed, native, predictions := 0, 0, 0
	for i, mode := range []string{"model", "deterministic"} {
		row := s.Rows[i]
		check(row.Mode == mode && row.Joined && !row.Timeout && row.Requests == 2 && row.Workers == 2, "process accounting")
		check(row.Wall > 0 && row.Wall <= 90e9 && row.User >= 0 && row.User <= 180e9 && row.System >= 0 && row.System <= 180e9, "CPU scope")
		check(row.HostCPU == "UNOBSERVED" && row.ModelRAM == "UNOBSERVED", "resource scope")
		check(math.Abs(row.CPU-100*float64(row.User+row.System)/float64(row.Wall)) < 1e-9, "CPU arithmetic")
		modePassed, modeNative, modePredictions := 0, 0, 0
		seen := map[int]bool{}
		d := json.NewDecoder(bytes.NewReader(raw(filepath.Join(root, mode+".stdout.jsonl"))))
		for {
			var r result
			err := d.Decode(&r)
			if err == io.EOF {
				break
			}
			must(err)
			check(r.Sequence >= 1 && r.Sequence <= 2 && !seen[r.Sequence] && r.Status == "completed", "result identity")
			seen[r.Sequence] = true
			request := requests[r.Sequence-1]
			check(r.Correlation == request.Correlation && r.Response.Report.Source == source, "result source")
			lang := []string{"ko", "en"}[r.Sequence-1]
			frozen := filepath.Join(os.Args[2], "expected", lang+"-"+mode, "run-1-generated.go")
			check(bytes.Equal([]byte(r.Response.Source), raw(frozen)), "generated Go differs")
			selection := r.Response.Report.Paths.Search.Selection
			want := 0
			if mode == "model" {
				want = 1
			}
			check(selection.Local != nil && *selection.Local == want && selection.External != nil && *selection.External == 0, "model calls")
			modePredictions += *selection.Local
			o := r.Execution.Observation
			check(o.Declared == 128 && len(o.Cases) == 128 && o.Replayed && o.Projection && len(o.Runs) == 2, "execution scope")
			for j, c := range o.Cases {
				check(c.Input == request.Cases.Cases[j].Input && c.Expected == request.Cases.Cases[j].Expected && c.Actual == c.Expected && c.Passed, "finite output")
				modePassed++
			}
			for _, run := range o.Runs {
				check(run.Started && run.Completed, "native run incomplete")
				modeNative++
			}
		}
		check(len(seen) == 2 && modePassed == row.Passed && modePassed == row.Total && modeNative == row.Native && modePredictions == row.Predictions, "row totals")
		passed += modePassed
		native += modeNative
		predictions += modePredictions
	}
	check(s.Requests == 4 && passed == 512 && passed == s.Passed && passed == s.Total && native == 8 && native == s.Native && predictions == 2 && predictions == s.Predictions, "summary totals")
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "scope": "saved worker outputs, int64 finite cases, frozen Go, original process CPU arithmetic", "original_source": source, "original_cases": passed, "original_native_runs": native, "original_model_predictions": predictions, "reader_model_predictions": 0, "reader_native_executions": 0}))
}
