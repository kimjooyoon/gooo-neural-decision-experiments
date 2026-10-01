package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type retainedInfo struct {
	Schema   string  `json:"schema"`
	Loaded   bool    `json:"loaded"`
	Metadata string  `json:"metadata_sha256,omitempty"`
	Weights  string  `json:"weights_sha256,omitempty"`
	Resident int     `json:"resident_tensor_bytes"`
	SetupMS  float64 `json:"setup_ms"`
	Scope    string  `json:"scope"`
}

type retainedEnvelope struct {
	Schema   string          `json:"schema"`
	Sequence int             `json:"sequence"`
	ID       string          `json:"correlation_id"`
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
	Failure  json.RawMessage `json:"failure_receipt"`
	Error    string          `json:"error"`
}

type retainedObservation struct {
	Arm     string  `json:"arm"`
	Mode    string  `json:"mode"`
	Index   int     `json:"request_index"`
	Case    string  `json:"case_id"`
	SHA     string  `json:"generated_go_sha256"`
	Calls   int     `json:"model_predictions"`
	Passed  int     `json:"finite_passed"`
	Cases   int     `json:"finite_cases"`
	Latency int64   `json:"observed_response_ns"`
	BodyMS  float64 `json:"native_total_ms"`
	LoadMS  float64 `json:"request_model_load_ms"`
}

type retainedCollection struct {
	pre          retainedPre
	files        map[string]string
	sources      map[string]string
	inputs       []int64
	observations []retainedObservation
	processes    map[string]retainedProcess
	values       map[string]nativeResult
}

func retainedReceiptInfo(raw []byte) *retainedInfo {
	var value struct {
		Report struct {
			Paths struct {
				Info *retainedInfo `json:"model_retention"`
			} `json:"body_paths"`
		} `json:"report"`
	}
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	return value.Report.Paths.Info
}

func validRetainedMetrics(p retainedProcess) bool {
	m := p.Metrics
	return m.Wall > 0 && m.User >= 0 && m.System >= 0 && m.RSS > 0 && m.CPU == 100*float64(m.User+m.System)/float64(m.Wall)
}

func verifyRetainedValue(raw []byte, row compoundstudy.Case, arm familyArm, ci pathplan.CIHint, info *retainedInfo) (nativeResult, error) {
	var v nativeResult
	if json.Unmarshal(raw, &v) != nil {
		return v, errors.New("native response missing")
	}
	observed := retainedReceiptInfo(raw)
	if !reflect.DeepEqual(observed, info) || info != nil && v.Report.Paths.Timing.Load != 0 {
		return v, errors.New("retained identity or request load differs")
	}
	if arm.Feedback {
		_, err := inspectNativeUnfixedFor(v, row, arm, ci, true, 1)
		return v, err
	}
	doc, _ := json.Marshal(row.Document)
	tests, _ := json.Marshal(row.Document.Cases)
	if v.Report.Paths.Unfixed || v.Report.Paths.OriginalSHA != "sha256:"+hash([]byte(row.Source)) ||
		v.Report.Paths.DocumentSHA != "sha256:"+hash(doc) || v.Report.Paths.SuiteSHA != "sha256:"+hash(tests) {
		return v, errors.New("disconnected source/document/suite differs")
	}
	_, err := inspectCompoundFor(v, row, arm, ci.SourceSHA)
	return v, err
}

