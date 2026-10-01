package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestNativeDiagnosisInspectionRejectsFalseWitnessAndPrediction(t *testing.T) {
	t.Chdir("../..")
	doc, err := nativeDiagnosisDocument("ko")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	search, body, err := prepared.Search(ctx, nil, doc.Cases, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	diagnosis, err := prepared.Diagnose(ctx, search.Selection.Choices, doc.Cases, []int64{2, 3}, 2)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"source": body.GoSource(), "report": map[string]any{
		"compiler_source_sha": "test-source", "decision": "PASS", "typecheck_passed": true, "deterministic_replay": true,
		"body_paths": map[string]any{"source_base_matched": true, "search": search, "finite_functional_completeness_percent": 100,
			"native_case_results": []map[string]any{{"input": 2, "expected": 0, "actual": 0, "passed": true}},
			"diagnosis":           diagnosis, "diagnosis_budget": 2, "diagnosis_options_sha256": "sha256:" + hash([]byte(`{"inputs":[2,3],"max_candidates":2}`)),
			"timing": map[string]any{"diagnosis_ms": 0.1}}}})
	var value nativeResult
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if _, err = inspectNativeDiagnosis(value, "ko", "offline", "test-source", true); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*nativeResult){
		func(v *nativeResult) { v.Report.Paths.Diagnosis.Candidates[1].Witness.Reference++ },
		func(v *nativeResult) { v.Report.Paths.Diagnosis.ModelPredictions = 1 },
		func(v *nativeResult) { v.Report.Paths.Diagnosis.PlanSHA256 = "wrong" },
		func(v *nativeResult) { v.Report.Paths.Diagnosis.Candidates[1].ProbeOutputsSHA256 = "wrong" },
		func(v *nativeResult) { v.Report.Paths.Search.Selection.ModelCalls = 1 },
		func(v *nativeResult) { v.Source = strings.ReplaceAll(v.Source, "input - int64(2)", "input + int64(2)") },
	} {
		var changed nativeResult
		json.Unmarshal(raw, &changed)
		mutate(&changed)
		if _, err = inspectNativeDiagnosis(changed, "ko", "offline", "test-source", true); err == nil {
			t.Fatal("tampered native diagnostic evidence accepted")
		}
	}
}

func TestNativeDiagnosisFrozenSmokeReplaysWithoutProcesses(t *testing.T) {
	t.Chdir("../..")
	root := "runs/native-path-diagnosis-feature-20261001"
	value, err := auditNativeDiagnosisSmoke(root, "18ae830c18b47b377ae8fa97dcc39df75f6b593b", "5ae486d05b3521ecc48b18bfa28d45cdd9041203")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := read(root + "/report.json")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.MarshalIndent(value, "", "  ")
	if string(append(got, '\n')) != string(raw) {
		t.Fatal("native smoke audit drifted")
	}
}
