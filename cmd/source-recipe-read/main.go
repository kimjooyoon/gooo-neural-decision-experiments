// Read saved receipts without calling a model, compiler or network provider.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type process struct {
	WallMS float64 `json:"wall_ms"`
	CPUMS  float64 `json:"cpu_ms"`
	CPU    float64 `json:"cpu_one_core_percent"`
	RSS    int64   `json:"peak_rss_bytes"`
}
type row struct {
	ID, Input      string
	Model, Resolve bool
	Generation     process
	Calls          int `json:"model_calls"`
	Passed, Total  int
	Document       string `json:"document_sha256"`
	Source         string `json:"emitted_sha256"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(path string, v any)    { b, err := os.ReadFile(path); must(err); must(json.Unmarshal(b, v)) }
func digest(b []byte) string     { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func median(v []float64) float64 { slices.Sort(v); return v[len(v)/2] }

func main() {
	dir := flag.String("dir", "publication/source-recipes-20261003", "saved observation directory")
	flag.Parse()
	var rows []row
	read(filepath.Join(*dir, "processes.json"), &rows)
	if len(rows) != 24 {
		panic("expected 24 recorded generations")
	}
	groups := map[string][]row{}
	pairs := map[string][]row{}
	ids := map[string]bool{}
	calls, passed, total, oraclePassed, oracleTotal := 0, 0, 0, 0, 0
	for _, r := range rows {
		if ids[r.ID] || filepath.Base(r.ID) != r.ID || r.Input != "recipe" && r.Input != "full" {
			panic("invalid row identity")
		}
		ids[r.ID] = true
		var g struct {
			Source string
			Report struct {
				Paths struct {
					Document string `json:"document_sha256"`
					Search   struct {
						Selection struct {
							Calls int `json:"local_model_predictions"`
						}
					} `json:"search"`
				} `json:"body_paths"`
			}
		}
		read(filepath.Join(*dir, r.ID+"-generation.json"), &g)
		if digest([]byte(g.Source)) != r.Source || g.Report.Paths.Document != r.Document || g.Report.Paths.Search.Selection.Calls != r.Calls {
			panic("generation record mismatch")
		}
		var runtime struct {
			Parent      []byte `json:"parent_receipt_bytes"`
			Observation struct {
				Stage string
				Cases []struct {
					Passed                  bool
					Input, Expected, Actual int64
				}
				Runs []struct {
					Completed bool
					ExitCode  int `json:"exit_code"`
				}
			}
		}
		read(filepath.Join(*dir, r.ID+"-runtime.json"), &runtime)
		if runtime.Observation.Stage != "COMPLETE" || len(runtime.Observation.Runs) != 2 {
			panic("missing compiled runs")
		}
		p := 0
		for _, c := range runtime.Observation.Cases {
			if c.Passed {
				p++
			}
		}
		if p != r.Passed || len(runtime.Observation.Cases) != r.Total || r.Total != 6 {
			panic("runtime count mismatch")
		}
		for _, run := range runtime.Observation.Runs {
			if !run.Completed || run.ExitCode != 0 {
				panic("execution incomplete")
			}
		}
		for _, private := range []string{"/Users/", "/private/", "/var/folders/", "BEGIN PRIVATE KEY", "hf_", "ghp_"} {
			if bytes.Contains(runtime.Parent, []byte(private)) {
				panic("private content in embedded parent")
			}
		}
		want := 0
		if r.Model && !r.Resolve {
			want = 1
		}
		if r.Calls != want {
			panic("joint prediction count mismatch")
		}
		m := r.Generation
		if m.WallMS <= 0 || m.CPUMS <= 0 || m.RSS <= 0 || math.Abs(m.CPU-100*m.CPUMS/m.WallMS) > 1e-7 {
			panic("process accounting mismatch")
		}
		key := fmt.Sprintf("%s/model=%t/resolve=%t", r.Input, r.Model, r.Resolve)
		groups[key] = append(groups[key], r)
		pairKey := strings.TrimPrefix(r.ID, r.Input+"-")
		pairs[pairKey] = append(pairs[pairKey], r)
		calls += r.Calls
		passed += r.Passed
		total += r.Total
		if r.Resolve {
			oraclePassed += r.Passed
			oracleTotal += r.Total
		}
	}
	for _, pair := range pairs {
		if len(pair) != 2 || pair[0].Input == pair[1].Input || pair[0].Document != pair[1].Document || pair[0].Source != pair[1].Source || pair[0].Passed != pair[1].Passed {
			panic("paired source or document mismatch")
		}
	}
	var keys []string
	for key := range groups {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var cells []any
	for _, key := range keys {
		var wall, cpu, rss []float64
		p, n := 0, 0
		for _, r := range groups[key] {
			wall = append(wall, r.Generation.WallMS)
			cpu = append(cpu, r.Generation.CPU)
			rss = append(rss, float64(r.Generation.RSS)/1048576)
			p += r.Passed
			n += r.Total
		}
		if len(wall) != 3 {
			panic("unexpected repetition count")
		}
		cells = append(cells, map[string]any{"cell": key, "repetitions": 3, "median_wall_ms": median(wall), "median_cpu_one_core_percent": median(cpu), "median_peak_rss_mib": median(rss), "passed": p, "total": n})
	}
	var manifest struct {
		Generations   int
		Calls         int `json:"model_predictions"`
		Passed, Total int
		Pairs         int `json:"paired_equal_documents_and_emissions"`
	}
	read(filepath.Join(*dir, "manifest.json"), &manifest)
	if manifest.Generations != len(rows) || manifest.Calls != calls || manifest.Passed != passed || manifest.Total != total || manifest.Pairs != len(pairs) {
		panic("manifest disagreement")
	}
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"schema": "gooo/source-recipe-independent-reading/v1", "status": "PASS", "paired_documents_and_emissions": len(pairs), "compiled_runs": 2 * len(rows), "model_predictions": calls, "passed": passed, "total": total, "oracle_passed": oraclePassed, "oracle_total": oracleTotal, "cells": cells, "reader_model_calls": 0}))
}
