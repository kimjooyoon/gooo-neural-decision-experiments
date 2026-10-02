package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are fabricated unit receipts, not evidence of optimizer or model calls.
func fixtureStage(t *testing.T) (string, stageReport) {
	t.Helper()
	directory := t.TempDir()
	r := stageReport{Steps: 800, Epochs: 100, Groups: 1024, Rows: 2048, InitialSHA: strings.Repeat("a", 64), SelectedSHA: strings.Repeat("b", 64), Selected: 1, Temperature: .5, NLL: 1, Wall: 1, CPU: 1, CPUPercent: 100, RSS: 1, MPS: 1, Driver: 1}
	f, err := os.Create(filepath.Join(directory, "optimizer-updates.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range 800 {
		raw, err := json.Marshal(map[string]any{"arm": "uniform-initial", "stage": "fp32", "epoch": i/8 + 1, "batch": i%8 + 1, "function_groups": 128, "actual_stage_updates": i + 1, "objective": 1.})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write(append(raw, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	for i := range 100 {
		e := epoch{i + 1, (i + 1) * 8, 1, map[string]float64{"0.5": 1, "1.0": 1, "2.0": 1, "4.0": 1}}
		r.History = append(r.History, e)
		raw, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directory, "epoch-"+pad3(i+1)+".json"), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return directory, r
}

func TestJointEpochTemperatureTieAndExactBudget(t *testing.T) {
	directory, r := fixtureStage(t)
	if err := verifyStage(directory, "uniform-initial", "fp32", r); err != nil {
		t.Fatal(err)
	}
	r.Selected = 2
	if err := verifyStage(directory, "uniform-initial", "fp32", r); err == nil {
		t.Fatal("later tied epoch selected")
	}
	r.Selected = 1
	r.Temperature = 1
	if err := verifyStage(directory, "uniform-initial", "fp32", r); err == nil {
		t.Fatal("larger tied temperature selected")
	}
	r.Temperature = .5
	r.Steps = 799
	if err := verifyStage(directory, "uniform-initial", "fp32", r); err == nil {
		t.Fatal("missing update accepted")
	}
}

func TestStageRejectsModifiedEpochAndExtraFile(t *testing.T) {
	directory, r := fixtureStage(t)
	if err := os.WriteFile(filepath.Join(directory, "extra.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyStage(directory, "uniform-initial", "fp32", r); err == nil {
		t.Fatal("extra receipt accepted")
	}
	if err := os.Remove(filepath.Join(directory, "extra.json")); err != nil {
		t.Fatal(err)
	}
	r.History[0].Train = 2
	if err := verifyStage(directory, "uniform-initial", "fp32", r); err == nil {
		t.Fatal("modified recorded objective accepted")
	}
}
