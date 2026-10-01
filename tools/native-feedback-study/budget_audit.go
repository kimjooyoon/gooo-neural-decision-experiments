package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type budgetObservation struct {
	ID                  string    `json:"id"`
	Arm                 string    `json:"arm"`
	Case                string    `json:"case_id"`
	Language            string    `json:"language"`
	Contract            string    `json:"contract"`
	Template            string    `json:"template"`
	Budget              int       `json:"candidate_budget"`
	Selected            uint16    `json:"selected_mask"`
	Intended            uint16    `json:"authored_intention_mask"`
	Sequence            []uint16  `json:"committed_candidate_masks"`
	Curve               []float64 `json:"committed_attempt_completion_percent"`
	BudgetArea          float64   `json:"mean_completion_over_budget_positions_percent"`
	Passed              int       `json:"finite_passed"`
	Cases               int       `json:"finite_cases"`
	LegalMaximum        int       `json:"independent_finite_legal_maximum_passed"`
	Attainment          float64   `json:"finite_legal_space_attainment_percent"`
	SeparatePassed      int       `json:"actual_separate_input_passed"`
	SeparateCases       int       `json:"actual_separate_input_cases"`
	Predictions         int       `json:"actual_model_predictions"`
	FeedbackPredictions int       `json:"feedback_predictions"`
	Skipped             int       `json:"fixed_coordinate_prediction_skips"`
	SHA                 string    `json:"generated_go_sha256"`
	Latency             int64     `json:"request_roundtrip_ns"`
	NativeMS            float64   `json:"native_total_ms"`
}

type budgetCollection struct {
	pre          budgetPre
	files        map[string]string
	sources      map[string]string
	inputs       []int64
	observations []budgetObservation
	processes    map[string]retainedProcess
}

func inspectBudgetValue(raw []byte, row compoundstudy.Case, arm familyArm, budget int, ci pathplan.CIHint,
	info retainedInfo, latency int64) (budgetObservation, string, error) {
	var v nativeResult
	if json.Unmarshal(raw, &v) != nil {
		return budgetObservation{}, "", errors.New("budget native response missing")
	}
	p := v.Report.Paths
	doc, _ := json.Marshal(row.Document)
	suite, _ := json.Marshal(row.Document.Cases)
	if v.Report.Decision != "PASS" || v.Report.Compiler != ci.SourceSHA || !v.Report.Types || !v.Report.Replay ||
		v.Report.Writes != 0 || v.Report.ActivityID != "compound-study://activity/compose-paths" || !p.Bound ||
		p.OriginalSHA != "sha256:"+hash([]byte(row.Source)) || p.DocumentSHA != "sha256:"+hash(doc) || p.SuiteSHA != "sha256:"+hash(suite) ||
		p.Unfixed != arm.Feedback || p.Timing.Load != 0 || !reflect.DeepEqual(retainedReceiptInfo(raw), &info) ||
		!p.Search.Selection.ExternalCallsKnown || p.Search.Selection.ExternalCalls != 0 || len(p.Cases) != len(row.Document.Cases) || latency <= 0 || p.Timing.Total <= 0 {
		return budgetObservation{}, "", errors.New("budget source, model, retained or native verification differs")
	}
	prepared, err := pathplan.Prepare(row.Document.Plan)
	if err != nil {
		return budgetObservation{}, "", err
	}
	if p.Search.Selection.PlanSHA256 != prepared.PlanSHA256() || p.Search.Selection.MetadataSHA256 != arm.Metadata || p.Search.Selection.WeightsSHA256 != arm.Weights {
		return budgetObservation{}, "", errors.New("search selection identity differs")
	}
	body, err := prepared.Compile(p.Search.Selection.Choices)
	if err != nil {
		return budgetObservation{}, "", err
	}
	actual, err := compoundFunction(v.Source)
	expected, e := compoundFunction(body.GoSource())
	if err != nil || e != nil || actual != expected {
		return budgetObservation{}, "", errors.New("actual native function differs from selected body")
	}
	capture := unfixedCapture{CaseID: row.ID, Arm: arm.Name, Unfixed: arm.Feedback, Progress: p.Progress, Feedback: p.Feedback, Search: p.Search, Source: body.GoSource(), WallNS: latency}
	skipped, err := verifyBoundedTrace(capture, row, arm, ci, false)
	if err != nil {
		return budgetObservation{}, "", err
	}
	count := len(p.Search.Attempts)
	if count < 1 || count > budget || p.Search.SelectedTrainingPassed < len(row.Document.Cases) && count != budget {
		return budgetObservation{}, "", errors.New("candidate budget or early completion differs")
	}
	mask, err := compoundstudy.Mask(row.Document.Plan, p.Search.Selection.Choices)
	if err != nil {
		return budgetObservation{}, "", err
	}
	best, _, err := finiteLegalMaximum(row)
	if err != nil || best != row.FiniteBestPassed || best <= 0 {
		return budgetObservation{}, "", errors.New("independent legal maximum differs")
	}
	o := budgetObservation{ID: fmt.Sprintf("%s-budget-%d-%s", budgetArmID(arm), budget, row.ID), Arm: budgetArmID(arm), Case: row.ID,
		Language: row.Language, Contract: row.Contract, Template: row.Template, Budget: budget, Selected: mask, Intended: row.IntendedMask,
		Passed: p.Search.SelectedTrainingPassed, Cases: len(row.Document.Cases), LegalMaximum: best,
		Attainment: 100 * float64(p.Search.SelectedTrainingPassed) / float64(best), Predictions: p.Search.Selection.ModelCalls,
		Skipped: skipped, SHA: hash([]byte(v.Source)), Latency: latency, NativeMS: p.Timing.Total}
	for _, f := range p.Feedback {
		o.FeedbackPredictions += f.ModelCalls
	}
	passed := 0
	for i, test := range row.Document.Cases {
		value, _ := compoundstudy.Oracle(row.Template, mask, test.Input)
		got := p.Cases[i]
		if got.Input != test.Input || got.Expected != test.Expected || got.Actual != value || got.Passed != (value == test.Expected) {
			return o, "", errors.New("native finite actual differs from state oracle")
		}
		if got.Passed {
			passed++
		}
	}
	if passed != o.Passed || p.Completeness != 100*float64(passed)/float64(o.Cases) {
		return o, "", errors.New("native completion denominator differs")
	}
	for _, a := range p.Search.Attempts {
		o.Sequence = append(o.Sequence, a.Mask)
	}
	committed, bestSoFar := 0, 0
	for _, point := range p.Progress {
		if len(point.NewAttempts) == 0 {
			if point.Attempted != committed || point.SelectedPassed != bestSoFar {
				return o, "", errors.New("non-attempt checkpoint changed completion")
			}
			continue
		}
		if len(point.NewAttempts) != 1 {
			return o, "", errors.New("more than one attempt per construction step")
		}
		committed++
		bestSoFar = max(bestSoFar, point.NewAttempts[0].Passed)
		if point.Attempted != committed || point.SelectedPassed != bestSoFar {
			return o, "", errors.New("progress differs from independently checked best-so-far")
		}
		o.Curve = append(o.Curve, 100*float64(point.SelectedPassed)/float64(o.Cases))
	}
	if len(o.Curve) != count {
		return o, "", errors.New("committed progress curve differs")
	}
	for i := 0; i < budget; i++ {
		o.BudgetArea += o.Curve[min(i, count-1)] / float64(budget)
	}
	return o, v.Source, nil
}

