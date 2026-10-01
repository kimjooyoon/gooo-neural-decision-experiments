package compositionstudy

import (
	"fmt"
	"math"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

// Oracle is ordinary arithmetic/state, with no typed interpreter, model,
// generated source or inspection of a constructed plan.
func Oracle(family string, config, mask int, input int64) (int64, error) {
	if config < 0 || config >= 48 || mask < 0 || mask > 3 {
		return 0, fmt.Errorf("oracle configuration or mask out of bounds")
	}
	d, s, t := Parameters(config)
	a, b := input+d, input*s
	first, second := mask&1 != 0, mask&2 != 0
	predicate := input <= t
	if config%2 == 1 {
		predicate = input == t
	}
	switch family {
	case "reference_operand":
		ref := a
		if first {
			ref = b
		}
		diff := ref - input
		if second {
			diff = input - ref
		}
		return diff + a + b, nil
	case "assignment_branch":
		if second {
			predicate = !predicate
		}
		if predicate {
			if first {
				b = a + d
			} else {
				a = a + d
			}
		} else {
			b *= s
		}
	case "predicate_branch":
		if first {
			predicate = t <= input
			if config%2 == 1 {
				predicate = t == input
			}
		}
		if second {
			predicate = !predicate
		}
		if predicate {
			a += d
		} else {
			a *= s
		}
	case "assignment_schedule":
		write := func() {
			if first {
				b = a + b
			} else {
				a = a + b
			}
		}
		if second {
			b = a * s
			write()
		} else {
			write()
			b = a * s
		}
	case "reference_schedule":
		if second {
			b = a * s
			a += b
		} else {
			a += b
			b = a * s
		}
		ref := a
		if first {
			ref = b
		}
		return ref + a + b, nil
	case "branch_schedule":
		if first {
			predicate = !predicate
		}
		branch := func() {
			if predicate {
				a += d
			} else {
				a *= s
			}
		}
		if second {
			branch()
			b = a + b
		} else {
			b = a + b
			branch()
		}
	default:
		return 0, fmt.Errorf("unknown oracle family")
	}
	return a + b, nil
}

// Cases preserves the complete ordered 16-input finite contract, including ties.
func Cases(family string, config, desired int) ([]pathplan.TestCase, error) {
	d, _, t := Parameters(config)
	inputs := [16]int64{math.MinInt64, math.MinInt64 + 1, math.MaxInt64 - 1, math.MaxInt64, -17, -7, -1, 0, 1, 7, 17, t - 1, t, t + 1, d, -d}
	cases := make([]pathplan.TestCase, len(inputs))
	for i, x := range inputs {
		y, err := Oracle(family, config, desired, x)
		if err != nil {
			return nil, err
		}
		cases[i] = pathplan.TestCase{Input: x, Expected: y}
	}
	return cases, nil
}

type FiniteTarget struct {
	BestMasks []int         `json:"best_finite_masks"`
	Marginals [2][2]float32 `json:"coordinate_marginals"`
	Passed    [4]int        `json:"passed_cases_by_mask"`
	Cases     int           `json:"ordered_case_denominator"`
}

// Target checks all four compiled combinations against the independent full
// oracle. Marginals retain every finite equivalence tie uniformly.
func Target(plan pathplan.Plan, family string, config, desired int) (FiniteTarget, error) {
	cases, err := Cases(family, config, desired)
	if err != nil {
		return FiniteTarget{}, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return FiniteTarget{}, err
	}
	target := FiniteTarget{Cases: len(cases)}
	for mask := 0; mask < 4; mask++ {
		choices, err := Choices(plan, mask)
		if err != nil {
			return target, err
		}
		program, err := prepared.Compile(choices)
		if err != nil {
			return target, err
		}
		for _, c := range cases {
			actual, err := program.Evaluate(c.Input)
			oracle, oracleErr := Oracle(family, config, mask, c.Input)
			if err != nil || oracleErr != nil || actual.Int != oracle {
				return target, fmt.Errorf("typed/oracle disagreement: family=%s config=%d mask=%d input=%d", family, config, mask, c.Input)
			}
			if actual.Int == c.Expected {
				target.Passed[mask]++
			}
		}
		if target.Passed[mask] == target.Cases {
			target.BestMasks = append(target.BestMasks, mask)
		}
	}
	if len(target.BestMasks) == 0 {
		return target, fmt.Errorf("desired independent full contract unreachable")
	}
	for _, mask := range target.BestMasks {
		for i := 0; i < 2; i++ {
			target.Marginals[i][(mask>>i)&1] += 1 / float32(len(target.BestMasks))
		}
	}
	return target, nil
}
