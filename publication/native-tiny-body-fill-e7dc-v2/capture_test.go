package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFrozenRunPlanPinsMatchDeclaredModelBundles(t *testing.T) {
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
		t.Fatalf("run-plan model count = %d, pin count = %d", len(plan.Models), len(frozenModelPins))
	}
	for variant, pin := range frozenModelPins {
		declared, ok := plan.Models[variant]
		if !ok || declared.MetadataSHA256 != pin.metadataSHA256 || declared.WeightsSHA256 != pin.weightsSHA256 {
			t.Errorf("frozen pins for %s do not match run-plan.json", variant)
		}
		if err := requireFrozenModelPins(variant, pin.metadataSHA256, pin.weightsSHA256); err != nil {
			t.Errorf("exact pins rejected for %s: %v", variant, err)
		}
		if err := requireFrozenModelPins(variant, pin.metadataSHA256, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
			t.Errorf("alternate weights accepted for %s", variant)
		}
		if err := requireFrozenModelPins(variant, "0000000000000000000000000000000000000000000000000000000000000000", pin.weightsSHA256); err == nil {
			t.Errorf("alternate metadata accepted for %s", variant)
		}
	}
}

func TestCaptureRejectsChangedRunPlanBytes(t *testing.T) {
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
