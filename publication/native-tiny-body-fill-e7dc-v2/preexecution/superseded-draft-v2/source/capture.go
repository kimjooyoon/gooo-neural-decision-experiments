// Standalone, standard-library-only capture wrapper for the native tiny_go
// body-codegen smoke fixtures. Run one wrapper process per fixture/model cell
// so RUSAGE_CHILDREN describes exactly one CLI child.
package main

import (
	"bytes"
	"context"
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
	"runtime"
	"strings"
	"syscall"
	"time"
)

const schema = "gooo/native-tiny-go-body-smoke/v2"

type testCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}

type candidate struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
}

type plan struct {
	Schema        string      `json:"schema"`
	Intent        string      `json:"intent"`
	HoleID        string      `json:"hole_id"`
	ProviderModel string      `json:"provider_model,omitempty"`
	Candidates    []candidate `json:"candidates"`
	TestCases     []testCase  `json:"test_cases"`
}

type modelMetadata struct {
	Schema        string `json:"schema"`
	Variant       string `json:"variant"`
	WeightsFile   string `json:"weights_file"`
	WeightsSHA256 string `json:"weights_sha256"`
}

type score struct {
	ID              string  `json:"id"`
	Expression      string  `json:"expression"`
	TypecheckPassed bool    `json:"typecheck_passed"`
	TestCasesPassed int     `json:"test_cases_passed"`
	TestCasesTotal  int     `json:"test_cases_total"`
	AccuracyPercent float64 `json:"accuracy_percent"`
}

type caseResult struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
	Passed   bool  `json:"passed"`
}

type dimension struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Numerator   int    `json:"numerator"`
	Denominator int    `json:"denominator"`
}

type bodyFillReport struct {
	ProposedCandidateID string  `json:"proposed_candidate_id"`
	ProposedAccuracy    float64 `json:"proposed_accuracy_percent"`
	SelectedCandidateID string  `json:"selected_candidate_id"`
	BestCandidateID     string  `json:"best_candidate_id"`
	BestAccuracy        float64 `json:"best_candidate_accuracy_percent"`
	Regret              float64 `json:"selection_regret_percentage_points"`
	Adjustment          string  `json:"selection_adjustment"`
	TestCasesPassed     int     `json:"test_cases_passed"`
	TestCasesTotal      int     `json:"test_cases_total"`
	FunctionalAccuracy  float64 `json:"functional_accuracy_percent"`
	LocalPredictions    int     `json:"local_model_predictions"`
	ExternalCalls       int     `json:"external_provider_calls"`
	ExternalCallsKnown  bool    `json:"external_provider_calls_known"`
	Decision            struct {
		Provider                 string `json:"provider"`
		Mode                     string `json:"mode"`
		Selected                 string `json:"selected"`
		FallbackReason           string `json:"fallback_reason"`
		TinyGoVariant            string `json:"tiny_go_variant"`
		TinyGoWeightsSHA256      string `json:"tiny_go_weights_sha256"`
		TinyGoMetadataSHA256     string `json:"tiny_go_metadata_sha256"`
		TinyGoPredictedOperation string `json:"tiny_go_predicted_operation"`
		TinyGoPredictionApplied  *bool  `json:"tiny_go_prediction_applied"`
		RequestSHA256            string `json:"request_sha256"`
	} `json:"decision"`
	Timing struct {
		ModelLoadMS *float64 `json:"tiny_model_load_ms"`
		DecisionMS  float64  `json:"tiny_decision_ms"`
		TotalMS     float64  `json:"total_ms"`
	} `json:"timing"`
	CandidateScores []score      `json:"candidate_scores"`
	CaseResults     []caseResult `json:"selected_case_results"`
}

type cliEnvelope struct {
	Report struct {
		Decision          string `json:"decision"`
		CompilerSourceSHA string `json:"compiler_source_sha"`
		Completeness      struct {
			Scope      map[string]any `json:"scope"`
			Dimensions []dimension    `json:"dimensions"`
		} `json:"completeness_receipt"`
		BodyFill bodyFillReport `json:"body_fill"`
	} `json:"report"`
	Source string `json:"source"`
}

