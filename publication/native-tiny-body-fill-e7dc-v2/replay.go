// Replay the six saved native body-codegen outputs by independently compiling
// and executing each emitted Go function against the same explicit finite suite.
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
	frozenRunPlanSHA256 = "1c5a86b321eacf9549d03f4b7df2c94f8ea8171b7dbb04f384094895869c3a58"
	frozenCellCount     = 8
	frozenCasesPerCell  = 6
	frozenCasesTotal    = frozenCellCount * frozenCasesPerCell
)

var frozenModelPins = map[string]modelPin{
	"fp32":        {metadataSHA256: "c5867ee872cbf0fb8227015bd3e94540415786437704ed01b66e6bb0b421c930", weightsSHA256: "854da0f8fc0dcf5560d1f1b8838ea30081752020dbc2005772207217c5d1d17a"},
	"ptq_ternary": {metadataSHA256: "5766a32a9d70ef16c895e7cf8765d7884d59ab6241fcfa225ac2291731a77a4e", weightsSHA256: "3c60ab7154f02f04a94b1bc68274e05c4ae86f10185b19dbe7251c26610a5c8b"},
	"qat_ternary": {metadataSHA256: "4746fc7cbd2864e9536b43d779c7ecc39aa35f8f6f64943c73b951f9e967c8a7", weightsSHA256: "bd4444d5233f07c18168a4dfa0053afde180ce97e390a993e4e0fbb33fb88337"},
}

type modelPin struct {
	metadataSHA256 string
	weightsSHA256  string
}

type planCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}

type replayCandidate struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
}

type smokeReceipt struct {
	Schema                   string `json:"schema"`
	Status                   string `json:"status"`
	ExecutionMode            string `json:"execution_mode"`
	FixtureID                string `json:"fixture_id"`
	Activity                 string `json:"activity"`
	FixtureSHA256            string `json:"fixture_sha256"`
	PlanFile                 string `json:"plan_file"`
	PlanFileSHA256           string `json:"plan_file_sha256"`
	RunPlanSHA256            string `json:"run_plan_sha256"`
	CompilerSourceSHA256     string `json:"compiler_source_sha256_expected"`
	CompilerSourceObserved   string `json:"compiler_source_sha256_observed"`
	CLIBinarySHA256Expected  string `json:"cli_binary_sha256_expected"`
	CLIBinarySHA256Observed  string `json:"cli_binary_sha256_observed"`
	CLIBinarySHA256After     string `json:"cli_binary_sha256_after"`
	StdoutSHA256             string `json:"stdout_sha256"`
	ModelVariant             string `json:"model_variant"`
	ModelMetadataSHA256      string `json:"model_metadata_sha256"`
	ModelMetadataSHA256After string `json:"model_metadata_sha256_after"`
	ModelWeightsSHA256       string `json:"model_weights_sha256"`
	ModelWeightsSHA256After  string `json:"model_weights_sha256_after"`
	GoldOperation            string `json:"gold_operation"`
	RawPredictionMatchesGold *bool  `json:"raw_prediction_matches_gold"`
	MappedProposalCandidate  string `json:"mapped_proposal_candidate_id"`
	MappedProposalScore      *struct {
		ID         string `json:"id"`
		Expression string `json:"expression"`
		Passed     int    `json:"passed"`
		Total      int    `json:"total"`
	} `json:"mapped_proposal_finite_score"`
	ValidationErrors []string `json:"validation_errors"`
	Decision         struct {
		Provider                 string `json:"provider"`
		Mode                     string `json:"mode"`
		FallbackReason           string `json:"fallback_reason"`
		ProposedCandidateID      string `json:"proposed_candidate_id"`
		ProposedScoreSemantics   string `json:"proposed_score_semantics"`
		SelectedCandidateID      string `json:"selected_candidate_id"`
		TinyGoPredictedOperation string `json:"tiny_go_predicted_operation"`
		TinyGoPredictionApplied  *bool  `json:"tiny_go_prediction_applied"`
		LocalPredictions         int    `json:"local_model_predictions"`
		ExternalCalls            int    `json:"external_provider_calls"`
		ExternalCallsKnown       bool   `json:"external_provider_calls_known"`
	} `json:"decision"`
}

