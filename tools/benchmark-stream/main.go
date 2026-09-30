// Command benchmark-stream measures the persistent decision stream process
// against the frozen synthetic test split. It never builds or changes the
// stream executable.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decisionstream"
)

const (
	benchmarkSchema    = "gooo/stream-end-to-end-benchmark/v1"
	manifestSchema     = "gooo/synthetic-operation-choice-manifest/v1"
	datasetSchema      = "gooo/synthetic-operation-choice-dataset/v1"
	rowsPerRun         = 256
	repeatsPerRun      = 16
	plannedPerRun      = rowsPerRun * repeatsPerRun
	perRunTimeout      = 12 * time.Second
	totalRunTimeout    = 30 * time.Second
	maxDatasetLineSize = 64 * 1024
	maxOutputLineSize  = 64 * 1024
)

var operationLabels = []string{"add", "subtract", "multiply", "less_than", "less_equal", "equal", "and", "or"}

var sourceClosurePaths = []string{
	"go.mod",
	"cmd/gooo-decision-stream/main.go",
	"internal/decisionstream/stream.go",
	"internal/decision/model.go",
	"internal/decision/bridge.go",
	"internal/decision/ir.go",
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

type rowCounts struct {
	Total int `json:"total"`
	Test  int `json:"test"`
}

type datasetManifest struct {
	Schema        string            `json:"schema"`
	DatasetSchema string            `json:"dataset_schema"`
	DatasetSHA    string            `json:"dataset_sha256"`
	SplitSHA      map[string]string `json:"split_sha256"`
	Rows          rowCounts         `json:"rows"`
}

type modelMetadata struct {
	Schema      string `json:"schema"`
	Variant     string `json:"variant"`
	WeightsFile string `json:"weights_file"`
	WeightsSHA  string `json:"weights_sha256"`
}

type modelPin struct {
	Variant               string `json:"variant"`
	MetadataSHA256        string `json:"metadata_sha256"`
	WeightsSHA256         string `json:"weights_sha256"`
	DeclaredWeightsSHA256 string `json:"declared_weights_sha256"`
	BundleSHA256          string `json:"bundle_sha256"`
	FileBytes             int64  `json:"file_bytes"`
}

type processResources struct {
	UserCPUSeconds   *float64 `json:"user_cpu_seconds,omitempty"`
	SystemCPUSeconds *float64 `json:"system_cpu_seconds,omitempty"`
	AverageCorePct   *float64 `json:"cpu_average_core_percent,omitempty"`
	PeakRSSBytes     *int64   `json:"peak_rss_bytes,omitempty"`
	RSSAvailable     bool     `json:"rss_available"`
}

type runReport struct {
	Variant                 string           `json:"variant"`
	Workers                 int              `json:"workers"`
	ModelBundleSHA256       string           `json:"model_bundle_sha256"`
	PlannedRecords          int              `json:"planned_records"`
	WrittenRecords          int              `json:"written_records"`
	ObservedOutputRecords   int              `json:"observed_output_records"`
	UniqueSequences         int              `json:"unique_sequences"`
	CorrelatedRecords       int              `json:"correlated_records"`
	MissingRecords          int              `json:"missing_records"`
	DuplicateSequences      int              `json:"duplicate_sequences"`
	OutOfRangeSequences     int              `json:"out_of_range_sequences"`
	CorrelationMismatches   int              `json:"correlation_mismatches"`
	MalformedOutputRecords  int              `json:"malformed_output_records"`
	UnterminatedOutputLines int              `json:"unterminated_output_lines"`
	RejectedResults         int              `json:"rejected_results"`
	ErrorRecords            int              `json:"error_records"`
	ProcessFailed           bool             `json:"process_failed"`
	BestLabelObserved       int              `json:"best_label_observed"`
	BestLabelCorrect        int              `json:"matched_best_label_correct"`
	AcceptedIR              int              `json:"accepted_ir"`
	AcceptedCorrect         int              `json:"accepted_correct"`
	AcceptedIncorrect       int              `json:"accepted_incorrect"`
	TypeAbstentions         int              `json:"type_abstentions"`
	OtherAbstentions        int              `json:"other_abstentions"`
	WallMilliseconds        float64          `json:"wall_milliseconds"`
	ProcessResources        processResources `json:"process_resources"`
	ResourcePlatform        string           `json:"resource_platform"`
	ResourceSource          string           `json:"resource_source"`
	ExitCode                *int             `json:"exit_code,omitempty"`
	TimedOut                bool             `json:"timed_out"`
	Cancelled               bool             `json:"cancelled"`
	InputWriteFailed        bool             `json:"input_write_failed"`
	ProcessStarted          bool             `json:"process_started"`
	Complete                bool             `json:"complete"`
}

type benchmarkReport struct {
	Schema                    string         `json:"schema"`
	Decision                  string         `json:"decision"`
	PlannedRuns               int            `json:"planned_runs"`
	CompletedRuns             int            `json:"completed_runs"`
	UnstartedRuns             int            `json:"unstarted_runs"`
	PlannedRecordsTotal       int            `json:"planned_records_total"`
	WrittenRecordsTotal       int            `json:"written_records_total"`
	ObservedOutputTotal       int            `json:"observed_output_records_total"`
	CorrelatedRecordsTotal    int            `json:"correlated_records_total"`
	UnvalidatedRecordsTotal   int            `json:"unvalidated_records_total"`
	RowsPerRun                int            `json:"unique_test_rows_per_run"`
	RepeatsPerRow             int            `json:"repeats_per_test_row"`
	PlannedRecordsPerRun      int            `json:"planned_records_per_run"`
	Workers                   []int          `json:"worker_counts"`
	Variants                  []modelPin     `json:"models"`
	BinarySHA256              string         `json:"stream_binary_sha256"`
	StreamSourceSHA256        string         `json:"stream_source_closure_sha256"`
	BenchmarkDriverSHA256     string         `json:"benchmark_driver_source_sha256"`
	SourcePinChecked          bool           `json:"source_pin_checked"`
	SourcePinMatched          bool           `json:"source_pin_matched"`
	SourceStable              bool           `json:"source_stable_during_run"`
	DriverStable              bool           `json:"benchmark_driver_stable_during_run"`
	DatasetManifestSHA256     string         `json:"dataset_manifest_sha256"`
	DatasetSHA256             string         `json:"dataset_sha256"`
	TestSplitSHA256           string         `json:"test_split_sha256"`
	TestRows                  int            `json:"test_rows"`
	TestRowsByLabel           map[string]int `json:"test_rows_by_label"`
	TestRowsByLanguage        map[string]int `json:"test_rows_by_language"`
	GoVersion                 string         `json:"benchmark_driver_go_version"`
	ResourceMethod            string         `json:"resource_method"`
	PeakRSSPlatformUnits      string         `json:"peak_rss_platform_units"`
	AverageCorePercentMeaning string         `json:"cpu_average_core_percent_meaning"`
	HotPathNote               string         `json:"hot_path_vs_end_to_end_note"`
	Runs                      []runReport    `json:"runs"`
}

type operands struct {
	left  string
	right string
	kind  decision.ValueType
}

var operandPatterns = []struct {
	pattern *regexp.Regexp
	kind    decision.ValueType
}{
	{regexp.MustCompile(`^Operands: ([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)\.`), decision.TypeInt},
	{regexp.MustCompile(`^Read the ordered pair \(([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)\)\.`), decision.TypeInt},
	{regexp.MustCompile(`^Use ([A-Za-z_][A-Za-z0-9_]*) as the first input; use ([A-Za-z_][A-Za-z0-9_]*) as the second\.`), decision.TypeInt},
	{regexp.MustCompile(`^Inputs are left=([A-Za-z_][A-Za-z0-9_]*); right=([A-Za-z_][A-Za-z0-9_]*)\.`), decision.TypeInt},
	{regexp.MustCompile(`^Boolean inputs: ([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)\.`), decision.TypeBool},
	{regexp.MustCompile(`^Read the ordered flags \(([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)\)\.`), decision.TypeBool},
	{regexp.MustCompile(`^Use ([A-Za-z_][A-Za-z0-9_]*) as the first flag; use ([A-Za-z_][A-Za-z0-9_]*) as the second\.`), decision.TypeBool},
	{regexp.MustCompile(`^Conditions are first=([A-Za-z_][A-Za-z0-9_]*); next=([A-Za-z_][A-Za-z0-9_]*)\.`), decision.TypeBool},
	{regexp.MustCompile(`^피연산자는 ([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)이다\.`), decision.TypeInt},
	{regexp.MustCompile(`^순서가 있는 입력 \(([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)\)을 읽어라\.`), decision.TypeInt},
	{regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)를 첫 입력으로, ([A-Za-z_][A-Za-z0-9_]*)를 다음 입력으로 사용하라\.`), decision.TypeInt},
	{regexp.MustCompile(`^입력 순서는 첫 값=([A-Za-z_][A-Za-z0-9_]*), 다음 값=([A-Za-z_][A-Za-z0-9_]*)이다\.`), decision.TypeInt},
	{regexp.MustCompile(`^불리언 입력은 ([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)이다\.`), decision.TypeBool},
	{regexp.MustCompile(`^순서가 있는 플래그 \(([A-Za-z_][A-Za-z0-9_]*), ([A-Za-z_][A-Za-z0-9_]*)\)을 읽어라\.`), decision.TypeBool},
	{regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)를 첫 플래그로, ([A-Za-z_][A-Za-z0-9_]*)를 다음 플래그로 사용하라\.`), decision.TypeBool},
	{regexp.MustCompile(`^조건 순서는 첫 값=([A-Za-z_][A-Za-z0-9_]*), 다음 값=([A-Za-z_][A-Za-z0-9_]*)이다\.`), decision.TypeBool},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("benchmark-stream", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	streamBinary := flags.String("stream-bin", "", "path to the prebuilt gooo-decision-stream executable")
	binarySHA := flags.String("binary-sha256", "", "required immutable SHA-256 pin for the executable")
	sourceSHA := flags.String("source-sha256", "", "optional expected SHA-256 of the frozen stream source closure")
	datasetPath := flags.String("dataset", "data/synthetic-ops-v1/dataset.jsonl", "frozen synthetic dataset JSONL")
	manifestPath := flags.String("manifest", "data/synthetic-ops-v1/manifest.json", "frozen dataset manifest")
	modelsDir := flags.String("models-dir", "runs/pilot-mps-20260930-v1/models", "directory containing the three model bundles")
	outputPath := flags.String("output", "", "new JSON report path (existing files are never overwritten)")
	repoRootArg := flags.String("repo-root", ".", "repository root used for source and data hashes")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *streamBinary == "" || *binarySHA == "" || *outputPath == "" {
		fmt.Fprintln(diagnostics, "usage: benchmark-stream --stream-bin FILE --binary-sha256 HEX --output NEW.json [--source-sha256 HEX]")
		return 2
	}
	if _, err := decodeSHA256(*binarySHA); err != nil {
		fmt.Fprintln(diagnostics, "binary SHA-256 must be 64 lowercase hexadecimal characters")
		return 2
	}
	if *sourceSHA != "" {
		if _, err := decodeSHA256(*sourceSHA); err != nil {
			fmt.Fprintln(diagnostics, "source SHA-256 must be 64 lowercase hexadecimal characters")
			return 2
		}
	}
	if _, err := os.Lstat(*outputPath); err == nil {
		fmt.Fprintln(diagnostics, "output already exists; choose a new report path")
		return 2
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(diagnostics, "output path cannot be checked")
		return 2
	}

	root, err := filepath.Abs(*repoRootArg)
	if err != nil {
		fmt.Fprintln(diagnostics, "repository root is unavailable")
		return 1
	}
	manifestFile := resolveFromRoot(root, *manifestPath)
	datasetFile := resolveFromRoot(root, *datasetPath)
	modelsRoot := resolveFromRoot(root, *modelsDir)
	binaryFile, err := filepath.Abs(*streamBinary)
	if err != nil {
		fmt.Fprintln(diagnostics, "stream executable path is unavailable")
		return 1
	}

	manifestRaw, manifestDigest, rows, rowCountsByLabel, rowCountsByLanguage, err := loadFrozenRows(manifestFile, datasetFile)
	if err != nil {
		fmt.Fprintln(diagnostics, "frozen dataset preflight failed")
		return 1
	}
	binaryDigest, err := hashRegularFile(binaryFile)
	if err != nil || binaryDigest != *binarySHA {
		fmt.Fprintln(diagnostics, "stream executable does not match the required SHA-256 pin")
		return 1
	}
	sourceDigest, err := hashSourceClosure(root)
	if err != nil {
		fmt.Fprintln(diagnostics, "stream source closure could not be hashed")
		return 1
	}
	sourcePinChecked := *sourceSHA != ""
	sourcePinMatched := sourcePinChecked && sourceDigest == *sourceSHA
	if sourcePinChecked && !sourcePinMatched {
		fmt.Fprintln(diagnostics, "stream source closure does not match the expected SHA-256 pin")
		return 1
	}
	driverDigest, err := hashRegularFile(filepath.Join(root, "tools", "benchmark-stream", "main.go"))
	if err != nil {
		fmt.Fprintln(diagnostics, "benchmark driver source could not be hashed")
		return 1
	}

	models := make([]modelPin, 0, 3)
	modelPaths := make(map[string]string, 3)
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		pin, modelPath, err := loadModelPin(filepath.Join(modelsRoot, variant), variant)
		if err != nil {
			fmt.Fprintf(diagnostics, "model preflight failed for %s\n", variant)
			return 1
		}
		models = append(models, pin)
		modelPaths[variant] = modelPath
	}

	ctx, cancel := context.WithTimeout(context.Background(), totalRunTimeout)
	defer cancel()
	report := benchmarkReport{
		Schema:                    benchmarkSchema,
		Decision:                  "COMPLETE",
		PlannedRuns:               6,
		PlannedRecordsTotal:       6 * plannedPerRun,
		RowsPerRun:                rowsPerRun,
		RepeatsPerRow:             repeatsPerRun,
		PlannedRecordsPerRun:      plannedPerRun,
		Workers:                   []int{1, 4},
		Variants:                  models,
		BinarySHA256:              binaryDigest,
		StreamSourceSHA256:        sourceDigest,
		BenchmarkDriverSHA256:     driverDigest,
		SourcePinChecked:          sourcePinChecked,
		SourcePinMatched:          sourcePinMatched,
		DatasetManifestSHA256:     manifestDigest,
		DatasetSHA256:             manifestRaw.DatasetSHA,
		TestSplitSHA256:           manifestRaw.SplitSHA["test"],
		TestRows:                  len(rows),
		TestRowsByLabel:           rowCountsByLabel,
		TestRowsByLanguage:        rowCountsByLanguage,
		GoVersion:                 runtime.Version(),
		ResourceMethod:            "child process getrusage via os.ProcessState.SysUsage",
		PeakRSSPlatformUnits:      peakRSSDescription(runtime.GOOS),
		AverageCorePercentMeaning: "(child user CPU seconds + child system CPU seconds) / child wall seconds * 100; a core-based process average, not host or machine CPU percent; values above 100 are possible",
		HotPathNote:               "This report measures the persistent child process end to end, including JSON parsing, queueing, response encoding, process startup, and stream I/O. The separate PredictInto hot-path benchmark excludes those costs and is not substituted for this measurement.",
		Runs:                      make([]runReport, 0, 6),
	}

	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		for _, workers := range []int{1, 4} {
			if ctx.Err() != nil {
				report.Decision = "PARTIAL"
				break
			}
			if currentSHA, hashErr := hashRegularFile(binaryFile); hashErr != nil || currentSHA != binaryDigest {
				report.Decision = "PARTIAL"
				break
			}
			pin := findModelPin(models, variant)
			if currentPin, _, pinErr := loadModelPin(filepath.Join(modelsRoot, variant), variant); pinErr != nil || currentPin != pin {
				report.Decision = "PARTIAL"
				break
			}
			runContext, runCancel := context.WithTimeout(ctx, perRunTimeout)
			result := runOne(runContext, binaryFile, modelPaths[variant], pin, rows, workers)
			runCancel()
			if currentSHA, hashErr := hashRegularFile(binaryFile); hashErr != nil || currentSHA != binaryDigest {
				result.Complete = false
				result.ErrorRecords++
				report.Decision = "PARTIAL"
			}
			if currentPin, _, pinErr := loadModelPin(filepath.Join(modelsRoot, variant), variant); pinErr != nil || currentPin != pin {
				result.Complete = false
				result.ErrorRecords++
				report.Decision = "PARTIAL"
			}
			report.Runs = append(report.Runs, result)
			if result.Complete {
				report.CompletedRuns++
			} else {
				report.Decision = "PARTIAL"
				if result.TimedOut || result.Cancelled || ctx.Err() != nil {
					break
				}
			}
		}
		if report.Decision != "COMPLETE" {
			break
		}
	}

	currentSourceDigest, sourceErr := hashSourceClosure(root)
	report.SourceStable = sourceErr == nil && currentSourceDigest == sourceDigest
	if !report.SourceStable {
		report.Decision = "PARTIAL"
	}
	currentDriverDigest, driverErr := hashRegularFile(filepath.Join(root, "tools", "benchmark-stream", "main.go"))
	report.DriverStable = driverErr == nil && currentDriverDigest == driverDigest
	if !report.DriverStable {
		report.Decision = "PARTIAL"
	}
	if report.CompletedRuns != report.PlannedRuns {
		report.Decision = "PARTIAL"
	}
	report.UnstartedRuns = report.PlannedRuns - len(report.Runs)
	for _, result := range report.Runs {
		report.WrittenRecordsTotal += result.WrittenRecords
		report.ObservedOutputTotal += result.ObservedOutputRecords
		report.CorrelatedRecordsTotal += result.CorrelatedRecords
	}
	report.UnvalidatedRecordsTotal = report.PlannedRecordsTotal - report.CorrelatedRecordsTotal
	if err := writeNewReport(*outputPath, report); err != nil {
		fmt.Fprintln(diagnostics, "benchmark report could not be created without overwriting an existing file")
		return 1
	}
	if report.Decision != "COMPLETE" {
		fmt.Fprintln(diagnostics, "benchmark finished with partial or failed runs; inspect the new report")
		return 1
	}
	fmt.Fprintln(diagnostics, "benchmark report written")
	return 0
}