func collectBudget(dir, revision, native string) (budgetCollection, error) {
	c := budgetCollection{files: map[string]string{}, sources: map[string]string{}, processes: map[string]retainedProcess{}}
	readFile := func(name string) ([]byte, error) {
		raw, err := read(filepath.Join(dir, name))
		if err == nil {
			c.files[name] = hash(raw)
		}
		return raw, err
	}
	raw, err := readFile("preexecution.json")
	if err != nil {
		return c, err
	}
	c.pre, err = decodeBudgetPre(raw, revision, native)
	if err != nil {
		return c, err
	}
	arms, err := compoundArms()
	if err != nil {
		return c, err
	}
	inputs := map[int64]bool{}
	for a, arm := range arms {
		rows, err := budgetRows(a)
		if err != nil {
			return c, err
		}
		for _, row := range rows {
			for _, test := range append(append([]pathplan.TestCase(nil), row.Document.Cases...), row.Separate...) {
				inputs[test.Input] = true
			}
		}
		for _, budget := range budgetOrder(a) {
			for chunk := 0; chunk < len(rows)/budgetChunk; chunk++ {
				id := budgetBlockID(arm, budget, chunk)
				raw, err := readFile("captures/" + id + ".jsonl")
				if err != nil {
					return c, err
				}
				metric, err := readFile("metrics/" + id + ".json")
				if err != nil {
					return c, err
				}
				var p retainedProcess
				var info retainedInfo
				if json.Unmarshal(metric, &p) != nil || !validRetainedMetrics(p) || p.StartupNS <= 0 || len(p.RoundtripNS) != budgetChunk+1 ||
					json.Unmarshal(p.Setup, &info) != nil || info.Schema != "gooo/retained-path-model/v1" || info.Loaded != (arm.Path != "") ||
					info.Metadata != arm.Metadata || info.Weights != arm.Weights || info.SetupMS < 0 ||
					info.Scope != "one constructor load; excluded from request timing; fresh source and plan each request" {
					return c, errors.New("budget constructor or whole-process metrics differ")
				}
				resident := 0
				if arm.Variant == "fp32" {
					resident = 50912
				} else if arm.Path != "" {
					resident = 12896
				}
				if info.Resident != resident {
					return c, errors.New("resident decoded tensor bytes differ")
				}
				c.processes[id] = p
				lines := bytes.Split(bytes.TrimSuffix(raw, []byte{'\n'}), []byte{'\n'})
				if len(lines) != budgetChunk+1 {
					return c, errors.New("budget worker record denominator differs")
				}
				for i, line := range lines {
					var e retainedEnvelope
					if json.Unmarshal(line, &e) != nil || e.Schema != "gooo/native-body-stream-result/v1" || e.Sequence != i+1 || p.RoundtripNS[i] <= 0 {
						return c, errors.New("sequential result identity differs")
					}
					if i == 0 {
						var f struct {
							Bound    bool                  `json:"source_base_matched"`
							Original string                `json:"original_source_sha256"`
							Info     *retainedInfo         `json:"model_retention"`
							Search   pathplan.SearchResult `json:"search"`
						}
						if e.ID != "bad-source" || e.Status != "rejected" || e.Error == "" || len(e.Response) != 0 || json.Unmarshal(e.Failure, &f) != nil || f.Bound || f.Original != "sha256:"+hash([]byte("invalid source")) || f.Search.Selection.ModelCalls != 0 || !reflect.DeepEqual(f.Info, &info) {
							return c, errors.New("source rejection predicted or lost provenance")
						}
						continue
					}
					row := rows[chunk*budgetChunk+i-1]
					row.Document.Max = budget
					if e.ID != row.ID || e.Status != "completed" || len(e.Failure) != 0 || e.Error != "" {
						return c, errors.New("valid budget request failed")
					}
					o, source, err := inspectBudgetValue(e.Response, row, arm, budget, c.pre.Hint, info, p.RoundtripNS[i])
					if err != nil {
						return c, fmt.Errorf("%s %s: %w", id, row.ID, err)
					}
					c.sources[o.SHA] = source
					c.observations = append(c.observations, o)
				}
			}
		}
	}
	for x := range inputs {
		c.inputs = append(c.inputs, x)
	}
	sort.Slice(c.inputs, func(i, j int) bool { return c.inputs[i] < c.inputs[j] })
	if len(c.observations) != 1944 || len(c.processes) != 162 {
		return c, errors.New("study inventory differs")
	}
	for _, part := range []string{"captures", "metrics"} {
		entries, err := os.ReadDir(filepath.Join(dir, part))
		if err != nil || len(entries) != 162 {
			return c, errors.New("extra or absent process files")
		}
	}
	if err := validateBudgetPrefixes(c); err != nil {
		return c, err
	}
	return c, nil
}

