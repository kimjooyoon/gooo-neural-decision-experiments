package main

import (
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

type counts struct {
	Views               int     `json:"function_views"`
	Cases               int     `json:"ordered_contract_cases"`
	ActualValues        int     `json:"actual_ordered_candidate_values"`
	Extra               int     `json:"extra_candidate_attempts"`
	Predictions         int     `json:"actual_model_predictions"`
	FeedbackPredictions int     `json:"actual_feedback_predictions"`
	Declines            int     `json:"zero_call_context_declines"`
	Unnecessary         int     `json:"sole_remaining_zero_call_feedback"`
	Curve               [8]int  `json:"complete_functions_budgets_1_to_8"`
	Partial             [8]int  `json:"best_passing_cases_budgets_1_to_8"`
	Pairs               int     `json:"bilingual_function_pairs"`
	Disagreement        int     `json:"bilingual_initial_mask_disagreements"`
	Outside             float64 `json:"summed_initial_mass_outside_passing_set"`
	SetNLL              float64 `json:"summed_initial_passing_set_nll"`
	WallNS              int64   `json:"summed_actual_sdk_session_wall_ns"`
	KernelNS            int64   `json:"summed_actual_recorded_prediction_ns"`
	MedianNS            int64   `json:"median_actual_sdk_session_wall_ns"`
	P95NS               int64   `json:"p95_actual_sdk_session_wall_ns"`
}

func (c *counts) add(v threecohort.View, o threefeedback.Capture) error {
	if err := threefeedback.VerifyFiniteAttempts(v, o.Search); err != nil {
		return err
	}
	s := o.Search
	c.Views++
	c.Cases += 16
	c.ActualValues += 16 * len(s.Attempts)
	c.Extra += len(s.Attempts) - 1
	c.Predictions += s.Selection.ModelCalls
	c.WallNS += o.WallNS
	initial := o.Progress[0].Selection
	if initial.Three != nil {
		c.KernelNS += initial.Three.PredictNS
	} else {
		for _, r := range initial.Receipts {
			c.KernelNS += r.PredictNS
		}
	}
	for _, f := range o.Feedback {
		c.FeedbackPredictions += f.ModelCalls
		if f.ContextDeclined {
			c.Declines++
		}
		if f.RankingUnnecessary {
			c.Unnecessary++
		}
		if f.Three != nil {
			c.KernelNS += f.Three.PredictNS
		}
		for _, j := range f.Judgments {
			c.KernelNS += j.Prediction.PredictNS
		}
	}
	best := 0
	for i := range 8 {
		if i < len(s.Attempts) {
			best = max(best, s.Attempts[i].Passed)
		}
		c.Partial[i] += best
		if best == 16 {
			c.Curve[i]++
		}
	}
	var distribution [8]float64
	if initial.Three != nil {
		for i, p := range initial.Three.Probabilities {
			distribution[i] = float64(p)
		}
	} else if len(s.EligibleProbabilities) == 3 {
		for mask := range 8 {
			distribution[mask] = 1
			for i := range 3 {
				distribution[mask] *= s.EligibleProbabilities[i][(mask>>i)&1]
			}
		}
	} else {
		distribution[s.Attempts[0].Mask] = 1
	}
	sum, valid := 0., 0.
	for mask, p := range distribution {
		if p < 0 || math.IsNaN(p) || math.IsInf(p, 0) {
			return errors.New("finite full initial distribution required")
		}
		sum += p
		if v.Target.Joint[mask] > 0 {
			valid += p
		} else {
			c.Outside += p
		}
	}
	if math.Abs(sum-1) > 2e-6 {
		return errors.New("complete initial mass differs")
	}
	c.SetNLL -= math.Log(math.Max(valid, 1e-12))
	return nil
}

type accumulator struct {
	Counts counts
	Masks  map[string]map[string]uint16
	Walls  [512]int64
}

func (a *accumulator) add(v threecohort.View, c threefeedback.Capture) error {
	if a.Counts.Views >= 512 {
		return errors.New("cell exceeds frozen view count")
	}
	if a.Masks == nil {
		a.Masks = map[string]map[string]uint16{}
	}
	if a.Masks[v.Group] == nil {
		a.Masks[v.Group] = map[string]uint16{}
	}
	if _, ok := a.Masks[v.Group][v.Language]; ok {
		return errors.New("duplicate bilingual language view")
	}
	a.Masks[v.Group][v.Language] = c.Search.Attempts[0].Mask
	a.Walls[a.Counts.Views] = c.WallNS
	return a.Counts.add(v, c)
}
func (a *accumulator) finish() (counts, error) {
	if a.Counts.Views != 512 || a.Counts.Curve[7] != 512 || len(a.Masks) != 256 {
		return counts{}, errors.New("complete frozen 512-view eight-mask cell required")
	}
	for _, pair := range a.Masks {
		en, enOK := pair["en"]
		ko, koOK := pair["ko"]
		if len(pair) != 2 || !enOK || !koOK {
			return counts{}, errors.New("exact complete Korean/English pair required")
		}
		a.Counts.Pairs++
		if en != ko {
			a.Counts.Disagreement++
		}
	}
	slices.Sort(a.Walls[:])
	a.Counts.MedianNS = a.Walls[256]
	a.Counts.P95NS = a.Walls[486]
	return a.Counts, nil
}
func choose(cal map[string]counts, models map[string]*model) string {
	ids := []string{}
	for id, m := range models {
		if m != nil {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, func(a, b string) int {
		ca, cb := cal[a], cal[b]
		for _, pair := range [][2]int{{ca.Extra, cb.Extra}, {ca.Predictions, cb.Predictions}, {ca.Disagreement, cb.Disagreement}, {models[a].Pin.Packed, models[b].Pin.Packed}} {
			if pair[0] < pair[1] {
				return -1
			}
			if pair[0] > pair[1] {
				return 1
			}
		}
		return strings.Compare(a, b)
	})
	return ids[0]
}
