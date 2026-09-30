package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

// Golden uses independent arithmetic, never the arena or model selection.
func Golden(input int64) int64 {
	if input > 0 && input <= 5 {
		return 2*input - 20
	}
	return 2*input + 10
}

func plan(language string) pathplan.Plan {
	base := bodyplan.Plan{Schema: bodyplan.Schema, ID: "boolean-comparison-assignment-order", Name: "ConditionalAssign", ResultType: decision.TypeInt,
		Expressions: []bodyplan.Expr{
			{Kind: "input", Name: "input"}, {Kind: "int", Int: 0}, {Kind: "int", Int: 5}, {Kind: "int", Int: 10},
			{Kind: "binary", Operation: "less_than", Left: 0, Right: 1}, {Kind: "binary", Operation: "less_equal", Left: 0, Right: 2},
			{Kind: "local", Name: "guard"}, {Kind: "local", Name: "guard_alt"}, {Kind: "binary", Operation: "and", Left: 6, Right: 7},
			{Kind: "binary", Operation: "add", Left: 0, Right: 2}, {Kind: "binary", Operation: "subtract", Left: 0, Right: 3},
			{Kind: "local", Name: "left"}, {Kind: "local", Name: "right"}, {Kind: "binary", Operation: "add", Left: 11, Right: 12},
			{Kind: "local", Name: "guard"},
		},
		Statements: []bodyplan.Stmt{
			{Kind: "let", Name: "left", Expr: 9}, {Kind: "let", Name: "right", Expr: 10}, {Kind: "let", Name: "guard", Expr: 4}, {Kind: "let", Name: "guard_alt", Expr: 5},
			{Kind: "assign", Name: "guard", Expr: 8}, {Kind: "if", Expr: 14, Then: []int{6}, Else: []int{7}},
			{Kind: "assign", Name: "left", Expr: 9}, {Kind: "assign", Name: "left", Expr: 10}, {Kind: "return", Expr: 13},
		}, Root: []int{0, 1, 2, 3, 4, 5, 8},
	}
	intents := []string{"Reverse the comparison operands so zero is on the left.", "Use the second Boolean local as the if condition.", "Store the Boolean conjunction in the second guard.", "Write the first branch's computed integer to the second variable.", "Swap the then and else bodies.", "Keep the Boolean update before the branch."}
	if language == "ko" {
		intents = []string{"비교 피연산자를 뒤집어 0을 왼쪽에 놓아라.", "if 조건은 두 번째 불리언 지역 변수를 사용해라.", "불리언 논리곱을 두 번째 조건 변수에 할당해라.", "첫 분기의 계산한 정수를 두 번째 변수에 할당해라.", "then과 else 본문을 서로 교환해라.", "불리언 갱신을 분기보다 먼저 실행해라."}
	}
	choices := []pathplan.Choice{
		{ID: "comparison", Kind: pathplan.OperandOrder, Target: 4, Fallback: "layout_forward", Options: []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}},
		{ID: "condition", Kind: pathplan.LocalReference, Target: 14, Fallback: "reference_first", Options: []pathplan.Option{{Label: "reference_first", Name: "guard"}, {Label: "reference_second", Name: "guard_alt"}}},
		{ID: "boolean_target", Kind: pathplan.AssignmentTarget, Target: 4, Fallback: "assign_first", Options: []pathplan.Option{{Label: "assign_first", Name: "guard"}, {Label: "assign_second", Name: "guard_alt"}}},
		{ID: "integer_target", Kind: pathplan.AssignmentTarget, Target: 6, Fallback: "assign_first", Options: []pathplan.Option{{Label: "assign_first", Name: "left"}, {Label: "assign_second", Name: "right"}}},
		{ID: "branches", Kind: pathplan.BranchLayout, Target: 5, Fallback: "layout_forward", Options: []pathplan.Option{{Label: "layout_forward"}, {Label: "layout_reverse", Reverse: true}}},
		{ID: "order", Kind: pathplan.RootOrder, Target: 0, Fallback: "schedule_forward", Options: []pathplan.Option{{Label: "schedule_forward", Order: []int{0, 1, 2, 3, 4, 5, 8}}, {Label: "schedule_reverse", Order: []int{0, 1, 2, 3, 5, 4, 8}}}},
	}
	for i := range choices {
		choices[i].Intent = "Gooo Integer input; typed Boolean guards and mutable Integer locals. intent: " + intents[i]
	}
	return pathplan.Plan{Schema: pathplan.Schema, Base: base, Decisions: choices}
}

