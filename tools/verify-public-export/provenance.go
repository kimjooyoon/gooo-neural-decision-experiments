package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	manifestSchemaV1         = "gooo/synthetic-operation-choice-manifest/v1"
	datasetSchemaV1          = "gooo/synthetic-operation-choice-dataset/v1"
	preexecutionSchema       = "gooo/tiny-ir-decision-training-preexecution/v1"
	trainingReportSchema     = "gooo/tiny-ir-decision-public-training-summary/v1"
	cardSchema               = "gooo/tiny-ir-decision-model-card/v1"
	goAuditSchema            = "gooo/tiny-ir-decision-go-audit/v1"
	pythonParitySchema       = "gooo/tiny-ir-decision-parity/v1"
	frozenDatasetSHA256      = "af0a637320a8256cd6ebcdb3486a6885f2766441c90f0cb154c11599f3c8ca3d"
	frozenManifestSHA256     = "90d745c1e2e2db636ddbb53df5ebf4c268916b802468905a21cc2219cd4a2247"
	frozenContractSHA256     = "e1926dc964c6fd0fa671a68861190afe4ec059ec5d4e4e66f877ba119ef8e6e2"
	frozenGeneratorSHA256    = "f9aeaa86cd17b017fbeaaf535913c1a1c442bf0f2a51ef4b1a027a18819689e5"
	frozenTrainingSHA256     = "d3725742c6bc1252ed738f769d13602327e7b8edb533e969edcd6b9ebe0ea142"
	frozenPreexecutionSHA256 = "2ae73ac67e396c47d931e7c9267933d4d79ea4f179833d06d7485294c1971b9f"
	frozenGoAuditSHA256      = "f28434deee5356610848c5d5906411abed7d388ee28828c82b39a8bf90dadc9d"
)

var (
	operationLabels = []string{"add", "subtract", "multiply", "less_than", "less_equal", "equal", "and", "or"}
	variantNames    = []string{"fp32", "ptq_ternary", "qat_ternary"}
	rolePaths       = map[string]string{
		"documentation":         "README.md",
		"dataset_manifest":      "data/synthetic-ops-v1/manifest.json",
		"synthetic_dataset":     "data/synthetic-ops-v1/dataset.jsonl",
		"model_contract":        "model-contract.json",
		"generator_source":      "cmd/dataset/main.go",
		"training_source":       "runs/pilot-mps-20260930-v1/training-source.py",
		"training_record":       "runs/pilot-mps-20260930-v1/preexecution.json",
		"training_report":       "runs/pilot-mps-20260930-v1/public-training-summary.json",
		"model_card":            "model-card.json",
		"external_verification": "runs/pilot-mps-20260930-v1/go-audit.json",
		"python_parity":         "runs/pilot-mps-20260930-v1/go-parity.json",
	}
)

type provenanceResult struct {
	DatasetSHA256               string         `json:"dataset_sha256"`
	ManifestSHA256              string         `json:"manifest_sha256"`
	ModelContractSHA256         string         `json:"model_contract_sha256"`
	GeneratorSourceSHA256       string         `json:"generator_source_sha256"`
	TrainingSourceSHA256        string         `json:"training_source_sha256"`
	PreexecutionSHA256          string         `json:"preexecution_sha256"`
	PublicTrainingSummarySHA256 string         `json:"public_training_summary_sha256"`
	ExternalVerificationSHA256  string         `json:"external_verification_sha256"`
	SplitRows                   map[string]int `json:"split_rows"`
	ParityRowsPerVariant        int            `json:"parity_rows_per_variant"`
	TestRowsPerVariant          int            `json:"test_rows_per_variant"`
	ExternalParityStatus        string         `json:"external_parity_status"`
}

type datasetRow struct {
	ID              string `json:"id"`
	TemplateID      string `json:"template_id"`
	ConfigurationID string `json:"configuration_id"`
	Language        string `json:"language"`
	Split           string `json:"split"`
	Text            string `json:"text"`
	Label           string `json:"label"`
}

type splitCounts struct {
	Total       int `json:"total"`
	Train       int `json:"train"`
	Calibration int `json:"calibration"`
	Test        int `json:"test"`
}

