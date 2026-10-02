// Package threecompositionstudy authors the preregistered three-choice corpus.
// It constructs legal fragments without reading any model or observed results.
package threecompositionstudy

import (
	"fmt"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var Families = [8]string{
	"chained_operands", "successive_assignments_reference", "branch_assignment_reference",
	"assignment_reference_operand", "schedule_assignment_operand", "nested_branches_reference",
	"comparison_branch_assignment", "boolean_reference_branch_operand",
}

func Parameters(config int) (delta, scale, threshold int64) {
	return int64((7*config+3)%19 - 9), int64(2 + (2*config+1)%5), int64((3*config+1)%17 - 8)
}

func FallbackMask(config int) int { return (5*config + 3) % 8 }

type arena struct {
	base             bodyplan.Plan
	x, d, s, t, a, b int
	first, second    string
	threshold        int64
}

func (a *arena) expression(v bodyplan.Expr) int {
	a.base.Expressions = append(a.base.Expressions, v)
	return len(a.base.Expressions) - 1
}

func (a *arena) binary(op string, left, right int) int {
	return a.expression(bodyplan.Expr{Kind: bodyplan.ExprBinary, Operation: op, Left: left, Right: right})
}

func (a *arena) statement(v bodyplan.Stmt) int {
	a.base.Statements = append(a.base.Statements, v)
	return len(a.base.Statements) - 1
}

func (a *arena) write(name string, value int) int {
	return a.statement(bodyplan.Stmt{Kind: bodyplan.StmtAssign, Name: name, Expr: value})
}

func (a *arena) reference(name string) int {
	return a.expression(bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: name})
}

func (a *arena) thresholdExpression() int {
	if a.t < 0 {
		a.t = a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: a.threshold})
	}
	return a.t
}

func (a *arena) branch(condition int, then, otherwise []int) int {
	return a.statement(bodyplan.Stmt{Kind: bodyplan.StmtIf, Expr: condition, Then: then, Else: otherwise})
}

func newArena(family string, config int) *arena {
	d, s, t := Parameters(config)
	a := &arena{first: fmt.Sprintf("localA%d", config), second: fmt.Sprintf("localB%d", config),
		t: -1, threshold: t}
	a.base = bodyplan.Plan{Schema: bodyplan.Schema, ID: fmt.Sprintf("three-%s-%02d", family, config),
		Name: "ChoosePath", ResultType: decision.TypeInt, Expressions: make([]bodyplan.Expr, 0, 32),
		Statements: make([]bodyplan.Stmt, 0, 16)}
	a.x = a.expression(bodyplan.Expr{Kind: bodyplan.ExprInput, Name: "input"})
	a.d = a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: d})
	a.s = a.expression(bodyplan.Expr{Kind: bodyplan.ExprInt, Int: s})
	a.statement(bodyplan.Stmt{Kind: bodyplan.StmtLet, Name: a.first, Expr: a.binary("add", a.x, a.d)})
	a.statement(bodyplan.Stmt{Kind: bodyplan.StmtLet, Name: a.second, Expr: a.binary("multiply", a.x, a.s)})
	a.a, a.b = a.reference(a.first), a.reference(a.second)
	return a
}

func named(kind string, target int, first, second string) pathplan.Choice {
	labels := [2]string{"reference_first", "reference_second"}
	if kind == pathplan.AssignmentTarget {
		labels = [2]string{"assign_first", "assign_second"}
	}
	return pathplan.Choice{Kind: kind, Target: target,
		Options: []pathplan.Option{{Label: labels[0], Name: first}, {Label: labels[1], Name: second}}}
}

func layout(kind string, target int) pathplan.Choice {
	return pathplan.Choice{Kind: kind, Target: target,
		Options: []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}}
}

func schedule(root []int) pathplan.Choice {
	reverse := append([]int(nil), root...)
	reverse[2], reverse[3] = reverse[3], reverse[2]
	return pathplan.Choice{Kind: pathplan.RootOrder, Options: []pathplan.Option{
		{Label: "schedule_forward", Order: append([]int(nil), root...)}, {Label: "schedule_reverse", Order: reverse}}}
}

func (a *arena) referenceResult() (int, int) {
	ref := a.reference(a.first)
	return ref, a.binary("subtract", a.binary("add", ref, a.a), a.b)
}

func Fixture(family string, config, goal int, language string) (pathplan.Plan, error) {
	if config < 0 || config >= 24 || goal < 0 || goal >= 8 || (language != "en" && language != "ko") {
		return pathplan.Plan{}, fmt.Errorf("three-choice fixture bounds exceeded")
	}
	a := newArena(family, config)
	choices, roots, result, err := a.compose(family, config)
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
		choices[i].Fallback = choices[i].Options[(FallbackMask(config)>>i)&1].Label
		choices[i].Intent = "caller context;intent: " + Natural(family, choices[i].Kind, i, config, goal, language)
	}
	return pathplan.Plan{Schema: pathplan.Schema, Base: a.base, Decisions: choices}, nil
}

func Choices(plan pathplan.Plan, mask int) (map[string]string, error) {
	if len(plan.Decisions) != 3 || mask < 0 || mask >= 8 {
		return nil, fmt.Errorf("three choices and absolute mask 0..7 required")
	}
	choices := make(map[string]string, 3)
	for i, choice := range plan.Decisions {
		choices[choice.ID] = choice.Options[(mask>>i)&1].Label
	}
	return choices, nil
}
