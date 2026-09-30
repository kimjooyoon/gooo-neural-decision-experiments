package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

type instruction struct {
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
type manifest struct {
	Schema     string         `json:"schema"`
	DatasetSHA string         `json:"dataset_sha256"`
	TotalRows  int            `json:"total_rows"`
	Rows       map[string]int `json:"rows"`
}
type parityRow struct {
	Variant       string       `json:"variant"`
	Text          string       `json:"text"`
	Features      [256]float32 `json:"features"`
	Logits        [8]float64   `json:"logits"`
	Probabilities [8]float64   `json:"probabilities"`
	Label         string       `json:"selected_label"`
}
type counter struct {
	Observed        int `json:"observed_views"`
	RawCorrect      int `json:"raw_label_correct"`
	SelectedCorrect int `json:"selected_label_correct"`
	Applied         int `json:"model_applied"`
	AppliedCorrect  int `json:"applied_label_correct"`
	Cases           int `json:"finite_cases"`
	Passed          int `json:"finite_cases_passed"`
	Whole           int `json:"whole_finite_program_views_correct"`
}
type cellReport struct {
	counter
	Families         map[string]counter `json:"by_family"`
	MetadataSHA      string             `json:"metadata_sha256,omitempty"`
	WeightsSHA       string             `json:"weights_sha256,omitempty"`
	ResidentBytes    int                `json:"resident_tensors_bytes,omitempty"`
	MatrixScaleBytes int                `json:"matrix_scale_bytes,omitempty"`
	Threshold        float32            `json:"confidence_threshold,omitempty"`
	Predictions      int                `json:"local_model_predictions"`
	RowsSHA          string             `json:"rows_sha256"`
}
type caseResult struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
	Passed   bool  `json:"passed"`
}
type observedRow struct {
	ID        string             `json:"id"`
	ProgramID string             `json:"program_id"`
	Family    string             `json:"family"`
	View      string             `json:"view"`
	Gold      string             `json:"gold_label"`
	Selection pathplan.Selection `json:"selection"`
	GoooSHA   string             `json:"gooo_source_sha256"`
	GoSHA     string             `json:"go_source_sha256"`
	Cases     []caseResult       `json:"cases"`
}

func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func read(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, errors.New("invalid bounded regular input")
	}
	return os.ReadFile(path)
}

func loadRows(dataset string) ([]instruction, string, string, error) {
	raw, err := read(dataset, 8<<20)
	if err != nil {
		return nil, "", "", err
	}
	meta, err := read(filepath.Join(filepath.Dir(dataset), "manifest.json"), 64<<10)
	if err != nil {
		return nil, "", "", err
	}
	// The generator manifest carries additional frozen source/group metadata.
	if err := decision.RejectDuplicateJSONKeys(meta); err != nil {
		return nil, "", "", err
	}
	var m manifest
	if err := json.Unmarshal(meta, &m); err != nil || m.Schema != "gooo/typed-path-curriculum/v1" || m.DatasetSHA != digest(raw) || m.TotalRows != 6240 {
		return nil, "", "", errors.New("path dataset manifest mismatch")
	}
	counts := make(map[string]int)
	ids, texts := make(map[string]bool), make(map[string]bool)
	groups := map[string]string{}
	templates := map[string]string{}
	configs := map[string]string{}
	programs := map[string]string{}
	var rows []instruction
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 4096)
	for scanner.Scan() {
		var r instruction
		if err := strictjson.Decode(scanner.Bytes(), &r); err != nil {
			return nil, "", "", err
		}
		if ids[r.ID] || texts[r.Text] || len(r.Text) == 0 || len(r.Text) > 512 || r.ConfigurationIndex < 0 || r.ConfigurationIndex >= 64 {
			return nil, "", "", errors.New("invalid or repeated instruction")
		}
		wantSplit := "train"
		if r.ConfigurationIndex >= 48 {
			wantSplit = "test"
		} else if r.ConfigurationIndex >= 40 {
			wantSplit = "calibration"
		}
		if r.Split != wantSplit || r.Language != "en" && r.Language != "ko" || r.View != "plain" && r.View != "gooo" && r.View != "prov" {
			return nil, "", "", errors.New("invalid group partition or enum")
		}
		validLabel := false
		for _, label := range decision.PathLabels() {
			validLabel = validLabel || r.Label == label
		}
		if !validLabel {
			return nil, "", "", errors.New("label outside path ABI")
		}
		for _, pair := range []struct {
			values map[string]string
			key    string
		}{{groups, r.InstructionID}, {templates, r.TemplateID}, {configs, r.ConfigurationID}, {programs, r.ProgramID}} {
			if old := pair.values[pair.key]; old != "" && old != r.Split {
				return nil, "", "", errors.New("split leakage")
			}
			pair.values[pair.key] = r.Split
		}
		ids[r.ID], texts[r.Text] = true, true
		counts[r.Split]++
		if r.Split == "test" {
			rows = append(rows, r)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", "", err
	}
	if counts["train"] != 4800 || counts["calibration"] != 480 || counts["test"] != 960 || len(ids) != 6240 {
		return nil, "", "", errors.New("row counts differ")
	}
	return rows, digest(raw), digest(meta), nil
}

