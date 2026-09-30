// Package pathstudy defines compiler-owned structural fixtures and independent
// arithmetic oracles. Oracle does not interpret or inspect the assembled plan.
package pathstudy

import (
	"fmt"
	"math"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var Families = [5]string{pathplan.LocalReference, pathplan.AssignmentTarget, pathplan.OperandOrder, pathplan.BranchLayout, pathplan.RootOrder}

func Parameters(config int) (offset, factor int64) { return int64(config - 32), int64(2 + config%7) }

func Fixture(family string, config int, intent string) (pathplan.Plan, error) {
	if config < 0 || config >= 96 {
		return pathplan.Plan{}, fmt.Errorf("configuration outside 0..95")
	}
	a, b := Parameters(config)
	input := bodyplan.Expr{Kind: bodyplan.ExprInput, Name: "input"}
	integer := func(value int64) bodyplan.Expr { return bodyplan.Expr{Kind: bodyplan.ExprInt, Int: value} }
	local := func(name string) bodyplan.Expr { return bodyplan.Expr{Kind: bodyplan.ExprLocal, Name: name} }
	binary := func(op string, left, right int) bodyplan.Expr {
		return bodyplan.Expr{Kind: bodyplan.ExprBinary, Operation: op, Left: left, Right: right}
	}
	base := bodyplan.Plan{Schema: bodyplan.Schema, ID: fmt.Sprintf("path-%s-%02d", family, config), Name: "ChoosePath", ResultType: decision.TypeInt}
	choice := pathplan.Choice{ID: "structure", Kind: family, Intent: intent}
	switch family {
	case pathplan.LocalReference:
		base.Expressions = []bodyplan.Expr{input, integer(a), binary("add", 0, 1), integer(b), binary("multiply", 0, 3), local("first"), local("first"), local("second"), binary("add", 6, 7), binary("add", 5, 8)}
		base.Statements = []bodyplan.Stmt{{Kind: "let", Name: "first", Expr: 2}, {Kind: "let", Name: "second", Expr: 4}, {Kind: "return", Expr: 9}}
		base.Root = []int{0, 1, 2}
		choice.Target, choice.Fallback = 5, "reference_first"
		choice.Options = []pathplan.Option{{Label: "reference_first", Name: "first"}, {Label: "reference_second", Name: "second"}}
	case pathplan.AssignmentTarget:
		base.Expressions = []bodyplan.Expr{input, integer(a), binary("add", 0, 1), integer(b), binary("multiply", 0, 3), integer(a + 7), binary("add", 0, 5), local("first"), local("second"), binary("subtract", 7, 8)}
		base.Statements = []bodyplan.Stmt{{Kind: "let", Name: "first", Expr: 2}, {Kind: "let", Name: "second", Expr: 4}, {Kind: "assign", Name: "first", Expr: 6}, {Kind: "return", Expr: 9}}
		base.Root = []int{0, 1, 2, 3}
		choice.Target, choice.Fallback = 2, "assign_first"
		choice.Options = []pathplan.Option{{Label: "assign_first", Name: "first"}, {Label: "assign_second", Name: "second"}}
	case pathplan.OperandOrder:
		base.Expressions = []bodyplan.Expr{input, integer(a), binary("subtract", 0, 1)}
		base.Statements, base.Root = []bodyplan.Stmt{{Kind: "return", Expr: 2}}, []int{0}
		choice.Target, choice.Fallback = 2, "layout_forward"
		choice.Options = []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}
	case pathplan.BranchLayout:
		base.Expressions = []bodyplan.Expr{input, integer(a), binary("less_equal", 0, 1), integer(b), binary("multiply", 0, 3), integer(a + 7), binary("add", 0, 5)}
		base.Statements = []bodyplan.Stmt{{Kind: "if", Expr: 2, Then: []int{1}, Else: []int{2}}, {Kind: "return", Expr: 4}, {Kind: "return", Expr: 6}}
		base.Root = []int{0}
		choice.Target, choice.Fallback = 0, "layout_forward"
		choice.Options = []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}
	case pathplan.RootOrder:
		base.Expressions = []bodyplan.Expr{input, integer(a), binary("add", 0, 1), integer(b), binary("multiply", 0, 3), local("first"), local("second"), binary("add", 5, 6), binary("subtract", 6, 5), binary("multiply", 5, 6)}
		base.Statements = []bodyplan.Stmt{{Kind: "let", Name: "first", Expr: 2}, {Kind: "let", Name: "second", Expr: 4}, {Kind: "assign", Name: "first", Expr: 7}, {Kind: "assign", Name: "second", Expr: 8}, {Kind: "return", Expr: 9}}
		base.Root = []int{0, 1, 2, 3, 4}
		choice.Target, choice.Fallback = 0, "schedule_forward"
		choice.Options = []pathplan.Option{{Label: "schedule_forward", Order: []int{0, 1, 2, 3, 4}}, {Label: "schedule_reverse", Order: []int{0, 1, 3, 2, 4}}}
	default:
		return pathplan.Plan{}, fmt.Errorf("unknown structural family")
	}
	return pathplan.Plan{Schema: pathplan.Schema, Base: base, Decisions: []pathplan.Choice{choice}}, nil
}

