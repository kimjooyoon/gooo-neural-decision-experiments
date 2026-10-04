package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestPublishedGraphObservationsRetainActualAndStoredCalls(t *testing.T) {
	const sha = "c00bd714b4aa8cefc68c2b4d2ccca54fb079b3ed"
	for _, replay := range []bool{false, true} {
		name := "model-0.json"
		if replay {
			name = "model-0-replay.json"
		}
		raw, err := os.ReadFile("../../publication/native-composition-20261004/" + name)
		if err != nil {
			t.Fatal(err)
		}
		row, _, err := readObservation(raw, sha, "model", 0, replay, resources{})
		if err != nil {
			t.Fatal(err)
		}
		calls := 2
		if replay {
			calls = 0
		}
		if row.ModelCalls != calls || row.StoredModelCalls != 2 || row.RuntimePassed != 49 || row.Deliveries != 35 || row.ObservedValues != 49 || row.NativeRuns != 2 {
			t.Fatalf("observation fields lost: %+v", row)
		}
	}
}

func TestMissingGraphObservationFieldsAreNotSuccessfulDefaults(t *testing.T) {
	raw, err := os.ReadFile("../../publication/native-composition-20261004/model-0.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"passed", "calls", "trace", "runtime", "source", "run"} {
		var observation envelope
		if err := json.Unmarshal(raw, &observation); err != nil {
			t.Fatal(err)
		}
		switch change {
		case "passed":
			observation.Runtime.Passed = 0
		case "calls":
			observation.Composition.Steps[0].Generation.Report.Paths.Search.Selection.Calls = 0
		case "trace":
			observation.Runtime.Traces[0].Deliveries[0].Passed = nil
		case "runtime":
			observation.Runtime.Replayed = false
		case "source":
			observation.Composition.Steps[0].Generation.Report.Compiler = "other"
		case "run":
			observation.Runtime.Runs[0].Exit = nil
		}
		encoded, _ := json.Marshal(observation)
		if _, _, err := readObservation(encoded, "c00bd714b4aa8cefc68c2b4d2ccca54fb079b3ed", "model", 0, false, resources{}); err == nil {
			t.Fatal("missing/changed observation accepted", change)
		}
	}
}
