package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func median(values []float64) float64 {
	sort.Float64s(values)
	n := len(values)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return values[n/2]
	}
	return (values[n/2-1] + values[n/2]) / 2
}
func executeGenerated(ctx context.Context, goBinary, source string, inputs []int64) ([]byte, error) {
	return executeFunction(ctx, goBinary, source, "ConditionalAssign", inputs)
}

func executeFunction(ctx context.Context, goBinary, source, function string, inputs []int64) ([]byte, error) {
	if function != "ConditionalAssign" && function != "ChoosePath" {
		return nil, errors.New("compiler-owned execution function required")
	}
	info, err := buildinfo.ReadFile(goBinary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return nil, errors.New("exact Go 1.27.1 execution toolchain required")
	}
	dir, err := os.MkdirTemp("", "gooo-native-feedback-execute-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err = os.Mkdir(filepath.Join(dir, "projection"), 0700); err != nil {
		return nil, err
	}
	var main strings.Builder
	main.WriteString("package main\nimport (\"encoding/json\";\"os\";p \"gooo.native.feedback/projection\")\nfunc main(){json.NewEncoder(os.Stdout).Encode([]int64{")
	for _, input := range inputs {
		fmt.Fprintf(&main, "p.%s(%d),", function, input)
	}
	main.WriteString("})}\n")
	for name, content := range map[string]string{"go.mod": "module gooo.native.feedback\n\ngo 1.27.1\n", "projection/generated.go": source, "main.go": main.String()} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			return nil, err
		}
	}
	raw, _, err := child(ctx, dir, goBinary, "run", ".")
	return raw, err
}
func audit(dir, root, goBinary string, execute bool) (map[string]any, error) {
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil {
		return nil, err
	}
	var report report
	if err = json.Unmarshal(raw, &report); err != nil || report.Schema != "gooo/native-feedback-study/v1" || len(report.Observations) != 28 || len(report.Pairs) != 12 ||
		report.Calls != 246 || report.FeedbackCalls != 102 || report.Attempts != 1080 || report.FinitePassed != 182 || report.FiniteCases != 196 ||
		report.Runner != "c6943ec61f1163c4b36618716c12b3d0df135973" || report.Native != "7430d23583223d59def01d4cff772dbab7e8bdab" {
		return nil, errors.New("fixed native integration study differs")
	}
	var pre struct {
		Inputs map[string]string `json:"source_inputs"`
	}
	preRaw, err := read(filepath.Join(dir, "preexecution.json"))
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(preRaw, &pre); err != nil {
		return nil, err
	}
	docs := map[string]document{}
	for _, lang := range []string{"en", "ko"} {
		name := "typed-path-conditional-assignment-plan.json"
		if lang == "ko" {
			name = "typed-path-conditional-assignment-ko-plan.json"
		}
		data, err := read(filepath.Join(root, "examples/body-codegen", name))
		if err != nil || hash(data) != pre.Inputs[lang] {
			return nil, errors.New("pinned native fixture changed")
		}
		var d document
		if err = json.Unmarshal(data, &d); err != nil {
			return nil, err
		}
		docs[lang] = d
	}
	data, err := read("studies/conditional-paths-v1/cohort/holdout-cases.json")
	if err != nil {
		return nil, err
	}
	var evaluation []pathplan.TestCase
	if err = json.Unmarshal(data, &evaluation); err != nil || len(evaluation) != 9 {
		return nil, errors.New("fixed separate-input cases required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	seen := map[string]observation{}
	executions := map[string][]int64{}
	calls, feedbackCalls, attempts, passed, evaluationPassed, rounds := 0, 0, 0, 0, 0, 0
	wall, cpu, rss, prediction := map[bool][]float64{}, map[bool][]float64{}, map[bool][]float64{}, map[bool][]float64{}
	for _, o := range report.Observations {
		mode := "baseline"
		if o.Feedback {
			mode = "feedback"
		}
		if o.ID != o.Language+"-"+o.Contract+"-"+o.Arm+"-"+mode || (o.Language != "en" && o.Language != "ko") ||
			(o.Contract != "complete" && o.Contract != "inconsistent") || (o.Arm != "offline" && modelPins[o.Arm] == "") || (o.Arm == "offline" && o.Feedback) {
			return nil, errors.New("policy identity outside fixed study")
		}
		if _, exists := seen[o.ID]; exists {
			return nil, errors.New("duplicate policy")
		}
		seen[o.ID] = o
		capture, err := read(filepath.Join(dir, "captures", o.ID+".json"))
		if err != nil || hash(capture) != o.Capture {
			return nil, errors.New("capture hash differs")
		}
		var native nativeResult
		if err = json.Unmarshal(capture, &native); err != nil {
			return nil, err
		}
		initialCalls := 0
		if o.Arm != "offline" {
			initialCalls = 6
		}
		if err = inspect(native, o.Feedback, initialCalls, report.Native); err != nil {
			return nil, err
		}
		p := native.Report.Paths
		if hash([]byte(native.Source)) != o.GoSHA || p.Search.Selection.ModelCalls != o.Calls || len(p.Feedback) != o.Rounds ||
			len(p.Search.Attempts) != o.Attempts || p.Search.SelectedTrainingPassed != o.Passed || p.Search.TrainingTotal != o.Cases ||
			p.Completeness != o.Completeness || p.Timing.Total != o.NativeMS || p.Timing.Search != o.SearchMS || p.Timing.Load != o.LoadMS {
			return nil, errors.New("policy differs from native capture")
		}
		d := docs[o.Language]
		d.Cases = append([]pathplan.TestCase(nil), d.Cases...)
		if o.Contract == "inconsistent" {
			d.Cases[6].Expected = 999
		}
		docRaw, err := json.MarshalIndent(d, "", "  ")
		if err != nil || hash(append(docRaw, '\n')) != o.DocumentSHA {
			return nil, errors.New("document bytes changed")
		}
		prepared, err := pathplan.Prepare(d.Plan)
		if err != nil {
			return nil, err
		}
		for _, a := range p.Search.Attempts {
			body, err := prepared.Compile(a.Choices)
			if err != nil {
				if a.Status != "TYPE_REJECTED" {
					return nil, errors.New("candidate type observation differs")
				}
				continue
			}
			if a.GoooSHA != hash([]byte(body.GoooSource())) || len(a.Results) != 7 {
				return nil, errors.New("candidate body or cases differ")
			}
			for i, c := range d.Cases {
				value, err := body.Evaluate(c.Input)
				if err != nil || a.Results[i].Actual != value.Int || a.Results[i].Expected != c.Expected || a.Results[i].Passed != (value.Int == c.Expected) {
					return nil, errors.New("candidate interpreted outcome differs")
				}
			}
		}
		body, err := prepared.Compile(p.Search.Selection.Choices)
		if err != nil {
			return nil, err
		}
		var inputs, expected []int64
		for i, c := range append(append([]pathplan.TestCase(nil), d.Cases...), evaluation...) {
			value, err := body.Evaluate(c.Input)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, c.Input)
			expected = append(expected, value.Int)
			if i < 7 {
				if p.Cases[i].Actual != value.Int || p.Cases[i].Input != c.Input || p.Cases[i].Expected != c.Expected {
					return nil, errors.New("native selected actual differs")
				}
			} else if value.Int == c.Expected {
				evaluationPassed++
			}
		}
		key := o.GoSHA
		actual, exists := executions[key]
		if !exists {
			name := filepath.Join(dir, "executions", key+".json")
			var execution []byte
			if execute {
				if _, err = os.Lstat(name); !os.IsNotExist(err) {
					return nil, errors.New("fresh execution capture required")
				}
				execution, err = executeGenerated(ctx, goBinary, native.Source, inputs)
				if err != nil {
					return nil, err
				}
				if err = os.MkdirAll(filepath.Dir(name), 0755); err != nil {
					return nil, err
				}
				if err = os.WriteFile(name, execution, 0644); err != nil {
					return nil, err
				}
			} else {
				execution, err = read(name)
				if err != nil {
					return nil, err
				}
			}
			if err = json.Unmarshal(execution, &actual); err != nil {
				return nil, err
			}
			executions[key] = actual
		}
		if !reflect.DeepEqual(actual, expected) {
			return nil, errors.New("actual generated Go differs from typed/native interpreters")
		}
		previous := ""
		priorFeedback := 0
		var predictionNS int64
		for _, progress := range p.Progress {
			sha := progress.SHA
			progress.SHA = ""
			encoded, err := json.Marshal(progress)
			if err != nil || hash(encoded) != sha {
				return nil, errors.New("progress digest differs")
			}
		}
		for _, prediction := range p.Search.Selection.Receipts {
			predictionNS += prediction.PredictNS
		}
		for _, f := range p.Feedback {
			sha := f.SHA
			f.SHA = ""
			encoded, err := json.Marshal(f)
			if err != nil || hash(encoded) != sha || f.PreviousSHA != previous || f.CumulativeCalls != initialCalls+priorFeedback+f.ModelCalls {
				return nil, errors.New("feedback digest or calls differ")
			}
			previous = sha
			priorFeedback += f.ModelCalls
			if f.MetadataSHA != modelPins[o.Arm] || f.CI == nil || f.CI.SourceSHA != "307159f041644a3aa56dfd325c345f5325aec902" || f.CI.Status != "PASS" || len(f.Judgments) != len(d.Plan.Decisions) {
				return nil, errors.New("feedback changed model or prior caller CI context")
			}
			for _, j := range f.Judgments {
				if hash([]byte(j.Input)) != j.InputSHA {
					return nil, errors.New("judgment input digest differs")
				}
				predictionNS += j.Prediction.PredictNS
				matched := false
				for _, choice := range d.Plan.Decisions {
					if j.DecisionID == choice.ID && strings.HasSuffix(j.Input, "\nintent: "+choice.Intent) && j.Prediction.IntentSHA256 == hash([]byte(choice.Intent)) {
						matched = true
					}
				}
				if !matched {
					return nil, errors.New("feedback rewrote original bounded intention")
				}
			}
		}
		if priorFeedback != o.FeedbackCalls || predictionNS != o.PredictionNS || o.Metrics.Wall <= 0 || o.Metrics.RSS <= 0 || o.Metrics.User < 0 || o.Metrics.System < 0 ||
			o.Metrics.CPU != 100*float64(o.Metrics.User+o.Metrics.System)/float64(o.Metrics.Wall) {
			return nil, errors.New("prediction/process totals differ")
		}
		calls += o.Calls
		feedbackCalls += o.FeedbackCalls
		attempts += o.Attempts
		passed += o.Passed
		rounds += o.Rounds
		if o.Arm != "offline" {
			wall[o.Feedback] = append(wall[o.Feedback], float64(o.Metrics.Wall)/1e6)
			cpu[o.Feedback] = append(cpu[o.Feedback], float64(o.Metrics.User+o.Metrics.System)/1e6)
			rss[o.Feedback] = append(rss[o.Feedback], float64(o.Metrics.RSS))
			prediction[o.Feedback] = append(prediction[o.Feedback], float64(o.PredictionNS)/1e3)
		}
	}
	paired := map[string]bool{}
	for _, pair := range report.Pairs {
		left, lok := seen[pair.Baseline]
		right, rok := seen[pair.Feedback]
		if !lok || !rok || left.Feedback || !right.Feedback || left.Language != right.Language || left.Contract != right.Contract || left.Arm != right.Arm ||
			pair.EqualGo != (left.GoSHA == right.GoSHA) || pair.EqualPassed != (left.Passed == right.Passed) || pair.AttemptDelta != right.Attempts-left.Attempts {
			return nil, errors.New("paired observation differs")
		}
		if paired[pair.Baseline] {
			return nil, errors.New("duplicate pair")
		}
		paired[pair.Baseline] = true
	}
	if calls != 246 || feedbackCalls != 102 || attempts != 1080 || passed != 182 || evaluationPassed != 252 || len(executions) != 2 {
		return nil, errors.New("audit aggregate differs")
	}
	medians := map[string]any{}
	for _, feedback := range []bool{false, true} {
		name := "baseline"
		if feedback {
			name = "feedback"
		}
		medians[name] = map[string]any{"native_wall_ms": median(wall[feedback]), "native_cpu_ms": median(cpu[feedback]), "native_child_max_rss_bytes": median(rss[feedback]), "summed_prediction_us": median(prediction[feedback])}
	}
	return map[string]any{"schema": "gooo/native-feedback-study-audit/v1", "decision": "PASS", "report_sha256": hash(raw), "preexecution_sha256": hash(preRaw), "native_calls": 28, "actual_local_predictions": calls, "feedback_predictions": feedbackCalls, "feedback_rounds": rounds, "candidate_attempts_reinterpreted": attempts, "repeated_finite_passes": passed, "repeated_finite_cases": 196, "separate_input_observations": 252, "separate_input_observations_passed": evaluationPassed, "unique_separate_inputs": 9, "actual_generated_go_processes_in_execution_capture_set": 2, "actual_generated_function_evaluations_in_execution_capture_set": 32, "actual_separate_input_evaluations_in_execution_capture_set": 18, "model_policy_medians": medians, "scope": "Audit reinterprets fixed candidate bodies and verifies native/progress/feedback hashes. Execute mode creates exactly two deduplicated Go execution captures; ordinary audit reuses them and makes no model, native or Go subprocess calls. Process medians are single-pass baseline-first observations, not host CPU utilization or causal performance gains."}, nil
}
