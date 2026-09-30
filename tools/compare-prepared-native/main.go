// compare-prepared-native pairs clean source-bound binaries on one frozen
// bilingual compound workload. It records partial outcomes, not only winners.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const baselineRevision = "9b8b900e32384b5397fb3d08dc545e22cc7530da"
const preparedRevision = "e155148f5ce114ee4c55e720a0e90cc34cc458ce"
const baselineSHA = "2545895533e9a1d4ab1e64c1b71f73595b9087c9a0d99c2fc0c78a841c112790"
const preparedSHA = "01a3be2815f260dedbc4353ae477cef114ad5521cc5524fb32f47f036eecc910"
const goSHA = "132b69336a1f809932a8a20b0201dbbb980e86e3a323ae32e893639d83d71598"

var metadataPins = map[string]string{
	"fp32":        "1ea3bada068f2487418f17270db4ba6785a98c3ad60e0ad7eb92359400a01bb5",
	"ptq_ternary": "7c4eb4068d76e83620a9b8e7ce7f8d948b5e2436d44c7b79cf241da609dab894",
	"qat_ternary": "368e37e7899cecba03874b69e933525a2f8c90ecc53d787d8e698de6aa21c087",
}

type document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Budget int                 `json:"max_attempts"`
}
type reply struct {
	Source string `json:"source"`
	Report struct {
		Decision  string `json:"decision"`
		Compiler  string `json:"compiler_source_sha"`
		ID        string `json:"activity_id"`
		Typecheck bool   `json:"typecheck_passed"`
		Replay    bool   `json:"deterministic_replay"`
		Writes    int    `json:"repository_writes"`
		Paths     struct {
			Matched    bool                  `json:"source_base_matched"`
			Search     pathplan.SearchResult `json:"search"`
			Functional float64               `json:"finite_functional_completeness_percent"`
			Timing     struct {
				Total    float64 `json:"total_ms"`
				Prepare  float64 `json:"plan_prepare_ms"`
				Binding  float64 `json:"source_binding_ms"`
				Load     float64 `json:"model_load_ms"`
				Search   float64 `json:"bounded_search_ms"`
				Emission float64 `json:"final_emission_ms"`
			} `json:"timing"`
		} `json:"body_paths"`
	} `json:"report"`
}
type process struct {
	WallNS     int64   `json:"wall_ns"`
	UserNS     int64   `json:"user_cpu_ns"`
	SystemNS   int64   `json:"system_cpu_ns"`
	RSS        int64   `json:"lifetime_peak_rss_bytes"`
	RSSKnown   bool    `json:"rss_known"`
	CPUPercent float64 `json:"one_core_cpu_percent"`
}
type cell struct {
	ID                string                `json:"id"`
	Pair              string                `json:"pair"`
	Version           string                `json:"version"`
	Arm               string                `json:"arm"`
	Language          string                `json:"language"`
	Budget            int                   `json:"budget"`
	Repetition        int                   `json:"repetition"`
	Position          int                   `json:"within_pair_position"`
	SourceSHA         string                `json:"generated_go_sha256"`
	ReplySHA          string                `json:"native_stdout_sha256"`
	Search            pathplan.SearchResult `json:"search"`
	Functional        float64               `json:"finite_contract_percent"`
	StructuralMatched int                   `json:"authored_structural_labels_matched"`
	StructuralTotal   int                   `json:"authored_structural_labels_total"`
	StageMS           float64               `json:"native_total_stage_ms"`
	Process           process               `json:"process"`
	HoldoutPassed     int                   `json:"independent_arithmetic_passed"`
	HoldoutTotal      int                   `json:"independent_arithmetic_total"`
}
type limited struct{ bytes.Buffer }

