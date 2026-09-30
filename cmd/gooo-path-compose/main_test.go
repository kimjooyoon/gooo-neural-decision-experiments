package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

func TestCLIUsesDeterministicPlanWithoutModel(t *testing.T) {
	plan, err := pathstudy.Fixture(pathplan.BranchLayout, 51, "Keep branches.")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"--plan", file}, &output); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Selection  pathplan.Selection `json:"selection"`
		GoooSource string             `json:"gooo_source"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Selection.ModelCalls != 0 || result.Selection.ExternalCalls != 0 || result.GoooSource == "" || result.Selection.Choices["structure"] != "layout_forward" {
		t.Fatal("wrong offline CLI result")
	}
	if err := run([]string{"--plan", file, "--seed", "sample"}, &output); err == nil {
		t.Fatal("offline seeded selection accepted")
	}
	testsFile := filepath.Join(t.TempDir(), "tests.json")
	expected, err := pathstudy.Oracle(pathplan.BranchLayout, true, 51, 0)
	if err != nil {
		t.Fatal(err)
	}
	tests, err := json.Marshal(struct {
		Schema string              `json:"schema"`
		Cases  []pathplan.TestCase `json:"cases"`
	}{"gooo/typed-path-finite-tests/v1", []pathplan.TestCase{{Input: 0, Expected: expected}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testsFile, tests, 0600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := run([]string{"--plan", file, "--tests", testsFile, "--max-attempts", "2"}, &output); err != nil {
		t.Fatal(err)
	}
	var tdd struct {
		Search pathplan.SearchResult `json:"search"`
	}
	if err := json.Unmarshal(output.Bytes(), &tdd); err != nil || tdd.Search.Status != "TRAINING_COMPLETE" || tdd.Search.Selection.ModelCalls != 0 || tdd.Search.Selection.Choices["structure"] != "layout_reverse" {
		t.Fatalf("wrong finite TDD result: %+v %v", tdd, err)
	}
	if err := run([]string{"--plan", file, "--tests", testsFile, "--max-attempts", "65"}, &output); err == nil {
		t.Fatal("unbounded CLI search accepted")
	}
	if err := os.WriteFile(file, append([]byte(`{"schema":"duplicate",`), raw[1:]...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--plan", file}, &output); err == nil {
		t.Fatal("duplicate JSON accepted")
	}
}
