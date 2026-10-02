package main

import (
	"fmt"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

type counts struct {
	Views    int    `json:"views"`
	Cases    int    `json:"cases"`
	Passed   int    `json:"initial_passed_cases"`
	Complete int    `json:"initial_complete"`
	Extra    int    `json:"static_ranked_extra_attempts"`
	Curve    [8]int `json:"complete_by_budget"`
	Masks    [8]int `json:"initial_mask_histogram"`
}

func (c *counts) add(v threecohort.View, p jointdecision.ThreePrediction) error {
	order := []int{0, 1, 2, 3, 4, 5, 6, 7}
	sort.Slice(order, func(i, j int) bool {
		a, b := order[i], order[j]
		return p.Probabilities[a] > p.Probabilities[b] || p.Probabilities[a] == p.Probabilities[b] && a < b
	})
	if int(p.Mask) != order[0] {
		return fmt.Errorf("ranking tie differs from model")
	}
	c.Views++
	c.Cases += v.Target.Cases
	c.Passed += v.Target.Passed[p.Mask]
	c.Masks[p.Mask]++
	if v.Target.Passed[p.Mask] == v.Target.Cases {
		c.Complete++
	}
	first := -1
	for i, mask := range order {
		if first < 0 && v.Target.Passed[mask] == v.Target.Cases {
			first = i
		}
		if first >= 0 {
			c.Curve[i]++
		}
	}
	if first < 0 {
		return fmt.Errorf("no finite passing path")
	}
	c.Extra += first
	return nil
}

type pairCounts struct {
	Pairs          int `json:"pairs"`
	Disagree       int `json:"mask_disagreements"`
	BothValid      int `json:"both_valid"`
	AgreeValid     int `json:"same_mask_both_valid"`
	DifferentValid int `json:"different_masks_both_valid"`
	SameWrong      int `json:"same_mask_both_wrong"`
}

func (c *pairCounts) add(en, ko uint16, valid uint8) {
	c.Pairs++
	a, b := valid&(1<<en) != 0, valid&(1<<ko) != 0
	if en != ko {
		c.Disagree++
	}
	if a && b {
		c.BothValid++
		if en == ko {
			c.AgreeValid++
		} else {
			c.DifferentValid++
		}
	}
	if en == ko && !a && !b {
		c.SameWrong++
	}
}

type result struct {
	Counts map[string]*counts     `json:"counts"`
	Pairs  map[string]*pairCounts `json:"pairs"`
}

func (r *result) add(v threecohort.View, p jointdecision.ThreePrediction) error {
	for _, key := range []string{v.Split + "/all", v.Split + "/language/" + v.Language, v.Split + "/family/" + v.Family} {
		if r.Counts[key] == nil {
			r.Counts[key] = &counts{}
		}
		if err := r.Counts[key].add(v, p); err != nil {
			return err
		}
	}
	return nil
}
