package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type pairedTotals struct {
	Constructions int       `json:"constructions"`
	ActivePassed  int       `json:"active_fields_passed"`
	ActiveTotal   int       `json:"active_fields_total"`
	GuardPassed   int       `json:"guard_fields_passed"`
	GuardTotal    int       `json:"guard_fields_total"`
	ChangedPassed int       `json:"required_changed_fields_passed"`
	ChangedTotal  int       `json:"required_changed_fields_total"`
	NamedPassed   int       `json:"named_outputs_passed"`
	NamedTotal    int       `json:"named_outputs_total"`
	Attempts      int       `json:"candidate_attempts"`
	BuildRSS      int64     `json:"build_peak_rss_bytes_max"`
	RunRSS        int64     `json:"run_peak_rss_bytes_max"`
	CPU           float64   `json:"outer_cpu_seconds_sum"`
	Wall          []float64 `json:"wall_ms_observations"`
}

func summarizePairedStudy(root, output string) {
	if root == "" || output == "" {
		panic("complete study directory and fresh summary file required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh paired summary required")
	}
	read := func(name string, value any) {
		raw, err := os.ReadFile(filepath.Join(root, name))
		check(err)
		check(json.Unmarshal(raw, value))
	}
	var audit struct {
		Schema string `json:"schema"`
		Rows   int    `json:"rows"`
		Parity int    `json:"parity_records"`
		Models map[string]struct {
			Counts   map[string]map[string]int `json:"counts"`
			Times    map[string]int64          `json:"predict_ns_median"`
			Weight   int                       `json:"weight_file_bytes"`
			Resident int                       `json:"resident_tensor_bytes"`
		} `json:"models"`
	}
	read("audit/report.json", &audit)
	if audit.Schema != "gooo/record-paired-model-audit/v2" || audit.Rows != pairedTotal || audit.Parity != 120 || len(audit.Models) != 4 {
		panic("complete audited paired models required")
	}
	var training struct {
		Updates int    `json:"actual_optimizer_updates"`
		Status  string `json:"status"`
		Stages  map[string]struct {
			Wall   float64 `json:"wall_seconds"`
			CPU    float64 `json:"cpu_seconds"`
			Tensor int64   `json:"sampled_mps_tensor_peak_bytes"`
			Driver int64   `json:"sampled_mps_driver_peak_bytes"`
		} `json:"training"`
	}
	read("trained/report.json", &training)
	if training.Updates != 3840 || training.Status != "TRAINED_AND_EXPORTED" || len(training.Stages) != 2 {
		panic("complete actual paired training required")
	}
	resourceRaw, err := os.ReadFile(filepath.Join(root, "training-resources.txt"))
	check(err)
	var real, user, system float64
	var rss int64
	lines := strings.Split(strings.TrimSpace(string(resourceRaw)), "\n")
	if len(lines) < 2 {
		panic("complete process resource capture required")
	}
	_, err = fmt.Sscanf(strings.TrimSpace(lines[0]), "%f real %f user %f sys", &real, &user, &system)
	check(err)
	_, err = fmt.Sscanf(strings.TrimSpace(lines[1]), "%d maximum resident set size", &rss)
	check(err)
	if real <= 0 || user+system <= 0 || rss <= 0 {
		panic("process resource measurement missing")
	}
	nativeRaw, err := os.ReadFile(filepath.Join(root, "native/summaries.jsonl"))
	check(err)
	scan := bufio.NewScanner(bytes.NewReader(nativeRaw))
	scan.Buffer(make([]byte, 4096), 1<<20)
	groups := map[string]*pairedTotals{}
	seen := map[string]bool{}
	for scan.Scan() {
		var s PairedNativeSummary
		check(json.Unmarshal(scan.Bytes(), &s))
		if s.ActiveTotal != 6 || s.GuardTotal != 6 || s.RuntimeTotal != 8 || s.ChangedTotal <= 0 || s.FieldsPassed != s.ActivePassed+s.GuardPassed {
			panic("native denominators differ")
		}
		id := fmt.Sprintf("%s-%s-b%d", s.ID, s.Profile, s.Budget)
		if seen[id] {
			panic("duplicate native summary")
		}
		seen[id] = true
		axis := "source"
		if s.Style > 0 {
			axis = "wording_new"
		}
		key := fmt.Sprintf("%s/%s/b%d", axis, s.Profile, s.Budget)
		if groups[key] == nil {
			groups[key] = &pairedTotals{}
		}
		g := groups[key]
		g.Constructions++
		g.ActivePassed += s.ActivePassed
		g.ActiveTotal += s.ActiveTotal
		g.GuardPassed += s.GuardPassed
		g.GuardTotal += s.GuardTotal
		g.ChangedPassed += s.ChangedPassed
		g.ChangedTotal += s.ChangedTotal
		g.NamedPassed += s.RuntimePassed
		g.NamedTotal += s.RuntimeTotal
		g.Attempts += s.Attempts
		g.BuildRSS = max(g.BuildRSS, s.BuildRSS)
		g.RunRSS = max(g.RunRSS, s.RunRSS)
		g.CPU += s.CPUSeconds
		g.Wall = append(g.Wall, s.WallMS)
	}
	check(scan.Err())
	if len(seen) != 360 || len(groups) != 30 {
		panic("complete matched native groups required")
	}
	medians := map[string]float64{}
	for key, g := range groups {
		expected := 16
		if strings.HasPrefix(key, "wording_new/") {
			expected = 8
		}
		if g.Constructions != expected {
			panic("matched group denominator differs")
		}
		slices.Sort(g.Wall)
		medians[key] = (g.Wall[(len(g.Wall)-1)/2] + g.Wall[len(g.Wall)/2]) / 2
	}
	loopWall, loopCPU := 0., 0.
	var tensor, driver int64
	for _, stage := range training.Stages {
		loopWall += stage.Wall
		loopCPU += stage.CPU
		tensor = max(tensor, stage.Tensor)
		driver = max(driver, stage.Driver)
	}
	save(output, map[string]any{"schema": "gooo/record-paired-readable-summary/v2", "audit_models": audit.Models, "native_groups": groups, "native_wall_ms_median": medians,
		"resources": map[string]any{"actual_optimizer_updates": training.Updates, "optimization_wall_seconds": loopWall, "optimization_cpu_seconds": loopCPU, "training_process_wall_seconds": real, "training_process_cpu_seconds": user + system, "training_process_average_one_core_percent": 100 * (user + system) / real, "training_process_peak_rss_bytes": rss, "sampled_mps_tensor_peak_bytes": tensor, "sampled_mps_driver_peak_bytes": driver, "host_cpu_utilization_delta_measured": false, "gpu_utilization_percent_measured": false},
		"scope":     "Matched compiler constructions; unchanged guards are separate from active fields. Required-changed fields are a subset of active fields. Process-average CPU uses CPU/wall time in one-core units and is not whole-host utilization. Native values were independently recounted separately."})
	fmt.Println("Gooo paired intent pilot: first complete mask / first two / first four; individual field requirements")
	for _, axis := range []string{"test_source", "test_wording_seen", "test_wording_new"} {
		for _, profile := range []string{"frozen_field_v1", "fp32", "ptq_ternary", "qat_ternary"} {
			m := audit.Models[profile]
			c := m.Counts[axis]
			denominator := 1536
			if axis != "test_source" {
				denominator = 512
			}
			if c["total"] != denominator || c["field_requirements_total"] != 3*denominator {
				panic("audit axis denominator differs")
			}
			fmt.Printf("%s %s: %d/%d (%.2f%%), %d/%d, %d/%d; fields %d/%d; Go %.3fµs\n", axis, profile, c["target_in_first_1"], denominator, 100*float64(c["target_in_first_1"])/float64(denominator), c["target_in_first_2"], denominator, c["target_in_first_4"], denominator, c["field_requirements_correct"], c["field_requirements_total"], float64(m.Times[axis])/1000)
		}
	}
	fmt.Println("Native budget one: active fields; unchanged guards; required changed fields")
	for _, axis := range []string{"source", "wording_new"} {
		for _, profile := range []string{"deterministic", "frozen_field_v1", "fp32", "ptq_ternary", "qat_ternary"} {
			g := groups[axis+"/"+profile+"/b1"]
			fmt.Printf("%s %s: %d/%d; %d/%d; %d/%d\n", axis, profile, g.ActivePassed, g.ActiveTotal, g.GuardPassed, g.GuardTotal, g.ChangedPassed, g.ChangedTotal)
		}
	}
	fmt.Printf("Training: %d updates; %.2fs process wall; %.2fs process CPU; %.2fMiB peak RSS; measured average %.2f%% of one CPU core. GPU utilization unmeasured.\n", training.Updates, real, user+system, float64(rss)/(1<<20), 100*(user+system)/real)
}
