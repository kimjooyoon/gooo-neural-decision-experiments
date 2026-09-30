// evaluate-bodyplans scores frozen synthetic plans and independently executes
// emitted Gooo through the pinned native compiler and pinned Go toolchain.
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
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodydecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

type record struct {
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
	NativeFailure    string                     `json:"native_failure,omitempty"`
}

type report struct {
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
	Failure               string                       `json:"failure,omitempty"`
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

type config struct {
	cohort, models, gooo, goooSHA, goBin, goSHA, output, revision, layaURL, layaRevision string
	limit                                                                                int
	cohortSHA                                                                            string
	expectedRows, expectedFamilies                                                       int
}

func main() {
	var cfg config
	flag.StringVar(&cfg.cohort, "cohort", "studies/body-plan-v1/cohort.jsonl", "frozen JSONL corpus")
	flag.StringVar(&cfg.cohortSHA, "cohort-sha256", "", "required frozen corpus digest")
	flag.IntVar(&cfg.expectedRows, "expected-intents", 128, "exact declared program row count")
	flag.IntVar(&cfg.expectedFamilies, "expected-families", 8, "exact declared family count")
	flag.StringVar(&cfg.models, "models", "runs/pilot-mps-20260930-v1/models", "frozen model directory")
	flag.StringVar(&cfg.gooo, "gooo-bin", "", "pinned native compiler")
	flag.StringVar(&cfg.goooSHA, "gooo-sha256", "", "required native binary digest")
	flag.StringVar(&cfg.goBin, "go-bin", "", "pinned physical Go executable")
	flag.StringVar(&cfg.goSHA, "go-sha256", "", "required Go executable digest")
	flag.StringVar(&cfg.output, "output-dir", "", "fresh evidence directory")
	flag.StringVar(&cfg.revision, "source-revision", "", "frozen runner source revision")
	flag.StringVar(&cfg.layaURL, "laya-url", "", "optional loopback reference service")
	flag.StringVar(&cfg.layaRevision, "laya-revision", "", "required exact multilingual revision when Laya enabled")
	flag.IntVar(&cfg.limit, "attempt-limit", 64, "candidate budget shared by model and model-free search")
	flag.Parse()
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func run(cfg config) (resultErr error) {
	if !lowerHex(cfg.revision, 40) || !lowerHex(cfg.cohortSHA, 64) || cfg.limit < 1 || cfg.limit > 128 || cfg.output == "" {
		return errors.New("exact source revision, fresh output, and attempt limit1..128 are required")
	}
	var pathErr error
	cfg.gooo, pathErr = filepath.Abs(cfg.gooo)
	if pathErr != nil {
		return pathErr
	}
	cfg.goBin, pathErr = filepath.Abs(cfg.goBin)
	if pathErr != nil {
		return pathErr
	}
	for _, pin := range []struct{ path, expected string }{{cfg.gooo, cfg.goooSHA}, {cfg.goBin, cfg.goSHA}} {
		actual, err := fileDigest(pin.path)
		if err != nil || len(pin.expected) != 64 || actual != pin.expected {
			return errors.New("required executable SHA-256 pin mismatch")
		}
	}
	cohortFile, err := os.Open(cfg.cohort)
	if err != nil {
		return err
	}
	cohort, readErr := io.ReadAll(io.LimitReader(cohortFile, (16<<20)+1))
	closeErr := cohortFile.Close()
	if readErr != nil || closeErr != nil || len(cohort) > 16<<20 {
		return errors.New("cannot read bounded cohort")
	}
	if digest(cohort) != cfg.cohortSHA {
		return errors.New("frozen cohort digest mismatch")
	}
	rows, err := readRecords(cohort)
	if err != nil {
		return err
	}
	if len(rows) < 100 || len(rows) > 256 || len(rows) != cfg.expectedRows {
		return errors.New("study row count differs from declared100..256 programs")
	}
	families := make(map[string]int)
	var totalTraining int
	for _, row := range rows {
		families[row.Family]++
		totalTraining += len(row.Training)
	}
	if len(families) != cfg.expectedFamilies {
		return errors.New("family count differs from declaration")
	}
	searchArms := 4
	if cfg.layaURL != "" {
		searchArms++
	}
	if totalTraining*cfg.limit*searchArms > 5000000 {
		return errors.New("study exceeds five million planned training evaluations")
	}
	models := make(map[string]*decision.Model)
	modelArtifacts := make(map[string]map[string]string)
	for _, name := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		m, err := decision.Load(filepath.Join(cfg.models, name, "model.json"))
		if err != nil {
			return err
		}
		models[name] = m
		metadataSHA, err := fileDigest(filepath.Join(cfg.models, name, "model.json"))
		if err != nil {
			return err
		}
		modelArtifacts[name] = map[string]string{"model_json": metadataSHA, "weights_bin": m.WeightsSHA256()}
	}
	if cfg.layaURL != "" {
		if !lowerHex(cfg.layaRevision, 40) {
			return errors.New("Laya requires exact revision pin")
		}
	}
	if err := os.Mkdir(cfg.output, 0755); err != nil {
		return fmt.Errorf("fresh output directory: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	runnerSHA, err := fileDigest(executable)
	if err != nil {
		return err
	}
	arms := []string{"deterministic", "deterministic_search", "fp32", "fp32_search", "ptq_ternary", "ptq_ternary_search", "qat_ternary", "qat_ternary_search"}
	if cfg.layaURL != "" {
		arms = append(arms, "laya_multilingual", "laya_multilingual_search")
	}
	metadata := report{Schema: "gooo/multi-node-body-plan-evaluation/v1", Status: "running", CohortSHA: digest(cohort), SourceRevision: cfg.revision, RunnerSHA: runnerSHA, GoooSHA: cfg.goooSHA, GoSHA: cfg.goSHA, LayaRevision: cfg.layaRevision, PlannedIntents: len(rows), PlannedCells: len(rows) * len(arms), AttemptLimit: cfg.limit, Limits: []string{"Synthetic preauthored structural plans; model only fills bounded operation holes.", "Training-only finite enumeration is not a learned feedback policy; heldout is not passed to selection/search.", "Repeated arms share the same intents; case-level success is not whole-domain proof.", "Separate Laya reference service uses its upstream PyTorch runtime; connector, selection, orchestration and verification are Go.", "Process-level CPU/RSS and per-request distribution are unmeasured by this evaluator."}}
	metadata.FamilyCounts = families
	metadata.ModelArtifactSHA256 = modelArtifacts
	for _, row := range rows {
		metadata.PlannedGoCases += (len(row.Training) + len(row.Heldout)) * len(arms)
		for _, expr := range row.Plan.Expressions {
			if expr.Kind == "hole" && cfg.layaURL != "" {
				metadata.PlannedLayaPOSTs++
			}
		}
	}
	if err := writeJSON(filepath.Join(cfg.output, "preexecution.json"), metadata); err != nil {
		return err
	}
	started := time.Now()
	defer func() {
		metadata.UnstartedCells = metadata.PlannedCells - metadata.ObservedCells
		for _, entry := range metadata.Cells {
			if entry.NativePass {
				metadata.NativePassed++
			} else {
				metadata.NativeFailedOrUnknown++
			}
			metadata.ObservedGoCases += entry.GoTests
		}
		metadata.UnknownGoCases = metadata.PlannedGoCases - metadata.ObservedGoCases
		metadata.ExecutablePinsStable = true
		for _, pin := range []struct{ path, sha string }{{cfg.gooo, cfg.goooSHA}, {cfg.goBin, cfg.goSHA}, {executable, runnerSHA}} {
			actual, err := fileDigest(pin.path)
			if err != nil || actual != pin.sha {
				metadata.ExecutablePinsStable = false
				if resultErr == nil {
					resultErr = errors.New("executable changed during execution")
				}
			}
		}
		metadata.WallMS = float64(time.Since(started).Nanoseconds()) / 1e6
		if resultErr != nil {
			metadata.Status = "failed_partial"
			metadata.Failure = "EVALUATION_STOPPED; inspect preserved raw captures and unknown counts"
		} else {
			metadata.Status = "completed"
		}
		if err := writeJSON(filepath.Join(cfg.output, "report.json"), metadata); resultErr == nil {
			resultErr = err
		}
	}()
	journal, err := os.OpenFile(filepath.Join(cfg.output, "selected-cells.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer journal.Close()
	journalEncoder := json.NewEncoder(journal)
	if cfg.layaURL != "" {
		if err := checkLaya(cfg.layaURL, cfg.layaRevision); err != nil {
			return err
		}
	}
	for _, row := range rows {
		selected := make(map[string]bodydecision.Selection)
		for _, provider := range []string{"deterministic", "fp32", "ptq_ternary", "qat_ternary"} {
			choice, err := bodydecision.Choose(row.Plan, models[provider], "")
			if err != nil {
				return fmt.Errorf("%s %s: %w", row.ID, provider, err)
			}
			selected[provider] = choice
			metadata.TinyPredictions += choice.ModelCalls
		}
		if cfg.layaURL != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			choice, exchanges, err := bodydecision.ChooseLaya(ctx, row.Plan, cfg.layaURL)
			cancel()
			metadata.LayaPOSTs += choice.ProviderCalls
			if saveErr := writeJSON(filepath.Join(cfg.output, row.ID+"-laya-exchanges.json"), exchanges); saveErr != nil {
				return saveErr
			}
			if err != nil {
				return fmt.Errorf("%s Laya unresolved: %w", row.ID, err)
			}
			selected["laya_multilingual"] = choice
		}
		for _, arm := range arms {
			provider := strings.TrimSuffix(arm, "_search")
			initial := selected[provider]
			choices := initial.Choices
			entry := cell{ID: row.ID, Family: row.Family, Arm: arm, Initial: initial, Status: "selected"}
			if strings.HasSuffix(arm, "_search") {
				search, err := bodydecision.Search(row.Plan, choices, row.Training, cfg.limit)
				if err != nil {
					return err
				}
				entry.Search = &search
				choices = search.Choices
			}
			entry.Choices = bodydecision.CloneChoices(choices)
			entry.TotalHoles = len(row.Oracle)
			for id, expected := range row.Oracle {
				if choices[id] == expected {
					entry.OracleHoles++
				}
			}
			program, err := bodyplan.Compile(row.Plan, choices)
			if err != nil {
				return err
			}
			entry.Training, err = bodydecision.ScoreProgram(program, row.Training)
			if err != nil {
				return err
			}
			entry.Heldout, err = bodydecision.ScoreProgram(program, row.Heldout)
			if err != nil {
				return err
			}
			dir := filepath.Join(cfg.output, arm, row.ID)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			entry.GoooSHA = digest([]byte(program.GoooSource()))
			if err := os.WriteFile(filepath.Join(dir, "input.gooo"), []byte(program.GoooSource()), 0644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "plan-generated.go.txt"), []byte(program.GoSource()), 0644); err != nil {
				return err
			}
			source, pass, nativeErr := nativeGenerate(cfg.gooo, dir, row.Plan.Name)
			entry.NativePass = pass
			entry.Status = "native_rejected"
			if nativeErr != nil {
				entry.NativeFailure = "COMMAND_OR_VALIDATION_FAILED"
				if errors.Is(nativeErr, context.DeadlineExceeded) {
					entry.NativeFailure = "TIMEOUT_EXECUTION_UNKNOWN"
				}
			}
			if nativeErr == nil {
				entry.GoSHA = digest(source)
				entry.Status = "native_generated"
				if err := os.WriteFile(filepath.Join(dir, "native-generated.go.txt"), source, 0644); err != nil {
					return err
				}
			}
			metadata.Cells = append(metadata.Cells, entry)
			metadata.ObservedCells++
			if err := journalEncoder.Encode(entry); err != nil {
				return err
			}
			if metadata.ObservedCells%32 == 0 {
				progress := metadata
				progress.Cells = nil
				if err := writeJSON(filepath.Join(cfg.output, "progress.json"), progress); err != nil {
					return err
				}
			}
		}
	}
	for _, arm := range arms {
		if err := compileArm(cfg, rows, arm, &metadata); err != nil {
			return err
		}
	}
	if metadata.ObservedCells != metadata.PlannedCells {
		return errors.New("planned and observed cell count differ")
	}
	return nil
}

func readRecords(data []byte) ([]record, error) {
	var rows []record
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 65536)
	ids := map[string]bool{}
	names := map[string]bool{}
	for scanner.Scan() {
		var row record
		if err := strictjson.Decode(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		if row.ID != row.Plan.ID || row.ID == "" || strings.ContainsAny(row.ID, "/\\.") || ids[row.ID] || names[row.Plan.Name] {
			return nil, errors.New("invalid or duplicate row identity")
		}
		ids[row.ID] = true
		names[row.Plan.Name] = true
		if _, err := bodydecision.Validate(row.Plan); err != nil {
			return nil, fmt.Errorf("%s: %w", row.ID, err)
		}
		oracle, err := bodyplan.Compile(row.Plan, row.Oracle)
		if err != nil {
			return nil, err
		}
		seen := map[int64]bool{}
		for _, cases := range [][]bodydecision.Case{row.Training, row.Heldout} {
			if len(cases) < 1 || len(cases) > 4096 {
				return nil, errors.New("missing or oversized case split")
			}
			for _, test := range cases {
				if seen[test.Input] {
					return nil, errors.New("train/heldout inputs must be unique and disjoint")
				}
				seen[test.Input] = true
			}
			score, err := bodydecision.ScoreProgram(oracle, cases)
			if err != nil || score.Correct != score.Total {
				return nil, fmt.Errorf("%s oracle does not reproduce frozen independent cases", row.ID)
			}
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

func nativeGenerate(binaryPath, dir, name string) ([]byte, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath, "body-codegen", "--json", "--activity", name, "input.gooo")
	cmd.Dir = dir
	cmd.Env = cleanEnv()
	if err := containCommand(cmd); err != nil {
		return nil, false, err
	}
	stdout := boundedBuffer{limit: 131072}
	stderr := boundedBuffer{limit: 65536}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if writeErr := os.WriteFile(filepath.Join(dir, "native-stdout.json"), stdout.Bytes(), 0644); writeErr != nil {
		return nil, false, writeErr
	}
	if writeErr := os.WriteFile(filepath.Join(dir, "native-stderr.txt"), stderr.Bytes(), 0644); writeErr != nil {
		return nil, false, writeErr
	}
	if err != nil {
		return nil, false, err
	}
	var reply struct {
		Report struct {
			Decision  string `json:"decision"`
			Typecheck bool   `json:"typecheck_passed"`
			Replay    bool   `json:"deterministic_replay"`
		} `json:"report"`
		Source string `json:"source"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &reply); err != nil {
		return nil, false, err
	}
	if reply.Report.Decision != "PASS" || !reply.Report.Typecheck || !reply.Report.Replay || reply.Source == "" {
		return nil, false, errors.New("native generation did not pass")
	}
	return []byte(reply.Source), true, nil
}

func cleanEnv() []string {
	result := []string{"PATH=/usr/bin:/bin", "GOTOOLCHAIN=local", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOENV=off"}
	for _, name := range []string{"HOME", "TMPDIR"} {
		if value := os.Getenv(name); value != "" {
			result = append(result, name+"="+value)
		}
	}
	return result
}

func compileArm(cfg config, rows []record, arm string, metadata *report) error {
	dir := filepath.Join(cfg.output, arm, "compiled-replay")
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	var code, tests strings.Builder
	code.WriteString("package bodyplan\n\n")
	tests.WriteString("package bodyplan\nimport (\"testing\";\"fmt\")\nfunc TestFrozenCases(t *testing.T) {\n")
	for _, row := range rows {
		path := filepath.Join(cfg.output, arm, row.ID, "native-generated.go.txt")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s native rejection prevents independent execution: %w", row.ID, err)
		}
		source := string(data)
		if !strings.HasPrefix(source, "package bodyplan\n") {
			return errors.New("unexpected native package")
		}
		code.WriteString(strings.TrimPrefix(source, "package bodyplan\n"))
		for _, split := range []struct {
			name  string
			cases []bodydecision.Case
		}{{"training", row.Training}, {"heldout", row.Heldout}} {
			for index, test := range split.cases {
				literal := fmt.Sprintf("int64(%d)", test.Expected.Int)
				if test.Expected.Type == decision.TypeBool {
					literal = fmt.Sprintf("%t", test.Expected.Bool)
				}
				fmt.Fprintf(&tests, "{ got := %s(int64(%d)); passed := got == %s; fmt.Printf(%q, passed); if !passed { t.Errorf(%q, got) } }\n", row.Plan.Name, test.Input, literal, "GOOO_CASE\t"+row.ID+"\t"+split.name+"\t"+strconv.Itoa(index)+"\t%t\n", row.ID+": got %v")
			}
		}
	}
	tests.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module frozen.bodyplan.replay\ngo 1.27.0\n"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "generated.go"), []byte(code.String()), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "generated_test.go"), []byte(tests.String()), 0644); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cfg.goBin, "test", "-count=1", "-v", "./...")
	cmd.Dir = dir
	cmd.Env = cleanEnv()
	if err := containCommand(cmd); err != nil {
		return err
	}
	capture := boundedBuffer{limit: 4 << 20}
	cmd.Stdout = &capture
	cmd.Stderr = &capture
	err := cmd.Run()
	output := capture.Bytes()
	if writeErr := os.WriteFile(filepath.Join(dir, "go-test-output.txt"), output, 0644); writeErr != nil {
		return writeErr
	}
	if ctx.Err() != nil {
		if partial, parseErr := parseCaseOutput(output, rows, false); parseErr == nil {
			applyObserved(metadata, arm, partial, false)
		}
		return ctx.Err()
	}
	// Failing behavior is an observed result, not a failed capture. Go's output
	// separates mismatches from compilation errors; all case invocations run.
	if err != nil && !bytes.Contains(output, []byte("--- FAIL: TestFrozenCases")) {
		return fmt.Errorf("%s independent Go build failed: %w", arm, err)
	}
	observed, parseErr := parseCases(output, rows)
	if parseErr != nil {
		return parseErr
	}
	mismatch := applyObserved(metadata, arm, observed, true)
	if mismatch {
		return fmt.Errorf("%s compiled behavior differs from interpreter; all captured scores retained", arm)
	}
	return nil
}

func applyObserved(metadata *report, arm string, observed map[string]map[string]bodydecision.Score, complete bool) bool {
	mismatch := false
	for i := range metadata.Cells {
		entry := &metadata.Cells[i]
		if entry.Arm == arm {
			scores := observed[entry.ID]
			entry.CompiledTraining = scores["training"]
			entry.CompiledHeldout = scores["heldout"]
			entry.GoTests = entry.CompiledTraining.Total + entry.CompiledHeldout.Total
			if entry.CompiledTraining != entry.Training || entry.CompiledHeldout != entry.Heldout {
				mismatch = true
			}
			entry.Status = "native_compiled_and_executed"
			if !complete {
				entry.Status = "native_runtime_partially_observed"
			}
		}
	}
	return mismatch
}

func lowerHex(value string, size int) bool {
	if len(value) != size {
		return false
	}
	for _, b := range value {
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f')) {
			return false
		}
	}
	return true
}

func checkLaya(endpoint, revision string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path != "/v1/systemone" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("unexpected Laya endpoint")
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return errors.New("Laya endpoint must use literal loopback address")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(endpoint, "/v1/systemone")+"/health", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(data) > 65536 || response.StatusCode != 200 {
		return errors.New("Laya health failed")
	}
	var health struct {
		Revisions map[string]string `json:"revisions"`
	}
	if err := json.Unmarshal(data, &health); err != nil {
		return err
	}
	if health.Revisions["multilingual"] != revision {
		return errors.New("Laya revision mismatch")
	}
	return nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.Len() {
		return 0, errors.New("capture byte budget exceeded")
	}
	return b.Buffer.Write(data)
}

func parseCases(output []byte, rows []record) (map[string]map[string]bodydecision.Score, error) {
	return parseCaseOutput(output, rows, true)
}

func parseCaseOutput(output []byte, rows []record, complete bool) (map[string]map[string]bodydecision.Score, error) {
	if !complete {
		if end := bytes.LastIndexByte(output, '\n'); end >= 0 {
			output = output[:end+1]
		} else {
			output = nil
		}
	}
	expected := make(map[string]map[string]int)
	scores := make(map[string]map[string]bodydecision.Score)
	seen := map[string]bool{}
	for _, row := range rows {
		expected[row.ID] = map[string]int{"training": len(row.Training), "heldout": len(row.Heldout)}
		scores[row.ID] = make(map[string]bodydecision.Score)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "GOOO_CASE\t") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 5 {
			return nil, errors.New("malformed Go case marker")
		}
		index, err := strconv.Atoi(parts[3])
		passed, parseErr := strconv.ParseBool(parts[4])
		count, exists := expected[parts[1]][parts[2]]
		key := strings.Join(parts[1:4], "\t")
		if err != nil || parseErr != nil || !exists || index < 0 || index >= count || seen[key] {
			return nil, errors.New("invalid, duplicate or unexpected Go case marker")
		}
		seen[key] = true
		score := scores[parts[1]][parts[2]]
		score.Total++
		if passed {
			score.Correct++
		}
		scores[parts[1]][parts[2]] = score
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if complete {
		for id, splits := range expected {
			for split, count := range splits {
				if scores[id][split].Total != count {
					return nil, fmt.Errorf("%s %s compiled Go case count incomplete", id, split)
				}
			}
		}
	}
	return scores, nil
}
