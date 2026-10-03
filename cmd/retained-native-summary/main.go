// Summarize completed retained-native-runtime records without mixing current
// resource observations with the historical source build.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type row struct {
	ID, Request, Mode, RuntimeSHA, GenerationSHA                                            string
	Budget, Trial, ModelCalls, Passed, Total, NativeRuns                                    int
	Reused, Built                                                                           bool
	ExecuteNS, BuildNS, FirstRunNS, SecondRunNS, CurrentChildCPUNS, MaxCurrentChildRSSBytes int64
}
type distribution struct{ Median, Min, Max float64 }

func stats(values []int64, divisor float64) distribution {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	n := len(values)
	if n == 0 {
		return distribution{}
	}
	median := float64(values[n/2])
	if n%2 == 0 {
		median = (median + float64(values[n/2-1])) / 2
	}
	return distribution{median / divisor, float64(values[0]) / divisor, float64(values[n-1]) / divisor}
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	if len(os.Args) != 2 {
		panic("usage: retained-native-summary completed-records.json")
	}
	b, err := os.ReadFile(os.Args[1])
	must(err)
	var rows []row
	must(json.Unmarshal(b, &rows))
	if len(rows) != 512 {
		panic("full 512-observation cohort required")
	}
	counts := map[string]int{"generations": len(rows)}
	groups := map[string]map[string][]int64{}
	pairs := map[string]map[string]row{}
	requests, seen := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		if seen[r.ID] || (r.Mode != "fresh" && r.Mode != "retained") || r.Trial < 0 || r.Trial > 1 || r.ModelCalls != 1 || r.Total != 8 || r.NativeRuns != 2 || r.Passed < 0 || r.Passed > r.Total {
			panic("invalid cohort record")
		}
		seen[r.ID], requests[r.Request] = true, true
		counts["predictions"] += r.ModelCalls
		counts["native_runs"] += r.NativeRuns
		counts["passed"] += r.Passed
		counts["total"] += r.Total
		if r.Built {
			counts["actual_builds"]++
		}
		if r.Reused {
			counts["artifact_reuses"]++
		}
		key := fmt.Sprintf("%s/trial%d", r.Mode, r.Trial)
		if groups[key] == nil {
			groups[key] = map[string][]int64{}
		}
		for name, value := range map[string]int64{"execution_ms": r.ExecuteNS, "build_ms": r.BuildNS, "first_run_ms": r.FirstRunNS, "second_run_ms": r.SecondRunNS, "current_child_cpu_ms": r.CurrentChildCPUNS, "max_current_child_rss_bytes": r.MaxCurrentChildRSSBytes} {
			groups[key][name] = append(groups[key][name], value)
		}
		if r.Trial == 1 {
			p := fmt.Sprintf("%s/b%d", r.Request, r.Budget)
			if pairs[p] == nil {
				pairs[p] = map[string]row{}
			}
			pairs[p][r.Mode] = r
		}
	}
	counts["sources"], counts["warm_pairs"] = len(requests), len(pairs)
	if counts["sources"] != 64 || counts["warm_pairs"] != 128 || counts["actual_builds"] != 384 || counts["artifact_reuses"] != 128 {
		panic("ownership counts differ")
	}
	var changes []int64
	for _, p := range pairs {
		fresh, a := p["fresh"]
		retained, z := p["retained"]
		if !a || !z || fresh.Passed != retained.Passed || !retained.Reused || retained.Built || !fresh.Built {
			panic("paired outcomes/ownership differ")
		}
		delta := fresh.ExecuteNS - retained.ExecuteNS
		changes = append(changes, delta)
		if delta > 0 {
			counts["reuse_faster_pairs"]++
		} else if delta < 0 {
			counts["reuse_slower_pairs"]++
		} else {
			counts["equal_pairs"]++
		}
	}
	result := map[string]any{}
	for group, axes := range groups {
		s := map[string]distribution{}
		for name, values := range axes {
			divisor := 1e6
			if name == "max_current_child_rss_bytes" {
				divisor = 1
			}
			s[name] = stats(values, divisor)
		}
		result[group] = s
	}
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{
		"schema": "gooo/retained-native-summary/v1", "records_sha256": fmt.Sprintf("%x", sha256.Sum256(b)),
		"counts": counts, "distributions": result, "warm_paired_saved_ms": stats(changes, 1e6),
		"scope": "Alternating-order known development cohort. Current children only; maximum single-child peak RSS, not parent/whole-host RSS. Partial failures retained. Timing is a same-machine observation, not a cross-platform guarantee.",
	}))
}