func loadFrozenRows(manifestPath, datasetPath string) (datasetManifest, string, []datasetRow, map[string]int, map[string]int, error) {
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return datasetManifest{}, "", nil, nil, nil, err
	}
	if err := decision.RejectDuplicateJSONKeys(manifestBytes); err != nil {
		return datasetManifest{}, "", nil, nil, nil, err
	}
	var manifest datasetManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return datasetManifest{}, "", nil, nil, nil, err
	}
	if manifest.Schema != manifestSchema || manifest.DatasetSchema != datasetSchema || manifest.Rows.Total != 2048 || manifest.Rows.Test != rowsPerRun || len(manifest.SplitSHA) != 3 {
		return datasetManifest{}, "", nil, nil, nil, errors.New("manifest contract mismatch")
	}

	dataset, err := os.Open(datasetPath)
	if err != nil {
		return datasetManifest{}, "", nil, nil, nil, err
	}
	defer dataset.Close()
	fullHasher := sha256.New()
	testHasher := sha256.New()
	testRows := make([]datasetRow, 0, rowsPerRun)
	seenIDs := make(map[string]struct{}, 2048)
	labelCounts := make(map[string]int, len(operationLabels))
	languageCounts := map[string]int{"en": 0, "ko": 0}
	allRows := 0
	reader := bufio.NewReaderSize(dataset, 32*1024)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > maxDatasetLineSize {
			return datasetManifest{}, "", nil, nil, nil, errors.New("dataset row exceeds the bounded line size")
		}
		if len(line) != 0 {
			_, _ = fullHasher.Write(line)
			rowBytes := bytes.TrimSuffix(line, []byte{'\n'})
			if err := decision.RejectDuplicateJSONKeys(rowBytes); err != nil {
				return datasetManifest{}, "", nil, nil, nil, errors.New("dataset row is not strict JSON")
			}
			var row datasetRow
			rowDecoder := json.NewDecoder(bytes.NewReader(rowBytes))
			rowDecoder.DisallowUnknownFields()
			if err := rowDecoder.Decode(&row); err != nil {
				return datasetManifest{}, "", nil, nil, nil, errors.New("dataset row schema mismatch")
			}
			if row.ID == "" || row.TemplateID == "" || row.ConfigurationID == "" || row.Text == "" || len(row.Text) > decision.InputMaxBytes || !validLabel(row.Label) || (row.Language != "en" && row.Language != "ko") {
				return datasetManifest{}, "", nil, nil, nil, errors.New("dataset row has invalid fields")
			}
			if _, exists := seenIDs[row.ID]; exists {
				return datasetManifest{}, "", nil, nil, nil, errors.New("dataset contains duplicate row IDs")
			}
			seenIDs[row.ID] = struct{}{}
			allRows++
			switch row.Split {
			case "train", "calibration":
			case "test":
				_, _ = testHasher.Write(line)
				parsed, ok := parseOperands(row.Text)
				if !ok {
					return datasetManifest{}, "", nil, nil, nil, errors.New("test prompt operand prefix is unrecognized")
				}
				goldRequiresBool := row.Label == "and" || row.Label == "or"
				if (parsed.kind == decision.TypeBool) != goldRequiresBool {
					return datasetManifest{}, "", nil, nil, nil, errors.New("test prompt operand type does not match the frozen task")
				}
				testRows = append(testRows, row)
				labelCounts[row.Label]++
				languageCounts[row.Language]++
			default:
				return datasetManifest{}, "", nil, nil, nil, errors.New("dataset contains unknown split")
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return datasetManifest{}, "", nil, nil, nil, errors.New("dataset read failed")
		}
	}
	if allRows != manifest.Rows.Total || len(testRows) != rowsPerRun || hex.EncodeToString(fullHasher.Sum(nil)) != manifest.DatasetSHA || hex.EncodeToString(testHasher.Sum(nil)) != manifest.SplitSHA["test"] {
		return datasetManifest{}, "", nil, nil, nil, errors.New("dataset digest or row count mismatch")
	}
	for _, label := range operationLabels {
		if labelCounts[label] != 32 {
			return datasetManifest{}, "", nil, nil, nil, errors.New("test label distribution mismatch")
		}
	}
	if languageCounts["en"] != 128 || languageCounts["ko"] != 128 {
		return datasetManifest{}, "", nil, nil, nil, errors.New("test language distribution mismatch")
	}
	manifestDigest := sha256.Sum256(manifestBytes)
	return manifest, hex.EncodeToString(manifestDigest[:]), testRows, labelCounts, languageCounts, nil
}