func save(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

func main() {
	if len(os.Args) != 2 {
		panic("fresh output directory required")
	}
	output := os.Args[1]
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("output must be fresh")
	}
	entries := map[string]string{}
	write := func(name string, raw []byte) {
		if err := save(filepath.Join(output, name), raw); err != nil {
			panic(err)
		}
		h := sha256.Sum256(raw)
		entries[name] = hex.EncodeToString(h[:])
	}
	prepared, err := pathplan.Prepare(plan("en"))
	if err != nil {
		panic(err)
	}
	body, _ := json.Marshal(prepared.Fallback().GoooBody())
	source := fmt.Sprintf("package pathstudy\nnamespace pathstudy\nentity Integer id \"pathstudy://entity/integer\"\nactivity ConditionalAssign(Integer) -> Integer computes %s\nactivity Unrelated(Integer) -> Integer computes \"return input\"\n", body)
	write("conditional-assignment.gooo.fixture", []byte(source))
	for _, language := range []string{"en", "ko"} {
		for _, budget := range []int{8, 64} {
			cases := []pathplan.TestCase{}
			for _, input := range []int64{-7, -1, 0, 1, 5, 6, 7} {
				cases = append(cases, pathplan.TestCase{Input: input, Expected: Golden(input)})
			}
			document := map[string]any{"schema": "gooo/body-codegen-typed-path-plan/v1", "path_plan": plan(language), "test_cases": cases, "max_attempts": budget}
			raw, err := json.MarshalIndent(document, "", "  ")
			if err != nil {
				panic(err)
			}
			write(fmt.Sprintf("%s-budget-%d.json", language, budget), append(raw, '\n'))
		}
	}
	gold := map[string]string{"comparison": "layout_reverse", "condition": "reference_second", "boolean_target": "assign_second", "integer_target": "assign_second", "branches": "layout_reverse", "order": "schedule_forward"}
	oracle, err := json.MarshalIndent(gold, "", "  ")
	if err != nil {
		panic(err)
	}
	write("structural-oracle.json", append(oracle, '\n'))
	holdout := []pathplan.TestCase{}
	for _, input := range []int64{-100, -2, 2, 3, 4, 8, 100, -1 << 63, 1<<63 - 1} {
		holdout = append(holdout, pathplan.TestCase{Input: input, Expected: Golden(input)})
	}
	holdoutBytes, err := json.MarshalIndent(holdout, "", "  ")
	if err != nil {
		panic(err)
	}
	write("holdout-cases.json", append(holdoutBytes, '\n'))
	program, err := prepared.Compile(gold)
	if err != nil {
		panic(err)
	}
	for _, input := range []int64{-100, -7, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 100, -1 << 63, 1<<63 - 1} {
		value, err := program.Evaluate(input)
		if err != nil || value.Int != Golden(input) {
			panic("gold typed assembly differs from independent arithmetic")
		}
	}
	manifest := map[string]any{"schema": "gooo/conditional-assignment-cohort/v1", "distinct_compound_intents": 1, "languages": []string{"en", "ko"}, "decisions_per_plan": 6, "declared_combinations": 64, "fixture_files_sha256": entries, "generator_model_predictions": 0, "external_provider_calls": 0, "optimizer_steps": 0, "scope": "Boolean comparison operand order, Boolean local/assignment, Integer assignment, if layout and sequencing under the existing closed path model ABI; language/budget views are not separate ideas."}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := save(filepath.Join(output, "manifest.json"), append(raw, '\n')); err != nil {
		panic(err)
	}
	fmt.Println("Generated one six-decision conditional assignment study; model calls: 0")
}
