package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bilingualstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

func TestCompilerCurriculumFixtureUsesIndependentFiniteTargets(t *testing.T) {
	pairs, err := bilingualstudy.Load("../../data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range pairs {
		for _, row := range pair.Rows {
			doc, source, err := compilerFixture(row)
			if err != nil || doc.Max != 2 || !strings.Contains(string(source), "activity ChoosePath(Integer) -> Integer computes") ||
				strings.Contains(string(source), "intent:") {
				t.Fatal("source fixture differs")
			}
			for _, label := range pair.Options {
				compiled, err := pathplan.Compile(doc.Plan, map[string]string{"structure": label})
				if err != nil {
					t.Fatal(err)
				}
				for _, test := range row.Cases {
					actual, err := compiled.Evaluate(test.Input)
					expected, oracleErr := pathstudy.Oracle(row.Family, label == pair.Options[1], row.Configuration, test.Input)
					if err != nil || oracleErr != nil || actual.Int != expected {
						t.Fatal("independent candidate oracle differs")
					}
				}
			}
		}
	}
}

func TestCompilerCapturePreservesRawReceiptBytes(t *testing.T) {
	raw := []byte("{\"schema\": \"test\"}\n")
	captured := compilerCapture{ID: "public", Receipt: raw}
	encoded, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	var decoded compilerCapture
	if err = json.Unmarshal(encoded, &decoded); err != nil || hash(decoded.Receipt) != hash(raw) {
		t.Fatal("raw receipt altered")
	}
	if _, err = inspectCompilerExport(raw, document{}, nil); err == nil {
		t.Fatal("unbound export accepted")
	}
}