func loadModelPin(directory, variant string) (modelPin, string, error) {
	metadataPath := filepath.Join(directory, "model.json")
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return modelPin{}, "", err
	}
	if err := decision.RejectDuplicateJSONKeys(metadataBytes); err != nil {
		return modelPin{}, "", err
	}
	var metadata modelMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return modelPin{}, "", err
	}
	if metadata.Schema != decision.MetadataSchema || metadata.Variant != variant || metadata.WeightsFile != "weights.bin" || metadata.WeightsSHA == "" {
		return modelPin{}, "", errors.New("model metadata mismatch")
	}
	weightsPath := filepath.Join(directory, "weights.bin")
	weightsSHA, err := hashRegularFile(weightsPath)
	if err != nil || weightsSHA != metadata.WeightsSHA {
		return modelPin{}, "", errors.New("model weights digest mismatch")
	}
	metadataSHA := sha256.Sum256(metadataBytes)
	combined := sha256.New()
	_, _ = io.WriteString(combined, "model.json\x00")
	_, _ = combined.Write(metadataSHA[:])
	_, _ = io.WriteString(combined, "\x00weights.bin\x00")
	weightsRaw, err := os.ReadFile(weightsPath)
	if err != nil {
		return modelPin{}, "", err
	}
	_, _ = combined.Write(weightsRaw)
	pin := modelPin{
		Variant:               variant,
		MetadataSHA256:        hex.EncodeToString(metadataSHA[:]),
		WeightsSHA256:         weightsSHA,
		DeclaredWeightsSHA256: metadata.WeightsSHA,
		BundleSHA256:          hex.EncodeToString(combined.Sum(nil)),
		FileBytes:             int64(len(metadataBytes) + len(weightsRaw)),
	}
	return pin, metadataPath, nil
}

