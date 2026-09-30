// audit-path-search records bounded model-ranked TDD over reserved bilingual
// instructions. Expected cases come from independent arithmetic, not the model.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type instruction struct {
	ID            string `json:"id"`
	Family        string `json:"family"`
	Configuration int    `json:"configuration"`
	Reverse       bool   `json:"reverse"`
	Language      string `json:"language"`
	View          string `json:"view"`
	Text          string `json:"text"`
	Gold          string `json:"gold_label"`
}
type caseResult struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
	Passed   bool  `json:"passed"`
}
type observation struct {
	Instruction instruction           `json:"instruction"`
	Search      pathplan.SearchResult `json:"search"`
	Holdout     []caseResult          `json:"unseen_input_cases"`
	ElapsedNS   int64                 `json:"search_elapsed_ns"`
	GoSHA       string                `json:"go_source_sha256"`
}
type counter struct {
	Views                   int `json:"views"`
	InitialCorrect          int `json:"initial_typed_label_correct"`
	InitialTrainingComplete int `json:"initial_candidate_training_complete"`
	FinalLabelCorrect       int `json:"selected_typed_label_correct"`
	TrainingComplete        int `json:"finite_training_complete_views"`
	Candidates              int `json:"candidates_evaluated"`
	TypeRejected            int `json:"candidates_type_rejected"`
	Unattempted             int `json:"unattempted_declared_candidates"`
	HoldoutCases            int `json:"unseen_input_cases"`
	HoldoutPassed           int `json:"unseen_input_cases_passed"`
	WholeHoldout            int `json:"whole_unseen_input_views_passed"`
	Predictions             int `json:"local_model_predictions"`
	Abstentions             int `json:"global_confidence_abstentions_observed"`
}
type arm struct {
	counter
	ByLanguage         map[string]counter `json:"by_language"`
	ByFamily           map[string]counter `json:"by_family"`
	ModelMetadataSHA   string             `json:"model_metadata_sha256,omitempty"`
	ModelWeightsSHA    string             `json:"model_weights_sha256,omitempty"`
	RowsSHA            string             `json:"raw_rows_sha256"`
	SearchMedianNS     int64              `json:"search_median_ns"`
	SearchP95NS        int64              `json:"search_p95_ns"`
	PredictionMedianNS int64              `json:"prediction_median_ns,omitempty"`
	PredictionP95NS    int64              `json:"prediction_p95_ns,omitempty"`
}

