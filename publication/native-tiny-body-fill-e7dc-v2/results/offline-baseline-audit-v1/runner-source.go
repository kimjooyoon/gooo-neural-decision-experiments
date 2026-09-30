// Versioned independent audit for the two saved no-model baseline captures.
// It preflights both raw rows against the native receipt contract before it
// compiles or executes either generated program.
package main

import (
	"bytes"
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
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	planSHA256          = "1c5a86b321eacf9549d03f4b7df2c94f8ea8171b7dbb04f384094895869c3a58"
	compilerSourceSHA   = "e7dc8198600f27fa3d6fc07d2a8fd8b6a4af3b27"
	compilerBinarySHA   = "5e7f75badf8079d15d971bc05f65c08516a9b79c4bb0539036ccb318c093f79c"
	captureSourceSHA    = "a23c785113a4eacec67c29317e1091af2a655b50e87b2ad871079a88a4e9ffeb"
	captureBinarySHA    = "fe1b55cc8cd513e40d8d6bba22d4746c59c256df4b66f698c87dff6b76fac7a6"
	baseReplaySourceSHA = "d3aa7f2e07c681203fc57b40cea94d3bac468f579189bf9428fc932ec997174d"
	baseReplayBinarySHA = "3e5fd623e3f5f5d960b7821c75f23bb1eb3d90fe94ad43282290649bf58783d0"
	cellCases           = 6
	plannedCells        = 2
	plannedCases        = cellCases * plannedCells
)

var knownCaptureErrors = []string{
	"offline baseline completeness scope is not deterministic and local",
	"completeness receipt counts tiny_go as a Laya decision",
}

type candidate struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
}

type testCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}

type planFile struct {
	Candidates []candidate `json:"candidates"`
	TestCases  []testCase  `json:"test_cases"`
}

type dimension struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Numerator   int    `json:"numerator"`
	Denominator int    `json:"denominator"`
}

type completenessScope struct {
	DecisionProvider string `json:"decision_provider"`
	LayaProvider     string `json:"laya_provider"`
	DecisionMode     string `json:"decision_mode"`
	LayaMode         string `json:"laya_mode"`
	CompilerSource   string `json:"compiler_source_sha"`
	PlanSHA256       string `json:"plan_sha256"`
}

type completenessReceipt struct {
	Decision   string            `json:"decision"`
	Scope      completenessScope `json:"scope"`
	Dimensions []dimension       `json:"dimensions"`
}

type receiptDecision struct {
	Provider            string `json:"provider"`
	Mode                string `json:"mode"`
	FallbackReason      string `json:"fallback_reason"`
	ProposedCandidateID string `json:"proposed_candidate_id"`
	SelectedCandidateID string `json:"selected_candidate_id"`
	RequestSHA256       string `json:"request_sha256"`
	LocalPredictions    int    `json:"local_model_predictions"`
	ExternalCalls       int    `json:"external_provider_calls"`
	ExternalCallsKnown  bool   `json:"external_provider_calls_known"`
}

type captureCompleteness struct {
	DecisionProvider string      `json:"decision_provider"`
	LayaProvider     string      `json:"laya_provider"`
	Dimensions       []dimension `json:"dimensions"`
}

type captureReceipt struct {
	Schema                   string              `json:"schema"`
	Status                   string              `json:"status"`
	ExecutionMode            string              `json:"execution_mode"`
	FixtureID                string              `json:"fixture_id"`
	Activity                 string              `json:"activity"`
	FixtureFile              string              `json:"fixture_file"`
	FixtureSHA256            string              `json:"fixture_sha256"`
	PlanFile                 string              `json:"plan_file"`
	PlanFileSHA256           string              `json:"plan_file_sha256"`
	PlanSemanticSHA256       string              `json:"plan_semantic_sha256"`
	RunPlanSHA256            string              `json:"run_plan_sha256"`
	CompilerSourceExpected   string              `json:"compiler_source_sha256_expected"`
	CompilerSourceObserved   string              `json:"compiler_source_sha256_observed"`
	CLIBinaryExpected        string              `json:"cli_binary_sha256_expected"`
	CLIBinaryObserved        string              `json:"cli_binary_sha256_observed"`
	CLIBinaryAfter           string              `json:"cli_binary_sha256_after"`
	RunnerSourceSHA256       string              `json:"runner_source_sha256"`
	RunnerBinarySHA256       string              `json:"runner_binary_sha256"`
	ModelVariant             string              `json:"model_variant"`
	ModelMetadataSHA256      string              `json:"model_metadata_sha256"`
	ModelMetadataSHA256After string              `json:"model_metadata_sha256_after"`
	ModelWeightsSHA256       string              `json:"model_weights_sha256"`
	ModelWeightsSHA256After  string              `json:"model_weights_sha256_after"`
	ModelMetadataFile        string              `json:"model_metadata_file"`
	LayaURLEnv               string              `json:"laya_url_child_environment"`
	LayaKeyEnv               string              `json:"laya_key_child_environment"`
	ChildCount               int                 `json:"child_count"`
	Command                  []string            `json:"path_neutral_command"`
	ExitCode                 int                 `json:"exit_code"`
	StdoutBytes              int                 `json:"stdout_bytes"`
	StdoutSHA256             string              `json:"stdout_sha256"`
	StderrBytes              int                 `json:"stderr_bytes"`
	StderrSHA256             string              `json:"stderr_sha256"`
	ValidationErrors         []string            `json:"validation_errors"`
	Decision                 receiptDecision     `json:"decision"`
	Completeness             captureCompleteness `json:"completeness"`
}

type bodyFillDecision struct {
	Provider       string `json:"provider"`
	Mode           string `json:"mode"`
	FallbackReason string `json:"fallback_reason"`
	Selected       string `json:"selected"`
	RequestSHA256  string `json:"request_sha256"`
}

