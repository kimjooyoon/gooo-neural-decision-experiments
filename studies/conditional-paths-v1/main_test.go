package main

import (
	"context"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestAll64ConditionalAssignmentBodiesAreTyped(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		declared := plan(language)
		prepared, err := pathplan.Prepare(declared)
		if err != nil {
			t.Fatal(err)
		}
		for mask := 0; mask < 64; mask++ {
			choices := map[string]string{}
			for i, choice := range declared.Decisions {
				choices[choice.ID] = choice.Options[(mask>>i)&1].Label
			}
			body, err := prepared.Compile(choices)
			if err != nil {
				t.Fatalf("%s mask%d: %v", language, mask, err)
			}
			if _, err := body.Evaluate(-9223372036854775808); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestBoundedSearchKeepsShortBudgetPartialOutcomes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	prepared, err := pathplan.Prepare(plan("en"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []pathplan.TestCase{}
	for _, input := range []int64{-7, -1, 0, 1, 5, 6, 7} {
		cases = append(cases, pathplan.TestCase{Input: input, Expected: Golden(input)})
	}
	short, _, err := prepared.Search(ctx, nil, cases, 8, "")
	if err != nil || short.Status != "PARTIAL" || short.Unattempted != 56 {
		t.Fatalf("short search: %+v %v", short, err)
	}
	full, body, err := prepared.Search(ctx, nil, cases, 64, "")
	if err != nil || full.Status != "TRAINING_COMPLETE" {
		t.Fatalf("full search: %+v %v", full, err)
	}
	for _, input := range []int64{-100, -2, 2, 3, 4, 8, 100, -9223372036854775808, 9223372036854775807} {
		value, err := body.Evaluate(input)
		if err != nil || value.Int != Golden(input) {
			t.Fatalf("independent case %d: %+v %v", input, value, err)
		}
	}
}
