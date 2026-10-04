package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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
func raw(p string) []byte  { b, err := os.ReadFile(p); must(err); return b }
func read(p string, v any) { must(json.Unmarshal(raw(p), v)) }

func main() {
	check(len(os.Args) == 2, "usage saved-root")
	root := os.Args[1]
	passed, native, predictions := 0, 0, 0
	var orderGo []byte
	var rows []map[string]any
	for _, dir := range []string{"direct-order-example-model", "direct-order-example-deterministic", "hf-downloaded-compound-example"} {
		base := filepath.Join(root, dir)
		var cases struct {
			Cases []struct{ Input, Expected int64 }
		}
		read(filepath.Join(base, "cases.json"), &cases)
		count, want := 8, 1
		if dir == "direct-order-example-deterministic" {
			want = 0
		}
		if dir == "hf-downloaded-compound-example" {
			count = 3
		}
		check(len(cases.Cases) == count, "case denominator")
		var summary struct {
			Schema string
			Rows   []struct {
				Status        string
				Passed, Total int
				Native        int     `json:"native_runs"`
				Reused        bool    `json:"artifact_reused"`
				Response      float64 `json:"response_ms"`
			}
		}
		read(filepath.Join(base, "summary.json"), &summary)
		check(summary.Schema == "gooo/body-path-file-run/v1" && len(summary.Rows) == 2, "summary scope")
		var firstGo []byte
		for n := 1; n <= 2; n++ {
			prefix := filepath.Join(base, fmt.Sprintf("run-%d-", n))
			var g struct {
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
			read(prefix+"generation.json", &g)
			check(g.Report.Source == "2420ad198480ca4592306df7db630a82de6d9ed4", "source")
			calls := g.Report.Paths.Search.Selection
			check(calls.Local != nil && *calls.Local == want && calls.External != nil && *calls.External == 0, "model calls")
			predictions += *calls.Local
			goBytes := raw(prefix + "generated.go")
			check(bytes.Equal(goBytes, []byte(g.Source)), "saved Go binding")
			if n == 1 {
				firstGo = goBytes
			} else {
				check(bytes.Equal(goBytes, firstGo), "repeat Go differs")
			}
			if count == 8 {
				if orderGo == nil {
					orderGo = goBytes
				} else {
					check(bytes.Equal(goBytes, orderGo), "model/deterministic Go differs")
				}
			}
			var runtime struct {
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
			read(prefix+"runtime.json", &runtime)
			o := runtime.Observation
			check(o.Declared == count && len(o.Cases) == count && o.Replayed && o.Projection && len(o.Runs) == 2, "execution scope")
			for i, c := range o.Cases {
				check(c.Input == cases.Cases[i].Input && c.Expected == cases.Cases[i].Expected && c.Actual == c.Expected && c.Passed, "actual int64 output")
				passed++
			}
			for _, run := range o.Runs {
				check(run.Started && run.Completed, "native execution")
				native++
			}
			var timing struct {
				Source   string `json:"producer_source_sha"`
				Modified string `json:"producer_modified"`
				Status   string
			}
			read(prefix+"timing.json", &timing)
			check(timing.Source == g.Report.Source && timing.Modified == "false" && timing.Status == "completed", "timing producer")
			row := summary.Rows[n-1]
			check(row.Status == "completed" && row.Passed == count && row.Total == count && row.Native == 2 && row.Reused == (n == 2) && row.Response > 0, "summary row")
			rows = append(rows, map[string]any{"arm": dir, "request": n, "original_response_ms": row.Response, "original_cases": count, "original_predictions": want})
		}
	}
	check(passed == 38 && native == 12 && predictions == 4, "aggregate")
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "scope": "existing authored quickstarts; source, saved generated Go and int64 output consistency", "original_constructions": 6, "original_cases": passed, "original_native_runs": native, "original_predictions": predictions, "new_intents": 0, "training_updates": 0, "reader_predictions": 0, "reader_executions": 0, "rows": rows}))
}
