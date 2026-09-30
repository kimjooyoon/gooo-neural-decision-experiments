package main

import (
	"bytes"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestSemanticComparisonDoesNotEraseMeasuredTimings(t *testing.T) {
	search := pathplan.SearchResult{Selection: pathplan.Selection{Receipts: []pathplan.Receipt{{ID: "condition", PredictNS: 12345}}, Choices: map[string]string{"condition": "reference_second"}}}
	first, err := searchSemantics(search)
	if err != nil {
		t.Fatal(err)
	}
	if search.Selection.Receipts[0].PredictNS != 12345 {
		t.Fatal("comparison mutated original prediction timing")
	}
	search.Selection.Receipts[0].PredictNS = 54321
	second, err := searchSemantics(search)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("timings affected semantic equality")
	}
	search.Selection.Choices["condition"] = "reference_first"
	third, err := searchSemantics(search)
	if err != nil || bytes.Equal(first, third) {
		t.Fatal("changed structural choice was not detected")
	}
}

func TestIndependentArithmeticOverflowUsesInt64Ring(t *testing.T) {
	for input, want := range map[int64]int64{-9223372036854775808: 10, 9223372036854775807: 8, 0: 10, 1: -18, 5: -10, 6: 22} {
		if got := gold(input); got != want {
			t.Fatalf("gold(%d)=%d want %d", input, got, want)
		}
	}
}