type bodyFill struct {
	IRPlanSHA256               string          `json:"ir_plan_sha256"`
	ProposedCandidateID        string          `json:"proposed_candidate_id"`
	SelectedCandidateID        string          `json:"selected_candidate_id"`
	TestCasesPassed            int             `json:"test_cases_passed"`
	TestCasesTotal             int             `json:"test_cases_total"`
	LocalModelPredictions      int             `json:"local_model_predictions"`
	ExternalProviderCalls      int             `json:"external_provider_calls"`
	ExternalProviderCallsKnown bool            `json:"external_provider_calls_known"`
	Decision                   json.RawMessage `json:"decision"`
}

type rawReport struct {
	Decision          string              `json:"decision"`
	CompilerSourceSHA string              `json:"compiler_source_sha"`
	PlanSHA256        string              `json:"plan_sha256"`
	GeneratedDigest   string              `json:"generated_digest"`
	Completeness      completenessReceipt `json:"completeness_receipt"`
	BodyFill          bodyFill            `json:"body_fill"`
}

type rawCLI struct {
	Report rawReport `json:"report"`
	Source string    `json:"source"`
}

type captureSummary struct {
	NativeSource string `json:"native_source_commit"`
	NativeBinary string `json:"native_cli_binary_sha256"`
	RunPlanSHA   string `json:"run_plan_sha256"`
	Rows         []struct {
		CellID           string   `json:"cell_id"`
		WrapperExitCode  int      `json:"wrapper_exit_code"`
		CLIExitCode      int      `json:"cli_exit_code"`
		CaptureStatus    string   `json:"capture_status"`
		ValidationErrors []string `json:"validation_errors"`
	} `json:"rows"`
}

type caseResult struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
	Passed   bool  `json:"passed"`
}

type preflightCell struct {
	CellID                  string     `json:"cell_id"`
	FixtureID               string     `json:"fixture_id"`
	CaptureStatus           string     `json:"capture_status"`
	KnownValidationErrors   []string   `json:"known_validation_errors"`
	WrapperExitCode         int        `json:"wrapper_exit_code"`
	CLIExitCode             int        `json:"cli_exit_code"`
	RawCLISHA256            string     `json:"raw_cli_sha256"`
	CaptureReceiptSHA256    string     `json:"capture_receipt_sha256"`
	FixtureSHA256           string     `json:"fixture_sha256"`
	PlanFileSHA256          string     `json:"plan_file_sha256"`
	PlanSemanticSHA256      string     `json:"plan_semantic_sha256"`
	GeneratedSourceSHA256   string     `json:"generated_source_sha256"`
	LayaProvider            string     `json:"laya_provider"`
	LayaObservation         dimension  `json:"laya_decision_observation"`
	ExternalNetworkBoundary dimension  `json:"external_network_boundary"`
	ProviderAccounting      dimension  `json:"provider_execution_accounting"`
	LocalPredictions        int        `json:"local_model_predictions"`
	ExternalProviderCalls   int        `json:"external_provider_calls"`
	ExternalProviderKnown   bool       `json:"external_provider_calls_known"`
	ModelMetadataSHA256     string     `json:"model_metadata_sha256,omitempty"`
	ModelWeightsSHA256      string     `json:"model_weights_sha256,omitempty"`
	ProposedCandidateID     string     `json:"proposed_candidate_id"`
	SelectedCandidateID     string     `json:"selected_candidate_id"`
	Errors                  []string   `json:"errors"`
	GeneratedSource         string     `json:"-"`
	Cases                   []testCase `json:"-"`
	Activity                string     `json:"-"`
}

type caseAudit struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
	Passed   bool  `json:"passed"`
}

type cellAudit struct {
	CellID                  string      `json:"cell_id"`
	Status                  string      `json:"status"`
	FixtureID               string      `json:"fixture_id"`
	CaptureStatus           string      `json:"original_capture_status"`
	CaptureValidationErrors []string    `json:"original_capture_validation_errors"`
	KnownErrorAdjudication  string      `json:"known_error_adjudication"`
	PlannedCases            int         `json:"planned_cases"`
	ObservedCases           int         `json:"observed_cases"`
	PassedCases             int         `json:"passed_cases"`
	FailedCases             int         `json:"failed_cases"`
	Cases                   []caseAudit `json:"cases,omitempty"`
	RawCLISHA256            string      `json:"raw_cli_sha256"`
	CaptureReceiptSHA256    string      `json:"capture_receipt_sha256"`
	GeneratedSourceSHA256   string      `json:"generated_source_sha256"`
	GoBuildExitCode         int         `json:"go_build_exit_code"`
	GoBuildWallMS           float64     `json:"go_build_wall_ms"`
	GoBuildStdoutSHA256     string      `json:"go_build_stdout_sha256"`
	GoBuildStderrSHA256     string      `json:"go_build_stderr_sha256"`
	ExecutionExitCode       int         `json:"generated_execution_exit_code"`
	ExecutionWallMS         float64     `json:"generated_execution_wall_ms"`
	ExecutionStdoutSHA256   string      `json:"generated_execution_stdout_sha256"`
	ExecutionStderrSHA256   string      `json:"generated_execution_stderr_sha256"`
	Errors                  []string    `json:"errors"`
}

