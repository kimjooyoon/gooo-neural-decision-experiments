package main

import (
	"encoding/json"
	"testing"
)

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
