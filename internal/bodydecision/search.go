package bodydecision

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

type Case struct {
	Input    int64          `json:"input"`
	Expected bodyplan.Value `json:"expected"`
}

func (test *Case) UnmarshalJSON(data []byte) error {
	var fields struct {
		Input    json.RawMessage `json:"input"`
		Expected json.RawMessage `json:"expected"`
	}
	if err := strictjson.Decode(data, &fields); err != nil {
		return err
	}
	if len(fields.Input) == 0 || bytes.Equal(bytes.TrimSpace(fields.Input), []byte("null")) || len(fields.Expected) == 0 || bytes.Equal(bytes.TrimSpace(fields.Expected), []byte("null")) {
		return errors.New("case requires explicit non-null input and expected")
	}
	var input int64
	if err := json.Unmarshal(fields.Input, &input); err != nil {
		return err
	}
	var expected bodyplan.Value
	if err := strictjson.Decode(fields.Expected, &expected); err != nil {
		return err
	}
	if expected.Type != decision.TypeInt && expected.Type != decision.TypeBool {
		return errors.New("case expected type must be Int or Bool")
	}
	var rawExpected map[string]json.RawMessage
	if err := json.Unmarshal(fields.Expected, &rawExpected); err != nil {
		return err
	}
	for _, name := range []string{"int", "bool"} {
		if value, exists := rawExpected[name]; exists && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return errors.New("case expected scalar cannot be null")
		}
	}
	if (expected.Type == decision.TypeInt && expected.Bool) || (expected.Type == decision.TypeBool && expected.Int != 0) {
		return errors.New("case expected contains a value for a different type")
	}
	*test = Case{Input: input, Expected: expected}
	return nil
}

type Score struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

func ScoreProgram(program *bodyplan.Program, cases []Case) (Score, error) {
	result := Score{Total: len(cases)}
	if program == nil {
		return result, errors.New("program is nil")
	}
	for _, test := range cases {
		got, err := program.Evaluate(test.Input)
		if err != nil {
			return result, err
		}
		if test.Expected.Type != decision.TypeInt && test.Expected.Type != decision.TypeBool {
			return result, errors.New("expected case type must be Int or Bool")
		}
		if got.Type != test.Expected.Type {
			return result, errors.New("case and program result types differ")
		}
		if (got.Type == decision.TypeInt && got.Int == test.Expected.Int) || (got.Type == decision.TypeBool && got.Bool == test.Expected.Bool) {
			result.Correct++
		}
	}
	return result, nil
}

type Attempt struct {
	Choices map[string]string `json:"choices"`
	Score   Score             `json:"training_score"`
}

type SearchResult struct {
	Choices    map[string]string `json:"choices"`
	Initial    Score             `json:"initial_training_score"`
	Final      Score             `json:"final_training_score"`
	Attempts   []Attempt         `json:"attempts"`
	Limit      int               `json:"maximum_attempts"`
	StopReason string            `json:"stop_reason"`
	ModelCalls int               `json:"feedback_model_calls"`
}

// Search uses training examples only. It is finite deterministic enumeration,
// not a learned feedback policy; hidden cases are supplied only to later scoring.
func Search(plan bodyplan.Plan, initial map[string]string, training []Case, limit int) (SearchResult, error) {
	if limit < 1 || limit > 128 || len(training) < 1 || len(training) > 4096 {
		return SearchResult{}, errors.New("search requires 1..128 attempts and 1..4096 training cases")
	}
	if _, err := Validate(plan); err != nil {
		return SearchResult{}, err
	}
	first, err := bodyplan.Compile(plan, initial)
	if err != nil {
		return SearchResult{}, err
	}
	score, err := ScoreProgram(first, training)
	if err != nil {
		return SearchResult{}, err
	}
	result := SearchResult{Choices: CloneChoices(initial), Initial: score, Final: score, Limit: limit, Attempts: []Attempt{{Choices: CloneChoices(initial), Score: score}}}
	if score.Correct == score.Total {
		result.StopReason = "all_training_cases_passed"
		return result, nil
	}
	var holes []bodyplan.Expr
	for _, expr := range plan.Expressions {
		if expr.Kind == "hole" {
			holes = append(holes, expr)
		}
	}
	indices := make([]int, len(holes))
	exhausted := len(holes) == 0
	for !exhausted && len(result.Attempts) < limit {
		trial := make(map[string]string, len(holes))
		for i, hole := range holes {
			trial[hole.HoleID] = hole.Allowed[indices[i]]
		}
		if !sameChoices(trial, initial) {
			program, compileErr := bodyplan.Compile(plan, trial)
			if compileErr != nil {
				return result, fmt.Errorf("prevalidated candidate: %w", compileErr)
			}
			candidateScore, scoreErr := ScoreProgram(program, training)
			if scoreErr != nil {
				return result, scoreErr
			}
			result.Attempts = append(result.Attempts, Attempt{Choices: CloneChoices(trial), Score: candidateScore})
			if candidateScore.Correct > result.Final.Correct {
				result.Choices, result.Final = CloneChoices(trial), candidateScore
			}
			if result.Final.Correct == result.Final.Total {
				result.StopReason = "all_training_cases_passed"
				return result, nil
			}
		}
		exhausted = true
		for i := len(indices) - 1; i >= 0; i-- {
			indices[i]++
			if indices[i] < len(holes[i].Allowed) {
				exhausted = false
				break
			}
			indices[i] = 0
		}
	}
	if exhausted {
		result.StopReason = "candidate_space_exhausted"
	} else {
		result.StopReason = "attempt_budget_reached"
	}
	return result, nil
}

func sameChoices(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for id, operation := range left {
		if right[id] != operation {
			return false
		}
	}
	return true
}