type datasetManifest struct {
	Schema                string                 `json:"schema"`
	DatasetSchema         string                 `json:"dataset_schema"`
	ModelContractSHA256   string                 `json:"model_contract_sha256"`
	GeneratorSourceSHA256 string                 `json:"generator_source_sha256"`
	DatasetSHA256         string                 `json:"dataset_sha256"`
	SplitSHA256           map[string]string      `json:"split_sha256"`
	Rows                  splitCounts            `json:"rows"`
	RowsByLabel           map[string]splitCounts `json:"rows_by_label"`
	RowsByLanguage        map[string]splitCounts `json:"rows_by_language"`
	OperationLabels       []string               `json:"operation_labels"`
	NoGoldLabelCheck      bool                   `json:"no_gold_label_in_prompt_check"`
	SyntheticProvenance   struct {
		Kind           string `json:"kind"`
		SourceMaterial string `json:"source_material"`
		SensitiveData  string `json:"sensitive_data"`
		IntendedUse    string `json:"intended_use"`
	} `json:"synthetic_provenance"`
}

type preexecutionRecord struct {
	BatchSize            int            `json:"batch_size"`
	DataProvenance       string         `json:"data_provenance"`
	Schema               string         `json:"schema"`
	Epochs               int            `json:"epochs"`
	DatasetSHA256        string         `json:"dataset_sha256"`
	TrainingSourceSHA256 string         `json:"training_source_sha256"`
	SplitCounts          map[string]int `json:"split_counts"`
	TestUsed             bool           `json:"test_used_for_training_or_calibration"`
	Variants             []string       `json:"variants"`
	Seed                 int64          `json:"seed"`
	GPUDeterminism       string         `json:"gpu_determinism"`
	NumpyVersion         string         `json:"numpy_version"`
	TorchVersion         string         `json:"torch_version"`
	Hardware             struct {
		Chip             string `json:"chip"`
		GPUCores         int    `json:"gpu_cores"`
		UnifiedMemoryGiB int    `json:"unified_memory_gib"`
	} `json:"hardware"`
}

type publicTrainingSummary struct {
	Schema               string                    `json:"schema"`
	PreexecutionSHA256   string                    `json:"preexecution_sha256"`
	DatasetSHA256        string                    `json:"dataset_sha256"`
	TrainingSourceSHA256 string                    `json:"training_source_sha256"`
	Variants             map[string]summaryVariant `json:"variants"`
}

type summaryVariant struct {
	TestCorrect  int     `json:"test_correct"`
	TestPlanned  int     `json:"test_planned"`
	TestAccuracy float64 `json:"test_accuracy"`
}

type publicModelCard struct {
	Schema                     string `json:"schema"`
	ModelOrigin                string `json:"model_origin"`
	LayaFinetuned              *bool  `json:"laya_finetuned"`
	TaskScope                  string `json:"task_scope"`
	ArbitrarySourceGeneration  *bool  `json:"arbitrary_source_generation"`
	ExternalGoParityStatus     string `json:"external_go_parity_status"`
	ExternalVerificationSHA256 string `json:"external_verification_sha256"`
}

type goAudit struct {
	Schema            string                   `json:"schema"`
	Decision          string                   `json:"decision"`
	Status            string                   `json:"status"`
	Tolerance         float64                  `json:"absolute_tolerance"`
	DatasetSHA256     string                   `json:"dataset_sha256"`
	ParitySHA256      string                   `json:"parity_sha256"`
	Inputs            map[string]string        `json:"input_sha256"`
	Parity            map[string]parityVariant `json:"parity"`
	TestScores        map[string]auditScore    `json:"test_scores"`
	FailureCount      int                      `json:"failure_count"`
	FailuresTruncated bool                     `json:"failures_truncated"`
}

type parityVariant struct {
	Planned                int     `json:"planned"`
	Observed               int     `json:"observed"`
	Matched                int     `json:"matched"`
	FeatureMaxAbsError     float64 `json:"features_max_abs_error"`
	LogitMaxAbsError       float64 `json:"logits_max_abs_error"`
	ProbabilityMaxAbsError float64 `json:"probabilities_max_abs_error"`
	SelectedLabelMatches   int     `json:"selected_label_matches"`
}

