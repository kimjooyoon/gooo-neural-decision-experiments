package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestDocumentEnvelopeRetainsNativeSchema(t *testing.T) {
	var d document
	if err := json.Unmarshal([]byte(`{"schema":"gooo/body-codegen-typed-path-plan/v1","max_attempts":64}`), &d); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(d)
	if err != nil || !strings.Contains(string(raw), `"schema":"gooo/body-codegen-typed-path-plan/v1"`) {
		t.Fatal("study lost the compiler document schema")
	}
}

func inspectionFixture() nativeResult {
	var value nativeResult
	r := &value.Report
	r.Decision, r.Compiler, r.Types, r.Replay = "PASS", strings.Repeat("a", 40), true, true
	p := &r.Paths
	p.Bound, p.Completeness = true, 100
	p.Search.TrainingTotal, p.Search.SelectedTrainingPassed, p.Search.Evaluated = 7, 7, 1
	for range 7 {
		p.Cases = append(p.Cases, struct {
			Input, Expected, Actual int64
			Passed                  bool
		}{Passed: true})
	}
	p.Progress = []pathplan.SessionProgress{{Sequence: 1, SHA: "first"},
		{Sequence: 2, PreviousSHA: "first", SHA: "second", Attempted: 1,
			NewAttempts: []pathplan.SearchAttempt{{Mask: 0}}}}
	return value
}

func TestNativeInspectionRejectsRepeatedCallsAndOutcomes(t *testing.T) {
	valid := inspectionFixture()
	if err := inspect(valid, false, 0, strings.Repeat("a", 40)); err != nil {
		t.Fatal(err)
	}
	for _, alter := range []func(*nativeResult){
		func(v *nativeResult) { v.Report.Writes = 1 },
		func(v *nativeResult) { v.Report.Compiler = "wrong" },
		func(v *nativeResult) { v.Report.Paths.Cases[0].Actual = 1 },
		func(v *nativeResult) { v.Report.Paths.Search.Selection.ModelCalls = 1 },
		func(v *nativeResult) { v.Report.Paths.Progress[1].PredictionsThisAdvance = 1 },
		func(v *nativeResult) {
			v.Report.Paths.Progress[1].NewAttempts = append(v.Report.Paths.Progress[1].NewAttempts, pathplan.SearchAttempt{Mask: 0})
		},
		func(v *nativeResult) {
			v.Report.Paths.Feedback = []pathplan.FeedbackReceipt{{Round: 1, Applied: true, CIIsAuthority: true}}
		},
	} {
		value := inspectionFixture()
		alter(&value)
		if err := inspect(value, false, 0, strings.Repeat("a", 40)); err == nil {
			t.Fatal("invalid native observation accepted")
		}
	}
}

func TestBoundedOutputAndProcessCancellation(t *testing.T) {
	var out bounded
	if _, err := out.Write(make([]byte, (2<<20)+1)); err == nil || out.Len() != 0 {
		t.Fatal("oversized output retained")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, _, err := child(ctx, t.TempDir(), "/bin/sh", "-c", "sleep 10")
	if err == nil || time.Since(started) > 3*time.Second {
		t.Fatal("canceled child retained a live process group or output wait")
	}
}