type auditReport struct {
	Schema                 string          `json:"schema"`
	Status                 string          `json:"status"`
	AuditorSourceSHA256    string          `json:"auditor_source_sha256"`
	AuditorBinarySHA256    string          `json:"auditor_binary_sha256"`
	CaptureSourceSHA256    string          `json:"capture_runner_source_sha256"`
	CaptureBinarySHA256    string          `json:"capture_runner_binary_sha256"`
	BaseReplaySourceSHA256 string          `json:"base_replay_runner_source_sha256"`
	BaseReplayBinarySHA256 string          `json:"base_replay_runner_binary_sha256"`
	CompilerSourceSHA256   string          `json:"compiler_source_sha256"`
	CLIBinarySHA256        string          `json:"cli_binary_sha256"`
	RunPlanSHA256          string          `json:"run_plan_sha256"`
	GoVersion              string          `json:"go_version"`
	PlannedCells           int             `json:"planned_cells"`
	ObservedCells          int             `json:"observed_cells"`
	PassedCells            int             `json:"passed_cells"`
	CasesPlanned           int             `json:"cases_planned"`
	CasesObserved          int             `json:"cases_observed"`
	CasesPassed            int             `json:"cases_passed"`
	CasesFailed            int             `json:"cases_failed"`
	CasesUnknown           int             `json:"cases_unknown_from_planned_denominator"`
	PreflightStatus        string          `json:"preflight_status"`
	PreflightErrors        []string        `json:"preflight_errors"`
	CaptureSummarySHA256   string          `json:"capture_summary_sha256"`
	Scope                  string          `json:"scope"`
	Cells                  []cellAudit     `json:"cells"`
	PreflightCells         []preflightCell `json:"preflight_cells"`
}

type preparedCell struct {
	Preflight preflightCell
	Receipt   captureReceipt
	Raw       rawCLI
}

func main() {
	var (
		experimentDir = flag.String("experiment-dir", "", "absolute v2 experiment directory")
		runDir        = flag.String("run-dir", "", "absolute directory containing original results/<cell> captures")
		fixtureDir    = flag.String("fixture-dir", "", "absolute directory containing frozen fixtures and plans")
		goBinary      = flag.String("go", "", "absolute Go 1.27 binary")
		cliBinary     = flag.String("cli-binary", "", "absolute pinned native CLI binary")
		outputDir     = flag.String("output-dir", "", "new output directory for baseline audit")
		sourcePath    = flag.String("runner-source", "", "absolute path to this audit runner source")
		sourceSHA     = flag.String("runner-source-sha256", "", "SHA-256 of this exact audit runner source")
	)
	flag.Parse()
	if err := run(*experimentDir, *runDir, *fixtureDir, *goBinary, *cliBinary, *outputDir, *sourcePath, *sourceSHA); err != nil {
		fmt.Fprintln(os.Stderr, "offline baseline audit:", err)
		os.Exit(2)
	}
}

