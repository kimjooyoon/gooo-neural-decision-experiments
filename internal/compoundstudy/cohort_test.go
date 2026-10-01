package compoundstudy

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestAllFourInteractingPathsAgreeWithIndependentOracle(t *testing.T) {
	for _, template := range Templates {
		for _, language := range []string{"en", "ko"} {
			plan, err := Fixture(template, language, 0)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := pathplan.Prepare(plan)
			if err != nil {
				t.Fatal(template, err)
			}
			for mask := uint16(0); mask < 4; mask++ {
				body, err := prepared.Compile(Choices(plan, mask))
				if err != nil {
					t.Fatal(err)
				}
				selected, err := Mask(plan, Choices(plan, mask))
				if err != nil || selected != mask {
					t.Fatal("mask mismatch")
				}
				for _, x := range []int64{math.MinInt64, -100, -1, 0, 1, A - 1, A, A + 1, 100, math.MaxInt64} {
					actual, err := body.Evaluate(x)
					expected, e := Oracle(template, mask, x)
					if err != nil || e != nil || actual.Int != expected {
						t.Fatalf("%s mask=%d x=%d got=%d expected=%d %v", template, mask, x, actual.Int, expected, err)
					}
				}
			}
		}
	}
}
func TestCohortPreservesAmbiguityAndContradictions(t *testing.T) {
	rows, err := Cohort()
	if err != nil || len(rows) != 72 {
		t.Fatal(err, len(rows))
	}
	fresh, _ := Cohort()
	a, _ := json.Marshal(rows)
	b, _ := json.Marshal(fresh)
	if string(a) != string(b) {
		t.Fatal("cohort not deterministic")
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if seen[r.ID] || r.Document.Max != 4 || len(r.Document.Plan.Decisions) != 2 || len(r.FiniteBestMasks) == 0 {
			t.Fatal("cohort bounds", r.ID)
		}
		seen[r.ID] = true
		if r.Contract == "contradictory" && (len(r.Document.Cases) != 8 || r.FiniteBestPassed != 7) {
			t.Fatal("contradictory denominator lost")
		}
		if r.Contract == "sparse" && (len(r.Document.Cases) != 1 || len(r.FiniteBestMasks) < 2) {
			t.Fatal("sparse ambiguity lost", r.ID)
		}
		for _, s := range r.Separate {
			for _, c := range r.Document.Cases {
				if s.Input == c.Input {
					t.Fatal("selected input leaked into separate observations")
				}
			}
		}
	}
	for _, x := range []int64{math.MinInt64, -9, 0, 9, math.MaxInt64} {
		first, _ := Oracle("reference_schedule", 1, x)
		second, _ := Oracle("reference_schedule", 3, x)
		if first != second {
			t.Fatal("declared commuting-update equivalence lost")
		}
	}
}
