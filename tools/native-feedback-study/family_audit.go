package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/familystudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

var familyReports = map[string]bool{
	"58090950d759d47d7c0c42ca9e6edd1929c50864cceb5a4b9f37fbc33c00da00": true,
	"9c00cf48dae46bdc793b39c5ae94f15c825194d33eabe2bcf3002161aba87c2d": false,
}

func auditFamily(dir string) (map[string]any, error) {
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil {
		return nil, err
	}
	pilot, ok := familyReports[hash(raw)]
	if !ok {
		return nil, errors.New("frozen family report required")
	}
	var report struct {
		Pilot          bool                `json:"pilot"`
		Runner         string              `json:"runner_revision"`
		Native         string              `json:"native_revision"`
		Inputs         []int64             `json:"execution_inputs"`
		Observations   []familyObservation `json:"observations"`
		Calls          int                 `json:"native_calls"`
		Predictions    int                 `json:"actual_model_predictions"`
		Feedback       int                 `json:"feedback_predictions"`
		Attempts       int                 `json:"candidate_attempts"`
		Passed         int                 `json:"repeated_finite_passes"`
		Cases          int                 `json:"repeated_finite_cases"`
		SeparatePassed int                 `json:"repeated_separate_input_passes"`
		SeparateCases  int                 `json:"repeated_separate_input_cases"`
		GoProcesses    int                 `json:"actual_generated_go_processes"`
		GoEvaluations  int                 `json:"actual_generated_function_evaluations"`
	}
	if err = json.Unmarshal(raw, &report); err != nil || report.Pilot != pilot || report.Runner != "30fbf81b951e12d504aca19d06e8d1a8dc08a865" || report.Native != familyNativeSHA || len(report.Observations) != report.Calls {
		return nil, errors.New("family report tuple differs")
	}
	preRaw, err := read(filepath.Join(dir, "preexecution.json"))
	if err != nil {
		return nil, err
	}
	var pre struct {
		Runner  string `json:"runner_revision"`
		Native  string `json:"native_revision"`
		Cohort  string `json:"cohort_sha256"`
		Binary  string `json:"binary_sha256"`
		Go      string `json:"go_binary_sha256"`
		Planned int    `json:"planned_native_calls"`
	}
	if err = json.Unmarshal(preRaw, &pre); err != nil || pre.Runner != report.Runner || pre.Native != familyNativeSHA || pre.Cohort != familyCohortSHA || pre.Binary != "d6c5c191b334ec9736f13fa34674f6c4785479dcb686d95a0ab98772fb03d26d" || pre.Go != "132b69336a1f809932a8a20b0201dbbb980e86e3a323ae32e893639d83d71598" || pre.Planned != report.Calls {
		return nil, errors.New("family preexecution identities differ")
	}
	rows, err := familyRows()
	if err != nil {
		return nil, err
	}
	byID := map[string]familystudy.Case{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	arms := map[string]familyArm{"offline": {Name: "offline"}}
	for _, c := range []struct{ Name, Root, Variant, Pin string }{
		{"parent_fp32", "runs/typed-path-positioned-random-20261001/models", "fp32", modelPins["fp32"]},
		{"feedback_fp32", "runs/feedback-path-soft-target-mps-20261001/models", "fp32", trainedPins["fp32"]},
		{"feedback_ptq", "runs/feedback-path-soft-target-mps-20261001/models", "ptq_ternary", trainedPins["ptq_ternary"]},
		{"feedback_qat", "runs/feedback-path-soft-target-mps-20261001/models", "qat_ternary", trainedPins["qat_ternary"]},
	} {
		m, err := decision.LoadPath(filepath.Join(c.Root, c.Variant, "model.json"))
		if err != nil || m.MetadataSHA256() != c.Pin {
			return nil, errors.New("family audit model differs")
		}
		arms[c.Name] = familyArm{Name: c.Name, Variant: c.Variant, Path: c.Root, Metadata: c.Pin, Weights: m.WeightsSHA256()}
	}
	seen := map[string]bool{}
	executions := map[string][]int64{}
	executionHashes := map[string]string{}
	calls, feedback, attempts, passed, cases, separatePassed, separateCases := 0, 0, 0, 0, 0, 0, 0
	pairs := map[string]familyObservation{}
	samePairs := 0
	for _, o := range report.Observations {
		r, ok := byID[o.CaseID]
		if !ok || seen[o.ID] {
			return nil, errors.New("unknown or repeated family view")
		}
		seen[o.ID] = true
		a, ok := arms[o.Arm]
		if !ok || o.Arm == "offline" && o.Feedback {
			return nil, errors.New("unknown family policy")
		}
		a.Feedback = o.Feedback
		data, err := read(filepath.Join(dir, "captures", o.ID+".json"))
		if err != nil || hash(data) != o.Capture {
			return nil, errors.New("family capture hash differs")
		}
		var v nativeResult
		if err = json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		fresh, err := inspectFamily(v, r, a)
		if err != nil {
			return nil, err
		}
		fresh.ID, fresh.Capture, fresh.Metrics = o.ID, o.Capture, o.Metrics
		if !reflect.DeepEqual(fresh, o) {
			return nil, errors.New("family observation does not reconcile")
		}
		m := o.Metrics
		if m.Wall <= 0 || m.RSS <= 0 || m.User < 0 || m.System < 0 || m.CPU != 100*float64(m.User+m.System)/float64(m.Wall) {
			return nil, errors.New("family process counters differ")
		}
		actual, exists := executions[o.GoSHA]
		if !exists {
			data, err := read(filepath.Join(dir, "executions", o.GoSHA+".json"))
			if err != nil {
				return nil, err
			}
			if err = json.Unmarshal(data, &actual); err != nil || len(actual) != len(report.Inputs) {
				return nil, errors.New("family execution shape differs")
			}
			executions[o.GoSHA] = actual
			executionHashes[o.GoSHA] = hash(data)
		}
		for i, input := range report.Inputs {
			expected, err := pathstudy.Oracle(r.Family, o.Selected == pathstudy.GoldLabel(r.Family, true), r.Configuration, input)
			if err != nil || actual[i] != expected {
				return nil, errors.New("family actual Go differs from independent oracle")
			}
		}
		calls += o.Predictions
		feedback += o.FeedbackPredictions
		attempts += o.Attempts
		passed += o.Passed
		cases += o.Cases
		separatePassed += o.SeparatePassed
		separateCases += o.SeparateCases
		key := o.CaseID + "/" + o.Arm
		if o.Feedback {
			base, ok := pairs[key]
			if !ok || base.GoSHA != o.GoSHA || base.Passed != o.Passed || base.SeparatePassed != o.SeparatePassed || base.Attempts != o.Attempts {
				return nil, errors.New("family feedback pair outcome differs")
			}
			samePairs++
		} else {
			pairs[key] = o
		}
	}
	if calls != report.Predictions || feedback != report.Feedback || attempts != report.Attempts || passed != report.Passed || cases != report.Cases || separatePassed != report.SeparatePassed || separateCases != report.SeparateCases || len(executions) != report.GoProcesses || len(executions)*len(report.Inputs) != report.GoEvaluations || !pilot && (report.Calls != 1080 || samePairs != 480) || pilot && report.Calls != 10 {
		return nil, errors.New("family aggregate or pair count differs")
	}
	return map[string]any{"schema": "gooo/native-feedback-family-audit/v1", "decision": "PASS", "report_sha256": hash(raw), "preexecution_sha256": hash(preRaw), "recorded_native_calls": report.Calls, "recorded_model_predictions": calls, "recorded_feedback_predictions": feedback, "reinterpreted_candidates": attempts, "same_feedback_pairs": samePairs, "recorded_go_processes": report.GoProcesses, "recorded_go_function_evaluations": report.GoEvaluations, "execution_sha256": executionHashes, "new_model_predictions": 0, "new_native_calls": 0, "new_go_processes": 0,
		"scope": "Reconcile frozen source/cohort/model/progress and counterexample bindings; reinterpret candidates and independently check captured actual-Go values. Recorded process/resource observations are not new executions. All two-option feedback pairs retain identical outcomes; a sole remaining option offers no ranking freedom."}, nil
}