func run(experimentDir, runDir, fixtureDir, goBinary, cliBinary, outputDir, sourcePath, sourceSHA string) error {
	for label, value := range map[string]string{
		"experiment directory": experimentDir, "run directory": runDir, "fixture directory": fixtureDir,
		"Go binary": goBinary, "CLI binary": cliBinary, "output directory": outputDir,
		"runner source": sourcePath, "runner source SHA-256": sourceSHA,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	for label, value := range map[string]string{"experiment directory": experimentDir, "run directory": runDir, "fixture directory": fixtureDir, "Go binary": goBinary, "CLI binary": cliBinary, "runner source": sourcePath} {
		if !filepath.IsAbs(value) {
			return fmt.Errorf("%s must be absolute", label)
		}
	}
	if !isHex(sourceSHA, 64) {
		return errors.New("runner source SHA-256 must be 64 lowercase hexadecimal characters")
	}
	if _, err := os.Stat(outputDir); err == nil {
		return errors.New("output directory must be new")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output directory: %w", err)
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return errors.New("this offline audit is pinned to the Darwin/arm64 capture host")
	}

	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil || sha256Hex(sourceBytes) != sourceSHA {
		return errors.New("audit runner source bytes do not match the supplied archive pin")
	}
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate audit runner binary: %w", err)
	}
	auditorBytes, err := os.ReadFile(executablePath)
	if err != nil {
		return fmt.Errorf("read audit runner binary: %w", err)
	}
	cliBytes, err := os.ReadFile(cliBinary)
	if err != nil || sha256Hex(cliBytes) != compilerBinarySHA {
		return errors.New("native CLI binary differs from the source-bound capture pin")
	}
	goVersionBytes, err := exec.Command(goBinary, "version").Output()
	if err != nil {
		return fmt.Errorf("read Go version: %w", err)
	}
	goVersion := strings.TrimSpace(string(goVersionBytes))
	if goVersion != "go version go1.27.0 darwin/arm64" {
		return fmt.Errorf("unexpected Go version: %s", goVersion)
	}
	if err := verifyFilePin(experimentDir, "capture.go", captureSourceSHA); err != nil {
		return err
	}
	if err := verifyFilePin(experimentDir, "capture-runner", captureBinarySHA); err != nil {
		return err
	}
	if err := verifyFilePin(experimentDir, "replay.go", baseReplaySourceSHA); err != nil {
		return err
	}
	if err := verifyFilePin(experimentDir, "replay-runner", baseReplayBinarySHA); err != nil {
		return err
	}
	if err := verifyFilePin(experimentDir, "run-plan.json", planSHA256); err != nil {
		return err
	}
	for fixture, pins := range map[string][2]string{
		"arithmetic": {"7896b7f65db047bf077a03920901ff4487576582896f492fabe5d1fc82c1d748", "685b22ffb073ad0e8fb597bd008ecbf7dd7696137d3a2e0d9418ab25ee7468e5"},
		"boolean":    {"1b2e12bc510ec7215f49e84b23cc00e4da3adadd7244030596671d7cb78af6f1", "a3f68051f964ee8b2917879e7fcec1d81b0c5bec20a975b2344410b08898e694"},
	} {
		if err := verifyFilePin(fixtureDir, fixture+".gooo", pins[0]); err != nil {
			return err
		}
		if err := verifyFilePin(fixtureDir, fixture+".plan.json", pins[1]); err != nil {
			return err
		}
	}
	if err := verifyFilePin(experimentDir, filepath.Join("results", "capture-summary.json"), "405f782ee8866be0bf6b2f4e342b2a6986783295d653436fe95a8bb566dd3e57"); err != nil {
		return err
	}
	if err := verifyFilePin(experimentDir, filepath.Join("results", "capture-evidence-manifest.json"), "69cb3f5c8db02437e84c4d5266370f46674eac52a2442ee7c02510fe8b8162b3"); err != nil {
		return err
	}
	var summary captureSummary
	summaryBytes, err := os.ReadFile(filepath.Join(experimentDir, "results", "capture-summary.json"))
	if err != nil || decodeStrictJSON(summaryBytes, &summary) != nil {
		return errors.New("capture summary is unavailable or invalid")
	}
	if summary.NativeSource != compilerSourceSHA || summary.NativeBinary != compilerBinarySHA || summary.RunPlanSHA != planSHA256 {
		return errors.New("capture summary source, binary, or run-plan provenance differs from frozen pins")
	}
	if summaryCell := findSummaryRow(summary, "arithmetic-offline_baseline"); summaryCell == nil || summaryCell.WrapperExitCode != 2 || summaryCell.CLIExitCode != 0 || summaryCell.CaptureStatus != "CAPTURED_WITH_VALIDATION_ERRORS" || !sameStrings(summaryCell.ValidationErrors, knownCaptureErrors) {
		return errors.New("arithmetic wrapper/capture summary does not match the exact known validation issue")
	}
	if summaryCell := findSummaryRow(summary, "boolean-offline_baseline"); summaryCell == nil || summaryCell.WrapperExitCode != 2 || summaryCell.CLIExitCode != 0 || summaryCell.CaptureStatus != "CAPTURED_WITH_VALIDATION_ERRORS" || !sameStrings(summaryCell.ValidationErrors, knownCaptureErrors) {
		return errors.New("boolean wrapper/capture summary does not match the exact known validation issue")
	}

	preflight := make([]preparedCell, 0, plannedCells)
	for _, cellID := range []string{"arithmetic-offline_baseline", "boolean-offline_baseline"} {
		preflight = append(preflight, preflightOfflineCell(runDir, fixtureDir, cellID)...)
	}
	preflightErrors := []string{}
	for i := range preflight {
		if len(preflight[i].Preflight.Errors) != 0 {
			preflightErrors = append(preflightErrors, preflight[i].Preflight.CellID+": "+strings.Join(preflight[i].Preflight.Errors, "; "))
		}
	}

	if err := os.Mkdir(outputDir, 0700); err != nil {
		return fmt.Errorf("create fresh audit output directory: %w", err)
	}
	if err := writeFile(filepath.Join(outputDir, "runner-source.go"), sourceBytes); err != nil {
		return err
	}
	report := auditReport{
		Schema: "gooo/native-tiny-go-offline-baseline-audit/v1", Status: "PREFLIGHT_REJECTED_NO_GENERATED_COMPILE",
		AuditorSourceSHA256: sourceSHA, AuditorBinarySHA256: sha256Hex(auditorBytes), CaptureSourceSHA256: captureSourceSHA,
		CaptureBinarySHA256: captureBinarySHA, BaseReplaySourceSHA256: baseReplaySourceSHA, BaseReplayBinarySHA256: baseReplayBinarySHA,
		CompilerSourceSHA256: compilerSourceSHA, CLIBinarySHA256: compilerBinarySHA, RunPlanSHA256: planSHA256,
		GoVersion: goVersion, PlannedCells: plannedCells, CasesPlanned: plannedCases,
		PreflightStatus: "FAILED", PreflightErrors: preflightErrors, CaptureSummarySHA256: sha256Hex(summaryBytes),
		Scope: "adjudicate only the two saved offline_baseline rows that have exactly the known completeness-validator mismatch; independently verify raw no-provider route and completeness, then compile and execute their generated Go against six frozen finite cases each; no model, native CLI, or provider calls",
	}
	for _, item := range preflight {
		report.PreflightCells = append(report.PreflightCells, item.Preflight)
	}
	if len(preflightErrors) != 0 {
		if err := writeJSON(filepath.Join(outputDir, "preflight-report.json"), report); err != nil {
			return err
		}
		return errors.New("preflight rejected at least one raw capture; no generated-source compile or execution was started")
	}
	report.PreflightStatus = "PASS_EXACT_KNOWN_COMPLETENESS_MISMATCH"
	if err := writeJSON(filepath.Join(outputDir, "preflight-report.json"), report); err != nil {
		return err
	}

	for _, item := range preflight {
		cell := compileAndRun(goBinary, outputDir, item)
		report.Cells = append(report.Cells, cell)
		report.ObservedCells++
		report.CasesObserved += cell.ObservedCases
		report.CasesPassed += cell.PassedCases
		report.CasesFailed += cell.FailedCases
		if cell.Status == "PASS_WITH_KNOWN_CAPTURE_VALIDATION_MISMATCH" {
			report.PassedCells++
		} else {
			report.Status = "PARTIAL"
		}
	}
	if report.Status != "PARTIAL" {
		report.Status = "PASS_WITH_KNOWN_CAPTURE_VALIDATION_MISMATCH"
	}
	if report.CasesObserved <= report.CasesPlanned {
		report.CasesUnknown = report.CasesPlanned - report.CasesObserved
	}
	if err := writeJSON(filepath.Join(outputDir, "audit-report.json"), report); err != nil {
		return err
	}
	if report.Status == "PARTIAL" {
		return errors.New("one or more generated baseline programs failed independent Go replay; raw evidence retained")
	}
	return nil
}

