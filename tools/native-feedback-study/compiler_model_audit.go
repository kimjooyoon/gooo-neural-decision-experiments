package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type compilerModelCounts struct {
	Views             int     `json:"development_views"`
	Intent            int     `json:"initial_intention_label_agreement"`
	Accepted          int     `json:"initial_sparse_accepted_set_coverage"`
	SparseTies        int     `json:"sparse_ambiguous_views"`
	Before            int     `json:"full_contract_cases_before_tdd"`
	After             int     `json:"full_contract_cases_after_tdd"`
	Cases             int     `json:"full_contract_cases"`
	BeforeComplete    int     `json:"whole_full_contract_before_tdd"`
	AfterComplete     int     `json:"whole_full_contract_after_tdd"`
	Extras            int     `json:"additional_candidate_attempts"`
	Calls             int     `json:"actual_model_predictions"`
	LanguageDifferent int     `json:"bilingual_initial_disagreements"`
	Metadata          string  `json:"metadata_sha256,omitempty"`
	Weights           string  `json:"weights_sha256,omitempty"`
	Resident          int     `json:"resident_tensor_bytes,omitempty"`
	Scales            int     `json:"matrix_scale_bytes,omitempty"`
	PredictionNS      []int64 `json:"prediction_ns"`
}

type compilerModelObservation struct {
	ID     string                `json:"id"`
	Input  string                `json:"input_sha256"`
	Search pathplan.SearchResult `json:"search"`
}