type auditScore struct {
	Model struct {
		ModelJSONSHA256 string `json:"model_json_sha256"`
		WeightsSHA256   string `json:"weights_sha256"`
	} `json:"model"`
	Overall struct {
		Planned  int     `json:"planned"`
		Observed int     `json:"observed"`
		Correct  int     `json:"correct"`
		Accuracy float64 `json:"accuracy"`
	} `json:"overall"`
}

type parityFile struct {
	Schema string         `json:"schema"`
	Rows   []paritySample `json:"rows"`
}

type paritySample struct {
	Variant string `json:"variant"`
	Text    string `json:"text"`
}

func verifyTrainingProvenance(bundle string, entries map[string]allowedArtifact, models []modelResult) (provenanceResult, error) {
	if err := requireRolePaths(entries); err != nil {
		return provenanceResult{}, err
	}
	files, err := readProvenanceFiles(bundle, rolePaths)
	if err != nil {
		return provenanceResult{}, err
	}
	manifest, err := decodeManifest(files["dataset_manifest"])
	if err != nil {
		return provenanceResult{}, err
	}
	counts, datasetDigest, splitDigests, err := inspectDataset(files["synthetic_dataset"])
	if err != nil {
		return provenanceResult{}, err
	}
	contractDigest := digestHex(files["model_contract"])
	generatorDigest := digestHex(files["generator_source"])
	trainingDigest := digestHex(files["training_source"])
	if trainingDigest != frozenTrainingSHA256 {
		return provenanceResult{}, errors.New("training source differs from the independently reviewed v1 bytes")
	}
	if digestHex(files["dataset_manifest"]) != frozenManifestSHA256 {
		return provenanceResult{}, errors.New("dataset manifest differs from the independently reviewed v1 bytes")
	}
	if err := checkManifest(manifest, datasetDigest, splitDigests, counts, contractDigest, generatorDigest); err != nil {
		return provenanceResult{}, err
	}
	preexecution, err := decodePreexecution(files["training_record"])
	if err != nil {
		return provenanceResult{}, err
	}
	preexecutionDigest := digestHex(files["training_record"])
	if preexecutionDigest != frozenPreexecutionSHA256 {
		return provenanceResult{}, errors.New("pre-execution record differs from the independently reviewed v1 bytes")
	}
	if err := checkPreexecution(preexecution, datasetDigest, trainingDigest, counts); err != nil {
		return provenanceResult{}, err
	}
	summary, err := decodePublicSummary(files["training_report"])
	if err != nil {
		return provenanceResult{}, err
	}
	if err := checkSummary(summary, preexecutionDigest, datasetDigest, trainingDigest); err != nil {
		return provenanceResult{}, err
	}
	card, err := decodeModelCard(files["model_card"])
	if err != nil {
		return provenanceResult{}, err
	}
	auditDigest := digestHex(files["external_verification"])
	if auditDigest != frozenGoAuditSHA256 {
		return provenanceResult{}, errors.New("external Go audit differs from the independently reviewed v1 bytes")
	}
	if err := checkModelCard(card, auditDigest); err != nil {
		return provenanceResult{}, err
	}
	parityDigest := digestHex(files["python_parity"])
	if err := checkParityFile(files["python_parity"]); err != nil {
		return provenanceResult{}, err
	}
	audit, err := decodeGoAudit(files["external_verification"])
	if err != nil {
		return provenanceResult{}, err
	}
	if err := checkGoAudit(audit, entries, models, datasetDigest, parityDigest, summary); err != nil {
		return provenanceResult{}, err
	}
	return provenanceResult{
		DatasetSHA256: datasetDigest, ManifestSHA256: digestHex(files["dataset_manifest"]),
		ModelContractSHA256: contractDigest, GeneratorSourceSHA256: generatorDigest,
		TrainingSourceSHA256: trainingDigest, PreexecutionSHA256: preexecutionDigest,
		PublicTrainingSummarySHA256: digestHex(files["training_report"]),
		ExternalVerificationSHA256:  auditDigest, SplitRows: map[string]int{
			"train": counts.Train, "calibration": counts.Calibration, "test": counts.Test,
		}, ParityRowsPerVariant: 32, TestRowsPerVariant: 256, ExternalParityStatus: "PASS",
	}, nil
}

