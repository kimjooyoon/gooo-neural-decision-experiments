package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
)

type processSummary struct {
	CPU      float64 `json:"aggregate_cpu_percent_of_one_core"`
	Children int     `json:"children"`
	MaxRSS   int64   `json:"max_child_peak_rss_bytes"`
	RSS      int64   `json:"median_child_peak_rss_bytes"`
	Median   int64   `json:"median_wall_ns"`
	P95      int64   `json:"nearest_rank_p95_wall_ns"`
}

func summarizeProcesses(values []metrics) (processSummary, error) {
	if len(values) != 48 {
		return processSummary{}, errors.New("48 measured child processes required")
	}
	var walls, resident [48]int64
	var wall, cpu int64
	for i, m := range values {
		if !validMetrics(m) || m.User > math.MaxInt64-m.System || wall > math.MaxInt64-m.Wall || cpu > math.MaxInt64-m.User-m.System {
			return processSummary{}, errors.New("invalid or overflowing child measurements")
		}
		walls[i], resident[i] = m.Wall, m.RSS
		wall += m.Wall
		cpu += m.User + m.System
	}
	slices.Sort(walls[:])
	slices.Sort(resident[:])
	return processSummary{Children: 48, Median: walls[23] + (walls[24]-walls[23])/2, P95: walls[45],
		RSS: resident[23] + (resident[24]-resident[23])/2, MaxRSS: resident[47], CPU: 100 * float64(cpu) / float64(wall)}, nil
}

func nativeProcessSummary(dataset, models, sdk, captures, output string) error {
	if output == "" {
		return errors.New("fresh metrics report required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh metrics report required")
	}
	// Reconstruct source, paths and ordered execution values before summarizing.
	// This audit reads captures; it does not execute inference or generated Go.
	temp, err := os.MkdirTemp("", "gooo-native-metrics-audit-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	if err = nativeStudy(dataset, models, sdk, captures, "", "", "", filepath.Join(temp, "audit.json")); err != nil {
		return err
	}
	cells := map[string]map[string]processSummary{}
	for _, policy := range []string{"offline", "selected", "set-feedback", "set-initial", "uniform-initial"} {
		var codegen, execution []metrics
		for _, family := range jointcompositionstudy.Families {
			for goal := 0; goal < 4; goal++ {
				for _, language := range []string{"en", "ko"} {
					var x executionCapture
					name := fmt.Sprintf("%s-goal%d-%s-%s-execution.json", family, goal, language, policy)
					if err = read(filepath.Join(captures, name), &x); err != nil {
						return err
					}
					codegen = append(codegen, x.Codegen)
					execution = append(execution, x.Execution)
				}
			}
		}
		c, err := summarizeProcesses(codegen)
		if err != nil {
			return err
		}
		g, err := summarizeProcesses(execution)
		if err != nil {
			return err
		}
		cells[policy] = map[string]processSummary{"actual_codegen_children": c, "actual_compile_and_run_children": g}
	}
	return save(output, map[string]any{"schema": "gooo/own-joint-feedback-native-process-summary/v2", "status": "PASS",
		"actual_codegen_children": 240, "actual_compile_and_run_children": 240, "policies": cells,
		"scope": "Recorded child process metrics; fixed policy order and already observed corpus. Compile-and-run includes compilation. CPU normalized to one core, RSS is a child's measured peak; neither causal host CPU change nor simultaneous process-tree RAM is measured."})
}