type cliPayload struct {
	Report struct {
		Decision          string `json:"decision"`
		CompilerSourceSHA string `json:"compiler_source_sha"`
		BodyFill          struct {
			ProposedCandidateID string `json:"proposed_candidate_id"`
			SelectedCandidateID string `json:"selected_candidate_id"`
			Decision            struct {
				Provider                 string `json:"provider"`
				Mode                     string `json:"mode"`
				FallbackReason           string `json:"fallback_reason"`
				TinyGoVariant            string `json:"tiny_go_variant"`
				TinyGoWeightsSHA256      string `json:"tiny_go_weights_sha256"`
				TinyGoMetadataSHA256     string `json:"tiny_go_metadata_sha256"`
				TinyGoPredictedOperation string `json:"tiny_go_predicted_operation"`
				TinyGoPredictionApplied  *bool  `json:"tiny_go_prediction_applied"`
			} `json:"decision"`
			LocalModelPredictions      int  `json:"local_model_predictions"`
			ExternalProviderCalls      int  `json:"external_provider_calls"`
			ExternalProviderCallsKnown bool `json:"external_provider_calls_known"`
		} `json:"body_fill"`
	} `json:"report"`
	Source string `json:"source"`
}

type caseResult struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
	Actual   int64 `json:"actual"`
	Passed   bool  `json:"passed"`
}

type cellReceipt struct {
	CellID                   string `json:"cell_id"`
	Status                   string `json:"status"`
	ExecutionMode            string `json:"execution_mode"`
	FixtureID                string `json:"fixture_id"`
	ModelVariant             string `json:"model_variant"`
	GoldOperation            string `json:"gold_operation"`
	ProposedScoreSemantics   string `json:"proposed_score_semantics"`
	TinyGoPredictedOperation string `json:"tiny_go_predicted_operation,omitempty"`
	TinyGoPredictionApplied  *bool  `json:"tiny_go_prediction_applied,omitempty"`
	RawPredictionMatchesGold *bool  `json:"raw_prediction_matches_gold,omitempty"`
	MappedProposalCandidate  string `json:"mapped_proposal_candidate_id,omitempty"`
	MappedProposalScore      *struct {
		ID         string `json:"id"`
		Expression string `json:"expression"`
		Passed     int    `json:"passed"`
		Total      int    `json:"total"`
	} `json:"mapped_proposal_finite_score,omitempty"`
	ProviderProposalCandidate  string       `json:"provider_proposal_candidate_id"`
	FinalCandidateID           string       `json:"final_candidate_id"`
	FinalFiniteAccuracyPct     *float64     `json:"generated_replay_observed_accuracy_percent,omitempty"`
	LocalPredictions           int          `json:"local_model_predictions"`
	ExternalProviderCalls      int          `json:"external_provider_calls"`
	ExternalProviderCallsKnown bool         `json:"external_provider_calls_known"`
	CompilerSourceSHA256       string       `json:"compiler_source_sha256"`
	CLIBinarySHA256            string       `json:"cli_binary_sha256"`
	CaptureReceiptSHA256       string       `json:"capture_receipt_sha256"`
	RawCLISHA256               string       `json:"raw_cli_sha256"`
	PlanSHA256                 string       `json:"plan_file_sha256"`
	RunPlanSHA256              string       `json:"run_plan_sha256"`
	GeneratedSourceSHA256      string       `json:"generated_go_source_sha256"`
	HarnessChange              string       `json:"independent_harness_change"`
	GoVersion                  string       `json:"go_version"`
	GoBuildExitCode            int          `json:"go_build_exit_code"`
	GoBuildWallMS              float64      `json:"go_build_wall_ms"`
	GoBuildStdoutSHA256        string       `json:"go_build_stdout_sha256"`
	GoBuildStderrSHA256        string       `json:"go_build_stderr_sha256"`
	ExecutionExitCode          int          `json:"generated_execution_exit_code"`
	ExecutionWallMS            float64      `json:"generated_execution_wall_ms"`
	ExecutionStdoutSHA256      string       `json:"generated_execution_stdout_sha256"`
	ExecutionStderrSHA256      string       `json:"generated_execution_stderr_sha256"`
	PlannedCases               int          `json:"planned_cases"`
	ObservedCases              int          `json:"observed_cases"`
	PassedCases                int          `json:"passed_cases"`
	FailedCases                int          `json:"failed_cases"`
	Cases                      []caseResult `json:"cases,omitempty"`
	Errors                     []string     `json:"errors"`
}

type topReceipt struct {
	Schema               string        `json:"schema"`
	Status               string        `json:"status"`
	CompilerSourceSHA256 string        `json:"compiler_source_sha256"`
	CLIBinarySHA256      string        `json:"cli_binary_sha256"`
	RunPlanSHA256        string        `json:"run_plan_sha256"`
	GoVersion            string        `json:"go_version"`
	CellsPlanned         int           `json:"cells_planned"`
	CellsCompiled        int           `json:"cells_compiled"`
	CellsPassed          int           `json:"cells_passed"`
	CasesPlanned         int           `json:"cases_planned"`
	CasesObserved        int           `json:"cases_observed"`
	CasesPassed          int           `json:"cases_passed"`
	CasesFailed          int           `json:"cases_failed"`
	CasesUnknown         int           `json:"cases_unknown_from_planned_denominator"`
	Cells                []cellReceipt `json:"cells"`
	ResourceScope        string        `json:"resource_scope"`
}

