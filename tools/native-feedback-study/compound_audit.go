package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var compoundPins = map[string]struct {
	Preexecution string
	Pilot        bool
}{
	"dec30461109eafcea587e7d78a533087df21879fb23bb7edc68c5645aefca66e": {"b449327381128312c8632351b1fc400321bba61a5e17cc0a01b5de69eeee18ec", false},
	"11fa52d7fbc6c1bbd04eeacffd7ea82db440fd552ca03d0836de9b03cab44525": {"9f99827899ef5ff098f4597d6366d109108a281abc657e5b01960acc6b1e0239", true},
}

type compoundPair struct {
	Case            string `json:"case_id"`
	Arm             string `json:"arm"`
	SameGo          bool   `json:"same_emitted_go"`
	SameFinite      bool   `json:"same_finite_pass_count"`
	SameSeparate    bool   `json:"same_separate_input_pass_count"`
	SameSequence    bool   `json:"same_candidate_sequence"`
	AttemptDelta    int    `json:"feedback_minus_baseline_attempts"`
	PredictionDelta int    `json:"feedback_minus_baseline_predictions"`
}
type compoundArmSummary struct {
	Arm              string  `json:"arm"`
	Feedback         bool    `json:"feedback_enabled"`
	Views            int     `json:"repeated_views"`
	Predictions      int     `json:"actual_model_predictions"`
	FeedbackCalls    int     `json:"feedback_predictions"`
	FixedPredictions int     `json:"fixed_coordinate_feedback_predictions"`
	Attempts         int     `json:"candidate_attempts"`
	FinitePassed     int     `json:"finite_passed"`
	FiniteCases      int     `json:"finite_cases"`
	SeparatePassed   int     `json:"separate_passed"`
	SeparateCases    int     `json:"separate_cases"`
	IntentionMatches int     `json:"original_intention_mask_matches"`
	MedianWall       float64 `json:"median_native_wall_ms"`
	MedianCPU        float64 `json:"median_native_process_cpu_ms"`
	MedianRSS        float64 `json:"median_native_peak_rss_bytes"`
	MedianCPURatio   float64 `json:"median_native_process_cpu_percent_one_core"`
}

