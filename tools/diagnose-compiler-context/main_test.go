package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bilingualstudy"
)

func TestDiagnosticsRejectForgedActual(t *testing.T) {
	t.Chdir("../..")
	pairs, err := bilingualstudy.Load("data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("runs/compiler-context-model-audit-20261001/compiler_context-fp32.json")
	if err != nil {
		t.Fatal(err)
	}
	var observations []observation
	if err = json.Unmarshal(raw, &observations); err != nil {
		t.Fatal(err)
	}
	for _, pair := range pairs {
		for _, row := range pair.Rows {
			if row.ID == observations[0].ID {
				if err = diagnose(row, observations[0]); err != nil {
					t.Fatal(err)
				}
				observations[0].Search.Attempts[0].Results[0].Actual++
				if err = diagnose(row, observations[0]); err == nil {
					t.Fatal("forged finite result accepted")
				}
				return
			}
		}
	}
	t.Fatal("frozen row missing")
}
