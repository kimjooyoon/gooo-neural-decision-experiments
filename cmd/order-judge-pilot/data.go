package main

import (
	"fmt"
	"strings"
)

type task struct {
	ID, Group, Family, Language        string
	SourceOrder, WantedOrder, Template int
	NewConstants, Presentation         bool
}

func tasks() []task {
	var result []task
	for _, group := range []string{"train", "new-template", "presentation", "new-constants"} {
		for _, family := range []string{"add-multiply", "subtract-multiply", "negate-add", "square-add"} {
			for _, language := range []string{"en", "ko"} {
				for sourceOrder := range 2 {
					for wanted := range 2 {
						templates := []int{0}
						if group == "train" {
							templates = []int{0, 1}
						} else if group == "new-template" {
							templates = []int{2}
						}
						for _, template := range templates {
							id := fmt.Sprintf("%s-%s-%s-s%d-w%d-t%d", group, family, language, sourceOrder, wanted, template)
							result = append(result, task{id, group, family, language, sourceOrder, wanted, template, group == "new-constants", group == "presentation"})
						}
					}
				}
			}
		}
	}
	return result
}

func (t task) constants() (int64, int64) {
	switch t.Family {
	case "add-multiply":
		if t.NewConstants {
			return 2, 3
		}
		return 1, 2
	case "subtract-multiply":
		if t.NewConstants {
			return 1, 4
		}
		return 3, 2
	case "negate-add":
		if t.NewConstants {
			return 0, 2
		}
		return 0, 4
	case "square-add":
		if t.NewConstants {
			return 0, 3
		}
		return 0, 1
	}
	panic("unknown family")
}

func (t task) expected(x int64) int64 {
	a, b := t.constants()
	switch t.Family {
	case "add-multiply":
		if t.WantedOrder == 0 {
			return (x + a) * b
		}
		return x*b + a
	case "subtract-multiply":
		if t.WantedOrder == 0 {
			return (x - a) * b
		}
		return x*b - a
	case "negate-add":
		if t.WantedOrder == 0 {
			return -x + b
		}
		return -(x + b)
	case "square-add":
		if t.WantedOrder == 0 {
			return x*x + b
		}
		return (x + b) * (x + b)
	}
	panic("unknown family")
}

func (t task) source() string {
	a, b := t.constants()
	var rhs [2]string
	switch t.Family {
	case "add-multiply":
		rhs = [2]string{fmt.Sprintf("value + %d", a), fmt.Sprintf("value * %d", b)}
	case "subtract-multiply":
		rhs = [2]string{fmt.Sprintf("value - %d", a), fmt.Sprintf("value * %d", b)}
	case "negate-add":
		rhs = [2]string{"0 - value", fmt.Sprintf("value + %d", b)}
	case "square-add":
		rhs = [2]string{"value * value", fmt.Sprintf("value + %d", b)}
	}
	if t.Presentation {
		for i, s := range rhs {
			if strings.Contains(s, " + ") || strings.Contains(s, " * ") {
				parts := strings.Split(s, " ")
				rhs[i] = parts[2] + " " + parts[1] + " " + parts[0]
			}
		}
	}
	source := fmt.Sprintf("package orderpilot\nnamespace orderpilot\nentity Integer id \"orderpilot://integer\"\nactivity Compose(Integer) -> Integer computes \"let value = input; value = %s; value = %s; return value\"\n", rhs[t.SourceOrder], rhs[1-t.SourceOrder])
	if t.Presentation {
		source = strings.ReplaceAll(source, "value", "state")
	}
	return source
}

func (t task) intent() string {
	a, b := t.constants()
	numbers := []string{"zero", "one", "two", "three", "four"}
	var clauses [2]string
	if t.Language == "en" {
		switch t.Family {
		case "add-multiply":
			clauses = [2]string{"add " + numbers[a], "multiply by " + numbers[b]}
		case "subtract-multiply":
			clauses = [2]string{"subtract " + numbers[a], "multiply by " + numbers[b]}
		case "negate-add":
			clauses = [2]string{"negate the value", "add " + numbers[b]}
		case "square-add":
			clauses = [2]string{"square the value", "add " + numbers[b]}
		}
		formats := []string{"Step: %s. Step: %s. Step: end.", "First %s; after that %s.", "Compute in this order: %s, then %s."}
		return fmt.Sprintf(formats[t.Template], clauses[t.WantedOrder], clauses[1-t.WantedOrder])
	}
	switch t.Family {
	case "add-multiply":
		clauses = [2]string{fmt.Sprintf("%d을 더한다", a), fmt.Sprintf("%d를 곱한다", b)}
	case "subtract-multiply":
		clauses = [2]string{fmt.Sprintf("%d을 뺀다", a), fmt.Sprintf("%d를 곱한다", b)}
	case "negate-add":
		clauses = [2]string{"부호를 반대로 바꾼다", fmt.Sprintf("%d을 더한다", b)}
	case "square-add":
		clauses = [2]string{"값을 제곱한다", fmt.Sprintf("%d을 더한다", b)}
	}
	formats := []string{"단계: %s. 단계: %s. 단계: 끝.", "먼저 %s. 그다음 %s.", "계산 순서는 다음과 같다: %s. 이어서 %s."}
	return fmt.Sprintf(formats[t.Template], clauses[t.WantedOrder], clauses[1-t.WantedOrder])
}

func (t task) recipe() map[string]any {
	operand := "Keep the declared operand order."
	if t.Language == "ko" {
		operand = "선언된 피연산자 순서를 유지한다."
	}
	var cases []map[string]int64
	for _, x := range []int64{-2, 0, 3} {
		cases = append(cases, map[string]int64{"input": x, "expected": t.expected(x)})
	}
	return map[string]any{"schema": "gooo/source-typed-path-recipe/v1", "choices": []any{
		map[string]any{"id": "order", "kind": "root_order", "occurrence": 1, "intent": t.intent()},
		map[string]any{"id": "first-operands", "kind": "operand_order", "occurrence": 0, "intent": operand},
		map[string]any{"id": "second-operands", "kind": "operand_order", "occurrence": 1, "intent": operand},
	}, "test_cases": cases, "max_attempts": 8}
}
