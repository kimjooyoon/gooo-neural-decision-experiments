package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type measurement struct {
	Mode                  string     `json:"mode"`
	Budget                int        `json:"attempt_budget,omitempty"`
	Trial                 *int       `json:"trial,omitempty"`
	Saved                 bool       `json:"saved"`
	SelectedMask          uint16     `json:"selected_mask"`
	Ranking               []uint16   `json:"ranking"`
	Attempts              int        `json:"evaluated_candidates"`
	SelectionPassed       int        `json:"selection_passed"`
	SelectionTotal        int        `json:"selection_total"`
	SelectionFieldsPassed int        `json:"selection_fields_passed"`
	SelectionFieldsTotal  int        `json:"selection_fields_total"`
	RuntimePassed         int        `json:"runtime_passed"`
	RuntimeTotal          int        `json:"runtime_total"`
	RuntimeFieldsPassed   int        `json:"runtime_fields_passed"`
	RuntimeFieldsTotal    int        `json:"runtime_fields_total"`
	ActualCalls           int        `json:"actual_model_calls"`
	StoredCalls           int        `json:"stored_model_calls"`
	PredictNS             int64      `json:"stored_prediction_ns"`
	GenerationMS          float64    `json:"generation_ms"`
	RuntimeMS             float64    `json:"runtime_ms"`
	Resources             *resources `json:"resources,omitempty"`
	GoSHA                 string     `json:"generated_go_sha256"`
	SourceSHA             string     `json:"selected_source_sha256"`
}

type fieldCase struct {
	Expected json.RawMessage `json:"expected"`
	Actual   json.RawMessage `json:"actual"`
	Passed   bool            `json:"passed"`
	Fields   []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Expected string `json:"expected"`
		Actual   string `json:"actual"`
		Passed   bool   `json:"passed"`
	} `json:"fields"`
}
type fieldReceipt struct {
	Passed       int               `json:"passed"`
	Total        int               `json:"total"`
	FieldsPassed int               `json:"fields_passed"`
	FieldsTotal  int               `json:"fields_total"`
	Selected     uint16            `json:"selected_mask"`
	Ranking      []uint16          `json:"ranking"`
	Attempts     []json.RawMessage `json:"attempts"`
	Cases        []fieldCase       `json:"cases"`
	Calls        int               `json:"model_calls"`
	PredictNS    int64             `json:"predict_ns"`
	Model        *struct {
		Metadata string `json:"metadata_sha256"`
		Weights  string `json:"weights_sha256"`
		Bytes    int    `json:"resident_tensor_bytes"`
	} `json:"model"`
}
type captured struct {
	Generated   bool `json:"generated_now"`
	Composition struct {
		Stage     string `json:"stage"`
		Elapsed   int64  `json:"elapsed_ns"`
		GoSHA     string `json:"generated_sha256"`
		SourceSHA string `json:"selected_source_sha256"`
		Gooo      string `json:"gooo_source"`
		Steps     []struct {
			Generation struct {
				Report struct {
					Compiler string        `json:"compiler_source_sha"`
					Record   *fieldReceipt `json:"record_assembly"`
				} `json:"report"`
			} `json:"generation"`
		} `json:"steps"`
	} `json:"composition"`
	Runtime struct {
		Stage    string `json:"stage"`
		Compiler string `json:"producer_source_sha"`
		Calls    int    `json:"model_calls"`
		Elapsed  int64  `json:"elapsed_ns"`
		Passed   int    `json:"finite_passed"`
		Total    int    `json:"finite_total"`
		Replayed bool   `json:"runtime_replayed"`
		Runs     []struct {
			Completed bool `json:"completed"`
			Exit      *int `json:"exit_code"`
		} `json:"runs"`
		Traces []struct {
			Index      int `json:"case_index"`
			Deliveries []struct {
				Actual   json.RawMessage `json:"actual"`
				Expected json.RawMessage `json:"expected"`
				Passed   *bool           `json:"passed"`
			} `json:"deliveries"`
		} `json:"traces"`
	} `json:"runtime"`
}

