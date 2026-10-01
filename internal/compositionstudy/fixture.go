// Package compositionstudy defines the frozen fresh two-choice Gooo cohort.
// It constructs typed arenas; the independent arithmetic oracle lives separately.
package compositionstudy

import (
	"fmt"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var Families = [6]string{"reference_operand", "assignment_branch", "predicate_branch", "assignment_schedule", "reference_schedule", "branch_schedule"}

func Parameters(config int) (delta, scale, threshold int64) {
	return int64(config%13 - 6), int64(2 + config%3), int64(config%11 - 5)
}

type arena struct {
	base          bodyplan.Plan
	x, d, s, a, b int
	first, second string
}

func (a *arena) expression(value bodyplan.Expr) int {
	a.base.Expressions = append(a.base.Expressions, value)
	return len(a.base.Expressions) - 1
}

func (a *arena) binary(op string, left, right int) int {
	return a.expression(bodyplan.Expr{Kind: bodyplan.ExprBinary, Operation: op, Left: left, Right: right})
}

func (a *arena) statement(value bodyplan.Stmt) int {
	a.base.Statements = append(a.base.Statements, value)
	return len(a.base.Statements) - 1
}

func newArena(family string, config int) *arena {
	d, s, _ := Parameters(config)
	a := &arena{first: fmt.Sprintf("alpha%d", config), second: fmt.Sprintf("beta%d", config)}
	a.base = bodyplan.Plan{Schema: bodyplan.Schema, ID: fmt.Sprintf("fresh-%s-%02d", family, config), Name: "ChoosePath", ResultType: decision.TypeInt}
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

func (a *arena) predicate(config int) int {
	_, _, t := Parameters(config)
	threshold := a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: t})
	op := "less_equal"
	if config%2 == 1 {
		op = "equal"
	}
	return a.binary(op, a.x, threshold)
}

func (a *arena) arithmeticBranch(config int, targetChoice bool) (branch, write int) {
	predicate := a.predicate(config)
	write = a.statement(bodyplan.Stmt{Kind: bodyplan.StmtAssign, Name: a.first, Expr: a.binary("add", a.a, a.d)})
	name, ref := a.first, a.a
	if targetChoice {
		name, ref = a.second, a.b
	}
	other := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtAssign, Name: name, Expr: a.binary("multiply", ref, a.s)})
	branch = a.statement(bodyplan.Stmt{Kind: bodyplan.StmtIf, Expr: predicate, Then: []int{write}, Else: []int{other}})
	return branch, write
}

func namedChoice(kind string, target int, first, second string) pathplan.Choice {
	labels := [2]string{"reference_first", "reference_second"}
	if kind == pathplan.AssignmentTarget {
		labels = [2]string{"assign_first", "assign_second"}
	}
	return pathplan.Choice{Kind: kind, Target: target, Options: []pathplan.Option{{Label: labels[0], Name: first}, {Label: labels[1], Name: second}}}
}

func layoutChoice(kind string, target int) pathplan.Choice {
	return pathplan.Choice{Kind: kind, Target: target, Options: []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}}
}

func scheduleChoice(root []int) pathplan.Choice {
	reverse := append([]int(nil), root...)
	reverse[2], reverse[3] = reverse[3], reverse[2]
	return pathplan.Choice{Kind: pathplan.RootOrder, Options: []pathplan.Option{{Label: "schedule_forward", Order: append([]int(nil), root...)}, {Label: "schedule_reverse", Order: reverse}}}
}

// Fixture uses independent display aliases and source fallback=config mod 4.
// Both original orientations and every complete combination remain typed.
func Fixture(family string, config, desired int, language string) (pathplan.Plan, error) {
	if config < 0 || config >= 48 || desired < 0 || desired > 3 || language != "en" && language != "ko" {
		return pathplan.Plan{}, fmt.Errorf("fresh configuration, mask or language outside frozen bounds")
	}
	a := newArena(family, config)
	var choices []pathplan.Choice
	var roots []int
	result := -1
	switch family {
	case "reference_operand":
		ref := a.expression(bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: a.first})
		diff := a.binary("subtract", ref, a.x)
		result = a.binary("add", a.binary("add", diff, a.a), a.b)
		choices = []pathplan.Choice{namedChoice(pathplan.LocalReference, ref, a.first, a.second), layoutChoice(pathplan.OperandOrder, diff)}
		roots = []int{0, 1}
	case "assignment_branch":
		branch, write := a.arithmeticBranch(config, true)
		choices = []pathplan.Choice{namedChoice(pathplan.AssignmentTarget, write, a.first, a.second), layoutChoice(pathplan.BranchLayout, branch)}
		roots = []int{0, 1, branch}
	case "predicate_branch":
		branch, _ := a.arithmeticBranch(config, false)
		predicate := a.base.Statements[branch].Expr
		choices = []pathplan.Choice{layoutChoice(pathplan.OperandOrder, predicate), layoutChoice(pathplan.BranchLayout, branch)}
		roots = []int{0, 1, branch}
	case "assignment_schedule", "reference_schedule":
		write := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtAssign, Name: a.first, Expr: a.binary("add", a.a, a.b)})
		other := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtAssign, Name: a.second, Expr: a.binary("multiply", a.a, a.s)})
		roots = []int{0, 1, write, other}
		if family == "assignment_schedule" {
			choices = []pathplan.Choice{namedChoice(pathplan.AssignmentTarget, write, a.first, a.second)}
		} else {
			ref := a.expression(bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: a.first})
			result = a.binary("add", ref, a.binary("add", a.a, a.b))
			choices = []pathplan.Choice{namedChoice(pathplan.LocalReference, ref, a.first, a.second)}
		}
	case "branch_schedule":
		write := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtAssign, Name: a.second, Expr: a.binary("add", a.a, a.b)})
		branch, _ := a.arithmeticBranch(config, false)
		roots = []int{0, 1, write, branch}
		choices = []pathplan.Choice{layoutChoice(pathplan.BranchLayout, branch)}
	default:
		return pathplan.Plan{}, fmt.Errorf("unknown fresh composition")
	}
	if result < 0 {
		result = a.binary("add", a.a, a.b)
	}
	ret := a.statement(bodyplan.Stmt{Kind: bodyplan.StmtReturn, Expr: result})
	a.base.Root = append(roots, ret)
	if len(choices) == 1 {
		choices = append(choices, scheduleChoice(a.base.Root))
	}
	for i := range choices {
		choices[i].ID = fmt.Sprintf("choice%d", i)
		choices[i].Fallback = choices[i].Options[(config%4>>i)&1].Label
		choices[i].Intent = "caller context;intent: " + Natural(family, choices[i].Kind, i, config, desired, language)
	}
	return pathplan.Plan{Schema: pathplan.Schema, Base: a.base, Decisions: choices}, nil
}

func Choices(plan pathplan.Plan, mask int) (map[string]string, error) {
	if len(plan.Decisions) != 2 || mask < 0 || mask > 3 {
		return nil, fmt.Errorf("two choices and a four-path mask required")
	}
	return map[string]string{plan.Decisions[0].ID: plan.Decisions[0].Options[mask&1].Label, plan.Decisions[1].ID: plan.Decisions[1].Options[(mask>>1)&1].Label}, nil
}