func main() {
	var (
		runDir     = flag.String("run-dir", "", "directory containing the eight saved result cells")
		fixtureDir = flag.String("fixture-dir", "", "directory containing frozen .gooo and plan files")
		goBinary   = flag.String("go", "", "absolute path to the pinned Go toolchain binary")
		sourceSHA  = flag.String("compiler-source-sha", "", "expected compiler source commit SHA")
		binaryPath = flag.String("cli-binary", "", "absolute path to the source-pinned native CLI")
		binarySHA  = flag.String("cli-binary-sha256", "", "expected native CLI SHA-256")
		outputDir  = flag.String("output-dir", "", "new directory for independent replay evidence")
	)
	flag.Parse()
	if err := run(*runDir, *fixtureDir, *goBinary, *sourceSHA, *binaryPath, *binarySHA, *outputDir); err != nil {
		fmt.Fprintln(os.Stderr, "independent replay:", err)
		os.Exit(2)
	}
}

func run(runDir, fixtureDir, goBinary, sourceSHA, binaryPath, binarySHA, outputDir string) error {
	for name, value := range map[string]string{
		"run directory": runDir, "fixture directory": fixtureDir, "Go binary": goBinary,
		"source SHA": sourceSHA, "CLI binary": binaryPath, "CLI binary SHA": binarySHA,
		"output directory": outputDir,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if !filepath.IsAbs(runDir) || !filepath.IsAbs(fixtureDir) || !filepath.IsAbs(goBinary) || !filepath.IsAbs(binaryPath) {
		return errors.New("run, fixture, Go, and CLI binary paths must be absolute")
	}
	runPlanPath := filepath.Join(filepath.Dir(filepath.Clean(fixtureDir)), "run-plan.json")
	if err := verifyFrozenRunPlan(runPlanPath); err != nil {
		return err
	}
	cliBytes, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("read pinned CLI: %w", err)
	}
	if sha256Hex(cliBytes) != binarySHA {
		return errors.New("CLI binary changed before independent replay")
	}
	versionOutput, versionErr := exec.Command(goBinary, "version").Output()
	if versionErr != nil {
		return fmt.Errorf("read Go toolchain version: %w", versionErr)
	}
	goVersion := strings.TrimSpace(string(versionOutput))
	if err := os.Mkdir(outputDir, 0700); err != nil {
		return fmt.Errorf("output directory must be new: %w", err)
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return errors.New("this source-bound replay is pinned to the Darwin/arm64 execution host")
	}
	_ = os.Setenv("GOTOOLCHAIN", "local")
	_ = os.Setenv("GOPROXY", "off")
	_ = os.Setenv("GOSUMDB", "off")
	_ = os.Setenv("GOWORK", "off")
	_ = os.Setenv("GO111MODULE", "off")
	_ = os.Setenv("PATH", filepath.Dir(goBinary)+string(os.PathListSeparator)+os.Getenv("PATH"))

	cellIDs := []string{
		"arithmetic-fp32", "arithmetic-ptq_ternary", "arithmetic-qat_ternary",
		"boolean-fp32", "boolean-ptq_ternary", "boolean-qat_ternary",
		"arithmetic-offline_baseline", "boolean-offline_baseline",
	}
	if len(cellIDs) != frozenCellCount {
		return errors.New("internal frozen cell count differs from the run-plan contract")
	}
	receipt := topReceipt{
		Schema: "gooo/native-tiny-go-independent-go-replay/v2", Status: "PASS",
		CompilerSourceSHA256: sourceSHA, CLIBinarySHA256: binarySHA,
		GoVersion: goVersion, CellsPlanned: frozenCellCount, CasesPlanned: frozenCasesTotal,
		RunPlanSHA256: frozenRunPlanSHA256,
		ResourceScope: "independent generated-source Go compile/execute only; separate interval from the six tiny_go and two offline baseline native CLI captures; same declared finite cases, not a holdout or full-domain evidence",
	}
	for _, cellID := range cellIDs {
		cell := replayCell(runDir, fixtureDir, goBinary, sourceSHA, binarySHA, goVersion, outputDir, cellID)
		receipt.Cells = append(receipt.Cells, cell)
		receipt.CasesObserved += cell.ObservedCases
		receipt.CasesPassed += cell.PassedCases
		receipt.CasesFailed += cell.FailedCases
		if cell.GoBuildExitCode == 0 {
			receipt.CellsCompiled++
		}
		if cell.Status == "PASS" {
			receipt.CellsPassed++
		} else {
			receipt.Status = "PARTIAL"
		}
		if cell.PlannedCases != frozenCasesPerCell {
			receipt.Status = "PARTIAL"
		}
	}
	if receipt.CasesObserved <= frozenCasesTotal {
		receipt.CasesUnknown = receipt.CasesPlanned - receipt.CasesObserved
	} else {
		receipt.Status = "PARTIAL"
	}
	if err := writeJSON(filepath.Join(outputDir, "independent-replay.json"), receipt); err != nil {
		return err
	}
	return nil
}

