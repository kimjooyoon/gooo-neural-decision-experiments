package main

import (
	"errors"
	"math"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
)

type counts struct {
	Views               int     `json:"function_views"`
	Cases               int     `json:"ordered_finite_cases"`
	InitialComplete     int     `json:"initial_complete_functions"`
	Extra               int     `json:"extra_candidate_attempts"`
	Predictions         int     `json:"actual_model_predictions"`
	FeedbackPredictions int     `json:"actual_feedback_predictions"`
	Declines            int     `json:"zero_call_context_declines"`
	Unnecessary         int     `json:"sole_remaining_zero_call_feedback"`
	Curve               [4]int  `json:"complete_function_curve_budgets_1_to_4"`
	Partial             [4]int  `json:"best_passing_case_curve_budgets_1_to_4"`
	Pairs               int     `json:"bilingual_function_pairs"`
	Disagreement        int     `json:"bilingual_initial_mask_disagreements"`
	Outside             float64 `json:"summed_initial_probability_mass_outside_full_target"`
	SetNLL              float64 `json:"summed_initial_passing_set_nll"`
	UniformCE           float64 `json:"summed_initial_uniform_target_cross_entropy"`
	WallNS              int64   `json:"summed_sdk_session_wall_ns"`
}

func (c *counts) add(v jointcohort.View, o jointfeedback.Capture) error {
	s := o.Search
	if err := jointfeedback.VerifyFiniteAttempts(v, s); err != nil {
		return err
	}
	c.Views++
	c.Cases += 16
	c.Extra += len(s.Attempts) - 1
	c.Predictions += s.Selection.ModelCalls
	c.WallNS += o.WallNS
	if len(s.Attempts) == 1 {
		c.InitialComplete++
	}
	for _, f := range o.Feedback {
		c.FeedbackPredictions += f.ModelCalls
		if f.ContextDeclined {
			c.Declines++
		}
		if f.RankingUnnecessary {
			c.Unnecessary++
		}
	}
	best := 0
	for i := 0; i < 4; i++ {
		if i < len(s.Attempts) {
			best = max(best, s.Attempts[i].Passed)
		}
		c.Partial[i] += best
		if best == 16 {
			c.Curve[i]++
		}
	}
	var p [4]float64
	if s.Selection.Joint != nil {
		for i, mass := range s.Selection.Joint.Probabilities {
			p[i] = float64(mass)
		}
	} else if len(s.EligibleProbabilities) == 2 {
		for mask := range p {
			p[mask] = s.EligibleProbabilities[0][mask&1] * s.EligibleProbabilities[1][(mask>>1)&1]
		}
	} else {
		p[s.Attempts[0].Mask] = 1
	}
	full, valid := 0.0, 0.0
	for mask, mass := range p {
		if math.IsNaN(mass) || math.IsInf(mass, 0) || mass < 0 {
			return errors.New("finite initial probabilities required")
		}
		full += mass
		if v.Target.Joint[mask] > 0 {
			valid += mass
			c.UniformCE -= float64(v.Target.Joint[mask]) * math.Log(math.Max(mass, 1e-12))
		} else {
			c.Outside += mass
		}
	}
	if math.Abs(full-1) > 1e-5 {
		return errors.New("initial full-mask distribution must sum to one")
	}
	c.SetNLL -= math.Log(math.Max(valid, 1e-12))
	return nil
}
