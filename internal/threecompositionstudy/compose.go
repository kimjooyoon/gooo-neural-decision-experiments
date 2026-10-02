package threecompositionstudy

import (
	"fmt"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func (a *arena) compose(family string, config int) ([]pathplan.Choice, []int, int, error) {
	roots := []int{0, 1}
	var choices []pathplan.Choice
	var result int
	switch family {
	case "chained_operands":
		first := a.binary("subtract", a.a, a.b)
		second := a.binary("subtract", a.d, first)
		result = a.binary("subtract", a.s, second)
		choices = []pathplan.Choice{layout(pathplan.OperandOrder, first), layout(pathplan.OperandOrder, second),
			layout(pathplan.OperandOrder, result)}
	case "successive_assignments_reference":
		first := a.write(a.first, a.binary("add", a.a, a.b))
		second := a.write(a.first, a.binary("add", a.binary("multiply", a.a, a.s), a.d))
		ref := a.reference(a.first)
		result = a.binary("subtract", ref, a.x)
		choices = []pathplan.Choice{named(pathplan.AssignmentTarget, first, a.first, a.second),
			named(pathplan.AssignmentTarget, second, a.first, a.second), named(pathplan.LocalReference, ref, a.first, a.second)}
		roots = append(roots, first, second)
	case "branch_assignment_reference":
		write := a.write(a.first, a.binary("add", a.binary("add", a.a, a.b), a.d))
		other := a.write(a.second, a.binary("add", a.binary("multiply", a.b, a.s), a.a))
		branch := a.branch(a.binary("less_equal", a.x, a.thresholdExpression()), []int{write}, []int{other})
		ref, value := a.referenceResult()
		result = value
		choices = []pathplan.Choice{layout(pathplan.BranchLayout, branch),
			named(pathplan.AssignmentTarget, write, a.first, a.second), named(pathplan.LocalReference, ref, a.first, a.second)}
		roots = append(roots, branch)
	case "assignment_reference_operand":
		write := a.write(a.first, a.binary("add", a.a, a.b))
		ref := a.reference(a.first)
		result = a.binary("subtract", ref, a.b)
		choices = []pathplan.Choice{named(pathplan.AssignmentTarget, write, a.first, a.second),
			named(pathplan.LocalReference, ref, a.first, a.second), layout(pathplan.OperandOrder, result)}
		roots = append(roots, write)
	case "schedule_assignment_operand":
		first := a.write(a.first, a.binary("add", a.a, a.b))
		second := a.write(a.first, a.binary("subtract", a.binary("multiply", a.a, a.s), a.b))
		result = a.binary("subtract", a.a, a.b)
		choices = []pathplan.Choice{{Kind: pathplan.RootOrder},
			named(pathplan.AssignmentTarget, second, a.first, a.second), layout(pathplan.OperandOrder, result)}
		roots = append(roots, first, second)
	case "nested_branches_reference":
		return a.nested(roots)
	case "comparison_branch_assignment":
		operation := "less_equal"
		if config%2 == 1 {
			operation = "equal"
		}
		condition := a.binary(operation, a.x, a.thresholdExpression())
		write := a.write(a.first, a.binary("add", a.binary("add", a.a, a.b), a.d))
		other := a.write(a.second, a.binary("subtract", a.binary("multiply", a.a, a.s), a.b))
		branch := a.branch(condition, []int{write}, []int{other})
		result = a.binary("add", a.binary("add", a.a, a.b), a.x)
		choices = []pathplan.Choice{layout(pathplan.OperandOrder, condition), layout(pathplan.BranchLayout, branch),
			named(pathplan.AssignmentTarget, write, a.first, a.second)}
		roots = append(roots, branch)
	case "boolean_reference_branch_operand":
		return a.boolean(roots, config)
	default:
		return nil, nil, 0, fmt.Errorf("unknown three-choice composition %q", family)
	}
	return choices, roots, result, nil
}

func (a *arena) nested(roots []int) ([]pathplan.Choice, []int, int, error) {
	first := a.write(a.first, a.binary("add", a.a, a.b))
	second := a.write(a.second, a.binary("subtract", a.a, a.b))
	inner := a.branch(a.binary("equal", a.x, a.d), []int{first}, []int{second})
	otherA := a.write(a.first, a.binary("add", a.binary("multiply", a.a, a.s), a.b))
	otherB := a.write(a.second, a.binary("add", a.b, a.d))
	outer := a.branch(a.binary("less_equal", a.x, a.thresholdExpression()), []int{inner}, []int{otherA, otherB})
	ref, result := a.referenceResult()
	choices := []pathplan.Choice{layout(pathplan.BranchLayout, outer), layout(pathplan.BranchLayout, inner),
		named(pathplan.LocalReference, ref, a.first, a.second)}
	return choices, append(roots, outer), result, nil
}

func (a *arena) boolean(roots []int, config int) ([]pathplan.Choice, []int, int, error) {
	lowName, highName := fmt.Sprintf("conditionLow%d", config), fmt.Sprintf("conditionHigh%d", config)
	low := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtLet, Name: lowName, Expr: a.binary("less_equal", a.x, a.thresholdExpression())})
	high := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtLet, Name: highName, Expr: a.binary("less_equal", a.t, a.x)})
	lowRef, highRef, ref := a.reference(lowName), a.reference(highName), a.reference(lowName)
	// Both locals remain used for every option; equality is intentionally retained.
	condition := a.binary("or", ref, a.binary("and", lowRef, highRef))
	write := a.write(a.first, a.binary("add", a.binary("add", a.a, a.b), a.d))
	other := a.write(a.second, a.binary("subtract", a.binary("multiply", a.a, a.s), a.b))
	branch := a.branch(condition, []int{write}, []int{other})
	result := a.binary("subtract", a.a, a.b)
	choices := []pathplan.Choice{named(pathplan.LocalReference, ref, lowName, highName),
		layout(pathplan.BranchLayout, branch), layout(pathplan.OperandOrder, result)}
	return choices, append(roots, low, high, branch), result, nil
}
