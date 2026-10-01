package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
)

// Timing/source revision may differ. Source authority, model inputs and executable
// semantic observations must be exactly equal between feature and installed main.
func normalizedCompilerNative(value nativeResult, cell compilerNativeCell) ([]byte, error) {
	p := value.Report.Paths
	inputs := []string{}
	if p.Context != nil {
		for _, input := range p.Context.Inputs {
			inputs = append(inputs, input.SHA)
		}
	}
	contextIdentity := []string{}
	if p.Context != nil {
		c := p.Context
		contextIdentity = []string{c.Schema, c.Status, c.Activity, c.Source, c.Original, c.Ranked, c.Metadata, c.Feature}
	}
	return json.Marshal(struct {
		ID                                                      string
		Original, Document, Suite, Semantic, Activity, GoSource string
		Inputs, Context                                         []string
		Choices, Initial                                        map[string]string
		Metadata, Weights                                       string
		Calls, Attempts, Passed, Total                          int
		AttemptCases                                            any
		Actuals                                                 any
		Completeness                                            float64
		Executed                                                bool
		GoValues                                                []int64
	}{cell.ID, p.OriginalSHA, p.DocumentSHA, p.SuiteSHA, p.Binding.Source, value.Report.ActivityID, hash([]byte(value.Source)),
		inputs, contextIdentity, p.Search.Selection.Choices, p.Search.InitialProposals, p.Search.Selection.MetadataSHA256, p.Search.Selection.WeightsSHA256,
		p.Search.Selection.ModelCalls, p.Search.Evaluated, p.Search.SelectedTrainingPassed, p.Search.TrainingTotal,
		p.Search.Attempts, p.Cases, p.Completeness, cell.GoExecution, cell.GoValues})
}

func compareCompilerNativeDogfood(left, right string) (map[string]any, error) {
	load := func(root string) ([]compilerNativeCell, error) {
		raw, err := read(filepath.Join(root, "report.json"))
		if err != nil {
			return nil, err
		}
		var report struct {
			Cells []compilerNativeCell `json:"cells"`
		}
		if err = json.Unmarshal(raw, &report); err != nil || len(report.Cells) != 160 {
			return nil, errors.New("160 native cells required")
		}
		return report.Cells, nil
	}
	a, err := load(left)
	if err != nil {
		return nil, err
	}
	b, err := load(right)
	if err != nil {
		return nil, err
	}
	var hashes []map[string]string
	for i, cell := range a {
		other := b[i]
		if cell.ID != other.ID || cell.Row != other.Row || cell.Arm != other.Arm {
			return nil, errors.New("native replay cell identity differs")
		}
		readCell := func(root string, c compilerNativeCell) ([]byte, error) {
			raw, err := read(filepath.Join(root, c.ID+".json"))
			if err != nil || hash(raw) != c.Capture {
				return nil, errors.New("native capture digest differs")
			}
			var v nativeResult
			if err = json.Unmarshal(raw, &v); err != nil {
				return nil, err
			}
			return normalizedCompilerNative(v, c)
		}
		x, err := readCell(left, cell)
		if err != nil {
			return nil, err
		}
		y, err := readCell(right, other)
		if err != nil {
			return nil, err
		}
		if string(x) != string(y) {
			return nil, errors.New("native replay deterministic observation differs: " + cell.ID)
		}
		hashes = append(hashes, map[string]string{"id": cell.ID, "normalized_sha256": hash(x)})
	}
	return map[string]any{"schema": "gooo/compiler-native-feature-main-comparison/v1", "status": "PASS", "matched_cells": 160,
		"normalized_cells": hashes, "new_model_predictions": 0, "native_calls": 0,
		"scope": "source/semantic/input/model bindings, choices, candidate attempts, finite actuals, emitted Go and executed Go values equal; revision/timing/resources recorded separately"}, nil
}