func preflightOfflineCell(runDir, fixtureDir, cellID string) []preparedCell {
	parts := strings.Split(cellID, "-")
	fixture := parts[0]
	pre := preflightCell{CellID: cellID, FixtureID: fixture, CaptureStatus: "UNKNOWN", WrapperExitCode: -1, CLIExitCode: -1,
		LayaObservation: dimension{ID: "laya_decision_observation", Status: "UNKNOWN", Numerator: 0, Denominator: 1}}
	prepared := preparedCell{Preflight: pre}
	cellDir := filepath.Join(runDir, cellID)
	captureBytes, err := os.ReadFile(filepath.Join(cellDir, "invocation.json"))
	if err != nil {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "original invocation receipt missing")
		return []preparedCell{prepared}
	}
	prepared.Preflight.CaptureReceiptSHA256 = sha256Hex(captureBytes)
	var capture captureReceipt
	if err := decodeStrictJSON(captureBytes, &capture); err != nil {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "original invocation receipt is invalid JSON")
		return []preparedCell{prepared}
	}
	prepared.Receipt = capture
	prepared.Preflight.CaptureStatus = capture.Status
	prepared.Preflight.CLIExitCode = capture.ExitCode
	prepared.Preflight.KnownValidationErrors = append([]string(nil), capture.ValidationErrors...)
	if capture.Status != "CAPTURED_WITH_VALIDATION_ERRORS" || !sameStrings(capture.ValidationErrors, knownCaptureErrors) {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "capture status or complete validation-error list differs from the exact known issue")
	}
	if capture.Schema != "gooo/native-tiny-go-body-smoke/v2" || capture.ExecutionMode != "offline_baseline" || capture.FixtureID != fixture ||
		capture.FixtureFile != fixture+".gooo" || capture.PlanFile != fixture+".plan.json" || capture.Activity != expectedActivity(fixture) ||
		capture.RunPlanSHA256 != planSHA256 || capture.CompilerSourceExpected != compilerSourceSHA || capture.CompilerSourceObserved != compilerSourceSHA ||
		capture.CLIBinaryExpected != compilerBinarySHA || capture.CLIBinaryObserved != compilerBinarySHA || capture.CLIBinaryAfter != compilerBinarySHA ||
		capture.RunnerSourceSHA256 != captureSourceSHA || capture.RunnerBinarySHA256 != captureBinarySHA ||
		capture.ChildCount != 1 || capture.ExitCode != 0 || capture.LayaURLEnv != "unset" || capture.LayaKeyEnv != "unset" {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "capture identity, source, binary, one-child, CLI-success, or provider-environment binding failed")
	}
	if capture.ModelVariant != "" || capture.ModelMetadataSHA256 != "" || capture.ModelMetadataSHA256After != "" || capture.ModelWeightsSHA256 != "" || capture.ModelWeightsSHA256After != "" || capture.ModelMetadataFile != "" {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "offline baseline capture contains model provenance")
	}
	if capture.Decision.Provider != "deterministic" || capture.Decision.Mode != "deterministic_fallback" || capture.Decision.FallbackReason != "NOT_CONFIGURED" ||
		capture.Decision.LocalPredictions != 0 || capture.Decision.ExternalCalls != 0 || !capture.Decision.ExternalCallsKnown {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "invocation summary does not prove deterministic no-provider route and counters")
	}
	if capture.Completeness.DecisionProvider != "deterministic" || capture.Completeness.LayaProvider != "deterministic" {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "invocation completeness view does not identify deterministic provider route")
	}
	if d, err := findDimension(capture.Completeness.Dimensions, "laya_decision_observation"); err != nil || d.Status != "UNKNOWN" || d.Numerator != 0 || d.Denominator != 1 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "invocation completeness Laya observation is not UNKNOWN 0/1")
	} else {
		prepared.Preflight.LayaObservation = d
	}
	prepared.Preflight.LayaProvider = capture.Completeness.LayaProvider

	rawBytes, err := os.ReadFile(filepath.Join(cellDir, "raw-cli.json"))
	if err != nil {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw CLI JSON missing")
		return []preparedCell{prepared}
	}
	prepared.Preflight.RawCLISHA256 = sha256Hex(rawBytes)
	if capture.StdoutBytes != len(rawBytes) || capture.StdoutSHA256 != prepared.Preflight.RawCLISHA256 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw CLI bytes differ from capture stdout size or SHA")
	}
	stderrBytes, err := os.ReadFile(filepath.Join(cellDir, "raw-stderr.txt"))
	if err != nil || len(stderrBytes) != capture.StderrBytes || sha256Hex(stderrBytes) != capture.StderrSHA256 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw CLI stderr bytes differ from invocation size or SHA")
	}
	var raw rawCLI
	if err := decodeStrictJSON(rawBytes, &raw); err != nil {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw CLI JSON contains duplicates or is invalid")
		return []preparedCell{prepared}
	}
	prepared.Raw = raw
	if raw.Report.Decision != "PASS" || raw.Report.CompilerSourceSHA != compilerSourceSHA || raw.Source == "" || sha256Hex([]byte(raw.Source)) == "" ||
		raw.Report.GeneratedDigest != "sha256:"+sha256Hex([]byte(raw.Source)) {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw report does not bind successful source-pinned generated output bytes")
	}
	if capture.CompilerSourceObserved != raw.Report.CompilerSourceSHA {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw compiler source differs from invocation receipt")
	}
	planBytes, err := os.ReadFile(filepath.Join(fixtureDir, fixture+".plan.json"))
	if err != nil || sha256Hex(planBytes) != capture.PlanFileSHA256 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "frozen plan bytes differ from the capture receipt")
		return []preparedCell{prepared}
	}
	fixtureBytes, err := os.ReadFile(filepath.Join(fixtureDir, fixture+".gooo"))
	if err != nil || sha256Hex(fixtureBytes) != capture.FixtureSHA256 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "frozen fixture bytes differ from the capture receipt")
		return []preparedCell{prepared}
	}
	prepared.Preflight.FixtureSHA256 = sha256Hex(fixtureBytes)
	prepared.Preflight.PlanFileSHA256 = sha256Hex(planBytes)
	var plan planFile
	if err := decodeStrictJSON(planBytes, &plan); err != nil || len(plan.TestCases) != cellCases || len(plan.Candidates) == 0 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "frozen plan JSON is invalid or not six cases")
		return []preparedCell{prepared}
	}
	prepared.Preflight.PlanSemanticSHA256 = capture.PlanSemanticSHA256
	prepared.Preflight.GeneratedSource = raw.Source
	prepared.Preflight.Cases = plan.TestCases
	prepared.Preflight.Activity = capture.Activity

	semantic := "sha256:" + capture.PlanSemanticSHA256
	if capture.PlanSemanticSHA256 == "" || raw.Report.BodyFill.IRPlanSHA256 != semantic || raw.Report.PlanSHA256 != expectedReportPlanSHA(fixture) {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw report IR-plan or frozen run-plan digest mismatch")
	}
	prepared.Preflight.GeneratedSourceSHA256 = sha256Hex([]byte(raw.Source))
	if raw.Report.Completeness.Scope.DecisionProvider != "deterministic" || raw.Report.Completeness.Scope.LayaProvider != "deterministic" ||
		raw.Report.Completeness.Scope.DecisionMode != "deterministic_fallback" || raw.Report.Completeness.Scope.LayaMode != "deterministic_fallback" ||
		raw.Report.Completeness.Scope.CompilerSource != compilerSourceSHA || raw.Report.Completeness.Scope.PlanSHA256 != raw.Report.PlanSHA256 ||
		raw.Report.Completeness.Decision != "PASS_WITHIN_DECLARED_FIXTURE_SCOPE" {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw completeness scope does not satisfy deterministic offline source-bound contract")
	}
	if laya, err := findDimension(raw.Report.Completeness.Dimensions, "laya_decision_observation"); err != nil || laya.Status != "UNKNOWN" || laya.Numerator != 0 || laya.Denominator != 1 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw completeness Laya observation is not UNKNOWN 0/1")
	} else if laya != prepared.Preflight.LayaObservation {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw and invocation completeness Laya observations differ")
	}
	for _, spec := range []struct {
		id, status string
		num, den   int
	}{
		{"external_network_boundary", "PASS", 1, 1},
		{"provider_execution_accounting", "PASS", 1, 1},
		{"declared_suite_functional_accuracy", "PASS", 6, 6},
		{"route_choice_protocol", "PASS", 1, 1},
	} {
		d, err := findDimension(raw.Report.Completeness.Dimensions, spec.id)
		if err != nil || d.Status != spec.status || d.Numerator != spec.num || d.Denominator != spec.den {
			prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw completeness dimension does not satisfy expected contract: "+spec.id)
		}
	}
	for _, spec := range []struct {
		id, status string
		num, den   int
	}{
		{"external_network_boundary", "PASS", 1, 1},
		{"provider_execution_accounting", "PASS", 1, 1},
	} {
		d, err := findDimension(capture.Completeness.Dimensions, spec.id)
		if err != nil || d.Status != spec.status || d.Numerator != spec.num || d.Denominator != spec.den {
			prepared.Preflight.Errors = append(prepared.Preflight.Errors, "invocation completeness dimension does not satisfy expected contract: "+spec.id)
		}
	}
	if raw.Report.BodyFill.ProposedCandidateID != capture.Decision.ProposedCandidateID || raw.Report.BodyFill.SelectedCandidateID != capture.Decision.SelectedCandidateID ||
		raw.Report.BodyFill.LocalModelPredictions != 0 || raw.Report.BodyFill.ExternalProviderCalls != 0 || !raw.Report.BodyFill.ExternalProviderCallsKnown ||
		raw.Report.BodyFill.TestCasesTotal != cellCases {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw body-fill result differs from invocation or records provider activity")
	}
	var typedDecision bodyFillDecision
	if err := decodeStrictJSON(raw.Report.BodyFill.Decision, &typedDecision); err != nil || typedDecision.Provider != "deterministic" || typedDecision.Mode != "deterministic_fallback" ||
		typedDecision.FallbackReason != "NOT_CONFIGURED" || typedDecision.Selected != capture.Decision.ProposedCandidateID || typedDecision.RequestSHA256 != capture.Decision.RequestSHA256 {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw typed receipt does not match deterministic no-provider invocation")
	}
	var decisionKeys map[string]json.RawMessage
	if err := decodeStrictJSON(raw.Report.BodyFill.Decision, &decisionKeys); err != nil {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw typed receipt JSON invalid")
	} else {
		for _, key := range []string{"tiny_go_variant", "tiny_go_metadata_sha256", "tiny_go_weights_sha256", "tiny_go_predicted_operation", "tiny_go_prediction_applied", "model_metadata_sha256", "model_weights_sha256"} {
			if _, exists := decisionKeys[key]; exists {
				prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw no-model receipt unexpectedly contains field "+key)
			}
		}
	}
	prepared.Preflight.LocalPredictions = raw.Report.BodyFill.LocalModelPredictions
	prepared.Preflight.ExternalProviderCalls = raw.Report.BodyFill.ExternalProviderCalls
	prepared.Preflight.ExternalProviderKnown = raw.Report.BodyFill.ExternalProviderCallsKnown
	prepared.Preflight.ExternalNetworkBoundary, _ = findDimension(raw.Report.Completeness.Dimensions, "external_network_boundary")
	prepared.Preflight.ProviderAccounting, _ = findDimension(raw.Report.Completeness.Dimensions, "provider_execution_accounting")

	if !candidateExists(plan.Candidates, raw.Report.BodyFill.ProposedCandidateID) || !candidateExists(plan.Candidates, raw.Report.BodyFill.SelectedCandidateID) {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "raw proposed or selected candidate is absent from the frozen plan")
	}
	if !sameStrings(capture.ValidationErrors, knownCaptureErrors) {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "capture has an unexpected additional validation error")
	}
	prepared.Preflight.KnownValidationErrors = append([]string(nil), capture.ValidationErrors...)
	if capture.ModelMetadataSHA256 != "" || capture.ModelWeightsSHA256 != "" || capture.Decision.ExternalCalls != 0 || !capture.Decision.ExternalCallsKnown {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, "offline no-model/external-provider evidence does not match the explicit counters")
	}
	if err := verifyWrapperError(runDir, cellID); err != nil {
		prepared.Preflight.Errors = append(prepared.Preflight.Errors, err.Error())
	}
	return []preparedCell{prepared}
}

