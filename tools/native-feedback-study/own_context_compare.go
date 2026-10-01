package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
)

// Compare deterministic observations only; inference timings and CPU floats are
// preserved in raw captures but are not equality constraints across platforms.
func compareOwnNativeContext(left, right string) (map[string]any, error) {
	rows, err := ownContextRows()
	if err != nil {
		return nil, err
	}
	hashes := map[string]string{}
	for _, row := range rows {
		for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary", "offline"} {
			id := row.ID + "-" + arm
			a, err := ownContextComparable(left, id)
			if err != nil {
				return nil, err
			}
			b, err := ownContextComparable(right, id)
			if err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(a, b) {
				return nil, errors.New("deterministic native context comparison differs: " + id)
			}
			raw, _ := json.Marshal(a)
			hashes[id] = hash(raw)
		}
	}
	return map[string]any{"schema": "gooo/own-model-native-context-comparison/v1", "decision": "PASS",
		"cells_compared": 32, "normalized_sha256": hashes, "new_native_calls": 0, "new_model_predictions": 0, "new_go_processes": 0,
		"scope": "Selected Go, source binding, original/ranked context identities and input hashes, choices, counts, finite actuals and compiled execution values. Compiler revision, raw probabilities, latency and resources remain distinct captured observations."}, nil
}

func ownContextComparable(root, id string) (map[string]any, error) {
	raw, err := read(filepath.Join(root, id+".json"))
	if err != nil {
		return nil, err
	}
	var v nativeResult
	if err = json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	p := v.Report.Paths
	raw, err = read(filepath.Join(root, id+"-execution.json"))
	if err != nil {
		return nil, err
	}
	var execution ownContextExecution
	if err = json.Unmarshal(raw, &execution); err != nil {
		return nil, err
	}
	return map[string]any{"source": v.Source, "activity": v.Report.ActivityID, "original": p.OriginalSHA, "document": p.DocumentSHA,
		"suite": p.SuiteSHA, "bound": p.Bound, "binding": p.Binding, "context": p.Context, "choices": p.Search.Selection.Choices,
		"metadata": p.Search.Selection.MetadataSHA256, "weights": p.Search.Selection.WeightsSHA256, "plan": p.Search.Selection.PlanSHA256,
		"calls": p.Search.Selection.ModelCalls, "status": p.Search.Status, "combinations": p.Search.DeclaredCombinations,
		"unattempted": p.Search.Unattempted, "evaluated": p.Search.Evaluated, "type_rejected": p.Search.TypeRejected,
		"attempts": p.Search.Attempts, "passed": p.Search.SelectedTrainingPassed, "total": p.Search.TrainingTotal, "cases": p.Cases,
		"compiled_values": execution.Values, "finite_percent": p.Completeness}, nil
}
