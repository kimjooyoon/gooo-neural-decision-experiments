package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestDisconnectedNativeUsesOriginalUnencodedPlan(t *testing.T) {
	v := testView(t)
	doc, src, err := originalDocument(v)
	if err != nil {
		t.Fatal(err)
	}
	original, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if original.PlanSHA256() == v.Prepared.PlanSHA256() {
		t.Fatal("fixture does not distinguish original and model-encoded intents")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s, _, progress, err := original.SearchBatches(ctx, nil, v.Cases, 4, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	n := nativeResult{Source: "synthetic unit projection"}
	n.Report.Decision = "PASS"
	n.Report.Compiler = nativeDeployed
	n.Report.Types = true
	n.Report.Replay = true
	n.Report.Paths.Original = "sha256:" + jointcohort.SHA(src)
	docRaw, _ := json.Marshal(doc)
	n.Report.Paths.Document = "sha256:" + jointcohort.SHA(docRaw)
	n.Report.Paths.Bound = true
	n.Report.Paths.Binding.Equivalent = true
	n.Report.Paths.Search = s
	n.Report.Paths.Progress = progress
	n.Report.Paths.Completeness = 100
	n.Report.Paths.Cases = s.Attempts[len(s.Attempts)-1].Results
	raw, _ := json.Marshal(n)
	if _, _, err = inspectNative(raw, v, doc, src, nil, 1); err != nil {
		t.Fatal(err)
	}
	n.Report.Paths.Search.Selection.PlanSHA256 = v.Prepared.PlanSHA256()
	raw, _ = json.Marshal(n)
	if _, _, err = inspectNative(raw, v, doc, src, nil, 1); err == nil {
		t.Fatal("disconnected native accepted a different model-encoded plan")
	}
}
