package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceFormsRetainSnapshotsAndOnlyChangeDeclaredDimensions(t *testing.T) {
	raw := "computes `return input0` assembling { alternative \"copy.title + \\\":\\\" + copy.state\" value_case \"한글\" }"
	forms, err := sourceForms(raw)
	if err != nil || !strings.Contains(forms[0], "saved.reason + old") || !strings.Contains(forms[1], "saved.state") ||
		strings.Contains(forms[2], "copy") || !strings.Contains(forms[3], "변경") {
		t.Fatal(forms, err)
	}
	if _, err = sourceForms("no source assembly"); err == nil {
		t.Fatal("incompatible source accepted")
	}
}

func TestProbeChecksRejectMissingAndFailedObservations(t *testing.T) {
	if probeChecks(nil)["eight_exports_required"] {
		t.Fatal("absent exports passed")
	}
	rows := make([]observation, 8)
	rows[0].Error = "retained failure"
	if probeChecks(rows)["zero_predictions_and_candidate_executions"] {
		t.Fatal("failed export was accepted")
	}
}

func TestProbePreservesFailedExportsAndDoesNotOverwriteOutput(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "fixture.gooo")
	raw := "computes `return input0` assembling { alternative \"copy.title + \\\":\\\" + copy.state\" value_case \"한글\" }"
	if err := os.WriteFile(source, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "observations")
	if run(filepath.Join(root, "missing-compiler"), source, output) == nil {
		t.Fatal("failed command passed")
	}
	data, err := os.ReadFile(filepath.Join(output, "observations.json"))
	if err != nil || !strings.Contains(string(data), `"error"`) {
		t.Fatal("failed exports not retained", err)
	}
	if run("anything", source, output) == nil {
		t.Fatal("existing observations overwritten")
	}
}
