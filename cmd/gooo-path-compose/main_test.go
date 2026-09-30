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
	if err := os.WriteFile(file, append([]byte(`{"schema":"duplicate",`), raw[1:]...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--plan", file}, &output); err == nil {
		t.Fatal("duplicate JSON accepted")
	}
}
