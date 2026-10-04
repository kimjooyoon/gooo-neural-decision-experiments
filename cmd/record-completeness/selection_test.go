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