func replayCell(runDir, fixtureDir, goBinary, sourceSHA, binarySHA, goVersion, outputDir, cellID string) cellReceipt {
	cellDir := filepath.Join(outputDir, cellID)
	receipt := cellReceipt{CellID: cellID, Status: "PARTIAL", GoVersion: goVersion,
		CompilerSourceSHA256: sourceSHA, CLIBinarySHA256: binarySHA,
		RunPlanSHA256: frozenRunPlanSHA256, PlannedCases: frozenCasesPerCell,
		HarnessChange:   "generated Go source package clause changed from tiny_smoke to main, then an independent finite-case main appended; generated activity function body is byte-preserved",
		GoBuildExitCode: -1, ExecutionExitCode: -1}
	if err := os.Mkdir(cellDir, 0700); err != nil {
		receipt.Errors = append(receipt.Errors, "cell evidence directory could not be created")
		return receipt
	}
	parts := strings.Split(cellID, "-")
	if len(parts) < 2 {
		receipt.Errors = append(receipt.Errors, "cell ID malformed")
		return receipt
	}
	fixture := parts[0]
	if fixture != "arithmetic" && fixture != "boolean" {
		receipt.Errors = append(receipt.Errors, "unknown fixture ID")
		return receipt
	}
	receipt.FixtureID = fixture
	// Each frozen cell plans six cases even when capture is missing or invalid.
	// Keep that denominator visible so an unstarted replay cannot shrink it.
	wantVariant := strings.TrimPrefix(cellID, fixture+"-")
	wantMode := "tiny_go"
	if wantVariant == "offline_baseline" {
		wantMode, wantVariant = "offline_baseline", ""
	}
	receipt.ExecutionMode = wantMode
	captureDir := filepath.Join(runDir, cellID)
	captureBytes, err := os.ReadFile(filepath.Join(captureDir, "invocation.json"))
	if err != nil {
		receipt.Errors = append(receipt.Errors, "capture receipt unavailable")
		return receipt
	}
	receipt.CaptureReceiptSHA256 = sha256Hex(captureBytes)
	var capture smokeReceipt
	if err := decodeOne(captureBytes, &capture); err != nil {
		receipt.Errors = append(receipt.Errors, "capture receipt JSON invalid")
		return receipt
	}
	if capture.Status != "CAPTURED_AND_SOURCE_BOUND" || capture.ExecutionMode != wantMode || capture.FixtureID != fixture || capture.ModelVariant != wantVariant ||
		capture.CompilerSourceSHA256 != sourceSHA || capture.CompilerSourceObserved != sourceSHA ||
		capture.CLIBinarySHA256Expected != binarySHA || capture.CLIBinarySHA256Observed != binarySHA ||
		capture.CLIBinarySHA256After != binarySHA || capture.RunPlanSHA256 != frozenRunPlanSHA256 || len(capture.ValidationErrors) != 0 {
		receipt.Errors = append(receipt.Errors, "capture provenance/provider status failed independent replay precondition")
	}
	receipt.ModelVariant = capture.ModelVariant
	receipt.GoldOperation = goldOperationForFixture(fixture)
	receipt.LocalPredictions = capture.Decision.LocalPredictions
	receipt.ExternalProviderCalls = capture.Decision.ExternalCalls
	receipt.ExternalProviderCallsKnown = capture.Decision.ExternalCallsKnown
	if wantMode == "tiny_go" {
		if capture.Decision.Provider != "tiny_go" || capture.Decision.Mode == "" || !validOperation(capture.Decision.TinyGoPredictedOperation) || capture.Decision.TinyGoPredictionApplied == nil ||
			capture.ModelMetadataSHA256 == "" || capture.ModelMetadataSHA256After != capture.ModelMetadataSHA256 ||
			capture.ModelWeightsSHA256 == "" || capture.ModelWeightsSHA256After != capture.ModelWeightsSHA256 {
			receipt.Errors = append(receipt.Errors, "tiny_go capture lacks provider mode or explicit raw prediction provenance")
		}
		if err := requireFrozenModelPins(wantVariant, capture.ModelMetadataSHA256, capture.ModelWeightsSHA256); err != nil ||
			capture.ModelMetadataSHA256After != frozenModelPins[wantVariant].metadataSHA256 || capture.ModelWeightsSHA256After != frozenModelPins[wantVariant].weightsSHA256 {
			receipt.Errors = append(receipt.Errors, "tiny_go capture model hashes differ from frozen run-plan bundle pins")
		}
		expectedSemantics := "fallback_candidate_score_not_model_accuracy"
		if capture.Decision.TinyGoPredictionApplied != nil && *capture.Decision.TinyGoPredictionApplied {
			expectedSemantics = "model_applied_operation_candidate"
		}
		if capture.Decision.ProposedScoreSemantics != expectedSemantics {
			receipt.Errors = append(receipt.Errors, "tiny_go proposal score is not labeled according to its applied flag")
		}
		receipt.ProposedScoreSemantics = capture.Decision.ProposedScoreSemantics
		receipt.TinyGoPredictedOperation = capture.Decision.TinyGoPredictedOperation
		receipt.TinyGoPredictionApplied = capture.Decision.TinyGoPredictionApplied
		receipt.RawPredictionMatchesGold = capture.RawPredictionMatchesGold
		receipt.MappedProposalCandidate = capture.MappedProposalCandidate
	} else {
		if capture.Decision.Provider != "deterministic" || capture.Decision.Mode != "deterministic_fallback" || capture.Decision.FallbackReason != "NOT_CONFIGURED" ||
			capture.Decision.TinyGoPredictedOperation != "" || capture.Decision.TinyGoPredictionApplied != nil ||
			capture.Decision.LocalPredictions != 0 || capture.Decision.ExternalCalls != 0 || !capture.Decision.ExternalCallsKnown ||
			capture.ModelMetadataSHA256 != "" || capture.ModelWeightsSHA256 != "" {
			receipt.Errors = append(receipt.Errors, "offline baseline used a model, external provider, or non-deterministic route")
		}
		if capture.Decision.ProposedScoreSemantics != "deterministic_no_model_fallback_score" {
			receipt.Errors = append(receipt.Errors, "offline baseline score is not labeled as a no-model fallback")
		}
		receipt.ProposedScoreSemantics = capture.Decision.ProposedScoreSemantics
	}
	rawCLI, err := os.ReadFile(filepath.Join(captureDir, "raw-cli.json"))
	if err != nil {
		receipt.Errors = append(receipt.Errors, "raw CLI report unavailable")
		return receipt
	}
	if sha256Hex(rawCLI) != capture.StdoutSHA256 {
		receipt.Errors = append(receipt.Errors, "raw CLI report SHA differs from capture receipt")
	}
	if filepath.Base(capture.PlanFile) != fixture+".plan.json" {
		receipt.Errors = append(receipt.Errors, "capture receipt names a different plan file")
	}
	receipt.RawCLISHA256 = sha256Hex(rawCLI)
	planPath := filepath.Join(fixtureDir, fixture+".plan.json")
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		receipt.Errors = append(receipt.Errors, "frozen plan unavailable")
		return receipt
	}
	if sha256Hex(planBytes) != capture.PlanFileSHA256 {
		receipt.Errors = append(receipt.Errors, "frozen plan SHA differs from source-bound capture")
	}
	_, expectedPlanSHA := frozenHashes(fixture)
	if expectedPlanSHA == "" || sha256Hex(planBytes) != expectedPlanSHA {
		receipt.Errors = append(receipt.Errors, "plan bytes differ from the paired v2 fixture pin")
	}
	receipt.PlanSHA256 = sha256Hex(planBytes)
	var p struct {
		Candidates []replayCandidate `json:"candidates"`
		TestCases  []planCase        `json:"test_cases"`
	}
	if err := decodeOne(planBytes, &p); err != nil {
		receipt.Errors = append(receipt.Errors, "frozen plan JSON invalid")
		return receipt
	}
	if len(p.TestCases) != 6 {
		receipt.Errors = append(receipt.Errors, "frozen finite plan no longer has its six-case denominator")
	}
	fixtureBytes, err := os.ReadFile(filepath.Join(fixtureDir, fixture+".gooo"))
	expectedFixtureSHA, _ := frozenHashes(fixture)
	if err != nil || sha256Hex(fixtureBytes) != capture.FixtureSHA256 || sha256Hex(fixtureBytes) != expectedFixtureSHA {
		receipt.Errors = append(receipt.Errors, "frozen source SHA differs from source-bound capture")
	}
	var payload cliPayload
	if err := decodeOne(rawCLI, &payload); err != nil {
		receipt.Errors = append(receipt.Errors, "raw CLI JSON invalid")
		return receipt
	}
	if payload.Report.Decision != "PASS" || payload.Report.CompilerSourceSHA != sourceSHA || strings.TrimSpace(payload.Source) == "" {
		receipt.Errors = append(receipt.Errors, "saved CLI report lacks PASS, pinned compiler source, or generated source")
	}
	receipt.ProviderProposalCandidate = payload.Report.BodyFill.ProposedCandidateID
	receipt.FinalCandidateID = payload.Report.BodyFill.SelectedCandidateID
	if payload.Report.BodyFill.ProposedCandidateID != capture.Decision.ProposedCandidateID ||
		payload.Report.BodyFill.SelectedCandidateID != capture.Decision.SelectedCandidateID ||
		payload.Report.BodyFill.Decision.Provider != capture.Decision.Provider ||
		payload.Report.BodyFill.Decision.Mode != capture.Decision.Mode ||
		payload.Report.BodyFill.Decision.FallbackReason != capture.Decision.FallbackReason {
		receipt.Errors = append(receipt.Errors, "raw CLI decision differs from its invocation summary")
	}
	if wantMode == "tiny_go" {
		d := payload.Report.BodyFill.Decision
		if d.TinyGoPredictedOperation != capture.Decision.TinyGoPredictedOperation || d.TinyGoPredictionApplied == nil ||
			capture.Decision.TinyGoPredictionApplied == nil || *d.TinyGoPredictionApplied != *capture.Decision.TinyGoPredictionApplied ||
			d.TinyGoVariant != capture.ModelVariant || d.TinyGoMetadataSHA256 != capture.ModelMetadataSHA256 || d.TinyGoWeightsSHA256 != capture.ModelWeightsSHA256 {
			receipt.Errors = append(receipt.Errors, "raw CLI tiny_go prediction or model provenance differs from invocation summary")
		}
		wantGold := "add"
		if fixture == "boolean" {
			wantGold = "less_equal"
		}
		matches := d.TinyGoPredictedOperation == wantGold
		if capture.GoldOperation != wantGold || capture.RawPredictionMatchesGold == nil || *capture.RawPredictionMatchesGold != matches {
			receipt.Errors = append(receipt.Errors, "raw closed operation gold comparison differs from the fixture contract")
		}
		mappedID := mapOperationToCandidate(fixture, p.Candidates, d.TinyGoPredictedOperation)
		if mappedID != capture.MappedProposalCandidate {
			receipt.Errors = append(receipt.Errors, "mapped model operation candidate differs from the raw label")
		}
		if d.TinyGoPredictionApplied != nil {
			if *d.TinyGoPredictionApplied {
				if d.Mode != "tiny_go" || mappedID == "" || payload.Report.BodyFill.ProposedCandidateID != mappedID || d.FallbackReason != "" {
					receipt.Errors = append(receipt.Errors, "applied raw operation does not match the provider proposal")
				}
			} else if d.Mode != "deterministic_fallback" || payload.Report.BodyFill.ProposedCandidateID != p.Candidates[0].ID ||
				(d.FallbackReason != "TINY_GO_LOW_CONFIDENCE" && d.FallbackReason != "TINY_GO_OPERATION_NOT_OFFERED") {
				receipt.Errors = append(receipt.Errors, "non-applied prediction does not preserve deterministic fallback")
			}
		}
		mappedScore := scoreMappedProposal(fixture, p.Candidates, p.TestCases, d.TinyGoPredictedOperation)
		if mappedScore == nil {
			if capture.MappedProposalScore != nil {
				receipt.Errors = append(receipt.Errors, "unoffered raw operation has an unexpected mapped candidate score")
			}
		} else if capture.MappedProposalScore == nil || *capture.MappedProposalScore != *mappedScore {
			receipt.Errors = append(receipt.Errors, "mapped candidate finite score differs from independent replay")
		} else {
			receipt.MappedProposalScore = mappedScore
		}
		receipt.RawPredictionMatchesGold = capture.RawPredictionMatchesGold
		receipt.TinyGoPredictedOperation = d.TinyGoPredictedOperation
		receipt.TinyGoPredictionApplied = d.TinyGoPredictionApplied
		receipt.MappedProposalCandidate = mappedID
	} else {
		d := payload.Report.BodyFill.Decision
		if d.TinyGoPredictedOperation != "" || d.TinyGoPredictionApplied != nil ||
			payload.Report.BodyFill.LocalModelPredictions != 0 || payload.Report.BodyFill.ExternalProviderCalls != 0 || !payload.Report.BodyFill.ExternalProviderCallsKnown {
			receipt.Errors = append(receipt.Errors, "raw offline baseline report contains model or external provider activity")
		}
		if d.Provider != "deterministic" || d.Mode != "deterministic_fallback" || d.FallbackReason != "NOT_CONFIGURED" ||
			payload.Report.BodyFill.ProposedCandidateID != p.Candidates[0].ID {
			receipt.Errors = append(receipt.Errors, "raw offline baseline did not retain the declared no-provider fallback")
		}
	}
	receipt.GeneratedSourceSHA256 = sha256Hex([]byte(payload.Source))
	program, err := appendTestMain(payload.Source, capture.Activity, p.TestCases)
	if err != nil {
		receipt.Errors = append(receipt.Errors, "could not prepare independent Go entry point")
		return receipt
	}
	if err := writeFile(filepath.Join(cellDir, "generated-check.go"), []byte(program)); err != nil {
		receipt.Errors = append(receipt.Errors, "independent Go source could not be saved")
		return receipt
	}
	receipt.GoBuildExitCode = -1
	buildCmd := exec.Command(goBinary, "build", "-trimpath", "-o", "generated-check", "generated-check.go")
	buildCmd.Dir = cellDir
	buildCmd.Env = offlineGoEnv()
	var buildOut, buildErr bytes.Buffer
	buildCmd.Stdout, buildCmd.Stderr = &buildOut, &buildErr
	started := time.Now()
	commandErr := buildCmd.Run()
	receipt.GoBuildWallMS = float64(time.Since(started)) / float64(time.Millisecond)
	if err := writeFile(filepath.Join(cellDir, "go-build.stdout"), buildOut.Bytes()); err != nil {
		receipt.Errors = append(receipt.Errors, "build stdout could not be preserved")
	}
	if err := writeFile(filepath.Join(cellDir, "go-build.stderr"), buildErr.Bytes()); err != nil {
		receipt.Errors = append(receipt.Errors, "build stderr could not be preserved")
	}
	receipt.GoBuildStdoutSHA256, receipt.GoBuildStderrSHA256 = sha256Hex(buildOut.Bytes()), sha256Hex(buildErr.Bytes())
	receipt.GoBuildExitCode = exitCode(commandErr)
	if commandErr != nil {
		receipt.Errors = append(receipt.Errors, "independent Go build failed; raw diagnostics preserved")
		finalizeCell(cellDir, &receipt)
		return receipt
	}
	execution := exec.Command("./generated-check")
	execution.Dir = cellDir
	var executionOut, executionErr bytes.Buffer
	execution.Stdout, execution.Stderr = &executionOut, &executionErr
	started = time.Now()
	executionErrValue := execution.Run()
	receipt.ExecutionWallMS = float64(time.Since(started)) / float64(time.Millisecond)
	if err := writeFile(filepath.Join(cellDir, "generated-execution.stdout.json"), executionOut.Bytes()); err != nil {
		receipt.Errors = append(receipt.Errors, "generated execution stdout could not be preserved")
	}
	if err := writeFile(filepath.Join(cellDir, "generated-execution.stderr"), executionErr.Bytes()); err != nil {
		receipt.Errors = append(receipt.Errors, "generated execution stderr could not be preserved")
	}
	receipt.ExecutionStdoutSHA256 = sha256Hex(executionOut.Bytes())
	receipt.ExecutionStderrSHA256 = sha256Hex(executionErr.Bytes())
	receipt.ExecutionExitCode = exitCode(executionErrValue)
	if executionErrValue != nil {
		receipt.Errors = append(receipt.Errors, "generated program exited unsuccessfully; raw diagnostics preserved")
	} else {
		if err := decodeOne(executionOut.Bytes(), &receipt.Cases); err != nil {
			receipt.Errors = append(receipt.Errors, "generated program emitted invalid JSON case results")
		} else {
			for i, observed := range receipt.Cases {
				if i >= len(p.TestCases) {
					break
				}
				want := p.TestCases[i]
				if observed.Input != want.Input || observed.Expected != want.Expected || !observed.Passed || observed.Actual != want.Expected {
					receipt.FailedCases++
				} else {
					receipt.PassedCases++
				}
			}
			receipt.ObservedCases = len(receipt.Cases)
			if receipt.ObservedCases > 0 {
				accuracy := float64(receipt.PassedCases) * 100 / float64(receipt.ObservedCases)
				receipt.FinalFiniteAccuracyPct = &accuracy
			}
			if receipt.ObservedCases != receipt.PlannedCases {
				receipt.Errors = append(receipt.Errors, "generated case count differs from frozen finite denominator")
			}
		}
	}
	if err := finalizeCell(cellDir, &receipt); err != nil {
		receipt.Errors = append(receipt.Errors, "cell receipt could not be saved")
	}
	return receipt
}

