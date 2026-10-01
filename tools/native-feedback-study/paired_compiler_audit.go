package main

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const pairedProtocol = "docs/paired-compiler-context-preregistration-20261002.md"
const pairedDatasetSHA = "8c2b4fb884e001f7a58fa84ce862f61abe0f99b26855064d0011f631d67cecb3"

var pairedArms = []string{"js0", "js01", "js03"}
var pairedVariants = []string{"fp32", "ptq_ternary", "qat_ternary"}

type pairedCounts struct {
	Views        int     `json:"views"`
	Intent       int     `json:"initial_intention_label_agreement"`
	Accepted     int     `json:"initial_sparse_accepted_set_coverage"`
	Ties         int     `json:"sparse_ambiguous_views"`
	Before       int     `json:"full_contract_cases_before_tdd"`
	After        int     `json:"full_contract_cases_after_tdd"`
	Cases        int     `json:"full_contract_cases"`
	Complete     int     `json:"whole_full_contract_before_tdd"`
	CompleteTDD  int     `json:"whole_full_contract_after_tdd"`
	Extras       int     `json:"additional_candidate_attempts"`
	Calls        int     `json:"actual_model_predictions"`
	Declines     int     `json:"model_abstentions_observed"`
	Disagree     int     `json:"bilingual_initial_disagreements"`
	FiniteNLL    float64 `json:"mean_sparse_finite_nll"`
	PredictionNS []int64 `json:"prediction_ns"`
}

type pairedScore struct {
	Total    pairedCounts            `json:"total"`
	Families map[string]pairedCounts `json:"family_language"`
}

type pairedSelection struct {
	Schema      string                 `json:"schema"`
	ID          string                 `json:"selected_candidate"`
	Dataset     string                 `json:"dataset_sha256"`
	Source      string                 `json:"runner_revision"`
	Scores      map[string]pairedScore `json:"calibration"`
	Metadata    string                 `json:"metadata_sha256"`
	Weights     string                 `json:"weights_sha256"`
	Model       string                 `json:"model_relative_path"`
	Predictions int                    `json:"actual_calibration_predictions"`
	Rule        string                 `json:"rule"`
}

type pairedAuditReport struct {
	Schema       string                 `json:"schema"`
	Status       string                 `json:"status"`
	Calibration  map[string]pairedScore `json:"calibration"`
	Development  map[string]pairedScore `json:"development"`
	Selected     string                 `json:"selected_candidate"`
	SelectionSHA string                 `json:"selection_sha256"`
	Parity       map[string]float64     `json:"parity_max_absolute_error"`
	Calls        int                    `json:"actual_model_predictions"`
	ParityCalls  int                    `json:"actual_parity_predictions"`
	TotalCalls   int                    `json:"total_actual_go_predictions"`
	Control      map[string]bool        `json:"zero_weight_matches_prior_model"`
	Workspace    int                    `json:"workspace_bytes"`
	Scope        string                 `json:"scope"`
}

func loadPairedModels(root string) (map[string]*decision.Model, error) {
	raw, err := read(filepath.Join(root, "report.json"))
	if err != nil {
		return nil, err
	}
	var report struct {
		Status  string `json:"status"`
		Steps   int    `json:"optimizer_steps"`
		Exports map[string]map[string]struct {
			Metadata string `json:"metadata_sha256"`
			Weights  string `json:"weights_sha256"`
		} `json:"exports"`
	}
	if err = json.Unmarshal(raw, &report); err != nil || report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 840 {
		return nil, errors.New("bounded paired training report required")
	}
	models := map[string]*decision.Model{}
	for _, arm := range pairedArms {
		for _, variant := range pairedVariants {
			model, err := decision.LoadPath(filepath.Join(root, arm, "models", variant, "model.json"))
			pin := report.Exports[arm][variant]
			if err != nil || model.MetadataSHA256() != pin.Metadata || model.WeightsSHA256() != pin.Weights || model.FeatureVersion() != "split_context_intent_ngrams_v2" {
				return nil, errors.New("paired model/source feature pins differ")
			}
			models[arm+"-"+variant] = model
		}
	}
	return models, nil
}

