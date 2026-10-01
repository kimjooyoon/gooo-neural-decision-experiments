// Package familystudy authors a finite bilingual native continuation cohort.
package familystudy

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type Document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Max    int                 `json:"max_attempts"`
}
type Case struct {
	ID               string              `json:"id"`
	Family           string              `json:"family"`
	Configuration    int                 `json:"configuration"`
	Reverse          bool                `json:"reverse"`
	Language         string              `json:"language"`
	Contract         string              `json:"contract"`
	IntentionLabel   string              `json:"original_intention_label"`
	FiniteBestLabels []string            `json:"finite_best_labels"`
	FiniteBestPassed int                 `json:"finite_best_passed"`
	Source           string              `json:"gooo_source"`
	Document         Document            `json:"document"`
	Separate         []pathplan.TestCase `json:"separate_input_cases"`
}

func expected(family string, reverse bool, config int, inputs []int64) ([]pathplan.TestCase, error) {
	result := make([]pathplan.TestCase, 0, len(inputs))
	for _, input := range inputs {
		value, err := pathstudy.Oracle(family, reverse, config, input)
		if err != nil {
			return nil, err
		}
		result = append(result, pathplan.TestCase{Input: input, Expected: value})
	}
	return result, nil
}
func row(family string, config int, reverse bool, language, contract string) (Case, error) {
	intent, err := pathstudy.ProbeInstruction(family, reverse, config, language)
	if err != nil {
		return Case{}, err
	}
	plan, err := pathstudy.Fixture(family, config, intent)
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
	source := fmt.Sprintf("package family_study\nnamespace family_study\nentity Integer id \"family-study://integer\"\nactivity ChoosePath(Integer) -> Integer computes %s\nactivity Unrelated(Integer) -> Integer computes \"return input\"\n", quoted)
	a, _ := pathstudy.Parameters(config)
	inputs := []int64{-7, -1, 0, 1, a - 1, a, a + 1}
	if contract == "sparse" {
		input := inputs[0]
		// Find a finite ambiguity witness when one exists in this declared range.
		for x := int64(-64); x <= 64; x++ {
			left, _ := pathstudy.Oracle(family, false, config, x)
			right, _ := pathstudy.Oracle(family, true, config, x)
			if left == right {
				input = x
				break
			}
		}
		inputs = []int64{input}
	}
	cases, err := expected(family, reverse, config, inputs)
	if err != nil {
		return Case{}, err
	}
	if contract == "contradictory" {
		cases = append(cases, pathplan.TestCase{Input: cases[0].Input, Expected: cases[0].Expected + 1})
	}
	separate, err := expected(family, reverse, config, []int64{-100, -2, 2, 3, 5, 8, 100, math.MinInt64, math.MaxInt64})
	if err != nil {
		return Case{}, err
	}
	// Sparse witnesses can coincide with the proposed separate set; retain only
	// genuinely unselected inputs and their explicit denominator.
	filtered := separate[:0]
	for _, c := range separate {
		used := false
		for _, selected := range cases {
			used = used || c.Input == selected.Input
		}
		if !used {
			filtered = append(filtered, c)
		}
	}
	result := Case{ID: fmt.Sprintf("%s-%d-%t-%s-%s", family, config, reverse, language, contract), Family: family, Configuration: config,
		Reverse: reverse, Language: language, Contract: contract, IntentionLabel: pathstudy.GoldLabel(family, reverse), Source: source,
		Document: Document{Schema: "gooo/body-codegen-typed-path-plan/v1", Plan: plan, Cases: cases, Max: 2}, Separate: filtered}
	best := -1
	for _, direction := range []bool{false, true} {
		label := pathstudy.GoldLabel(family, direction)
		body, err := prepared.Compile(map[string]string{"structure": label})
		if err != nil {
			return Case{}, err
		}
		passed := 0
		for _, c := range cases {
			actual, err := body.Evaluate(c.Input)
			if err != nil {
				return Case{}, err
			}
			oracle, _ := pathstudy.Oracle(family, direction, config, c.Input)
			if actual.Int != oracle {
				return Case{}, fmt.Errorf("typed option differs from independent oracle")
			}
			if actual.Int == c.Expected {
				passed++
			}
		}
		if passed > best {
			best = passed
			result.FiniteBestLabels = []string{label}
		} else if passed == best {
			result.FiniteBestLabels = append(result.FiniteBestLabels, label)
		}
	}
	result.FiniteBestPassed = best
	return result, nil
}
func Cohort() ([]Case, error) {
	result := make([]Case, 0, 120)
	for _, family := range pathstudy.Families {
		for _, config := range []int{64, 79} {
			for _, reverse := range []bool{false, true} {
				for _, language := range []string{"en", "ko"} {
					for _, contract := range []string{"complete", "sparse", "contradictory"} {
						c, err := row(family, config, reverse, language, contract)
						if err != nil {
							return nil, err
						}
						result = append(result, c)
					}
				}
			}
		}
	}
	return result, nil
}
