package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestFieldSelectionAndRuntimeScopesStaySeparate(t *testing.T) {
	for _, test := range []struct{ budget, fields, runtimeFields int }{{1, 6, 9}, {2, 9, 13}, {4, 12, 17}, {8, 15, 21}} {
		raw, err := os.ReadFile(fmt.Sprintf("../../publication/record-field-assembly-20261004/candidate/budget-%d-model-0.json", test.budget))
		if err != nil {
			t.Fatal(err)
		}
		r, err := measure(raw)
		if err != nil || len(r.Selection) != 1 {
			t.Fatal(r, err)
		}
		s := r.Selection[0]
		if s.Activity != "Select" || s.Fields.Passed != test.fields || s.Fields.Total != 15 ||
			r.Fields.Passed != test.runtimeFields || r.Fields.Total != 21 || r.Outputs.Total != 14 ||
			len(s.Gaps) != 15-test.fields || s.Fields.Percent == nil {
			t.Fatal(r)
		}
		var changed observation
		if err := json.Unmarshal(raw, &changed); err != nil {
			t.Fatal(err)
		}
		changed.Composition.Steps[0].Generation.Report.Record.FieldsPassed++
		mutated, err := json.Marshal(changed)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := measure(mutated); err == nil {
			t.Fatal("selection counter differs from values")
		}
	}
}

func TestTypeRejectedCandidatesAreSeparateFromEvaluatedCandidates(t *testing.T) {
	raw, err := os.ReadFile("../../publication/record-field-assembly-20261004/candidate/budget-8-model-0.json")
	if err != nil {
		t.Fatal(err)
	}
	var source observation
	if err = json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	record := source.Composition.Steps[0].Generation.Report.Record
	record.Attempts[3] = json.RawMessage(`{"mask":3,"status":"TYPECHECK_FAILED","reason":"declared and not used: saved","passed":0,"total":0,"fields_passed":0,"fields_total":0}`)
	scores, err := measureSelections(source)
	if err != nil || len(scores) != 1 || scores[0].Attempts != 7 || scores[0].Rejected != 1 || scores[0].Attempted != 8 ||
		scores[0].Fields.Passed != 15 || scores[0].Fields.Total != 15 || len(scores[0].Rejections) != 1 || scores[0].Rejections[0].Mask != 3 || scores[0].Rejections[0].Reason != "declared and not used: saved" {
		t.Fatal("unexecuted rejection changed finite completeness", scores, err)
	}
	for _, invalid := range []string{
		`{"status":"TYPECHECK_FAILED","reason":"saved unused","total":5}`,
		`{"status":"TYPECHECK_FAILED","reason":"saved unused","fields_passed":1}`,
		`{"status":"TYPECHECK_FAILED"}`,
		`{"status":"other"}`,
		`{"passed":6,"total":5,"fields_passed":15,"fields_total":15}`,
	} {
		record.Attempts[3] = json.RawMessage(invalid)
		if _, err = measureSelections(source); err == nil {
			t.Fatal("invalid candidate counts were accepted", invalid)
		}
	}
}