func validateParity(models map[string]*decision.Model, raw []byte) (map[string]any, error) {
	var fixture struct {
		Schema string      `json:"schema"`
		Rows   []parityRow `json:"rows"`
	}
	if err := strictjson.Decode(raw, &fixture); err != nil || fixture.Schema != "gooo/typed-path-parity/v1" || len(fixture.Rows) != 96 {
		return nil, errors.New("parity fixture differs")
	}
	maxError := 0.0
	counts := map[string]int{}
	for _, r := range fixture.Rows {
		model := models[r.Variant]
		if model == nil {
			return nil, errors.New("unknown parity variant")
		}
		var features [256]float32
		var workspace decision.Workspace
		var output decision.Prediction
		if err := decision.FeaturesInto(r.Text, &features); err != nil {
			return nil, err
		}
		if err := model.PredictInto(r.Text, &workspace, &output); err != nil {
			return nil, err
		}
		for i, value := range features {
			diff := math.Abs(float64(value - r.Features[i]))
			if diff > 1e-6 || math.IsNaN(diff) {
				return nil, errors.New("feature parity failure")
			}
		}
		for i := 0; i < 8; i++ {
			for _, pair := range [][2]float64{{float64(output.Logits[i]), r.Logits[i]}, {float64(output.Probabilities[i]), r.Probabilities[i]}} {
				diff := math.Abs(pair[0] - pair[1])
				if math.IsNaN(diff) || math.IsInf(diff, 0) || diff > 1e-4 {
					return nil, errors.New("logit/probability parity failure")
				}
				maxError = math.Max(maxError, diff)
			}
		}
		if model.PredictLabel(&output) != r.Label {
			return nil, errors.New("parity selected label differs")
		}
		counts[r.Variant]++
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		if counts[variant] != 32 {
			return nil, errors.New("parity variant count differs")
		}
	}
	return map[string]any{"decision": "PASS", "rows": 96, "local_model_predictions": 96, "tolerance": 1e-4, "maximum_logit_or_probability_absolute_error": maxError, "fixture_sha256": digest(raw)}, nil
}

func add(c *counter, r observedRow) {
	c.Observed++
	receipt := r.Selection.Receipts[0]
	if receipt.Proposed == r.Gold {
		c.RawCorrect++
	}
	if receipt.Selected == r.Gold {
		c.SelectedCorrect++
	}
	if receipt.Mode == "model_global_argmax" {
		c.Applied++
		if receipt.Selected == r.Gold {
			c.AppliedCorrect++
		}
	}
	whole := true
	for _, value := range r.Cases {
		c.Cases++
		if value.Passed {
			c.Passed++
		} else {
			whole = false
		}
	}
	if whole {
		c.Whole++
	}
}