func requireRolePaths(entries map[string]allowedArtifact) error {
	if len(entries) != len(rolePaths)+6 {
		return errors.New("allowlist must contain exactly the 17 reviewed v1 bundle files")
	}
	allowedRunPaths := make(map[string]bool)
	for role, expectedPath := range rolePaths {
		allowedRunPaths[expectedPath] = true
		found := 0
		for name, entry := range entries {
			if entry.Role != role {
				continue
			}
			found++
			if name != expectedPath {
				return fmt.Errorf("role %q must use the frozen v1 path %q", role, expectedPath)
			}
		}
		if found != 1 {
			return fmt.Errorf("allowlist must contain exactly one %q artifact", role)
		}
	}
	for _, modelDir := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		base := "runs/pilot-mps-20260930-v1/models/" + modelDir + "/"
		allowedRunPaths[base+"model.json"] = true
		allowedRunPaths[base+"weights.bin"] = true
	}
	for name := range entries {
		if strings.HasPrefix(name, "runs/") && !allowedRunPaths[name] {
			return fmt.Errorf("run artifact is outside the reviewed public v1 bundle: %q", name)
		}
	}
	return nil
}

func readProvenanceFiles(bundle string, paths map[string]string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(paths))
	for role, name := range paths {
		raw, err := os.ReadFile(filepath.Join(bundle, filepath.FromSlash(name)))
		if err != nil {
			return nil, fmt.Errorf("could not read %q provenance artifact", role)
		}
		result[role] = raw
	}
	return result, nil
}

func decodeManifest(raw []byte) (datasetManifest, error) {
	var value datasetManifest
	if err := decodeJSON(raw, &value, false); err != nil {
		return value, fmt.Errorf("dataset manifest: %w", err)
	}
	return value, nil
}

func decodePreexecution(raw []byte) (preexecutionRecord, error) {
	var value preexecutionRecord
	if err := decodeJSON(raw, &value, true); err != nil {
		return value, fmt.Errorf("training record: %w", err)
	}
	return value, nil
}

func decodePublicSummary(raw []byte) (publicTrainingSummary, error) {
	var value publicTrainingSummary
	if err := decodeJSON(raw, &value, true); err != nil {
		return value, fmt.Errorf("public training summary: %w", err)
	}
	return value, nil
}

func decodeModelCard(raw []byte) (publicModelCard, error) {
	var value publicModelCard
	if err := decodeJSON(raw, &value, false); err != nil {
		return value, fmt.Errorf("model card: %w", err)
	}
	return value, nil
}

func decodeGoAudit(raw []byte) (goAudit, error) {
	var value goAudit
	if err := decodeJSON(raw, &value, false); err != nil {
		return value, fmt.Errorf("external Go verification: %w", err)
	}
	return value, nil
}

