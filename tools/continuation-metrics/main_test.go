package main

import (
	"math"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestBatchMeanExcludesRejudgmentAndKeepsCost(t *testing.T) {
	p := []pathplan.SessionProgress{{Cases: 7},
		{Attempted: 8, SelectedPassed: 3, Cases: 7, Selection: pathplan.Selection{ModelCalls: 6}},
		{Attempted: 8, SelectedPassed: 3, Cases: 7, Selection: pathplan.Selection{ModelCalls: 12}},
		{Attempted: 16, SelectedPassed: 6, Cases: 7, Selection: pathplan.Selection{ModelCalls: 12}},
		{Attempted: 24, SelectedPassed: 6, Cases: 7, Selection: pathplan.Selection{ModelCalls: 18}}}
	c, err := summarize(p)
	if err != nil || len(c.Points) != 3 || c.FirstBest != 16 || c.Points[1].Predictions != 12 || c.Points[2].Predictions != 18 || math.Abs(c.Mean-100*float64(15)/21) > 1e-10 {
		t.Fatalf("%+v %v", c, err)
	}
	p[3].SelectedPassed = 2
	if _, err := summarize(p); err == nil {
		t.Fatal("lost previous partial result accepted")
	}
}
