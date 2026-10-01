package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestOfflineFeedbackControlRemainsExactAndMakesZeroPredictions(t *testing.T) {
	raw, err := read("../../studies/conditional-paths-v1/cohort/en-budget-64.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc document
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Cases[6].Expected = 999
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	first, firstBody, err := search(ctx, prepared, nil, doc.Cases, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, secondBody, err := search(ctx, prepared, nil, doc.Cases, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if firstBody.GoooSource() != secondBody.GoooSource() || !reflect.DeepEqual(first.Progress, second.Progress) || len(first.Feedback) != 0 || len(second.Feedback) != 0 {
		t.Fatal("offline feedback control changed execution")
	}
	last := second.Progress[len(second.Progress)-1]
	if last.Attempted != 64 || last.SelectedPassed != 6 || last.Cases != 7 || last.Selection.ModelCalls != 0 || !last.Exhausted {
		t.Fatal("offline finite or call denominator differs")
	}
}
func TestPilotUsesActualFrozenModelAndKeepsFeedbackCallsSeparate(t *testing.T) {
	raw, err := read("../../studies/conditional-paths-v1/cohort/en-budget-64.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc document
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Cases[6].Expected = 999
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		t.Fatal(err)
	}
	model, err := decision.LoadPath("../../runs/typed-path-positioned-random-20261001/models/fp32/model.json")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	value, _, err := search(ctx, prepared, model, doc.Cases, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	last := value.Progress[len(value.Progress)-1]
	if len(value.Feedback) != 2 || last.FeedbackPredictions != 12 || last.Selection.ModelCalls != 18 || last.SelectedPassed != 6 || !last.Exhausted {
		t.Fatal("feedback calls or partial evidence hidden")
	}
}
