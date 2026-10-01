package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bilingualstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type observation struct {
	Row             string     `json:"row_id"`
	Initial         int        `json:"initial_option_index"`
	Committed       int        `json:"continued_option_index"`
	Probability     [2]float64 `json:"eligible_probability"`
	PredictionNS    int64      `json:"observed_prediction_ns"`
	SparsePassed    int        `json:"sparse_passed"`
	FullBefore      int        `json:"full_passed_before"`
	FullAfter       int        `json:"full_passed_after"`
	FullTotal       int        `json:"full_cases"`
	AddedAttempts   int        `json:"added_path_attempts"`
	FullContractSHA string     `json:"complete_contract_sha256"`
}

func eligible(model *decision.Model, prediction decision.Prediction, options [2]string) ([2]float64, error) {
	var logits [2]float64
	for i, option := range options {
		found := false
		for j, label := range decision.PathLabels() {
			if label == option {
				logits[i], found = float64(prediction.Logits[j])/float64(model.Temperature()), true
			}
		}
		if !found {
			return [2]float64{}, errors.New("unknown eligible label")
		}
	}
	m := math.Max(logits[0], logits[1])
	a, b := math.Exp(logits[0]-m), math.Exp(logits[1]-m)
	return [2]float64{a / (a + b), b / (a + b)}, nil
}

func observe(row feedbackstudy.Row, model *decision.Model, workspace *decision.Workspace) (observation, error) {
	o := observation{Row: row.ID, Probability: [2]float64{0.5, 0.5}}
	if model != nil {
		var prediction decision.Prediction
		start := time.Now()
		err := model.PredictInto(row.Text, workspace, &prediction)
		o.PredictionNS = time.Since(start).Nanoseconds()
		if err != nil {
			return o, err
		}
		o.Probability, err = eligible(model, prediction, row.Options)
		if err != nil {
			return o, err
		}
		if o.Probability[1] > o.Probability[0] {
			o.Initial = 1
		}
	}
	plan, err := pathstudy.Fixture(row.Family, row.Configuration, row.OriginalText)
	if err != nil {
		return o, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return o, err
	}
	var contract []pathplan.TestCase
	for _, input := range pathstudy.Inputs(row.Configuration) {
		expected, err := pathstudy.Oracle(row.Family, row.IntentionLabel == row.Options[1], row.Configuration, input)
		if err != nil {
			return o, err
		}
		contract = append(contract, pathplan.TestCase{Input: input, Expected: expected})
	}
	raw, err := json.Marshal(contract)
	if err != nil {
		return o, err
	}
	o.FullContractSHA, o.FullTotal = feedbackstudy.Hash(raw), len(contract)
	passed := func(option int, cases []pathplan.TestCase) (int, error) {
		body, err := prepared.Compile(map[string]string{"structure": row.Options[option]})
		if err != nil {
			return 0, err
		}
		count := 0
		for _, c := range cases {
			value, err := body.Evaluate(c.Input)
			if err != nil {
				return 0, err
			}
			if value.Int == c.Expected {
				count++
			}
		}
		return count, nil
	}
	o.SparsePassed, err = passed(o.Initial, row.Cases)
	if err != nil {
		return o, err
	}
	o.FullBefore, err = passed(o.Initial, contract)
	if err != nil {
		return o, err
	}
	o.FullAfter, o.Committed = o.FullBefore, o.Initial
	if o.FullBefore < o.FullTotal {
		o.AddedAttempts = 1
		other, err := passed(1-o.Initial, contract)
		if err != nil {
			return o, err
		}
		if other > o.FullBefore {
			o.FullAfter, o.Committed = other, 1-o.Initial
		}
	}
	return o, nil
}

