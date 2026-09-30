// gooo-model-audit independently checks exported Go inference against saved
// Python parity vectors and scores the frozen test split.
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
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const (
	paritySchema = "gooo/tiny-ir-decision-parity/v1"
	auditSchema  = "gooo/tiny-ir-decision-go-audit/v1"
	tolerance    = 1e-4
)

var variants = []string{"fp32", "ptq_ternary", "qat_ternary"}

type parityFile struct {
	Schema string      `json:"schema"`
	Rows   []parityRow `json:"rows"`
}

type parityRow struct {
	Variant       string    `json:"variant"`
	Text          string    `json:"text"`
	Features      []float64 `json:"features"`
	Logits        []float64 `json:"logits"`
	Probabilities []float64 `json:"probabilities"`
	SelectedLabel string    `json:"selected_label"`
}

type datasetRow struct {
	ID            string `json:"id"`
	Template      string `json:"template_id"`
	Configuration string `json:"configuration_id"`
	Language      string `json:"language"`
	Split         string `json:"split"`
	Text          string `json:"text"`
	Label         string `json:"label"`
}

type auditReport struct {
	Schema               string                    `json:"schema"`
	Decision             string                    `json:"decision"`
	Status               string                    `json:"status"`
	Tolerance            float64                   `json:"absolute_tolerance"`
	DatasetSHA256        string                    `json:"dataset_sha256"`
	ExpectedDatasetRows  int                       `json:"expected_dataset_rows"`
	ParitySHA256         string                    `json:"parity_sha256"`
	DecisionBinarySHA256 string                    `json:"decision_binary_sha256,omitempty"`
	RuntimeSourceSHA256  map[string]string         `json:"runtime_source_sha256"`
	Inputs               map[string]string         `json:"input_sha256"`
	Parity               map[string]paritySummary  `json:"parity"`
	TestScores           map[string]variantSummary `json:"test_scores"`
	CLICold              map[string]coldCLISummary `json:"cli_cold"`
	FailureCount         int                       `json:"failure_count"`
	Failures             []string                  `json:"failures,omitempty"`
	FailuresTruncated    bool                      `json:"failures_truncated"`
}

type paritySummary struct {
	Planned               int     `json:"planned"`
	Observed              int     `json:"observed"`
	Matched               int     `json:"matched"`
	FeaturesMaxAbsError   float64 `json:"features_max_abs_error"`
	LogitsMaxAbsError     float64 `json:"logits_max_abs_error"`
	ProbabilitiesMaxError float64 `json:"probabilities_max_abs_error"`
	SelectedLabelMatches  int     `json:"selected_label_matches"`
}

type variantSummary struct {
	Model       modelSummary            `json:"model"`
	Overall     scoreSummary            `json:"overall"`
	ByLanguage  map[string]scoreSummary `json:"by_language"`
	ByLabel     map[string]scoreSummary `json:"by_label"`
	ByTemplate  map[string]scoreSummary `json:"by_template"`
	Selective   selectiveSummary        `json:"selective"`
	TypedBridge bridgeSummary           `json:"typed_bridge"`
	HotPredict  hotBenchmarkSummary     `json:"hot_predict_benchmark"`
}

type hotBenchmarkSummary struct {
	Iterations    int   `json:"iterations"`
	NanosecondsOp int64 `json:"ns_per_op"`
	BytesOp       int64 `json:"bytes_per_op"`
	AllocsOp      int64 `json:"allocs_per_op"`
}

type coldCLISummary struct {
	Planned              int          `json:"planned"`
	Observed             int          `json:"observed"`
	BinarySHA256         string       `json:"decision_binary_sha256"`
	ProcessRSSConvention string       `json:"rss_convention"`
	Runs                 []coldCLIRun `json:"runs"`
}