func GoldLabel(family string, reverse bool) string {
	index := 0
	if reverse {
		index = 1
	}
	switch family {
	case pathplan.LocalReference:
		return [2]string{"reference_first", "reference_second"}[index]
	case pathplan.AssignmentTarget:
		return [2]string{"assign_first", "assign_second"}[index]
	case pathplan.OperandOrder, pathplan.BranchLayout:
		return [2]string{"layout_forward", "layout_reverse"}[index]
	case pathplan.RootOrder:
		return [2]string{"schedule_forward", "schedule_reverse"}[index]
	default:
		return ""
	}
}

func Oracle(family string, reverse bool, config int, input int64) (int64, error) {
	if config < 0 || config >= 96 {
		return 0, fmt.Errorf("configuration outside 0..95")
	}
	a, b := Parameters(config)
	switch family {
	case pathplan.LocalReference:
		first, second := input+a, input*b
		if reverse {
			return first + second + second, nil
		}
		return first + second + first, nil
	case pathplan.AssignmentTarget:
		if reverse {
			return (input + a) - (input + a + 7), nil
		}
		return (input + a + 7) - (input * b), nil
	case pathplan.OperandOrder:
		if reverse {
			return a - input, nil
		}
		return input - a, nil
	case pathplan.BranchLayout:
		multiply := input <= a
		if reverse {
			multiply = !multiply
		}
		if multiply {
			return input * b, nil
		}
		return input + a + 7, nil
	case pathplan.RootOrder:
		first, second := input+a, input*b
		if reverse {
			second = second - first
			first = first + second
		} else {
			first = first + second
			second = second - first
		}
		return first * second, nil
	default:
		return 0, fmt.Errorf("unknown structural family")
	}
}

// ProbeInstruction is reserved for configurations 64..95 and new templates.
// These examples are never included in training/checkpoint calibration.
func ProbeInstruction(family string, reverse bool, config int, language string) (string, error) {
	if config < 64 || config >= 96 || language != "en" && language != "ko" {
		return "", fmt.Errorf("probe outside reserved group")
	}
	var en, ko [2]string
	switch family {
	case pathplan.LocalReference:
		en = [2]string{"Take the sum of both locals and add another copy of the first one.", "Take the sum of both locals and add another copy of the second one."}
		ko = [2]string{"두 지역 변수의 합에 첫째 변수 값을 추가로 합친다.", "두 지역 변수의 합에 둘째 변수 값을 추가로 합친다."}
	case pathplan.AssignmentTarget:
		en = [2]string{"Send the update into the first variable and leave the other as it is.", "Send the update into the second variable and leave the other as it is."}
		ko = [2]string{"첫째 변수만 새 값으로 갱신하고 나머지 변수는 그대로 둔다.", "둘째 변수만 새 값으로 갱신하고 나머지 변수는 그대로 둔다."}
	case pathplan.OperandOrder:
		en = [2]string{"Subtract the offset from the input.", "Subtract the input from the offset."}
		ko = [2]string{"입력값에서 오프셋을 차감한다.", "오프셋에서 입력값을 차감한다."}
	case pathplan.BranchLayout:
		en = [2]string{"When the comparison holds use multiplication, otherwise use addition.", "When the comparison holds use addition, otherwise use multiplication."}
		ko = [2]string{"조건을 만족하면 곱하고, 그렇지 않으면 더한다.", "조건을 만족하면 더하고, 그렇지 않으면 곱한다."}
	case pathplan.RootOrder:
		en = [2]string{"Execute the first variable update, followed by the second.", "Execute the second variable update, followed by the first."}
		ko = [2]string{"첫째 변수 갱신을 끝낸 다음 둘째 변수 갱신에 착수한다.", "둘째 변수 갱신을 끝낸 다음 첫째 변수 갱신에 착수한다."}
	default:
		return "", fmt.Errorf("unknown probe structural family")
	}
	index := 0
	if reverse {
		index = 1
	}
	text := en[index]
	if language == "ko" {
		text = ko[index]
	}
	a, b := Parameters(config)
	return fmt.Sprintf("%s [offset=%d; factor=%d; updated_offset=%d]", text, a, b, a+7), nil
}

