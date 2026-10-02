package threecompositionstudy

import (
	"fmt"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type FiniteTarget struct {
	BestMasks []int         `json:"best_finite_masks"`
	Marginals [3][2]float32 `json:"coordinate_marginals"`
	Joint     [8]float32    `json:"joint_mask_targets"`
	Outside   float32       `json:"product_marginal_mass_outside_full_target"`
	Passed    [8]int        `json:"passed_cases_by_mask"`
	Cases     int           `json:"ordered_case_denominator"`
}

// Target retains every full passing mask, including conditional/equality ties.
// It proves typed evaluation and the independent oracle agree for every mask.
func Target(plan pathplan.Plan, family string, config, goal int) (FiniteTarget, error) {
	cases, err := Cases(family, config, goal)
	if err != nil {
		return FiniteTarget{}, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return FiniteTarget{}, err
	}
	target := FiniteTarget{Cases: len(cases)}
	for mask := 0; mask < 8; mask++ {
		choices, err := Choices(plan, mask)
		if err != nil {
			return target, err
		}
		program, err := prepared.Compile(choices)
		if err != nil {
			return target, fmt.Errorf("family=%s config=%d mask=%d: %w", family, config, mask, err)
		}
		for _, test := range cases {
			actual, err := program.Evaluate(test.Input)
			independent, other := Oracle(family, config, mask, test.Input)
			if err != nil || other != nil || actual.Int != independent {
				return target, fmt.Errorf("three typed/oracle disagreement family=%s config=%d mask=%d input=%d",
					family, config, mask, test.Input)
			}
			if actual.Int == test.Expected {
				target.Passed[mask]++
			}
		}
		if target.Passed[mask] == len(cases) {
			target.BestMasks = append(target.BestMasks, mask)
		}
	}
	if len(target.BestMasks) == 0 {
		return target, fmt.Errorf("authored goal has no complete finite target")
	}
	mass := float32(1) / float32(len(target.BestMasks))
	for _, mask := range target.BestMasks {
		target.Joint[mask] = mass
		for coordinate := range 3 {
			target.Marginals[coordinate][(mask>>coordinate)&1] += mass
		}
	}
	for mask := range 8 {
		if target.Joint[mask] == 0 {
			product := float32(1)
			for coordinate := range 3 {
				product *= target.Marginals[coordinate][(mask>>coordinate)&1]
			}
			target.Outside += product
		}
	}
	return target, nil
}