type coldCLIRun struct {
	RunID           string `json:"run_id"`
	ExitCode        int    `json:"exit_code"`
	Status          string `json:"response_status"`
	WallNanoseconds int64  `json:"wall_nanoseconds"`
	PeakRSSBytes    *int64 `json:"peak_rss_bytes,omitempty"`
	RequestSHA256   string `json:"request_sha256"`
	StdoutSHA256    string `json:"stdout_sha256"`
	StderrSHA256    string `json:"stderr_sha256"`
	RequestPath     string `json:"request_path"`
	StdoutPath      string `json:"stdout_path"`
	StderrPath      string `json:"stderr_path"`
}

type modelSummary struct {
	Variant              string  `json:"variant"`
	MetadataSHA256       string  `json:"model_json_sha256"`
	WeightsSHA256        string  `json:"weights_sha256"`
	PackedFileBytes      int     `json:"packed_file_bytes"`
	MatrixTensorBytes    int     `json:"matrix_tensor_bytes"`
	BiasTensorBytes      int     `json:"bias_tensor_bytes"`
	ResidentTensorBytes  int     `json:"resident_tensor_bytes"`
	MatrixScaleBytes     int     `json:"matrix_scale_storage_bytes"`
	WorkspaceBytes       int     `json:"workspace_scratch_bytes_per_worker"`
	PredictionBytes      int     `json:"prediction_output_bytes"`
	PredictionValueBytes int     `json:"prediction_value_arrays_bytes"`
	Threshold            float32 `json:"confidence_threshold"`
}

type scoreSummary struct {
	Planned         int              `json:"planned"`
	Observed        int              `json:"observed"`
	Correct         int              `json:"correct"`
	Accuracy        *float64         `json:"accuracy"`
	NLL             *float64         `json:"nll"`
	BrierMulticlass *float64         `json:"brier_multiclass_sum"`
	ECEEqualWidth10 *float64         `json:"ece_equal_width_10"`
	CalibrationBins []calibrationBin `json:"calibration_bins"`
}

type calibrationBin struct {
	Lower          float64  `json:"lower"`
	Upper          float64  `json:"upper"`
	Count          int      `json:"count"`
	Accuracy       *float64 `json:"accuracy"`
	MeanConfidence *float64 `json:"mean_confidence"`
}

type selectiveSummary struct {
	Planned         int      `json:"planned"`
	Observed        int      `json:"observed"`
	Accepted        int      `json:"accepted"`
	CorrectAccepted int      `json:"correct_accepted"`
	Accuracy        *float64 `json:"accuracy"`
}

type bridgeSummary struct {
	Planned                   int            `json:"planned"`
	Observed                  int            `json:"observed"`
	Decision                  int            `json:"decision"`
	LowConfidenceAbstentions  int            `json:"low_confidence_abstentions"`
	TypeMismatchAbstentions   int            `json:"type_mismatch_abstentions"`
	EmittedTypedIR            int            `json:"emitted_typed_ir"`
	PredictedExpressionsBuilt int            `json:"predicted_expressions_built"`
	GoldIRConstructed         int            `json:"gold_ir_constructed"`
	GoldExpressionsBuilt      int            `json:"gold_expressions_built"`
	Statuses                  map[string]int `json:"statuses"`
}

type scoredRow struct {
	label      int
	correct    bool
	confidence float64
	probs      [decision.LabelCount]float64
	language   string
	template   string
}