func (b *limited) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 2<<20 {
		return 0, errors.New("bounded process output exceeded")
	}
	return b.Buffer.Write(raw)
}
func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func read(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, errors.New("bounded regular file required")
	}
	return os.ReadFile(path)
}
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func checkBinary(path, sha, revision string) error {
	raw, err := read(path, 128<<20)
	if err != nil || hash(raw) != sha {
		return errors.New("binary digest mismatch")
	}
	info, err := buildinfo.ReadFile(path)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("Go build identity mismatch")
	}
	if revision != "" {
		settings := map[string]string{}
		for _, item := range info.Settings {
			settings[item.Key] = item.Value
		}
		if settings["vcs.revision"] != revision || settings["vcs.modified"] != "false" {
			return errors.New("clean compiler source identity mismatch")
		}
	}
	return nil
}
func child(ctx context.Context, dir, binary string, args ...string) ([]byte, []byte, process, error) {
	boundedContext, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(boundedContext, binary, args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "GOPROXY=off", "GOSUMDB=off", "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY="}
	for _, name := range []string{"HOME", "TMPDIR"} {
		if value := os.Getenv(name); value != "" {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	var out, stderr limited
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	started := time.Now()
	err := cmd.Run()
	metrics := process{WallNS: time.Since(started).Nanoseconds()}
	if cmd.ProcessState != nil {
		metrics.UserNS = cmd.ProcessState.UserTime().Nanoseconds()
		metrics.SystemNS = cmd.ProcessState.SystemTime().Nanoseconds()
		metrics.RSS, metrics.RSSKnown = peakRSS(cmd.ProcessState)
		metrics.CPUPercent = 100 * float64(metrics.UserNS+metrics.SystemNS) / float64(metrics.WallNS)
	}
	return out.Bytes(), stderr.Bytes(), metrics, err
}
func median(values []float64) float64 {
	values = append([]float64(nil), values...)
	sort.Float64s(values)
	if len(values) == 0 {
		return 0
	}
	if len(values)%2 == 1 {
		return values[len(values)/2]
	}
	return (values[len(values)/2-1] + values[len(values)/2]) / 2
}
func gold(input int64) int64 {
	if input > 0 && input <= 5 {
		return 2*input - 20
	}
	return 2*input + 10
}

// Normalize a deep copy: raw prediction timings remain in the observations.
func searchSemantics(search pathplan.SearchResult) ([]byte, error) {
	raw, err := json.Marshal(search)
	if err != nil {
		return nil, err
	}
	var owned pathplan.SearchResult
	if err := json.Unmarshal(raw, &owned); err != nil {
		return nil, err
	}
	for i := range owned.Selection.Receipts {
		owned.Selection.Receipts[i].PredictNS = 0
	}
	return json.Marshal(owned)
}

func run(oldBinary, newBinary, goBinary, output, revision string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh output required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("runner revision mismatch")
	}
	status, statusErr := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", "tools/compare-prepared-native", "studies/conditional-paths-v1", "internal/pathplan", "internal/bodyplan", "internal/decision").Output()
	if statusErr != nil || len(status) != 0 {
		return errors.New("runner inputs must be committed")
	}
	for _, binary := range [][3]string{{oldBinary, baselineSHA, baselineRevision}, {newBinary, preparedSHA, preparedRevision}, {goBinary, goSHA, ""}} {
		if err := checkBinary(binary[0], binary[1], binary[2]); err != nil {
			return err
		}
	}
	cohort := filepath.Join("studies", "conditional-paths-v1", "cohort")
	manifestRaw, err := read(filepath.Join(cohort, "manifest.json"), 128<<10)
	if err != nil {
		return err
	}
	var manifest struct {
		Files        map[string]string `json:"fixture_files_sha256"`
		Decisions    int               `json:"decisions_per_plan"`
		Combinations int               `json:"declared_combinations"`
	}
	if json.Unmarshal(manifestRaw, &manifest) != nil || manifest.Decisions != 6 || manifest.Combinations != 64 || len(manifest.Files) != 7 {
		return errors.New("fixed cohort manifest mismatch")
	}
	for name, sha := range manifest.Files {
		raw, err := read(filepath.Join(cohort, name), 128<<10)
		if err != nil || hash(raw) != sha {
			return errors.New("cohort digest mismatch")
		}
	}
	sourcePath, err := filepath.Abs(filepath.Join(cohort, "conditional-assignment.gooo.fixture"))
	if err != nil {
		return err
	}
	var oracle map[string]string
	oracleRaw, _ := read(filepath.Join(cohort, "structural-oracle.json"), 128<<10)
	if json.Unmarshal(oracleRaw, &oracle) != nil || len(oracle) != 6 {
		return errors.New("structural oracle mismatch")
	}
	var holdout []pathplan.TestCase
	holdoutRaw, _ := read(filepath.Join(cohort, "holdout-cases.json"), 128<<10)
	if json.Unmarshal(holdoutRaw, &holdout) != nil || len(holdout) != 9 {
		return errors.New("holdout mismatch")
	}
	for _, item := range holdout {
		if item.Expected != gold(item.Input) {
			return errors.New("independent arithmetic oracle mismatch")
		}
	}
	models := map[string]string{}
	modelEvidence := map[string]any{}
	for arm, pin := range metadataPins {
		path := filepath.Join("runs", "typed-path-positioned-random-20261001", "models", arm, "model.json")
		model, err := decision.LoadPath(path)
		if err != nil || model.MetadataSHA256() != pin {
			return errors.New("frozen model mismatch")
		}
		models[arm], err = filepath.Abs(path)
		if err != nil {
			return err
		}
		modelEvidence[arm] = map[string]string{"metadata_sha256": pin, "weights_sha256": model.WeightsSHA256()}
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	testRoot, err := os.MkdirTemp("", "gooo-prepared-native-execution-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(testRoot)
	if err := os.WriteFile(filepath.Join(testRoot, "go.mod"), []byte("module gooo.prepared.native\n\ngo 1.27.1\n"), 0644); err != nil {
		return err
	}
	type design struct {
		language, arm string
		budget        int
	}
	var designs []design
	for _, language := range []string{"en", "ko"} {
		for _, arm := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
			for _, budget := range []int{8, 64} {
				designs = append(designs, design{language, arm, budget})
			}
		}
	}
	random := rand.New(rand.NewPCG(20261001, 6))
	var planned []string
	for repetition := 0; repetition < 5; repetition++ {
		random.Shuffle(len(designs), func(i, j int) { designs[i], designs[j] = designs[j], designs[i] })
		for i, item := range designs {
			versions := []string{"baseline", "prepared"}
			if (i+repetition)%2 == 1 {
				versions[0], versions[1] = versions[1], versions[0]
			}
			pair := fmt.Sprintf("r%d-%s-%s-b%d", repetition, item.language, item.arm, item.budget)
			for _, version := range versions {
				planned = append(planned, pair+"-"+version)
			}
		}
	}
	runner, err := os.Executable()
	if err != nil {
		return err
	}
	runnerRaw, err := read(runner, 128<<20)
	if err != nil {
		return err
	}
	if err := checkBinary(runner, hash(runnerRaw), revision); err != nil {
		return err
	}
	pre := map[string]any{"schema": "gooo/prepared-native-comparison-preexecution/v1", "runner_revision": revision, "runner_binary_sha256": hash(runnerRaw), "baseline_revision": baselineRevision, "baseline_binary_sha256": baselineSHA, "prepared_revision": preparedRevision, "prepared_binary_sha256": preparedSHA, "go_binary_sha256": goSHA, "cohort_manifest_sha256": hash(manifestRaw), "models": modelEvidence, "planned_native_calls": 160, "planned_model_predictions": 720, "external_calls": 0, "optimizer_steps": 0, "execution_order": planned, "repetitions": 5, "balanced_old_first_pairs": 40, "balanced_new_first_pairs": 40, "scope": "One six-decision compound intent; bilingual/model/budget/repeated views are not independent ideas. Structural target labels and nine arithmetic inputs are evaluated after selection, never supplied to model APIs. No warmup calls. Per-stage timing excludes process startup; child RSS is lifetime high-water; CPU percent is relative to one core, not host utilization delta."}
	if err := save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	var rows []cell
	pairReference := map[string]cell{}
	totalCalls := 0
	for index, id := range planned {
		var repetition, budget int
		var language, arm, version string
		parts := strings.Split(id, "-")
		if len(parts) != 5 {
			return errors.New("planned identifier malformed")
		}
		if _, err := fmt.Sscanf(parts[0], "r%d", &repetition); err != nil {
			return err
		}
		language, arm, version = parts[1], parts[2], parts[4]
		if _, err := fmt.Sscanf(parts[3], "b%d", &budget); err != nil {
			return err
		}
		pair := strings.Join(parts[:4], "-")
		dir := filepath.Join(output, id)
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
		documentPath, err := filepath.Abs(filepath.Join(cohort, fmt.Sprintf("%s-budget-%d.json", language, budget)))
		if err != nil {
			return err
		}
		docRaw, err := read(documentPath, 128<<10)
		if err != nil {
			return err
		}
		var doc document
		if json.Unmarshal(docRaw, &doc) != nil {
			return errors.New("document decode failed")
		}
		binary, nativeSHA := oldBinary, baselineRevision
		if version == "prepared" {
			binary, nativeSHA = newBinary, preparedRevision
		}
		args := []string{"body-codegen", "--json", "--path-plan", documentPath, "--activity", "ConditionalAssign"}
		expectedCalls := 0
		if arm != "offline" {
			args = append(args, "--path-model", models[arm])
			expectedCalls = 6
		}
		args = append(args, sourcePath)
		stdout, stderr, metrics, err := child(ctx, ".", binary, args...)
		if writeErr := os.WriteFile(filepath.Join(dir, "native-stdout.json"), stdout, 0644); writeErr != nil {
			return writeErr
		}
		if len(stderr) > 0 {
			if writeErr := os.WriteFile(filepath.Join(dir, "native-stderr.txt"), stderr, 0644); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			return errors.New("native invocation failed; raw output retained")
		}
		var result reply
		if json.Unmarshal(stdout, &result) != nil {
			return errors.New("native JSON failed")
		}
		search := result.Report.Paths.Search
		if result.Report.Decision != "PASS" || result.Report.Compiler != nativeSHA || !result.Report.Typecheck || !result.Report.Replay || result.Report.Writes != 0 || !result.Report.Paths.Matched || search.Selection.ModelCalls != expectedCalls || search.Selection.ExternalCalls != 0 || !search.Selection.ExternalCallsKnown || search.DeclaredCombinations != 64 || len(search.Attempts) > budget || search.TrainingTotal != 7 || len(search.Selection.Choices) != 6 {
			return errors.New("native contract or accounting mismatch")
		}
		if version == "prepared" && result.Report.Paths.Timing.Prepare <= 0 {
			return errors.New("new preparation stage not observed")
		}
		if arm != "offline" && (search.Selection.MetadataSHA256 != metadataPins[arm] ||
			search.Selection.WeightsSHA256 != modelEvidence[arm].(map[string]string)["weights_sha256"] ||
			search.Selection.ModelVariant != arm) {
			return errors.New("native selected a different model")
		}
		selected, err := pathplan.Compile(doc.Plan, search.Selection.Choices)
		if err != nil {
			return err
		}
		generated := []byte(result.Source)
		if err := os.WriteFile(filepath.Join(dir, "generated.go.txt"), generated, 0644); err != nil {
			return err
		}
		row := cell{ID: id, Pair: pair, Version: version, Arm: arm, Language: language, Budget: budget, Repetition: repetition, Position: index % 2, SourceSHA: hash(generated), ReplySHA: hash(stdout), Search: search, Functional: result.Report.Paths.Functional, StructuralTotal: 6, StageMS: result.Report.Paths.Timing.Total, Process: metrics, HoldoutTotal: 9}
		for key, want := range oracle {
			if search.Selection.Choices[key] == want {
				row.StructuralMatched++
			}
		}
		if first, ok := pairReference[pair]; ok {
			beforeRaw, err := searchSemantics(first.Search)
			if err != nil {
				return err
			}
			afterRaw, err := searchSemantics(row.Search)
			if err != nil {
				return err
			}
			if first.SourceSHA != row.SourceSHA || !bytes.Equal(beforeRaw, afterRaw) || first.Functional != row.Functional {
				return errors.New("paired semantic search or source differs")
			}
		} else {
			copyRaw, _ := json.Marshal(row)
			var copy cell
			_ = json.Unmarshal(copyRaw, &copy)
			pairReference[pair] = copy
		}
		// Test every exact emitted source. Arena parity is a hard check; independent
		// intent arithmetic is recorded, including failures from short budgets.
		parsed, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, parser.PackageClauseOnly)
		if err != nil {
			return err
		}
		pkg := filepath.Join(testRoot, id)
		if err := os.Mkdir(pkg, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(pkg, "generated.go"), generated, 0644); err != nil {
			return err
		}
		var test bytes.Buffer
		fmt.Fprintf(&test, "package %s\nimport \"testing\"\nfunc TestExactNativeObservation(t *testing.T){\n", parsed.Name.Name)
		for _, item := range holdout {
			value, err := selected.Evaluate(item.Input)
			if err != nil {
				return err
			}
			fmt.Fprintf(&test, "{input:=int64(%d);got:=ConditionalAssign(input);if got!=int64(%d){t.Fatalf(\"arena mismatch input=%%d got=%%d\",input,got)};expected:=int64(%d);t.Logf(\"GOOO_OBSERVATION {\\\"cell\\\":\\\"%s\\\",\\\"input\\\":%%d,\\\"expected\\\":%%d,\\\"actual\\\":%%d,\\\"passed\\\":%%t}\",input,expected,got,got==expected)}\n", item.Input, value.Int, item.Expected, id)
		}
		test.WriteString("}\n")
		if err := os.WriteFile(filepath.Join(pkg, "generated_test.go"), test.Bytes(), 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "independent-test.go.txt"), test.Bytes(), 0644); err != nil {
			return err
		}
		if err := save(filepath.Join(dir, "process.json"), metrics); err != nil {
			return err
		}
		totalCalls += search.Selection.ModelCalls
		rows = append(rows, row)
	}
	stdout, stderr, goProcess, err := child(ctx, testRoot, goBinary, "test", "-count=1", "-json", "./...")
	if writeErr := os.WriteFile(filepath.Join(output, "independent-go-tests.jsonl"), stdout, 0644); writeErr != nil {
		return writeErr
	}
	if len(stderr) > 0 {
		_ = os.WriteFile(filepath.Join(output, "independent-go-stderr.txt"), stderr, 0644)
	}
	if err != nil {
		return errors.New("exact Go execution failed; outputs retained")
	}
	byID := map[string]*cell{}
	for i := range rows {
		byID[rows[i].ID] = &rows[i]
	}
	observed := map[string]map[int64]bool{}
	packages, tests, observations := 0, 0, 0
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	scanner.Buffer(make([]byte, 8192), 2<<20)
	for scanner.Scan() {
		var event struct{ Action, Test, Output string }
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			return errors.New("Go test JSON invalid")
		}
		if event.Action == "pass" {
			if event.Test == "" {
				packages++
			} else if event.Test == "TestExactNativeObservation" {
				tests++
			}
		}
		position := strings.Index(event.Output, "GOOO_OBSERVATION ")
		if position < 0 {
			continue
		}
		var item struct {
			Cell     string `json:"cell"`
			Input    int64  `json:"input"`
			Expected int64  `json:"expected"`
			Actual   int64  `json:"actual"`
			Passed   bool   `json:"passed"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(event.Output[position+len("GOOO_OBSERVATION "):])), &item) != nil {
			return errors.New("arithmetic observation decode failed")
		}
		row := byID[item.Cell]
		if row == nil || item.Expected != gold(item.Input) || item.Passed != (item.Actual == item.Expected) {
			return errors.New("arithmetic observation contract failed")
		}
		if observed[item.Cell] == nil {
			observed[item.Cell] = map[int64]bool{}
		}
		if observed[item.Cell][item.Input] {
			return errors.New("duplicate arithmetic observation")
		}
		observed[item.Cell][item.Input] = true
		if item.Passed {
			row.HoldoutPassed++
		}
		observations++
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	if len(rows) != 160 || totalCalls != 720 || packages != 160 || tests != 160 || observations != 1440 {
		return errors.New("exact planned accounting did not match")
	}
	var summaries []map[string]any
	for _, version := range []string{"baseline", "prepared"} {
		var stages, walls, cpus, rss []float64
		passed, structural := 0, 0
		for _, row := range rows {
			if row.Version != version {
				continue
			}
			if len(observed[row.ID]) != 9 {
				return errors.New("missing arithmetic observations")
			}
			stages = append(stages, row.StageMS)
			walls = append(walls, float64(row.Process.WallNS)/1e6)
			cpus = append(cpus, row.Process.CPUPercent)
			if row.Process.RSSKnown {
				rss = append(rss, float64(row.Process.RSS))
			}
			passed += row.HoldoutPassed
			structural += row.StructuralMatched
		}
		summaries = append(summaries, map[string]any{"version": version, "median_native_total_stage_ms": median(stages), "median_process_wall_ms": median(walls), "median_one_core_cpu_percent": median(cpus), "median_lifetime_child_rss_bytes": median(rss), "independent_arithmetic_passed": passed, "independent_arithmetic_total": 720, "selected_authored_structure_labels_matched": structural, "selected_authored_structure_labels_total": 480})
	}
	var differences []float64
	for i := 0; i < len(rows); i += 2 {
		old, new := rows[i], rows[i+1]
		if old.Version != "baseline" {
			old, new = new, old
		}
		differences = append(differences, old.StageMS-new.StageMS)
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/prepared-native-comparison/v1", "runner_revision": revision, "native_calls": len(rows), "model_predictions": totalCalls, "external_calls": 0, "optimizer_steps": 0, "exact_go_arena_parity_tests_passed": tests, "arithmetic_observations": observations, "all_80_pairs_same_source_and_search": true, "median_paired_native_stage_saving_ms": median(differences), "summaries": summaries, "independent_go_test_process": goProcess, "cells": rows, "scope": "Single local balanced-order five-repetition comparison on one compound intent. It does not establish universal speedup, unseen-language accuracy, calibrated completeness, host CPU delta or packed-kernel efficiency. Final authored structural-label agreement is separate from finite functional success."})
}
func main() {
	old := flag.String("baseline", "", "clean old compiler")
	prepared := flag.String("prepared", "", "clean prepared compiler")
	goBinary := flag.String("go", "", "Go 1.27.1 executable")
	output := flag.String("output", "", "fresh output")
	revision := flag.String("source-revision", "", "clean runner source SHA")
	flag.Parse()
	if err := run(*old, *prepared, *goBinary, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, "compare-prepared-native:", err)
		os.Exit(1)
	}
	fmt.Println("PASS: 160 native calls, 720 local predictions, 1440 independent observations; no training or external calls")
}