func addPairedCount(count *pairedCounts, row compilerTrainingRow, search pathplan.SearchResult) error {
	if len(search.Attempts) == 0 || len(search.EligibleProbabilities) != 1 || search.Selection.ModelCalls != 1 ||
		len(search.Selection.Receipts) != 1 || search.Selection.Receipts[0].IntentSHA256 != row.SHA || search.Evaluated < 1 || search.Evaluated > 2 {
		return errors.New("paired single canonical judgment bound differs")
	}
	initial := search.InitialProposals["structure"]
	if initial != row.Options[0] && initial != row.Options[1] {
		return errors.New("illegal initial path")
	}
	if search.Attempts[0].Choices["structure"] != initial || search.TrainingTotal != 12 || search.SelectedTrainingPassed != 12 ||
		search.Attempts[len(search.Attempts)-1].Passed != 12 {
		return errors.New("complete assembly/initial choice counters differ")
	}
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
		count.Ties++
	}
	count.Before += search.Attempts[0].Passed
	count.After += search.SelectedTrainingPassed
	count.Cases += search.TrainingTotal
	if search.Attempts[0].Passed == search.TrainingTotal {
		count.Complete++
	}
	if search.SelectedTrainingPassed == search.TrainingTotal {
		count.CompleteTDD++
	}
	count.Extras += search.Evaluated - 1
	count.Calls += search.Selection.ModelCalls
	count.Declines += search.ModelAbstentionsObserved
	count.PredictionNS = append(count.PredictionNS, search.Selection.Receipts[0].PredictNS)
	p := search.EligibleProbabilities[0]
	if math.IsNaN(p[0]) || math.IsNaN(p[1]) || p[0] < 0 || p[1] < 0 || math.Abs(p[0]+p[1]-1) > 1e-6 {
		return errors.New("finite eligible probabilities required")
	}
	for i := range p {
		count.FiniteNLL -= float64(row.Targets[i]) * math.Log(math.Max(p[i], 1e-12))
	}
	return nil
}

func summarizePaired(rows []compilerTrainingRow, split string, observations []compilerModelObservation) (pairedScore, error) {
	score := pairedScore{Families: map[string]pairedCounts{}}
	pairs := map[string]string{}
	index := 0
	for _, row := range rows {
		if row.Split != split {
			continue
		}
		if index >= len(observations) {
			return score, errors.New("paired observations missing")
		}
		o := observations[index]
		index++
		if o.ID != row.ID || o.Input != row.SHA {
			return score, errors.New("canonical observation order/hash differs")
		}
		doc, _, err := compilerNativeDocument(row)
		if err != nil {
			return score, err
		}
		prepared, err := pathplan.Prepare(doc.Plan)
		if err != nil {
			return score, err
		}
		if err = auditOwnContextAttempts(prepared, doc, o.Search); err != nil {
			return score, err
		}
		if err = addPairedCount(&score.Total, row, o.Search); err != nil {
			return score, err
		}
		key := row.Family + "/" + row.Language
		family := score.Families[key]
		if err = addPairedCount(&family, row, o.Search); err != nil {
			return score, err
		}
		score.Families[key] = family
		initial := o.Search.InitialProposals["structure"]
		if prior, ok := pairs[row.Pair]; ok {
			if prior != initial {
				score.Total.Disagree++
			}
			delete(pairs, row.Pair)
		} else {
			pairs[row.Pair] = initial
		}
	}
	if index != len(observations) || len(pairs) != 0 {
		return score, errors.New("paired cohort denominator differs")
	}
	finishPairedCount(&score.Total)
	for key, count := range score.Families {
		finishPairedCount(&count)
		score.Families[key] = count
	}
	return score, nil
}

func finishPairedCount(count *pairedCounts) {
	if count.Views > 0 {
		count.FiniteNLL /= float64(count.Views)
	}
	sort.Slice(count.PredictionNS, func(i, j int) bool { return count.PredictionNS[i] < count.PredictionNS[j] })
}