func compileAndRun(goBinary, outputDir string, prepared preparedCell) cellAudit {
	pre := prepared.Preflight
	cellDir := filepath.Join(outputDir, pre.CellID)
	cell := cellAudit{
		CellID: pre.CellID, Status: "PARTIAL", FixtureID: pre.FixtureID, CaptureStatus: pre.CaptureStatus,
		CaptureValidationErrors: append([]string(nil), pre.KnownValidationErrors...),
		KnownErrorAdjudication:  "Only the two known offline completeness-validator errors were present; raw receipt independently proves deterministic provider, zero/known external activity, no model provenance, and Laya observation UNKNOWN 0/1. Original capture remains unchanged.",
		PlannedCases:            cellCases, RawCLISHA256: pre.RawCLISHA256, CaptureReceiptSHA256: pre.CaptureReceiptSHA256,
		GeneratedSourceSHA256: pre.GeneratedSourceSHA256, GoBuildExitCode: -1, ExecutionExitCode: -1,
	}
	if err := os.Mkdir(cellDir, 0700); err != nil {
		cell.Errors = append(cell.Errors, "cell output directory creation failed")
		return cell
	}
	program, err := appendMain(pre.GeneratedSource, pre.Activity, pre.Cases)
	if err != nil {
		cell.Errors = append(cell.Errors, "independent Go test harness could not be assembled")
		writeJSON(filepath.Join(cellDir, "audit-cell.json"), cell)
		return cell
	}
	if err := writeFile(filepath.Join(cellDir, "generated-check.go"), []byte(program)); err != nil {
		cell.Errors = append(cell.Errors, "generated replay source could not be saved")
		writeJSON(filepath.Join(cellDir, "audit-cell.json"), cell)
		return cell
	}
	build := exec.Command(goBinary, "build", "-trimpath", "-buildvcs=false", "-o", "generated-check", "generated-check.go")
	build.Dir = cellDir
	build.Env = offlineGoEnv()
	var buildOut, buildErr bytes.Buffer
	build.Stdout, build.Stderr = &buildOut, &buildErr
	start := time.Now()
	err = build.Run()
	cell.GoBuildWallMS = float64(time.Since(start)) / float64(time.Millisecond)
	_ = writeFile(filepath.Join(cellDir, "go-build.stdout"), buildOut.Bytes())
	_ = writeFile(filepath.Join(cellDir, "go-build.stderr"), buildErr.Bytes())
	cell.GoBuildStdoutSHA256, cell.GoBuildStderrSHA256 = sha256Hex(buildOut.Bytes()), sha256Hex(buildErr.Bytes())
	cell.GoBuildExitCode = exitCode(err)
	if err != nil {
		cell.Errors = append(cell.Errors, "generated baseline failed Go compilation; raw output retained")
		writeJSON(filepath.Join(cellDir, "audit-cell.json"), cell)
		return cell
	}
	run := exec.Command("./generated-check")
	run.Dir = cellDir
	var runOut, runErr bytes.Buffer
	run.Stdout, run.Stderr = &runOut, &runErr
	start = time.Now()
	err = run.Run()
	cell.ExecutionWallMS = float64(time.Since(start)) / float64(time.Millisecond)
	_ = writeFile(filepath.Join(cellDir, "generated-execution.stdout.json"), runOut.Bytes())
	_ = writeFile(filepath.Join(cellDir, "generated-execution.stderr"), runErr.Bytes())
	cell.ExecutionStdoutSHA256, cell.ExecutionStderrSHA256 = sha256Hex(runOut.Bytes()), sha256Hex(runErr.Bytes())
	cell.ExecutionExitCode = exitCode(err)
	if err != nil {
		cell.Errors = append(cell.Errors, "generated baseline failed execution; raw output retained")
	} else {
		var got []caseResult
		if err := decodeStrictJSON(runOut.Bytes(), &got); err != nil {
			cell.Errors = append(cell.Errors, "generated baseline emitted invalid result JSON")
		} else {
			cell.ObservedCases = len(got)
			for i, want := range pre.Cases {
				if i >= len(got) {
					cell.FailedCases++
					continue
				}
				observed := got[i]
				passed := observed.Input == want.Input && observed.Expected == want.Expected && observed.Actual == want.Expected && observed.Passed
				cell.Cases = append(cell.Cases, caseAudit{Input: observed.Input, Expected: observed.Expected, Actual: observed.Actual, Passed: passed})
				if passed {
					cell.PassedCases++
				} else {
					cell.FailedCases++
				}
			}
			if len(got) != cellCases {
				cell.Errors = append(cell.Errors, "generated baseline result count differs from six frozen finite cases")
			}
		}
	}
	if cell.GoBuildExitCode == 0 && cell.ExecutionExitCode == 0 && cell.ObservedCases == cellCases && cell.FailedCases == 0 && len(cell.Errors) == 0 {
		cell.Status = "PASS_WITH_KNOWN_CAPTURE_VALIDATION_MISMATCH"
	}
	if err := writeJSON(filepath.Join(cellDir, "audit-cell.json"), cell); err != nil {
		cell.Errors = append(cell.Errors, "cell receipt could not be saved")
		cell.Status = "PARTIAL"
	}
	return cell
}

