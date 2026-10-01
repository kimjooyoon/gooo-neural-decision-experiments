package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
)

const parentSHA = "942b76c3eee49c83fc0abccc3b33f3e92835c9ad7743918fa9c6a6b976d99f7b"

type original struct {
	ID              string `json:"id"`
	Instruction     string `json:"instruction_id"`
	Program         string `json:"program_id"`
	Template        string `json:"template_id"`
	ConfigurationID string `json:"configuration_id"`
	Configuration   int    `json:"configuration_index"`
	Family          string `json:"family"`
	Language        string `json:"language"`
	Split           string `json:"split"`
	View            string `json:"view"`
	Text            string `json:"text"`
	Label           string `json:"label"`
}

func generate() ([]byte, map[string]map[string]int, error) {
	raw, err := os.ReadFile("data/typed-path-positioned-v2/dataset.jsonl")
	if err != nil || feedbackstudy.Hash(raw) != parentSHA {
		return nil, nil, errors.New("fixed parent structural curriculum required")
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	counts := map[string]map[string]int{}
	seen := map[string]bool{}
	groups := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		var o original
		if err := json.Unmarshal(scanner.Bytes(), &o); err != nil {
			return nil, nil, err
		}
		if seen[o.ID] || (groups[o.Instruction] != "" && groups[o.Instruction] != o.Split) {
			return nil, nil, errors.New("duplicate row or instruction split leakage")
		}
		seen[o.ID], groups[o.Instruction] = true, o.Split
		row, err := feedbackstudy.Derive(feedbackstudy.Original{ID: o.ID, InstructionID: o.Instruction,
			ProgramID: o.Program, TemplateID: o.Template, ConfigurationID: o.ConfigurationID,
			ConfigurationIndex: o.Configuration, Family: o.Family, Language: o.Language,
			Split: o.Split, View: o.View, Text: o.Text, Label: o.Label})
		if err != nil {
			return nil, nil, err
		}
		if counts[row.Split] == nil {
			counts[row.Split] = map[string]int{}
		}
		count := counts[row.Split]
		count["rows"]++
		count[row.Representation]++
		count[row.Contract]++
		if len(row.Accepted) > 1 {
			count["multiple_best_finite_labels"]++
		}
		if err := encoder.Encode(row); err != nil {
			return nil, nil, err
		}
	}
	if err := scanner.Err(); err != nil || len(seen) != 6240 {
		return nil, nil, errors.New("complete parent dataset required")
	}
	return output.Bytes(), counts, nil
}

func run(output string) error {
	if output == "" {
		return errors.New("fresh output directory required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("existing output cannot be overwritten")
	}
	raw, counts, err := generate()
	if err != nil {
		return err
	}
	sources := map[string]string{}
	for _, name := range []string{"cmd/feedback-curriculum/main.go", "internal/feedbackstudy/curriculum.go", "internal/pathstudy/study.go", "internal/pathplan/prepared.go", "internal/bodyplan/bodyplan.go"} {
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		sources[name] = feedbackstudy.Hash(data)
	}
	manifest := map[string]any{"schema": "gooo/feedback-path-curriculum/v1", "dataset_sha256": feedbackstudy.Hash(raw),
		"parent_dataset_sha256": parentSHA, "source_sha256": sources, "total_rows": 6240, "rows": counts,
		"model_schema": decision.PathMetadataSchema, "labels": decision.PathLabels(), "input_max_bytes": decision.InputMaxBytes,
		"preserved_original_instruction_groups": 2080, "independent_new_intention_families": 0,
		"oracle": "independent arithmetic pathstudy.Oracle; two typed candidates evaluated per row",
		"target": "all labels tied for maximum declared finite pass count; separate original intention label retained",
		"scope":  "Existing closed five-family bilingual synthetic curriculum with new finite-feedback views. Sparse and contradictory cases retain ambiguity and partial denominators. Over-bound contexts remain recorded without truncation and are ineligible for model training. Caller CI status is unverified context. Reused source/test groups are development evaluation, not an untouched holdout or 6240 new independent tasks."}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(output, "dataset.jsonl"), raw, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "manifest.json"), append(data, '\n'), 0644)
}

func main() {
	output := flag.String("out", "", "fresh feedback curriculum directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional argument")
		os.Exit(2)
	}
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