func collectRetained(dir, revision, native string) (retainedCollection, error) {
	c := retainedCollection{files: map[string]string{}, sources: map[string]string{}, processes: map[string]retainedProcess{}, values: map[string]nativeResult{}}
	sha40 := regexp.MustCompile(`^[a-f0-9]{40}$`)
	sha64 := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if !sha40.MatchString(revision) || !sha40.MatchString(native) {
		return c, errors.New("explicit source pins required")
	}
	readFile := func(path string) ([]byte, error) {
		raw, err := read(filepath.Join(dir, path))
		if err == nil {
			c.files[path] = hash(raw)
		}
		return raw, err
	}
	raw, err := readFile("preexecution.json")
	if err != nil || json.Unmarshal(raw, &c.pre) != nil || c.pre.Schema != "gooo/retained-native-preexecution/v1" ||
		c.pre.Runner != revision || c.pre.Native != native || !sha64.MatchString(c.pre.CLI) || !sha64.MatchString(c.pre.Worker) ||
		!sha64.MatchString(c.pre.Go) || c.pre.Hint != (pathplan.CIHint{SourceSHA: native, Status: "UNKNOWN"}) {
		return c, errors.New("retained preexecution tuple differs")
	}
	arms, err := retainedArms()
	if err != nil {
		return c, err
	}
	rows, err := retainedRows()
	if err != nil {
		return c, err
	}
	inputs := map[int64]bool{}
	for _, row := range rows {
		for _, test := range append(append([]pathplan.TestCase(nil), row.Document.Cases...), row.Separate...) {
			inputs[test.Input] = true
		}
	}
	for value := range inputs {
		c.inputs = append(c.inputs, value)
	}
	sort.Slice(c.inputs, func(i, j int) bool { return c.inputs[i] < c.inputs[j] })
	add := func(v nativeResult, arm, mode string, index int, row compoundstudy.Case, latency int64) {
		sha := hash([]byte(v.Source))
		c.sources[sha] = v.Source
		key := fmt.Sprintf("%s-%s-%02d", arm, mode, index)
		c.values[key] = v
		c.observations = append(c.observations, retainedObservation{Arm: arm, Mode: mode, Index: index, Case: row.ID, SHA: sha,
			Calls: v.Report.Paths.Search.Selection.ModelCalls, Passed: v.Report.Paths.Search.SelectedTrainingPassed, Cases: len(row.Document.Cases),
			Latency: latency, BodyMS: v.Report.Paths.Timing.Total, LoadMS: v.Report.Paths.Timing.Load})
	}
	for _, arm := range arms {
		for i, row := range rows {
			id := fmt.Sprintf("%s-fresh-%02d", arm.Name, i)
			raw, err := readFile("captures/" + id + ".json")
			if err != nil {
				return c, err
			}
			v, err := verifyRetainedValue(raw, row, arm, c.pre.Hint, nil)
			if err != nil {
				return c, fmt.Errorf("%s: %w", id, err)
			}
			metric, err := readFile("metrics/" + id + ".json")
			if err != nil {
				return c, err
			}
			var p retainedProcess
			if json.Unmarshal(metric, &p) != nil || !validRetainedMetrics(p) || len(p.Setup) != 0 || len(p.RoundtripNS) != 0 || p.StartupNS != 0 {
				return c, errors.New("fresh metrics differ")
			}
			c.processes[id] = p
			add(v, arm.Name, "fresh", i, row, p.Metrics.Wall)
		}
		for _, workers := range []int{1, 4} {
			mode := fmt.Sprintf("retained-%d", workers)
			id := arm.Name + "-" + mode
			raw, err := readFile("captures/" + id + ".jsonl")
			if err != nil {
				return c, err
			}
			metric, err := readFile("metrics/" + id + ".json")
			if err != nil {
				return c, err
			}
			var p retainedProcess
			var setup retainedInfo
			if json.Unmarshal(metric, &p) != nil || !validRetainedMetrics(p) || p.StartupNS <= 0 || len(p.RoundtripNS) != 13 ||
				json.Unmarshal(p.Setup, &setup) != nil || setup.Schema != "gooo/retained-path-model/v1" || setup.Loaded != arm.Feedback ||
				setup.Metadata != arm.Metadata || setup.Weights != arm.Weights || setup.SetupMS < 0 || setup.Scope != "one constructor load; excluded from request timing; fresh source and plan each request" {
				return c, errors.New("worker setup or metrics differ")
			}
			resident := 0
			if arm.Variant == "fp32" {
				resident = 50912
			} else if arm.Feedback {
				resident = 12896
			}
			if setup.Resident != resident {
				return c, errors.New("decoded tensor bytes differ")
			}
			c.processes[id] = p
			lines := bytes.Split(bytes.TrimSuffix(raw, []byte{'\n'}), []byte{'\n'})
			if len(lines) != 13 {
				return c, errors.New("worker record denominator differs")
			}
			seen := map[int]bool{}
			for _, line := range lines {
				var e retainedEnvelope
				if json.Unmarshal(line, &e) != nil || e.Schema != "gooo/native-body-stream-result/v1" || e.Sequence < 1 || e.Sequence > 13 || seen[e.Sequence] || p.RoundtripNS[e.Sequence-1] <= 0 {
					return c, errors.New("worker sequence differs")
				}
				seen[e.Sequence] = true
				if e.Sequence == 1 {
					var f struct {
						Bound    bool                  `json:"source_base_matched"`
						Original string                `json:"original_source_sha256"`
						Info     *retainedInfo         `json:"model_retention"`
						Search   pathplan.SearchResult `json:"search"`
					}
					if e.ID != "bad-source" || e.Status != "rejected" || len(e.Response) != 0 || e.Error == "" || json.Unmarshal(e.Failure, &f) != nil ||
						f.Bound || f.Original != "sha256:"+hash([]byte("invalid source")) || f.Search.Selection.ModelCalls != 0 || !reflect.DeepEqual(f.Info, &setup) {
						return c, errors.New("source failure lost or predicted")
					}
					continue
				}
				i := e.Sequence - 2
				if e.Status != "completed" || e.ID != fmt.Sprintf("request-%02d", i) || len(e.Failure) != 0 || e.Error != "" {
					return c, errors.New("valid worker transport differs")
				}
				v, err := verifyRetainedValue(e.Response, rows[i], arm, c.pre.Hint, &setup)
				if err != nil {
					return c, fmt.Errorf("%s-%d: %w", id, i, err)
				}
				base := c.values[fmt.Sprintf("%s-fresh-%02d", arm.Name, i)]
				if v.Source != base.Source || !reflect.DeepEqual(v.Report.Paths.Search.Attempts, base.Report.Paths.Search.Attempts) ||
					!reflect.DeepEqual(v.Report.Paths.Cases, base.Report.Paths.Cases) {
					return c, errors.New("retained request changed candidate sequence, selected body or actuals")
				}
				add(v, arm.Name, mode, i, rows[i], p.RoundtripNS[e.Sequence-1])
			}
		}
	}
	return c, nil
}

