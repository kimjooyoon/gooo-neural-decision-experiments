// Package jointcompositionstudy defines the frozen new nonlinear two-choice cohort.
package jointcompositionstudy

import (
	"fmt"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var Families = [6]string{"assignment_reference", "operand_assignment", "branch_reference", "predicate_assignment", "schedule_operand", "schedule_branch"}

func Parameters(c int) (d, s, t int64) {
	return int64((7*c+3)%19 - 9), int64(2 + (2*c+1)%5), int64((3*c+1)%17 - 8)
}
func FallbackMask(c int) int { return (3*c + 1) % 4 }

type arena struct {
	base          bodyplan.Plan
	x, d, s, a, b int
	first, second string
}

func (a *arena) expression(v bodyplan.Expr) int {
	a.base.Expressions = append(a.base.Expressions, v)
	return len(a.base.Expressions) - 1
}
func (a *arena) binary(op string, l, r int) int {
	return a.expression(bodyplan.Expr{Kind: bodyplan.ExprBinary, Operation: op, Left: l, Right: r})
}
func (a *arena) statement(v bodyplan.Stmt) int {
	a.base.Statements = append(a.base.Statements, v)
	return len(a.base.Statements) - 1
}
func (a *arena) write(name string, expr int) int {
	return a.statement(bodyplan.Stmt{Kind: bodyplan.StmtAssign, Name: name, Expr: expr})
}
func newArena(family string, c int) *arena {
	d, s, _ := Parameters(c)
	a := &arena{first: fmt.Sprintf("localA%d", c), second: fmt.Sprintf("localB%d", c)}
	a.base = bodyplan.Plan{Schema: bodyplan.Schema, ID: fmt.Sprintf("joint-%s-%02d", family, c), Name: "ChoosePath", ResultType: decision.TypeInt}
	a.x = a.expression(bodyplan.Expr{Kind: bodyplan.ExprInput, Name: "input"})
	a.d = a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: d})
	a.s = a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: s})
	initA, initB := a.binary("add", a.x, a.d), a.binary("multiply", a.x, a.s)
	a.statement(bodyplan.Stmt{Kind: bodyplan.StmtLet, Name: a.first, Expr: initA})
	a.statement(bodyplan.Stmt{Kind: bodyplan.StmtLet, Name: a.second, Expr: initB})
	a.a = a.expression(bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: a.first})
	a.b = a.expression(bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: a.second})
	return a
}
func (a *arena) predicate(c int) int {
	_, _, t := Parameters(c)
	threshold := a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: t})
	op := "less_equal"
	if c%2 == 1 {
		op = "equal"
	}
	return a.binary(op, a.x, threshold)
}
func named(kind string, target int, a, b string) pathplan.Choice {
	labels := [2]string{"reference_first", "reference_second"}
	if kind == pathplan.AssignmentTarget {
		labels = [2]string{"assign_first", "assign_second"}
	}
	return pathplan.Choice{Kind: kind, Target: target, Options: []pathplan.Option{{Label: labels[0], Name: a}, {Label: labels[1], Name: b}}}
}
func layout(kind string, target int) pathplan.Choice {
	return pathplan.Choice{Kind: kind, Target: target, Options: []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}}
}
func schedule(root []int) pathplan.Choice {
	reverse := append([]int(nil), root...)
	reverse[2], reverse[3] = reverse[3], reverse[2]
	return pathplan.Choice{Kind: pathplan.RootOrder, Options: []pathplan.Option{{Label: "schedule_forward", Order: append([]int(nil), root...)}, {Label: "schedule_reverse", Order: reverse}}}
}
func (a *arena) branch(c int, then, otherwise []int) int {
	return a.statement(bodyplan.Stmt{Kind: bodyplan.StmtIf, Expr: a.predicate(c), Then: then, Else: otherwise})
}
func (a *arena) compose(family string, c int) ([]pathplan.Choice, []int, int, error) {
	roots := []int{0, 1}
	var choices []pathplan.Choice
	var result int
	switch family {
	case "assignment_reference":
		write := a.write(a.first, a.binary("add", a.a, a.b))
		ref := a.expression(bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: a.first})
		result = a.binary("subtract", ref, a.x)
		choices = []pathplan.Choice{named(pathplan.AssignmentTarget, write, a.first, a.second), named(pathplan.LocalReference, ref, a.first, a.second)}
		roots = append(roots, write)
	case "operand_assignment":
		one := a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: 1})
		z := a.binary("add", a.binary("multiply", a.x, a.d), one)
		diff := a.binary("subtract", a.a, a.b)
		write := a.write(a.first, a.binary("add", z, diff))
		result = a.binary("add", a.binary("add", a.a, a.b), a.binary("multiply", a.a, a.b))
		choices = []pathplan.Choice{layout(pathplan.OperandOrder, diff), named(pathplan.AssignmentTarget, write, a.first, a.second)}
		roots = append(roots, write)
	case "branch_reference":
		then := []int{a.write(a.first, a.binary("add", a.a, a.d)), a.write(a.second, a.binary("multiply", a.b, a.s))}
		otherwise := []int{a.write(a.first, a.binary("multiply", a.a, a.s)), a.write(a.second, a.binary("add", a.b, a.d))}
		branch := a.branch(c, then, otherwise)
		ref := a.expression(bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: a.first})
		result = a.binary("subtract", a.binary("add", a.binary("add", ref, a.x), a.a), a.b)
		choices = []pathplan.Choice{layout(pathplan.BranchLayout, branch), named(pathplan.LocalReference, ref, a.first, a.second)}
		roots = append(roots, branch)
	case "predicate_assignment":
		write := a.write(a.first, a.binary("add", a.binary("multiply", a.a, a.b), a.d))
		other := a.write(a.second, a.binary("subtract", a.a, a.b))
		branch := a.branch(c, []int{write}, []int{other})
		result = a.binary("add", a.binary("add", a.a, a.b), a.x)
		choices = []pathplan.Choice{layout(pathplan.OperandOrder, a.base.Statements[branch].Expr), named(pathplan.AssignmentTarget, write, a.first, a.second)}
		roots = append(roots, branch)
	case "schedule_operand":
		write := a.write(a.first, a.binary("add", a.a, a.b))
		other := a.write(a.second, a.binary("multiply", a.a, a.b))
		roots = append(roots, write, other)
		result = a.binary("subtract", a.a, a.b)
		choices = []pathplan.Choice{{Kind: pathplan.RootOrder}, layout(pathplan.OperandOrder, result)}
	case "schedule_branch":
		write := a.write(a.first, a.binary("add", a.a, a.b))
		then := a.write(a.second, a.binary("add", a.binary("multiply", a.a, a.b), a.d))
		other := a.write(a.second, a.binary("subtract", a.a, a.b))
		branch := a.branch(c, []int{then}, []int{other})
		roots = append(roots, write, branch)
		result = a.binary("add", a.a, a.b)
		choices = []pathplan.Choice{{Kind: pathplan.RootOrder}, layout(pathplan.BranchLayout, branch)}
	default:
		return nil, nil, 0, fmt.Errorf("unknown joint composition")
	}
	return choices, roots, result, nil
}

