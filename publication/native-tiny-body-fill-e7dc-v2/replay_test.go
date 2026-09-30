package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReplayKeepsSixCasesWhenEvidenceDirectoryCannotBeCreated(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "missing-parent", "replay")
	cell := replayCell("/missing/captures", "/missing/fixtures", "/missing/go", "source", "binary", "go", outputDir, "arithmetic-fp32")
	if cell.PlannedCases != frozenCasesPerCell {
		t.Fatalf("failed cell planned_cases = %d, want %d", cell.PlannedCases, frozenCasesPerCell)
	}
	if cell.Status != "PARTIAL" || len(cell.Errors) == 0 {
		t.Fatalf("mkdir failure not retained as partial evidence: %+v", cell)
	}
}

func TestReplayTopDenominatorIsFixedAtEightBySix(t *testing.T) {
	if frozenCellCount != 8 || frozenCasesPerCell != 6 || frozenCasesTotal != 48 {
		t.Fatalf("unexpected frozen dimensions: cells=%d cases_per_cell=%d total=%d", frozenCellCount, frozenCasesPerCell, frozenCasesTotal)
	}
}

func TestReplayModelPinsRejectAlternateBundles(t *testing.T) {
	for variant, pin := range frozenModelPins {
		if err := requireFrozenModelPins(variant, pin.metadataSHA256, pin.weightsSHA256); err != nil {
			t.Errorf("exact pins rejected for %s: %v", variant, err)
		}
		if err := requireFrozenModelPins(variant, pin.metadataSHA256, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); err == nil {
			t.Errorf("alternate weights accepted for %s", variant)
		}
	}
	if err := requireFrozenModelPins("unknown", "", ""); err == nil {
		t.Fatal("unknown model variant accepted")
	}
}

func TestReplayModelPinsMatchFrozenRunPlan(t *testing.T) {
	if err := verifyFrozenRunPlan("run-plan.json"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("run-plan.json")
	if err != nil {
		t.Fatal(err)
	}
	var plan struct {
		Models map[string]struct {
			MetadataSHA256 string `json:"metadata_sha256"`
			WeightsSHA256  string `json:"weights_sha256"`
		} `json:"models"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Models) != len(frozenModelPins) {
		t.Fatalf("run-plan model count = %d, replay pin count = %d", len(plan.Models), len(frozenModelPins))
	}
	for variant, pin := range frozenModelPins {
		declared, ok := plan.Models[variant]
		if !ok || declared.MetadataSHA256 != pin.metadataSHA256 || declared.WeightsSHA256 != pin.weightsSHA256 {
			t.Errorf("replay pins for %s do not match run-plan.json", variant)
		}
	}
}

func TestReplayRejectsChangedRunPlanBytes(t *testing.T) {
	data, err := os.ReadFile("run-plan.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "run-plan.json")
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyFrozenRunPlan(path); err == nil {
		t.Fatal("modified run plan was accepted")
	}
}