func appendTestMain(source, activity string, cases []planCase) (string, error) {
	if !strings.HasPrefix(source, "package tiny_smoke\n") {
		return "", errors.New("unexpected generated package clause")
	}
	source = strings.Replace(source, "package tiny_smoke\n", "package main\n\nimport (\n\t\"encoding/json\"\n\t\"os\"\n)\n", 1)
	var main strings.Builder
	main.WriteString("\nfunc main() {\n")
	main.WriteString("\tcases := []struct { Input int64; Expected int64 }{\n")
	for _, tc := range cases {
		main.WriteString("\t\t{Input: ")
		main.WriteString(strconv.FormatInt(tc.Input, 10))
		main.WriteString(", Expected: ")
		main.WriteString(strconv.FormatInt(tc.Expected, 10))
		main.WriteString("},\n")
	}
	main.WriteString("\t}\n\ttype result struct { Input int64 `json:\"input\"`; Expected int64 `json:\"expected\"`; Actual int64 `json:\"actual\"`; Passed bool `json:\"passed\"` }\n")
	main.WriteString("\tresults := make([]result, 0, len(cases))\n")
	main.WriteString("\tfor _, testCase := range cases {\n")
	main.WriteString("\t\tactual := ")
	main.WriteString(activity)
	main.WriteString("(testCase.Input)\n")
	main.WriteString("\t\tresults = append(results, result{Input: testCase.Input, Expected: testCase.Expected, Actual: actual, Passed: actual == testCase.Expected})\n")
	main.WriteString("\t}\n\t_ = json.NewEncoder(os.Stdout).Encode(results)\n}\n")
	return source + main.String(), nil
}