type invocationReceipt struct {
	Schema                   string            `json:"schema"`
	Status                   string            `json:"status"`
	ExecutionMode            string            `json:"execution_mode"`
	FixtureID                string            `json:"fixture_id"`
	Activity                 string            `json:"activity"`
	FixtureFile              string            `json:"fixture_file"`
	FixtureSHA256            string            `json:"fixture_sha256"`
	PlanFile                 string            `json:"plan_file"`
	PlanFileSHA256           string            `json:"plan_file_sha256"`
	PlanSemanticSHA256       string            `json:"plan_semantic_sha256"`
	CompilerSourceSHA256     string            `json:"compiler_source_sha256_expected"`
	CompilerSourceObserved   string            `json:"compiler_source_sha256_observed,omitempty"`
	CLIBinarySHA256Expected  string            `json:"cli_binary_sha256_expected"`
	CLIBinarySHA256Observed  string            `json:"cli_binary_sha256_observed"`
	CLIBinarySHA256After     string            `json:"cli_binary_sha256_after"`
	RunnerSourceSHA256       string            `json:"runner_source_sha256"`
	RunnerSourceFile         string            `json:"runner_source_file"`
	RunnerBinarySHA256       string            `json:"runner_binary_sha256"`
	RunnerSourceBytes        int64             `json:"runner_source_bytes"`
	ModelVariant             string            `json:"model_variant"`
	ModelMetadataSHA256      string            `json:"model_metadata_sha256"`
	ModelMetadataSHA256After string            `json:"model_metadata_sha256_after"`
	ModelWeightsSHA256       string            `json:"model_weights_sha256"`
	ModelWeightsSHA256After  string            `json:"model_weights_sha256_after"`
	ModelWeightsBytes        int64             `json:"model_weights_bytes"`
	ModelMetadataFile        string            `json:"model_metadata_file"`
	LayaURLChildEnvironment  string            `json:"laya_url_child_environment"`
	LayaKeyChildEnvironment  string            `json:"laya_key_child_environment"`
	ChildCount               int               `json:"child_count"`
	ChildTimeoutMS           int64             `json:"child_timeout_ms"`
	Command                  []string          `json:"path_neutral_command"`
	ExitCode                 int               `json:"exit_code"`
	ChildRunError            string            `json:"child_run_error,omitempty"`
	WallMS                   float64           `json:"cli_active_wall_ms"`
	ChildUserCPUMS           float64           `json:"child_user_cpu_ms"`
	ChildSystemCPUMS         float64           `json:"child_system_cpu_ms"`
	ChildMaxRSS              int64             `json:"child_max_rss"`
	ChildMaxRSSUnit          string            `json:"child_max_rss_unit"`
	ResourceScope            string            `json:"resource_scope"`
	ResourceError            string            `json:"resource_error,omitempty"`
	StdoutBytes              int               `json:"stdout_bytes"`
	StdoutSHA256             string            `json:"stdout_sha256"`
	StderrBytes              int               `json:"stderr_bytes"`
	StderrSHA256             string            `json:"stderr_sha256"`
	ValidationErrors         []string          `json:"validation_errors"`
	Decision                 *decisionSummary  `json:"decision,omitempty"`
	GoldOperation            string            `json:"gold_operation"`
	RawPredictionMatchesGold *bool             `json:"raw_prediction_matches_gold,omitempty"`
	MappedProposalCandidate  string            `json:"mapped_proposal_candidate_id,omitempty"`
	MappedProposalScore      *candidateEval    `json:"mapped_proposal_finite_score,omitempty"`
	FiniteSuite              *suiteSummary     `json:"finite_suite,omitempty"`
	Completeness             *completenessView `json:"completeness,omitempty"`
}

type decisionSummary struct {
	Provider                 string   `json:"provider"`
	Mode                     string   `json:"mode"`
	FallbackReason           string   `json:"fallback_reason,omitempty"`
	ProposedCandidateID      string   `json:"proposed_candidate_id"`
	ProposedAccuracy         float64  `json:"proposed_accuracy_percent"`
	ProposedScoreSemantics   string   `json:"proposed_score_semantics"`
	SelectedCandidateID      string   `json:"selected_candidate_id"`
	BestCandidateID          string   `json:"best_candidate_id"`
	BestAccuracy             float64  `json:"best_candidate_accuracy_percent"`
	SelectionRegret          float64  `json:"selection_regret_percentage_points"`
	SelectionAdjustment      string   `json:"selection_adjustment"`
	TinyGoVariant            string   `json:"tiny_go_variant"`
	TinyGoWeightsSHA256      string   `json:"tiny_go_weights_sha256"`
	TinyGoMetadataSHA256     string   `json:"tiny_go_metadata_sha256"`
	TinyGoPredictedOperation string   `json:"tiny_go_predicted_operation,omitempty"`
	TinyGoPredictionApplied  *bool    `json:"tiny_go_prediction_applied,omitempty"`
	RequestSHA256            string   `json:"request_sha256"`
	LocalPredictions         int      `json:"local_model_predictions"`
	ExternalCalls            int      `json:"external_provider_calls"`
	ExternalCallsKnown       bool     `json:"external_provider_calls_known"`
	TinyModelLoadMS          *float64 `json:"tiny_model_load_ms,omitempty"`
	TinyDecisionMS           float64  `json:"tiny_decision_ms"`
	BodycodegenTotalMS       float64  `json:"bodycodegen_total_ms"`
}

type suiteSummary struct {
	Cases                []testCase      `json:"cases"`
	CandidateScores      []score         `json:"candidate_scores"`
	ReportedCaseResults  []caseResult    `json:"reported_selected_case_results"`
	IndependentCandidate []candidateEval `json:"independent_candidate_scores"`
	IndependentSelected  []caseResult    `json:"independent_selected_case_results"`
	SelectedPassed       int             `json:"selected_cases_passed"`
	SelectedTotal        int             `json:"selected_cases_total"`
	FunctionalAccuracy   float64         `json:"functional_accuracy_percent"`
}

type candidateEval struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
	Passed     int    `json:"passed"`
	Total      int    `json:"total"`
}

type completenessView struct {
	DecisionProvider string      `json:"decision_provider"`
	LayaProvider     string      `json:"laya_provider"`
	Dimensions       []dimension `json:"dimensions"`
}

func main() {
	var (
		mode         = flag.String("mode", "", "tiny_go or offline_baseline")
		binaryPath   = flag.String("binary", "", "absolute path to the pinned native gooo CLI binary")
		modelPath    = flag.String("model", "", "absolute path to one trained model.json")
		modelVariant = flag.String("model-variant", "", "expected fp32, ptq_ternary, or qat_ternary variant")
		metadataSHA  = flag.String("model-metadata-sha256", "", "expected SHA-256 of model.json bytes")
		weightsSHA   = flag.String("model-weights-sha256", "", "expected SHA-256 of weights.bin bytes")
		sourceSHA    = flag.String("compiler-source-sha", "", "expected compiler source commit SHA")
		binarySHA    = flag.String("binary-sha256", "", "expected native CLI binary SHA-256")
		runnerSHA    = flag.String("runner-source-sha256", "", "SHA-256 of this exact smoke.go source")
		runnerPath   = flag.String("runner-source", "", "absolute path to this exact smoke.go source")
		fixtureID    = flag.String("fixture", "", "frozen fixture ID: arithmetic or boolean")
		outputDir    = flag.String("output-dir", "", "new per-cell capture directory")
	)
	flag.Parse()
	if err := run(*mode, *binaryPath, *modelPath, *modelVariant, *metadataSHA, *weightsSHA, *sourceSHA, *binarySHA, *runnerSHA, *runnerPath, *fixtureID, *outputDir); err != nil {
		fmt.Fprintln(os.Stderr, "tiny smoke:", err)
		os.Exit(2)
	}
}

