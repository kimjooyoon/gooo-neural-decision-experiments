package main

import (
	"encoding/json"
	"os"
	"testing"
)

const candidateSHA = "38962397144d4418b75f571471225ff02d712617"

func readCapture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../publication/retained-composition-20261005/candidate/deterministic-0-fresh.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCurrentOrderedValuesAndPartialExpectations(t *testing.T) {
	r, err := measure(readCapture(t), candidateSHA, false)
	if err != nil {
		t.Fatal(err)
	}
	for i, frame := range r.Frames {
		if frame.Passed != [...]int{4, 2, 1}[i%3] || frame.Total != [...]int{4, 2, 2}[i%3] || frame.FieldsPassed != frame.FieldsTotal {
			t.Fatalf("frame %d: %+v", i, frame)
		}
	}
	if r.Profile != "deterministic" || r.Calls != 0 || r.StoredCalls != 0 {
		t.Fatal(r)
	}
}

func TestRejectChangedEvidence(t *testing.T) {
	mutations := map[string]func(*envelope){
		"previous input": func(e *envelope) {
			e.History[1].Traces[0].Deliveries[0].Inputs[0].Value = e.History[0].Traces[0].Deliveries[0].Inputs[0].Value
		},
		"false complete":       func(e *envelope) { e.History[2].Passed = 2 },
		"current build reused": func(e *envelope) { e.History[1].Build.Started = true },
		"changed program":      func(e *envelope) { e.History[1].GoSHA = "changed" },
		"missing run":          func(e *envelope) { e.History[2].Runs = e.History[2].Runs[:1] },
		"stale latest":         func(e *envelope) { e.Runtime = e.History[0] },
		"runtime inference":    func(e *envelope) { e.History[2].Calls = 1 },
		"wrong source":         func(e *envelope) { e.History[2].Compiler = "other" },
		"missing input":        func(e *envelope) { e.History[2].Traces[0].Deliveries[0].Inputs = nil },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var e envelope
			if err := json.Unmarshal(readCapture(t), &e); err != nil {
				t.Fatal(err)
			}
			mutate(&e)
			raw, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = measure(raw, candidateSHA, false); err == nil {
				t.Fatal("changed evidence accepted")
			}
		})
	}
}

func TestSavedCallsAreCurrentZero(t *testing.T) {
	raw, err := os.ReadFile("../../publication/retained-composition-20261005/candidate/fp32-0-saved.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := measure(raw, candidateSHA, true)
	if err != nil {
		t.Fatal(err)
	}
	if r.Calls != 0 || r.StoredCalls != 1 || r.PredictNS <= 0 {
		t.Fatal(r)
	}
	if _, err = measure(raw, candidateSHA, false); err == nil {
		t.Fatal("saved inference treated as new")
	}
}
