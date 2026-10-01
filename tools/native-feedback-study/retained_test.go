package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestRetainedFrozenNativeReplaysWithoutProcessesOrPredictions(t *testing.T) {
	t.Chdir("../..")
	root := "runs/retained-native-feature-pilot-20261001"
	value, err := auditRetained(root, retainedFeatureRunner, retainedFeatureNative)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := read(filepath.Join(root, "report.json"))
	if err != nil || hash(expected) != retainedFeatureReport || !bytes.Equal(append(raw, '\n'), expected) {
		t.Fatal("frozen audit differs")
	}
	summary, err := retainedMetrics(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	expected, err = read(filepath.Join(root, "metrics-summary.json"))
	if err != nil || !bytes.Equal(append(raw, '\n'), expected) {
		t.Fatal("cost arithmetic differs")
	}
	if _, err := collectRetained(root, retainedFeatureRunner, compoundNative); err == nil {
		t.Fatal("other compiler accepted")
	}
}

func TestRetainedPlanReusesViewsWithReverseSecondPass(t *testing.T) {
	t.Chdir("../..")
	rows, err := retainedRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 12 {
		t.Fatal("wrong construction denominator")
	}
	for i := range 6 {
		if rows[i].ID != rows[11-i].ID {
			t.Fatal("reverse second pass differs")
		}
	}
	arms, err := retainedArms()
	if err != nil || len(arms) != 5 {
		t.Fatal("frozen arms differ", err)
	}
	for _, arm := range arms {
		raw, err := retainedWire(rows[0], arm, familyCIHint(familySpec{Revision: compoundNative, CIStatus: "UNKNOWN"}), "case", false)
		var request retainedRequest
		if err != nil || json.Unmarshal(raw, &request) != nil || request.Source != rows[0].Source || request.Options.Step != 1 {
			t.Fatal("request differs")
		}
		if arm.Feedback && (request.Options.Rounds != 3 || !request.Options.Unfixed || request.Options.CI.Status != "UNKNOWN") {
			t.Fatal("feedback context differs")
		}
		if !arm.Feedback && (request.Options.Rounds != 0 || request.Options.CI != nil) {
			t.Fatal("offline unexpectedly feeds back")
		}
	}
}

func TestRetainedAuditRejectsAbsentSourcePins(t *testing.T) {
	if _, err := collectRetained("missing", "", "native"); err == nil {
		t.Fatal("missing pins accepted")
	}
	if validRetainedMetrics(retainedProcess{}) {
		t.Fatal("absent measurements accepted")
	}
}

func TestRetainedCompletenessDistinguishesDeclaredAndAttainableScores(t *testing.T) {
	t.Chdir("../..")
	root := "runs/retained-native-feature-pilot-20261001"
	value, err := retainedCompleteness(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := read(filepath.Join(root, "completeness-summary.json"))
	if err != nil || hash(expected) != "8befd5ef343abd3d1194e39f960a9242e4d78ef701a89a3b60170c63886f2ea0" || !bytes.Equal(append(raw, '\n'), expected) {
		t.Fatal("completeness derivation differs")
	}
	for _, curve := range value["observations"].([]completionCurve) {
		if curve.DeclaredCases != 8 || curve.MaximumPassed != 7 || curve.AttainablePercent != 100 || curve.GapPercent != 12.5 || len(curve.Curve) != 4 {
			t.Fatal("completeness denominators collapsed")
		}
	}
	if err := saveRetainedCompleteness(root, filepath.Join(root, "report.json")); err == nil {
		t.Fatal("original report overwrite allowed")
	}
}
