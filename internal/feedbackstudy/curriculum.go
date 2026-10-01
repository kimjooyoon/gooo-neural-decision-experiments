// Package feedbackstudy derives finite behavioral targets from typed Gooo paths.
package feedbackstudy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type Original struct {
	ID, InstructionID, ProgramID, TemplateID, ConfigurationID string
	ConfigurationIndex                                        int
	Family, Language, Split, View, Text, Label                string
}

type Row struct {
	ID                 string              `json:"id"`
	InstructionID      string              `json:"instruction_id"`
	ProgramID          string              `json:"program_id"`
	TemplateID         string              `json:"template_id"`
	ConfigurationID    string              `json:"configuration_id"`
	Configuration      int                 `json:"configuration_index"`
	Family             string              `json:"family"`
	Language           string              `json:"language"`
	Split              string              `json:"split"`
	View               string              `json:"original_view"`
	OriginalText       string              `json:"original_text"`
	OriginalSHA        string              `json:"original_text_sha256"`
	IntentionLabel     string              `json:"intention_label"`
	Text               string              `json:"text"`
	InputSHA           string              `json:"input_sha256"`
	InputBytes         int                 `json:"input_bytes"`
	Eligible           bool                `json:"training_eligible"`
	Representation     string              `json:"representation"`
	Contract           string              `json:"finite_contract"`
	PlanSHA            string              `json:"plan_sha256"`
	Cases              []pathplan.TestCase `json:"finite_cases"`
	Options            [2]string           `json:"eligible_labels"`
	OptionPassed       [2]int              `json:"option_finite_passed"`
	BestPassed         int                 `json:"best_finite_passed"`
	Accepted           []string            `json:"best_finite_labels"`
	Initial            string              `json:"initial_label"`
	InitialPassed      int                 `json:"initial_finite_passed"`
	CIStatus           string              `json:"caller_ci_status"`
	CIIsAuthority      bool                `json:"ci_hint_is_authority"`
	FunctionalIsIntent bool                `json:"finite_targets_prove_intent"`
}

func Hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }

func contract(source Original, reverse bool) ([]pathplan.TestCase, string, error) {
	a, _ := pathstudy.Parameters(source.ConfigurationIndex)
	inputs := []int64{-7, -1, 0, 1, a - 1, a, a + 1}
	kind := "complete_finite"
	if source.View == "gooo" {
		inputs, kind = []int64{a}, "sparse_finite"
	} else if source.View == "prov" {
		kind = "inconsistent_finite"
	}
	seen := map[int64]bool{}
	var cases []pathplan.TestCase
	for _, input := range inputs {
		if seen[input] {
			continue
		}
		seen[input] = true
		expected, err := pathstudy.Oracle(source.Family, reverse, source.ConfigurationIndex, input)
		if err != nil {
			return nil, "", err
		}
		cases = append(cases, pathplan.TestCase{Input: input, Expected: expected})
	}
	if kind == "inconsistent_finite" {
		// One input has contradictory requirements; no program can satisfy both.
		cases = append(cases, pathplan.TestCase{Input: cases[0].Input, Expected: cases[0].Expected + 1})
	}
	return cases, kind, nil
}

func Derive(source Original) (Row, error) {
	if len(source.Text) < 1 || len(source.Text) > decision.InputMaxBytes || !utf8.ValidString(source.Text) ||
		(source.Language != "en" && source.Language != "ko") ||
		(source.View != "plain" && source.View != "gooo" && source.View != "prov") ||
		(source.Split != "train" && source.Split != "calibration" && source.Split != "test") {
		return Row{}, errors.New("bounded original bilingual structural row required")
	}
	forward, reverse := pathstudy.GoldLabel(source.Family, false), pathstudy.GoldLabel(source.Family, true)
	if source.Label != forward && source.Label != reverse || forward == "" {
		return Row{}, errors.New("original intention label does not belong to the typed pair")
	}
	cases, kind, err := contract(source, source.Label == reverse)
	if err != nil {
		return Row{}, err
	}
	plan, err := pathstudy.Fixture(source.Family, source.ConfigurationIndex, source.Text)
	if err != nil {
		return Row{}, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return Row{}, err
	}
	row := Row{ID: source.ID + "-feedback", InstructionID: source.InstructionID, ProgramID: source.ProgramID,
		TemplateID: source.TemplateID, ConfigurationID: source.ConfigurationID, Configuration: source.ConfigurationIndex,
		Family: source.Family, Language: source.Language, Split: source.Split, View: source.View,
		OriginalText: source.Text, OriginalSHA: Hash([]byte(source.Text)), IntentionLabel: source.Label,
		Contract: kind, PlanSHA: prepared.PlanSHA256(), Cases: cases, Options: [2]string{forward, reverse},
		Representation: "bounded", Eligible: true}
	var firstMismatch [2]string
	for option, label := range row.Options {
		body, err := prepared.Compile(map[string]string{"structure": label})
		if err != nil {
			return Row{}, err
		}
		for _, c := range cases {
			value, err := body.Evaluate(c.Input)
			if err != nil {
				return Row{}, err
			}
			if value.Int == c.Expected {
				row.OptionPassed[option]++
			} else if firstMismatch[option] == "" {
				firstMismatch[option] = fmt.Sprintf(" mismatch=%d:%d:%d", c.Input, value.Int, c.Expected)
			}
		}
		if row.OptionPassed[option] > row.BestPassed {
			row.BestPassed = row.OptionPassed[option]
		}
	}
	for i, label := range row.Options {
		if row.OptionPassed[i] == row.BestPassed {
			row.Accepted = append(row.Accepted, label)
		}
	}
	semantic := 0
	if source.Label == reverse {
		semantic = 1
	}
	semanticExpected := len(cases)
	if kind == "inconsistent_finite" {
		semanticExpected--
	}
	if row.OptionPassed[semantic] != semanticExpected {
		return Row{}, errors.New("typed source behavior disagrees with independent arithmetic oracle")
	}
	initial := source.ConfigurationIndex % 2
	row.Initial, row.InitialPassed = row.Options[initial], row.OptionPassed[initial]
	row.CIStatus = [3]string{"PASS", "FAIL", "UNKNOWN"}[source.ConfigurationIndex%3]
	prefix := fmt.Sprintf("feedback: tried=1 passed=%d/%d rejected=0 remaining=1", row.InitialPassed, len(cases))
	prefix += firstMismatch[initial] + " ci=" + row.CIStatus + " selected=" + row.Initial
	row.Text = prefix + "\nintent: " + source.Text
	row.InputBytes, row.InputSHA = len(row.Text), Hash([]byte(row.Text))
	if row.InputBytes > decision.InputMaxBytes {
		row.Eligible, row.Representation = false, "context_declined"
	}
	if !strings.HasSuffix(row.Text, "\nintent: "+row.OriginalText) {
		return Row{}, errors.New("original intention was changed")
	}
	return row, nil
}
