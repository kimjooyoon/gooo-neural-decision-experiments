package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const continuedReportSHA = "a5ceff7c5d373333a89e301e23a86a601a73d4fd39eaa28d093a9cdcb84d72fd"

func auditContinuedBound(dir string) (map[string]any, error) {
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil || hash(raw) != continuedReportSHA {
		return nil, errors.New("fixed continued-construction report required")
	}
	var report struct {
		Native       string  `json:"native_revision"`
		ExecutionSHA string  `json:"execution_sha256"`
		Inputs       []int64 `json:"execution_inputs"`
		Actual       []int64 `json:"execution_actual"`
		Observations []struct {
			Language, Mode string
			Capture        string `json:"capture_sha256"`
			Plan           string `json:"plan_sha256"`
			GoSHA          string `json:"generated_go_sha256"`
		} `json:"observations"`
	}
	if err = json.Unmarshal(raw, &report); err != nil || len(report.Observations) != 4 ||
		report.Native != "6a3e45d7f185c74f5136ab81b560a14a1366f367" {
		return nil, errors.New("continued report source differs")
	}
	execution, err := read(filepath.Join(dir, "go-execution.json"))
	var actual []int64
	if err != nil || hash(execution) != report.ExecutionSHA {
		return nil, errors.New("captured Go execution differs")
	}
	if err = json.Unmarshal(execution, &actual); err != nil || !reflect.DeepEqual(actual, report.Actual) {
		return nil, errors.New("actual emitted Go observations differ")
	}
	previous := map[string]nativeResult{}
	attempts := 0
	for _, o := range report.Observations {
		planRaw, err := read("runs/feedback-input-bound-20261001/" + o.Language + "-plan.json")
		if err != nil || hash(planRaw) != o.Plan {
			return nil, errors.New("original long intention differs")
		}
		var d document
		if err = json.Unmarshal(planRaw, &d); err != nil {
			return nil, err
		}
		capture, err := read(filepath.Join(dir, o.Language+"-"+o.Mode+".json"))
		if err != nil || hash(capture) != o.Capture {
			return nil, errors.New("continued native capture differs")
		}
		var v nativeResult
		if err = json.Unmarshal(capture, &v); err != nil || hash([]byte(v.Source)) != o.GoSHA {
			return nil, errors.New("continued emitted Go differs")
		}
		if o.Mode == "feedback" {
			if err = inspectDeclines(v, d, report.Native); err != nil {
				return nil, err
			}
			base := previous[o.Language]
			if base.Source != v.Source || base.Report.ActivityID != v.Report.ActivityID ||
				!reflect.DeepEqual(base.Report.Paths.Search.Attempts, v.Report.Paths.Search.Attempts) {
				return nil, errors.New("declined feedback changed stable output or candidate order")
			}
		} else {
			if err = inspect(v, false, 6, report.Native); err != nil {
				return nil, err
			}
			previous[o.Language] = v
		}
		prepared, err := pathplan.Prepare(d.Plan)
		if err != nil {
			return nil, err
		}
		for _, attempt := range v.Report.Paths.Search.Attempts {
			body, err := prepared.Compile(attempt.Choices)
			if err != nil || attempt.GoooSHA != hash([]byte(body.GoooSource())) || len(attempt.Results) != 7 {
				return nil, errors.New("recorded candidate body differs")
			}
			for i, c := range d.Cases {
				value, err := body.Evaluate(c.Input)
				if err != nil || attempt.Results[i].Actual != value.Int || attempt.Results[i].Expected != c.Expected ||
					attempt.Results[i].Passed != (value.Int == c.Expected) {
					return nil, errors.New("candidate finite observation differs")
				}
			}
			attempts++
		}
		body, err := prepared.Compile(v.Report.Paths.Search.Selection.Choices)
		if err != nil {
			return nil, err
		}
		for i, input := range report.Inputs {
			value, err := body.Evaluate(input)
			if err != nil || value.Int != actual[i] || value.Int != v.Report.Paths.Cases[i].Actual ||
				v.Report.Paths.Cases[i].Input != input || v.Report.Paths.Cases[i].Expected != d.Cases[i].Expected {
				return nil, errors.New("actual generated Go and native/typed values differ")
			}
		}
	}
	if attempts != 256 {
		return nil, errors.New("candidate audit total differs")
	}
	return map[string]any{"schema": "gooo/continued-bound-audit/v1", "decision": "PASS", "report_sha256": hash(raw),
		"candidate_attempts_reinterpreted": attempts, "recorded_native_calls": 4, "recorded_local_predictions": 24,
		"recorded_zero_call_context_declines": 4, "recorded_feedback_predictions": 0,
		"recorded_go_processes": 1, "recorded_go_function_evaluations": 7,
		"new_native_calls": 0, "new_model_predictions": 0, "new_go_processes": 0,
		"scope": "Reinterpret fixed captured candidates and verify emitted-Go observations, decline input/receipt hashes and unchanged candidate order. Recorded process metrics are observations, not new execution or host utilization measurements."}, nil
}