func run(mode, binaryPath, modelPath, modelVariant, expectedMetadataSHA, expectedWeightsSHA, expectedSourceSHA, expectedBinarySHA, runnerSourceSHA, runnerSourcePath, fixtureID, outputDir string) error {
	if fixtureID != "arithmetic" && fixtureID != "boolean" {
		return errors.New("fixture must be arithmetic or boolean")
	}
	if mode != "tiny_go" && mode != "offline_baseline" {
		return errors.New("mode must be tiny_go or offline_baseline")
	}
	for label, value := range map[string]string{
		"binary":              binaryPath,
		"compiler source SHA": expectedSourceSHA, "binary SHA-256": expectedBinarySHA,
		"runner source SHA-256": runnerSourceSHA, "runner source": runnerSourcePath, "output directory": outputDir,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	for label, value := range map[string]string{
		"binary SHA-256": expectedBinarySHA, "runner source SHA-256": runnerSourceSHA,
	} {
		if !isLowerHexDigest(value, 64) {
			return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", label)
		}
	}
	if !isLowerHexDigest(expectedSourceSHA, 40) {
		return errors.New("compiler source SHA must be a 40-character lowercase commit SHA")
	}
	if !filepath.IsAbs(binaryPath) || !filepath.IsAbs(runnerSourcePath) {
		return errors.New("binary and runner source paths must be absolute")
	}
	if mode == "tiny_go" {
		if modelVariant != "fp32" && modelVariant != "ptq_ternary" && modelVariant != "qat_ternary" {
			return errors.New("tiny_go model variant must be fp32, ptq_ternary, or qat_ternary")
		}
		if !filepath.IsAbs(modelPath) || !isLowerHexDigest(expectedMetadataSHA, 64) || !isLowerHexDigest(expectedWeightsSHA, 64) {
			return errors.New("tiny_go mode requires an absolute model path and both model SHA-256 pins")
		}
	} else if modelPath != "" || modelVariant != "" || expectedMetadataSHA != "" || expectedWeightsSHA != "" {
		return errors.New("offline_baseline must not specify a tiny model or model pins")
	}
	runnerSourceBytes, err := os.ReadFile(runnerSourcePath)
	if err != nil {
		return fmt.Errorf("read runner source: %w", err)
	}
	if sha256Hex(runnerSourceBytes) != runnerSourceSHA {
		return errors.New("runner source SHA-256 differs from the supplied exact-source pin")
	}
	fixtureRoot, err := filepath.Abs("fixtures")
	if err != nil {
		return fmt.Errorf("resolve frozen fixture directory: %w", err)
	}
	fixtureName := fixtureID + ".gooo"
	planName := fixtureID + ".plan.json"
	fixtureBytes, err := os.ReadFile(filepath.Join(fixtureRoot, fixtureName))
	if err != nil {
		return fmt.Errorf("read frozen fixture: %w", err)
	}
	planBytes, err := os.ReadFile(filepath.Join(fixtureRoot, planName))
	if err != nil {
		return fmt.Errorf("read frozen plan: %w", err)
	}
	var planDoc plan
	if err := decodeStrictOneJSON(planBytes, &planDoc); err != nil {
		return fmt.Errorf("decode frozen plan: %w", err)
	}
	if planDoc.ProviderModel != "" {
		return errors.New("frozen tiny_go plan must not contain a provider_model selector")
	}
	activity := "ArithmeticHole"
	if fixtureID == "boolean" {
		activity = "BooleanHole"
	}
	if err := validateFrozenFixture(fixtureID, string(fixtureBytes), planDoc, planBytes); err != nil {
		return err
	}
	metadataDigest, actualWeightsSHA, weightsPath := "", "", ""
	var weightsBytes []byte
	if mode == "tiny_go" {
		metadataBytes, readErr := os.ReadFile(modelPath)
		if readErr != nil {
			return fmt.Errorf("read model metadata: %w", readErr)
		}
		metadataDigest = sha256Hex(metadataBytes)
		if metadataDigest != expectedMetadataSHA {
			return errors.New("model metadata SHA-256 differs from the supplied source-bound pin")
		}
		var metadata modelMetadata
		if err := decodeOneJSON(metadataBytes, &metadata); err != nil {
			return fmt.Errorf("decode model metadata: %w", err)
		}
		if metadata.Schema != "gooo/tiny-ir-decision-model/v1" || metadata.Variant != modelVariant {
			return errors.New("model schema or variant differs from its source-bound pin")
		}
		if metadata.WeightsFile == "" || filepath.IsAbs(metadata.WeightsFile) || filepath.Base(metadata.WeightsFile) != metadata.WeightsFile || strings.ContainsAny(metadata.WeightsFile, `/\\`) || metadata.WeightsFile == "." || metadata.WeightsFile == ".." {
			return errors.New("model weights_file must be a simple basename")
		}
		weightsPath = filepath.Join(filepath.Dir(modelPath), metadata.WeightsFile)
		weightsBytes, err = os.ReadFile(weightsPath)
		if err != nil {
			return fmt.Errorf("read model weights: %w", err)
		}
		actualWeightsSHA = sha256Hex(weightsBytes)
		if actualWeightsSHA != expectedWeightsSHA || actualWeightsSHA != metadata.WeightsSHA256 {
			return errors.New("model weights SHA-256 does not match metadata and the supplied pin")
		}
	}
	cliBytes, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("read CLI binary: %w", err)
	}
	actualBinarySHA := sha256Hex(cliBytes)
	if actualBinarySHA != expectedBinarySHA {
		return errors.New("CLI binary SHA-256 differs from the supplied clean-build pin")
	}
	runnerExecutable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve runner executable: %w", err)
	}
	runnerBytes, err := os.ReadFile(runnerExecutable)
	if err != nil {
		return fmt.Errorf("read runner executable: %w", err)
	}
	if err := mkdirFresh(outputDir); err != nil {
		return err
	}
	modelFile := ""
	if mode == "tiny_go" {
		modelFile = "model.json"
	}
	commandLabel := []string{"gooo", "body-codegen", "--json", "--fill-plan", planName, "--activity", activity, fixtureName}
	if mode == "tiny_go" {
		commandLabel = []string{"gooo", "body-codegen", "--json", "--tiny-model", "<pinned-model.json>", "--fill-plan", planName, "--activity", activity, fixtureName}
	}
	rootReceipt := invocationReceipt{
		Schema: schema, Status: "CAPTURING", ExecutionMode: mode, FixtureID: fixtureID, Activity: activity,
		FixtureFile: fixtureName, FixtureSHA256: sha256Hex(fixtureBytes), PlanFile: planName,
		PlanFileSHA256: sha256Hex(planBytes), PlanSemanticSHA256: semanticPlanSHA(planDoc),
		CompilerSourceSHA256: expectedSourceSHA, CLIBinarySHA256Expected: expectedBinarySHA,
		CLIBinarySHA256Observed: actualBinarySHA, RunnerSourceSHA256: runnerSourceSHA,
		RunnerBinarySHA256: sha256Hex(runnerBytes), RunnerSourceFile: filepath.Base(runnerSourcePath),
		RunnerSourceBytes: int64(len(runnerSourceBytes)), ModelVariant: modelVariant,
		ModelMetadataSHA256: metadataDigest, ModelWeightsSHA256: actualWeightsSHA,
		ModelWeightsBytes: int64(len(weightsBytes)), ModelMetadataFile: modelFile,
		LayaURLChildEnvironment: "unset", LayaKeyChildEnvironment: "unset", ChildCount: 1,
		Command:  commandLabel,
		ExitCode: -1, ChildMaxRSSUnit: rssUnit(),
		ChildTimeoutMS: 30000,
		ResourceScope:  "one native gooo body-codegen child only; model metadata/weights are hashed before child timing when tiny_go; RUSAGE_CHILDREN CPU is delta around this sole child; max RSS belongs to this sole child",
	}
	if err := writeJSON(filepath.Join(outputDir, "prelaunch.json"), rootReceipt); err != nil {
		return err
	}
	childContext, cancelChild := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelChild()
	args := []string{"body-codegen", "--json"}
	if mode == "tiny_go" {
		args = append(args, "--tiny-model", modelPath)
	}
	args = append(args, "--fill-plan", planName, "--activity", activity, fixtureName)
	command := exec.CommandContext(childContext, binaryPath, args...)
	command.Dir = fixtureRoot
	command.Env = withoutLayaEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	var before, after syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &before); err != nil {
		return fmt.Errorf("get child usage before invocation: %w", err)
	}
	started := time.Now()
	runErr := command.Run()
	wallMS := float64(time.Since(started)) / float64(time.Millisecond)
	if err := writeFile(filepath.Join(outputDir, "raw-cli.json"), stdout.Bytes()); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outputDir, "raw-stderr.txt"), stderr.Bytes()); err != nil {
		return err
	}
	afterErr := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &after)
	postCLIBytes, postCLIReadErr := os.ReadFile(binaryPath)
	var postMetadataBytes, postWeightsBytes []byte
	var postMetadataReadErr, postWeightsReadErr error
	if mode == "tiny_go" {
		postMetadataBytes, postMetadataReadErr = os.ReadFile(modelPath)
		postWeightsBytes, postWeightsReadErr = os.ReadFile(weightsPath)
	}
	if postCLIReadErr == nil {
		rootReceipt.CLIBinarySHA256After = sha256Hex(postCLIBytes)
	} else {
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "CLI binary could not be rehashed after execution")
	}
	if mode == "tiny_go" && postMetadataReadErr == nil {
		rootReceipt.ModelMetadataSHA256After = sha256Hex(postMetadataBytes)
	} else if mode == "tiny_go" {
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "model metadata could not be rehashed after execution")
	}
	if mode == "tiny_go" && postWeightsReadErr == nil {
		rootReceipt.ModelWeightsSHA256After = sha256Hex(postWeightsBytes)
	} else if mode == "tiny_go" {
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "model weights could not be rehashed after execution")
	}
	if rootReceipt.CLIBinarySHA256After != expectedBinarySHA || (mode == "tiny_go" && (rootReceipt.ModelMetadataSHA256After != expectedMetadataSHA || rootReceipt.ModelWeightsSHA256After != expectedWeightsSHA)) {
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "source-bound binary or trained bundle changed during the cell")
	}
	rootReceipt.ExitCode = exitCode(runErr)
	if runErr != nil {
		rootReceipt.ChildRunError = "child exited unsuccessfully; inspect exit_code and preserved raw output"
	}
	rootReceipt.WallMS = wallMS
	if afterErr != nil {
		rootReceipt.ResourceError = "post-child RUSAGE_CHILDREN observation failed"
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, rootReceipt.ResourceError)
	} else {
		rootReceipt.ChildUserCPUMS = timevalMillis(after.Utime) - timevalMillis(before.Utime)
		rootReceipt.ChildSystemCPUMS = timevalMillis(after.Stime) - timevalMillis(before.Stime)
	}
	rootReceipt.ChildMaxRSS = after.Maxrss
	rootReceipt.StdoutBytes, rootReceipt.StdoutSHA256 = len(stdout.Bytes()), sha256Hex(stdout.Bytes())
	rootReceipt.StderrBytes, rootReceipt.StderrSHA256 = len(stderr.Bytes()), sha256Hex(stderr.Bytes())
	if rootReceipt.ChildUserCPUMS < 0 || rootReceipt.ChildSystemCPUMS < 0 {
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "child CPU resource counters decreased")
	}
	containsLocalPath := func(data []byte) bool {
		text := string(data)
		return strings.Contains(text, binaryPath) || (modelPath != "" && strings.Contains(text, modelPath)) ||
			strings.Contains(text, "/Users/") || strings.Contains(text, "/tmp/")
	}
	if containsLocalPath(stdout.Bytes()) || containsLocalPath(stderr.Bytes()) {
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "raw CLI stdout/stderr contains a local absolute path")
	}
	if rootReceipt.ExitCode != 0 {
		rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "native CLI did not exit successfully")
	} else {
		var payload cliEnvelope
		if err := decodeOneJSON(stdout.Bytes(), &payload); err != nil {
			rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors, "CLI stdout is not exactly one JSON report: "+err.Error())
		} else {
			rootReceipt.CompilerSourceObserved = payload.Report.CompilerSourceSHA
			rootReceipt.Decision = summarizeDecision(payload.Report.BodyFill)
			rootReceipt.GoldOperation = goldOperation(fixtureID)
			if mode == "tiny_go" {
				if payload.Report.BodyFill.Decision.TinyGoPredictedOperation != "" {
					matches := payload.Report.BodyFill.Decision.TinyGoPredictedOperation == rootReceipt.GoldOperation
					rootReceipt.RawPredictionMatchesGold = &matches
				}
				rootReceipt.MappedProposalCandidate, rootReceipt.MappedProposalScore = mappedProposal(fixtureID, planDoc, payload.Report.BodyFill.Decision.TinyGoPredictedOperation)
			}
			rootReceipt.Completeness = summarizeCompleteness(payload.Report.Completeness.Scope, payload.Report.Completeness.Dimensions)
			rootReceipt.FiniteSuite = verifySuite(fixtureID, planDoc, payload.Report.BodyFill)
			rootReceipt.ValidationErrors = append(rootReceipt.ValidationErrors,
				validateEnvelope(payload, rootReceipt, planDoc, mode, modelVariant, actualWeightsSHA, metadataDigest)...)
		}
	}
	if len(rootReceipt.ValidationErrors) == 0 {
		rootReceipt.Status = "CAPTURED_AND_SOURCE_BOUND"
	} else {
		rootReceipt.Status = "CAPTURED_WITH_VALIDATION_ERRORS"
	}
	if err := writeJSON(filepath.Join(outputDir, "invocation.json"), rootReceipt); err != nil {
		return err
	}
	if len(rootReceipt.ValidationErrors) != 0 {
		return fmt.Errorf("cell captured with %d validation error(s); inspect invocation.json", len(rootReceipt.ValidationErrors))
	}
	return nil
}

