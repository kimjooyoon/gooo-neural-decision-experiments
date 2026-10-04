package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const candidateSource = "fa3c892791a2fd775a265c7524c9ee59adce45bf"

func observationFile(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "publication", "native-record-values-20261004", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPublishedRecordControls(t *testing.T) {
	calls, outputs, ports, edges, fields, runs := 0, 0, 0, 0, 0, 0
	var original envelope
	for _, mode := range []string{"model", "deterministic"} {
		for stage := range 3 {
			for _, replayed := range []bool{false, true} {
				stem := fmt.Sprintf("%s-%d", mode, stage)
				if replayed {
					stem += "-replay"
				}
				row, current, err := readObservation(observationFile(t, stem), candidateSource, mode, stage, replayed, resources{})
				if err != nil {
					t.Fatalf("%s: %v", stem, err)
				}
				if outputs != 0 && !sameProgram(original, current) {
					t.Fatal("program fixed point differs")
				}
				original = current
				calls += row.ModelCalls
				outputs += row.RuntimePassed
				ports += row.InputSlots
				edges += row.Deliveries
				fields += row.Fields
				runs += row.NativeRuns
			}
		}
	}
	if calls != 3 || outputs != 432 || ports != 648 || edges != 432 || fields != 1152 || runs != 24 {
		t.Fatalf("counts: %d/%d/%d/%d/%d/%d", calls, outputs, ports, edges, fields, runs)
	}
}

func TestRejectIncompleteOrAlteredRecords(t *testing.T) {
	mutations := map[string]func(*envelope){
		"missing field": func(e *envelope) {
			e.Runtime.Traces[0].Deliveries[1].ActualFields = e.Runtime.Traces[0].Deliveries[1].ActualFields[:1]
		},
		"reordered fields": func(e *envelope) {
			d := &e.Runtime.Traces[0].Deliveries[1]
			d.ActualFields[0], d.ActualFields[1] = d.ActualFields[1], d.ActualFields[0]
		},
		"different field value": func(e *envelope) {
			e.Runtime.Traces[0].Deliveries[1].ActualFields[0].Value = json.RawMessage(`"different"`)
		},
		"wrong field identity": func(e *envelope) { e.Runtime.Traces[0].Deliveries[2].InputFields[0].ID = "changed" },
		"different producer value": func(e *envelope) {
			e.Runtime.Traces[0].Deliveries[3].Inputs[0].Value = json.RawMessage(`{"state":"wait","title":"gooo"}`)
		},
		"absent multiport fields":         func(e *envelope) { e.Runtime.Traces[0].Deliveries[5].Inputs[1].Fields = nil },
		"missing record metadata":         func(e *envelope) { e.Composition.Plan.Records = nil },
		"different native field identity": func(e *envelope) { e.Composition.Plan.Records[0].Fields[0].GoName = "Title" },
		"unexpected provider":             func(e *envelope) { e.Composition.Steps[0].Generation.Report.Paths.Search.Selection.External = 1 },
		"different weights":               func(e *envelope) { e.Composition.Steps[0].Generation.Report.Paths.Search.Selection.Weights = "changed" },
		"missing native output":           func(e *envelope) { e.Runtime.Traces[0].Deliveries[3].Actual = nil },
		"failed expectation":              func(e *envelope) { failed := false; e.Runtime.Traces[0].Deliveries[2].Passed = &failed },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var e envelope
			if err := json.Unmarshal(observationFile(t, "model-0"), &e); err != nil {
				t.Fatal(err)
			}
			mutate(&e)
			raw, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := readObservation(raw, candidateSource, "model", 0, false, resources{}); err == nil {
				t.Fatal("altered observation accepted")
			}
		})
	}
}

func TestPublishedAdditionalControls(t *testing.T) {
	for _, mode := range []string{"model", "deterministic"} {
		if _, _, err := readObservation(observationFile(t, "parallel-"+mode), candidateSource, mode, 0, false, resources{}); err != nil {
			t.Fatal(err)
		}
		row, _, err := readObservationCases(observationFile(t, "additional-"+mode), candidateSource, mode, 0, true, resources{}, 8)
		if err != nil || row.RuntimePassed != 48 || row.ModelCalls != 0 || row.Fields != 128 {
			t.Fatalf("new runtime-only cases: %v / %+v", err, row)
		}
	}
	if err := checkPartial(observationFile(t, "partial"), candidateSource); err != nil {
		t.Fatal(err)
	}
}
