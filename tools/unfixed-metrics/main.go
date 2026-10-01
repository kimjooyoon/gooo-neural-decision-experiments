// unfixed-metrics derives descriptive paired SDK costs from frozen captures.
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
	"strings"
)

type observation struct {
	ID       string  `json:"id"`
	Unfixed  bool    `json:"unfixed"`
	Calls    int     `json:"model_calls"`
	Attempts int     `json:"candidate_attempts"`
	Skipped  int     `json:"fixed_coordinate_receipts"`
	Wall     float64 `json:"wall_ns"`
}

func run(output string) error {
	raw, err := os.ReadFile("runs/unfixed-feedback-sdk-20261001/report.json")
	h := sha256.Sum256(raw)
	if err != nil || hex.EncodeToString(h[:]) != "41b0e37c2ba99fa44d2173a25e4c2d7174e21115b9dff0a29d3c512664bf839e" {
		return errors.New("frozen unfixed SDK report differs")
	}
	var report struct {
		Observations []observation `json:"observations"`
	}
	if json.Unmarshal(raw, &report) != nil || len(report.Observations) != 576 {
		return errors.New("576 repeated SDK observations required")
	}
	var summaries []map[string]any
	for _, name := range []string{"parent_fp32", "feedback_fp32", "feedback_ptq", "feedback_qat"} {
		for _, unfixed := range []bool{false, true} {
			var walls []float64
			calls, attempts, skipped := 0, 0, 0
			for _, o := range report.Observations {
				if o.Unfixed != unfixed || !strings.Contains(o.ID, "-"+name+"-") {
					continue
				}
				walls = append(walls, o.Wall/1e6)
				calls += o.Calls
				attempts += o.Attempts
				skipped += o.Skipped
			}
			if len(walls) != 72 {
				return errors.New("arm denominator differs")
			}
			sort.Float64s(walls)
			summaries = append(summaries, map[string]any{"model": name, "unfixed": unfixed,
				"repeated_views": len(walls), "predictions": calls, "candidate_attempts": attempts,
				"skipped_predictions": skipped, "prepared_session_p50_ms": (walls[35] + walls[36]) / 2,
				"prepared_session_p95_ms": walls[int(math.Ceil(.95*float64(len(walls))))-1]})
		}
	}
	value := map[string]any{"schema": "gooo/unfixed-feedback-descriptive-metrics/v1", "decision": "PASS",
		"source_report_sha256": hex.EncodeToString(h[:]), "arm_summaries": summaries,
		"total_prediction_reduction_percent": 100 * 97.0 / 1014, "feedback_prediction_reduction_percent": 100 * 97.0 / 438,
		"new_model_predictions": 0, "new_native_calls": 0, "new_go_processes": 0,
		"scope": "Offline arithmetic over 576 frozen SDK session observations. Prepared-session times include two initial predictions, bounded TDD, optional feedback, receipts and SDK Go rendering; they exclude model load, plan preparation, native processes and Go execution. Fixed legacy-first order, reused development data, all outliers retained, nearest-rank p95 and midpoint median; no causal timing improvement, host CPU utilization or untouched language accuracy claim."}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(data, '\n'), 0644)
}

func main() {
	output := flag.String("output", "", "descriptive metric output")
	flag.Parse()
	if *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "output required")
		os.Exit(1)
	}
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
