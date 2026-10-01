package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestBilingualNativeInspectionRetainsSparseWrongAndRejectsFullWrong(t *testing.T) {
	t.Chdir("../..")
	doc, err := bilingualDocument("en", false)
	if err != nil {
		t.Fatal(err)
	}
	p, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	search, _, err := p.Search(ctx, nil, doc.Cases, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	search.Selection.Choices["operands"] = "layout_reverse"
	body, err := p.Compile(search.Selection.Choices)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"source": body.GoSource(), "report": map[string]any{
		"compiler_source_sha": "test-source", "decision": "PASS", "typecheck_passed": true, "deterministic_replay": true,
		"body_paths": map[string]any{"source_base_matched": true, "search": search,
			"native_case_results": []map[string]any{{"input": 2, "expected": 0, "actual": 0, "passed": true}}}}})
	if err != nil {
		t.Fatal(err)
	}
	var value nativeResult
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if mask, err := inspectBilingual(value, "en", "offline", false, "test-source"); err != nil || mask != 1 {
		t.Fatal("sparse wrong intention must remain valid finite evidence", mask, err)
	}
	if _, err := inspectBilingual(value, "en", "offline", true, "test-source"); err == nil {
		t.Fatal("sparse evidence certified full contract")
	}
	value.Report.Paths.Search.Selection.ModelCalls = 1
	if _, err := inspectBilingual(value, "en", "offline", false, "test-source"); err == nil {
		t.Fatal("offline calls hidden")
	}
}
