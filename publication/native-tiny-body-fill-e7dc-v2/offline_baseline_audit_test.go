package main

import (
	"path/filepath"
	"testing"
)

func TestOfflineRawRowsPassContractPreflightWithoutCompiling(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, cellID := range []string{"arithmetic-offline_baseline", "boolean-offline_baseline"} {
		rows := preflightOfflineCell(filepath.Join(root, "results"), filepath.Join(root, "fixtures"), cellID)
		if len(rows) != 1 {
			t.Fatalf("%s preflight rows = %d, want 1", cellID, len(rows))
		}
		if got := rows[0].Preflight.Errors; len(got) != 0 {
			t.Errorf("%s failed contract preflight: %v", cellID, got)
		}
		if rows[0].Preflight.LocalPredictions != 0 || rows[0].Preflight.ExternalProviderCalls != 0 || !rows[0].Preflight.ExternalProviderKnown {
			t.Errorf("%s does not prove known zero provider activity", cellID)
		}
		if rows[0].Preflight.LayaObservation.Status != "UNKNOWN" || rows[0].Preflight.LayaObservation.Numerator != 0 || rows[0].Preflight.LayaObservation.Denominator != 1 {
			t.Errorf("%s Laya observation is not UNKNOWN 0/1", cellID)
		}
	}
}

func TestOfflineKnownErrorAllowlistDoesNotAcceptAdditionalFailures(t *testing.T) {
	mutated := append(append([]string(nil), knownCaptureErrors...), "unexpected validation error")
	if sameStrings(mutated, knownCaptureErrors) {
		t.Fatal("additional capture validation error was accepted")
	}
	if sameStrings([]string{knownCaptureErrors[1], knownCaptureErrors[0]}, knownCaptureErrors) {
		t.Fatal("reordered capture validation errors were accepted")
	}
}
