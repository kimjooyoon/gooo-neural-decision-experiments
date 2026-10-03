package main

import (
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/orderjudge"
)

func TestFixedCohortAndReferenceArithmetic(t *testing.T) {
	counts := map[string]int{}
	ids := map[string]bool{}
	for _, r := range tasks() {
		counts[r.Group]++
		if ids[r.ID] {
			t.Fatal("duplicate task")
		}
		ids[r.ID] = true
		if r.source() == "" || r.intent() == "" {
			t.Fatal("empty fixture")
		}
		var features [orderjudge.IntentDim]float32
		if err := orderjudge.IntentFeatures(r.intent(), &features); err != nil {
			t.Fatal(err)
		}
	}
	if counts["train"] != 64 || counts["new-template"] != 32 || counts["presentation"] != 32 || counts["new-constants"] != 32 {
		t.Fatal(counts)
	}
	for _, row := range []struct {
		family  string
		changed bool
		want    [2]int64
	}{
		{"add-multiply", false, [2]int64{6, 5}}, {"subtract-multiply", false, [2]int64{-2, 1}},
		{"negate-add", false, [2]int64{2, -6}}, {"square-add", false, [2]int64{5, 9}},
		{"add-multiply", true, [2]int64{12, 8}}, {"subtract-multiply", true, [2]int64{4, 7}},
		{"negate-add", true, [2]int64{0, -4}}, {"square-add", true, [2]int64{7, 25}},
	} {
		for order := range 2 {
			r := task{Family: row.family, NewConstants: row.changed, WantedOrder: order}
			if r.expected(2) != row.want[order] {
				t.Fatal("reference arithmetic", row, order)
			}
		}
	}
}