func Inputs(config int) []int64 {
	a, _ := Parameters(config)
	values := []int64{math.MinInt64, -123, -7, -1, 0, 1, 5, 99, math.MaxInt64, a - 1, a, a + 1}
	seen := make(map[int64]bool)
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

// Instruction templates are partitioned as 0..3 train, 4..5 calibration,
// 6..7 test. Numeric configurations are partitioned independently.
func Instruction(family string, reverse bool, config, template int) (string, string, error) {
	if template < 0 || template >= 8 || config < 0 || config >= 64 {
		return "", "", fmt.Errorf("instruction outside frozen bounds")
	}
	a, b := Parameters(config)
	var verbs [8][2]string
	switch family {
	case pathplan.LocalReference:
		verbs = [8][2]string{
			{"Add the first stored value to the combined total.", "Add the second stored value to the combined total."},
			{"두 변수의 합계에 첫 번째 변수 값을 더하라.", "두 변수의 합계에 두 번째 변수 값을 더하라."},
			{"Read the first binding for the extra term in the total.", "Read the second binding for the extra term in the total."},
			{"합계의 추가 항에는 첫 번째 지역 변수를 사용한다.", "합계의 추가 항에는 두 번째 지역 변수를 사용한다."},
			{"Choose the first local to add to the two-variable total.", "Choose the second local to add to the two-variable total."},
			{"두 변수 합계에 더할 값은 첫 번째 저장 변수에서 읽는다.", "두 변수 합계에 더할 값은 두 번째 저장 변수에서 읽는다."},
			{"Increase the combined total by the first local variable.", "Increase the combined total by the second local variable."},
			{"두 변수의 합계에 첫 번째 변수의 값을 한 번 더 합산한다.", "두 변수의 합계에 두 번째 변수의 값을 한 번 더 합산한다."},
		}
	case pathplan.AssignmentTarget:
		verbs = [8][2]string{
			{"Write the update to the first stored value.", "Write the update to the second stored value."},
			{"갱신 값을 첫 번째 변수에 할당하라.", "갱신 값을 두 번째 변수에 할당하라."},
			{"Replace the first local with the update.", "Replace the second local with the update."},
			{"첫 번째 지역 변수에 새 값을 대입한다.", "두 번째 지역 변수에 새 값을 대입한다."},
			{"Assign the update into the first binding.", "Assign the update into the second binding."},
			{"대입 대상은 첫 번째 저장 변수다.", "대입 대상은 두 번째 저장 변수다."},
			{"Store the changed value in the first local variable.", "Store the changed value in the second local variable."},
			{"새로 계산한 값을 첫 번째 변수에 저장한다.", "새로 계산한 값을 두 번째 변수에 저장한다."},
		}
	case pathplan.OperandOrder:
		verbs = [8][2]string{
			{"Keep input before offset in the subtraction.", "Reverse subtraction operands: offset before input."},
			{"뺄셈에서 입력 뒤에 상수를 둔다.", "뺄셈 피연산자 순서를 뒤집어 상수에서 입력을 뺀다."},
			{"Preserve the subtraction operand order.", "Swap the two subtraction operands."},
			{"입력에서 상수를 빼는 순서를 유지한다.", "상수에서 입력을 빼도록 좌우를 교환한다."},
			{"Use the existing left-to-right subtraction.", "Exchange left and right in the subtraction."},
			{"좌우 피연산자는 현재 순서대로 사용한다.", "좌우 피연산자는 반대 순서대로 사용한다."},
			{"Retain input minus offset as written.", "Invert the operand placement to offset minus input."},
			{"입력에서 상수를 빼는 배치를 그대로 둔다.", "입력과 상수의 위치를 바꿔 뺄셈을 수행한다."},
		}
	case pathplan.BranchLayout:
		verbs = [8][2]string{
			{"Keep multiplication in the then branch and addition in else.", "Swap branches: addition in then, multiplication in else."},
			{"참 분기는 곱셈, 거짓 분기는 덧셈으로 유지한다.", "분기를 교환해 참이면 덧셈, 거짓이면 곱셈을 한다."},
			{"Preserve the declared then and else layout.", "Reverse the declared then and else layout."},
			{"현재 조건문의 두 분기 배치를 유지한다.", "조건문의 참 분기와 거짓 분기를 맞바꾼다."},
			{"Retain the current conditional branch bodies.", "Exchange the conditional branch bodies."},
			{"분기 본문을 현재 위치 그대로 실행한다.", "분기 본문을 서로 반대 위치로 옮긴다."},
			{"Leave the then and else bodies in their original positions.", "Interchange the then and else bodies."},
			{"참일 때와 거짓일 때의 본문을 원래대로 둔다.", "참일 때와 거짓일 때 실행할 본문을 서로 바꾼다."},
		}
	case pathplan.RootOrder:
		verbs = [8][2]string{
			{"Run the first assignment before the second assignment.", "Run the second assignment before the first assignment."},
			{"첫 번째 대입 후 두 번째 대입을 실행한다.", "두 번째 대입 후 첫 번째 대입을 실행한다."},
			{"Keep the first-to-second assignment schedule.", "Reverse the assignment schedule to second-to-first."},
			{"첫 대입을 먼저 처리하고 둘째 대입을 나중에 처리한다.", "둘째 대입을 먼저 처리하고 첫 대입을 나중에 처리한다."},
			{"Schedule the first update ahead of the second update.", "Schedule the second update ahead of the first update."},
			{"대입 순서는 첫 번째 다음 두 번째다.", "대입 순서는 두 번째 다음 첫 번째다."},
			{"Complete the first assignment, then perform the second.", "Complete the second assignment, then perform the first."},
			{"첫 번째 대입을 마친 뒤 두 번째 대입으로 진행한다.", "두 번째 대입을 마친 뒤 첫 번째 대입으로 진행한다."},
		}
	default:
		return "", "", fmt.Errorf("unknown structural family")
	}
	index := 0
	if reverse {
		index = 1
	}
	language := "en"
	if template%2 == 1 {
		language = "ko"
	}
	text := fmt.Sprintf("%s [offset=%d; factor=%d; updated_offset=%d]", verbs[template][index], a, b, a+7)
	return text, language, nil
}

func View(plain, family, view string, config int) (string, error) {
	switch view {
	case "plain":
		return plain, nil
	case "gooo":
		fixture, err := Fixture(family, config, plain)
		if err != nil {
			return "", err
		}
		program, err := pathplan.Compile(fixture, map[string]string{"structure": fixture.Decisions[0].Fallback})
		if err != nil {
			return "", err
		}
		lines := strings.Split(strings.TrimSpace(program.GoooSource()), "\n")
		return lines[len(lines)-1] + "\nintent: " + plain, nil
	case "prov":
		return "prov:Activity gooo:CompileBody; prov:used gooo:TypedPathPlan; prov:wasAssociatedWith gooo:TinyPathModel; kind=" + family + "; intent: " + plain, nil
	default:
		return "", fmt.Errorf("unknown instruction view")
	}
}
