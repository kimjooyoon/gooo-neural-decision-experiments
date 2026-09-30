// Command main independently checks the frozen native body-plan CI evidence.
// It reads the repository and CI capture; it does not run models or compilers.
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
	"reflect"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodydecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

const expectedRevision = "7d626b9ab0b503f5c3d571be09524d689271bd48"

type row struct {
	ID       string              `json:"id"`
	Family   string              `json:"family"`
	Plan     bodyplan.Plan       `json:"plan"`
	Oracle   map[string]string   `json:"oracle_operations"`
	Training []bodydecision.Case `json:"training_cases"`
	Heldout  []bodydecision.Case `json:"heldout_cases"`
}

type cell struct {
	ID               string                     `json:"id"`
	Family           string                     `json:"family"`
	Arm              string                     `json:"arm"`
	Initial          bodydecision.Selection     `json:"initial_selection"`
	Search           *bodydecision.SearchResult `json:"training_search,omitempty"`
	Choices          map[string]string          `json:"emitted_choices"`
	Training         bodydecision.Score         `json:"training"`
	Heldout          bodydecision.Score         `json:"heldout"`
	OracleHoles      int                        `json:"oracle_correct_holes"`
	TotalHoles       int                        `json:"total_holes"`
	NativePass       bool                       `json:"native_gooo_generation_pass"`
	GoTests          int                        `json:"compiled_go_cases_observed"`
	CompiledTraining bodydecision.Score         `json:"compiled_go_training"`
	CompiledHeldout  bodydecision.Score         `json:"compiled_go_heldout"`
	GoooSHA          string                     `json:"gooo_source_sha256"`
	GoSHA            string                     `json:"native_go_source_sha256,omitempty"`
	Status           string                     `json:"status"`
}

type evaluation struct {
	Schema                string                       `json:"schema"`
	Status                string                       `json:"status"`
	CohortSHA             string                       `json:"cohort_sha256"`
	SourceRevision        string                       `json:"source_revision"`
	RunnerSHA             string                       `json:"runner_binary_sha256"`
	GoooSHA               string                       `json:"gooo_binary_sha256"`
	GoSHA                 string                       `json:"go_tool_sha256"`
	LayaRevision          string                       `json:"laya_revision,omitempty"`
	PlannedIntents        int                          `json:"planned_intents"`
	PlannedCells          int                          `json:"planned_cells"`
	ObservedCells         int                          `json:"observed_selected_cells"`
	AttemptLimit          int                          `json:"training_candidate_attempt_limit"`
	LayaPOSTs             int                          `json:"laya_posts_attempted"`
	PlannedLayaPOSTs      int                          `json:"laya_posts_planned"`
	TinyPredictions       int                          `json:"tiny_predictions_observed"`
	WallMS                float64                      `json:"wall_ms"`
	Cells                 []cell                       `json:"cells"`
	Limits                []string                     `json:"limits"`
	FamilyCounts          map[string]int               `json:"family_counts"`
	UnstartedCells        int                          `json:"unobserved_selection_cells"`
	NativePassed          int                          `json:"native_generated_cells"`
	NativeFailedOrUnknown int                          `json:"native_failed_or_unknown_cells"`
	PlannedGoCases        int                          `json:"compiled_go_cases_planned"`
	ObservedGoCases       int                          `json:"compiled_go_cases_observed"`
	UnknownGoCases        int                          `json:"compiled_go_cases_unknown"`
	ExecutablePinsStable  bool                         `json:"executable_pins_stable_after_execution"`
	ModelArtifactSHA256   map[string]map[string]string `json:"model_artifact_sha256"`
}

type nativeReply struct {
	Report struct {
		Decision  string `json:"decision"`
		Typecheck bool   `json:"typecheck_passed"`
		Replay    bool   `json:"deterministic_replay"`
	} `json:"report"`
	Source string `json:"source"`
}

type goAudit struct {
	Decision   string                       `json:"decision"`
	Status     string                       `json:"status"`
	DatasetSHA string                       `json:"dataset_sha256"`
	ParitySHA  string                       `json:"parity_sha256"`
	InputSHA   map[string]string            `json:"input_sha256"`
	TestScores map[string]auditVariantScore `json:"test_scores"`
}

type auditVariantScore struct {
	Model struct {
		ModelJSONSHA string `json:"model_json_sha256"`
		WeightsSHA   string `json:"weights_sha256"`
	} `json:"model"`
}

type preexecution struct {
	Schema      string         `json:"schema"`
	DatasetSHA  string         `json:"dataset_sha256"`
	TrainingSHA string         `json:"training_source_sha256"`
	SplitCounts map[string]int `json:"split_counts"`
	TestUsed    bool           `json:"test_used_for_training_or_calibration"`
	Variants    []string       `json:"variants"`
}

type datasetManifest struct {
	DatasetSHA  string `json:"dataset_sha256"`
	ContractSHA string `json:"model_contract_sha256"`
}

type publicSummary struct {
	PreexecutionSHA string `json:"preexecution_sha256"`
	DatasetSHA      string `json:"dataset_sha256"`
	TrainingSHA     string `json:"training_source_sha256"`
}