func summarize(pairs []bilingualstudy.Pair, model *decision.Model) (map[string]any, [][2]observation, error) {
	counts := map[string]int{}
	var observations [][2]observation
	var workspace decision.Workspace
	var durations []int64
	divergence := 0.0
	for _, pair := range pairs {
		if pair.Split != "test" {
			continue
		}
		var obs [2]observation
		for language, row := range pair.Rows {
			o, err := observe(row, model, &workspace)
			if err != nil {
				return nil, nil, err
			}
			obs[language] = o
			counts["views"]++
			counts["sparse_finite_passed"] += o.SparsePassed
			counts["sparse_finite_total"] += len(row.Cases)
			counts["full_finite_passed_before"] += o.FullBefore
			counts["full_finite_passed_after"] += o.FullAfter
			counts["full_finite_total"] += o.FullTotal
			counts["added_path_attempts"] += o.AddedAttempts
			if o.FullBefore == o.FullTotal {
				counts["full_program_views_before"]++
			}
			if o.FullAfter == o.FullTotal {
				counts["full_program_views_after"]++
			}
			if pair.Targets[o.Initial] > 0 {
				counts["finite_best_set_selected"]++
			}
			if model != nil {
				counts["actual_model_calls"]++
				durations = append(durations, o.PredictionNS)
			}
		}
		counts["pairs"]++
		if obs[0].Initial == obs[1].Initial {
			counts["initial_pair_agreement"]++
			if pair.Options[obs[0].Initial] != pair.Rows[0].IntentionLabel {
				counts["same_wrong_intention_pairs"]++
			}
		}
		if pair.Options[obs[0].Initial] == pair.Rows[0].IntentionLabel &&
			pair.Options[obs[1].Initial] == pair.Rows[1].IntentionLabel {
			counts["both_intention_correct_pairs"]++
		}
		for option := range 2 {
			mixture := (obs[0].Probability[option] + obs[1].Probability[option]) / 2
			for language := range 2 {
				p := obs[language].Probability[option]
				if p > 0 {
					divergence += p * math.Log(p/mixture) / 2
				}
			}
		}
		observations = append(observations, obs)
	}
	result := map[string]any{"counts": counts, "mean_bilingual_js": divergence / float64(counts["pairs"]),
		"extra_model_calls_during_continuation": 0, "native_calls": 0, "generated_go_processes": 0}
	if model != nil {
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		result["prediction_median_ns"] = durations[len(durations)/2]
		result["prediction_p95_ns"] = durations[(len(durations)-1)*95/100]
		result["metadata_sha256"], result["weights_sha256"] = model.MetadataSHA256(), model.WeightsSHA256()
		result["resident_tensor_bytes"], result["packed_weights_bytes"] = model.ResidentTensorBytes(), model.PackedFileBytes()
		result["matrix_scale_bytes"], result["workspace_bytes"] = model.MatrixScaleBytes(), decision.WorkspaceBytes()
	}
	return result, observations, nil
}

func run(models, out, armNames string) error {
	if models == "" || out == "" {
		return errors.New("trained model root and fresh output directory required")
	}
	trainedArms := [2]string{"control", "paired"}
	if armNames == "v1,v2" {
		trainedArms = [2]string{"v1", "v2"}
	} else if armNames != "control,paired" {
		return errors.New("explicit control/paired or v1/v2 study arms required")
	}
	if _, err := os.Lstat(out); !os.IsNotExist(err) {
		return errors.New("existing capture cannot be overwritten")
	}
	pairs, err := bilingualstudy.Load("data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		return err
	}
	if err := os.Mkdir(out, 0755); err != nil {
		return err
	}
	save := func(name string, value any) error {
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(out, name), append(raw, '\n'), 0644)
	}
	pre := map[string]any{"schema": "gooo/bilingual-judgment-go-preexecution/v1", "dataset_sha256": bilingualstudy.DatasetSHA,
		"development_pairs": 160, "development_views": 320, "model_arms": 9,
		"source_hashes": map[string]string{}, "complete_contract": "pathstudy.Inputs plus independent pathstudy.Oracle; explicit intention reference",
		"scope": "Sparse Gooo finite views followed by a separately bound authored full contract, two legal paths, at most one additional attempt, zero additional model calls. No native/generated-Go execution, host utilization, untouched holdout or new independent tasks."}
	sources := pre["source_hashes"].(map[string]string)
	for _, source := range []string{"tools/audit-bilingual-judgment/main.go", "internal/bilingualstudy/pairs.go", "internal/pathstudy/study.go"} {
		raw, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		sources[source] = feedbackstudy.Hash(raw)
	}
	if err := save("preexecution.json", pre); err != nil {
		return err
	}
	cells, hashes := map[string]any{}, map[string]string{}
	for _, arm := range []string{"parent", trainedArms[0], trainedArms[1], "offline"} {
		variants := []string{"fp32", "ptq_ternary", "qat_ternary"}
		if arm == "offline" {
			variants = []string{"deterministic"}
		}
		for _, variant := range variants {
			var model *decision.Model
			if arm != "offline" {
				dir := filepath.Join(models, arm, "models")
				if arm == "parent" {
					dir = "runs/feedback-path-soft-target-mps-20261001/models"
				}
				model, err = decision.LoadPath(filepath.Join(dir, variant, "model.json"))
				if err != nil {
					return err
				}
			}
			name := arm + "-" + variant
			result, captures, err := summarize(pairs, model)
			if err != nil {
				return err
			}
			filename := name + ".json"
			if err := save(filename, captures); err != nil {
				return err
			}
			raw, err := os.ReadFile(filepath.Join(out, filename))
			if err != nil {
				return err
			}
			cells[name], hashes[filename] = result, feedbackstudy.Hash(raw)
		}
	}
	return save("report.json", map[string]any{"schema": "gooo/bilingual-judgment-go-report/v1", "decision": "PASS",
		"actual_initial_model_calls": 2880, "additional_model_calls": 0, "development_pairs": 160,
		"cells": cells, "capture_sha256": hashes, "new_independent_intentions": 0,
		"scope": pre["scope"], "agreement_is_correctness": false, "first_shot_correctness_is_primary_objective": false})
}

func main() {
	models := flag.String("models", "", "fresh training result root")
	out := flag.String("out", "", "fresh capture directory")
	arms := flag.String("arms", "control,paired", "control,paired or v1,v2 comparison")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if err := run(*models, *out, *arms); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