func decodeJSON(raw []byte, target any, strict bool) error {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func inspectDataset(raw []byte) (splitCounts, string, map[string]string, error) {
	var counts splitCounts
	if len(raw) == 0 || !utf8.Valid(raw) || raw[len(raw)-1] != '\n' {
		return counts, "", nil, errors.New("dataset must be nonempty UTF-8 JSONL ending in a newline")
	}
	allIDs, allTexts := make(map[string]bool), make(map[string]bool)
	templateSplits := make(map[string]string)
	labelCounts, languageCounts := freshCounts(operationLabels), freshCounts([]string{"en", "ko"})
	splitHashes := map[string]*hashBuffer{"train": {}, "calibration": {}, "test": {}}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 1024), 1<<20)
	rowCount := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			return counts, "", nil, errors.New("dataset contains a blank line")
		}
		var row datasetRow
		if err := decodeJSON(line, &row, true); err != nil {
			return counts, "", nil, fmt.Errorf("dataset row %d: %w", rowCount+1, err)
		}
		if err := validateDatasetRow(row); err != nil {
			return counts, "", nil, fmt.Errorf("dataset row %d: %w", rowCount+1, err)
		}
		if allIDs[row.ID] || allTexts[row.Text] {
			return counts, "", nil, errors.New("dataset contains a duplicate row ID or prompt")
		}
		allIDs[row.ID], allTexts[row.Text] = true, true
		if previous, ok := templateSplits[row.TemplateID]; ok && previous != row.Split {
			return counts, "", nil, errors.New("a prompt template crosses dataset splits")
		}
		templateSplits[row.TemplateID] = row.Split
		if err := addSplitCount(&counts, row.Split); err != nil {
			return counts, "", nil, err
		}
		labelCounts[row.Label][row.Split]++
		languageCounts[row.Language][row.Split]++
		splitHashes[row.Split].Write(line)
		splitHashes[row.Split].Write([]byte{'\n'})
		rowCount++
	}
	if err := scanner.Err(); err != nil {
		return counts, "", nil, errors.New("dataset JSONL exceeds the row size limit")
	}
	if err := checkFrozenCounts(counts, labelCounts, languageCounts, rowCount); err != nil {
		return counts, "", nil, err
	}
	splitDigests := make(map[string]string, len(splitHashes))
	for split, hash := range splitHashes {
		splitDigests[split] = hash.SumHex()
	}
	return counts, digestHex(raw), splitDigests, nil
}

func freshCounts(keys []string) map[string]map[string]int {
	result := make(map[string]map[string]int, len(keys))
	for _, key := range keys {
		result[key] = map[string]int{"train": 0, "calibration": 0, "test": 0}
	}
	return result
}

func validateDatasetRow(row datasetRow) error {
	if row.ID == "" || row.TemplateID == "" || row.ConfigurationID == "" {
		return errors.New("row identity fields are required")
	}
	if row.Language != "en" && row.Language != "ko" {
		return errors.New("row language is unsupported")
	}
	if row.Split != "train" && row.Split != "calibration" && row.Split != "test" {
		return errors.New("row split is unsupported")
	}
	if !contains(operationLabels, row.Label) {
		return errors.New("row label is unsupported")
	}
	if !utf8.ValidString(row.Text) || len([]byte(row.Text)) == 0 || len([]byte(row.Text)) > 512 {
		return errors.New("row text is empty or outside the UTF-8 byte limit")
	}
	return nil
}

func addSplitCount(counts *splitCounts, split string) error {
	counts.Total++
	switch split {
	case "train":
		counts.Train++
	case "calibration":
		counts.Calibration++
	case "test":
		counts.Test++
	default:
		return errors.New("unknown dataset split")
	}
	return nil
}

type hashBuffer struct{ bytes.Buffer }

func (h hashBuffer) SumHex() string {
	digest := sha256.Sum256(h.Bytes())
	return hex.EncodeToString(digest[:])
}

func checkFrozenCounts(total splitCounts, labels, languages map[string]map[string]int, rowCount int) error {
	if rowCount != 2048 || total != (splitCounts{Total: 2048, Train: 1536, Calibration: 256, Test: 256}) {
		return errors.New("dataset split counts do not match the frozen v1 plan")
	}
	for _, counts := range labels {
		if counts["train"] != 192 || counts["calibration"] != 32 || counts["test"] != 32 {
			return errors.New("dataset label counts do not match the frozen v1 plan")
		}
	}
	for _, counts := range languages {
		if counts["train"] != 768 || counts["calibration"] != 128 || counts["test"] != 128 {
			return errors.New("dataset language counts do not match the frozen v1 plan")
		}
	}
	return nil
}

