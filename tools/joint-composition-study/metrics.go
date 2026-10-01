package main

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
)

type distribution struct {
	Mean    float64 `json:"mean"`
	Median  float64 `json:"median"`
	P95     float64 `json:"p95_nearest_rank"`
	Maximum float64 `json:"maximum"`
}

func describe(values []float64) (distribution, error) {
	if len(values) == 0 {
		return distribution{}, errors.New("observed values required")
	}
	v := append([]float64(nil), values...)
	sort.Float64s(v)
	var d distribution
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 {
			return d, errors.New("finite nonnegative observations required")
		}
		d.Mean += x
	}
	d.Mean /= float64(len(v))
	d.Median = v[len(v)/2]
	if len(v)%2 == 0 {
		d.Median = (v[len(v)/2-1] + v[len(v)/2]) / 2
	}
	d.P95 = v[int(math.Ceil(.95*float64(len(v))))-1]
	d.Maximum = v[len(v)-1]
	return d, nil
}
func summarizeNativeMetrics(native, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh metric output required")
	}
	var audit struct {
		Status      string `json:"status"`
		Native      string `json:"native_revision"`
		Calls       int    `json:"native_captures"`
		Predictions int    `json:"recorded_native_model_predictions"`
	}
	if err := decodeFile(filepath.Join(native, "independent-audit.json"), &audit); err != nil {
		return err
	}
	if audit.Status != "PASS" || audit.Native != nativeDeployed || audit.Calls != 192 || audit.Predictions != 470 {
		return errors.New("audited native captures required")
	}
	var report struct {
		Status  string               `json:"status"`
		Totals  map[string]counts    `json:"totals"`
		Metrics map[string][]metrics `json:"native_process_measurements"`
	}
	if err := decodeFile(filepath.Join(native, "report.json"), &report); err != nil {
		return err
	}
	if report.Status != "PASS" || len(report.Metrics) != 4 || len(report.Totals) != 4 {
		return errors.New("four native policies required")
	}
	result := map[string]map[string]any{}
	for _, policy := range []string{"selected", "independent", "joint", "offline"} {
		rows := report.Metrics[policy]
		c := report.Totals[policy]
		if len(rows) != 48 || c.Views != 48 || c.FinalComplete != 48 {
			return errors.New("native metric denominator differs")
		}
		var wall, cpu, rss []float64
		for _, r := range rows {
			if r.Wall <= 0 || r.RSS <= 0 {
				return errors.New("positive process observations required")
			}
			wall = append(wall, float64(r.Wall)/1e6)
			cpu = append(cpu, r.CPU)
			rss = append(rss, float64(r.RSS))
		}
		w, err := describe(wall)
		if err != nil {
			return err
		}
		p, err := describe(cpu)
		if err != nil {
			return err
		}
		m, err := describe(rss)
		if err != nil {
			return err
		}
		result[policy] = map[string]any{"native_calls": 48, "compiler_child_wall_ms": w, "process_cpu_percent_one_core_normalization": p, "compiler_child_max_rss_bytes": m, "recorded_model_predictions": c.Calls, "additional_candidate_attempts": c.Extras, "finite_complete_functions": c.FinalComplete}
	}
	sha, err := fileHash(filepath.Join(native, "report.json"))
	if err != nil {
		return err
	}
	return save(output, map[string]any{"schema": "gooo/joint-composition-native-process-metrics/v1", "status": "PASS", "native_revision": nativeDeployed, "native_report_sha256": sha, "policies": result, "actual_native_calls": 192, "recorded_model_predictions": 470, "host_cpu_utilization_delta_measured": false, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_updates": 0, "scope": "Descriptive fixed-order measurements of 48 compiler children per policy. CPU is user+system time divided by child wall time, normalized to one core, not global host utilization or a causal delta. RSS is the observed child process accounting peak, not model-only RAM. Emitted Go compilation/execution is separately verified; its time/RSS is excluded here. Selected/reference repeat the same model."})
}