type armStats struct {
	Cells                    int `json:"cells"`
	TrainingCorrect          int `json:"training_correct"`
	TrainingTotal            int `json:"training_total"`
	HeldoutCorrect           int `json:"heldout_correct"`
	HeldoutTotal             int `json:"heldout_total"`
	GoCases                  int `json:"compiled_go_cases_observed"`
	InitialOracleCorrectHole int `json:"initial_oracle_correct_holes"`
	TotalOracleHoles         int `json:"total_oracle_holes"`
	EmittedOracleCorrectHole int `json:"emitted_oracle_correct_holes"`
	SearchCells              int `json:"search_cells,omitempty"`
	SearchAttempts           int `json:"search_attempts,omitempty"`
	SearchMaxAttempt         int `json:"search_max_attempts,omitempty"`
	SearchAt64               int `json:"search_cells_at_64_attempts"`
	FullHeldoutCells         int `json:"cells_with_all_heldout_cases_correct,omitempty"`
	InitialSameAsOut         int `json:"initial_choices_equal_emitted,omitempty"`
}

type auditResult struct {
	Schema             string              `json:"schema"`
	Decision           string              `json:"decision"`
	Scope              string              `json:"scope"`
	RepositoryRevision string              `json:"repository_revision"`
	CaptureReportSHA   string              `json:"capture_report_sha256"`
	SelectedCellsSHA   string              `json:"selected_cells_sha256"`
	CohortSHA          string              `json:"cohort_sha256"`
	ManifestSHA        string              `json:"cohort_manifest_sha256"`
	RunnerSourceSHA    string              `json:"runner_source_sha256"`
	NativePinInCI      string              `json:"native_compiler_source_pin_in_workflow"`
	ArmStats           map[string]armStats `json:"arm_stats"`
	ModelBindings      map[string]any      `json:"model_bindings"`
	NativeEvidence     map[string]any      `json:"native_evidence"`
	SearchComparisons  map[string]any      `json:"search_comparisons_vs_deterministic_control"`
	LayaCapture        map[string]any      `json:"laya_capture,omitempty"`
	VerifierSources    map[string]string   `json:"verifier_and_dependency_source_sha256"`
	Caveats            []string            `json:"caveats"`
}

type markerResult struct {
	ID     string
	Split  string
	Index  int
	Passed bool
}

