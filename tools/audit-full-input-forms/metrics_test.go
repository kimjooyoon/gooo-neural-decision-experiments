package main

import (
	"math"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

func TestMetricsSeparateInitialPartialRankedAndPairedOutcomes(t *testing.T) {
	target := threecompositionstudy.FiniteTarget{Cases: 16, Passed: [8]int{16, 8, 4, 0}}
	p := jointdecision.ThreePrediction{Mask: 3, Probabilities: [8]float32{.1, .2, .3, .4}}
	var m metric
	if err := m.add(target, p, 1); err != nil {
		t.Fatal(err)
	}
	if m.Views != 1 || m.Cases != 16 || m.Complete != 0 || m.Passed != 0 || m.Extra != 3 || m.CompleteByBudget != [8]int{0, 0, 0, 1, 1, 1, 1, 1} || m.MatchedByBudget != [8]int{0, 4, 8, 16, 16, 16, 16, 16} {
		t.Fatalf("wrong scope: %+v", m)
	}
	if math.Abs(m.PassingNLLSum-math.Log(8)) > 1e-12 || m.Bins[4].Views != 1 || m.Bins[4].Valid != 0 {
		t.Fatal("calibration/loss accounting differs")
	}
	var paired bilingualMetric
	for _, masks := range [][2]uint16{{0, 1}, {2, 2}, {0, 2}} {
		if err := paired.add(masks[0], masks[1], 3); err != nil {
			t.Fatal(err)
		}
	}
	if paired != (bilingualMetric{Pairs: 3, Different: 2, BothValid: 1, DifferentValid: 1, SameWrong: 1}) {
		t.Fatal("paired validity conflated", paired)
	}
}

func TestMetricBoundariesAndStableLoss(t *testing.T) {
	var uniform jointdecision.ThreePrediction
	for i := range uniform.Probabilities {
		uniform.Probabilities[i] = .125
	}
	target := threecompositionstudy.FiniteTarget{Cases: 1, Passed: [8]int{1, 1, 1, 1, 1, 1, 1, 1}}
	var m metric
	if err := m.add(target, uniform, 1); err != nil || m.Complete != 1 || m.PassingNLLSum != 0 {
		t.Fatal("uniform valid target", err, m)
	}
	if rank(uniform.Probabilities) != [8]uint16{0, 1, 2, 3, 4, 5, 6, 7} {
		t.Fatal("equal-probability tie order differs")
	}
	for _, mutate := range []func(*jointdecision.ThreePrediction){
		func(p *jointdecision.ThreePrediction) { p.Mask = 8 },
		func(p *jointdecision.ThreePrediction) { p.Mask = 1 },
		func(p *jointdecision.ThreePrediction) { p.Probabilities[0] = float32(math.NaN()) },
		func(p *jointdecision.ThreePrediction) { p.Probabilities[0] = .9 },
		func(p *jointdecision.ThreePrediction) { p.Logits[0] = float32(math.Inf(1)) },
	} {
		p := uniform
		mutate(&p)
		before := m
		if err := m.add(target, p, 1); err == nil || m != before {
			t.Fatal("invalid observation changed metrics")
		}
	}
	if got := stableNLL([8]float32{1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000}, 1, 2); math.Abs(got-2000) > 1e-10 {
		t.Fatal("underflow changed stable loss", got)
	}
}

func TestCalibrationSummaryKeepsEmptyEstimatesUnknown(t *testing.T) {
	var m metric
	empty := m.summary()
	if empty.FirstFiniteFraction != nil || empty.FirstCompleteFraction != nil || empty.PassingNLL != nil || empty.ValidityBrier != nil || empty.ValidityECE != nil {
		t.Fatal("empty estimates became zero success")
	}
	target := threecompositionstudy.FiniteTarget{Cases: 16, Passed: [8]int{16, 8, 4, 0}}
	p := jointdecision.ThreePrediction{Mask: 3, Probabilities: [8]float32{.1, .2, .3, .4}}
	if err := m.add(target, p, 1); err != nil {
		t.Fatal(err)
	}
	s := m.summary()
	confidence := float64(p.Probabilities[3])
	if *s.FirstFiniteFraction != 0 || *s.FirstCompleteFraction != 0 || math.Abs(*s.PassingNLL-math.Log(8)) > 1e-12 || *s.ValidityECE != confidence || *s.ValidityBrier != confidence*confidence {
		t.Fatal("derived calibration differs", s)
	}
}
