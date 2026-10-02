package threecompositionstudy

import (
	"fmt"
	"math"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

// Oracle uses independent ordinary Go arithmetic and state. It never inspects
// arena nodes, generated source, model predictions or a path interpreter.
func Oracle(family string, config, mask int, x int64) (int64, error) {
	if config < 0 || config >= 24 || mask < 0 || mask >= 8 {
		return 0, fmt.Errorf("three-choice oracle bounds exceeded")
	}
	d, s, t := Parameters(config)
	a, b := x+d, x*s
	first, second, third := mask&1 != 0, mask&2 != 0, mask&4 != 0
	ref := func(chooseSecond bool) int64 {
		if chooseSecond {
			return b
		}
		return a
	}
	write := func(chooseSecond bool, value int64) {
		if chooseSecond {
			b = value
		} else {
			a = value
		}
	}
	switch family {
	case "chained_operands":
		value := a - b
		if first {
			value = b - a
		}
		if second {
			value -= d
		} else {
			value = d - value
		}
		if third {
			return value - s, nil
		}
		return s - value, nil
	case "successive_assignments_reference":
		write(first, a+b)
		write(second, a*s+d)
		return ref(third) - x, nil
	case "branch_assignment_reference":
		condition := x <= t
		if first {
			condition = !condition
		}
		if condition {
			write(second, a+b+d)
		} else {
			b = b*s + a
		}
		return ref(third) + a - b, nil
	case "assignment_reference_operand":
		write(first, a+b)
		value := ref(second)
		if third {
			return b - value, nil
		}
		return value - b, nil
	case "schedule_assignment_operand":
		dependent := func() { write(second, a*s-b) }
		if first {
			dependent()
			a += b
		} else {
			a += b
			dependent()
		}
		if third {
			return b - a, nil
		}
		return a - b, nil
	case "nested_branches_reference":
		condition := x <= t
		if first {
			condition = !condition
		}
		if condition {
			inner := x == d
			if second {
				inner = !inner
			}
			if inner {
				a += b
			} else {
				b = a - b
			}
		} else {
			a = a*s + b
			b += d
		}
		return ref(third) + a - b, nil
	case "comparison_branch_assignment":
		condition := x <= t
		if first {
			condition = t <= x
		}
		if config%2 == 1 {
			condition = x == t
		}
		if second {
			condition = !condition
		}
		if condition {
			write(third, a+b+d)
		} else {
			b = a*s - b
		}
		return a + b + x, nil
	case "boolean_reference_branch_operand":
		low, high := x <= t, t <= x
		condition := low
		if first {
			condition = high
		}
		condition = condition || low && high
		if second {
			condition = !condition
		}
		if condition {
			a += b + d
		} else {
			b = a*s - b
		}
		if third {
			return b - a, nil
		}
		return a - b, nil
	default:
		return 0, fmt.Errorf("unknown three-choice oracle family %q", family)
	}
}

func Cases(family string, config, goal int) ([]pathplan.TestCase, error) {
	d, _, t := Parameters(config)
	inputs := [16]int64{math.MinInt64, math.MinInt64 + 1, math.MaxInt64 - 1, math.MaxInt64,
		-23, -9, -1, 0, 1, 9, 23, t - 1, t, t + 1, d, -d}
	cases := make([]pathplan.TestCase, len(inputs))
	for i, input := range inputs {
		value, err := Oracle(family, config, goal, input)
		if err != nil {
			return nil, err
		}
		cases[i] = pathplan.TestCase{Input: input, Expected: value}
	}
	return cases, nil
}
