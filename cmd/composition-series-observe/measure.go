package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type process struct {
	Started   bool  `json:"started"`
	Completed bool  `json:"completed"`
	Exit      *int  `json:"exit_code"`
	Wall      int64 `json:"wall_ns"`
	User      int64 `json:"user_ns"`
	System    int64 `json:"system_ns"`
}
type suite struct {
	Schema string `json:"schema"`
	Cases  []struct {
		Inputs   map[string]json.RawMessage `json:"inputs"`
		Expected map[string]json.RawMessage `json:"expected"`
	} `json:"cases"`
}
type frame struct {
	Stage         string    `json:"stage"`
	Compiler      string    `json:"producer_source_sha"`
	GoSHA         string    `json:"generated_sha256"`
	DriverSHA     string    `json:"driver_sha256"`
	SuiteSHA      string    `json:"runtime_suite_sha256"`
	ExecutableSHA string    `json:"executable_sha256"`
	Elapsed       int64     `json:"elapsed_ns"`
	Passed        int       `json:"finite_passed"`
	Total         int       `json:"finite_total"`
	Calls         int       `json:"model_calls"`
	Replayed      bool      `json:"runtime_replayed"`
	Build         process   `json:"build"`
	Toolchain     process   `json:"toolchain"`
	Runs          []process `json:"runs"`
	Artifact      *struct {
		Key      string  `json:"key_sha256"`
		Reused   bool    `json:"reused"`
		Verified bool    `json:"executable_verified"`
		Original process `json:"source_build"`
	} `json:"artifact"`
	Traces []struct {
		Index      int `json:"case_index"`
		Deliveries []struct {
			ID       string          `json:"activity_id"`
			Actual   json.RawMessage `json:"actual"`
			Expected json.RawMessage `json:"expected"`
			Passed   *bool           `json:"passed"`
			Inputs   []struct {
				Value json.RawMessage `json:"value"`
			} `json:"inputs"`
		} `json:"deliveries"`
	} `json:"traces"`
}
type frameMeasurement struct {
	Passed       int     `json:"passed"`
	Total        int     `json:"total"`
	FieldsPassed int     `json:"fields_passed"`
	FieldsTotal  int     `json:"fields_total"`
	Reused       bool    `json:"reused"`
	BuildMS      float64 `json:"current_build_ms"`
	RuntimeMS    float64 `json:"runtime_ms"`
	ChildCPUms   float64 `json:"current_child_cpu_ms"`
	FirstRunMS   float64 `json:"first_native_run_ms"`
	ReplayRunMS  float64 `json:"replay_native_run_ms"`
}
type measurement struct {
	Profile     string             `json:"profile"`
	Saved       bool               `json:"saved"`
	Calls       int                `json:"current_model_calls"`
	StoredCalls int                `json:"stored_model_calls"`
	PredictNS   int64              `json:"stored_predict_ns"`
	TensorBytes int                `json:"resident_tensor_bytes"`
	GoSHA       string             `json:"generated_go_sha256"`
	DriverSHA   string             `json:"driver_sha256"`
	Frames      []frameMeasurement `json:"frames"`
}
type envelope struct {
	Generated   bool `json:"generated_now"`
	Composition struct {
		Steps []struct {
			Generation struct {
				Report struct {
					Compiler string `json:"compiler_source_sha"`
					Record   *struct {
						Calls   int   `json:"model_calls"`
						Predict int64 `json:"predict_ns"`
						Passed  int   `json:"fields_passed"`
						Total   int   `json:"fields_total"`
						Model   *struct {
							Bytes    int    `json:"resident_tensor_bytes"`
							Metadata string `json:"metadata_sha256"`
						} `json:"model"`
					} `json:"record_assembly"`
				} `json:"report"`
			} `json:"generation"`
		} `json:"steps"`
	} `json:"composition"`
	Runtime frame   `json:"runtime"`
	History []frame `json:"runtime_history"`
	Series  struct {
		Suites []suite `json:"suites"`
	} `json:"case_series"`
}

func equalJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	xb, _ := json.Marshal(x)
	yb, _ := json.Marshal(y)
	return bytes.Equal(xb, yb)
}
func complete(p process) bool { return p.Started && p.Completed && p.Exit != nil && *p.Exit == 0 }