func main() {
	modelsDir := flag.String("models", "", "directory containing one model bundle per variant")
	parityPath := flag.String("parity", "", "saved Python go-parity.json")
	datasetPath := flag.String("dataset", "", "frozen JSONL dataset")
	expectedRows := flag.Int("expected-rows", 2048, "explicit frozen row count, 1..8192; v1 default remains 2048")
	outputPath := flag.String("output", "", "audit report JSON output path")
	decisionBin := flag.String("decision-bin", "", "optional gooo-decision executable for cold CLI runs")
	coldDir := flag.String("cold-dir", "", "directory for raw cold CLI request/response captures")
	flag.Parse()
	if *modelsDir == "" || *parityPath == "" || *datasetPath == "" || *outputPath == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: gooo-model-audit --models DIR --parity go-parity.json --dataset dataset.jsonl --output report.json")
		os.Exit(2)
	}
	if (*decisionBin == "") != (*coldDir == "") {
		fmt.Fprintln(os.Stderr, "--decision-bin and --cold-dir must be provided together")
		os.Exit(2)
	}
	if err := runWithRows(*modelsDir, *parityPath, *datasetPath, *outputPath, *decisionBin, *coldDir, *expectedRows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(modelsDir, parityPath, datasetPath, outputPath, decisionBin, coldDir string) error {
	return runWithRows(modelsDir, parityPath, datasetPath, outputPath, decisionBin, coldDir, 2048)
}

func runWithRows(modelsDir, parityPath, datasetPath, outputPath, decisionBin, coldDir string, expectedRows int) error {
	if expectedRows < 1 || expectedRows > 8192 {
		return errors.New("expected dataset row count must be 1..8192")
	}
	report := auditReport{Schema: auditSchema, Decision: "PASS", Status: "PASS", Tolerance: tolerance,
		ExpectedDatasetRows: expectedRows, Inputs: make(map[string]string), Parity: make(map[string]paritySummary), TestScores: make(map[string]variantSummary), CLICold: make(map[string]coldCLISummary)}
	parityRaw, err := os.ReadFile(parityPath)
	if err != nil {
		return fmt.Errorf("read parity vectors: %w", err)
	}
	datasetRaw, err := os.ReadFile(datasetPath)
	if err != nil {
		return fmt.Errorf("read dataset: %w", err)
	}
	parity, err := decodeParity(parityRaw)
	if err != nil {
		return err
	}
	dataset, err := decodeDatasetWithRows(datasetRaw, expectedRows)
	if err != nil {
		return err
	}
	report.Inputs[filepath.Clean(parityPath)] = sha256Hex(parityRaw)
	report.Inputs[filepath.Clean(datasetPath)] = sha256Hex(datasetRaw)
	report.ParitySHA256 = sha256Hex(parityRaw)
	report.DatasetSHA256 = sha256Hex(datasetRaw)
	report.RuntimeSourceSHA256 = make(map[string]string)
	for _, source := range []string{"go.mod", "internal/decision/model.go", "internal/decision/bridge.go", "internal/decision/ir.go", "cmd/gooo-decision/main.go", "cmd/gooo-model-audit/main.go", "cmd/gooo-model-audit/rss_unix.go", "cmd/gooo-model-audit/rss_other.go"} {
		raw, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("read runtime source %s: %w", source, err)
		}
		report.RuntimeSourceSHA256[source] = sha256Hex(raw)
	}
	models := make(map[string]*decision.Model, len(variants))
	modelMetadataHashes := make(map[string]string, len(variants))
	for _, variant := range variants {
		modelPath := filepath.Join(modelsDir, variant, "model.json")
		model, err := decision.Load(modelPath)
		if err != nil {
			return fmt.Errorf("load %s model: %w", variant, err)
		}
		models[variant] = model
		modelRaw, err := os.ReadFile(modelPath)
		if err != nil {
			return err
		}
		weightsRaw, err := os.ReadFile(filepath.Join(modelsDir, variant, "weights.bin"))
		if err != nil {
			return err
		}
		report.Inputs[filepath.Clean(modelPath)] = sha256Hex(modelRaw)
		modelMetadataHashes[variant] = sha256Hex(modelRaw)
		report.Inputs[filepath.Clean(filepath.Join(modelsDir, variant, "weights.bin"))] = sha256Hex(weightsRaw)
	}

	for _, variant := range variants {
		rows := make([]parityRow, 0, len(parity.Rows)/len(variants))
		for _, row := range parity.Rows {
			if row.Variant == variant {
				rows = append(rows, row)
			}
		}
		summary := checkParity(variant, models[variant], rows, &report)
		report.Parity[variant] = summary
	}

	testRows := make([]datasetRow, 0)
	for _, row := range dataset {
		if row.Split == "test" {
			testRows = append(testRows, row)
		}
	}
	for _, variant := range variants {
		summary := scoreVariant(variant, models[variant], testRows, &report)
		summary.Model.MetadataSHA256 = modelMetadataHashes[variant]
		summary.HotPredict = benchmarkHotPredict(models[variant])
		report.TestScores[variant] = summary
	}
	if decisionBin != "" {
		for _, variant := range variants {
			cold := runColdCLI(variant, filepath.Join(modelsDir, variant, "model.json"), decisionBin, filepath.Join(coldDir, variant), &report)
			report.CLICold[variant] = cold
			if report.DecisionBinarySHA256 == "" {
				report.DecisionBinarySHA256 = cold.BinarySHA256
			} else if cold.BinarySHA256 != "" && report.DecisionBinarySHA256 != cold.BinarySHA256 {
				fail(&report, "decision binary bytes changed between cold CLI variant runs")
			}
		}
	}
	if report.FailureCount > 0 {
		report.Decision = "FAIL"
		report.Status = "FAIL"
	}
	if err := writeJSON(outputPath, report); err != nil {
		return fmt.Errorf("write audit report: %w", err)
	}
	if report.Status != "PASS" {
		return fmt.Errorf("audit completed with %d failures; details saved to %s", report.FailureCount, outputPath)
	}
	return nil
}