func validateFrozenFixture(id, source string, p plan, planBytes []byte) error {
	if p.Schema != "gooo/body-codegen-ir-fill-plan/v1" || len(p.Candidates) != 3 || len(p.TestCases) != 6 {
		return errors.New("frozen fixture plan has unexpected schema or dimensions")
	}
	wantSourceSHA, wantPlanSHA := frozenFixtureHashes(id)
	if wantSourceSHA == "" || sha256Hex([]byte(source)) != wantSourceSHA {
		return errors.New("frozen fixture source bytes differ from the paired finite study pin")
	}
	if !strings.Contains(source, "__GOOO_BODY_HOLE_"+p.HoleID+"__") {
		return errors.New("frozen source does not contain the plan's one named body hole")
	}
	if !strings.Contains(source, "let result = ") || !strings.Contains(source, " else {") ||
		!strings.Contains(source, "return result") || strings.Count(source, "result = result ") < 2 {
		return errors.New("frozen source does not exercise a local assignment, if/else branches, updates, and return")
	}
	want := map[string][]candidate{
		"arithmetic": {{ID: "sum", Expression: "input + 2"}, {ID: "difference", Expression: "input - 2"}, {ID: "product", Expression: "input * 2"}},
		"boolean":    {{ID: "less_than", Expression: "input < 0"}, {ID: "less_equal", Expression: "input <= 0"}, {ID: "equal", Expression: "input == 0"}},
	}[id]
	if len(want) != len(p.Candidates) {
		return errors.New("frozen candidate count differs from the declared oracle")
	}
	if sha256Hex(planBytes) != wantPlanSHA {
		return errors.New("frozen fill-plan bytes differ from the paired finite study pin")
	}
	if id == "arithmetic" && (p.Intent != "Add the input and two before the conditional adjustment." || p.HoleID != "arithmetic") {
		return errors.New("arithmetic fixture intent or hole ID differs from the paired plan")
	}
	if id == "boolean" && (p.Intent != "Use the less-than-or-equal comparison to select zero and negative inputs." || p.HoleID != "boolean") {
		return errors.New("Boolean fixture intent or hole ID differs from the paired plan")
	}
	for i := range want {
		if want[i] != p.Candidates[i] {
			return fmt.Errorf("candidate %d differs from the independent frozen oracle", i)
		}
	}
	inputs := []int64{math.MinInt64, -9, -1, 0, 6, math.MaxInt64}
	if id == "boolean" {
		inputs = []int64{math.MinInt64, -7, -1, 0, 1, math.MaxInt64}
	}
	if len(inputs) != len(p.TestCases) {
		return errors.New("frozen finite suite has unexpected count")
	}
	for i, input := range inputs {
		wantExpected := arithmeticExpected(input)
		if id == "boolean" {
			wantExpected = booleanExpected(input)
		}
		if p.TestCases[i] != (testCase{Input: input, Expected: wantExpected}) {
			return fmt.Errorf("finite oracle case %d differs from the separately specified input/expected contract", i)
		}
	}
	return nil
}

