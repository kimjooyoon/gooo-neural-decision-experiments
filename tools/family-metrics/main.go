// family-metrics reports descriptive metrics from frozen actual native captures.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
)

type observation struct {
	Arm            string `json:"arm"`
	Feedback       bool   `json:"feedback_enabled"`
	Predictions    int    `json:"actual_model_predictions"`
	NoChoice       int    `json:"zero_call_ranking_unnecessary_receipts"`
	Passed         int    `json:"finite_passed"`
	Cases          int    `json:"finite_cases"`
	SeparatePassed int    `json:"separate_cases_passed"`
	SeparateCases  int    `json:"separate_cases"`
	Metrics        struct {
		Wall   int64   `json:"wall_ns"`
		User   int64   `json:"user_cpu_ns"`
		System int64   `json:"system_cpu_ns"`
		RSS    int64   `json:"child_max_rss_bytes"`
		CPU    float64 `json:"process_cpu_percent_of_one_core_over_wall"`
	} `json:"process_metrics"`
}
type summary struct {
	Edition        string  `json:"edition"`
	Arm            string  `json:"arm"`
	Views          int     `json:"views"`
	Predictions    int     `json:"actual_model_predictions"`
	NoChoice       int     `json:"zero_call_receipts"`
	Passed         int     `json:"finite_passed"`
	Cases          int     `json:"finite_cases"`
	SeparatePassed int     `json:"separate_passed"`
	SeparateCases  int     `json:"separate_cases"`
	Wall           float64 `json:"median_wall_ms"`
	P95            float64 `json:"nearest_rank_p95_wall_ms"`
	CPUTime        float64 `json:"median_process_cpu_ms"`
	CPU            float64 `json:"median_process_cpu_percent_of_one_core"`
	RSS            float64 `json:"median_process_peak_rss_bytes"`
}

func median(values []float64) float64 {
	sort.Float64s(values)
	if len(values)%2 == 1 {
		return values[len(values)/2]
	}
	return (values[len(values)/2-1] + values[len(values)/2]) / 2
}
func summarize(edition, key string, rows []observation) (summary, error) {
	s := summary{Edition: edition, Arm: key, Views: len(rows)}
	if len(rows) != 120 {
		return s, errors.New("120 repeated policy views per arm required")
	}
	var wall, cpu, rss, cpuTime []float64
	for _, o := range rows {
		m := o.Metrics
		if m.Wall <= 0 || m.RSS <= 0 || m.User < 0 || m.System < 0 || m.CPU != 100*float64(m.User+m.System)/float64(m.Wall) {
			return s, errors.New("process resource counters differ")
		}
		wall = append(wall, float64(m.Wall)/1e6)
		cpuTime = append(cpuTime, float64(m.User+m.System)/1e6)
		cpu = append(cpu, m.CPU)
		rss = append(rss, float64(m.RSS))
		s.Predictions += o.Predictions
		s.NoChoice += o.NoChoice
		s.Passed += o.Passed
		s.Cases += o.Cases
		s.SeparatePassed += o.SeparatePassed
		s.SeparateCases += o.SeparateCases
	}
	s.Wall = median(wall)
	s.P95 = wall[int(math.Ceil(.95*float64(len(wall))))-1]
	s.CPUTime = median(cpuTime)
	s.CPU = median(cpu)
	s.RSS = median(rss)
	return s, nil
}
func generate() (map[string]any, error) {
	var summaries []summary
	pins := map[string]string{}
	for _, item := range []struct{ Edition, Root, SHA string }{
		{"baseline_sdk_2_4_main", "runs/feedback-family-matrix-20261001", "9c00cf48dae46bdc793b39c5ae94f15c825194d33eabe2bcf3002161aba87c2d"},
		{"optimized_sdk_2_5_feature", "runs/feedback-family-optimized-feature-20261001", "0346231d22f540a844fc9296b408a64c59bb33d538223603c10f65a281c63d62"},
	} {
		path := item.Root + "/report.json"
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 2<<20 {
			return nil, errors.New("bounded frozen report required")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(raw)
		if hex.EncodeToString(h[:]) != item.SHA {
			return nil, errors.New("frozen report hash changed")
		}
		pins[path] = item.SHA
		var report struct {
			Rows []observation `json:"observations"`
		}
		if err = json.Unmarshal(raw, &report); err != nil || len(report.Rows) != 1080 {
			return nil, errors.New("frozen policy view count differs")
		}
		groups := map[string][]observation{}
		for _, o := range report.Rows {
			key := o.Arm
			if o.Feedback {
				key += "/feedback"
			} else {
				key += "/rank_once"
			}
			groups[key] = append(groups[key], o)
		}
		var keys []string
		for k := range groups {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) != 9 {
			return nil, errors.New("frozen arm count differs")
		}
		for _, key := range keys {
			s, err := summarize(item.Edition, key, groups[key])
			if err != nil {
				return nil, err
			}
			summaries = append(summaries, s)
		}
	}
	return map[string]any{"schema": "gooo/native-family-descriptive-metrics/v1", "decision": "PASS", "source_report_sha256": pins, "summaries": summaries, "new_model_predictions": 0, "new_native_calls": 0, "new_go_processes": 0,
		"scope": "Whole native compiler subprocess lifetime per repeated policy view; CPU percentage is process CPU time divided by wall time in units of one core, not host utilization or utilization increase. Single baseline-first ordering per edition; no repeated randomized trials, isolated model kernel attribution or causal speedup claim. Every recorded outlier retained. Main SDK 2.4 and feature SDK 2.5 deployment states remain distinct. Separate-input observations reuse actual Go executions."}, nil
}
func main() {
	out := flag.String("output", "", "descriptive metrics output")
	flag.Parse()
	value, err := generate()
	if err == nil {
		var raw []byte
		raw, err = json.MarshalIndent(value, "", "  ")
		if err == nil {
			if *out == "" {
				err = errors.New("output required")
			} else {
				err = os.WriteFile(*out, append(raw, '\n'), 0644)
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "family-metrics:", err)
		os.Exit(1)
	}
}
