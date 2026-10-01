package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

const datasetSHA = "570d1d73bfea32662b869e5cfbf77e6d04df838b703191b076be58c1d2180dbf"

func read(path string, limit int64) ([]byte, error) {
	stat, err := os.Lstat(path)
	if err != nil || !stat.Mode().IsRegular() || stat.Size() <= 0 || stat.Size() > limit {
		return nil, errors.New("bounded regular evidence file required")
	}
	return os.ReadFile(path)
}

func loadRows() ([]feedbackstudy.Row, error) {
	raw, err := read("data/feedback-path-v1/dataset.jsonl", 16<<20)
	if err != nil || feedbackstudy.Hash(raw) != datasetSHA {
		return nil, errors.New("fixed feedback dataset required")
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	var test []feedbackstudy.Row
	counts := map[string]int{}
	for scanner.Scan() {
		var r feedbackstudy.Row
		if err = strictjson.Decode(scanner.Bytes(), &r); err != nil {
			return nil, err
		}
		original := feedbackstudy.Original{ID: strings.TrimSuffix(r.ID, "-feedback"), InstructionID: r.InstructionID,
			ProgramID: r.ProgramID, TemplateID: r.TemplateID, ConfigurationID: r.ConfigurationID,
			ConfigurationIndex: r.Configuration, Family: r.Family, Language: r.Language, Split: r.Split,
			View: r.View, Text: r.OriginalText, Label: r.IntentionLabel}
		derived, err := feedbackstudy.Derive(original)
		if err != nil || !reflect.DeepEqual(derived, r) {
			return nil, errors.New("finite target differs from typed/oracle reconstruction")
		}
		counts[r.Split]++
		if r.Split == "test" && r.Eligible {
			test = append(test, r)
		}
	}
	if scanner.Err() != nil || counts["train"] != 4800 || counts["calibration"] != 480 || counts["test"] != 960 || len(test) != 960 {
		return nil, errors.New("fixed split counts differ")
	}
	return test, nil
}

type counters struct {
	Views        int     `json:"observed_views"`
	Best         int     `json:"finite_best_set_selected"`
	Intent       int     `json:"original_intention_label_agreement"`
	Passed       int     `json:"finite_cases_passed"`
	Total        int     `json:"finite_cases_total"`
	Ambiguous    int     `json:"multiple_best_finite_labels"`
	Abstentions  int     `json:"global_abstentions"`
	Calls        int     `json:"actual_local_predictions"`
	PredictionNS int64   `json:"summed_observed_prediction_ns"`
	NLL          float64 `json:"paired_soft_target_nll"`
	Mass         float64 `json:"mean_probability_on_best_set"`
}

func probability(model *decision.Model, prediction decision.Prediction, options [2]string) [2]float64 {
	var logits [2]float64
	for i, option := range options {
		for j, label := range decision.PathLabels() {
			if label == option {
				logits[i] = float64(prediction.Logits[j]) / float64(model.Temperature())
			}
		}
	}
	m := math.Max(logits[0], logits[1])
	a, b := math.Exp(logits[0]-m), math.Exp(logits[1]-m)
	return [2]float64{a / (a + b), b / (a + b)}
}

func observe(rows []feedbackstudy.Row, model *decision.Model) (map[string]any, error) {
	var workspace decision.Workspace
	c := counters{}
	for _, r := range rows {
		selected := 0
		probs := [2]float64{0.5, 0.5}
		if model != nil {
			var prediction decision.Prediction
			started := time.Now()
			err := model.PredictInto(r.Text, &workspace, &prediction)
			c.PredictionNS += time.Since(started).Nanoseconds()
			c.Calls++
			if err != nil {
				return nil, err
			}
			probs = probability(model, prediction, r.Options)
			if probs[1] > probs[0] {
				selected = 1
			}
			if prediction.Abstained {
				c.Abstentions++
			}
		}
		label := r.Options[selected]
		c.Views++
		if label == r.IntentionLabel {
			c.Intent++
		}
		for _, allowed := range r.Accepted {
			if allowed == label {
				c.Best++
			}
			for i, option := range r.Options {
				if allowed == option {
					c.NLL -= math.Log(math.Max(probs[i], 1e-12)) / float64(len(r.Accepted))
					c.Mass += probs[i]
				}
			}
		}
		if len(r.Accepted) > 1 {
			c.Ambiguous++
		}
		plan, err := pathstudy.Fixture(r.Family, r.Configuration, r.OriginalText)
		if err != nil {
			return nil, err
		}
		prepared, err := pathplan.Prepare(plan)
		if err != nil {
			return nil, err
		}
		body, err := prepared.Compile(map[string]string{"structure": label})
		if err != nil {
			return nil, err
		}
		passed := 0
		for _, test := range r.Cases {
			value, err := body.Evaluate(test.Input)
			if err != nil {
				return nil, err
			}
			c.Total++
			if value.Int == test.Expected {
				c.Passed++
				passed++
			}
		}
		if passed != r.OptionPassed[selected] {
			return nil, errors.New("selected body differs from finite curriculum observation")
		}
	}
	c.NLL /= float64(c.Views)
	c.Mass /= float64(c.Views)
	result := map[string]any{"scores": c, "native_calls": 0, "generated_go_processes": 0}
	if model != nil {
		result["metadata_sha256"], result["weights_sha256"] = model.MetadataSHA256(), model.WeightsSHA256()
		result["resident_tensor_bytes"], result["packed_weights_file_bytes"] = model.ResidentTensorBytes(), model.PackedFileBytes()
		result["matrix_scale_bytes"], result["workspace_bytes"] = model.MatrixScaleBytes(), decision.WorkspaceBytes()
		result["confidence_threshold"], result["temperature"] = model.ConfidenceThreshold(), model.Temperature()
	}
	return result, nil
}

type parityRow struct {
	Variant       string       `json:"variant"`
	Text          string       `json:"text"`
	Features      [256]float32 `json:"features"`
	Logits        [8]float64   `json:"logits"`
	Probabilities [8]float64   `json:"probabilities"`
	Label         string       `json:"selected_label"`
}

func parity(models map[string]*decision.Model, path string) (map[string]any, error) {
	raw, err := read(path, 2<<20)
	if err != nil {
		return nil, err
	}
	var fixture struct {
		Schema string      `json:"schema"`
		Rows   []parityRow `json:"rows"`
	}
	if err = strictjson.Decode(raw, &fixture); err != nil || fixture.Schema != "gooo/typed-path-parity/v1" || len(fixture.Rows) != 96 {
		return nil, errors.New("fixed parity fixture required")
	}
	maximum := 0.0
	for _, row := range fixture.Rows {
		model := models[row.Variant]
		if model == nil {
			return nil, errors.New("unknown parity variant")
		}
		var features [256]float32
		var workspace decision.Workspace
		var predicted decision.Prediction
		if err = model.FeaturesInto(row.Text, &features); err != nil {
			return nil, err
		}
		if err = model.PredictInto(row.Text, &workspace, &predicted); err != nil {
			return nil, err
		}
		for i, value := range features {
			if math.Abs(float64(value-row.Features[i])) > 1e-6 {
				return nil, errors.New("feature parity differs")
			}
		}
		for i := range 8 {
			for _, values := range [][2]float64{{float64(predicted.Logits[i]), row.Logits[i]}, {float64(predicted.Probabilities[i]), row.Probabilities[i]}} {
				difference := math.Abs(values[0] - values[1])
				if math.IsNaN(difference) || math.IsInf(difference, 0) || difference > 1e-4 {
					return nil, errors.New("Go/Python numeric parity differs")
				}
				maximum = math.Max(maximum, difference)
			}
		}
		if model.PredictLabel(&predicted) != row.Label {
			return nil, errors.New("global parity label differs")
		}
	}
	return map[string]any{"decision": "PASS", "rows": 96, "actual_local_predictions": 96,
		"maximum_logit_or_probability_absolute_error": maximum, "tolerance": 1e-4,
		"fixture_sha256": feedbackstudy.Hash(raw)}, nil
}

func run(modelsDir, output string) error {
	if output == "" {
		return errors.New("fresh output file required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("existing audit cannot be overwritten")
	}
	rows, err := loadRows()
	if err != nil {
		return err
	}
	cells := map[string]any{}
	newModels := map[string]*decision.Model{}
	for _, edition := range []string{"parent", "feedback"} {
		root := modelsDir
		if edition == "parent" {
			root = "runs/typed-path-positioned-random-20261001/models"
		}
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			model, err := decision.LoadPath(filepath.Join(root, variant, "model.json"))
			if err != nil {
				return err
			}
			cells[edition+"-"+variant], err = observe(rows, model)
			if err != nil {
				return err
			}
			if edition == "feedback" {
				newModels[variant] = model
			}
		}
	}
	cells["offline"], err = observe(rows, nil)
	if err != nil {
		return err
	}
	parityResult, err := parity(newModels, filepath.Join(filepath.Dir(modelsDir), "go-parity.json"))
	if err != nil {
		return err
	}
	result := map[string]any{"schema": "gooo/feedback-path-model-audit/v1", "decision": "PASS", "dataset_sha256": datasetSHA,
		"development_test_views": 960, "existing_original_test_instructions": 320,
		"model_scoring_actual_predictions": 5760, "parity_actual_predictions": 96, "total_actual_predictions": 5856,
		"cells": cells, "parity": parityResult, "new_native_calls": 0, "new_generated_go_processes": 0,
		"scope": "Go predictions on six model arms and deterministic offline control, selected typed-body finite outcomes and numeric parity. Reused development split, closed eligible pairs and 76 finite ties are explicit. Pair scores are ranking hints, not general intention correctness, native codegen or process-wide memory/utilization measurements."}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(raw, '\n'), 0644)
}

func main() {
	models := flag.String("models", "", "new feedback model directories")
	output := flag.String("output", "", "fresh audit output file")
	flag.Parse()
	if err := run(*models, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