func checkManifest(manifest datasetManifest, datasetDigest string, splitDigests map[string]string, counts splitCounts, contractDigest, generatorDigest string) error {
	if manifest.Schema != manifestSchemaV1 || manifest.DatasetSchema != datasetSchemaV1 {
		return errors.New("dataset manifest schema is not the frozen v1 schema")
	}
	if datasetDigest != frozenDatasetSHA256 || contractDigest != frozenContractSHA256 || generatorDigest != frozenGeneratorSHA256 ||
		manifest.DatasetSHA256 != datasetDigest || manifest.ModelContractSHA256 != contractDigest || manifest.GeneratorSourceSHA256 != generatorDigest {
		return errors.New("dataset manifest digest bindings do not match included source/data/contract")
	}
	if !sameStringMap(manifest.SplitSHA256, splitDigests) || manifest.Rows != counts || !sameStrings(manifest.OperationLabels, operationLabels) {
		return errors.New("dataset manifest split/count/label declarations differ from recomputed rows")
	}
	if !sameCountsMap(manifest.RowsByLabel, []string{"add", "subtract", "multiply", "less_than", "less_equal", "equal", "and", "or"}, 192, 32, 32) ||
		!sameCountsMap(manifest.RowsByLanguage, []string{"en", "ko"}, 768, 128, 128) {
		return errors.New("dataset manifest class or language counts do not match the frozen v1 plan")
	}
	if !manifest.NoGoldLabelCheck || !strings.Contains(manifest.SyntheticProvenance.SourceMaterial, "No repository source") ||
		manifest.SyntheticProvenance.SourceMaterial != "No repository source, user content, model output, external dataset, or private text is used." ||
		!strings.Contains(manifest.SyntheticProvenance.IntendedUse, "not arbitrary source-code generation") {
		return errors.New("dataset manifest does not declare the reviewed synthetic-only scope")
	}
	return nil
}

func checkPreexecution(record preexecutionRecord, datasetDigest, sourceDigest string, counts splitCounts) error {
	if record.Schema != preexecutionSchema || record.DatasetSHA256 != datasetDigest || record.TrainingSourceSHA256 != sourceDigest || record.TestUsed ||
		datasetDigest != frozenDatasetSHA256 || sourceDigest != frozenTrainingSHA256 {
		return errors.New("pre-execution record does not bind the dataset/source or claims test use")
	}
	if !sameStrings(record.Variants, variantNames) || record.SplitCounts["train"] != counts.Train ||
		record.SplitCounts["calibration"] != counts.Calibration || record.SplitCounts["test"] != counts.Test || len(record.SplitCounts) != 3 {
		return errors.New("pre-execution variant or split declarations do not match the frozen v1 plan")
	}
	if record.BatchSize != 256 || record.Epochs != 120 || record.Seed != 20360930 || record.Hardware.Chip != "Apple M4" ||
		record.Hardware.GPUCores != 10 || record.Hardware.UnifiedMemoryGiB != 16 {
		return errors.New("pre-execution record does not match the reviewed v1 pilot configuration")
	}
	return nil
}

func checkSummary(summary publicTrainingSummary, preexecutionDigest, datasetDigest, sourceDigest string) error {
	if summary.Schema != trainingReportSchema || summary.PreexecutionSHA256 != preexecutionDigest || summary.DatasetSHA256 != datasetDigest || summary.TrainingSourceSHA256 != sourceDigest {
		return errors.New("public training summary does not bind to the frozen pre-execution record")
	}
	if len(summary.Variants) != len(variantNames) {
		return errors.New("public training summary must contain all three model variants")
	}
	for _, name := range variantNames {
		value, ok := summary.Variants[name]
		if !ok || value.TestPlanned != 256 || value.TestCorrect < 0 || value.TestCorrect > value.TestPlanned ||
			!finite(value.TestAccuracy) || math.Abs(value.TestAccuracy-float64(value.TestCorrect)/float64(value.TestPlanned)) > 1e-12 {
			return fmt.Errorf("public training summary has invalid test score for %q", name)
		}
	}
	return nil
}

func checkModelCard(card publicModelCard, auditDigest string) error {
	if card.Schema != cardSchema || card.ModelOrigin != "independently_initialized" || card.LayaFinetuned == nil || *card.LayaFinetuned ||
		card.TaskScope != "closed_eight_operation_typed_binary_ir_selection" || card.ArbitrarySourceGeneration == nil || *card.ArbitrarySourceGeneration ||
		card.ExternalGoParityStatus != "PASS" || card.ExternalVerificationSHA256 != auditDigest {
		return errors.New("model card scope/origin or external Go verification binding is invalid")
	}
	return nil
}