func validOperation(operation string) bool {
	switch operation {
	case "add", "subtract", "multiply", "less_than", "less_equal", "equal", "and", "or":
		return true
	default:
		return false
	}
}

func goldOperationForFixture(fixture string) string {
	if fixture == "arithmetic" {
		return "add"
	}
	if fixture == "boolean" {
		return "less_equal"
	}
	return ""
}

func frozenHashes(fixture string) (sourceSHA, planSHA string) {
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

func mapOperationToCandidate(fixture string, candidates []replayCandidate, operation string) string {
	for _, candidate := range candidates {
		if fixture == "arithmetic" {
			want := map[string]string{"sum": "add", "difference": "subtract", "product": "multiply"}[candidate.ID]
			if want == operation {
				return candidate.ID
			}
		}
		if fixture == "boolean" {
			want := map[string]string{"less_than": "less_than", "less_equal": "less_equal", "equal": "equal"}[candidate.ID]
			if want == operation {
				return candidate.ID
			}
		}
	}
	return ""
}

func scoreMappedProposal(fixture string, candidates []replayCandidate, cases []planCase, operation string) *struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
	Passed     int    `json:"passed"`
	Total      int    `json:"total"`
} {
	id := mapOperationToCandidate(fixture, candidates, operation)
	if id == "" {
		return nil
	}
	for _, candidate := range candidates {
		if candidate.ID != id {
			continue
		}
		passed := 0
		for _, testCase := range cases {
			if evalCandidate(fixture, candidate.ID, testCase.Input) == testCase.Expected {
				passed++
			}
		}
		return &struct {
			ID         string `json:"id"`
			Expression string `json:"expression"`
			Passed     int    `json:"passed"`
			Total      int    `json:"total"`
		}{ID: candidate.ID, Expression: candidate.Expression, Passed: passed, Total: len(cases)}
	}
	return nil
}

