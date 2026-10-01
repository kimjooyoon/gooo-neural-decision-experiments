package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestBudgetWireKeepsEvaluatorOnlyInformationOutOfSelection(t *testing.T) {
	t.Chdir("../..")
	arms, err := compoundArms()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	calls := 0
	for i, arm := range arms {
		id := budgetArmID(arm)
		if ids[id] {
			t.Fatal("duplicate arm")
		}
		ids[id] = true
		rows, err := budgetRows(i)
		if err != nil {
			t.Fatal(err)
		}
		for _, budget := range budgetOrder(i) {
			for _, row := range rows {
				raw, err := budgetWire(row, arm, budget, pathplan.CIHint{SourceSHA: budgetNative, Status: "UNKNOWN"}, row.ID, false)
				var request retainedRequest
				if err != nil || json.Unmarshal(raw, &request) != nil || request.Document.Max != budget || request.Options.Step != 1 || len(request.Document.Cases) != len(row.Document.Cases) {
					t.Fatal("budget wire differs")
				}
				for _, key := range []string{"original_intention_mask", "finite_best_masks", "finite_best_passed", "separate_input_cases"} {
					if bytes.Contains(raw, []byte(key)) {
						t.Fatal("evaluation information leaked", key)
					}
				}
				if arm.Feedback != (request.Options.CI != nil) {
					t.Fatal("feedback mode differs")
				}
				calls++
			}
		}
	}
	if len(ids) != 9 || calls != 1944 {
		t.Fatal("wrong study inventory")
	}
	rows, _ := budgetRows(0)
	if _, err := budgetWire(rows[0], arms[0], 3, pathplan.CIHint{}, "bad", false); err == nil {
		t.Fatal("undeclared budget accepted")
	}
}

func TestBudgetPrefixValidationRejectsBudgetDependentSearch(t *testing.T) {
	c := budgetCollection{}
	for _, b := range []int{1, 2, 4} {
		seq := []uint16{0, 1, 2, 3}
		curve := []float64{0, 50, 75, 100}
		c.observations = append(c.observations, budgetObservation{ID: "offline-budget-" + fmtInt(b) + "-case", Arm: "offline", Case: "case", Budget: b, Sequence: seq[:b], Curve: curve[:b]})
	}
	if err := validateBudgetPrefixes(c); err != nil {
		t.Fatal(err)
	}
	c.observations[1].Sequence = []uint16{2, 1}
	if err := validateBudgetPrefixes(c); err == nil {
		t.Fatal("budget-dependent first candidate accepted")
	}
}

func TestPartialTraceCheckPreservesOriginalMaximumRequirement(t *testing.T) {
	t.Chdir("../..")
	rows, err := nativeUnfixedRows()
	if err != nil {
		t.Fatal(err)
	}
	arms, err := retainedArms()
	if err != nil {
		t.Fatal(err)
	}
	arm := arms[1]
	raw, err := read(filepath.Join("runs/unfixed-feedback-sdk-20261001/captures", rows[0].ID+"-"+arm.Name+"-true.json"))
	var c unfixedCapture
	if err != nil || json.Unmarshal(raw, &c) != nil {
		t.Fatal("frozen capture missing")
	}
	row := rows[0]
	row.FiniteBestPassed++
	ci := pathplan.CIHint{SourceSHA: compoundNative, Status: "PASS"}
	if _, err := verifyUnfixedWithCI(c, row, arm, ci); err == nil {
		t.Fatal("original global maximum check weakened")
	}
	if _, err := verifyBoundedTrace(c, row, arm, ci, false); err != nil {
		t.Fatal("best-so-far trace rejected", err)
	}
}
