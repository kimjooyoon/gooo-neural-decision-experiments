package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFamilyFrozenAuditAndChangedPilotCapture(t *testing.T) {
	t.Chdir("../..")
	for _, name := range []string{"feedback-family-pilot-20261001", "feedback-family-matrix-20261001"} {
		value, err := auditFamily(filepath.Join("runs", name))
		if err != nil || value["new_model_predictions"] != 0 {
			t.Fatalf("%s %v", name, err)
		}
	}
	if _, err := auditFamily(t.TempDir()); err == nil {
		t.Fatal("missing frozen record accepted")
	}
	const original = "runs/feedback-family-pilot-20261001"
	dir := t.TempDir()
	for _, name := range []string{"report.json", "preexecution.json"} {
		raw, err := read(filepath.Join(original, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "captures"), 0700); err != nil {
		t.Fatal(err)
	}
	name := "assignment_target-64-false-en-complete-offline-false.json"
	raw, err := read(filepath.Join(original, "captures", name))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "captures", name), append(raw, ' '), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := auditFamily(dir); err == nil {
		t.Fatal("changed pilot capture accepted")
	}
}
