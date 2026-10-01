package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestCompoundFrozenCohortRetainsFourPathSemantics(t *testing.T) {
	t.Chdir("../..")
	rows, err := compoundRows()
	if err != nil || len(rows) != 72 {
		t.Fatal("frozen authored compound cohort differs", err)
	}
	for _, r := range rows {
		if _, err := inspectCompound(nativeResult{}, r, familyArm{}); err == nil {
			t.Fatal("unverified native result accepted")
		}
	}
}

func TestCompoundProjectionAllowsOnlyEquivalentLiteralWrappers(t *testing.T) {
	original := "package p;func ComposePaths(input int64) int64{return input + 4}"
	expected, err := compoundFunction(original)
	if err != nil {
		t.Fatal(err)
	}
	equivalent, err := compoundFunction(strings.Replace(original, "+ 4", "+ int64(4)", 1))
	if err != nil || equivalent != expected {
		t.Fatal("equivalent literal wrapper differs", err)
	}
	for _, replacement := range []string{"+ 5", "+ int32(4)", "+ int64(input)", "+ int64(9223372036854775808)"} {
		actual, err := compoundFunction(strings.Replace(original, "+ 4", replacement, 1))
		if err == nil && actual == expected {
			t.Fatal("changed expression accepted", replacement)
		}
	}
	if _, err := compoundFunction(original + ";func init(){}"); err == nil {
		t.Fatal("extra executable declaration accepted")
	}
}
func TestCompoundPilotAndMatrixReplayWithoutInference(t *testing.T) {
	t.Chdir("../..")
	for _, root := range []string{"runs/compound-path-pilot-20261001", "runs/compound-path-main-20261001"} {
		value, err := auditCompound(root)
		if err != nil || value["new_model_predictions"] != 0 {
			t.Fatal(root, err)
		}
	}
	rows, _ := compoundRows()
	r := rows[0]
	raw, err := read("runs/compound-path-pilot-20261001/captures/" + r.ID + "-offline-false.json")
	if err != nil {
		t.Fatal(err)
	}
	var v nativeResult
	if json.Unmarshal(raw, &v) != nil {
		t.Fatal("pilot malformed")
	}
	if _, err := inspectCompound(v, r, familyArm{Name: "offline"}); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*nativeResult){
		func(v *nativeResult) { v.Source = strings.Replace(v.Source, "return (first - second)", "return 0", 1) },
		func(v *nativeResult) { v.Report.ActivityID = "unrelated://activity" },
		func(v *nativeResult) { v.Report.Paths.Search.Selection.ModelCalls++ },
		func(v *nativeResult) { v.Report.Paths.Search.Attempts[0].Mask = 3 },
	} {
		if json.Unmarshal(raw, &v) != nil {
			t.Fatal("pilot malformed")
		}
		mutate(&v)
		if _, err := inspectCompound(v, r, familyArm{Name: "offline"}); err == nil {
			t.Fatal("changed compound observation accepted")
		}
	}
}
func TestFixedCoordinatesAreDistinctFromSoleRemainingPath(t *testing.T) {
	search := pathplan.SearchResult{Attempts: []pathplan.SearchAttempt{{Mask: 0}, {Mask: 1}, {Mask: 2}}}
	receipts := []pathplan.FeedbackReceipt{{Attempted: 1, ModelCalls: 2}, {Attempted: 2, ModelCalls: 2}, {Attempted: 3, ModelCalls: 0}}
	if fixedCompoundFeedback(search, receipts) != 1 {
		t.Fatal("fixed coordinate accounting differs")
	}
}