func choosePaired(scores map[string]pairedScore) (string, error) {
	var ids []string
	for id, score := range scores {
		c := score.Total
		if c.Views == 160 && c.Calls == 160 && c.CompleteTDD == 160 && c.After == c.Cases && c.Cases == 1920 && !math.IsNaN(c.FiniteNLL) && !math.IsInf(c.FiniteNLL, 0) {
			ids = append(ids, id)
		}
	}
	if len(ids) != 9 {
		return "", errors.New("all nine complete calibration candidates required")
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := scores[ids[i]].Total, scores[ids[j]].Total
		if a.Extras != b.Extras {
			return a.Extras < b.Extras
		}
		if a.Disagree != b.Disagree {
			return a.Disagree < b.Disagree
		}
		if a.FiniteNLL != b.FiniteNLL {
			return a.FiniteNLL < b.FiniteNLL
		}
		return ids[i] < ids[j]
	})
	return ids[0], nil
}

func pairedEvaluate(rows []compilerTrainingRow, split string, model *decision.Model) ([]compilerModelObservation, error) {
	var result []compilerModelObservation
	for _, row := range rows {
		if row.Split != split {
			continue
		}
		search, err := observeCompilerModel(row, model)
		if err != nil {
			return nil, err
		}
		result = append(result, compilerModelObservation{row.ID, row.SHA, search})
	}
	return result, nil
}

func runPairedCompilerAudit(curriculum, models, output, revision string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	rows, err := compilerTrainingRows(curriculum)
	if err != nil {
		return err
	}
	data, err := readCompilerCurriculumFile(filepath.Join(curriculum, "dataset.jsonl"))
	if err != nil || hash(data) != pairedDatasetSHA {
		return errors.New("frozen paired curriculum differs")
	}
	loaded, err := loadPairedModels(models)
	if err != nil {
		return err
	}
	protocol, err := read(pairedProtocol)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	pins := map[string]map[string]string{}
	for id, model := range loaded {
		pins[id] = map[string]string{"metadata": model.MetadataSHA256(), "weights": model.WeightsSHA256()}
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/paired-compiler-audit-preexecution/v1", "source_revision": revision, "dataset_sha256": hash(data), "protocol_sha256": hash(protocol), "models": pins, "planned_calibration_predictions": 1440, "planned_development_predictions": 2880, "planned_parity_predictions": 288, "new_independent_intentions": 0}); err != nil {
		return err
	}
	parity := map[string]float64{}
	for _, arm := range pairedArms {
		parity[arm], err = compilerParity(models, arm)
		if err != nil {
			return err
		}
	}
	calibration := map[string]pairedScore{}
	for _, arm := range pairedArms {
		for _, variant := range pairedVariants {
			id := arm + "-" + variant
			o, err := pairedEvaluate(rows, "calibration", loaded[id])
			if err != nil {
				return err
			}
			calibration[id], err = summarizePaired(rows, "calibration", o)
			if err != nil {
				return err
			}
			if err = save(filepath.Join(output, id+"-calibration.json"), o); err != nil {
				return err
			}
		}
	}
	selected, err := choosePaired(calibration)
	if err != nil {
		return err
	}
	var selectedPath string
	for _, arm := range pairedArms {
		for _, variant := range pairedVariants {
			if arm+"-"+variant == selected {
				selectedPath = filepath.Join(arm, "models", variant, "model.json")
			}
		}
	}
	selection := pairedSelection{Schema: "gooo/paired-compiler-calibration-selection/v1", ID: selected, Dataset: hash(data), Source: revision, Scores: calibration, Metadata: loaded[selected].MetadataSHA256(), Weights: loaded[selected].WeightsSHA256(), Model: selectedPath, Predictions: 1440, Rule: "calibration only: extra full-contract candidates, bilingual disagreements, finite NLL, lexicographic ID; frozen before development predictions"}
	if err = save(filepath.Join(output, "selection.json"), selection); err != nil {
		return err
	}
	selectionRaw, err := read(filepath.Join(output, "selection.json"))
	if err != nil {
		return err
	}
	development := map[string]pairedScore{}
	for _, arm := range pairedArms {
		for _, variant := range pairedVariants {
			id := arm + "-" + variant
			o, err := pairedEvaluate(rows, "test", loaded[id])
			if err != nil {
				return err
			}
			development[id], err = summarizePaired(rows, "test", o)
			if err != nil {
				return err
			}
			if err = save(filepath.Join(output, id+"-development.json"), o); err != nil {
				return err
			}
		}
	}
	control := map[string]bool{}
	for _, variant := range pairedVariants {
		prior, err := decision.LoadPath(filepath.Join("runs/compiler-context-training-mps-20261001/compiler_context/models", variant, "model.json"))
		if err != nil {
			return err
		}
		control[variant] = prior.WeightsSHA256() == loaded["js0-"+variant].WeightsSHA256() && prior.MetadataSHA256() == loaded["js0-"+variant].MetadataSHA256()
	}
	return save(filepath.Join(output, "report.json"), pairedAuditReport{"gooo/paired-compiler-model-audit/v1", "PASS", calibration, development, selected, hash(selectionRaw), parity, 4320, 288, 4608, control, decision.WorkspaceBytes(), "reused finite diagnostic cohort; selection frozen on calibration before development; max one extra candidate; all nine variants retained; no default promotion or general correctness claim"})
}