func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func probe() ([]instruction, error) {
	var rows []instruction
	for _, family := range pathstudy.Families {
		for config := 64; config < 96; config++ {
			for _, reverse := range []bool{false, true} {
				for _, language := range []string{"en", "ko"} {
					plain, err := pathstudy.ProbeInstruction(family, reverse, config, language)
					if err != nil {
						return nil, err
					}
					for _, view := range []string{"plain", "gooo", "prov"} {
						text, err := pathstudy.View(plain, family, view, config)
						if err != nil || len(text) > 512 {
							return nil, errors.New("invalid reserved probe text")
						}
						rows = append(rows, instruction{fmt.Sprintf("%s-%d-%t-%s-%s", family, config, reverse, language, view), family, config, reverse, language, view, text, pathstudy.GoldLabel(family, reverse)})
					}
				}
			}
		}
	}
	return rows, nil
}
func finiteCases(r instruction) ([]pathplan.TestCase, []int64, error) {
	a, _ := pathstudy.Parameters(r.Configuration)
	trainingInputs := []int64{0, 1, a, a + 1}
	seen := map[int64]bool{}
	var training []pathplan.TestCase
	for _, input := range trainingInputs {
		if seen[input] {
			continue
		}
		seen[input] = true
		want, err := pathstudy.Oracle(r.Family, r.Reverse, r.Configuration, input)
		if err != nil {
			return nil, nil, err
		}
		training = append(training, pathplan.TestCase{Input: input, Expected: want})
	}
	var holdout []int64
	for _, input := range pathstudy.Inputs(r.Configuration) {
		if !seen[input] {
			holdout = append(holdout, input)
		}
	}
	return training, holdout, nil
}
func add(c *counter, row observation) {
	c.Views++
	c.Predictions += row.Search.Selection.ModelCalls
	c.Abstentions += row.Search.ModelAbstentionsObserved
	c.Candidates += row.Search.Evaluated
	c.TypeRejected += row.Search.TypeRejected
	c.Unattempted += row.Search.Unattempted
	if row.Search.InitialProposals["structure"] == row.Instruction.Gold {
		c.InitialCorrect++
	}
	if len(row.Search.Attempts) > 0 && row.Search.Attempts[0].Status == "EVALUATED" && row.Search.Attempts[0].Passed == row.Search.Attempts[0].Total {
		c.InitialTrainingComplete++
	}
	if row.Search.Selection.Choices["structure"] == row.Instruction.Gold {
		c.FinalLabelCorrect++
	}
	if row.Search.Status == "TRAINING_COMPLETE" {
		c.TrainingComplete++
	}
	whole := len(row.Holdout) > 0
	for _, test := range row.Holdout {
		c.HoldoutCases++
		if test.Passed {
			c.HoldoutPassed++
		} else {
			whole = false
		}
	}
	if whole {
		c.WholeHoldout++
	}
}
func percentile(values []int64, q float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values[int(float64(len(values)-1)*q)]
}
func run(modelsRoot, output, revision string) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("source revision must be a full commit")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("output must be fresh")
	}
	models := map[string]*decision.Model{}
	modelBindings := map[string]any{}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		model, err := decision.LoadPath(filepath.Join(modelsRoot, variant, "model.json"))
		if err != nil {
			return err
		}
		models[variant] = model
		modelBindings[variant] = map[string]any{"metadata_sha256": model.MetadataSHA256(), "weights_sha256": model.WeightsSHA256(), "resident_tensor_bytes": model.ResidentTensorBytes(), "matrix_scale_bytes": model.MatrixScaleBytes()}
	}
	rows, err := probe()
	if err != nil {
		return err
	}
	if len(rows) != 1920 {
		return errors.New("probe count mismatch")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	probeFile, err := os.Create(filepath.Join(output, "probe.jsonl"))
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(probeFile)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			probeFile.Close()
			return err
		}
	}
	if err := probeFile.Close(); err != nil {
		return err
	}
	probeRaw, err := os.ReadFile(filepath.Join(output, "probe.jsonl"))
	if err != nil {
		return err
	}
	sources := map[string]string{}
	for _, name := range []string{"tools/audit-path-search/main.go", "internal/pathplan/search.go", "internal/pathstudy/study.go", "internal/decision/model.go"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		sources[name] = digest(raw)
	}
	if err := save(filepath.Join(output, "preexecution.json"), map[string]any{
		"schema": "gooo/typed-path-probe-preexecution/v1", "source_revision": revision, "source_files_sha256": sources, "probe_sha256": digest(probeRaw), "model_bindings": modelBindings,
		"planned_views_per_arm": 1920, "original_instructions": 640, "program_configurations": 320, "planned_local_model_predictions": 5760,
		"candidate_budget": 2, "deadline_per_view_ms": 2000, "training_inputs": "deduplicated {0,1,offset,offset+1}", "holdout_inputs": "fixed boundary inputs excluding all training inputs",
		"checkpoint_updates": 0, "warmups": 0, "retries": 0, "seeds": []string{}, "planned_native_calls": 0, "planned_external_calls": 0,
		"model_choice": "Positioned random initialization is one fixed comparison arm; this probe cannot select a checkpoint or establish model superiority.",
	}); err != nil {
		return err
	}
	results := map[string]arm{}
	started := time.Now()
	for _, variant := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
		model := models[variant]
		cell := arm{ByLanguage: map[string]counter{}, ByFamily: map[string]counter{}}
		if model != nil {
			cell.ModelMetadataSHA, cell.ModelWeightsSHA = model.MetadataSHA256(), model.WeightsSHA256()
		}
		file, err := os.Create(filepath.Join(output, variant+"-rows.jsonl"))
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(file)
		encoder.SetEscapeHTML(false)
		var elapsed, predictionTimes []int64
		for _, r := range rows {
			plan, err := pathstudy.Fixture(r.Family, r.Configuration, r.Text)
			if err != nil {
				file.Close()
				return err
			}
			training, holdout, err := finiteCases(r)
			if err != nil {
				file.Close()
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			start := time.Now()
			search, program, err := pathplan.Search(ctx, plan, model, training, 2, "")
			spent := time.Since(start).Nanoseconds()
			cancel()
			if err != nil {
				file.Close()
				return err
			}
			observed := observation{Instruction: r, Search: search, ElapsedNS: spent, GoSHA: digest([]byte(program.GoSource()))}
			elapsed = append(elapsed, spent)
			for _, receipt := range search.Selection.Receipts {
				if model != nil {
					predictionTimes = append(predictionTimes, receipt.PredictNS)
				}
			}
			// Holdout is evaluated only after the search has returned a selection.
			for _, input := range holdout {
				want, err := pathstudy.Oracle(r.Family, r.Reverse, r.Configuration, input)
				if err != nil {
					file.Close()
					return err
				}
				value, err := program.Evaluate(input)
				if err != nil {
					file.Close()
					return err
				}
				observed.Holdout = append(observed.Holdout, caseResult{input, want, value.Int, want == value.Int})
			}
			add(&cell.counter, observed)
			language := cell.ByLanguage[r.Language]
			add(&language, observed)
			cell.ByLanguage[r.Language] = language
			family := cell.ByFamily[r.Family]
			add(&family, observed)
			cell.ByFamily[r.Family] = family
			if err := encoder.Encode(observed); err != nil {
				file.Close()
				return err
			}
		}
		if err := file.Close(); err != nil {
			return err
		}
		raw, err := os.ReadFile(filepath.Join(output, variant+"-rows.jsonl"))
		if err != nil {
			return err
		}
		cell.RowsSHA = digest(raw)
		cell.SearchMedianNS, cell.SearchP95NS = percentile(elapsed, .5), percentile(elapsed, .95)
		cell.PredictionMedianNS, cell.PredictionP95NS = percentile(predictionTimes, .5), percentile(predictionTimes, .95)
		results[variant] = cell
		if err := save(filepath.Join(output, variant+"-summary.json"), cell); err != nil {
			return err
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/typed-path-probe-tdd-report/v1", "status": "OBSERVED", "source_revision": revision, "probe_sha256": digest(probeRaw), "arms": results, "wall_ns": time.Since(started).Nanoseconds(), "native_compiler_calls": 0, "external_provider_calls": 0, "checkpoint_updates": 0,
		"limitations": []string{"Five families with exactly two compiler-owned alternatives each; not arbitrary body synthesis.", "1920 views represent 640 bilingual instructions and 320 program configurations.", "Conditional eligible probabilities rank alternatives and are not calibrated probabilities of intent completion.", "Training cases are authored by independent arithmetic; hidden-input checks remain finite and synthetic.", "This audit executes the typed Go interpreter; independent native/generated-Go replay is a separate measurement.", "Single sequential run; timings include typed validation/search and are not a host CPU utilization increase.", "Global confidence abstention is observed; finite TDD acceptance is a separate explicitly requested experiment."}})
}
func main() {
	models := flag.String("models", "", "fixed structural model variants directory")
	output := flag.String("output", "", "fresh evidence directory")
	revision := flag.String("source-revision", "", "full runner source commit")
	flag.Parse()
	if *models == "" || *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "audit-path-search: supply --models --output --source-revision")
		os.Exit(2)
	}
	if err := run(*models, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, "audit-path-search:", err)
		os.Exit(1)
	}
	fmt.Println(`{"status":"OBSERVED","views_per_arm":1920,"local_model_predictions":5760,"native_calls":0,"external_calls":0}`)
}
