package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFrozenPathDatasetGrouping(t *testing.T) {
	path := filepath.Join("..", "..", "data", "typed-path-v1", "dataset.jsonl")
	rows, sha, manifestSHA, err := loadRows(path)
	if err != nil || len(rows) != 960 || len(sha) != 64 || len(manifestSHA) != 64 {
		t.Fatalf("frozen rows: %d %v", len(rows), err)
	}
	instructions, programs := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		instructions[r.InstructionID] = true
		programs[r.ProgramID] = true
	}
	if len(instructions) != 320 || len(programs) != 160 {
		t.Fatalf("group counts: %d %d", len(instructions), len(programs))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dataset.jsonl"), raw[:len(raw)-2], 0600); err != nil {
		t.Fatal(err)
	}
	meta := map[string]any{"schema": "gooo/typed-path-curriculum/v1", "dataset_sha256": sha, "total_rows": 6240}
	manifestRaw, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifestRaw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := loadRows(filepath.Join(dir, "dataset.jsonl")); err == nil {
		t.Fatal("dataset byte drift accepted")
	}
}