func auditBudget(dir, revision, native string) (map[string]any, error) {
	c, err := collectBudget(dir, revision, native)
	if err != nil {
		return nil, err
	}
	rows, err := compoundRows()
	if err != nil {
		return nil, err
	}
	byCase := map[string]compoundstudy.Case{}
	for _, r := range rows {
		byCase[r.ID] = r
	}
	actuals := map[string][]int64{}
	for sha := range c.sources {
		name := "executions/" + sha + ".json"
		raw, err := read(filepath.Join(dir, name))
		var values []int64
		if err != nil || json.Unmarshal(raw, &values) != nil || len(values) != len(c.inputs) {
			return nil, errors.New("actual generated-Go execution missing")
		}
		c.files[name] = hash(raw)
		actuals[sha] = values
	}
	entries, err := os.ReadDir(filepath.Join(dir, "executions"))
	if err != nil || len(entries) != len(actuals) {
		return nil, errors.New("extra or absent generated-Go processes")
	}
	for i := range c.observations {
		o := &c.observations[i]
		row := byCase[o.Case]
		for j, x := range c.inputs {
			expected, _ := compoundstudy.Oracle(row.Template, o.Selected, x)
			if actuals[o.SHA][j] != expected {
				return nil, errors.New("actual emitted-Go value differs from independent state oracle")
			}
		}
		o.SeparateCases = len(row.Separate)
		for _, test := range row.Separate {
			j := sort.Search(len(c.inputs), func(i int) bool { return c.inputs[i] >= test.Input })
			if actuals[o.SHA][j] == test.Expected {
				o.SeparatePassed++
			}
		}
	}
	return summarizeBudget(c), nil
}
