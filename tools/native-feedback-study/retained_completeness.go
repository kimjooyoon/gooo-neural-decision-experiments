package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
)

type completionCurve struct {
	Arm               string    `json:"arm"`
	Mode              string    `json:"mode"`
	Index             int       `json:"request_index"`
	Case              string    `json:"case_id"`
	DeclaredCases     int       `json:"declared_cases"`
	LegalPaths        int       `json:"declared_legal_paths"`
	MaximumPassed     int       `json:"finite_legal_space_maximum_passed"`
	Maximizers        []uint16  `json:"finite_maximizing_masks"`
	AttainablePercent float64   `json:"finite_legal_space_attainment_percent"`
	GapPercent        float64   `json:"irreducible_declared_case_gap_percent"`
	Curve             []float64 `json:"best_declared_case_completion_after_each_committed_attempt"`
	AreaPercent       float64   `json:"mean_committed_attempt_completion_percent"`
}

// Exhaustive maximum is independently recomputed over this four-path finite
// space. It is not an impossibility proof for other paths or all int64 inputs.
func finiteLegalMaximum(row compoundstudy.Case) (int, []uint16, error) {
	best := -1
	var masks []uint16
	for mask := uint16(0); mask < 4; mask++ {
		passed := 0
		for _, test := range row.Document.Cases {
			value, err := compoundstudy.Oracle(row.Template, mask, test.Input)
			if err != nil {
				return 0, nil, err
			}
			if value == test.Expected {
				passed++
			}
		}
		if passed > best {
			best = passed
			masks = nil
		}
		if passed == best {
			masks = append(masks, mask)
		}
	}
	return best, masks, nil
}

func retainedCompleteness(dir string) (map[string]any, error) {
	if _, err := retainedMetrics(dir); err != nil {
		return nil, err
	}
	c, err := collectRetained(dir, retainedFeatureRunner, retainedFeatureNative)
	if err != nil {
		return nil, err
	}
	rows, err := retainedRows()
	if err != nil {
		return nil, err
	}
	var curves []completionCurve
	for _, o := range c.observations {
		row := rows[o.Index]
		v := c.values[fmt.Sprintf("%s-%s-%02d", o.Arm, o.Mode, o.Index)]
		best, maximizers, err := finiteLegalMaximum(row)
		if err != nil {
			return nil, err
		}
		p := v.Report.Paths
		if p.Search.DeclaredCombinations != 4 || p.Search.Unattempted != 0 ||
			p.Search.Evaluated != 4 || p.Search.TypeRejected != 0 || best != row.FiniteBestPassed || best <= 0 {
			return nil, errors.New("exhaustive finite space not established")
		}
		curve := completionCurve{Arm: o.Arm, Mode: o.Mode, Index: o.Index, Case: o.Case, DeclaredCases: o.Cases, LegalPaths: 4,
			MaximumPassed: best, Maximizers: maximizers, AttainablePercent: 100 * float64(o.Passed) / float64(best),
			GapPercent: 100 * float64(o.Cases-best) / float64(o.Cases)}
		previous := 0
		committed := 0
		for _, progress := range p.Progress {
			if len(progress.NewAttempts) == 0 && progress.Attempted == committed && progress.SelectedPassed == previous {
				continue
			}
			if progress.Attempted != committed+1 || len(progress.NewAttempts) != 1 || progress.SelectedPassed < previous || progress.SelectedPassed > best {
				return nil, fmt.Errorf("one-committed-attempt curve differs: %s %s %d sequence %d", o.Arm, o.Mode, o.Index, progress.Sequence)
			}
			previous = progress.SelectedPassed
			committed++
			point := 100 * float64(progress.SelectedPassed) / float64(o.Cases)
			curve.Curve = append(curve.Curve, point)
			curve.AreaPercent += point / 4
		}
		if committed != 4 || previous != o.Passed {
			return nil, errors.New("curve terminal differs")
		}
		curves = append(curves, curve)
	}
	return map[string]any{"schema": "gooo/finite-construction-completeness/v1", "source_report_sha256": retainedFeatureReport,
		"observations": curves, "observation_count": len(curves), "distinct_structural_templates": 3, "bilingual_views": 6,
		"new_predictions": 0, "new_native_processes": 0, "new_go_processes": 0, "new_training_steps": 0,
		"metric_definitions": map[string]string{
			"declared_completion":   "selected finite passed / declared cases; current report 7/8 = 87.5%",
			"attainable_completion": "selected finite passed / independently enumerated legal-path maximum; 7/7 = 100% does not mean all declared cases pass",
			"irreducible_gap":       "(declared cases - maximum finite passed) / declared cases; restricted to the four declared legal paths",
			"progress_area":         "arithmetic mean of best declared-case completion after each of four committed attempts; equal attempt weight, not time weight or first-shot acceptance"},
		"scope": "Offline derivation over original feature captures, unchanged weights and original UNKNOWN CI hints. Complete enumeration of four declared typed paths and eight finite cases only. An irreducible finite gap cannot establish all-input or natural-language impossibility. Structural intent ambiguity remains distinct from maximizing finite tests. Repeated model/mode/language views are not independent experiments. No raw receipt, prediction, body or case rewritten."}, nil
}

func saveRetainedCompleteness(dir, output string) error {
	value, err := retainedCompleteness(dir)
	if err != nil {
		return err
	}
	runAbs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	outputAbs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(runAbs, outputAbs)
	if err != nil {
		return err
	}
	if rel != "completeness-summary.json" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("original run evidence cannot be overwritten")
	}
	return save(output, value)
}