func measure(raw []byte, sha string, saved bool) (measurement, captured, error) {
	var observation captured
	row := measurement{Saved: saved}
	if err := json.Unmarshal(raw, &observation); err != nil {
		return row, observation, err
	}
	c, r := observation.Composition, observation.Runtime
	if len(sha) != 40 || c.Stage != "COMPLETE" || r.Stage != "COMPLETE" || observation.Generated == saved ||
		r.Compiler != sha || r.Calls != 0 || !r.Replayed || len(c.Steps) != 2 || len(r.Runs) != 2 || len(r.Traces) != 7 {
		return row, observation, fmt.Errorf("complete source-bound two-node/seven-case native observation required")
	}
	for _, run := range r.Runs {
		if !run.Completed || run.Exit == nil || *run.Exit != 0 {
			return row, observation, fmt.Errorf("native run incomplete")
		}
	}
	for _, step := range c.Steps {
		if step.Generation.Report.Compiler != sha {
			return row, observation, fmt.Errorf("generation source differs")
		}
	}
	f := c.Steps[0].Generation.Report.Record
	if f == nil || c.Steps[1].Generation.Report.Record != nil || len(f.Cases) != 5 || len(f.Attempts) < 1 || len(f.Attempts) > 8 || len(f.Ranking) != 8 {
		return row, observation, fmt.Errorf("five-case/three-choice selection observation required")
	}
	for _, test := range f.Cases {
		if len(test.Fields) != 3 || test.Passed != equalJSON(test.Actual, test.Expected) {
			return row, observation, fmt.Errorf("selection case values differ")
		}
		var actual, expected map[string]string
		if json.Unmarshal(test.Actual, &actual) != nil || json.Unmarshal(test.Expected, &expected) != nil || len(actual) != 3 || len(expected) != 3 {
			return row, observation, fmt.Errorf("selection record fields incomplete")
		}
		seen := make(map[string]bool, 3)
		if test.Passed {
			row.SelectionPassed++
		}
		row.SelectionTotal++
		for _, field := range test.Fields {
			a, present := actual[field.Name]
			w, declared := expected[field.Name]
			if field.ID == "" || seen[field.Name] || !present || !declared || a != field.Actual || w != field.Expected || field.Passed != (a == w) {
				return row, observation, fmt.Errorf("selection field values differ")
			}
			seen[field.Name] = true
			if field.Passed {
				row.SelectionFieldsPassed++
			}
			row.SelectionFieldsTotal++
		}
	}
	if row.SelectionPassed != f.Passed || row.SelectionTotal != f.Total || row.SelectionFieldsPassed != f.FieldsPassed || row.SelectionFieldsTotal != f.FieldsTotal {
		return row, observation, fmt.Errorf("selection counters differ")
	}
	for i, trace := range r.Traces {
		if trace.Index != i || len(trace.Deliveries) != 2 {
			return row, observation, fmt.Errorf("runtime case/graph incomplete")
		}
		for n, delivery := range trace.Deliveries {
			if delivery.Passed == nil || *delivery.Passed != equalJSON(delivery.Actual, delivery.Expected) {
				return row, observation, fmt.Errorf("runtime named result differs")
			}
			if *delivery.Passed {
				row.RuntimePassed++
			}
			row.RuntimeTotal++
			if n != 0 {
				continue
			}
			var actual, expected map[string]string
			if json.Unmarshal(delivery.Actual, &actual) != nil || json.Unmarshal(delivery.Expected, &expected) != nil || len(actual) != 3 || len(expected) != 3 {
				return row, observation, fmt.Errorf("runtime record fields incomplete")
			}
			for name, value := range expected {
				a, present := actual[name]
				if !present {
					return row, observation, fmt.Errorf("runtime expected field is missing")
				}
				if a == value {
					row.RuntimeFieldsPassed++
				}
				row.RuntimeFieldsTotal++
			}
			if i < 5 && (!equalJSON(delivery.Actual, f.Cases[i].Actual) || !equalJSON(delivery.Expected, f.Cases[i].Expected)) {
				return row, observation, fmt.Errorf("overlapping native and interpreted selection values differ")
			}
		}
	}
	if row.RuntimePassed != r.Passed || row.RuntimeTotal != r.Total {
		return row, observation, fmt.Errorf("runtime counters differ")
	}
	row.Mode = "deterministic"
	if f.Model != nil {
		row.Mode = "model"
		if f.Model.Metadata != "e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b" ||
			f.Model.Weights != "f5af1e35288cbad938d8416dd51873d83dff369a0e0c6e7e16f6e1f75b7f429b" || f.Model.Bytes != 74624 || f.Calls != 1 || f.PredictNS < 1 {
			return row, observation, fmt.Errorf("frozen own-model identity or prediction observation differs")
		}
	} else if f.Calls != 0 {
		return row, observation, fmt.Errorf("disconnected selection called a model")
	}
	row.SelectedMask, row.Ranking, row.Attempts = f.Selected, f.Ranking, len(f.Attempts)
	row.StoredCalls, row.PredictNS = f.Calls, f.PredictNS
	if !saved {
		row.ActualCalls = f.Calls
	}
	row.GenerationMS, row.RuntimeMS = float64(c.Elapsed)/1e6, float64(r.Elapsed)/1e6
	row.GoSHA, row.SourceSHA = c.GoSHA, c.SourceSHA
	return row, observation, nil
}

func equalJSON(a, b []byte) bool {
	var first, second any
	if json.Unmarshal(a, &first) != nil || json.Unmarshal(b, &second) != nil {
		return false
	}
	x, _ := json.Marshal(first)
	y, _ := json.Marshal(second)
	return bytes.Equal(x, y)
}