func evalCandidate(fixture, id string, input int64) int64 {
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

func finalizeCell(dir string, receipt *cellReceipt) error {
	if receipt.GoBuildExitCode == 0 && receipt.ExecutionExitCode == 0 && receipt.ObservedCases == receipt.PlannedCases &&
		receipt.FailedCases == 0 && len(receipt.Errors) == 0 {
		receipt.Status = "PASS"
	} else {
		receipt.Status = "PARTIAL"
	}
	return writeJSON(filepath.Join(dir, "verification.json"), *receipt)
}

func offlineGoEnv() []string {
	env := make([]string, 0, len(os.Environ())+5)
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if ok && (name == "GOTOOLCHAIN" || name == "GOPROXY" || name == "GOSUMDB" || name == "GOWORK" || name == "GO111MODULE") {
			continue
		}
		env = append(env, item)
	}
	return append(env, "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOWORK=off", "GO111MODULE=off")
}

func decodeOne(data []byte, dst any) error {
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

func writeFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
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

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func requireFrozenModelPins(variant, metadataSHA, weightsSHA string) error {
	pins, ok := frozenModelPins[variant]
	if !ok || metadataSHA != pins.metadataSHA256 || weightsSHA != pins.weightsSHA256 {
		return errors.New("model metadata/weights hashes do not match the frozen run-plan bundle for this variant")
	}
	return nil
}

func verifyFrozenRunPlan(path string) error {
	planBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read frozen run plan: %w", err)
	}
	if sha256Hex(planBytes) != frozenRunPlanSHA256 {
		return errors.New("frozen run-plan SHA-256 differs from the source-bound plan pin")
	}
	return nil
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