func decodeParity(raw []byte) (parityFile, error) {
	var file parityFile
	if err := decodeStrict(raw, &file); err != nil {
		return file, fmt.Errorf("decode parity vectors: %w", err)
	}
	if file.Schema != paritySchema || len(file.Rows) != 96 {
		return file, fmt.Errorf("parity schema or row count invalid: schema=%q rows=%d", file.Schema, len(file.Rows))
	}
	return file, nil
}

func decodeDataset(raw []byte) ([]datasetRow, error) {
	return decodeDatasetWithRows(raw, 2048)
}

func decodeDatasetWithRows(raw []byte, expectedRows int) ([]datasetRow, error) {
	if expectedRows < 1 || expectedRows > 8192 {
		return nil, errors.New("expected dataset row count must be 1..8192")
	}
	rows := make([]datasetRow, 0, expectedRows)
	ids, texts, groups := make(map[string]bool), make(map[string]bool), make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for line := 1; scanner.Scan(); line++ {
		var row datasetRow
		if err := decodeStrict(scanner.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("decode dataset line %d: %w", line, err)
		}
		if row.ID == "" || row.Text == "" || row.Template == "" || row.Configuration == "" || row.Language == "" || row.Split == "" || labelIndex(row.Label) < 0 {
			return nil, fmt.Errorf("dataset line %d has missing or unsupported fields", line)
		}
		if ids[row.ID] || texts[row.Text] || (row.Split != "train" && row.Split != "calibration" && row.Split != "test") ||
			(row.Language != "en" && row.Language != "ko") {
			return nil, fmt.Errorf("dataset line %d has duplicate identity/text or invalid enum", line)
		}
		if prior, exists := groups[row.Template]; exists && prior != row.Split {
			return nil, fmt.Errorf("dataset line %d crosses a template split", line)
		}
		ids[row.ID], texts[row.Text], groups[row.Template] = true, true, row.Split
		rows = append(rows, row)
		if len(rows) > expectedRows {
			return nil, fmt.Errorf("dataset exceeds frozen %d rows", expectedRows)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dataset JSONL: %w", err)
	}
	if len(rows) != expectedRows {
		return nil, fmt.Errorf("dataset row count %d; want %d", len(rows), expectedRows)
	}
	return rows, nil
}

func checkParity(variant string, model *decision.Model, rows []parityRow, report *auditReport) paritySummary {
	result := paritySummary{Planned: 32}
	if len(rows) != result.Planned {
		fail(report, fmt.Sprintf("%s parity rows=%d, want 32", variant, len(rows)))
	}
	var workspace decision.Workspace
	var prediction decision.Prediction
	for index, row := range rows {
		if row.Variant != variant {
			fail(report, fmt.Sprintf("%s parity row %d carries variant %q", variant, index, row.Variant))
			continue
		}
		var features [decision.FeatureDim]float32
		if err := decision.FeaturesInto(row.Text, &features); err != nil {
			fail(report, fmt.Sprintf("%s parity row %d feature extraction: %v", variant, index, err))
			continue
		}
		featureError, featuresOK := compareFloatVector(row.Features, features[:], tolerance)
		if len(row.Logits) != decision.LabelCount || len(row.Probabilities) != decision.LabelCount {
			fail(report, fmt.Sprintf("%s parity row %d logits/probability vector length invalid", variant, index))
			continue
		}
		if err := model.PredictInto(row.Text, &workspace, &prediction); err != nil {
			fail(report, fmt.Sprintf("%s parity row %d predict: %v", variant, index, err))
			continue
		}
		logitError, logitsOK := compareFloatVector(row.Logits, prediction.Logits[:], tolerance)
		probabilityError, probabilitiesOK := compareFloatVector(row.Probabilities, prediction.Probabilities[:], tolerance)
		result.Observed++
		result.FeaturesMaxAbsError = math.Max(result.FeaturesMaxAbsError, featureError)
		result.LogitsMaxAbsError = math.Max(result.LogitsMaxAbsError, logitError)
		result.ProbabilitiesMaxError = math.Max(result.ProbabilitiesMaxError, probabilityError)
		labelMatches := model.PredictLabel(&prediction) == row.SelectedLabel
		if labelMatches {
			result.SelectedLabelMatches++
		} else {
			fail(report, fmt.Sprintf("%s parity row %d selected label got=%q want=%q", variant, index, model.PredictLabel(&prediction), row.SelectedLabel))
		}
		if featuresOK && logitsOK && probabilitiesOK && labelMatches {
			result.Matched++
		} else {
			fail(report, fmt.Sprintf("%s parity row %d exceeds absolute tolerance %.1e (features %.3g logits %.3g probabilities %.3g)", variant, index, tolerance, featureError, logitError, probabilityError))
		}
	}
	return result
}

func scoreVariant(variant string, model *decision.Model, rows []datasetRow, report *auditReport) variantSummary {
	summary := variantSummary{
		Model: modelSummary{Variant: model.Variant(), WeightsSHA256: model.WeightsSHA256(), PackedFileBytes: model.PackedFileBytes(),
			MatrixTensorBytes: model.MatrixTensorBytes(), BiasTensorBytes: model.BiasTensorBytes(), ResidentTensorBytes: model.ResidentTensorBytes(),
			MatrixScaleBytes: model.MatrixScaleBytes(), WorkspaceBytes: decision.WorkspaceBytes(),
			PredictionBytes: decision.PredictionBytes(), PredictionValueBytes: decision.PredictionValueArrayBytes(), Threshold: model.ConfidenceThreshold()},
		ByLanguage: make(map[string]scoreSummary), ByLabel: make(map[string]scoreSummary), ByTemplate: make(map[string]scoreSummary),
		TypedBridge: bridgeSummary{Planned: len(rows), Statuses: make(map[string]int)},
	}
	groupsLanguage := make(map[string][]scoredRow)
	groupsLabel := make(map[string][]scoredRow)
	groupsTemplate := make(map[string][]scoredRow)
	plannedLanguage := make(map[string]int)
	plannedLabel := make(map[string]int)
	plannedTemplate := make(map[string]int)
	all := make([]scoredRow, 0, len(rows))
	var workspace decision.Workspace
	var prediction decision.Prediction
	for index, row := range rows {
		label := labelIndex(row.Label)
		plannedLanguage[row.Language]++
		plannedLabel[row.Label]++
		plannedTemplate[row.Template]++
		summaryScoreError := func(message string) {
			fail(report, fmt.Sprintf("%s test row %d (%s): %s", variant, index, row.ID, message))
		}
		if label < 0 {
			summaryScoreError("unsupported label")
			continue
		}
		if err := model.PredictInto(row.Text, &workspace, &prediction); err != nil {
			summaryScoreError("prediction failed: " + err.Error())
			continue
		}
		item := scoredRow{label: label, correct: prediction.TopIndex == label, confidence: float64(prediction.Confidence), language: row.Language, template: row.Template}
		for i, probability := range prediction.Probabilities {
			item.probs[i] = float64(probability)
		}
		all = append(all, item)
		groupsLanguage[row.Language] = append(groupsLanguage[row.Language], item)
		groupsLabel[row.Label] = append(groupsLabel[row.Label], item)
		groupsTemplate[row.Template] = append(groupsTemplate[row.Template], item)

		left, right := operandsFor(row.Label)
		goldIR, err := decision.BuildTypedBinary(row.Label, left, right)
		if err != nil {
			summaryScoreError("gold typed IR construction failed: " + err.Error())
		} else {
			summary.TypedBridge.GoldIRConstructed++
			if _, err := decision.AssembleGoExpression(goldIR); err != nil {
				summaryScoreError("gold expression assembly failed: " + err.Error())
			} else {
				summary.TypedBridge.GoldExpressionsBuilt++
			}
		}
		response, err := model.Decide(decision.DecisionRequest{Schema: decision.DecisionRequestSchema, Text: row.Text, Left: left, Right: right}, &workspace)
		if err != nil {
			summaryScoreError("typed bridge failed: " + err.Error())
			continue
		}
		summary.TypedBridge.Observed++
		summary.TypedBridge.Statuses[response.Status]++
		switch response.Status {
		case "decision":
			summary.TypedBridge.Decision++
			if response.TypedBinaryIR == nil {
				summaryScoreError("decision status omitted typed IR")
				continue
			}
			summary.TypedBridge.EmittedTypedIR++
			if _, err := decision.AssembleGoExpression(*response.TypedBinaryIR); err != nil {
				summaryScoreError("predicted expression assembly failed: " + err.Error())
			} else {
				summary.TypedBridge.PredictedExpressionsBuilt++
			}
		case "abstained":
			switch response.AbstainReason {
			case "LOW_CONFIDENCE":
				summary.TypedBridge.LowConfidenceAbstentions++
			case "TYPE_MISMATCH":
				summary.TypedBridge.TypeMismatchAbstentions++
			default:
				summaryScoreError("unknown abstention reason " + response.AbstainReason)
			}
		default:
			summaryScoreError("unknown bridge status " + response.Status)
		}
	}
	summary.Overall = summarize(all, len(rows))
	for key, planned := range plannedLanguage {
		summary.ByLanguage[key] = summarize(groupsLanguage[key], planned)
	}
	for key, planned := range plannedLabel {
		summary.ByLabel[key] = summarize(groupsLabel[key], planned)
	}
	for key, planned := range plannedTemplate {
		summary.ByTemplate[key] = summarize(groupsTemplate[key], planned)
	}
	selective := selectiveSummary{Planned: len(rows), Observed: len(all)}
	correctAccepted := 0
	for _, row := range all {
		if row.confidence >= float64(model.ConfidenceThreshold()) {
			selective.Accepted++
			if row.correct {
				correctAccepted++
			}
		}
	}
	selective.CorrectAccepted = correctAccepted
	if selective.Accepted > 0 {
		selective.Accuracy = floatPtr(float64(correctAccepted) / float64(selective.Accepted))
	}
	summary.Selective = selective
	if summary.Overall.Planned != 256 {
		fail(report, fmt.Sprintf("%s test planned=%d, want 256", variant, summary.Overall.Planned))
	}
	if summary.Overall.Observed != summary.Overall.Planned {
		fail(report, fmt.Sprintf("%s test observed=%d planned=%d", variant, summary.Overall.Observed, summary.Overall.Planned))
	}
	if summary.TypedBridge.Observed != summary.TypedBridge.Planned || summary.TypedBridge.GoldIRConstructed != summary.TypedBridge.Planned || summary.TypedBridge.GoldExpressionsBuilt != summary.TypedBridge.Planned {
		fail(report, fmt.Sprintf("%s typed bridge evidence is incomplete", variant))
	}
	return summary
}

func summarize(rows []scoredRow, planned int) scoreSummary {
	result := scoreSummary{Planned: planned, Observed: len(rows), CalibrationBins: make([]calibrationBin, 10)}
	if len(rows) == 0 {
		return result
	}
	var nllSum, brierSum, ece float64
	correct := 0
	binCount := [10]int{}
	binCorrect := [10]int{}
	binConfidence := [10]float64{}
	for _, row := range rows {
		if row.correct {
			correct++
		}
		probability := math.Max(row.probs[row.label], 1e-12)
		nllSum -= math.Log(probability)
		for i, p := range row.probs {
			want := 0.0
			if i == row.label {
				want = 1
			}
			delta := p - want
			brierSum += delta * delta
		}
		bin := int(row.confidence * 10)
		if bin > 9 {
			bin = 9
		}
		binCount[bin]++
		if row.correct {
			binCorrect[bin]++
		}
		binConfidence[bin] += row.confidence
	}
	result.Correct = correct
	result.Accuracy = floatPtr(float64(correct) / float64(len(rows)))
	result.NLL = floatPtr(nllSum / float64(len(rows)))
	result.BrierMulticlass = floatPtr(brierSum / float64(len(rows)))
	for i := 0; i < 10; i++ {
		lower, upper := float64(i)/10, float64(i+1)/10
		bin := calibrationBin{Lower: lower, Upper: upper, Count: binCount[i]}
		if i == 9 {
			bin.Upper = 1
		}
		if binCount[i] > 0 {
			accuracy := float64(binCorrect[i]) / float64(binCount[i])
			confidence := binConfidence[i] / float64(binCount[i])
			bin.Accuracy = floatPtr(accuracy)
			bin.MeanConfidence = floatPtr(confidence)
			ece += float64(binCount[i]) / float64(len(rows)) * math.Abs(accuracy-confidence)
		}
		result.CalibrationBins[i] = bin
	}
	result.ECEEqualWidth10 = floatPtr(ece)
	return result
}

func operandsFor(label string) (decision.Identifier, decision.Identifier) {
	if label == "and" || label == "or" {
		return decision.Identifier{Name: "left_value", Type: decision.TypeBool}, decision.Identifier{Name: "right_value", Type: decision.TypeBool}
	}
	return decision.Identifier{Name: "left_value", Type: decision.TypeInt}, decision.Identifier{Name: "right_value", Type: decision.TypeInt}
}

func labelIndex(label string) int {
	for index, value := range decision.Labels() {
		if value == label {
			return index
		}
	}
	return -1
}

func compareFloatVector(want []float64, got []float32, maxDifference float64) (float64, bool) {
	if len(want) != len(got) {
		return math.Inf(1), false
	}
	maximum := 0.0
	valid := true
	for i, value := range got {
		difference := math.Abs(want[i] - float64(value))
		maximum = math.Max(maximum, difference)
		if math.IsNaN(difference) || difference > maxDifference {
			valid = false
		}
	}
	return maximum, valid
}

func decodeStrict(raw []byte, target any) error {
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func sha256Hex(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}

func fail(report *auditReport, message string) {
	report.FailureCount++
	if len(report.Failures) < 100 {
		report.Failures = append(report.Failures, message)
	} else {
		report.FailuresTruncated = true
	}
}

func floatPtr(value float64) *float64 { return &value }

func benchmarkHotPredict(model *decision.Model) hotBenchmarkSummary {
	text := "Read the ordered pair of values and compute their sum."
	result := testing.Benchmark(func(b *testing.B) {
		var workspace decision.Workspace
		var prediction decision.Prediction
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := model.PredictInto(text, &workspace, &prediction); err != nil {
				b.Fatal(err)
			}
		}
	})
	return hotBenchmarkSummary{Iterations: result.N, NanosecondsOp: result.NsPerOp(), BytesOp: result.AllocedBytesPerOp(), AllocsOp: result.AllocsPerOp()}
}

func runColdCLI(variant, modelPath, binaryPath, outputDir string, report *auditReport) coldCLISummary {
	summary := coldCLISummary{Planned: 3, Runs: make([]coldCLIRun, 0, 3), ProcessRSSConvention: processRSSConvention()}
	binaryInfo, err := os.Lstat(binaryPath)
	if err != nil || !binaryInfo.Mode().IsRegular() || binaryInfo.Mode()&os.ModeSymlink != 0 {
		fail(report, fmt.Sprintf("%s cold CLI binary is missing or not a regular file", variant))
		return summary
	}
	binaryRaw, err := os.ReadFile(binaryPath)
	if err != nil {
		fail(report, fmt.Sprintf("%s cold CLI binary could not be read: %v", variant, err))
		return summary
	}
	summary.BinarySHA256 = sha256Hex(binaryRaw)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fail(report, fmt.Sprintf("%s cold CLI capture directory: %v", variant, err))
		return summary
	}
	requestRaw := []byte(`{"schema":"gooo/tiny-ir-decision-request/v1","text":"Multiply the quantity by the unit price.","left":{"name":"quantity","type":"Int"},"right":{"name":"unit_price","type":"Int"}}` + "\n")
	for index := 1; index <= summary.Planned; index++ {
		id := fmt.Sprintf("run-%02d", index)
		start := time.Now()
		command := exec.Command(binaryPath, "--model", modelPath)
		command.Stdin = bytes.NewReader(requestRaw)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		result := coldCLIRun{RunID: id, ExitCode: -1, WallNanoseconds: time.Since(start).Nanoseconds(),
			RequestSHA256: sha256Hex(requestRaw), StdoutSHA256: sha256Hex(stdout.Bytes()), StderrSHA256: sha256Hex(stderr.Bytes())}
		result.RequestPath = filepath.Join(outputDir, id+"-request.json")
		result.StdoutPath = filepath.Join(outputDir, id+"-stdout.json")
		result.StderrPath = filepath.Join(outputDir, id+"-stderr.txt")
		if writeErr := os.WriteFile(result.RequestPath, requestRaw, 0o600); writeErr != nil {
			fail(report, fmt.Sprintf("%s %s could not preserve request: %v", variant, id, writeErr))
		}
		if writeErr := os.WriteFile(result.StdoutPath, stdout.Bytes(), 0o600); writeErr != nil {
			fail(report, fmt.Sprintf("%s %s could not preserve stdout: %v", variant, id, writeErr))
		}
		if writeErr := os.WriteFile(result.StderrPath, stderr.Bytes(), 0o600); writeErr != nil {
			fail(report, fmt.Sprintf("%s %s could not preserve stderr: %v", variant, id, writeErr))
		}
		if command.ProcessState != nil {
			result.ExitCode = command.ProcessState.ExitCode()
			result.PeakRSSBytes, _ = processPeakRSS(command.ProcessState)
		}
		var response decision.DecisionResponse
		if decodeErr := decodeStrict(stdout.Bytes(), &response); decodeErr == nil {
			result.Status = response.Status
		}
		summary.Runs = append(summary.Runs, result)
		if err != nil || result.ExitCode != 0 || (result.Status != "decision" && result.Status != "abstained") {
			fail(report, fmt.Sprintf("%s %s cold CLI failed: exit=%d status=%q err=%v", variant, id, result.ExitCode, result.Status, err))
			continue
		}
		summary.Observed++
	}
	return summary
}