func checkParityFile(raw []byte) error {
	var value parityFile
	if err := decodeJSON(raw, &value, false); err != nil {
		return fmt.Errorf("Python parity sample: %w", err)
	}
	if value.Schema != pythonParitySchema || len(value.Rows) != 96 {
		return errors.New("Python parity sample must use the frozen schema and contain 96 rows")
	}
	counts := make(map[string]int, len(variantNames))
	for _, row := range value.Rows {
		if row.Text == "" || !contains(variantNames, row.Variant) {
			return errors.New("Python parity sample contains an invalid variant or empty prompt")
		}
		counts[row.Variant]++
	}
	for _, name := range variantNames {
		if counts[name] != 32 {
			return fmt.Errorf("Python parity sample has %d rows for %q, expected 32", counts[name], name)
		}
	}
	return nil
}

func checkGoAudit(audit goAudit, entries map[string]allowedArtifact, models []modelResult, datasetDigest, parityDigest string, summary publicTrainingSummary) error {
	if audit.Schema != goAuditSchema || audit.Decision != "PASS" || audit.Status != "PASS" ||
		audit.DatasetSHA256 != datasetDigest || audit.ParitySHA256 != parityDigest || !finite(audit.Tolerance) || audit.Tolerance <= 0 ||
		audit.FailureCount != 0 || audit.FailuresTruncated || len(audit.Parity) != len(variantNames) || len(audit.TestScores) != len(variantNames) {
		return errors.New("external Go audit is incomplete or not PASS")
	}
	expectedInputs := map[string]string{
		"data/synthetic-ops-v1/dataset.jsonl":       datasetDigest,
		"runs/pilot-mps-20260930-v1/go-parity.json": parityDigest,
	}
	modelsByVariant := make(map[string]modelResult, len(models))
	for _, model := range models {
		modelsByVariant[model.Variant] = model
		weightPath := strings.TrimSuffix(model.Path, "model.json") + "weights.bin"
		expectedInputs[model.Path] = entries[model.Path].SHA256
		expectedInputs[weightPath] = entries[weightPath].SHA256
	}
	if !sameStringMap(audit.Inputs, expectedInputs) {
		return errors.New("external Go audit input hashes do not bind exactly to the bundled dataset/parity/models")
	}
	for _, name := range variantNames {
		parity, ok := audit.Parity[name]
		if !ok || parity.Planned != 32 || parity.Observed != 32 || parity.Matched != 32 || parity.SelectedLabelMatches != 32 ||
			!finite(parity.FeatureMaxAbsError) || !finite(parity.LogitMaxAbsError) || !finite(parity.ProbabilityMaxAbsError) ||
			parity.FeatureMaxAbsError > audit.Tolerance || parity.LogitMaxAbsError > audit.Tolerance || parity.ProbabilityMaxAbsError > audit.Tolerance {
			return fmt.Errorf("external Go parity does not pass all 32 rows for %q", name)
		}
		model, ok := modelsByVariant[name]
		if !ok {
			return fmt.Errorf("bundled model %q is missing", name)
		}
		score := audit.TestScores[name]
		publicScore := summary.Variants[name]
		if score.Model.ModelJSONSHA256 != entries[model.Path].SHA256 || score.Model.WeightsSHA256 != model.WeightsSHA256 ||
			score.Overall.Planned != 256 || score.Overall.Observed != 256 ||
			score.Overall.Correct != publicScore.TestCorrect || math.Abs(score.Overall.Accuracy-publicScore.TestAccuracy) > 1e-12 {
			return fmt.Errorf("external Go test score or model binding differs for %q", name)
		}
	}
	return nil
}

func sameCountsMap(actual map[string]splitCounts, names []string, train, calibration, test int) bool {
	if len(actual) != len(names) {
		return false
	}
	for _, name := range names {
		value, ok := actual[name]
		if !ok || value != (splitCounts{Total: train + calibration + test, Train: train, Calibration: calibration, Test: test}) {
			return false
		}
	}
	return true
}

func sameStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range right {
		if left[key] != value {
			return false
		}
	}
	return true
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func digestHex(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