func main() {
	repoFlag := flag.String("repo", ".", "repository root")
	captureFlag := flag.String("capture", "/tmp/gooo-body-plan-ci-36722049671-20260930", "completed CI evidence directory")
	outFlag := flag.String("out", "review/body-plan-ci-v1-20260930/audit.json", "write audit receipt here")
	flag.Parse()
	if err := run(*repoFlag, *captureFlag, *outFlag); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(repo, capture, output string) error {
	repo, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	capture, err = filepath.Abs(capture)
	if err != nil {
		return err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	sourcePinsBefore, err := verifierSourceDigests(repo)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	revisionBytes, err := commandOutput(repo, "git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	revision := strings.TrimSpace(string(revisionBytes))
	reportBytes, err := readFile(filepath.Join(capture, "report.json"), 32<<20)
	if err != nil {
		return err
	}
	var report evaluation
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		return err
	}
	cohortBytes, err := readFile(filepath.Join(repo, "studies/body-plan-v1/cohort.jsonl"), 16<<20)
	if err != nil {
		return err
	}
	manifestBytes, err := readFile(filepath.Join(repo, "studies/body-plan-v1/manifest.json"), 2<<20)
	if err != nil {
		return err
	}
	if digest(cohortBytes) != report.CohortSHA {
		return errors.New("capture cohort SHA does not bind to checked-in cohort bytes")
	}
	if report.SourceRevision != expectedRevision || revision != report.SourceRevision {
		return fmt.Errorf("source revision mismatch: repo=%s report=%s expected=%s", revision, report.SourceRevision, expectedRevision)
	}
	rows, err := readRows(cohortBytes)
	if err != nil {
		return err
	}
	manifest, err := readManifest(manifestBytes)
	if err != nil {
		return err
	}
	if manifest["cohort_sha256"] != report.CohortSHA || manifest["plan_count"] != float64(len(rows)) {
		return errors.New("cohort manifest does not bind plan count and digest")
	}
	if len(rows) != 128 {
		return fmt.Errorf("cohort has %d rows, expected 128", len(rows))
	}
	rowByID, families, holeCount := make(map[string]row), make(map[string]int), 0
	for _, item := range rows {
		rowByID[item.ID] = item
		families[item.Family]++
		for _, expr := range item.Plan.Expressions {
			if expr.Kind == "hole" {
				holeCount++
			}
		}
	}
	if !reflect.DeepEqual(families, report.FamilyCounts) {
		return errors.New("capture family counts differ from frozen cohort")
	}
	if err := checkModelBindings(repo, report); err != nil {
		return err
	}
	if err := checkTrainingProvenance(repo); err != nil {
		return err
	}
	workflowBytes, err := readFile(filepath.Join(repo, ".github/workflows/go-model-validation.yml"), 1<<20)
	if err != nil {
		return err
	}
	workflow := string(workflowBytes)
	nativePin := "60cf7f49b0e302a6bebb42bc8da90f3ed19b2b82"
	if !strings.Contains(workflow, "ref: "+nativePin) || !strings.Contains(workflow, "--gooo-sha256") || !strings.Contains(workflow, "--go-sha256") {
		return errors.New("CI workflow does not pin native compiler and Go executable digests")
	}
	runnerBytes, err := readFile(filepath.Join(repo, "tools/evaluate-bodyplans/main.go"), 2<<20)
	if err != nil {
		return err
	}
	selectedBytes, err := readFile(filepath.Join(capture, "selected-cells.jsonl"), 32<<20)
	if err != nil {
		return err
	}
	selectedCells, err := readCells(selectedBytes)
	if err != nil {
		return err
	}
	if len(selectedCells) != 1024 || len(report.Cells) != 1024 || report.PlannedCells != 1024 || report.ObservedCells != 1024 {
		return errors.New("capture does not contain exactly 1024 planned, journaled and reported cells")
	}
	if report.Status != "completed" || report.UnstartedCells != 0 || report.NativePassed != 1024 || report.NativeFailedOrUnknown != 0 || report.UnknownGoCases != 0 || !report.ExecutablePinsStable {
		return errors.New("capture reports incomplete cells, unknown cases, failure, or changed executable pins")
	}
	if report.LayaPOSTs != 0 || report.PlannedLayaPOSTs != 0 {
		return errors.New("unexpected Laya request in model-only CI capture")
	}
	if report.AttemptLimit != 64 {
		return errors.New("search attempt limit differs from declared 64")
	}
	arms := []string{"deterministic", "deterministic_search", "fp32", "fp32_search", "ptq_ternary", "ptq_ternary_search", "qat_ternary", "qat_ternary_search"}
	if report.PlannedIntents != len(rows) || report.TinyPredictions != 3*holeCount {
		return errors.New("intent or prediction-call denominator differs from frozen plans")
	}
	plannedCases := 0
	for _, item := range rows {
		plannedCases += (len(item.Training) + len(item.Heldout)) * len(arms)
	}
	if report.PlannedGoCases != plannedCases || plannedCases != 36160 {
		return errors.New("planned compiled-Go case denominator differs from the 128-plan corpus")
	}
	reportByKey, selectedByKey := make(map[string]cell), make(map[string]cell)
	for _, item := range report.Cells {
		key := item.ID + "\x00" + item.Arm
		if _, exists := reportByKey[key]; exists {
			return errors.New("duplicate report cell")
		}
		reportByKey[key] = item
	}
	for _, item := range selectedCells {
		key := item.ID + "\x00" + item.Arm
		if _, exists := selectedByKey[key]; exists {
			return errors.New("duplicate selected journal cell")
		}
		selectedByKey[key] = item
	}
	if len(reportByKey) != 1024 || len(selectedByKey) != 1024 {
		return errors.New("cell IDs do not cover the declared matrix")
	}
	stats := make(map[string]armStats)
	markerCount := 0
	for _, arm := range arms {
		if err := checkArmReplay(repo, capture, arm, rows, reportByKey, selectedByKey, report.ModelArtifactSHA256, nil, &stats, &markerCount); err != nil {
			return fmt.Errorf("arm %s: %w", arm, err)
		}
	}
	if report.ObservedGoCases != plannedCases || markerCount != plannedCases {
		return fmt.Errorf("compiled Go markers: report=%d markers=%d planned=%d", report.ObservedGoCases, markerCount, plannedCases)
	}
	if err := checkArmScoreExpectations(stats); err != nil {
		return err
	}
	bindings := map[string]any{}
	for _, arm := range arms {
		item := stats[arm]
		item.Cells = 128
		stats[arm] = item
	}
	searchComparisons := make(map[string]any)
	controlAttempts := stats["deterministic_search"].SearchAttempts
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		attempts := stats[variant+"_search"].SearchAttempts
		delta := controlAttempts - attempts
		searchComparisons[variant] = map[string]any{
			"deterministic_control_attempts_including_initial": controlAttempts,
			"model_initialization_attempts_including_initial":  attempts,
			"attempts_saved":             delta,
			"attempt_reduction_fraction": float64(delta) / float64(controlAttempts),
			"interpretation":             "Same 64-per-cell finite training-case enumeration and final heldout success; these savings describe how the initial model choice changes candidate evaluations, not a learned repair model.",
		}
	}
	layaCapture, err := auditLayaCapture(repo, filepath.Join(repo, "runs/body-plan-v1-laya-7d626b9-20260930"), report, stats)
	if err != nil {
		return fmt.Errorf("Laya follow-up audit: %w", err)
	}
	sourcePinsAfter, err := verifierSourceDigests(repo)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(sourcePinsBefore, sourcePinsAfter) {
		return errors.New("verifier/dependency source bytes changed during this audit")
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		bindings[variant] = report.ModelArtifactSHA256[variant]
	}
	bindings["tiny_predictions_observed"] = report.TinyPredictions
	bindings["all_model_external_provider_calls"] = 0
	bindings["all_per_hole_text_hashes_checked"] = true
	bindings["selection_provenance"] = "Report receipts were checked for text hashes, model variant/weight pins, probability-vector label order/sum/argmax and allowed-option fallback rules. Model weights were hashed and linked to both the CI report and the independent model-audit receipt; this verifier did not run model prediction calls."
	result := auditResult{
		Schema: "gooo/body-plan-ci-independent-review/v1", Decision: "PASS",
		Scope:              "Frozen source/data/model/native provenance and compiled-Go case markers for the model-free CI evaluation, plus raw exchange and compiled-case binding for the completed local Laya follow-up; no model prediction or native compiler execution by this verifier.",
		RepositoryRevision: revision, CaptureReportSHA: digest(reportBytes), SelectedCellsSHA: digest(selectedBytes),
		CohortSHA: report.CohortSHA, ManifestSHA: digest(manifestBytes), RunnerSourceSHA: digest(runnerBytes),
		NativePinInCI: nativePin, ArmStats: stats, ModelBindings: bindings,
		NativeEvidence:    map[string]any{"native_compile_and_typecheck_pass_cells": report.NativePassed, "native_compile_fail_or_unknown_cells": report.NativeFailedOrUnknown, "compiled_go_case_markers_planned": plannedCases, "compiled_go_case_markers_unique_and_checked": markerCount, "compiled_go_cases_unknown": report.UnknownGoCases, "native_gated_source_pin": nativePin, "runner_checks_binary_hash_before_and_after": report.ExecutablePinsStable, "native_stdout_source_hashes_and_report_hashes_matched": true},
		SearchComparisons: searchComparisons,
		LayaCapture:       layaCapture,
		VerifierSources:   sourcePinsAfter,
		Caveats: []string{
			"The CI artifact reports binary digests and the runner rechecks pinned executable bytes; those transient native compiler/toolchain binaries are not included in the downloaded evidence, so this audit binds their provenance through the workflow and completed report rather than rehashing the binaries locally.",
			"The frozen CI run used bodyplan.go at the source revision above. A later evaluator optimization is present in the current checkout; the local audit used its current bytes, recorded separately below, to replay stored markers and scores. The experiment-revision file digest is also independently pinned in this receipt.",
			"The handwritten expected-value formulas and generated plans share their authored semantics; this finite suite is not a whole-domain or full-language proof.",
			"Repeated arm cells reuse 128 plans and do not add distinct intents. Finite search uses training examples only; its final 100% heldout score is not attributable to the model alone.",
			"Laya and tiny-model first-choice protocols expose different context: Laya receives allowed-operation descriptions with the hole text, while tiny models classify the text against global labels.",
		},
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(output, data, 0644); err != nil {
		return err
	}
	fmt.Printf("PASS: 128 plans, 1024 cells, %d/%d unique Go case markers; report %s\n", markerCount, plannedCases, digest(reportBytes))
	return nil
}

func checkTrainingProvenance(repo string) error {
	data, err := readFile(filepath.Join(repo, "data/synthetic-ops-v1/dataset.jsonl"), 64<<20)
	if err != nil {
		return err
	}
	datasetManifestBytes, err := readFile(filepath.Join(repo, "data/synthetic-ops-v1/manifest.json"), 2<<20)
	if err != nil {
		return err
	}
	var dataset datasetManifest
	if err := json.Unmarshal(datasetManifestBytes, &dataset); err != nil {
		return err
	}
	if digest(data) != dataset.DatasetSHA {
		return errors.New("model-training dataset SHA differs from its manifest")
	}
	preBytes, err := readFile(filepath.Join(repo, "runs/pilot-mps-20260930-v1/preexecution.json"), 2<<20)
	if err != nil {
		return err
	}
	var pre preexecution
	if err := json.Unmarshal(preBytes, &pre); err != nil {
		return err
	}
	trainingBytes, err := readFile(filepath.Join(repo, "training/train_pilot.py"), 2<<20)
	if err != nil {
		return err
	}
	contractBytes, err := readFile(filepath.Join(repo, "model-contract.json"), 2<<20)
	if err != nil {
		return err
	}
	if pre.DatasetSHA != digest(data) || pre.TrainingSHA != digest(trainingBytes) || pre.TestUsed || pre.SplitCounts["train"] != 1536 || pre.SplitCounts["calibration"] != 256 || pre.SplitCounts["test"] != 256 {
		return errors.New("preexecution record does not bind the frozen dataset, source and held-out split")
	}
	if dataset.ContractSHA != digest(contractBytes) {
		return errors.New("dataset manifest does not bind current model contract")
	}
	summaryBytes, err := readFile(filepath.Join(repo, "runs/pilot-mps-20260930-v1/public-training-summary.json"), 2<<20)
	if err != nil {
		return err
	}
	var summary publicSummary
	if err := json.Unmarshal(summaryBytes, &summary); err != nil {
		return err
	}
	if summary.PreexecutionSHA != digest(preBytes) || summary.DatasetSHA != pre.DatasetSHA || summary.TrainingSHA != pre.TrainingSHA {
		return errors.New("public filtered training summary does not bind the preexecution receipt")
	}
	return nil
}

func checkModelBindings(repo string, report evaluation) error {
	auditBytes, err := readFile(filepath.Join(repo, "runs/pilot-mps-20260930-v1/go-audit.json"), 16<<20)
	if err != nil {
		return err
	}
	var audit goAudit
	if err := json.Unmarshal(auditBytes, &audit); err != nil {
		return err
	}
	if audit.Decision != "PASS" || audit.Status != "PASS" {
		return errors.New("independent Go model audit is not PASS")
	}
	parityBytes, err := readFile(filepath.Join(repo, "runs/pilot-mps-20260930-v1/go-parity.json"), 32<<20)
	if err != nil {
		return err
	}
	if digest(parityBytes) != audit.ParitySHA {
		return errors.New("Go parity receipt SHA does not bind")
	}
	dataBytes, err := readFile(filepath.Join(repo, "data/synthetic-ops-v1/dataset.jsonl"), 64<<20)
	if err != nil {
		return err
	}
	if digest(dataBytes) != audit.DatasetSHA {
		return errors.New("Go audit dataset SHA does not bind")
	}
	if report.ModelArtifactSHA256 == nil {
		return errors.New("CI report lacks model artifact hashes")
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		base := filepath.Join(repo, "runs/pilot-mps-20260930-v1/models", variant)
		modelBytes, err := readFile(filepath.Join(base, "model.json"), 2<<20)
		if err != nil {
			return err
		}
		weightsBytes, err := readFile(filepath.Join(base, "weights.bin"), 2<<20)
		if err != nil {
			return err
		}
		modelHash, weightsHash := digest(modelBytes), digest(weightsBytes)
		if report.ModelArtifactSHA256[variant]["model_json"] != modelHash || report.ModelArtifactSHA256[variant]["weights_bin"] != weightsHash {
			return fmt.Errorf("%s model bytes differ from evaluation report pins", variant)
		}
		if audit.InputSHA[filepath.ToSlash(filepath.Join("runs/pilot-mps-20260930-v1/models", variant, "model.json"))] != modelHash || audit.InputSHA[filepath.ToSlash(filepath.Join("runs/pilot-mps-20260930-v1/models", variant, "weights.bin"))] != weightsHash {
			return fmt.Errorf("%s model bytes differ from independent Go audit pins", variant)
		}
		if audit.TestScores[variant].Model.ModelJSONSHA != modelHash || audit.TestScores[variant].Model.WeightsSHA != weightsHash {
			return fmt.Errorf("%s Go audit model subreceipt differs", variant)
		}
		loaded, err := decision.Load(filepath.Join(base, "model.json"))
		if err != nil {
			return fmt.Errorf("%s model failed load: %w", variant, err)
		}
		if loaded.Variant() != variant || loaded.WeightsSHA256() != weightsHash {
			return fmt.Errorf("%s metadata does not bind variant and weights", variant)
		}
	}
	return nil
}

func checkArmReplay(repo, capture, arm string, rows []row, reportCells, selectedCells map[string]cell, modelHashes map[string]map[string]string, exchanges map[string][]layaExchange, allStats *map[string]armStats, markerTotal *int) error {
	stats := (*allStats)[arm]
	outputPath := filepath.Join(capture, arm, "compiled-replay", "go-test-output.txt")
	output, err := readFile(outputPath, 8<<20)
	if err != nil {
		return err
	}
	markers, err := readMarkers(output)
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	var combinedGo strings.Builder
	combinedGo.WriteString("package bodyplan\n\n")
	var combinedTest strings.Builder
	combinedTest.WriteString("package bodyplan\nimport (\"testing\";\"fmt\")\nfunc TestFrozenCases(t *testing.T) {\n")
	for _, item := range rows {
		key := item.ID + "\x00" + arm
		entry, ok := reportCells[key]
		if !ok {
			return fmt.Errorf("missing report cell %s", key)
		}
		journal, ok := selectedCells[key]
		if !ok {
			return fmt.Errorf("missing selection journal cell %s", key)
		}
		if entry.ID != item.ID || entry.Family != item.Family || journal.ID != item.ID || journal.Family != item.Family {
			return fmt.Errorf("cell identity/family mismatch for %s", item.ID)
		}
		if entry.Status != "native_compiled_and_executed" || !entry.NativePass {
			return fmt.Errorf("native generation status not complete for %s", item.ID)
		}
		if !sameJournalSelection(journal, entry) {
			return fmt.Errorf("selected cell journal disagrees with final report for %s", item.ID)
		}
		if err := checkSelection(item, entry, modelHashes, exchanges[item.ID]); err != nil {
			return fmt.Errorf("selection %s: %w", item.ID, err)
		}
		program, err := bodyplan.Compile(item.Plan, entry.Choices)
		if err != nil {
			return fmt.Errorf("emitted choices do not typecheck for %s: %w", item.ID, err)
		}
		trainingScore, err := bodydecision.ScoreProgram(program, item.Training)
		if err != nil {
			return err
		}
		heldoutScore, err := bodydecision.ScoreProgram(program, item.Heldout)
		if err != nil {
			return err
		}
		if entry.Training != trainingScore || entry.Heldout != heldoutScore {
			return fmt.Errorf("report score differs from frozen case interpreter for %s", item.ID)
		}
		if entry.TotalHoles != len(item.Oracle) || entry.OracleHoles != correctHoles(entry.Choices, item.Oracle) {
			return fmt.Errorf("oracle-choice count differs for %s", item.ID)
		}
		stats.InitialOracleCorrectHole += correctHoles(entry.Initial.Choices, item.Oracle)
		stats.EmittedOracleCorrectHole += entry.OracleHoles
		stats.TotalOracleHoles += len(item.Oracle)
		if strings.HasSuffix(arm, "_search") {
			baseArm := strings.TrimSuffix(arm, "_search")
			base := reportCells[item.ID+"\x00"+baseArm]
			if entry.Search == nil || entry.Search.ModelCalls != 0 || entry.Search.Limit != 64 {
				return fmt.Errorf("search receipt invalid for %s", item.ID)
			}
			expectedSearch, err := bodydecision.Search(item.Plan, base.Initial.Choices, item.Training, 64)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(*entry.Search, expectedSearch) {
				return fmt.Errorf("search receipt does not match training-only replay for %s", item.ID)
			}
			stats.SearchCells++
			stats.SearchAttempts += len(entry.Search.Attempts)
			if len(entry.Search.Attempts) > stats.SearchMaxAttempt {
				stats.SearchMaxAttempt = len(entry.Search.Attempts)
			}
			if len(entry.Search.Attempts) == 64 {
				stats.SearchAt64++
			}
			if reflect.DeepEqual(base.Initial.Choices, entry.Choices) {
				stats.InitialSameAsOut++
			}
		}
		if entry.Heldout.Correct == entry.Heldout.Total {
			stats.FullHeldoutCells++
		}
		stats.Cells++
		stats.TrainingCorrect += entry.Training.Correct
		stats.TrainingTotal += entry.Training.Total
		stats.HeldoutCorrect += entry.Heldout.Correct
		stats.HeldoutTotal += entry.Heldout.Total
		stats.GoCases += entry.GoTests
		if entry.GoTests != len(item.Training)+len(item.Heldout) || entry.CompiledTraining != entry.Training || entry.CompiledHeldout != entry.Heldout {
			return fmt.Errorf("compiled case denominator or score mismatch for %s", item.ID)
		}
		dir := filepath.Join(capture, arm, item.ID)
		goooSource := program.GoooSource()
		goooPath := filepath.Join(dir, "input.gooo")
		goooBytes, err := readFile(goooPath, 1<<20)
		if err != nil {
			return err
		}
		if !bytes.Equal(goooBytes, []byte(goooSource)) || digest(goooBytes) != entry.GoooSHA {
			return fmt.Errorf("Gooo input/source digest mismatch for %s", item.ID)
		}
		planGo := program.GoSource()
		planGoBytes, err := readFile(filepath.Join(dir, "plan-generated.go.txt"), 1<<20)
		if err != nil {
			return err
		}
		if !bytes.Equal(planGoBytes, []byte(planGo)) {
			return fmt.Errorf("plan Go source mismatch for %s", item.ID)
		}
		nativeBytes, err := readFile(filepath.Join(dir, "native-generated.go.txt"), 1<<20)
		if err != nil {
			return err
		}
		if digest(nativeBytes) != entry.GoSHA {
			return fmt.Errorf("native-generated source digest mismatch for %s", item.ID)
		}
		var reply nativeReply
		stdout, err := readFile(filepath.Join(dir, "native-stdout.json"), 2<<20)
		if err != nil {
			return err
		}
		var replyFields map[string]json.RawMessage
		if err := strictjson.Decode(stdout, &replyFields); err != nil {
			return err
		}
		if err := json.Unmarshal(stdout, &reply); err != nil {
			return err
		}
		if reply.Report.Decision != "PASS" || !reply.Report.Typecheck || !reply.Report.Replay || !bytes.Equal([]byte(reply.Source), nativeBytes) {
			return fmt.Errorf("native compiler reply does not bind emitted source for %s", item.ID)
		}
		combinedGo.WriteString(strings.TrimPrefix(string(nativeBytes), "package bodyplan\n"))
		caseLists := []struct {
			name  string
			cases []bodydecision.Case
		}{{"training", item.Training}, {"heldout", item.Heldout}}
		for _, split := range caseLists {
			for index, test := range split.cases {
				value, err := program.Evaluate(test.Input)
				if err != nil {
					return err
				}
				passed := valueMatches(value, test.Expected)
				markerKey := fmt.Sprintf("%s\x00%s\x00%d", item.ID, split.name, index)
				marker, exists := markers[markerKey]
				if !exists || seen[markerKey] {
					return fmt.Errorf("missing or duplicate Go execution marker %s", markerKey)
				}
				seen[markerKey] = true
				if marker.Passed != passed {
					return fmt.Errorf("compiled Go marker differs from frozen case evaluation %s", markerKey)
				}
				literal := fmt.Sprintf("int64(%d)", test.Expected.Int)
				if test.Expected.Type == decision.TypeBool {
					literal = strconv.FormatBool(test.Expected.Bool)
				}
				fmt.Fprintf(&combinedTest, "{ got := %s(int64(%d)); passed := got == %s; fmt.Printf(%q, passed); if !passed { t.Errorf(%q, got) } }\n", item.Plan.Name, test.Input, literal, "GOOO_CASE\t"+item.ID+"\t"+split.name+"\t"+strconv.Itoa(index)+"\t%t\n", item.ID+": got %v")
			}
		}
	}
	combinedTest.WriteString("}\n")
	if len(seen) != len(markers) {
		return fmt.Errorf("unexpected runtime markers: seen=%d marker_count=%d", len(seen), len(markers))
	}
	compiledDir := filepath.Join(capture, arm, "compiled-replay")
	compiledGo, err := readFile(filepath.Join(compiledDir, "generated.go"), 16<<20)
	if err != nil {
		return err
	}
	if !bytes.Equal(compiledGo, []byte(combinedGo.String())) {
		return errors.New("compiled Go source is not the concatenation of native emitted source files")
	}
	compiledTests, err := readFile(filepath.Join(compiledDir, "generated_test.go"), 16<<20)
	if err != nil {
		return err
	}
	if !bytes.Equal(compiledTests, []byte(combinedTest.String())) {
		return errors.New("compiled test source does not match frozen cases and marker plan")
	}
	*markerTotal += len(markers)
	(*allStats)[arm] = stats
	return nil
}

func checkSelection(item row, entry cell, modelHashes map[string]map[string]string, exchanges []layaExchange) error {
	provider := strings.TrimSuffix(entry.Arm, "_search")
	if provider == "laya_multilingual" {
		return checkLayaSelection(item, entry, exchanges)
	}
	if len(entry.Initial.Holes) != countHoles(item.Plan) {
		return errors.New("hole receipt count mismatch")
	}
	if len(entry.Initial.Choices) != countHoles(item.Plan) {
		return errors.New("initial choice count mismatch")
	}
	if provider == "deterministic" {
		if entry.Initial.ModelCalls != 0 || entry.Initial.ProviderCalls != 0 || entry.Initial.ModelVariant != "" {
			return errors.New("deterministic arm unexpectedly called a model or provider")
		}
		for _, expr := range item.Plan.Expressions {
			if expr.Kind != "hole" {
				continue
			}
			if entry.Initial.Choices[expr.HoleID] != expr.Fallback {
				return errors.New("deterministic choice is not declared fallback")
			}
			var receipt *bodydecision.HoleReceipt
			for i := range entry.Initial.Holes {
				if entry.Initial.Holes[i].ID == expr.HoleID {
					receipt = &entry.Initial.Holes[i]
					break
				}
			}
			if receipt == nil || receipt.Selected != expr.Fallback || receipt.Mode != "declared_fallback" || receipt.TextSHA256 != digest([]byte(expr.Text)) {
				return fmt.Errorf("deterministic hole receipt differs for %s", expr.HoleID)
			}
		}
		return nil
	}
	if entry.Initial.ModelCalls != len(entry.Initial.Holes) || entry.Initial.ProviderCalls != 0 || entry.Initial.ModelVariant != provider {
		return errors.New("model selection receipt has wrong route/call counts")
	}
	if entry.Initial.WeightsSHA256 == "" || entry.Initial.WeightsSHA256 != modelHashes[provider]["weights_bin"] {
		return errors.New("model selection weights do not match pinned artifact")
	}
	for _, expr := range item.Plan.Expressions {
		if expr.Kind != "hole" {
			continue
		}
		var receipt *bodydecision.HoleReceipt
		for i := range entry.Initial.Holes {
			if entry.Initial.Holes[i].ID == expr.HoleID {
				receipt = &entry.Initial.Holes[i]
				break
			}
		}
		if receipt == nil {
			return fmt.Errorf("missing hole receipt %s", expr.HoleID)
		}
		if digest([]byte(expr.Text)) != receipt.TextSHA256 {
			return fmt.Errorf("text hash differs for %s", expr.HoleID)
		}
		if entry.Initial.Choices[expr.HoleID] != receipt.Selected {
			return fmt.Errorf("selection map differs for %s", expr.HoleID)
		}
		if receipt.Selected != expr.Fallback && !contains(expr.Allowed, receipt.Selected) {
			return fmt.Errorf("selected operation outside allowed set for %s", expr.HoleID)
		}
		if len(receipt.Probabilities) != len(decision.Labels()) {
			return fmt.Errorf("probability vector size mismatch for %s", expr.HoleID)
		}
		maximum := float32(-1)
		label := ""
		total := float64(0)
		for i, probability := range receipt.Probabilities {
			if probability.Label != decision.Labels()[i] || math.IsNaN(float64(probability.Probability)) || math.IsInf(float64(probability.Probability), 0) || probability.Probability < 0 || probability.Probability > 1 {
				return fmt.Errorf("invalid probability entry for %s", expr.HoleID)
			}
			total += float64(probability.Probability)
			if probability.Probability > maximum {
				maximum, label = probability.Probability, probability.Label
			}
		}
		if math.Abs(total-1) > 2e-6 || math.Abs(float64(receipt.Confidence-maximum)) > 1e-7 || receipt.Proposed != label {
			return fmt.Errorf("probability sum/confidence/argmax inconsistent for %s", expr.HoleID)
		}
		switch receipt.Mode {
		case "model_global_argmax":
			if !contains(expr.Allowed, label) || receipt.Selected != label {
				return fmt.Errorf("model argmax routing inconsistent for %s", expr.HoleID)
			}
		case "fallback_global_label_outside_allowed":
			if contains(expr.Allowed, label) || receipt.Selected != expr.Fallback {
				return fmt.Errorf("outside-allowed fallback inconsistent for %s", expr.HoleID)
			}
		case "fallback_low_global_confidence":
			if receipt.Selected != expr.Fallback || receipt.Confidence >= 0.125 {
				return fmt.Errorf("low-confidence fallback inconsistent for %s", expr.HoleID)
			}
		default:
			return fmt.Errorf("unexpected model receipt mode %q", receipt.Mode)
		}
	}
	return nil
}

func checkArmScoreExpectations(stats map[string]armStats) error {
	expected := map[string]int{"deterministic": 384, "fp32": 950, "ptq_ternary": 679, "qat_ternary": 916}
	for arm, correct := range expected {
		if stats[arm].HeldoutCorrect != correct || stats[arm].HeldoutTotal != 1280 {
			return fmt.Errorf("%s first-choice heldout score mismatch: %+v", arm, stats[arm])
		}
	}
	if stats["deterministic"].InitialOracleCorrectHole != 0 || stats["deterministic"].TotalOracleHoles != 222 {
		return errors.New("deterministic fallback hole count differs from frozen gold assignments")
	}
	for _, arm := range []string{"deterministic_search", "fp32_search", "ptq_ternary_search", "qat_ternary_search"} {
		if stats[arm].HeldoutCorrect != 1280 || stats[arm].HeldoutTotal != 1280 || stats[arm].SearchCells != 128 {
			return fmt.Errorf("%s finite search did not complete all 128 heldout intent suites", arm)
		}
	}
	return nil
}

func sameJournalSelection(journal, final cell) bool {
	return journal.ID == final.ID && journal.Family == final.Family && journal.Arm == final.Arm &&
		reflect.DeepEqual(journal.Initial, final.Initial) && reflect.DeepEqual(journal.Search, final.Search) &&
		reflect.DeepEqual(journal.Choices, final.Choices) && journal.Training == final.Training && journal.Heldout == final.Heldout &&
		journal.OracleHoles == final.OracleHoles && journal.TotalHoles == final.TotalHoles &&
		journal.GoooSHA == final.GoooSHA && journal.GoSHA == final.GoSHA
}

func readRows(data []byte) ([]row, error) {
	var rows []row
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 65536)
	seen := make(map[string]bool)
	for scanner.Scan() {
		var item row
		if err := strictjson.Decode(scanner.Bytes(), &item); err != nil {
			return nil, err
		}
		if item.ID == "" || item.ID != item.Plan.ID || seen[item.ID] {
			return nil, errors.New("cohort identity missing, mismatched or duplicated")
		}
		seen[item.ID] = true
		if len(item.Training) == 0 || len(item.Heldout) == 0 {
			return nil, errors.New("cohort has an empty case split")
		}
		oracle, err := bodyplan.Compile(item.Plan, item.Oracle)
		if err != nil {
			return nil, err
		}
		goldTrain, err := bodydecision.ScoreProgram(oracle, item.Training)
		if err != nil || goldTrain.Correct != goldTrain.Total {
			return nil, fmt.Errorf("%s gold training does not score fully", item.ID)
		}
		goldHeldout, err := bodydecision.ScoreProgram(oracle, item.Heldout)
		if err != nil || goldHeldout.Correct != goldHeldout.Total {
			return nil, fmt.Errorf("%s gold heldout does not score fully", item.ID)
		}
		rows = append(rows, item)
	}
	return rows, scanner.Err()
}

func readCells(data []byte) ([]cell, error) {
	var cells []cell
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 65536)
	for scanner.Scan() {
		var entry cell
		if err := strictjson.Decode(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		cells = append(cells, entry)
	}
	return cells, scanner.Err()
}

func readManifest(data []byte) (map[string]any, error) {
	var value map[string]any
	if err := strictjson.Decode(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func readMarkers(data []byte) (map[string]markerResult, error) {
	result := make(map[string]markerResult)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 65536)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "GOOO_CASE\t") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 5 {
			return nil, errors.New("malformed runtime marker")
		}
		index, err := strconv.Atoi(parts[3])
		passed, boolErr := strconv.ParseBool(parts[4])
		if err != nil || boolErr != nil {
			return nil, errors.New("invalid runtime marker value")
		}
		key := fmt.Sprintf("%s\x00%s\x00%d", parts[1], parts[2], index)
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("duplicate runtime marker %s", key)
		}
		result[key] = markerResult{ID: parts[1], Split: parts[2], Index: index, Passed: passed}
	}
	return result, scanner.Err()
}