func arithmeticExpected(input int64) int64 {
	base := input + 2
	if input < 0 {
		return base + 2
	}
	return base
}

func booleanExpected(input int64) int64 {
	if input <= 0 {
		return 2
	}
	return 0
}

func evalCandidate(id, fixture string, input int64) int64 {
	if fixture == "arithmetic" {
		base := input
		switch id {
		case "sum":
			base = input + 2
		case "difference":
			base = input - 2
		case "product":
			base = input * 2
		}
		if input < 0 {
			return base + 2
		}
		return base
	}
	condition := false
	switch id {
	case "less_than":
		condition = input < 0
	case "less_equal":
		condition = input <= 0
	case "equal":
		condition = input == 0
	}
	if condition {
		return 2
	}
	return 0
}

func goldOperation(fixture string) string {
	switch fixture {
	case "arithmetic":
		return "add"
	case "boolean":
		return "less_equal"
	default:
		return ""
	}
}

func frozenFixtureHashes(fixture string) (sourceSHA, planSHA string) {
	switch fixture {
	case "arithmetic":
		return "7896b7f65db047bf077a03920901ff4487576582896f492fabe5d1fc82c1d748",
			"685b22ffb073ad0e8fb597bd008ecbf7dd7696137d3a2e0d9418ab25ee7468e5"
	case "boolean":
		return "1b2e12bc510ec7215f49e84b23cc00e4da3adadd7244030596671d7cb78af6f1",
			"a3f68051f964ee8b2917879e7fcec1d81b0c5bec20a975b2344410b08898e694"
	default:
		return "", ""
	}
}