func runOne(parent context.Context, binaryPath, modelPath string, pin modelPin, rows []datasetRow, workers int) runReport {
	report := runReport{
		Variant:           pin.Variant,
		Workers:           workers,
		ModelBundleSHA256: pin.BundleSHA256,
		PlannedRecords:    plannedPerRun,
		ResourcePlatform:  runtime.GOOS,
		ResourceSource:    "getrusage child process resource counters",
	}
	parser := newResultParser(rows, repeatsPerRun, pin.Variant, pin.WeightsSHA256)
	command := exec.CommandContext(parent, binaryPath, "--model", modelPath, "--workers", strconv.Itoa(workers))
	command.Env = replaceEnv(os.Environ(), "GOMAXPROCS", strconv.Itoa(workers))
	stdin, err := command.StdinPipe()
	if err != nil {
		return failedStart(report)
	}
	command.Stdout = parser
	command.Stderr = io.Discard
	startedAt := time.Now()
	if err := command.Start(); err != nil {
		_ = stdin.Close()
		return failedStart(report)
	}
	report.ProcessStarted = true
	feedDone := make(chan feedResult, 1)
	go func() { feedDone <- feedRequests(stdin, rows, repeatsPerRun) }()
	waitErr := command.Wait()
	wall := time.Since(startedAt)
	report.WallMilliseconds = float64(wall) / float64(time.Millisecond)
	feed := <-feedDone
	report.WrittenRecords = feed.written
	report.InputWriteFailed = feed.failed
	parser.finish()
	report.apply(parser.summary)
	if command.ProcessState != nil {
		if code := command.ProcessState.ExitCode(); code >= 0 {
			report.ExitCode = &code
		}
		report.ProcessResources = resources(command.ProcessState, wall)
	}
	if errors.Is(parent.Err(), context.DeadlineExceeded) {
		report.TimedOut = true
	} else if errors.Is(parent.Err(), context.Canceled) {
		report.Cancelled = true
	}
	if waitErr != nil && !report.TimedOut && !report.Cancelled {
		report.ProcessFailed = true
		report.ErrorRecords++
	}
	report.Complete = !report.TimedOut && !report.Cancelled && !report.ProcessFailed && !report.InputWriteFailed && report.ExitCode != nil && *report.ExitCode == 0 && report.WrittenRecords == plannedPerRun && report.ObservedOutputRecords == plannedPerRun && report.UniqueSequences == plannedPerRun && report.CorrelatedRecords == plannedPerRun && report.MissingRecords == 0 && report.DuplicateSequences == 0 && report.OutOfRangeSequences == 0 && report.CorrelationMismatches == 0 && report.MalformedOutputRecords == 0 && report.UnterminatedOutputLines == 0 && report.RejectedResults == 0 && report.ErrorRecords == 0
	return report
}

