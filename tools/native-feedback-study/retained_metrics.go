package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
)

const retainedFeatureRunner = "be3ccb633446ef4379af964007f8d25766d6726b"
const retainedFeatureNative = "faf932f85c6acc293ce3839a563665661cb53665"
const retainedFeatureReport = "581c09ebb1746505a636a44d8eb34cdeef333fd3204e6ad9120775ba1c5c7e9b"

// Costs are arithmetic over frozen observations. No clocks are rerun.
func retainedMetrics(dir string) (map[string]any, error) {
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil || hash(raw) != retainedFeatureReport {
		return nil, errors.New("frozen feature report required")
	}
	audit, err := auditRetained(dir, retainedFeatureRunner, retainedFeatureNative)
	if err != nil {
		return nil, err
	}
	replay, err := json.MarshalIndent(audit, "", "  ")
	if err != nil || !bytes.Equal(append(replay, '\n'), raw) {
		return nil, errors.New("offline report differs")
	}
	var report struct {
		Observations []retainedObservation      `json:"observations"`
		Processes    map[string]retainedProcess `json:"processes"`
	}
	if json.Unmarshal(raw, &report) != nil {
		return nil, errors.New("report decode failed")
	}
	arms, err := retainedArms()
	if err != nil {
		return nil, err
	}
	var summaries []map[string]any
	for _, arm := range arms {
		for _, mode := range []string{"fresh", "retained-1", "retained-4"} {
			var response, body, load, rss []float64
			wall, cpu, processes := int64(0), int64(0), 0
			for _, o := range report.Observations {
				if o.Arm == arm.Name && o.Mode == mode {
					response = append(response, float64(o.Latency)/1e6)
					body = append(body, o.BodyMS)
					load = append(load, o.LoadMS)
				}
			}
			for id, p := range report.Processes {
				match := id == arm.Name+"-"+mode
				if mode == "fresh" {
					for i := range 12 {
						if id == arm.Name+"-fresh-"+twoDigits(i) {
							match = true
						}
					}
				}
				if match {
					wall += p.Metrics.Wall
					cpu += p.Metrics.User + p.Metrics.System
					processes++
					rss = append(rss, float64(p.Metrics.RSS)/(1<<20))
				}
			}
			if len(response) != 12 || processes == 0 {
				return nil, errors.New("metric denominator differs")
			}
			summaries = append(summaries, map[string]any{"arm": arm.Name, "mode": mode, "valid_constructions": 12, "native_processes": processes,
				"response_p50_ms": median(response), "native_body_p50_ms": median(body), "per_request_model_load_p50_ms": median(load),
				"whole_process_wall_ms": float64(wall) / 1e6, "whole_process_cpu_ms": float64(cpu) / 1e6,
				"whole_process_cpu_ms_per_valid_construction": float64(cpu) / 1e6 / 12, "valid_constructions_per_whole_process_second": 12e9 / float64(wall),
				"process_peak_rss_p50_mib": median(rss), "aggregate_child_cpu_percent_of_one_core": 100 * float64(cpu) / float64(wall)})
		}
	}
	return map[string]any{"schema": "gooo/retained-native-cost-summary/v1", "report_sha256": retainedFeatureReport, "summaries": summaries,
		"scope": "Frozen feature observations: six reused bilingual contradictory views repeated forward/reverse, not independent experiments. Fresh response includes startup/load; retained-1 response excludes startup/setup; retained-4 includes queueing. Whole worker wall/CPU includes one setup and one source rejection in addition to 12 valid requests. Fresh RSS median is over 12 processes; retained RSS is the peak of one process. Child CPU is not host utilization; parallel CPU can exceed 100% of one core. Controller costs excluded. Increased CPU, native body time and RSS remain reported. No causal or generalized speedup; zero new inference, native/Go calls or training during summary."}, nil
}

func twoDigits(i int) string {
	if i < 10 {
		return "0" + fmtInt(i)
	}
	return fmtInt(i)
}