func validTinyGoOperation(operation string) bool {
	switch operation {
	case "add", "subtract", "multiply", "less_than", "less_equal", "equal", "and", "or":
		return true
	default:
		return false
	}
}

func candidateOperation(fixture, candidateID string) string {
	if fixture == "arithmetic" {
		switch candidateID {
		case "sum":
			return "add"
		case "difference":
			return "subtract"
		case "product":
			return "multiply"
		}
	}
	if fixture == "boolean" {
		switch candidateID {
		case "less_than":
			return "less_than"
		case "less_equal":
			return "less_equal"
		case "equal":
			return "equal"
		}
	}
	return ""
}

func mappedProposal(fixture string, p plan, operation string) (string, *candidateEval) {
	for _, candidate := range p.Candidates {
		if candidateOperation(fixture, candidate.ID) != operation {
			continue
		}
		passed := 0
		for _, tc := range p.TestCases {
			if evalCandidate(candidate.ID, fixture, tc.Input) == tc.Expected {
				passed++
			}
		}
		return candidate.ID, &candidateEval{ID: candidate.ID, Expression: candidate.Expression, Passed: passed, Total: len(p.TestCases)}
	}
	return "", nil
}

func verifySuite(fixture string, p plan, fill bodyFillReport) *suiteSummary {
	result := &suiteSummary{Cases: append([]testCase(nil), p.TestCases...), CandidateScores: append([]score(nil), fill.CandidateScores...), ReportedCaseResults: append([]caseResult(nil), fill.CaseResults...)}
	for _, c := range p.Candidates {
		passed := 0
		for _, tc := range p.TestCases {
			if evalCandidate(c.ID, fixture, tc.Input) == tc.Expected {
				passed++
			}
		}
		result.IndependentCandidate = append(result.IndependentCandidate, candidateEval{ID: c.ID, Expression: c.Expression, Passed: passed, Total: len(p.TestCases)})
	}
	for _, tc := range p.TestCases {
		actual := evalCandidate(fill.SelectedCandidateID, fixture, tc.Input)
		result.IndependentSelected = append(result.IndependentSelected, caseResult{Input: tc.Input, Expected: tc.Expected, Actual: actual, Passed: actual == tc.Expected})
		if actual == tc.Expected {
			result.SelectedPassed++
		}
	}
	result.SelectedTotal = len(p.TestCases)
	if result.SelectedTotal != 0 {
		result.FunctionalAccuracy = float64(result.SelectedPassed) * 100 / float64(result.SelectedTotal)
	}
	return result
}