func compoundMedian(v []float64) float64 {
	sort.Float64s(v)
	if len(v)%2 == 1 {
		return v[len(v)/2]
	}
	return (v[len(v)/2-1] + v[len(v)/2]) / 2
}
func compoundArms() ([]familyArm, error) {
	arms := []familyArm{{Name: "offline"}}
	for _, c := range []struct{ Name, Root, Variant, Pin string }{
		{"parent_fp32", "runs/typed-path-positioned-random-20261001/models", "fp32", modelPins["fp32"]},
		{"feedback_fp32", "runs/feedback-path-soft-target-mps-20261001/models", "fp32", trainedPins["fp32"]},
		{"feedback_ptq", "runs/feedback-path-soft-target-mps-20261001/models", "ptq_ternary", trainedPins["ptq_ternary"]},
		{"feedback_qat", "runs/feedback-path-soft-target-mps-20261001/models", "qat_ternary", trainedPins["qat_ternary"]},
	} {
		path := filepath.Join(c.Root, c.Variant, "model.json")
		model, err := decision.LoadPath(path)
		if err != nil || model.MetadataSHA256() != c.Pin {
			return nil, errors.New("frozen compound model identity differs")
		}
		for _, feedback := range []bool{false, true} {
			arms = append(arms, familyArm{Name: c.Name, Variant: c.Variant, Path: path, Metadata: c.Pin, Weights: model.WeightsSHA256(), Feedback: feedback})
		}
	}
	return arms, nil
}
func auditCompound(dir string) (map[string]any, error) {
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil {
		return nil, err
	}
	reportSHA := hash(raw)
	pin, ok := compoundPins[reportSHA]
	if !ok {
		return nil, errors.New("frozen compound report required")
	}
	pre, err := read(filepath.Join(dir, "preexecution.json"))
	if err != nil || hash(pre) != pin.Preexecution {
		return nil, errors.New("frozen compound preexecution differs")
	}
	var report struct {
		Schema           string                `json:"schema"`
		Decision         string                `json:"decision"`
		Runner           string                `json:"runner_revision"`
		Native           string                `json:"native_revision"`
		Pilot            bool                  `json:"pilot"`
		Observations     []compoundObservation `json:"observations"`
		Calls            int                   `json:"native_calls"`
		Predictions      int                   `json:"actual_model_predictions"`
		Feedback         int                   `json:"feedback_predictions"`
		Attempts         int                   `json:"candidate_attempts"`
		Passed           int                   `json:"repeated_finite_passes"`
		Cases            int                   `json:"repeated_finite_cases"`
		SeparatePassed   int                   `json:"repeated_separate_input_passes"`
		SeparateCases    int                   `json:"repeated_separate_input_cases"`
		IntentionMatches int                   `json:"original_intention_mask_matches"`
		NoChoice         int                   `json:"zero_call_ranking_unnecessary_receipts"`
		Changed          int                   `json:"feedback_judgments_different_from_initial_proposal"`
		GoProcesses      int                   `json:"actual_generated_go_processes"`
		GoEvaluations    int                   `json:"actual_generated_function_evaluations"`
		Inputs           []int64               `json:"execution_inputs"`
	}
	if json.Unmarshal(raw, &report) != nil || report.Schema != "gooo/compound-path-study/v1" || report.Decision != "PASS" || report.Runner != "0127ed66e5fae44d134e07593bec683246285979" || report.Native != compoundNative || report.Pilot != pin.Pilot {
		return nil, errors.New("compound source report tuple differs")
	}
	rows, err := compoundRows()
	if err != nil {
		return nil, err
	}
	arms, err := compoundArms()
	if err != nil {
		return nil, err
	}
	if pin.Pilot {
		subset := rows[:0:0]
		for _, r := range rows {
			if r.IntendedMask == 0 && r.Language == "en" && r.Contract == "complete" {
				subset = append(subset, r)
			}
		}
		rows = subset
		arms = arms[:1]
	}
	if len(report.Observations) != len(rows)*len(arms) || report.Calls != len(report.Observations) {
		return nil, errors.New("compound native call inventory differs")
	}
	byCase := map[string]compoundstudy.Case{}
	inputsSet := map[int64]bool{}
	for _, r := range rows {
		byCase[r.ID] = r
		for _, c := range r.Document.Cases {
			inputsSet[c.Input] = true
		}
		for _, c := range r.Separate {
			inputsSet[c.Input] = true
		}
	}
	var inputs []int64
	for x := range inputsSet {
		inputs = append(inputs, x)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i] < inputs[j] })
	if !reflect.DeepEqual(inputs, report.Inputs) {
		return nil, errors.New("execution input union differs")
	}
	byObservation := map[string]compoundObservation{}
	sequences := map[string][]uint16{}
	byArm := map[string][]compoundObservation{}
	goMasks := map[string]compoundObservation{}
	fixedByID := map[string]int{}
	fixedPredictions := 0
	predictions, feedback, attempts, passed, cases, separatePassed, separateCases, noChoice, changed, intentions := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	for i, r := range rows {
		for j, a := range arms {
			o := report.Observations[i*len(arms)+j]
			id := fmt.Sprintf("%s-%s-%t", r.ID, a.Name, a.Feedback)
			if o.ID != id || byObservation[id].ID != "" {
				return nil, errors.New("compound observation inventory/order differs")
			}
			capture, err := read(filepath.Join(dir, "captures", id+".json"))
			if err != nil || hash(capture) != o.Capture {
				return nil, errors.New("compound capture bytes differ")
			}
			var v nativeResult
			if json.Unmarshal(capture, &v) != nil {
				return nil, errors.New("compound native result malformed")
			}
			expected, err := inspectCompound(v, r, a)
			if err != nil {
				return nil, err
			}
			expected.ID, expected.Capture, expected.Metrics = o.ID, o.Capture, o.Metrics
			if !reflect.DeepEqual(o, expected) {
				return nil, errors.New("compound captured observation differs")
			}
			m := o.Metrics
			if m.Wall <= 0 || m.RSS <= 0 || m.User < 0 || m.System < 0 || m.CPU != 100*float64(m.User+m.System)/float64(m.Wall) {
				return nil, errors.New("native process metrics differ")
			}
			for _, attempt := range v.Report.Paths.Search.Attempts {
				sequences[id] = append(sequences[id], attempt.Mask)
			}
			byObservation[id] = o
			fixedByID[id] = fixedCompoundFeedback(v.Report.Paths.Search, v.Report.Paths.Feedback)
			fixedPredictions += fixedByID[id]
			goMasks[o.GoSHA] = o
			key := fmt.Sprintf("%s-%t", a.Name, a.Feedback)
			byArm[key] = append(byArm[key], o)
			predictions += o.Predictions
			feedback += o.FeedbackPredictions
			attempts += o.Attempts
			passed += o.Passed
			cases += o.Cases
			separatePassed += o.SeparatePassed
			separateCases += o.SeparateCases
			noChoice += o.NoChoice
			changed += o.ChangedJudgments
			if o.IntentAgreement {
				intentions++
			}
		}
	}
	if predictions != report.Predictions || feedback != report.Feedback || attempts != report.Attempts || passed != report.Passed || cases != report.Cases || separatePassed != report.SeparatePassed || separateCases != report.SeparateCases || noChoice != report.NoChoice || changed != report.Changed || intentions != report.IntentionMatches {
		return nil, errors.New("compound aggregate accounting differs")
	}
	entries, err := os.ReadDir(filepath.Join(dir, "executions"))
	if err != nil || len(entries) != len(goMasks) || report.GoProcesses != len(goMasks) || report.GoEvaluations != len(goMasks)*len(inputs) {
		return nil, errors.New("deduplicated Go execution inventory differs")
	}
	executionHashes := map[string]string{}
	actualGo := map[string][]int64{}
	for sha, o := range goMasks {
		data, err := read(filepath.Join(dir, "executions", sha+".json"))
		if err != nil {
			return nil, err
		}
		var values []int64
		if json.Unmarshal(data, &values) != nil || len(values) != len(inputs) {
			return nil, errors.New("actual Go execution shape differs")
		}
		r := byCase[o.CaseID]
		for i, x := range inputs {
			expected, _ := compoundstudy.Oracle(r.Template, o.Selected, x)
			if values[i] != expected {
				return nil, errors.New("actual Go value differs from arithmetic oracle")
			}
		}
		executionHashes[sha] = hash(data)
		actualGo[sha] = values
	}
	for _, o := range report.Observations {
		r := byCase[o.CaseID]
		for i, x := range inputs {
			expected, _ := compoundstudy.Oracle(r.Template, o.Selected, x)
			if actualGo[o.GoSHA][i] != expected {
				return nil, errors.New("reused actual Go value differs from policy observation oracle")
			}
		}
	}
	var pairs []compoundPair
	if !pin.Pilot {
		for _, r := range rows {
			for _, a := range arms {
				if a.Path == "" || a.Feedback {
					continue
				}
				id := r.ID + "-" + a.Name
				base, updated := byObservation[id+"-false"], byObservation[id+"-true"]
				pairs = append(pairs, compoundPair{r.ID, a.Name, base.GoSHA == updated.GoSHA, base.Passed == updated.Passed, base.SeparatePassed == updated.SeparatePassed, reflect.DeepEqual(sequences[base.ID], sequences[updated.ID]), updated.Attempts - base.Attempts, updated.Predictions - base.Predictions})
			}
		}
	}
	var summaries []compoundArmSummary
	for _, a := range arms {
		rows := byArm[fmt.Sprintf("%s-%t", a.Name, a.Feedback)]
		s := compoundArmSummary{Arm: a.Name, Feedback: a.Feedback, Views: len(rows)}
		var wall, cpu, rss, ratio []float64
		for _, o := range rows {
			s.Predictions += o.Predictions
			s.FixedPredictions += fixedByID[o.ID]
			s.FeedbackCalls += o.FeedbackPredictions
			s.Attempts += o.Attempts
			s.FinitePassed += o.Passed
			s.FiniteCases += o.Cases
			s.SeparatePassed += o.SeparatePassed
			s.SeparateCases += o.SeparateCases
			if o.IntentAgreement {
				s.IntentionMatches++
			}
			wall = append(wall, float64(o.Metrics.Wall)/1e6)
			cpu = append(cpu, float64(o.Metrics.User+o.Metrics.System)/1e6)
			rss = append(rss, float64(o.Metrics.RSS))
			ratio = append(ratio, o.Metrics.CPU)
		}
		s.MedianWall = compoundMedian(wall)
		s.MedianCPU = compoundMedian(cpu)
		s.MedianRSS = compoundMedian(rss)
		s.MedianCPURatio = compoundMedian(ratio)
		summaries = append(summaries, s)
	}
	return map[string]any{"schema": "gooo/compound-path-audit/v1", "decision": "PASS", "report_sha256": reportSHA, "preexecution_sha256": hash(pre), "execution_sha256": executionHashes, "recorded_native_calls": report.Calls, "recorded_model_predictions": predictions, "recorded_feedback_predictions": feedback, "fixed_coordinate_feedback_predictions": fixedPredictions, "reinterpreted_candidates": attempts, "recorded_go_processes": report.GoProcesses, "recorded_go_function_evaluations": report.GoEvaluations, "reused_actual_go_value_observations": report.Calls*len(inputs) - report.GoEvaluations, "same_source_function_and_selected_typed_body": report.Calls, "paired_policy_observations": pairs, "arm_summaries": summaries, "new_model_predictions": 0, "new_native_calls": 0, "new_go_processes": 0, "scope": "Independent reconciliation of frozen native source/model/progress/failure links, selected typed function AST modulo in-range int64 literal wrappers, and captured actual-Go values for every policy observation. The audit executes no inference or subprocess. Fixed-coordinate predictions identify a ranking cost opportunity, not a verified skipped-call optimization. Three templates/one numeric configuration are development evidence. Structural intention agreement and finite/separate functional observations have separate denominators. Fixed arm order and whole compiler process resources do not prove causal speedup or host utilization increase."}, nil
}

func fixedCompoundFeedback(search pathplan.SearchResult, receipts []pathplan.FeedbackReceipt) int {
	fixed := 0
	for _, receipt := range receipts {
		if receipt.ModelCalls == 0 {
			continue
		}
		var tried [4]bool
		for _, attempt := range search.Attempts[:receipt.Attempted] {
			tried[attempt.Mask] = true
		}
		for bit := range 2 {
			var values uint8
			for mask := range 4 {
				if !tried[mask] {
					values |= 1 << (mask >> bit & 1)
				}
			}
			if values == 1 || values == 2 {
				fixed++
			}
		}
	}
	return fixed
}
