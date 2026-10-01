package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestRecordedAuditRejectsChangedCountsAndExecutedGo(t *testing.T) {
	t.Chdir("../..")
	const original = "runs/native-feedback-integration-fixed-20261001"
	if _, err := audit(original, "studies/native-feedback-v1", "", false); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := filepath.WalkDir(original, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(original, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(dir, rel), 0700)
		}
		raw, err := read(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, rel), raw, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := read(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value report
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	value.Observations[0].Calls++
	if err := save(filepath.Join(dir, "report.json"), value); err != nil {
		t.Fatal(err)
	}
	if _, err := audit(dir, "studies/native-feedback-v1", "", false); err == nil {
		t.Fatal("changed local prediction observation accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "executions"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "executions", entries[0].Name())
	raw, err = read(path)
	if err != nil {
		t.Fatal(err)
	}
	var values []int64
	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	values[0]++
	if err := save(path, values); err != nil {
		t.Fatal(err)
	}
	if _, err := audit(dir, "studies/native-feedback-v1", "", false); err == nil {
		t.Fatal("changed generated Go execution accepted")
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

func TestContinuedConstructionAuditRejectsChangedCaptures(t *testing.T) {
	t.Chdir("../..")
	const original = "runs/feedback-context-decline-20261001"
	if _, err := auditContinuedBound(original); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	entries, err := os.ReadDir(original)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		raw, err := read(filepath.Join(original, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := auditContinuedBound(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "en-feedback.json")
	raw, err := read(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, ' '), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := auditContinuedBound(dir); err == nil {
		t.Fatal("changed continued-construction capture accepted")
	}
}