func validateEnvelope(payload cliEnvelope, receipt invocationReceipt, p plan, mode, variant, weightsSHA, metadataSHA string) []string {
	var problems []string
	add := func(ok bool, msg string) {
		if !ok {
			problems = append(problems, msg)
		}
	}
	r := payload.Report
	f := r.BodyFill
	add(r.Decision == "PASS", "CLI report decision is not PASS")
	add(r.CompilerSourceSHA == receipt.CompilerSourceSHA256, "reported compiler source SHA differs from the expected source pin")
	add(r.Completeness.Scope["compiler_source_sha"] == receipt.CompilerSourceSHA256, "completeness receipt does not bind the expected compiler source SHA")
	add(strings.HasPrefix(f.Decision.RequestSHA256, "sha256:") && isLowerHexDigest(strings.TrimPrefix(f.Decision.RequestSHA256, "sha256:"), 64), "decision receipt lacks a typed request digest")
	add(f.Decision.Mode == "deterministic_fallback" || f.Decision.Mode == "tiny_go", "decision receipt has an unknown mode")
	add(f.ProposedCandidateID == f.Decision.Selected, "body-fill proposal differs from provider receipt selection")
	if mode == "tiny_go" {
		add(f.Decision.Provider == "tiny_go" && (f.Decision.Mode == "tiny_go" || f.Decision.Mode == "deterministic_fallback"), "decision receipt is not a local tiny_go prediction or its explicit deterministic fallback")
		add(f.Decision.TinyGoVariant == variant, "decision variant differs from the loaded model variant")
		add(f.Decision.TinyGoWeightsSHA256 == weightsSHA, "decision receipt weights SHA differs from loaded weights")
		add(f.Decision.TinyGoMetadataSHA256 == metadataSHA, "decision receipt metadata SHA differs from loaded metadata")
		add(f.Decision.TinyGoPredictedOperation != "" && validTinyGoOperation(f.Decision.TinyGoPredictedOperation), "tiny_go raw closed operation is absent or invalid")
		add(f.Decision.TinyGoPredictionApplied != nil, "tiny_go prediction-applied flag is absent")
		add(f.LocalPredictions == 1 && f.ExternalCalls == 0 && f.ExternalCallsKnown, "provider counts differ from one local prediction and zero external calls")
		add(f.Timing.ModelLoadMS != nil && *f.Timing.ModelLoadMS >= 0 && f.Timing.DecisionMS >= 0 && f.Timing.TotalMS >= 0,
			"bodycodegen omitted nonnegative model-load, decision, or total timing")
		add(r.Completeness.Scope["decision_provider"] == "tiny_go" && r.Completeness.Scope["laya_provider"] == "", "completeness scope conflates tiny_go with Laya")
		if f.Decision.TinyGoPredictionApplied != nil {
			applied := *f.Decision.TinyGoPredictionApplied
			add(applied == (f.Decision.Mode == "tiny_go"), "explicit applied flag disagrees with tiny_go decision mode")
			if applied {
				mappedID, _ := mappedProposal(receipt.FixtureID, p, f.Decision.TinyGoPredictedOperation)
				add(mappedID != "" && f.ProposedCandidateID == mappedID, "applied tiny_go operation does not map to the body-fill proposal")
				add(f.Decision.FallbackReason == "", "applied tiny_go prediction unexpectedly has a fallback reason")
			} else {
				add(f.Decision.FallbackReason == "TINY_GO_LOW_CONFIDENCE" || f.Decision.FallbackReason == "TINY_GO_OPERATION_NOT_OFFERED", "non-applied tiny_go prediction lacks a recognized fallback reason")
				add(f.ProposedCandidateID == p.Candidates[0].ID, "fallback selection differs from the plan's deterministic first candidate")
			}
		}
		gold := goldOperation(receipt.FixtureID)
		add(receipt.GoldOperation == gold, "recorded gold operation differs from fixture contract")
		matches := f.Decision.TinyGoPredictedOperation == gold
		add(receipt.RawPredictionMatchesGold != nil && *receipt.RawPredictionMatchesGold == matches, "raw closed-label gold match differs from the explicit operation comparison")
		mappedID, mappedScore := mappedProposal(receipt.FixtureID, p, f.Decision.TinyGoPredictedOperation)
		add(receipt.MappedProposalCandidate == mappedID, "mapped proposal candidate does not correspond to the raw operation label")
		if mappedID == "" {
			add(receipt.MappedProposalScore == nil, "unoffered raw operation unexpectedly has a candidate score")
		} else {
			add(receipt.MappedProposalScore != nil && *receipt.MappedProposalScore == *mappedScore, "mapped proposal finite score differs from independently recomputed score")
		}
	} else {
		add(receipt.ExecutionMode == "offline_baseline", "baseline execution mode differs from its frozen plan")
		add(f.Decision.Provider == "deterministic" && f.Decision.Mode == "deterministic_fallback" && f.Decision.FallbackReason == "NOT_CONFIGURED", "offline baseline did not use the unconfigured deterministic route")
		add(f.Decision.TinyGoVariant == "" && f.Decision.TinyGoWeightsSHA256 == "" && f.Decision.TinyGoMetadataSHA256 == "", "offline baseline unexpectedly reports tiny_go model provenance")
		add(f.Decision.TinyGoPredictedOperation == "" && f.Decision.TinyGoPredictionApplied == nil, "offline baseline unexpectedly reports a tiny_go prediction")
		add(f.LocalPredictions == 0 && f.ExternalCalls == 0 && f.ExternalCallsKnown, "offline baseline provider counts are not known zero/zero")
		add(f.Timing.ModelLoadMS == nil && f.Timing.DecisionMS >= 0 && f.Timing.TotalMS >= 0, "offline baseline has model-load timing or invalid compiler timings")
		add(r.Completeness.Scope["decision_provider"] == "deterministic" && r.Completeness.Scope["laya_provider"] == "", "offline baseline completeness scope is not deterministic and local")
		add(receipt.GoldOperation == goldOperation(receipt.FixtureID) && receipt.RawPredictionMatchesGold == nil, "offline baseline unexpectedly records raw model label evidence")
		add(receipt.MappedProposalCandidate == "" && receipt.MappedProposalScore == nil, "offline baseline unexpectedly has a model-mapped proposal")
	}
	add(payload.Source != "", "CLI report omitted generated Go source")
	add(f.TestCasesTotal == len(p.TestCases), "selected finite denominator differs from the frozen plan")
	add(len(f.CandidateScores) == len(p.Candidates), "reported candidate score count differs from the frozen plan")
	add(len(f.CaseResults) == len(p.TestCases), "reported selected case result count differs from the frozen plan")
	expectedScores := map[string]candidateEval{}
	for _, c := range p.Candidates {
		passed := 0
		for _, tc := range p.TestCases {
			if evalCandidate(c.ID, receipt.FixtureID, tc.Input) == tc.Expected {
				passed++
			}
		}
		expectedScores[c.ID] = candidateEval{ID: c.ID, Expression: c.Expression, Passed: passed, Total: len(p.TestCases)}
	}
	for i, observed := range f.CandidateScores {
		if i >= len(p.Candidates) {
			break
		}
		want := expectedScores[p.Candidates[i].ID]
		add(observed.ID == want.ID && observed.Expression == want.Expression && observed.TypecheckPassed &&
			observed.TestCasesPassed == want.Passed && observed.TestCasesTotal == want.Total &&
			near(observed.AccuracyPercent, percent(want.Passed, want.Total)), "candidate score differs from the independent finite oracle")
	}
	selectedExists, bestID, bestPass := false, "", -1
	for _, c := range p.Candidates {
		s := expectedScores[c.ID]
		if s.Passed > bestPass {
			bestPass, bestID = s.Passed, c.ID
		}
		if c.ID == f.SelectedCandidateID {
			selectedExists = true
		}
	}
	add(selectedExists, "selected candidate is not declared in the frozen plan")
	selectedPass := -1
	if s, ok := expectedScores[f.SelectedCandidateID]; ok {
		selectedPass = s.Passed
	}
	add(selectedPass >= 0 && f.TestCasesPassed == selectedPass && near(f.FunctionalAccuracy, percent(selectedPass, len(p.TestCases))), "selected finite score differs from the independent oracle")
	add(f.BestCandidateID == bestID && near(f.BestAccuracy, percent(bestPass, len(p.TestCases))), "best candidate summary differs from recomputed candidate scores")
	wantSelected, wantAdjustment := f.ProposedCandidateID, "proposal_retained"
	if proposed, ok := expectedScores[f.ProposedCandidateID]; ok && proposed.Passed < bestPass {
		wantSelected, wantAdjustment = bestID, "replaced_with_best_scoring_candidate"
	}
	add(f.SelectedCandidateID == wantSelected && f.Adjustment == wantAdjustment, "final selection does not follow finite-score correction policy")
	if f.Decision.Mode == "deterministic_fallback" {
		add(f.Decision.FallbackReason != "", "deterministic tiny_go fallback omitted its reason")
	} else {
		add(f.Decision.FallbackReason == "", "non-fallback tiny_go decision unexpectedly reports a fallback reason")
	}
	if proposed, ok := expectedScores[f.ProposedCandidateID]; ok {
		add(near(f.ProposedAccuracy, percent(proposed.Passed, len(p.TestCases))), "proposed finite score differs from the independent oracle")
		add(near(f.Regret, percent(bestPass-proposed.Passed, len(p.TestCases))), "selection regret differs from the independent oracle")
	} else {
		problems = append(problems, "raw model proposal is not one of the frozen candidate IDs")
	}
	for i, observed := range f.CaseResults {
		if i >= len(p.TestCases) {
			break
		}
		tc := p.TestCases[i]
		actual := evalCandidate(f.SelectedCandidateID, receipt.FixtureID, tc.Input)
		add(observed.Input == tc.Input && observed.Expected == tc.Expected && observed.Actual == actual && observed.Passed == (actual == tc.Expected), "selected case result differs from the independent oracle")
	}
	seenExternal, seenLaya := false, false
	for _, d := range r.Completeness.Dimensions {
		if d.ID == "external_network_boundary" {
			seenExternal = d.Status == "PASS" && d.Numerator == 1 && d.Denominator == 1
		}
		if d.ID == "laya_decision_observation" {
			seenLaya = d.Denominator == 0 && d.Numerator == 0
		}
	}
	add(seenExternal, "completeness receipt does not show a passing zero-external-network boundary")
	add(seenLaya, "completeness receipt counts tiny_go as a Laya decision")
	return problems
}

