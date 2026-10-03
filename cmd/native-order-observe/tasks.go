package main

import "fmt"

type orderTask struct {
	ID          string
	Clauses     [2][2]string // English, Korean; source order within each language.
	Assignments [2]string
	Expected    func(int64, int) int64
}

func orderTasks() []orderTask {
	return []orderTask{
		{"add-multiply", [2][2]string{{"add one", "multiply by two"}, {"1을 더한다", "2를 곱한다"}},
			[2]string{"value + 1", "value * 2"}, func(x int64, order int) int64 {
				if order == 0 {
					return (x + 1) * 2
				}
				return x*2 + 1
			}},
		{"subtract-multiply", [2][2]string{{"subtract three", "multiply by two"}, {"3을 뺀다", "2를 곱한다"}},
			[2]string{"value - 3", "value * 2"}, func(x int64, order int) int64 {
				if order == 0 {
					return (x - 3) * 2
				}
				return x*2 - 3
			}},
		{"negate-add", [2][2]string{{"negate the value", "add four"}, {"부호를 반대로 바꾼다", "4를 더한다"}},
			[2]string{"0 - value", "value + 4"}, func(x int64, order int) int64 {
				if order == 0 {
					return -x + 4
				}
				return -(x + 4)
			}},
		{"square-add", [2][2]string{{"square the value", "add one"}, {"값을 제곱한다", "1을 더한다"}},
			[2]string{"value * value", "value + 1"}, func(x int64, order int) int64 {
				if order == 0 {
					return x*x + 1
				}
				return (x + 1) * (x + 1)
			}},
	}
}

func (t orderTask) source() string {
	return fmt.Sprintf("package orders\nnamespace orders\nentity Integer id \"orders://integer\"\nactivity Compose(Integer) -> Integer computes \"let value = input; value = %s; value = %s; return value\"\n", t.Assignments[0], t.Assignments[1])
}

func (t orderTask) intent(language, order int) string {
	prefix, end := "Step: ", "Step: end."
	if language == 1 {
		prefix, end = "단계: ", "단계: 끝."
	}
	return prefix + t.Clauses[language][order] + ". " + prefix + t.Clauses[language][1-order] + ". " + end
}

func (t orderTask) cases(order int, inputs []int64) []map[string]int64 {
	rows := make([]map[string]int64, 0, len(inputs))
	for _, input := range inputs {
		rows = append(rows, map[string]int64{"input": input, "expected": t.Expected(input, order)})
	}
	return rows
}

func (t orderTask) recipe(language, order, budget int) map[string]any {
	operandIntent := "Keep the declared operand order."
	if language == 1 {
		operandIntent = "선언된 피연산자 순서를 유지한다."
	}
	return map[string]any{"schema": "gooo/source-typed-path-recipe/v1", "choices": []any{
		map[string]any{"id": "order", "kind": "root_order", "occurrence": 1, "intent": t.intent(language, order)},
		map[string]any{"id": "first-operands", "kind": "operand_order", "occurrence": 0, "intent": operandIntent},
		map[string]any{"id": "second-operands", "kind": "operand_order", "occurrence": 1, "intent": operandIntent},
	}, "test_cases": t.cases(order, []int64{-2, 0, 3}), "max_attempts": budget}
}
