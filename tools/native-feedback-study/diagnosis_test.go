package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestDiagnosisOracleRejectsTamperedWitnessesAndAccounting(t *testing.T) {
	rows, err := compoundstudy.Cohort()
	if err != nil {
		t.Fatal(err)
	}
	row := rows[1]
	if row.Contract != "sparse" {
		t.Fatal("sparse contract required")
	}
	prepared, err := pathplan.Prepare(row.Document.Plan)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	search, _, err := prepared.Search(ctx, nil, row.Document.Cases, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	diagnosis, err := prepared.Diagnose(ctx, search.Selection.Choices, row.Document.Cases, diagnosisProbes(), 4)
	if err != nil {
		t.Fatal(err)
	}
	capture := diagnosisCapture{ID: row.ID, Arm: "offline", Search: search, Diagnosis: diagnosis, SearchNS: 1, DiagnosisNS: 1}
	if err = validateDiagnosisCapture(row, capture); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(capture)
	for _, change := range []string{"witness", "passed", "vector", "reference", "observed", "result", "selected", "calls", "model", "status"} {
		var altered diagnosisCapture
		if err = json.Unmarshal(raw, &altered); err != nil {
			t.Fatal(err)
		}
		switch change {
		case "witness":
			found := false
			for _, c := range altered.Diagnosis.Candidates {
				if c.Witness != nil {
					c.Witness.Alternative++
					found = true
					break
				}
			}
			if !found {
				t.Fatal("expected observed ambiguous alternative")
			}
		case "passed":
			altered.Diagnosis.Candidates[0].Passed++
		case "vector":
			altered.Diagnosis.Candidates[0].CaseOutputsSHA256 = "forged"
		case "reference":
			altered.Diagnosis.ReferenceMask ^= 1
		case "observed":
			altered.Diagnosis.Observed--
		case "result":
			altered.Search.Attempts[0].Results[0].Actual++
		case "selected":
			altered.Search.SelectedTrainingPassed++
		case "calls":
			altered.Diagnosis.ModelPredictions++
		case "model":
			altered.Search.Selection.MetadataSHA256 = diagnosisModelPin
		case "status":
			altered.Search.Status = "PARTIAL"
		}
		if err = validateDiagnosisCapture(row, altered); err == nil {
			t.Fatalf("accepted altered %s", change)
		}
	}
}

func TestFrozenPathDiagnosisReplaysWithoutModelOrExecution(t *testing.T) {
	t.Chdir("../..")
	value, err := auditDiagnosisStudy("runs/path-diagnosis-sdk-20261001", "a2b7c0c6960888d93f408f0f45af698c579d68cf")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := read("runs/path-diagnosis-sdk-20261001/report.json")
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.MarshalIndent(value, "", "  ")
	if !bytes.Equal(append(encoded, '\n'), raw) {
		t.Fatal("frozen diagnosis report differs")
	}
}
