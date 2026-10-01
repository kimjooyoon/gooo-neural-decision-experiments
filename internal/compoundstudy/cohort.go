// Package compoundstudy authors interacting, finite bilingual body choices.
package compoundstudy

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const Configuration = 101
const A, B, C int64 = 4, 14, 10

var Templates = [3]string{"reference_assignment", "operand_branch", "reference_schedule"}

type Document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Max    int                 `json:"max_attempts"`
}
type Case struct {
	ID               string              `json:"id"`
	Template         string              `json:"template"`
	Language         string              `json:"language"`
	Contract         string              `json:"contract"`
	IntendedMask     uint16              `json:"original_intention_mask"`
	FiniteBestMasks  []uint16            `json:"finite_best_masks"`
	FiniteBestPassed int                 `json:"finite_best_passed"`
	Source           string              `json:"gooo_source"`
	Document         Document            `json:"document"`
	Separate         []pathplan.TestCase `json:"separate_input_cases"`
}

func input() bodyplan.Expr            { return bodyplan.Expr{Kind: bodyplan.ExprInput, Name: "input"} }
func integer(v int64) bodyplan.Expr   { return bodyplan.Expr{Kind: bodyplan.ExprInt, Int: v} }
func local(name string) bodyplan.Expr { return bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: name} }
func binary(op string, left, right int) bodyplan.Expr {
	return bodyplan.Expr{Kind: bodyplan.ExprBinary, Operation: op, Left: left, Right: right}
}
func named(kind, name string, expr int) bodyplan.Stmt {
	return bodyplan.Stmt{Kind: kind, Name: name, Expr: expr}
}

func text(kind string, reverse bool, language string) (string, error) {
	if language != "en" && language != "ko" {
		return "", fmt.Errorf("unknown language")
	}
	var en, ko [2]string
	switch kind {
	case pathplan.LocalReference:
		en = [2]string{"Read the first local variable first for the update.", "Read the second local variable second for the update."}
		ko = [2]string{"갱신 값에는 첫 번째 지역 변수 first를 참조한다.", "갱신 값에는 두 번째 지역 변수 second를 참조한다."}
	case pathplan.AssignmentTarget:
		en = [2]string{"Assign the update to the first local variable first.", "Assign the update to the second local variable second."}
		ko = [2]string{"갱신 값을 첫 번째 지역 변수 first에 대입한다.", "갱신 값을 두 번째 지역 변수 second에 대입한다."}
	case pathplan.OperandOrder:
		en = [2]string{"Subtract the offset from the input.", "Subtract the input from the offset."}
		ko = [2]string{"입력값에서 오프셋을 뺀다.", "오프셋에서 입력값을 뺀다."}
	case pathplan.BranchLayout:
		en = [2]string{"Keep multiplication in then and addition in else.", "Swap branches: addition in then and multiplication in else."}
		ko = [2]string{"참 분기는 곱셈, 거짓 분기는 덧셈으로 유지한다.", "분기를 바꿔 참이면 덧셈, 거짓이면 곱셈을 한다."}
	case pathplan.RootOrder:
		en = [2]string{"Execute the first assignment before the second assignment.", "Execute the second assignment before the first assignment."}
		ko = [2]string{"첫 번째 대입을 한 뒤 두 번째 대입을 실행한다.", "두 번째 대입을 한 뒤 첫 번째 대입을 실행한다."}
	default:
		return "", fmt.Errorf("unknown choice kind")
	}
	i := 0
	if reverse {
		i = 1
	}
	value := en[i]
	if language == "ko" {
		value = ko[i]
	}
	if kind == pathplan.OperandOrder || kind == pathplan.BranchLayout {
		return fmt.Sprintf("%s [offset=%d; threshold=%d; addition=%d; multiplier=2]", value, B, A, C), nil
	}
	return fmt.Sprintf("%s [first=input*%d; second=input*%d; offset=%d; update=%d]", value, A, B, B, C), nil
}

