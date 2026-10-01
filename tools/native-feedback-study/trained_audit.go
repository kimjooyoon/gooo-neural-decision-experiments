package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var trainedReports = map[string]string{
	"cda64b53a48ff769a21319084a23239c26667e95dda449c89fa29f16a859303f": "6a3e45d7f185c74f5136ab81b560a14a1366f367",
	"fbe9c4b853af382dd45b5bb375fd31cebe27101951f5dbce77cacfdcf765a8d2": "4dced73b26dde567cb7129f3e4ba5733850196d0",
}

// auditTrainedDogfood reinterprets captured paths without making predictions or
// starting native/Go processes. Both frozen editions retain their own source.
func auditTrainedDogfood(dir string) (map[string]any, error) {
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil || trainedReports[hash(raw)] == "" {
		return nil, errors.New("frozen trained dogfood report required")
	}
	var report struct {
		Native       string  `json:"native_revision"`
		Inputs       []int64 `json:"execution_inputs"`
		Expected     []int64 `json:"execution_expected"`
		Observations []struct {
			Language, Variant string
			Capture           string  `json:"capture_sha256"`
			GoSHA             string  `json:"generated_go_sha256"`
			Metrics           metrics `json:"process_metrics"`
		} `json:"observations"`
	}
	if err = json.Unmarshal(raw, &report); err != nil || report.Native != trainedReports[hash(raw)] || len(report.Observations) != 6 || len(report.Inputs) != 16 || len(report.Expected) != 16 {
		return nil, errors.New("trained source or inventory differs")
	}
	preRaw, err := read(filepath.Join(dir, "preexecution.json"))
	if err != nil {
		return nil, err
	}
	var pre struct {
		Native    string            `json:"native_revision"`
		Runner    string            `json:"runner_revision"`
		Models    map[string]string `json:"models"`
		Weights   map[string]string `json:"weights"`
		Source    string            `json:"source_sha256"`
		CI        pathplan.CIHint   `json:"caller_ci_hint"`
		Authority bool              `json:"ci_hint_is_authority"`
	}
	if err = json.Unmarshal(preRaw, &pre); err != nil || pre.Native != report.Native || pre.Runner != "1cfeb4753fdcb62aa2b9cfad3002c1c0172b5328" || !reflect.DeepEqual(pre.Models, trainedPins) || pre.Authority || pre.CI.SourceSHA != "3b2c11b418479d4a5c4df845425678fd25b0a695" || pre.CI.Status != "PASS" {
		return nil, errors.New("trained execution preconditions differ")
	}
	source, err := read("studies/native-feedback-v1/examples/body-codegen/typed-path-conditional-assignment.gooo.fixture")
	if err != nil || hash(source) != pre.Source {
		return nil, errors.New("original source differs")
	}
	for variant, pin := range trainedPins {
		model, err := decision.LoadPath(filepath.Join("runs/feedback-path-soft-target-mps-20261001/models", variant, "model.json"))
		if err != nil || model.MetadataSHA256() != pin || pre.Weights[variant] != model.WeightsSHA256() {
			return nil, errors.New("trained weights differ")
		}
	}
	seen, executions := map[string]bool{}, map[string]bool{}
	attempts := 0
	for _, o := range report.Observations {
		id := o.Language + "-" + o.Variant
		if seen[id] || (o.Language != "en" && o.Language != "ko") || trainedPins[o.Variant] == "" {
			return nil, errors.New("duplicate or unknown trained observation")
		}
		seen[id] = true
		name := "typed-path-conditional-assignment-plan.json"
		if o.Language == "ko" {
			name = "typed-path-conditional-assignment-ko-plan.json"
		}
		docRaw, err := read(filepath.Join("studies/native-feedback-v1/examples/body-codegen", name))
		if err != nil {
			return nil, err
		}
		var d document
		if err = json.Unmarshal(docRaw, &d); err != nil || len(d.Cases) != 7 {
			return nil, errors.New("original bilingual document differs")
		}
		for i, c := range d.Cases {
			if c.Input != report.Inputs[i] || c.Expected != report.Expected[i] {
				return nil, errors.New("original execution cases differ")
			}
		}
		d.Cases[6].Expected = 999
		capture, err := read(filepath.Join(dir, id+".json"))
		if err != nil || hash(capture) != o.Capture {
			return nil, errors.New("trained capture digest differs")
		}
		var v nativeResult
		if err = json.Unmarshal(capture, &v); err != nil {
			return nil, err
		}
		if err = inspect(v, true, 6, report.Native); err != nil {
			return nil, err
		}
		p := v.Report.Paths
		if hash([]byte(v.Source)) != o.GoSHA || p.Search.Selection.MetadataSHA256 != trainedPins[o.Variant] || p.Search.Selection.WeightsSHA256 != pre.Weights[o.Variant] || p.Search.Selection.ModelCalls != 18 || p.Search.Evaluated != 64 || len(p.Feedback) != 2 {
			return nil, errors.New("trained native accounting differs")
		}
		prepared, err := pathplan.Prepare(d.Plan)
		if err != nil || prepared.PlanSHA256() != p.Search.Selection.PlanSHA256 {
			return nil, errors.New("typed plan digest differs")
		}
		for _, a := range p.Search.Attempts {
			body, err := prepared.Compile(a.Choices)
			if err != nil || a.GoooSHA != hash([]byte(body.GoooSource())) || len(a.Results) != 7 {
				return nil, errors.New("trained candidate body differs")
			}
			passed := 0
			for i, c := range d.Cases {
				value, err := body.Evaluate(c.Input)
				if err != nil || a.Results[i].Input != c.Input || a.Results[i].Actual != value.Int || a.Results[i].Expected != c.Expected || a.Results[i].Passed != (value.Int == c.Expected) {
					return nil, errors.New("trained finite outcome differs")
				}
				if value.Int == c.Expected {
					passed++
				}
			}
			if a.Passed != passed {
				return nil, errors.New("trained candidate pass total differs")
			}
			attempts++
		}
		progresses := map[string]pathplan.SessionProgress{}
		var recorded []pathplan.SearchAttempt
		for _, progress := range p.Progress {
			sha := progress.SHA
			progress.SHA = ""
			encoded, err := json.Marshal(progress)
			if err != nil || hash(encoded) != sha || progress.Interrupted {
				return nil, errors.New("trained progress digest differs")
			}
			recorded = append(recorded, progress.NewAttempts...)
			progress.SHA = sha
			progresses[sha] = progress
		}
		if !reflect.DeepEqual(recorded, p.Search.Attempts) {
			return nil, errors.New("progress does not describe captured candidates")
		}
		for i, f := range p.Feedback {
			sha := f.SHA
			f.SHA = ""
			encoded, err := json.Marshal(f)
			prior := progresses[f.FromProgressSHA]
			if err != nil || hash(encoded) != sha || f.MetadataSHA != trainedPins[o.Variant] || f.WeightsSHA != pre.Weights[o.Variant] || f.CumulativeCalls != 12+i*6 || f.CI == nil || *f.CI != pre.CI || len(f.Judgments) != 6 || f.Passed != prior.SelectedPassed {
				return nil, errors.New("trained feedback digest or model binding differs")
			}
			prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d", f.Attempted, f.Passed, f.Cases, f.TypeRejected, 64-f.Attempted)
			if f.FirstFailure != nil {
				prefix += fmt.Sprintf(" mismatch=%d:%d:%d", f.FirstFailure.Input, f.FirstFailure.Actual, f.FirstFailure.Expected)
			}
			prefix += " ci=" + f.CI.Status
			for j, judgment := range f.Judgments {
				choice := d.Plan.Decisions[j]
				input := prefix + " selected=" + prior.Selection.Choices[choice.ID] + "\nintent: " + choice.Intent
				if judgment.Input != input || judgment.InputSHA != hash([]byte(input)) || judgment.DecisionID != choice.ID || judgment.Prediction.IntentSHA256 != hash([]byte(choice.Intent)) {
					return nil, errors.New("feedback changed observed context or original intention")
				}
			}
		}
		body, err := prepared.Compile(p.Search.Selection.Choices)
		if err != nil {
			return nil, err
		}
		execution, err := read(filepath.Join(dir, "executions", o.GoSHA+".json"))
		var actual []int64
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(execution, &actual); err != nil || !reflect.DeepEqual(actual, report.Expected) {
			return nil, errors.New("actual emitted Go observations differ")
		}
		for i, input := range report.Inputs {
			value, err := body.Evaluate(input)
			if err != nil || value.Int != actual[i] {
				return nil, errors.New("actual Go and selected typed body differ")
			}
			if i < 7 && (p.Cases[i].Input != input || p.Cases[i].Expected != d.Cases[i].Expected || p.Cases[i].Actual != value.Int) {
				return nil, errors.New("actual Go and native selected outcome differ")
			}
		}
		executions[o.GoSHA] = true
		m := o.Metrics
		if m.Wall <= 0 || m.RSS <= 0 || m.User < 0 || m.System < 0 || m.CPU != 100*float64(m.User+m.System)/float64(m.Wall) {
			return nil, errors.New("trained process counters differ")
		}
	}
	if attempts != 384 || len(executions) != 1 {
		return nil, errors.New("trained study totals differ")
	}
	return map[string]any{"schema": "gooo/trained-feedback-dogfood-audit/v1", "decision": "PASS", "report_sha256": hash(raw), "preexecution_sha256": hash(preRaw), "native_revision": report.Native,
		"candidate_attempts_reinterpreted": attempts, "recorded_local_predictions": 108, "recorded_feedback_predictions": 72, "recorded_native_calls": 6, "recorded_go_processes": 1, "recorded_function_evaluations": 16,
		"new_model_predictions": 0, "new_native_calls": 0, "new_go_processes": 0, "scope": "Frozen feature or main edition. Reinterpret candidates and verify source/model/receipt/context digests and actual Go values without new inference or execution. One existing compound intention, two languages, three models; no new independent task or host-utilization measurement."}, nil
}
