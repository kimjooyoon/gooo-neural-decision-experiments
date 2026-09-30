package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type row struct {
	ID                 string `json:"id"`
	InstructionID      string `json:"instruction_id"`
	ProgramID          string `json:"program_id"`
	TemplateID         string `json:"template_id"`
	ConfigurationID    string `json:"configuration_id"`
	ConfigurationIndex int    `json:"configuration_index"`
	Family             string `json:"family"`
	Language           string `json:"language"`
	Split              string `json:"split"`
	View               string `json:"view"`
	Text               string `json:"text"`
	Label              string `json:"label"`
}

func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

func generate() ([]byte, map[string]int, error) {
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetEscapeHTML(false)
	counts := map[string]int{"train": 0, "calibration": 0, "test": 0}
	seenText := make(map[string]bool)
	for _, family := range pathstudy.Families {
		for config := 0; config < 64; config++ {
			split, begin, end := "train", 0, 4
			if config >= 48 {
				split, begin, end = "test", 6, 8
			} else if config >= 40 {
				split, begin, end = "calibration", 4, 6
			}
			for _, reverse := range []bool{false, true} {
				for template := begin; template < end; template++ {
					plain, language, err := pathstudy.Instruction(family, reverse, config, template)
					if err != nil {
						return nil, nil, err
					}
					instructionID := fmt.Sprintf("%s-c%02d-r%t-t%d", family, config, reverse, template)
					for _, view := range []string{"plain", "gooo", "prov"} {
						text, err := pathstudy.View(plain, family, view, config)
						if err != nil || len(text) == 0 || len(text) > decision.InputMaxBytes || seenText[text] {
							return nil, nil, errors.New("invalid, repeated or oversized path instruction")
						}
						seenText[text] = true
						record := row{ID: instructionID + "-" + view, InstructionID: instructionID, ProgramID: fmt.Sprintf("%s-c%02d-r%t", family, config, reverse), TemplateID: fmt.Sprintf("%s-t%d", family, template), ConfigurationID: fmt.Sprintf("%s-c%02d", family, config), ConfigurationIndex: config, Family: family, Language: language, Split: split, View: view, Text: text, Label: pathstudy.GoldLabel(family, reverse)}
						if err := encoder.Encode(record); err != nil {
							return nil, nil, err
						}
						counts[split]++
					}
				}
			}
		}
	}
	return data.Bytes(), counts, nil
}

func run(output string) error {
	if output == "" {
		return errors.New("output directory required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("output directory must be fresh")
	}
	data, counts, err := generate()
	if err != nil {
		return err
	}
	sources := make(map[string]string)
	for _, path := range []string{"cmd/path-curriculum/main.go", "internal/pathstudy/study.go", "internal/pathplan/pathplan.go", "internal/decision/model.go"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sources[path] = hash(raw)
	}
	labels := decision.PathLabels()
	manifest := map[string]any{
		"schema": "gooo/typed-path-curriculum/v1", "dataset_sha256": hash(data), "total_rows": 6240, "rows": counts,
		"model_schema": decision.PathMetadataSchema, "labels": labels, "generator_source_sha256": sources,
		"configuration_ranges":          map[string][2]int{"train": {0, 39}, "calibration": {40, 47}, "test": {48, 63}},
		"template_ranges":               map[string][2]int{"train": {0, 3}, "calibration": {4, 5}, "test": {6, 7}},
		"unique_instructions":           map[string]int{"train": 1600, "calibration": 160, "test": 320},
		"unique_program_configurations": map[string]int{"train": 400, "calibration": 80, "test": 160},
		"views_per_instruction":         3, "families": pathstudy.Families,
		"scope": "Five closed structural choice kinds; disjoint templates and numeric configurations. PROV-O vocabulary conditioning is not OWL reasoning. Generated Gooo view shows the fixed fallback body, never a gold-filled body.",
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(output, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "dataset.jsonl"), data, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "manifest.json"), append(raw, '\n'), 0644)
}

func main() {
	output := flag.String("out", "", "fresh path curriculum directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "path-curriculum: unexpected positional arguments")
		os.Exit(2)
	}
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, "path-curriculum:", err)
		os.Exit(1)
	}
	fmt.Println(`{"status":"GENERATED","rows":6240,"heldout_original_instructions":320,"heldout_program_configurations":160}`)
}