func appendMain(source, activity string, cases []testCase) (string, error) {
	if !strings.HasPrefix(source, "package tiny_smoke\n") {
		return "", errors.New("unexpected generated package clause")
	}
	source = strings.Replace(source, "package tiny_smoke\n", "package main\n\nimport (\"encoding/json\"; \"os\")\n", 1)
	var main strings.Builder
	main.WriteString("\nfunc main() {\n\tcases := []struct { Input int64; Expected int64 }{\n")
	for _, tc := range cases {
		main.WriteString("\t\t{Input: ")
		main.WriteString(strconv.FormatInt(tc.Input, 10))
		main.WriteString(", Expected: ")
		main.WriteString(strconv.FormatInt(tc.Expected, 10))
		main.WriteString("},\n")
	}
	main.WriteString("\t}\n\ttype result struct { Input int64 `json:\"input\"`; Expected int64 `json:\"expected\"`; Actual int64 `json:\"actual\"`; Passed bool `json:\"passed\"` }\n")
	main.WriteString("\tresults := make([]result, 0, len(cases))\n\tfor _, testCase := range cases {\n\t\tactual := ")
	main.WriteString(activity)
	main.WriteString("(testCase.Input)\n\t\tresults = append(results, result{Input: testCase.Input, Expected: testCase.Expected, Actual: actual, Passed: actual == testCase.Expected})\n\t}\n\t_ = json.NewEncoder(os.Stdout).Encode(results)\n}\n")
	return source + main.String(), nil
}

