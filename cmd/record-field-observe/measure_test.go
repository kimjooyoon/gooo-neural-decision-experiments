package main

import (
	"encoding/json"
	"os"
	"testing"
)

const candidate = "48f7027584281ef87e43220f680093bc693f4488"

func capturedFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata-" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCompleteAndPartialFiniteFieldMeasurements(t *testing.T) {
	for _, name := range []string{"complete", "partial"} {
		row, _, err := measure(capturedFixture(t, name), candidate, false)
		if err != nil {
			t.Fatal(err)
		}
		fields, cases, runtimeFields, runtimeOutputs := 15, 5, 21, 14
		if name == "partial" {
			fields, cases, runtimeFields, runtimeOutputs = 6, 2, 9, 6
		}
		if row.SelectionFieldsPassed != fields || row.SelectionPassed != cases || row.RuntimeFieldsPassed != runtimeFields || row.RuntimePassed != runtimeOutputs ||
			row.SelectionFieldsTotal != 15 || row.RuntimeFieldsTotal != 21 || row.RuntimeTotal != 14 {
			t.Fatalf("%s observations: %+v", name, row)
		}
	}
}

func TestCounterFieldAndSourceChangesAreReported(t *testing.T) {
	baseline := capturedFixture(t, "complete")
	mutations := []func(*captured){
		func(v *captured) { v.Runtime.Passed-- },
		func(v *captured) { v.Runtime.Compiler = "changed" },
		func(v *captured) { v.Composition.Steps[0].Generation.Report.Record.FieldsPassed-- },
		func(v *captured) {
			v.Composition.Steps[0].Generation.Report.Record.Cases[0].Fields[0].Actual = "changed"
		},
		func(v *captured) { v.Composition.Steps[0].Generation.Report.Record.Cases[0].Fields[0].Name = "missing" },
		func(v *captured) { v.Composition.Steps[0].Generation.Report.Record.Model.Weights = "changed" },
		func(v *captured) {
			v.Runtime.Traces[0].Deliveries[0].Actual = json.RawMessage(`{"reason":"검토:accepted","state":"ready","different":"한글"}`)
		},
	}
	for i, mutate := range mutations {
		var changed captured
		if err := json.Unmarshal(baseline, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		raw, err := json.Marshal(changed)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = measure(raw, candidate, false); err == nil {
			t.Fatalf("changed observation %d accepted", i)
		}
	}
}

func TestSavedObservationCarriesHistoricalCostsSeparately(t *testing.T) {
	var changed captured
	if err := json.Unmarshal(capturedFixture(t, "complete"), &changed); err != nil {
		t.Fatal(err)
	}
	changed.Generated = false
	raw, _ := json.Marshal(changed)
	row, _, err := measure(raw, candidate, true)
	if err != nil || row.ActualCalls != 0 || row.StoredCalls != 1 || row.PredictNS < 1 {
		t.Fatal("historical prediction accounting", row, err)
	}
}