func measure(raw []byte, sha string, saved bool) (measurement, error) {
	var e envelope
	r := measurement{Profile: "deterministic", Saved: saved}
	if err := json.Unmarshal(raw, &e); err != nil {
		return r, err
	}
	if len(sha) != 40 || e.Generated == saved || len(e.Composition.Steps) != 2 ||
		len(e.Series.Suites) != 3 || len(e.History) != 6 {
		return r, fmt.Errorf("three ordered suites repeated twice required")
	}
	for i, step := range e.Composition.Steps {
		p := step.Generation.Report
		if p.Compiler != sha {
			return r, fmt.Errorf("construction revision differs")
		}
		if i == 0 {
			if p.Record == nil || p.Record.Passed != 15 || p.Record.Total != 15 {
				return r, fmt.Errorf("complete field construction required")
			}
			r.StoredCalls, r.PredictNS = p.Record.Calls, p.Record.Predict
			if p.Record.Model != nil {
				profiles := map[string]string{
					"e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b": "fp32",
					"2bc62124e2ca72e2effe1e770842b7137f8968e2cd1238f139b7d1d503beb1bc": "ptq_ternary",
					"fc5446ea7ca31ba2eb4ff29957cec5614df9820cfd110bc1f51a79f996325c43": "qat_ternary",
				}
				r.Profile, r.TensorBytes = profiles[p.Record.Model.Metadata], p.Record.Model.Bytes
				if r.Profile == "" || r.StoredCalls != 1 || r.PredictNS < 1 {
					return r, fmt.Errorf("frozen model observation differs")
				}
			} else if r.StoredCalls != 0 {
				return r, fmt.Errorf("disconnected model called")
			}
		}
	}
	if !saved {
		r.Calls = r.StoredCalls
	}
	for i, f := range e.History {
		m, err := measureFrame(f, e.Series.Suites[i%3], sha)
		if err != nil {
			return r, fmt.Errorf("frame %d: %w", i, err)
		}
		if f.Artifact == nil || !f.Artifact.Verified || f.Artifact.Reused != (i > 0) ||
			!complete(f.Artifact.Original) || f.Build.Started != (i == 0) || f.Toolchain.Started != (i == 0) {
			return r, fmt.Errorf("current build/reuse observation differs")
		}
		first := e.History[0]
		if f.GoSHA != first.GoSHA || f.DriverSHA != first.DriverSHA || f.Artifact.Key != first.Artifact.Key || f.ExecutableSHA != first.ExecutableSHA {
			return r, fmt.Errorf("retained program identity changed")
		}
		if i >= 3 && f.SuiteSHA != e.History[i-3].SuiteSHA {
			return r, fmt.Errorf("repeated suite identity differs")
		}
		r.Frames = append(r.Frames, m)
	}
	last, _ := json.Marshal(e.History[5])
	latest, _ := json.Marshal(e.Runtime)
	if !bytes.Equal(last, latest) {
		return r, fmt.Errorf("latest runtime differs from history")
	}
	r.GoSHA, r.DriverSHA = e.History[0].GoSHA, e.History[0].DriverSHA
	return r, nil
}

func measureFrame(f frame, s suite, sha string) (frameMeasurement, error) {
	r := frameMeasurement{RuntimeMS: float64(f.Elapsed) / 1e6, BuildMS: float64(f.Build.Wall) / 1e6}
	if len(s.Cases) < 1 || len(s.Cases) > 128 || f.Stage != "COMPLETE" || f.Compiler != sha ||
		!f.Replayed || f.Calls != 0 || len(f.Runs) != 2 || len(f.Traces) != len(s.Cases) {
		return r, fmt.Errorf("current compiled values required")
	}
	for _, p := range append([]process{f.Build, f.Toolchain}, f.Runs...) {
		if p.Started && !complete(p) {
			return r, fmt.Errorf("child did not complete")
		}
		r.ChildCPUms += float64(p.User+p.System) / 1e6
	}
	for _, p := range f.Runs {
		if !complete(p) {
			return r, fmt.Errorf("missing native run")
		}
	}
	r.FirstRunMS, r.ReplayRunMS = float64(f.Runs[0].Wall)/1e6, float64(f.Runs[1].Wall)/1e6
	for i, trace := range f.Traces {
		if trace.Index != i || len(trace.Deliveries) != 2 {
			return r, fmt.Errorf("current case trace differs")
		}
		for j, d := range trace.Deliveries {
			name := [...]string{"Select", "Label"}[j]
			wanted := s.Cases[i].Expected[name]
			match := equalJSON(d.Actual, wanted)
			if d.ID != "fieldassembly://activity/"+[...]string{"select", "label"}[j] || !equalJSON(d.Expected, wanted) || d.Passed == nil || *d.Passed != match {
				return r, fmt.Errorf("current named value or match differs")
			}
			r.Total++
			if match {
				r.Passed++
			}
			if j != 0 {
				continue
			}
			if len(d.Inputs) != 2 || !equalJSON(d.Inputs[0].Value, s.Cases[i].Inputs["Select.input0"]) || !equalJSON(d.Inputs[1].Value, s.Cases[i].Inputs["Select.input1"]) {
				return r, fmt.Errorf("current ordered inputs differ")
			}
			var actual, expected map[string]string
			if json.Unmarshal(d.Actual, &actual) != nil || json.Unmarshal(wanted, &expected) != nil || len(actual) != 3 || len(expected) != 3 {
				return r, fmt.Errorf("record fields missing")
			}
			for k, w := range expected {
				a, ok := actual[k]
				if !ok {
					return r, fmt.Errorf("record field missing")
				}
				r.FieldsTotal++
				if a == w {
					r.FieldsPassed++
				}
			}
		}
	}
	if r.Total != f.Total || r.Passed != f.Passed {
		return r, fmt.Errorf("reported finite counts differ")
	}
	if f.Artifact != nil {
		r.Reused = f.Artifact.Reused
	}
	return r, nil
}
