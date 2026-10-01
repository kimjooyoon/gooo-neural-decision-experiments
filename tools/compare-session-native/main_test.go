package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

func TestAuditRejectsTamperedOrRepeatedProgress(t *testing.T) {
	raw, err := os.ReadFile("../../studies/conditional-paths-v1/cohort/en-budget-64.json")
	if err != nil {
		t.Fatal(err)
	}
	var document pathDocument
	if err := strictjson.Decode(raw, &document); err != nil {
		t.Fatal(err)
	}
	document.TestCases[6].Expected = 999
	prepared, err := pathplan.Prepare(document.Plan)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	search, _, progress, err := prepared.SearchBatches(ctx, nil, document.TestCases, 64, 8, "")
	if err != nil || checkProgress(progress, search, 0) != nil {
		t.Fatal("valid partial trace failed", err)
	}
	copy := append([]pathplan.SessionProgress(nil), progress...)
	copy[2].PreviousSHA = "tampered"
	if checkProgress(copy, search, 0) == nil {
		t.Fatal("tampered observation link was accepted")
	}
	copy = append([]pathplan.SessionProgress(nil), progress...)
	copy[2].Attempted++
	if checkProgress(copy, search, 0) == nil {
		t.Fatal("tampered cumulative count was accepted")
	}
	copy = append([]pathplan.SessionProgress(nil), progress...)
	copy[1].PredictionsThisAdvance = 6
	if checkProgress(copy, search, 0) == nil {
		t.Fatal("hidden repeated model calls were accepted")
	}
}

func TestLimitedProcessOutputDoesNotGrowBeyondBudget(t *testing.T) {
	var output limited
	if _, err := output.Write(make([]byte, 2<<20)); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("x")); err == nil || output.Len() != 2<<20 {
		t.Fatal("output budget exceeded")
	}
}