func verifyWrapperError(runDir, cellID string) error {
	stderr, err := os.ReadFile(filepath.Join(runDir, "wrapper-launch", cellID+".stderr.txt"))
	if err != nil || string(stderr) != "tiny smoke: cell captured with 2 validation error(s); inspect invocation.json\n" {
		return errors.New("wrapper error output does not match the recorded exact two-error failure")
	}
	stdout, err := os.ReadFile(filepath.Join(runDir, "wrapper-launch", cellID+".stdout.txt"))
	if err != nil || len(stdout) != 0 {
		return errors.New("wrapper stdout should be empty for the recorded validation failure")
	}
	return nil
}

func findDimension(items []dimension, id string) (dimension, error) {
	var found dimension
	count := 0
	for _, d := range items {
		if d.ID == id {
			found = d
			count++
		}
	}
	if count != 1 {
		return dimension{}, fmt.Errorf("dimension %s occurs %d times", id, count)
	}
	return found, nil
}

func findSummaryRow(summary captureSummary, id string) *struct {
	CellID           string   `json:"cell_id"`
	WrapperExitCode  int      `json:"wrapper_exit_code"`
	CLIExitCode      int      `json:"cli_exit_code"`
	CaptureStatus    string   `json:"capture_status"`
	ValidationErrors []string `json:"validation_errors"`
} {
	for i := range summary.Rows {
		if summary.Rows[i].CellID == id {
			return &summary.Rows[i]
		}
	}
	return nil
}

func candidateExists(candidates []candidate, id string) bool {
	for _, c := range candidates {
		if c.ID == id {
			return true
		}
	}
	return false
}

func expectedActivity(fixture string) string {
	if fixture == "arithmetic" {
		return "ArithmeticHole"
	}
	if fixture == "boolean" {
		return "BooleanHole"
	}
	return ""
}

func expectedReportPlanSHA(fixture string) string {
	switch fixture {
	case "arithmetic":
		return "sha256:acb54b246dd0909731a113984b092061202da40e9240c7e3cd58e9575566ea96"
	case "boolean":
		return "sha256:02c805160a3c5d8cbeb4f7ce20af87e899be153ddcb88e2584b18dd26c79381d"
	default:
		return ""
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func verifyFilePin(root, name, expected string) error {
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return fmt.Errorf("read pinned file %s: %w", name, err)
	}
	if sha256Hex(data) != expected {
		return fmt.Errorf("pinned file %s SHA differs", name)
	}
	return nil
}

func isHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func decodeStrictJSON(data []byte, out any) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	if err := d.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("JSON has trailing values")
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for d.More() {
				keyToken, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("object key is not a string")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			closeToken, err := d.Token()
			if err != nil || closeToken != json.Delim('}') {
				return errors.New("unterminated JSON object")
			}
		case '[':
			for d.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			closeToken, err := d.Token()
			if err != nil || closeToken != json.Delim(']') {
				return errors.New("unterminated JSON array")
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		return nil
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("JSON has trailing values")
	}
	return nil
}

func offlineGoEnv() []string {
	items := make([]string, 0, len(os.Environ())+6)
	for _, item := range os.Environ() {
		key := strings.SplitN(item, "=", 2)[0]
		if key == "GOOO_LAYA_URL" || key == "GOOO_LAYA_API_KEY" || key == "GOOO_PROVIDER" || key == "GOOO_DECISION_PROVIDER" {
			continue
		}
		items = append(items, item)
	}
	items = append(items, "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOWORK=off", "GO111MODULE=off")
	return items
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return writeFile(path, b)
}
