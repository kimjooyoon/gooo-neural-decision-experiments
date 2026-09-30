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

type planCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}

type smokeReceipt struct {
	Schema                  string   `json:"schema"`
	Status                  string   `json:"status"`
	FixtureID               string   `json:"fixture_id"`
	Activity                string   `json:"activity"`
	FixtureSHA256           string   `json:"fixture_sha256"`
	PlanFile                string   `json:"plan_file"`
	PlanFileSHA256          string   `json:"plan_file_sha256"`
	CompilerSourceSHA256    string   `json:"compiler_source_sha256_expected"`
	CompilerSourceObserved  string   `json:"compiler_source_sha256_observed"`
	CLIBinarySHA256Expected string   `json:"cli_binary_sha256_expected"`
	CLIBinarySHA256Observed string   `json:"cli_binary_sha256_observed"`
	StdoutSHA256            string   `json:"stdout_sha256"`
	ModelVariant            string   `json:"model_variant"`
	ModelMetadataSHA256     string   `json:"model_metadata_sha256"`
	ModelWeightsSHA256      string   `json:"model_weights_sha256"`
	ValidationErrors        []string `json:"validation_errors"`
	Decision                struct {
		Provider            string `json:"provider"`
		Mode                string `json:"mode"`
		FallbackReason      string `json:"fallback_reason"`
		ProposedCandidateID string `json:"proposed_candidate_id"`
		SelectedCandidateID string `json:"selected_candidate_id"`
	} `json:"decision"`
}

type cliPayload struct {
	Report struct {
		Decision          string `json:"decision"`
		CompilerSourceSHA string `json:"compiler_source_sha"`
		BodyFill          struct {
			Decision struct {
				Provider       string `json:"provider"`
				Mode           string `json:"mode"`
				FallbackReason string `json:"fallback_reason"`
				Selected       string `json:"selected"`
			} `json:"decision"`
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
	CellID                string       `json:"cell_id"`
	Status                string       `json:"status"`
	FixtureID             string       `json:"fixture_id"`
	ModelVariant          string       `json:"model_variant"`
	CompilerSourceSHA256  string       `json:"compiler_source_sha256"`
	CLIBinarySHA256       string       `json:"cli_binary_sha256"`
	CaptureReceiptSHA256  string       `json:"capture_receipt_sha256"`
	RawCLISHA256          string       `json:"raw_cli_sha256"`
	PlanSHA256            string       `json:"plan_file_sha256"`
	GeneratedSourceSHA256 string       `json:"generated_go_source_sha256"`
	HarnessChange         string       `json:"independent_harness_change"`
	GoVersion             string       `json:"go_version"`
	GoBuildExitCode       int          `json:"go_build_exit_code"`
	GoBuildWallMS         float64      `json:"go_build_wall_ms"`
	GoBuildStdoutSHA256   string       `json:"go_build_stdout_sha256"`
	GoBuildStderrSHA256   string       `json:"go_build_stderr_sha256"`
	ExecutionExitCode     int          `json:"generated_execution_exit_code"`
	ExecutionWallMS       float64      `json:"generated_execution_wall_ms"`
	ExecutionStdoutSHA256 string       `json:"generated_execution_stdout_sha256"`
	ExecutionStderrSHA256 string       `json:"generated_execution_stderr_sha256"`
	PlannedCases          int          `json:"planned_cases"`
	ObservedCases         int          `json:"observed_cases"`
	PassedCases           int          `json:"passed_cases"`
	FailedCases           int          `json:"failed_cases"`
	Cases                 []caseResult `json:"cases,omitempty"`
	Errors                []string     `json:"errors"`
}

type topReceipt struct {
	Schema               string        `json:"schema"`
	Status               string        `json:"status"`
	CompilerSourceSHA256 string        `json:"compiler_source_sha256"`
	CLIBinarySHA256      string        `json:"cli_binary_sha256"`
	GoVersion            string        `json:"go_version"`
	CellsPlanned         int           `json:"cells_planned"`
	CellsCompiled        int           `json:"cells_compiled"`
	CellsPassed          int           `json:"cells_passed"`
	CasesPlanned         int           `json:"cases_planned"`
	CasesObserved        int           `json:"cases_observed"`
	CasesPassed          int           `json:"cases_passed"`
	CasesFailed          int           `json:"cases_failed"`
	Cells                []cellReceipt `json:"cells"`
	ResourceScope        string        `json:"resource_scope"`
}

func main() {
	var (
		runDir     = flag.String("run-dir", "", "directory containing the six saved result cells")
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
	}
	receipt := topReceipt{
		Schema: "gooo/native-tiny-go-independent-go-replay/v1", Status: "PASS",
		CompilerSourceSHA256: sourceSHA, CLIBinarySHA256: binarySHA,
		GoVersion: goVersion, CellsPlanned: len(cellIDs),
		ResourceScope: "independent generated-source Go compile/execute only; separate interval from the six native CLI captures; this is the same finite suite, not holdout or full-domain evidence",
	}
	for _, cellID := range cellIDs {
		cell := replayCell(runDir, fixtureDir, goBinary, sourceSHA, binarySHA, goVersion, outputDir, cellID)
		receipt.Cells = append(receipt.Cells, cell)
		receipt.CasesPlanned += cell.PlannedCases
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
	wantVariant := strings.TrimPrefix(cellID, fixture+"-")
	if capture.Status != "CAPTURED_AND_SOURCE_BOUND" || capture.FixtureID != fixture || capture.ModelVariant != wantVariant ||
		capture.CompilerSourceSHA256 != sourceSHA || capture.CompilerSourceObserved != sourceSHA ||
		capture.CLIBinarySHA256Expected != binarySHA || capture.CLIBinarySHA256Observed != binarySHA ||
		capture.Decision.Provider != "tiny_go" || capture.Decision.Mode == "" || len(capture.ValidationErrors) != 0 {
		receipt.Errors = append(receipt.Errors, "capture provenance/provider status failed independent replay precondition")
	}
	receipt.ModelVariant = capture.ModelVariant
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
	receipt.PlanSHA256 = sha256Hex(planBytes)
	var p struct {
		TestCases []planCase `json:"test_cases"`
	}
	if err := decodeOne(planBytes, &p); err != nil {
		receipt.Errors = append(receipt.Errors, "frozen plan JSON invalid")
		return receipt
	}
	receipt.PlannedCases = len(p.TestCases)
	fixtureBytes, err := os.ReadFile(filepath.Join(fixtureDir, fixture+".gooo"))
	if err != nil || sha256Hex(fixtureBytes) != capture.FixtureSHA256 {
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