// Fixture keeps complete typed fragments and target-free source aliases.
func Fixture(family string, c, desired int, language string) (pathplan.Plan, error) {
	if c < 0 || c >= 48 || desired < 0 || desired > 3 || (language != "en" && language != "ko") {
		return pathplan.Plan{}, fmt.Errorf("joint frozen fixture bounds exceeded")
	}
	a := newArena(family, c)
	choices, roots, result, err := a.compose(family, c)
	if err != nil {
		return pathplan.Plan{}, err
	}
	ret := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtReturn, Expr: result})
	a.base.Root = append(roots, ret)
	for i := range choices {
		if choices[i].Kind == pathplan.RootOrder {
			choices[i] = schedule(a.base.Root)
		}
		choices[i].ID = fmt.Sprintf("choice%d", i)
		choices[i].Fallback = choices[i].Options[(FallbackMask(c)>>i)&1].Label
		choices[i].Intent = "caller context;intent: " + Natural(family, choices[i].Kind, i, c, desired, language)
	}
	return pathplan.Plan{Schema: pathplan.Schema, Base: a.base, Decisions: choices}, nil
}
func Choices(plan pathplan.Plan, mask int) (map[string]string, error) {
	if len(plan.Decisions) != 2 || mask < 0 || mask > 3 {
		return nil, fmt.Errorf("two choices and absolute mask 0..3 required")
	}
	return map[string]string{plan.Decisions[0].ID: plan.Decisions[0].Options[mask&1].Label, plan.Decisions[1].ID: plan.Decisions[1].Options[(mask>>1)&1].Label}, nil
}
