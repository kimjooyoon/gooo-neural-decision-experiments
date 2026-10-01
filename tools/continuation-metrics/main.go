// continuation-metrics summarizes recorded partial batches without inference.
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

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type point struct {
	Attempts    int     `json:"attempted_candidates"`
	Passed      int     `json:"finite_passed"`
	Cases       int     `json:"finite_cases"`
	Percent     float64 `json:"finite_completeness_percent"`
	Predictions int     `json:"cumulative_model_predictions_before_batch_completed"`
}
type curve struct {
	Points    []point `json:"batch_observations"`
	FirstBest int     `json:"first_observed_final_best_batch_budget"`
	Mean      float64 `json:"mean_observed_batch_completeness_percent"`
}

func summarize(progress []pathplan.SessionProgress) (curve, error) {
	var c curve
	prior, passed := 0, 0
	for _, p := range progress {
		if p.Attempted < prior || p.SelectedPassed < passed || p.Cases <= 0 || p.SelectedPassed > p.Cases {
			return c, errors.New("nonmonotonic or invalid partial observation")
		}
		if p.Attempted == prior {
			continue
		} // Rejudgment alone did not inspect a candidate.
		c.Points = append(c.Points, point{p.Attempted, p.SelectedPassed, p.Cases, 100 * float64(p.SelectedPassed) / float64(p.Cases), p.Selection.ModelCalls})
		prior, passed = p.Attempted, p.SelectedPassed
	}
	if len(c.Points) == 0 {
		return c, errors.New("candidate batch observations required")
	}
	best := c.Points[len(c.Points)-1].Passed
	for _, p := range c.Points {
		c.Mean += p.Percent / float64(len(c.Points))
		if c.FirstBest == 0 && p.Passed == best {
			c.FirstBest = p.Attempts
		}
	}
	return c, nil
}
func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func read(path string) ([]byte, error) {
	i, err := os.Lstat(path)
	if err != nil || !i.Mode().IsRegular() || i.Size() <= 0 || i.Size() > 2<<20 {
		return nil, errors.New("bounded regular capture required")
	}
	return os.ReadFile(path)
}
func run(output string) error {
	rows := []map[string]any{}
	for _, edition := range []struct{ Dir, Pin string }{
		{"runs/feedback-trained-native-20261001", "cda64b53a48ff769a21319084a23239c26667e95dda449c89fa29f16a859303f"},
		{"runs/feedback-trained-main-smokes-20261001", "fbe9c4b853af382dd45b5bb375fd31cebe27101951f5dbce77cacfdcf765a8d2"},
	} {
		raw, err := read(filepath.Join(edition.Dir, "report.json"))
		if err != nil || hash(raw) != edition.Pin {
			return errors.New("fixed source edition required")
		}
		var report struct {
			Native       string `json:"native_revision"`
			Observations []struct {
				Language, Variant string
				Capture           string `json:"capture_sha256"`
			} `json:"observations"`
		}
		if err = json.Unmarshal(raw, &report); err != nil || len(report.Observations) != 6 {
			return errors.New("fixed six-view report required")
		}
		for _, o := range report.Observations {
			data, err := read(filepath.Join(edition.Dir, o.Language+"-"+o.Variant+".json"))
			if err != nil || hash(data) != o.Capture {
				return errors.New("captured progression digest differs")
			}
			var capture struct {
				Report struct {
					Paths struct {
						Progress []pathplan.SessionProgress `json:"session_progress"`
					} `json:"body_paths"`
				} `json:"report"`
			}
			if err = json.Unmarshal(data, &capture); err != nil {
				return err
			}
			c, err := summarize(capture.Report.Paths.Progress)
			if err != nil {
				return err
			}
			rows = append(rows, map[string]any{"source_edition": edition.Dir, "native_revision": report.Native, "language": o.Language, "variant": o.Variant, "report_sha256": edition.Pin, "capture_sha256": o.Capture, "curve": c})
		}
	}
	value := map[string]any{"schema": "gooo/observed-batch-completeness-curves/v1", "decision": "PASS", "observations": rows, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_steps": 0,
		"scope": "Read twelve frozen source/policy/language views of one existing compound intention. Each point is the best finite result after an observed eight-candidate batch; duplicate rejudgment-only points are excluded from the mean. First observed final best is a batch boundary, not the exact first successful candidate. Mean is eight sampled endpoints, not continuous area or general language accuracy. It does not authorize early stopping or assume no better unattempted path exists."}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(raw, '\n'), 0644)
}
func main() {
	output := flag.String("output", "", "metrics receipt")
	flag.Parse()
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