// Fixture declares two binary decisions, hence four complete body paths.
func Fixture(template, language string, mask uint16) (pathplan.Plan, error) {
	if mask > 3 {
		return pathplan.Plan{}, fmt.Errorf("four declared masks required")
	}
	base := bodyplan.Plan{Schema: bodyplan.Schema, ID: "compound-study://body/" + template, Name: "ComposePaths", ResultType: decision.TypeInt}
	var choices []pathplan.Choice
	base.Expressions = []bodyplan.Expr{input(), integer(A), integer(B), binary("multiply", 0, 1), binary("multiply", 0, 2)}
	base.Statements = []bodyplan.Stmt{named(bodyplan.StmtLet, "first", 3), named(bodyplan.StmtLet, "second", 4)}
	switch template {
	case "reference_assignment":
		base.Expressions = append(base.Expressions, local("first"), integer(C), binary("add", 5, 6), local("first"), local("second"), binary("subtract", 8, 9))
		base.Statements = append(base.Statements, named(bodyplan.StmtAssign, "first", 7), bodyplan.Stmt{Kind: bodyplan.StmtReturn, Expr: 10})
		base.Root = []int{0, 1, 2, 3}
		choices = []pathplan.Choice{{ID: "read", Kind: pathplan.LocalReference, Target: 5, Options: []pathplan.Option{{Label: "reference_first", Name: "first"}, {Label: "reference_second", Name: "second"}}, Fallback: "reference_first"},
			{ID: "write", Kind: pathplan.AssignmentTarget, Target: 2, Options: []pathplan.Option{{Label: "assign_first", Name: "first"}, {Label: "assign_second", Name: "second"}}, Fallback: "assign_first"}}
	case "operand_branch":
		base.Expressions = []bodyplan.Expr{input(), integer(A), integer(B), binary("subtract", 0, 2), local("value"), binary("less_than", 0, 1), integer(C), binary("add", 4, 6), integer(2), binary("multiply", 4, 8)}
		base.Statements = []bodyplan.Stmt{named(bodyplan.StmtLet, "value", 3), {Kind: bodyplan.StmtIf, Expr: 5, Then: []int{2}, Else: []int{3}}, named(bodyplan.StmtAssign, "value", 9), named(bodyplan.StmtAssign, "value", 7), {Kind: bodyplan.StmtReturn, Expr: 4}}
		base.Root = []int{0, 1, 4}
		choices = []pathplan.Choice{{ID: "operands", Kind: pathplan.OperandOrder, Target: 3, Options: []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}, Fallback: "layout_forward"},
			{ID: "branches", Kind: pathplan.BranchLayout, Target: 1, Options: []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}, Fallback: "layout_forward"}}
	case "reference_schedule":
		base.Expressions = append(base.Expressions, local("first"), integer(C), binary("add", 5, 6), local("first"), integer(2), binary("multiply", 8, 9), local("first"), local("second"), binary("subtract", 11, 12))
		base.Statements = append(base.Statements, named(bodyplan.StmtAssign, "first", 7), named(bodyplan.StmtAssign, "second", 10), bodyplan.Stmt{Kind: bodyplan.StmtReturn, Expr: 13})
		base.Root = []int{0, 1, 2, 3, 4}
		choices = []pathplan.Choice{{ID: "read", Kind: pathplan.LocalReference, Target: 8, Options: []pathplan.Option{{Label: "reference_first", Name: "first"}, {Label: "reference_second", Name: "second"}}, Fallback: "reference_first"},
			{ID: "order", Kind: pathplan.RootOrder, Target: 0, Options: []pathplan.Option{{Label: "schedule_forward", Order: []int{0, 1, 2, 3, 4}}, {Label: "schedule_reverse", Order: []int{0, 1, 3, 2, 4}}}, Fallback: "schedule_forward"}}
	default:
		return pathplan.Plan{}, fmt.Errorf("unknown composition template")
	}
	for i := range choices {
		value, err := text(choices[i].Kind, mask>>i&1 == 1, language)
		if err != nil {
			return pathplan.Plan{}, err
		}
		choices[i].Intent = value
	}
	return pathplan.Plan{Schema: pathplan.Schema, Base: base, Decisions: choices}, nil
}

func Choices(plan pathplan.Plan, mask uint16) map[string]string {
	result := make(map[string]string, len(plan.Decisions))
	for i, choice := range plan.Decisions {
		result[choice.ID] = choice.Options[int(mask>>i&1)].Label
	}
	return result
}
func Mask(plan pathplan.Plan, choices map[string]string) (uint16, error) {
	if len(choices) != 2 || len(plan.Decisions) != 2 {
		return 0, fmt.Errorf("two declared selections required")
	}
	var mask uint16
	for i, c := range plan.Decisions {
		switch choices[c.ID] {
		case c.Options[0].Label:
		case c.Options[1].Label:
			mask |= 1 << i
		default:
			return 0, fmt.Errorf("undeclared selection")
		}
	}
	return mask, nil
}