func compilerTrainingRows(curriculum string) ([]compilerTrainingRow, error) {
	if err := auditCompilerCurriculum(curriculum); err != nil {
		return nil, err
	}
	raw, err := readCompilerCurriculumFile(filepath.Join(curriculum, "dataset.jsonl"))
	if err != nil {
		return nil, err
	}
	var result []compilerTrainingRow
	for _, line := range bytesLines(raw) {
		var row compilerTrainingRow
		if err = json.Unmarshal(line, &row); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
}

func bytesLines(raw []byte) [][]byte {
	var result [][]byte
	start := 0
	for i, b := range raw {
		if b == '\n' {
			if i > start {
				result = append(result, raw[start:i])
			}
			start = i + 1
		}
	}
	if start < len(raw) {
		result = append(result, raw[start:])
	}
	return result
}

func compilerFullCases(row compilerTrainingRow) ([]pathplan.TestCase, error) {
	var cases []pathplan.TestCase
	for _, input := range pathstudy.Inputs(row.Configuration) {
		expected, err := pathstudy.Oracle(row.Family, row.Intention == row.Options[1], row.Configuration, input)
		if err != nil {
			return nil, err
		}
		cases = append(cases, pathplan.TestCase{Input: input, Expected: expected})
	}
	return cases, nil
}

func observeCompilerModel(row compilerTrainingRow, model *decision.Model) (pathplan.SearchResult, error) {
	plan, err := pathstudy.Fixture(row.Family, row.Configuration, row.Text)
	if err != nil {
		return pathplan.SearchResult{}, err
	}
	cases, err := compilerFullCases(row)
	if err != nil {
		return pathplan.SearchResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, selected, err := pathplan.Search(ctx, plan, model, cases, 2, "")
	if err != nil {
		return result, err
	}
	if result.SelectedTrainingPassed != len(cases) || result.Evaluated > 2 || result.Selection.ExternalCalls != 0 {
		return result, errors.New("bounded complete finite assembly failed")
	}
	for _, c := range cases {
		actual, e := selected.Evaluate(c.Input)
		if e != nil || actual.Int != c.Expected {
			return result, errors.New("oracle final body differs")
		}
	}
	return result, nil
}

func compilerParity(models, arm string) (float64, error) {
	raw, err := read(filepath.Join(models, arm, "go-parity.json"))
	if err != nil {
		return 0, err
	}
	var parity struct {
		Rows []struct {
			Variant     string       `json:"variant"`
			Text        string       `json:"text"`
			Features    [256]float32 `json:"features"`
			Logits      [8]float64   `json:"logits"`
			Probability [8]float64   `json:"probabilities"`
			Label       string       `json:"selected_label"`
		} `json:"rows"`
	}
	if err = json.Unmarshal(raw, &parity); err != nil || len(parity.Rows) != 96 {
		return 0, errors.New("96 pinned parity inputs required")
	}
	maxError := 0.0
	counts := map[string]int{}
	loaded := map[string]*decision.Model{}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		model, err := decision.LoadPath(filepath.Join(models, arm, "models", variant, "model.json"))
		if err != nil {
			return 0, err
		}
		loaded[variant] = model
	}
	for _, row := range parity.Rows {
		counts[row.Variant]++
		model := loaded[row.Variant]
		if model == nil {
			return 0, errors.New("invalid parity variant")
		}
		var workspace decision.Workspace
		var prediction decision.Prediction
		var features [256]float32
		if err = model.PredictInto(row.Text, &workspace, &prediction); err != nil {
			return 0, err
		}
		if err = model.FeaturesInto(row.Text, &features); err != nil {
			return 0, err
		}
		if model.PredictLabel(&prediction) != row.Label {
			return 0, errors.New("parity selected label differs")
		}
		for i, v := range features {
			maxError = math.Max(maxError, math.Abs(float64(v-row.Features[i])))
		}
		for i := range row.Logits {
			maxError = math.Max(maxError, math.Abs(float64(prediction.Logits[i])-row.Logits[i]))
			maxError = math.Max(maxError, math.Abs(float64(prediction.Probabilities[i])-row.Probability[i]))
		}
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		if counts[variant] != 32 {
			return 0, errors.New("parity variant count differs")
		}
	}
	if math.IsNaN(maxError) || math.IsInf(maxError, 0) || maxError > 1e-6 {
		return maxError, errors.New("Go parity tolerance exceeded")
	}
	return maxError, nil
}

func runCompilerModelAudit(curriculum, models, output, revision string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	rows, err := compilerTrainingRows(curriculum)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	data, _ := readCompilerCurriculumFile(filepath.Join(curriculum, "dataset.jsonl"))
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/compiler-model-audit-preexecution/v1",
		"source_revision": revision, "dataset_sha256": hash(data), "planned_development_views_per_arm": 320, "planned_model_predictions": 1920,
		"planned_parity_predictions": 192, "native_calls": 0, "new_independent_intentions": 0}); err != nil {
		return err
	}
	counts := map[string]compilerModelCounts{}
	parity := map[string]float64{}
	for _, arm := range []string{"caller_context", "compiler_context", "offline"} {
		if arm != "offline" {
			parity[arm], err = compilerParity(models, arm)
			if err != nil {
				return err
			}
		}
		variants := []string{"fp32", "ptq_ternary", "qat_ternary"}
		if arm == "offline" {
			variants = []string{"deterministic"}
		}
		for _, variant := range variants {
			id := arm + "-" + variant
			var model *decision.Model
			var count compilerModelCounts
			if arm != "offline" {
				model, err = decision.LoadPath(filepath.Join(models, arm, "models", variant, "model.json"))
				if err != nil {
					return err
				}
				count.Metadata, count.Weights, count.Resident, count.Scales = model.MetadataSHA256(), model.WeightsSHA256(), model.ResidentTensorBytes(), model.MatrixScaleBytes()
			}
			var observations []compilerModelObservation
			priorPair, priorLabel := "", ""
			for _, row := range rows {
				if row.Split != "test" {
					continue
				}
				search, err := observeCompilerModel(row, model)
				if err != nil {
					return err
				}
				initial := search.InitialProposals["structure"]
				count.Views++
				if initial == row.Intention {
					count.Intent++
				}
				for _, label := range row.Accepted {
					if initial == label {
						count.Accepted++
					}
				}
				if len(row.Accepted) > 1 {
					count.SparseTies++
				}
				count.Cases += search.TrainingTotal
				count.Before += search.Attempts[0].Passed
				count.After += search.SelectedTrainingPassed
				if search.Attempts[0].Passed == search.TrainingTotal {
					count.BeforeComplete++
				}
				if search.SelectedTrainingPassed == search.TrainingTotal {
					count.AfterComplete++
				}
				count.Extras += search.Evaluated - 1
				count.Calls += search.Selection.ModelCalls
				for _, receipt := range search.Selection.Receipts {
					if receipt.IntentSHA256 != row.SHA {
						return errors.New("canonical prediction input hash differs")
					}
					count.PredictionNS = append(count.PredictionNS, receipt.PredictNS)
				}
				if priorPair == row.Pair && priorLabel != initial {
					count.LanguageDifferent++
				}
				priorPair, priorLabel = row.Pair, initial
				observations = append(observations, compilerModelObservation{row.ID, row.SHA, search})
			}
			sort.Slice(count.PredictionNS, func(i, j int) bool { return count.PredictionNS[i] < count.PredictionNS[j] })
			counts[id] = count
			if err = save(filepath.Join(output, id+".json"), observations); err != nil {
				return err
			}
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/compiler-model-audit/v1", "status": "PASS",
		"arms": counts, "parity_max_absolute_error": parity, "actual_model_predictions": 1920, "actual_parity_predictions": 192,
		"native_calls": 0, "workspace_bytes": decision.WorkspaceBytes(), "scope": "all arms receive canonical compiler inputs; finite arithmetic oracle, max one added candidate; reused synthetic development cohort; no selected native/Go execution in this audit"})
}
