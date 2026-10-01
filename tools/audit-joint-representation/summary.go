package main

import (
	"cmp"
	"encoding/json"
	"slices"
)

type sample struct {
	ID      string     `json:"function_view"`
	Family  string     `json:"family"`
	Config  int        `json:"configuration"`
	Goal    int        `json:"desired_mask"`
	Lang    string     `json:"language"`
	Split   string     `json:"split"`
	Target  [4]float32 `json:"finite_soft_target"`
	Support uint8      `json:"valid_choice_support_mask"`
}

type collision struct {
	InputSHA       string   `json:"representation_sha256"`
	CommonSupport  uint8    `json:"common_valid_choice_support_mask"`
	BestFixed      int      `json:"maximum_views_passed_by_one_fixed_choice"`
	DistinctTarget int      `json:"distinct_target_distributions"`
	Views          []sample `json:"views"`
}

type summary struct {
	Observations      int         `json:"observations"`
	Distinct          int         `json:"distinct_representations"`
	DifferentTargets  int         `json:"unequal_target_distribution_groups"`
	EmptyCommon       int         `json:"empty_common_valid_choice_groups"`
	FixedPass         int         `json:"maximum_views_passed_with_input_only_fixed_choices"`
	UnavoidableFailed int         `json:"minimum_failed_views_with_input_only_fixed_choices"`
	Collisions        []collision `json:"unequal_target_distribution_details"`
}

func summarize(groups map[string][]sample) summary {
	out := summary{Distinct: len(groups), Collisions: []collision{}}
	for sha, views := range groups {
		views = slices.Clone(views)
		slices.SortFunc(views, func(a, b sample) int { return cmp.Compare(a.ID, b.ID) })
		common, correct := uint8(15), [4]int{}
		targets := map[string]bool{}
		for _, v := range views {
			common &= v.Support
			raw, _ := json.Marshal(v.Target)
			targets[string(raw)] = true
			for mask := range correct {
				if v.Support&(1<<mask) != 0 {
					correct[mask]++
				}
			}
		}
		best := slices.Max(correct[:])
		out.Observations += len(views)
		out.FixedPass += best
		out.UnavoidableFailed += len(views) - best
		if common == 0 {
			out.EmptyCommon++
		}
		if len(targets) > 1 {
			out.DifferentTargets++
			out.Collisions = append(out.Collisions, collision{sha, common, best, len(targets), views})
		}
	}
	slices.SortFunc(out.Collisions, func(a, b collision) int {
		if a.InputSHA < b.InputSHA {
			return -1
		}
		if a.InputSHA > b.InputSHA {
			return 1
		}
		return 0
	})
	return out
}
