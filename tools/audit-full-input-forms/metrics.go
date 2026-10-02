package main

import (
	"errors"
	"math"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

type calibrationBin struct {
	Views         int     `json:"views"`
	Valid         int     `json:"selected_path_valid"`
	ConfidenceSum float64 `json:"confidence_sum"`
}
type metric struct {
	Views            int                `json:"views"`
	Cases            int                `json:"finite_expectations"`
	Passed           int                `json:"first_path_passed"`
	Complete         int                `json:"first_path_complete"`
	Extra            int                `json:"static_extra_candidates"`
	CompleteByBudget [8]int             `json:"complete_by_budget"`
	MatchedByBudget  [8]int             `json:"best_finite_matches_by_budget"`
	Masks            [8]int             `json:"selected_mask_histogram"`
	PassingNLLSum    float64            `json:"stable_passing_set_nll_sum"`
	SquaredErrorSum  float64            `json:"selected_validity_squared_error_sum"`
	Bins             [10]calibrationBin `json:"selected_validity_confidence_bins"`
}

// Rank has fixed storage and uses the model's lowest-mask tie rule.
func rank(probabilities [8]float32) [8]uint16 {
	var order [8]uint16
	for i := range order {
		at := i
		for at > 0 {
			prior := order[at-1]
			if probabilities[prior] > probabilities[i] || (probabilities[prior] == probabilities[i] && prior < uint16(i)) {
				break
			}
			order[at] = prior
			at--
		}
		order[at] = uint16(i)
	}
	return order
}
func passingSet(target threecompositionstudy.FiniteTarget) (uint8, error) {
	if target.Cases <= 0 {
		return 0, errors.New("observed finite denominator required")
	}
	var valid uint8
	for mask, n := range target.Passed {
		if n < 0 || n > target.Cases {
			return 0, errors.New("invalid finite count")
		}
		if n == target.Cases {
			valid |= 1 << mask
		}
	}
	if valid == 0 {
		return 0, errors.New("frozen target has no passing path")
	}
	return valid, nil
}
func stableNLL(logits [8]float32, temperature float64, valid uint8) float64 {
	all, pass := math.Inf(-1), math.Inf(-1)
	for i, l := range logits {
		v := float64(l) / temperature
		all = math.Max(all, v)
		if valid&(1<<i) != 0 {
			pass = math.Max(pass, v)
		}
	}
	total, passing := 0.0, 0.0
	for i, l := range logits {
		v := float64(l) / temperature
		total += math.Exp(v - all)
		if valid&(1<<i) != 0 {
			passing += math.Exp(v - pass)
		}
	}
	return all + math.Log(total) - pass - math.Log(passing)
}
func (m *metric) add(target threecompositionstudy.FiniteTarget, p jointdecision.ThreePrediction, temperature float64) error {
	valid, err := passingSet(target)
	if err != nil {
		return err
	}
	if temperature <= 0 || math.IsNaN(temperature) || math.IsInf(temperature, 0) || p.Mask >= 8 {
		return errors.New("invalid prediction contract")
	}
	total := 0.0
	for i, v := range p.Probabilities {
		l := float64(p.Logits[i])
		if v < 0 || v > 1 || math.IsNaN(float64(v)) || math.IsNaN(l) || math.IsInf(l, 0) {
			return errors.New("nonfinite prediction")
		}
		total += float64(v)
	}
	if math.Abs(total-1) > 1e-5 {
		return errors.New("probabilities do not sum to one")
	}
	order := rank(p.Probabilities)
	if order[0] != p.Mask {
		return errors.New("selected mask differs from stable ranking")
	}
	nll := stableNLL(p.Logits, temperature, valid)
	if math.IsNaN(nll) || math.IsInf(nll, 0) || nll < -1e-12 {
		return errors.New("invalid stable loss")
	}
	m.Views++
	m.Cases += target.Cases
	m.Passed += target.Passed[p.Mask]
	m.Masks[p.Mask]++
	m.PassingNLLSum += math.Max(0, nll)
	correct := 0.0
	if valid&(1<<p.Mask) != 0 {
		m.Complete++
		correct = 1
	}
	first, best := -1, 0
	for i, mask := range order {
		if first < 0 && valid&(1<<mask) != 0 {
			first = i
		}
		if first >= 0 {
			m.CompleteByBudget[i]++
		}
		best = max(best, target.Passed[mask])
		m.MatchedByBudget[i] += best
	}
	m.Extra += first
	confidence := float64(p.Probabilities[p.Mask])
	d := confidence - correct
	m.SquaredErrorSum += d * d
	bin := min(9, int(confidence*10))
	m.Bins[bin].Views++
	m.Bins[bin].Valid += int(correct)
	m.Bins[bin].ConfidenceSum += confidence
	return nil
}

type bilingualMetric struct {
	Pairs          int `json:"pairs"`
	Different      int `json:"mask_disagreements"`
	BothValid      int `json:"both_valid"`
	DifferentValid int `json:"different_masks_both_valid"`
	SameWrong      int `json:"same_mask_both_wrong"`
}

type metricSummary struct {
	Counts                metric   `json:"counts"`
	FirstFiniteFraction   *float64 `json:"first_path_finite_fraction"`
	FirstCompleteFraction *float64 `json:"first_path_complete_fraction"`
	PassingNLL            *float64 `json:"stable_passing_set_nll_mean"`
	ValidityBrier         *float64 `json:"selected_validity_brier"`
	ValidityECE           *float64 `json:"selected_validity_ece_10_bins"`
}

// Empty observations retain null estimates. ECE concerns selected-path validity;
// it is descriptive calibration on the stated split, not an intent-success prior.
func (m metric) summary() metricSummary {
	s := metricSummary{Counts: m}
	if m.Views == 0 {
		return s
	}
	finite := float64(m.Passed) / float64(m.Cases)
	complete := float64(m.Complete) / float64(m.Views)
	nll, brier := m.PassingNLLSum/float64(m.Views), m.SquaredErrorSum/float64(m.Views)
	ece := 0.0
	for _, bin := range m.Bins {
		ece += math.Abs(bin.ConfidenceSum-float64(bin.Valid)) / float64(m.Views)
	}
	s.FirstFiniteFraction = &finite
	s.FirstCompleteFraction = &complete
	s.PassingNLL = &nll
	s.ValidityBrier = &brier
	s.ValidityECE = &ece
	return s
}

func (m *bilingualMetric) add(en, ko uint16, valid uint8) error {
	if en >= 8 || ko >= 8 || valid == 0 {
		return errors.New("bounded paired masks and passing set required")
	}
	m.Pairs++
	a, b := valid&(1<<en) != 0, valid&(1<<ko) != 0
	if en != ko {
		m.Different++
	}
	if a && b {
		m.BothValid++
		if en != ko {
			m.DifferentValid++
		}
	}
	if en == ko && !a && !b {
		m.SameWrong++
	}
	return nil
}
