// Inspect saved native swap outcomes only; no model or compiler execution.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type object = map[string]any
type process struct {
	Attempt  int   `json:"attempt"`
	Exit     *int  `json:"exit_code"`
	Joined   *bool `json:"process_joined"`
	Timeout  *bool `json:"timeout"`
	Released *bool `json:"writer_released_fifo_reader"`
	Absent   *bool `json:"output_directory_absent"`
}

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
func raw(path string) []byte       { b, err := os.ReadFile(path); must(err); return b }
func read(path string, target any) { must(json.Unmarshal(raw(path), target)) }
func equal(path, expected string) {
	check(bytes.Equal(raw(path), raw(expected)), "bytes differ: "+path)
}

func completed(root, expected, input, source string) {
	var summary struct {
		Rows []struct {
			Status        string
			Native        int `json:"native_runs"`
			Passed, Total int
		}
	}
	read(filepath.Join(root, "summary.json"), &summary)
	check(len(summary.Rows) == 1 && summary.Rows[0].Status == "completed" && summary.Rows[0].Native == 2 && summary.Rows[0].Passed == 128 && summary.Rows[0].Total == 128, "scope changed")
	var runtime struct {
		Observation struct {
			Replayed bool `json:"runtime_replayed"`
			Declared int  `json:"declared_cases"`
			Cases    []struct {
				Input, Expected, Actual int64
				Passed                  bool
			}
		}
	}
	read(filepath.Join(root, "run-1-runtime.json"), &runtime)
	var inputs struct {
		Cases []struct{ Input, Expected int64 }
	}
	read(filepath.Join(input, "cases-128.json"), &inputs)
	check(runtime.Observation.Replayed && runtime.Observation.Declared == 128 && len(runtime.Observation.Cases) == 128 && len(inputs.Cases) == 128, "runtime observation changed")
	for i, c := range runtime.Observation.Cases {
		check(c.Passed && c.Input == inputs.Cases[i].Input && c.Expected == inputs.Cases[i].Expected && c.Actual == c.Expected, "native output changed")
	}
	var generation struct {
		Report struct {
			BodyPaths struct {
				Search struct {
					Selection struct {
						Local    *int `json:"local_model_predictions"`
						External *int `json:"external_provider_calls"`
					}
				}
			} `json:"body_paths"`
		}
	}
	read(filepath.Join(root, "run-1-generation.json"), &generation)
	s := generation.Report.BodyPaths.Search.Selection
	check(s.Local != nil && s.External != nil && *s.Local == 0 && *s.External == 0, "prediction accounting changed")
	var timing struct {
		Source   string `json:"producer_source_sha"`
		Modified string `json:"producer_modified"`
	}
	read(filepath.Join(root, "run-1-timing.json"), &timing)
	check(timing.Source == source && timing.Modified == "false", "clean source binding changed")
	equal(filepath.Join(root, "source.gooo"), filepath.Join(input, "ko-source.gooo"))
	equal(filepath.Join(root, "recipe.json"), filepath.Join(input, "ko-recipe.json"))
	equal(filepath.Join(root, "cases.json"), filepath.Join(input, "cases-128.json"))
	equal(filepath.Join(root, "run-1-generated.go"), filepath.Join(expected, "ko-deterministic", "run-1-generated.go"))
}

func inspect(root, scope, source, expected, input string, count, releases int) object {
	var summary struct {
		Attempts  int
		Confirmed int `json:"confirmed_fifo_waits"`
		Records   []process
	}
	read(filepath.Join(root, scope, "summary.json"), &summary)
	check(summary.Attempts == count && len(summary.Records) == count && summary.Confirmed == releases, "attempt scope changed")
	accepted, rejected, writers := 0, 0, 0
	for i, p := range summary.Records {
		check(p.Attempt == i+1 && p.Exit != nil && p.Joined != nil && *p.Joined && p.Timeout != nil && !*p.Timeout && p.Released != nil && p.Absent != nil, "unjoined/missing process facts")
		label := fmt.Sprintf("attempt-%02d", p.Attempt)
		var original process
		read(filepath.Join(root, scope, label+"-process.json"), &original)
		check(original.Attempt == p.Attempt && original.Exit != nil && *original.Exit == *p.Exit && original.Joined != nil && *original.Joined && original.Timeout != nil && !*original.Timeout && original.Released != nil && *original.Released == *p.Released && original.Absent != nil && *original.Absent == *p.Absent, "process differs from summary")
		if *p.Released {
			writers++
		}
		dest := filepath.Join(root, scope, label+"-results")
		if *p.Exit == 0 {
			check(!*p.Absent && !*p.Released, "regular completion confused with wait")
			completed(dest, expected, input, source)
			accepted++
		} else {
			check(*p.Exit == 1 && *p.Absent && len(raw(filepath.Join(root, scope, label+".stdout"))) == 0 && len(raw(filepath.Join(root, scope, label+".stderr"))) > 0, "early failure facts changed")
			_, err := os.Stat(dest)
			check(os.IsNotExist(err), "early failure generated a directory")
			rejected++
		}
	}
	check(writers == releases, "release scope changed")
	return object{"scope": scope, "source": source, "attempts": count, "regular_completions": accepted, "early_file_errors": rejected, "writer_releases": writers, "native_runs": accepted * 2, "observed_finite_passed": accepted * 128, "observed_finite_total": accepted * 128, "frozen_go_comparisons": accepted, "model_predictions": 0, "new_intent_tasks": 0}
}

func main() {
	check(len(os.Args) == 4 || len(os.Args) == 5, "usage: swap-readback saved-root inputs frozen-Go [revision2]")
	root, input, expected := os.Args[1], os.Args[2], os.Args[3]
	rows := []object{
		inspect(root, "baseline-race", "01d21e9260b2c0f1d02f8029824f3dda41631e9d", expected, input, 14, 1),
		inspect(root, "candidate-race", "1a47cb16b0abb9fe7faaaace158612f9b77ee504", expected, input, 64, 0),
	}
	if len(os.Args) == 5 {
		check(os.Args[4] == "revision2", "unknown revision")
		rows = append(rows, inspect(root, "revision2-race", "0eb69e5f15ce40a2b07050ae747fe1552d5a9f46", expected, input, 64, 0))
	}
	must(json.NewEncoder(os.Stdout).Encode(object{"status": "PASS", "scope": "saved swap attempts only; no new execution", "rows": rows, "model_predictions_added": 0, "native_runs_added": 0}))
}