func readPairedSelection(models, path string) (pairedSelection, error) {
	var selection pairedSelection
	raw, err := read(path)
	if err != nil {
		return selection, err
	}
	if err = json.Unmarshal(raw, &selection); err != nil {
		return selection, err
	}
	selected, err := choosePaired(selection.Scores)
	if err != nil || selection.Schema != "gooo/paired-compiler-calibration-selection/v1" || selection.Dataset != pairedDatasetSHA || selected != selection.ID || selection.Predictions != 1440 {
		return selection, errors.New("calibration-only selection differs")
	}
	loaded, err := loadPairedModels(models)
	if err != nil {
		return selection, err
	}
	model := loaded[selection.ID]
	if model == nil || model.MetadataSHA256() != selection.Metadata || model.WeightsSHA256() != selection.Weights ||
		filepath.Clean(selection.Model) != selection.Model || filepath.IsAbs(selection.Model) {
		return selection, errors.New("selected model pins differ")
	}
	validPath := false
	for _, arm := range pairedArms {
		for _, variant := range pairedVariants {
			if arm+"-"+variant == selection.ID {
				validPath = selection.Model == filepath.Join(arm, "models", variant, "model.json")
			}
		}
	}
	if !validPath {
		return selection, errors.New("selected model path differs")
	}
	return selection, nil
}

// Reconstructs finite scores and source bindings without model or native calls.
func auditPairedCompiler(curriculum, models, output string) error {
	rows, err := compilerTrainingRows(curriculum)
	if err != nil {
		return err
	}
	selection, err := readPairedSelection(models, filepath.Join(output, "selection.json"))
	if err != nil {
		return err
	}
	loaded, err := loadPairedModels(models)
	if err != nil {
		return err
	}
	raw, err := read(filepath.Join(output, "report.json"))
	if err != nil {
		return err
	}
	var report pairedAuditReport
	if err = json.Unmarshal(raw, &report); err != nil || report.Schema != "gooo/paired-compiler-model-audit/v1" || report.Status != "PASS" || report.Calls != 4320 || report.ParityCalls != 288 || report.TotalCalls != 4608 || report.Selected != selection.ID {
		return errors.New("paired audit report denominator/selection differs")
	}
	selectionRaw, err := read(filepath.Join(output, "selection.json"))
	if err != nil || hash(selectionRaw) != report.SelectionSHA {
		return errors.New("frozen selection hash differs")
	}
	for id, model := range loaded {
		for _, split := range []string{"calibration", "development"} {
			raw, err := read(filepath.Join(output, id+"-"+split+".json"))
			if err != nil {
				return err
			}
			var observations []compilerModelObservation
			if err = json.Unmarshal(raw, &observations); err != nil {
				return err
			}
			for _, o := range observations {
				if o.Search.Selection.MetadataSHA256 != model.MetadataSHA256() || o.Search.Selection.WeightsSHA256 != model.WeightsSHA256() {
					return errors.New("observation model pins differ")
				}
			}
			rowSplit := split
			expected := report.Calibration[id]
			if split == "development" {
				rowSplit = "test"
				expected = report.Development[id]
			}
			actual, err := summarizePaired(rows, rowSplit, observations)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(actual, expected) || (split == "calibration" && !reflect.DeepEqual(actual, selection.Scores[id])) {
				return errors.New("reconstructed finite scores differ")
			}
		}
	}
	return nil
}
