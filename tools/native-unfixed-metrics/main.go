// native-unfixed-metrics summarizes retained observations without inference.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

var reportPins = map[string]string{
	"runs/unfixed-native-main-pilot-20261001":    "6bf4030ee7fe769fd3befe8b5bee66bb6fe00fe272037153f4197d59cc6818f6",
	"runs/unfixed-native-feature-pilot-20261001": "4dba2dab42c17af2c069a442b18053fc804bc50a32f5b4276799ddf3f96d3962",
}

func digest(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func distribution(values []float64) map[string]float64 {
	sort.Float64s(values)
	n := len(values)
	median := values[n/2]
	if n%2 == 0 {
		median = (values[n/2-1] + median) / 2
	}
	return map[string]float64{"minimum": values[0], "p50": median, "p95": values[(95*n+99)/100-1], "maximum": values[n-1]}
}
func run(root, output string) error {
	raw, err := os.ReadFile(filepath.Join(root, "report.json"))
	if err != nil || reportPins[root] == "" || digest(raw) != reportPins[root] {
		return errors.New("fixed native pilot report required")
	}
	var report struct {
		Observations []struct {
			ID      string `json:"id"`
			Arm     string `json:"arm"`
			Unfixed bool   `json:"unfixed"`
			Calls   int    `json:"model_predictions"`
			Passed  int    `json:"finite_passed"`
			Cases   int    `json:"finite_cases"`
			Capture string `json:"capture_sha256"`
			Metrics struct {
				Wall   int64   `json:"wall_ns"`
				User   int64   `json:"user_cpu_ns"`
				System int64   `json:"system_cpu_ns"`
				RSS    int64   `json:"child_max_rss_bytes"`
				CPU    float64 `json:"process_cpu_percent_of_one_core_over_wall"`
			} `json:"process_metrics"`
		} `json:"observations"`
	}
	if json.Unmarshal(raw, &report) != nil || len(report.Observations) != 48 {
		return errors.New("48 retained native observations required")
	}
	groups := map[string]map[string][]float64{}
	counts := map[string]map[string]int{}
	for _, o := range report.Observations {
		mode := "legacy"
		if o.Unfixed {
			mode = "opt_in"
		}
		key := o.Arm + "/" + mode
		if groups[key] == nil {
			groups[key] = map[string][]float64{}
			counts[key] = map[string]int{}
		}
		capture, err := os.ReadFile(filepath.Join(root, "captures", o.ID+".json"))
		if err != nil || digest(capture) != o.Capture {
			return errors.New("native capture changed")
		}
		var value struct {
			Report struct {
				Paths struct {
					Timing struct {
						Load   float64 `json:"model_load_ms"`
						Search float64 `json:"bounded_search_ms"`
						Total  float64 `json:"total_ms"`
					} `json:"timing"`
				} `json:"body_paths"`
			} `json:"report"`
		}
		if json.Unmarshal(capture, &value) != nil {
			return errors.New("native timing decode failed")
		}
		for name, v := range map[string]float64{"whole_child_wall_ms": float64(o.Metrics.Wall) / 1e6,
			"whole_child_cpu_ms": float64(o.Metrics.User+o.Metrics.System) / 1e6, "whole_child_rss_mib": float64(o.Metrics.RSS) / (1 << 20),
			"whole_child_cpu_percent_one_core": o.Metrics.CPU, "compiler_total_ms": value.Report.Paths.Timing.Total,
			"model_load_ms": value.Report.Paths.Timing.Load, "bounded_search_ms": value.Report.Paths.Timing.Search} {
			groups[key][name] = append(groups[key][name], v)
		}
		counts[key]["observations"]++
		counts[key]["model_predictions"] += o.Calls
		counts[key]["finite_passed"] += o.Passed
		counts[key]["finite_cases"] += o.Cases
	}
	rows := map[string]any{}
	for key, metrics := range groups {
		if counts[key]["observations"] != 6 {
			return errors.New("six observations per model/mode required")
		}
		stats := map[string]any{}
		for name, values := range metrics {
			stats[name] = distribution(values)
		}
		rows[key] = map[string]any{"counts": counts[key], "descriptive_distributions": stats}
	}
	result := map[string]any{"schema": "gooo/native-unfixed-descriptive-metrics/v1", "report_sha256": digest(raw), "groups": rows,
		"scope": "Six reused observations per model/mode, 24 legacy-first pairs, all outliers retained. p50 is midpoint median; p95 is nearest-rank ceiling(0.95*n). Child CPU percent uses (user+system)/wall on one-core basis; this is not host CPU growth. Child wall includes process launch/model load/native codegen; native search/load clocks have narrower scopes. No causal speedup or language accuracy claim. Offline arithmetic makes zero predictions, native calls, Go execution processes or training steps."}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(encoded, '\n'), 0644)
}
func main() {
	root := flag.String("root", "runs/unfixed-native-feature-pilot-20261001", "fixed native pilot directory")
	output := flag.String("output", "", "metrics output")
	flag.Parse()
	if err := run(*root, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
