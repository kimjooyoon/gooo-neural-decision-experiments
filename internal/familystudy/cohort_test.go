package familystudy

import (
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestIndependentFiniteTargetsAndSelectionSeparation(t *testing.T) {
	rows, err := Cohort()
	if err != nil || len(rows) != 120 {
		t.Fatalf("%d %v", len(rows), err)
	}
	seen := map[string]bool{}
	ambiguous := 0
	for _, r := range rows {
		if seen[r.ID] || len(r.Separate) < 8 || r.Document.Max != 2 {
			t.Fatal("cohort identity or budget differs")
		}
		seen[r.ID] = true
		if _, err := pathplan.Prepare(r.Document.Plan); err != nil {
			t.Fatal(err)
		}
		if len(r.FiniteBestLabels) == 2 {
			ambiguous++
		}
		if r.Contract == "contradictory" && (len(r.Document.Cases) != 8 || r.FiniteBestPassed != 7) {
			t.Fatal("contradictory denominator lost")
		}
		if r.Contract == "complete" && r.FiniteBestPassed != 7 {
			t.Fatal("complete oracle witness differs")
		}
		if r.Contract == "sparse" && len(r.Document.Cases) != 1 {
			t.Fatal("sparse witness changed")
		}
		for _, selected := range r.Document.Cases {
			for _, separate := range r.Separate {
				if selected.Input == separate.Input {
					t.Fatal("selected input leaked into separate denominator")
				}
			}
		}
	}
	if ambiguous == 0 {
		t.Fatal("finite ambiguity disappeared")
	}
}
