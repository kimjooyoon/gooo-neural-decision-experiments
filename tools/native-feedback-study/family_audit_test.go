package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

func TestOptimizedFamilyRejectsMissingReceiptAndHiddenCall(t *testing.T) {
	t.Chdir("../..")
	rows, err := familyRows()
	if err != nil {
		t.Fatal(err)
	}
	var rowIndex int
	for i, r := range rows {
		if r.ID == "assignment_target-64-false-en-contradictory" {
			rowIndex = i
			break
		}
	}
	row := rows[rowIndex]
	raw, err := read("runs/feedback-family-optimized-feature-20261001/captures/" + row.ID + "-parent_fp32-true.json")
	if err != nil {
		t.Fatal(err)
	}
	model, err := decision.LoadPath("runs/typed-path-positioned-random-20261001/models/fp32/model.json")
	if err != nil {
		t.Fatal(err)
	}
	arm := familyArm{Name: "parent_fp32", Path: "model", Feedback: true, Metadata: model.MetadataSHA256(), Weights: model.WeightsSHA256()}
	spec := optimizedFamilies["0346231d22f540a844fc9296b408a64c59bb33d538223603c10f65a281c63d62"].Spec
	var value nativeResult
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if _, err = inspectFamilyFor(value, row, arm, spec); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*nativeResult){
		func(v *nativeResult) { v.Report.Paths.Feedback = nil },
		func(v *nativeResult) { v.Report.Paths.Feedback[0].ModelCalls = 1 },
		func(v *nativeResult) { v.Report.Paths.Feedback[0].RankingUnnecessary = false },
		func(v *nativeResult) { v.Report.Paths.Feedback[0].FirstFailure.Expected++ },
	} {
		if err = json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		mutate(&value)
		if len(value.Report.Paths.Feedback) > 0 {
			f := &value.Report.Paths.Feedback[0]
			f.SHA = ""
			encoded, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			f.SHA = hash(encoded)
		}
		if _, err = inspectFamilyFor(value, row, arm, spec); err == nil {
			t.Fatal("changed sole-path receipt accepted")
		}
	}
}

func TestFamilyUnknownCIContextDoesNotImplyPass(t *testing.T) {
	spec := legacyFamily
	spec.CIStatus = "UNKNOWN"
	hint := familyCIHint(spec)
	if hint.Status != "UNKNOWN" || hint.SourceSHA != familyNativeSHA || hint.Validate() != nil {
		t.Fatal("unknown context changed into PASS")
	}
	spec.CIStatus = "passed"
	hint = familyCIHint(spec)
	if hint.Validate() == nil {
		t.Fatal("non-enumerated caller context accepted")
	}
}

func TestFamilyFrozenAuditAndChangedPilotCapture(t *testing.T) {
	t.Chdir("../..")
	for _, name := range []string{"feedback-family-pilot-20261001", "feedback-family-matrix-20261001", "feedback-family-optimized-feature-20261001"} {
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
