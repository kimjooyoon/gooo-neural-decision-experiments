package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFixedPilotRecomputesCandidatesAndObservedDenominators(t *testing.T) {
	t.Chdir("../..")
	value, err := audit("runs/feedback-path-pilot-fixed-20261001")
	if err != nil {
		t.Fatal(err)
	}
	if value["actual_model_predictions"] != 246 || value["feedback_model_predictions"] != 102 || value["same_emitted_go_pairs"] != 16 || value["executed_finite_passed"] != 208 || value["executed_disjoint_input_passed"] != 288 {
		t.Fatal("measured denominators differ")
	}
}
func TestAuditRejectsChangedFeedbackOrExecutedResult(t *testing.T) {
	t.Chdir("../..")
	copyStudy := func() string {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "captures"), 0700); err != nil {
			t.Fatal(err)
		}
		for _, relative := range []string{"report.json", "captures"} {
			if relative == "report.json" {
				raw, err := read("runs/feedback-path-pilot-fixed-20261001/report.json")
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, relative), raw, 0600); err != nil {
					t.Fatal(err)
				}
				continue
			}
			entries, err := os.ReadDir("runs/feedback-path-pilot-fixed-20261001/captures")
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				raw, err := read(filepath.Join("runs/feedback-path-pilot-fixed-20261001/captures", entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, "captures", entry.Name()), raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		return dir
	}
	dir := copyStudy()
	var report map[string]json.RawMessage
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	var policies []policy
	if err = json.Unmarshal(report["policies"], &policies); err != nil {
		t.Fatal(err)
	}
	for i := range policies {
		if len(policies[i].Feedback) > 0 {
			policies[i].Feedback[0].ModelCalls++
			break
		}
	}
	report["policies"], _ = json.Marshal(policies)
	raw, _ = json.Marshal(report)
	if err = os.WriteFile(filepath.Join(dir, "report.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = audit(dir); err == nil {
		t.Fatal("tampered feedback calls accepted")
	}
	dir = copyStudy()
	if err = os.WriteFile(filepath.Join(dir, "captures", "en-complete-offline-false.executed.json"), []byte(`[0]`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = audit(dir); err == nil {
		t.Fatal("tampered executed Go output accepted")
	}
}