func run(dataset, modelRoot, parityPath, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("output must be fresh")
	}
	rows, dataSHA, manifestSHA, err := loadRows(dataset)
	if err != nil {
		return err
	}
	models := map[string]*decision.Model{}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		model, err := decision.LoadPath(filepath.Join(modelRoot, variant, "model.json"))
		if err != nil {
			return err
		}
		models[variant] = model
	}
	parityRaw, err := read(parityPath, 2<<20)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	if err := save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/typed-path-audit-preexecution/v1", "dataset_sha256": dataSHA, "manifest_sha256": manifestSHA, "planned_test_views_per_variant": 960, "planned_heldout_model_predictions": 2880, "planned_parity_predictions": 96, "planned_external_calls": 0, "planned_native_calls": 0, "seeds": []string{}, "warmups": 0, "retries": 0}); err != nil {
		return err
	}
	parity, err := validateParity(models, parityRaw)
	if err != nil {
		return err
	}
	if err := save(filepath.Join(output, "parity-audit.json"), parity); err != nil {
		return err
	}
	results := map[string]cellReport{}
	for _, variant := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
		model := models[variant]
		report := cellReport{Families: make(map[string]counter)}
		if model != nil {
			report.MetadataSHA, report.WeightsSHA, report.ResidentBytes, report.MatrixScaleBytes, report.Threshold = model.MetadataSHA256(), model.WeightsSHA256(), model.ResidentTensorBytes(), model.MatrixScaleBytes(), model.ConfidenceThreshold()
		}
		var rawRows bytes.Buffer
		encoder := json.NewEncoder(&rawRows)
		encoder.SetEscapeHTML(false)
		for _, r := range rows {
			plan, err := pathstudy.Fixture(r.Family, r.ConfigurationIndex, r.Text)
			if err != nil {
				return err
			}
			selection, program, err := pathplan.Choose(plan, model, "")
			if err != nil {
				return err
			}
			report.Predictions += selection.ModelCalls
			observed := observedRow{ID: r.ID, ProgramID: r.ProgramID, Family: r.Family, View: r.View, Gold: r.Label, Selection: selection, GoooSHA: digest([]byte(program.GoooSource())), GoSHA: digest([]byte(program.GoSource()))}
			reverse := r.Label == pathstudy.GoldLabel(r.Family, true)
			for _, input := range pathstudy.Inputs(r.ConfigurationIndex) {
				want, err := pathstudy.Oracle(r.Family, reverse, r.ConfigurationIndex, input)
				if err != nil {
					return err
				}
				got, err := program.Evaluate(input)
				if err != nil {
					return err
				}
				observed.Cases = append(observed.Cases, caseResult{input, want, got.Int, want == got.Int})
			}
			add(&report.counter, observed)
			family := report.Families[r.Family]
			add(&family, observed)
			report.Families[r.Family] = family
			if err := encoder.Encode(observed); err != nil {
				return err
			}
		}
		report.RowsSHA = digest(rawRows.Bytes())
		if err := os.WriteFile(filepath.Join(output, variant+"-rows.jsonl"), rawRows.Bytes(), 0644); err != nil {
			return err
		}
		results[variant] = report
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/typed-path-go-audit/v1", "decision": "PASS", "dataset_sha256": dataSHA, "manifest_sha256": manifestSHA, "test_views": 960, "original_test_instructions": 320, "unique_test_program_configurations": 160, "heldout_model_predictions": 2880, "parity_model_predictions": 96, "native_compiler_calls": 0, "external_provider_calls": 0, "variants": results, "workspace_bytes": decision.WorkspaceBytes(), "scope": "Go numerical parity and typed-plan interpreter cases only; no independent generated-Go/native replay in this audit. Same instructions are repeated across plain/Gooo/PROV-O views and variants. Finite case correctness does not prove full intent."})
}

func main() {
	dataset := flag.String("dataset", "", "path curriculum JSONL")
	models := flag.String("models", "", "path model variants directory")
	parity := flag.String("parity", "", "Python parity fixture")
	output := flag.String("output", "", "fresh audit directory")
	flag.Parse()
	if *dataset == "" || *models == "" || *parity == "" || *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "audit-path-model: supply --dataset --models --parity --output")
		os.Exit(2)
	}
	if err := run(*dataset, *models, *parity, *output); err != nil {
		fmt.Fprintln(os.Stderr, "audit-path-model:", err)
		os.Exit(1)
	}
	fmt.Println(`{"decision":"PASS","heldout_model_predictions":2880,"parity_predictions":96,"native_calls":0,"external_calls":0}`)
}
