package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type process struct {
	Wall   int64 `json:"wall_ns"`
	User   int64 `json:"user_cpu_ns"`
	System int64 `json:"system_cpu_ns"`
	RSS    int64 `json:"lifetime_peak_rss_bytes"`
}
type observation struct {
	Calls    int     `json:"local_model_predictions"`
	Attempts int     `json:"attempted_candidates"`
	Passed   int     `json:"selected_finite_passed"`
	Cases    int     `json:"finite_cases"`
	Stage    float64 `json:"native_stage_ms"`
	Process  process `json:"process"`
}

func median(x []float64) float64 { sort.Float64s(x); return (x[(len(x)-1)/2] + x[len(x)/2]) / 2 }
func main() {
	if len(os.Args) != 3 {
		panic("report and fresh summary paths required")
	}
	if _, err := os.Lstat(os.Args[2]); !os.IsNotExist(err) {
		panic("summary must be fresh")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var report struct {
		Status      string `json:"status"`
		NativeCalls int    `json:"native_calls"`
		Predictions int    `json:"fresh_local_model_predictions"`
		Cells       []struct {
			Arm      string        `json:"arm"`
			Contract string        `json:"contract"`
			Same     bool          `json:"same_final_body_and_search"`
			Prefixes int           `json:"matched_progress_prefixes"`
			Restart  []observation `json:"restart_observations"`
			Batch    observation   `json:"batched_observation"`
		} `json:"cells"`
	}
	if err = json.Unmarshal(raw, &report); err != nil {
		panic(err)
	}
	if report.Status != "PASS" || len(report.Cells) != 32 || report.NativeCalls != 288 || report.Predictions != 1296 {
		panic("incomplete fixed comparison")
	}
	var summaries []map[string]any
	for _, arm := range []string{"all", "offline", "fp32", "ptq_ternary", "qat_ternary"} {
		var restartWall, batchWall, restartCPU, batchCPU, restartStage, batchStage, restartRSS, batchRSS []float64
		var restartCalls, batchCalls, restartAttempts, batchAttempts, passed, cases, partials, prefixes int
		for _, c := range report.Cells {
			if arm != "all" && c.Arm != arm {
				continue
			}
			if !c.Same {
				panic("semantic mismatch")
			}
			wall, cpu, stage, rss := float64(0), float64(0), float64(0), int64(0)
			for _, r := range c.Restart {
				wall += float64(r.Process.Wall) / 1e6
				cpu += float64(r.Process.User+r.Process.System) / 1e6
				stage += r.Stage
				rss = max(rss, r.Process.RSS)
				restartCalls += r.Calls
				restartAttempts += r.Attempts
			}
			b := c.Batch
			restartWall = append(restartWall, wall)
			restartCPU = append(restartCPU, cpu)
			restartStage = append(restartStage, stage)
			restartRSS = append(restartRSS, float64(rss))
			batchRSS = append(batchRSS, float64(b.Process.RSS))
			batchWall = append(batchWall, float64(b.Process.Wall)/1e6)
			batchCPU = append(batchCPU, float64(b.Process.User+b.Process.System)/1e6)
			batchStage = append(batchStage, b.Stage)
			batchCalls += b.Calls
			batchAttempts += b.Attempts
			passed += b.Passed
			cases += b.Cases
			prefixes += c.Prefixes
			if b.Passed < b.Cases {
				partials++
			}
		}
		summaries = append(summaries, map[string]any{"arm": arm, "cells": len(batchWall), "restart_model_predictions": restartCalls,
			"continued_model_predictions": batchCalls, "restart_candidate_attempts": restartAttempts, "continued_candidate_attempts": batchAttempts,
			"median_restart_eight_call_wall_ms": median(restartWall), "median_continued_one_call_wall_ms": median(batchWall),
			"median_restart_eight_call_cpu_ms": median(restartCPU), "median_continued_one_call_cpu_ms": median(batchCPU),
			"median_restart_summed_native_stage_ms": median(restartStage), "median_continued_native_stage_ms": median(batchStage),
			"median_restart_max_child_rss_bytes": median(restartRSS), "median_continued_child_rss_bytes": median(batchRSS),
			"final_finite_passed": passed, "final_finite_cases": cases, "partial_cells_retained": partials, "matched_progress_prefixes": prefixes})
	}
	result := map[string]any{"schema": "gooo/incremental-native-summary/v1", "status": "PASS", "native_calls": report.NativeCalls,
		"fresh_local_model_predictions": report.Predictions, "optimizer_steps": 0, "gpu_training": false, "external_provider_calls": 0, "summaries": summaries,
		"scope": "One host and frozen compound intent. Eight independent budget requests versus one continued request; not equal-call microbenchmark throughput. CPU is summed child CPU time, not host utilization rise. RSS compares the maximum child high-water per policy, not additive RAM or model footprint. Two repeats per cell. Finite cases are used for selection and are not a new holdout."}
	raw, err = json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(os.Args[2], append(raw, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
}
