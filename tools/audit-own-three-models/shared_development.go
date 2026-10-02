package main

import (
	"errors"
	"math"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

type developmentCounts struct {
	Views           int     `json:"views"`
	Cases           int     `json:"case_denominator"`
	InitialCases    int     `json:"initial_passed_cases"`
	InitialComplete int     `json:"initial_complete_functions"`
	Extra           int     `json:"static_ranked_extra_attempts_to_first_complete"`
	Curve           [8]int  `json:"static_ranked_complete_functions_by_budget"`
	Partial         [8]int  `json:"static_ranked_best_passing_cases_by_budget"`
	PassingMass     float64 `json:"summed_passing_set_mass"`
	NLL             float64 `json:"summed_passing_set_nll"`
}
type developmentRow struct {
	View       string                        `json:"view_id"`
	Group      string                        `json:"program_contract_group"`
	Language   string                        `json:"language"`
	InputSHA   string                        `json:"input_sha256"`
	SourceSHA  string                        `json:"source_sha256"`
	Prediction jointdecision.ThreePrediction `json:"prediction"`
	Passed     [8]int                        `json:"passed_cases_by_mask"`
	Order      [8]int                        `json:"static_ranked_mask_order"`
}
type development struct {
	Counts    developmentCounts             `json:"counts"`
	Families  map[string]*developmentCounts `json:"families"`
	Languages map[string]*developmentCounts `json:"languages"`
	Pairs     int                           `json:"bilingual_pairs"`
	Disagree  int                           `json:"bilingual_initial_mask_disagreements"`
	SameWrong int                           `json:"bilingual_same_mask_both_wrong"`
	Rows      []developmentRow              `json:"rows"`
}

func (c *developmentCounts) add(v threecohort.View, p jointdecision.ThreePrediction, order [8]int) {
	c.Views++
	c.Cases += v.Target.Cases
	c.InitialCases += v.Target.Passed[p.Mask]
	if v.Target.Passed[p.Mask] == v.Target.Cases {
		c.InitialComplete++
	}
	passing := 0.
	for mask, prob := range p.Probabilities {
		if v.Target.Joint[mask] > 0 {
			passing += float64(prob)
		}
	}
	c.PassingMass += passing
	c.NLL -= math.Log(math.Max(passing, 1e-12))
	best, first := 0, -1
	for budget, mask := range order {
		best = max(best, v.Target.Passed[mask])
		c.Partial[budget] += best
		if best == v.Target.Cases {
			c.Curve[budget]++
			if first < 0 {
				first = budget
			}
		}
	}
	c.Extra += first
}
func evaluateDevelopment(m *jointdecision.ThreeModel, views []threecohort.View) (development, error) {
	r := development{Families: map[string]*developmentCounts{}, Languages: map[string]*developmentCounts{}}
	pairs := map[string]map[string]developmentRow{}
	var workspace jointdecision.ThreeWorkspace
	var prediction jointdecision.ThreePrediction
	for _, v := range views {
		if v.Split != "development" {
			continue
		}
		if err := m.PredictInto(v.Text, &workspace, &prediction); err != nil {
			return r, err
		}
		order := [8]int{0, 1, 2, 3, 4, 5, 6, 7}
		sort.Slice(order[:], func(i, j int) bool {
			a, b := order[i], order[j]
			return prediction.Probabilities[a] > prediction.Probabilities[b] || prediction.Probabilities[a] == prediction.Probabilities[b] && a < b
		})
		if order[0] != int(prediction.Mask) {
			return r, errors.New("prediction/ranking tie differs")
		}
		if r.Families[v.Family] == nil {
			r.Families[v.Family] = &developmentCounts{}
		}
		if r.Languages[v.Language] == nil {
			r.Languages[v.Language] = &developmentCounts{}
		}
		for _, counts := range []*developmentCounts{&r.Counts, r.Families[v.Family], r.Languages[v.Language]} {
			counts.add(v, prediction, order)
		}
		row := developmentRow{v.ID, v.Group, v.Language, threecohort.SHA([]byte(v.Text)), v.SourceSHA, prediction, v.Target.Passed, order}
		r.Rows = append(r.Rows, row)
		if pairs[v.Group] == nil {
			pairs[v.Group] = map[string]developmentRow{}
		}
		if _, ok := pairs[v.Group][v.Language]; ok {
			return r, errors.New("duplicate bilingual view")
		}
		pairs[v.Group][v.Language] = row
	}
	for _, pair := range pairs {
		en, enOK := pair["en"]
		ko, koOK := pair["ko"]
		if !enOK || !koOK || len(pair) != 2 {
			return r, errors.New("bilingual pair incomplete")
		}
		r.Pairs++
		if en.Prediction.Mask != ko.Prediction.Mask {
			r.Disagree++
		} else if en.Passed[en.Prediction.Mask] < 16 && ko.Passed[ko.Prediction.Mask] < 16 {
			r.SameWrong++
		}
	}
	if r.Counts.Views != 512 || r.Pairs != 256 || r.Counts.Curve[7] != 512 || len(r.Families) != 8 || len(r.Languages) != 2 {
		return r, errors.New("closed development denominator differs")
	}
	return r, nil
}