func failedStart(report runReport) runReport {
	report.ErrorRecords++
	report.MissingRecords = report.PlannedRecords
	return report
}

type feedResult struct {
	written int
	failed  bool
}

func feedRequests(writer io.WriteCloser, rows []datasetRow, repeats int) feedResult {
	defer writer.Close()
	result := feedResult{}
	for repeat := range repeats {
		for _, row := range rows {
			request, err := makeRequest(row, repeat)
			if err != nil {
				result.failed = true
				return result
			}
			encoded, err := json.Marshal(request)
			if err != nil {
				result.failed = true
				return result
			}
			encoded = append(encoded, '\n')
			for len(encoded) > 0 {
				written, writeErr := writer.Write(encoded)
				if writeErr != nil || written <= 0 {
					result.failed = true
					return result
				}
				encoded = encoded[written:]
			}
			result.written++
		}
	}
	return result
}

func makeRequest(row datasetRow, repeat int) (decisionstream.Request, error) {
	parsed, ok := parseOperands(row.Text)
	if !ok {
		return decisionstream.Request{}, errors.New("unrecognized prompt operand prefix")
	}
	return decisionstream.Request{
		Schema:        decisionstream.RequestSchema,
		CorrelationID: row.ID + "-r" + strconv.Itoa(repeat+1),
		Request: decision.DecisionRequest{
			Schema: decision.DecisionRequestSchema,
			Text:   row.Text,
			Left:   decision.Identifier{Name: parsed.left, Type: parsed.kind},
			Right:  decision.Identifier{Name: parsed.right, Type: parsed.kind},
		},
	}, nil
}

