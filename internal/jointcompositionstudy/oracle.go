package jointcompositionstudy

import (
	"fmt"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"math"
)

// Oracle is independently written ordinary Go signed arithmetic and state.
// It inspects neither the arena, generated Gooo/Go nor model outputs.
func Oracle(family string, c, mask int, x int64) (int64, error) {
	if c < 0 || c >= 48 || mask < 0 || mask > 3 {
		return 0, fmt.Errorf("joint oracle bounds exceeded")
	}
	d, s, t := Parameters(c)
	a, b := x+d, x*s
	first, second := mask&1 != 0, mask&2 != 0
	p := x <= t
	if c%2 == 1 {
		p = x == t
	}
	switch family {
	case "assignment_reference":
		if first {
			b = a + b
		} else {
			a = a + b
		}
		ref := a
		if second {
			ref = b
		}
		return ref - x, nil
	case "operand_assignment":
		z := x*d + 1
		diff := a - b
		if first {
			diff = b - a
		}
		if second {
			b = z + diff
		} else {
			a = z + diff
		}
		return a + b + a*b, nil
	case "branch_reference":
		if first {
			p = !p
		}
		if p {
			a += d
			b *= s
		} else {
			a *= s
			b += d
		}
		ref := a
		if second {
			ref = b
		}
		return ref + x + a - b, nil
	case "predicate_assignment":
		if first {
			p = t <= x
			if c%2 == 1 {
				p = t == x
			}
		}
		if p {
			if second {
				b = a*b + d
			} else {
				a = a*b + d
			}
		} else {
			b = a - b
		}
		return a + b + x, nil
	case "schedule_operand":
		if first {
			b = a * b
			a = a + b
		} else {
			a = a + b
			b = a * b
		}
		if second {
			return b - a, nil
		}
		return a - b, nil
	case "schedule_branch":
		if second {
			p = !p
		}
		branch := func() {
			if p {
				b = a*b + d
			} else {
				b = a - b
			}
		}
		if first {
			branch()
			a = a + b
		} else {
			a = a + b
			branch()
		}
		return a + b, nil
	default:
		return 0, fmt.Errorf("unknown joint oracle family")
	}
}
func Cases(family string, c, desired int) ([]pathplan.TestCase, error) {
	d, _, t := Parameters(c)
	inputs := [16]int64{math.MinInt64, math.MinInt64 + 1, math.MaxInt64 - 1, math.MaxInt64, -23, -9, -1, 0, 1, 9, 23, t - 1, t, t + 1, d, -d}
	cases := make([]pathplan.TestCase, len(inputs))
	for i, x := range inputs {
		y, err := Oracle(family, c, desired, x)
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
	Joint     [4]float32    `json:"joint_mask_targets"`
	Outside   float32       `json:"product_marginal_mass_outside_full_target"`
	Passed    [4]int        `json:"passed_cases_by_mask"`
	Cases     int           `json:"ordered_case_denominator"`
}

func Target(plan pathplan.Plan, family string, c, desired int) (FiniteTarget, error) {
	cases, err := Cases(family, c, desired)
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
		for _, test := range cases {
			actual, err := program.Evaluate(test.Input)
			oracle, other := Oracle(family, c, mask, test.Input)
			if err != nil || other != nil || actual.Int != oracle {
				return target, fmt.Errorf("joint typed/oracle disagreement family=%s config=%d mask=%d input=%d", family, c, mask, test.Input)
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
		return target, fmt.Errorf("full joint target unreachable")
	}
	for _, mask := range target.BestMasks {
		weight := 1 / float32(len(target.BestMasks))
		target.Joint[mask] = weight
		for i := 0; i < 2; i++ {
			target.Marginals[i][(mask>>i)&1] += weight
		}
	}
	for mask := 0; mask < 4; mask++ {
		if target.Joint[mask] == 0 {
			target.Outside += target.Marginals[0][mask&1] * target.Marginals[1][(mask>>1)&1]
		}
	}
	return target, nil
}