func correctHoles(choices, oracle map[string]string) int {
	count := 0
	for id, expected := range oracle {
		if choices[id] == expected {
			count++
		}
	}
	return count
}
func countHoles(plan bodyplan.Plan) int {
	count := 0
	for _, expr := range plan.Expressions {
		if expr.Kind == "hole" {
			count++
		}
	}
	return count
}
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func valueMatches(got, expected bodyplan.Value) bool {
	if got.Type != expected.Type {
		return false
	}
	if got.Type == decision.TypeInt {
		return got.Int == expected.Int
	}
	return got.Bool == expected.Bool
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func readFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds audit read limit: %s", path)
	}
	return data, nil
}

func commandOutput(directory, name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	command.Dir = directory
	return command.Output()
}

func verifierSourceDigests(repo string) (map[string]string, error) {
	result := make(map[string]string)
	for label, relative := range map[string]string{
		"audit_main_go":        "review/body-plan-ci-v1-20260930/main.go",
		"audit_laya_go":        "review/body-plan-ci-v1-20260930/laya.go",
		"bodyplan_runner_go":   "tools/evaluate-bodyplans/main.go",
		"bodyplan_go_worktree": "internal/bodyplan/bodyplan.go",
	} {
		data, err := readFile(filepath.Join(repo, relative), 4<<20)
		if err != nil {
			return nil, err
		}
		result[label] = digest(data)
	}
	for label, directory := range map[string]string{
		"bodydecision_package": "internal/bodydecision",
		"decision_package":     "internal/decision",
		"strictjson_package":   "internal/strictjson",
	} {
		value, err := sourceDirectoryDigest(repo, directory)
		if err != nil {
			return nil, err
		}
		result[label] = value
	}
	frozen, err := commandOutput(repo, "git", "show", expectedRevision+":internal/bodyplan/bodyplan.go")
	if err != nil {
		return nil, err
	}
	result["bodyplan_go_at_experiment_revision"] = digest(frozen)
	if result["bodyplan_go_at_experiment_revision"] != "286fc65f177bc77a02b54889ef2f4928b60590b802d003b4c1c145a0b02583ae" {
		return nil, errors.New("experiment revision bodyplan source digest mismatch")
	}
	return result, nil
}

func sourceDirectoryDigest(repo, relative string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(repo, relative))
	if err != nil {
		return "", err
	}
	var joined strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := readFile(filepath.Join(repo, relative, entry.Name()), 4<<20)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&joined, "%s\x00%s\n", entry.Name(), digest(data))
	}
	return digest([]byte(joined.String())), nil
}
