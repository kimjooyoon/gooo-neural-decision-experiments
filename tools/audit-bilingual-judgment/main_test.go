package main

import (
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bilingualstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

func TestOfflineContinuationAndAmbiguityAccounting(t *testing.T) {
	pairs, err := bilingualstudy.Load("../../data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	result, captures, err := summarize(pairs, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := result["counts"].(map[string]int)
	if len(captures) != 160 || c["views"] != 320 || c["initial_pair_agreement"] != 160 ||
		c["same_wrong_intention_pairs"] != 80 || c["actual_model_calls"] != 0 ||
		c["full_program_views_after"] != 320 || c["full_finite_passed_after"] != c["full_finite_total"] ||
		c["added_path_attempts"] == 0 || c["full_finite_passed_before"] >= c["full_finite_total"] {
		t.Fatal("agreement, partial denominator or deterministic continuation lost", c)
	}
	for _, pair := range captures {
		for _, row := range pair {
			if row.PredictionNS != 0 || len(row.FullContractSHA) != 64 || row.AddedAttempts > 1 {
				t.Fatal("source-bound bounded no-model accounting lost", row)
			}
		}
	}
}

func TestPairRankingRetainsFiniteTies(t *testing.T) {
	model, err := decision.LoadPath("../../runs/feedback-path-soft-target-mps-20261001/models/fp32/model.json")
	if err != nil {
		t.Fatal(err)
	}
	p := decision.Prediction{Logits: [8]float32{1e20, 0, 0, 0, 1000, 1000}}
	probs, err := eligible(model, p, [2]string{"layout_forward", "layout_reverse"})
	if err != nil || probs != [2]float64{0.5, 0.5} {
		t.Fatal("tie or closed eligible pair lost", probs, err)
	}
	if _, err := eligible(model, p, [2]string{"unknown", "layout_reverse"}); err == nil {
		t.Fatal("unknown label silently ranked")
	}
}