func parseOperands(text string) (operands, bool) {
	for _, candidate := range operandPatterns {
		match := candidate.pattern.FindStringSubmatch(text)
		if len(match) == 3 {
			return operands{left: match[1], right: match[2], kind: candidate.kind}, true
		}
	}
	return operands{}, false
}

type resultSummary struct {
	observed              int
	uniqueSequences       int
	correlated            int
	missing               int
	duplicates            int
	outOfRange            int
	correlationMismatches int
	malformed             int
	unterminated          int
	rejected              int
	errorRecords          int
	bestLabelObserved     int
	bestLabelCorrect      int
	accepted              int
	acceptedCorrect       int
	acceptedIncorrect     int
	typeAbstentions       int
	otherAbstentions      int
}

type resultParser struct {
	rows        []datasetRow
	repeats     int
	variant     string
	weightsSHA  string
	line        []byte
	discardLine bool
	seen        []bool
	summary     resultSummary
}

func newResultParser(rows []datasetRow, repeats int, variant, weightsSHA string) *resultParser {
	return &resultParser{
		rows: rows, repeats: repeats, variant: variant, weightsSHA: weightsSHA,
		line: make([]byte, 0, 4096), seen: make([]bool, len(rows)*repeats+1),
	}
}

func (parser *resultParser) Write(data []byte) (int, error) {
	total := len(data)
	for len(data) > 0 {
		newline := bytes.IndexByte(data, '\n')
		if newline < 0 {
			parser.appendFragment(data)
			break
		}
		parser.appendFragment(data[:newline])
		if parser.discardLine {
			parser.summary.malformed++
			parser.summary.errorRecords++
			parser.discardLine = false
			parser.line = parser.line[:0]
		} else {
			parser.parseLine(parser.line)
			parser.line = parser.line[:0]
		}
		data = data[newline+1:]
	}
	return total, nil
}