// Oracle uses arithmetic state transitions, independently of the typed arena.
// int64 arithmetic has the same fixed-width overflow behavior as generated Go.
func Oracle(template string, mask uint16, x int64) (int64, error) {
	if mask > 3 {
		return 0, fmt.Errorf("undeclared mask")
	}
	readSecond, reverse := mask&1 != 0, mask&2 != 0
	first, second := x*A, x*B
	switch template {
	case "reference_assignment":
		value := first
		if readSecond {
			value = second
		}
		value += C
		if reverse {
			second = value
		} else {
			first = value
		}
		return first - second, nil
	case "operand_branch":
		value := x - B
		if readSecond {
			value = B - x
		}
		multiply := x < A
		if reverse {
			multiply = !multiply
		}
		if multiply {
			value *= 2
		} else {
			value += C
		}
		return value, nil
	case "reference_schedule":
		if !reverse {
			first += C
		}
		value := first
		if readSecond {
			value = second
		}
		second = value * 2
		if reverse {
			first += C
		}
		return first - second, nil
	default:
		return 0, fmt.Errorf("unknown template")
	}
}

func row(template, language, contract string, mask uint16) (Case, error) {
	plan, err := Fixture(template, language, mask)
	if err != nil {
		return Case{}, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return Case{}, err
	}
	quoted, err := json.Marshal(prepared.Fallback().GoooBody())
	if err != nil {
		return Case{}, err
	}
	r := Case{ID: fmt.Sprintf("%s-%d-%s-%s", template, mask, language, contract), Template: template, Language: language, Contract: contract, IntendedMask: mask,
		Source: fmt.Sprintf("package compound_study\nnamespace compound_study\nentity Integer id \"compound-study://integer\"\nactivity ComposePaths(Integer) -> Integer computes %s\nactivity Unrelated(Integer) -> Integer computes \"return input\"\n", quoted), Document: Document{Schema: "gooo/body-codegen-typed-path-plan/v1", Plan: plan, Max: 4}}
	inputs := []int64{-7, -1, 0, 1, A - 1, A, A + 1}
	if contract == "sparse" {
		witness := int64(0)
		for x := int64(-64); x <= 64; x++ {
			expected, _ := Oracle(template, mask, x)
			for other := uint16(0); other < 4; other++ {
				actual, _ := Oracle(template, other, x)
				if other != mask && actual == expected {
					witness = x
					goto found
				}
			}
		}
	found:
		inputs = []int64{witness}
	}
	for _, x := range inputs {
		expected, _ := Oracle(template, mask, x)
		r.Document.Cases = append(r.Document.Cases, pathplan.TestCase{Input: x, Expected: expected})
	}
	if contract == "contradictory" {
		first := r.Document.Cases[0]
		first.Expected++
		r.Document.Cases = append(r.Document.Cases, first)
	}
	for _, x := range []int64{-100, -2, 2, 3, 5, 8, 100, math.MinInt64, math.MaxInt64} {
		used := false
		for _, c := range r.Document.Cases {
			used = used || c.Input == x
		}
		if !used {
			expected, _ := Oracle(template, mask, x)
			r.Separate = append(r.Separate, pathplan.TestCase{Input: x, Expected: expected})
		}
	}
	best := -1
	for candidate := uint16(0); candidate < 4; candidate++ {
		body, err := prepared.Compile(Choices(plan, candidate))
		if err != nil {
			return Case{}, err
		}
		passed := 0
		for _, c := range r.Document.Cases {
			v, err := body.Evaluate(c.Input)
			if err != nil {
				return Case{}, err
			}
			expected, _ := Oracle(template, candidate, c.Input)
			if v.Int != expected {
				return Case{}, fmt.Errorf("typed candidate differs from arithmetic oracle")
			}
			if v.Int == c.Expected {
				passed++
			}
		}
		if passed > best {
			best = passed
			r.FiniteBestMasks = []uint16{candidate}
		} else if passed == best {
			r.FiniteBestMasks = append(r.FiniteBestMasks, candidate)
		}
	}
	r.FiniteBestPassed = best
	return r, nil
}

func Cohort() ([]Case, error) {
	rows := make([]Case, 0, 72)
	for _, template := range Templates {
		for mask := uint16(0); mask < 4; mask++ {
			for _, language := range []string{"en", "ko"} {
				for _, contract := range []string{"complete", "sparse", "contradictory"} {
					r, err := row(template, language, contract, mask)
					if err != nil {
						return nil, err
					}
					rows = append(rows, r)
				}
			}
		}
	}
	return rows, nil
}