func summarizeDecision(f bodyFillReport) *decisionSummary {
	proposalScoreSemantics := "unclassified"
	switch f.Decision.Provider {
	case "tiny_go":
		if f.Decision.TinyGoPredictionApplied != nil && *f.Decision.TinyGoPredictionApplied {
			proposalScoreSemantics = "model_applied_operation_candidate"
		} else {
			proposalScoreSemantics = "fallback_candidate_score_not_model_accuracy"
		}
	case "deterministic":
		proposalScoreSemantics = "deterministic_no_model_fallback_score"
	}
	return &decisionSummary{
		Provider: f.Decision.Provider, Mode: f.Decision.Mode, FallbackReason: f.Decision.FallbackReason,
		ProposedCandidateID: f.ProposedCandidateID, ProposedAccuracy: f.ProposedAccuracy,
		ProposedScoreSemantics: proposalScoreSemantics,
		SelectedCandidateID:    f.SelectedCandidateID, BestCandidateID: f.BestCandidateID,
		BestAccuracy: f.BestAccuracy, SelectionRegret: f.Regret,
		SelectionAdjustment: f.Adjustment, TinyGoVariant: f.Decision.TinyGoVariant,
		TinyGoWeightsSHA256:      f.Decision.TinyGoWeightsSHA256,
		TinyGoMetadataSHA256:     f.Decision.TinyGoMetadataSHA256,
		TinyGoPredictedOperation: f.Decision.TinyGoPredictedOperation,
		TinyGoPredictionApplied:  f.Decision.TinyGoPredictionApplied,
		RequestSHA256:            f.Decision.RequestSHA256,
		LocalPredictions:         f.LocalPredictions, ExternalCalls: f.ExternalCalls,
		ExternalCallsKnown: f.ExternalCallsKnown, TinyModelLoadMS: f.Timing.ModelLoadMS,
		TinyDecisionMS: f.Timing.DecisionMS, BodycodegenTotalMS: f.Timing.TotalMS,
	}
}

func summarizeCompleteness(scope map[string]any, dims []dimension) *completenessView {
	decisionProvider, _ := scope["decision_provider"].(string)
	layaProvider, _ := scope["laya_provider"].(string)
	return &completenessView{DecisionProvider: decisionProvider, LayaProvider: layaProvider, Dimensions: append([]dimension(nil), dims...)}
}

func semanticPlanSHA(p plan) string { data, _ := json.Marshal(p); return sha256Hex(data) }
func percent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) * 100 / float64(total)
}
func near(a, b float64) bool    { return math.Abs(a-b) < 1e-9 }
func sha256Hex(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func isLowerHexDigest(s string, size int) bool {
	if len(s) != size || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func decodeOneJSON(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
func decodeStrictOneJSON(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
func withoutLayaEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, item := range env {
		name, _, ok := strings.Cut(item, "=")
		if ok && (name == "GOOO_LAYA_URL" || name == "GOOO_LAYA_API_KEY") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}
func mkdirFresh(path string) error {
	if err := os.Mkdir(path, 0700); err != nil {
		return fmt.Errorf("output directory must be new and empty: %w", err)
	}
	return nil
}
func writeFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(data, '\n'))
}
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}
func timevalMillis(tv syscall.Timeval) float64 { return float64(tv.Sec)*1000 + float64(tv.Usec)/1000 }
func rssUnit() string {
	if runtime.GOOS == "darwin" {
		return "bytes"
	}
	return "kilobytes"
}