func auditRetained(dir, revision, native string) (map[string]any, error) {
	c, err := collectRetained(dir, revision, native)
	if err != nil {
		return nil, err
	}
	rows, err := retainedRows()
	if err != nil {
		return nil, err
	}
	for sha := range c.sources {
		raw, err := read(filepath.Join(dir, "executions", sha+".json"))
		if err != nil {
			return nil, err
		}
		var actuals []int64
		if json.Unmarshal(raw, &actuals) != nil || len(actuals) != len(c.inputs) {
			return nil, errors.New("actual generated-Go values missing")
		}
		c.files["executions/"+sha+".json"] = hash(raw)
		for _, o := range c.observations {
			if o.SHA != sha {
				continue
			}
			v := c.values[fmt.Sprintf("%s-%s-%02d", o.Arm, o.Mode, o.Index)]
			row := rows[o.Index]
			mask, err := compoundstudy.Mask(row.Document.Plan, v.Report.Paths.Search.Selection.Choices)
			if err != nil {
				return nil, err
			}
			for i, input := range c.inputs {
				want, err := compoundstudy.Oracle(row.Template, mask, input)
				if err != nil || actuals[i] != want {
					return nil, errors.New("actual Go differs from independent state oracle")
				}
			}
		}
	}
	predictions, passed, cases := 0, 0, 0
	for _, o := range c.observations {
		predictions += o.Calls
		passed += o.Passed
		cases += o.Cases
	}
	return map[string]any{"schema": "gooo/retained-native-audit/v1", "decision": "PASS", "runner_revision": revision, "native_revision": native,
		"preexecution_sha256": c.files["preexecution.json"], "evidence_sha256": c.files, "valid_constructions": len(c.observations), "source_rejections": 10,
		"native_processes": len(c.processes), "actual_model_predictions": predictions, "finite_passes": passed, "finite_cases": cases,
		"finite_completeness_percent": 100 * float64(passed) / float64(cases), "selected_body_and_candidate_sequence_equal_pairs": 120,
		"actual_generated_go_processes": len(c.sources), "actual_function_evaluations": len(c.sources) * len(c.inputs), "execution_inputs": c.inputs,
		"observations": c.observations, "processes": c.processes, "new_audit_predictions": 0, "new_audit_native_processes": 0, "new_audit_go_processes": 0,
		"scope": c.pre.Scope + " Captures and costs precede inspection; independent source/function AST/receipt/finite/state-oracle reconciliation. Both repeated passes are development observations, not independent tasks. Fresh latency includes startup/load; retained-1 latency excludes one process startup/setup; retained-4 latency includes queueing. Whole process CPU/RSS covers native children, not host utilization or controller resources. No causal speedup, packed-compute or all-input correctness claim. Original UNKNOWN hint is preserved."}, nil
}
