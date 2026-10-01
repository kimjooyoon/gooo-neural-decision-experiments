// Read-only source-bound breakdown of the frozen own-model development audit.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bilingualstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type observation struct {
	ID     string                `json:"id"`
	Input  string                `json:"input_sha256"`
	Search pathplan.SearchResult `json:"search"`
}
type breakdown struct {
	Family   string `json:"family"`
	Language string `json:"language"`
	Views    int    `json:"views"`
	Correct  int    `json:"initial_intention_agreement"`
	Extra    int    `json:"extra_candidates"`
	Complete int    `json:"finite_complete_after_tdd"`
}

func hash(raw []byte) string { v := sha256.Sum256(raw); return hex.EncodeToString(v[:]) }
func diagnose(row feedbackstudy.Row, observed observation) error {
	search := observed.Search
	if observed.ID != row.ID || search.Evaluated != len(search.Attempts) || search.Evaluated < 1 || search.Evaluated > 2 ||
		search.Selection.ModelCalls != 1 || len(search.Selection.Receipts) != 1 || search.Selection.Receipts[0].IntentSHA256 != observed.Input {
		return errors.New("observation accounting differs")
	}
	plan, err := pathstudy.Fixture(row.Family, row.Configuration, row.OriginalText)
	if err != nil {
		return err
	}
	inputs := pathstudy.Inputs(row.Configuration)
	for _, attempt := range search.Attempts {
		program, err := pathplan.Compile(plan, attempt.Choices)
		if err != nil {
			return err
		}
		if len(attempt.Results) != len(inputs) || attempt.Total != len(inputs) {
			return errors.New("attempt finite denominator differs")
		}
		passed := 0
		for i, input := range inputs {
			actual, err := program.Evaluate(input)
			if err != nil {
				return err
			}
			expected, err := pathstudy.Oracle(row.Family, row.IntentionLabel == row.Options[1], row.Configuration, input)
			if err != nil {
				return err
			}
			r := attempt.Results[i]
			if r.Input != input || r.Expected != expected || r.Actual != actual.Int || r.Passed != (actual.Int == expected) {
				return errors.New("finite observation forged")
			}
			if r.Passed {
				passed++
			}
		}
		if passed != attempt.Passed {
			return errors.New("attempt pass count differs")
		}
	}
	if search.SelectedTrainingPassed != len(inputs) || search.TrainingTotal != len(inputs) {
		return errors.New("final finite denominator differs")
	}
	return nil
}
func run(audit, output string) error {
	pairs, err := bilingualstudy.Load("data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		return err
	}
	rows := map[string]feedbackstudy.Row{}
	for _, pair := range pairs {
		for _, row := range pair.Rows {
			if row.Split == "test" {
				rows[row.ID] = row
			}
		}
	}
	raw, err := os.ReadFile(filepath.Join(audit, "compiler_context-fp32.json"))
	if err != nil {
		return err
	}
	var observations []observation
	if err = json.Unmarshal(raw, &observations); err != nil || len(observations) != 320 {
		return errors.New("320 frozen FP32 observations required")
	}
	byCell := map[string]breakdown{}
	seen := map[string]bool{}
	for _, o := range observations {
		row, found := rows[o.ID]
		if !found || seen[o.ID] {
			return errors.New("unexpected/duplicate development view")
		}
		seen[o.ID] = true
		if err = diagnose(row, o); err != nil {
			return err
		}
		key := row.Family + "/" + row.Language
		b := byCell[key]
		b.Family, b.Language = row.Family, row.Language
		b.Views++
		b.Extra += o.Search.Evaluated - 1
		b.Complete++
		if o.Search.InitialProposals["structure"] == row.IntentionLabel {
			b.Correct++
		}
		byCell[key] = b
	}
	var keys []string
	for key := range byCell {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var cells []breakdown
	for _, key := range keys {
		cells = append(cells, byCell[key])
	}
	value := map[string]any{"schema": "gooo/compiler-context-bottleneck-diagnostics/v1", "status": "PASS", "audit_observations_sha256": hash(raw),
		"source_dataset_sha256": bilingualstudy.DatasetSHA, "model_predictions": 0, "native_calls": 0, "cells": cells,
		"scope": "source-bound reanalysis of frozen development observations, not another experiment; finite actuals independently rechecked; family/language differences do not establish a causal mechanism"}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(encoded, '\n'), 0644)
}
func main() {
	audit := flag.String("audit", "runs/compiler-context-model-audit-20261001", "frozen own-model audit")
	output := flag.String("output", "", "new diagnostic receipt")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "output required")
		os.Exit(2)
	}
	if err := run(*audit, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