func (parser *resultParser) appendFragment(fragment []byte) {
	if parser.discardLine {
		return
	}
	if len(fragment) > maxOutputLineSize-len(parser.line) {
		parser.discardLine = true
		parser.line = parser.line[:0]
		return
	}
	parser.line = append(parser.line, fragment...)
}

func (parser *resultParser) parseLine(line []byte) {
	parser.summary.observed++
	if len(line) == 0 || decision.RejectDuplicateJSONKeys(line) != nil {
		parser.summary.malformed++
		parser.summary.errorRecords++
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	var result decisionstream.Result
	if err := decoder.Decode(&result); err != nil {
		parser.summary.malformed++
		parser.summary.errorRecords++
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		parser.summary.malformed++
		parser.summary.errorRecords++
		return
	}
	if result.Schema != decisionstream.ResultSchema || result.Sequence == 0 || result.Sequence >= uint64(len(parser.seen)) {
		parser.summary.outOfRange++
		parser.summary.errorRecords++
		return
	}
	sequence := int(result.Sequence)
	if parser.seen[sequence] {
		parser.summary.duplicates++
		parser.summary.errorRecords++
		return
	}
	parser.seen[sequence] = true
	parser.summary.uniqueSequences++
	rowIndex := (sequence - 1) % len(parser.rows)
	repeat := (sequence - 1) / len(parser.rows)
	expectedCorrelation := parser.rows[rowIndex].ID + "-r" + strconv.Itoa(repeat+1)
	if result.CorrelationID != expectedCorrelation {
		parser.summary.correlationMismatches++
		parser.summary.errorRecords++
		return
	}
	parser.summary.correlated++
	if result.Status == "rejected" {
		parser.summary.rejected++
		parser.summary.errorRecords++
		return
	}
	if result.Status != "completed" || result.Response == nil || result.Response.Schema != decision.DecisionResponseSchema || result.Response.ModelVariant != parser.variant || result.Response.WeightsSHA256 != parser.weightsSHA {
		parser.summary.malformed++
		parser.summary.errorRecords++
		return
	}
	row := parser.rows[rowIndex]
	parsed, ok := parseOperands(row.Text)
	if !ok {
		parser.summary.malformed++
		parser.summary.errorRecords++
		return
	}
	response := result.Response
	if !validLabel(response.BestLabel) {
		parser.summary.malformed++
		parser.summary.errorRecords++
		return
	}
	parser.summary.bestLabelObserved++
	if response.BestLabel == row.Label {
		parser.summary.bestLabelCorrect++
	}
	switch response.Status {
	case "decision":
		if response.TypedBinaryIR == nil || response.TypedBinaryIR.Schema != decision.TypedBinaryIRSchema || response.TypedBinaryIR.Kind != "binary" || response.TypedBinaryIR.Operation != response.BestLabel || response.TypedBinaryIR.Left.Name != parsed.left || response.TypedBinaryIR.Left.Type != parsed.kind || response.TypedBinaryIR.Right.Name != parsed.right || response.TypedBinaryIR.Right.Type != parsed.kind {
			parser.summary.malformed++
			parser.summary.errorRecords++
			return
		}
		validatedIR, err := decision.BuildTypedBinary(response.TypedBinaryIR.Operation, response.TypedBinaryIR.Left, response.TypedBinaryIR.Right)
		if err != nil || validatedIR.ResultType != response.TypedBinaryIR.ResultType {
			parser.summary.malformed++
			parser.summary.errorRecords++
			return
		}
		parser.summary.accepted++
		if response.TypedBinaryIR.Operation == row.Label {
			parser.summary.acceptedCorrect++
		} else {
			parser.summary.acceptedIncorrect++
		}
	case "abstained":
		if response.TypedBinaryIR != nil {
			parser.summary.malformed++
			parser.summary.errorRecords++
			return
		}
		if response.AbstainReason == "TYPE_MISMATCH" {
			parser.summary.typeAbstentions++
		} else {
			parser.summary.otherAbstentions++
		}
	default:
		parser.summary.malformed++
		parser.summary.errorRecords++
	}
}

func (parser *resultParser) finish() {
	if len(parser.line) != 0 || parser.discardLine {
		parser.summary.unterminated++
		parser.summary.errorRecords++
	}
	parser.summary.missing = len(parser.seen) - 1 - parser.summary.uniqueSequences
}

func (report *runReport) apply(summary resultSummary) {
	report.ObservedOutputRecords = summary.observed
	report.UniqueSequences = summary.uniqueSequences
	report.CorrelatedRecords = summary.correlated
	report.MissingRecords = summary.missing
	report.DuplicateSequences = summary.duplicates
	report.OutOfRangeSequences = summary.outOfRange
	report.CorrelationMismatches = summary.correlationMismatches
	report.MalformedOutputRecords = summary.malformed
	report.UnterminatedOutputLines = summary.unterminated
	report.RejectedResults = summary.rejected
	report.ErrorRecords += summary.errorRecords
	report.BestLabelObserved = summary.bestLabelObserved
	report.BestLabelCorrect = summary.bestLabelCorrect
	report.AcceptedIR = summary.accepted
	report.AcceptedCorrect = summary.acceptedCorrect
	report.AcceptedIncorrect = summary.acceptedIncorrect
	report.TypeAbstentions = summary.typeAbstentions
	report.OtherAbstentions = summary.otherAbstentions
}

func resources(state *os.ProcessState, wall time.Duration) processResources {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage == nil {
		return processResources{}
	}
	user := timevalSeconds(int64(usage.Utime.Sec), int64(usage.Utime.Usec))
	system := timevalSeconds(int64(usage.Stime.Sec), int64(usage.Stime.Usec))
	resources := processResources{UserCPUSeconds: &user, SystemCPUSeconds: &system}
	if wall > 0 {
		average := (user + system) / wall.Seconds() * 100
		resources.AverageCorePct = &average
	}
	if bytes, ok := peakRSSBytes(usage.Maxrss, runtime.GOOS); ok {
		resources.PeakRSSBytes = &bytes
		resources.RSSAvailable = true
	}
	return resources
}

func timevalSeconds(seconds, microseconds int64) float64 {
	return float64(seconds) + float64(microseconds)/1_000_000
}

func peakRSSBytes(native int64, goos string) (int64, bool) {
	if native < 0 {
		return 0, false
	}
	switch goos {
	case "darwin":
		return native, true
	case "linux":
		const maxInt64 = int64(^uint64(0) >> 1)
		if native > maxInt64/1024 {
			return 0, false
		}
		return native * 1024, true
	default:
		return 0, false
	}
}

func peakRSSDescription(goos string) string {
	switch goos {
	case "darwin":
		return "getrusage ru_maxrss is bytes on Darwin"
	case "linux":
		return "getrusage ru_maxrss is KiB on Linux and is converted to bytes"
	default:
		return "platform ru_maxrss units are unsupported; peak RSS remains unknown"
	}
}

func hashSourceClosure(root string) (string, error) {
	paths := append([]string(nil), sourceClosurePaths...)
	// The fixed order is part of this source digest protocol.
	hasher := sha256.New()
	for _, relative := range paths {
		file, err := os.Open(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(hasher, relative)
		_, _ = hasher.Write([]byte{0})
		if _, err := io.Copy(hasher, file); err != nil {
			_ = file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
		_, _ = hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashRegularFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func decodeSHA256(value string) ([]byte, error) {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return nil, errors.New("invalid SHA-256 encoding")
	}
	return hex.DecodeString(value)
}

func validLabel(value string) bool {
	for _, label := range operationLabels {
		if value == label {
			return true
		}
	}
	return false
}

func resolveFromRoot(root, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}

func replaceEnv(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found || name != key {
			result = append(result, entry)
		}
	}
	return append(result, key+"="+value)
}

func findModelPin(models []modelPin, variant string) modelPin {
	for _, model := range models {
		if model.Variant == variant {
			return model
		}
	}
	return modelPin{}
}

func writeNewReport(path string, report benchmarkReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}
