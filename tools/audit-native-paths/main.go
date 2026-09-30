// audit-native-paths observes fresh in-process native structural predictions,
// bounded TDD and independently compiled exact emitted Go. It does not train.
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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

const nativeRevision = "d21ce275ec0832e38ad4962ad03f4546c1b46e9f"
const compilerSHA = "ab1c8b899b081ceb3e7d67eff36985856bead3ab15788d4cba9e0bd6e2e87aa4"
const goSHA = "132b69336a1f809932a8a20b0201dbbb980e86e3a323ae32e893639d83d71598"
const publicModelRevision = "9217041694904c2ef41dde9d8834b9fffb1eb7eb"

var modelMetadataPins = map[string]string{
	"fp32":        "1ea3bada068f2487418f17270db4ba6785a98c3ad60e0ad7eb92359400a01bb5",
	"ptq_ternary": "7c4eb4068d76e83620a9b8e7ce7f8d948b5e2436d44c7b79cf241da609dab894",
	"qat_ternary": "368e37e7899cecba03874b69e933525a2f8c90ecc53d787d8e698de6aa21c087",
}

type document struct {
	Schema      string              `json:"schema"`
	Plan        pathplan.Plan       `json:"path_plan"`
	Cases       []pathplan.TestCase `json:"test_cases"`
	MaxAttempts int                 `json:"max_attempts"`
}
type reply struct {
	Source string `json:"source"`
	Report struct {
		Decision  string `json:"decision"`
		Compiler  string `json:"compiler_source_sha"`
		ID        string `json:"activity_id"`
		Replay    bool   `json:"deterministic_replay"`
		Typecheck bool   `json:"typecheck_passed"`
		Writes    int    `json:"repository_writes"`
		Paths     struct {
			SourceMatched bool                  `json:"source_base_matched"`
			Search        pathplan.SearchResult `json:"search"`
			Functional    float64               `json:"finite_functional_completeness_percent"`
			Timing        struct {
				Total  float64 `json:"total_ms"`
				Load   float64 `json:"model_load_ms"`
				Search float64 `json:"bounded_search_ms"`
			} `json:"timing"`
		} `json:"body_paths"`
	} `json:"report"`
}
type observation struct {
	Cell     string `json:"cell"`
	Input    int64  `json:"input"`
	Expected int64  `json:"expected"`
	Actual   int64  `json:"actual"`
	Passed   bool   `json:"passed"`
}
type cell struct {
	ID              string                `json:"id"`
	Arm             string                `json:"arm"`
	Language        string                `json:"language"`
	Contract        string                `json:"contract"`
	Budget          int                   `json:"max_attempts"`
	NativeSourceSHA string                `json:"gooo_input_sha256"`
	DocumentSHA     string                `json:"input_document_sha256"`
	ReceiptSHA      string                `json:"native_receipt_sha256"`
	GeneratedSHA    string                `json:"generated_go_sha256"`
	Process         process               `json:"native_process"`
	Search          pathplan.SearchResult `json:"search"`
	Functional      float64               `json:"declared_finite_functional_completeness_percent"`
	TotalMS         float64               `json:"native_total_stage_ms"`
	LoadMS          float64               `json:"native_model_load_ms"`
	SearchMS        float64               `json:"native_bounded_search_ms"`
	HoldoutPassed   int                   `json:"independent_gold_cases_passed"`
	HoldoutTotal    int                   `json:"independent_gold_cases_total"`
}
type process struct {
	WallNS     int64   `json:"wall_ns"`
	UserNS     int64   `json:"user_cpu_ns"`
	SystemNS   int64   `json:"system_cpu_ns"`
	PeakRSS    int64   `json:"lifetime_peak_rss_bytes"`
	RSSKnown   bool    `json:"rss_known"`
	CPUPercent float64 `json:"one_core_cpu_percent_over_wall"`
}
type bounded struct{ bytes.Buffer }

func (b *bounded) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 1<<20 {
		return 0, errors.New("process output budget exceeded")
	}
	return b.Buffer.Write(raw)
}
func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func read(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, errors.New("bounded regular input required")
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
func child(dir, binary string, args ...string) ([]byte, []byte, process, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "GOPROXY=off", "GOSUMDB=off", "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY="}
	for _, name := range []string{"HOME", "TMPDIR"} {
		if value := os.Getenv(name); value != "" {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	var stdout, stderr bounded
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	metrics := process{WallNS: time.Since(start).Nanoseconds()}
	if cmd.ProcessState != nil {
		metrics.UserNS = cmd.ProcessState.UserTime().Nanoseconds()
		metrics.SystemNS = cmd.ProcessState.SystemTime().Nanoseconds()
		metrics.PeakRSS, metrics.RSSKnown = peakRSS(cmd.ProcessState)
		metrics.CPUPercent = 100 * float64(metrics.UserNS+metrics.SystemNS) / float64(metrics.WallNS)
	}
	return stdout.Bytes(), stderr.Bytes(), metrics, err
}
func gold(input int64) int64 {
	result := 5*input + 2
	if input < 0 {
		return result - 3
	}
	return result + 3
}
func median(values []float64) float64 {
	copy := append([]float64(nil), values...)
	sort.Float64s(copy)
	if len(copy) == 0 {
		return 0
	}
	if len(copy)%2 == 1 {
		return copy[len(copy)/2]
	}
	return (copy[len(copy)/2-1] + copy[len(copy)/2]) / 2
}
func run(compiler, goBinary, fixtureRoot, modelRoot, output, revision, compilerPin, goPin string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	head, headErr := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	diffErr := exec.CommandContext(ctx, "git", "diff", "--quiet", "HEAD", "--", "tools/audit-native-paths", "internal/pathplan", "internal/bodyplan", "internal/decision").Run()
	cancel()
	if headErr != nil || diffErr != nil || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) || strings.TrimSpace(string(head)) != revision {
		return errors.New("runner revision must equal clean tracked HEAD")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh output required")
	}
	for _, pin := range [][2]string{{compiler, compilerPin}, {goBinary, goPin}} {
		if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(pin[1]) {
			return errors.New("explicit binary SHA256 required")
		}
		raw, err := read(pin[0], 128<<20)
		if err != nil || hash(raw) != pin[1] {
			return errors.New("binary pin mismatch")
		}
		info, err := buildinfo.ReadFile(pin[0])
		if err != nil || info.GoVersion != "go1.27.1" {
			return errors.New("toolchain mismatch")
		}
	}
	nativeInfo, _ := buildinfo.ReadFile(compiler)
	nativeSettings := map[string]string{}
	for _, setting := range nativeInfo.Settings {
		nativeSettings[setting.Key] = setting.Value
	}
	if nativeSettings["vcs.revision"] != nativeRevision || nativeSettings["vcs.modified"] != "false" {
		return errors.New("native source binding mismatch")
	}
	source, err := read(filepath.Join(fixtureRoot, "typed-path-compound.gooo.fixture"), 128<<10)
	if err != nil {
		return err
	}
	documentRaw, err := read(filepath.Join(fixtureRoot, "typed-path-compound-plan.json"), 128<<10)
	if err != nil {
		return err
	}
	if hash(source) != "646c7fd40184058df2c1c2f49cd6c015e6e8136f02f0ced949cfdd0a6d930642" ||
		hash(documentRaw) != "6534e600b45240383363323ad88501d5d20c4c45d81a363f31378e80d072e6af" {
		return errors.New("native committed fixture bytes differ")
	}
	var base document
	if err := strictjson.Decode(documentRaw, &base); err != nil {
		return err
	}
	if base.Schema != "gooo/body-codegen-typed-path-plan/v1" || len(base.Plan.Decisions) != 3 || len(base.Cases) != 3 || base.MaxAttempts != 8 {
		return errors.New("fixed fixture contract mismatch")
	}
	models := map[string]map[string]string{}
	for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		model, err := decision.LoadPath(filepath.Join(modelRoot, arm, "model.json"))
		if err != nil {
			return err
		}
		if model.MetadataSHA256() != modelMetadataPins[arm] {
			return errors.New("model is not the fixed reviewed public snapshot")
		}
		models[arm] = map[string]string{"metadata_sha256": model.MetadataSHA256(), "weights_sha256": model.WeightsSHA256(), "variant": model.Variant()}
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeRaw, err := read(exe, 128<<20)
	if err != nil {
		return err
	}
	if err := save(filepath.Join(output, "preexecution.json"), map[string]any{
		"schema": "gooo/native-typed-path-preexecution/v1", "runner_revision": revision, "runner_binary_sha256": hash(exeRaw),
		"native_revision": nativeRevision, "native_binary_sha256": compilerPin, "go_binary_sha256": goPin,
		"hf_model_revision": publicModelRevision, "models": models, "gooo_fixture_sha256": hash(source), "document_fixture_sha256": hash(documentRaw),
		"planned_native_calls": 32, "planned_local_model_predictions": 72, "planned_external_calls": 0,
		"design":  "one compound body, 3 interacting choices/8 combinations; two languages x four arms x budgets 4/8 x full/partial contracts; at most two scope-invalid combinations, so budget4 permits a partial emitted candidate; variants and views are not independent ideas",
		"holdout": "ten integer inputs absent from search API; evaluation probe, not untouched natural-language holdout; weights frozen and not trained here",
	}); err != nil {
		return err
	}
	testRoot, err := os.MkdirTemp("", "gooo-native-path-execution-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(testRoot)
	if err := os.WriteFile(filepath.Join(testRoot, "go.mod"), []byte("module gooo.native.paths\n\ngo 1.27.1\n"), 0644); err != nil {
		return err
	}
	inputs := []int64{-100, -9, -1, 1, 2, 7, 100, 2147483647, -9223372036854775808, 9223372036854775807}
	var cells []cell
	for _, language := range []string{"en", "ko"} {
		for _, arm := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
			for _, budget := range []int{4, 8} {
				for _, contract := range []string{"full", "partial"} {
					var doc document
					if err := json.Unmarshal(documentRaw, &doc); err != nil {
						return err
					}
					doc.MaxAttempts = budget
					texts := []string{"Choose the second local variable.", "Swap the then and else bodies.", "Keep the declaration order unchanged."}
					if language == "ko" {
						texts = []string{"두 번째 지역 변수를 선택해라.", "then과 else 바디를 서로 바꿔라.", "선언 순서를 그대로 유지해라."}
					}
					for i := range doc.Plan.Decisions {
						doc.Plan.Decisions[i].Intent = "Gooo typed compound body. intent: " + texts[i]
					}
					if contract == "partial" {
						doc.Cases[2].Expected = 999
					}
					id := fmt.Sprintf("%s-%s-%d-%s", language, arm, budget, contract)
					dir := filepath.Join(output, id)
					if err := os.Mkdir(dir, 0755); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(dir, "input.gooo"), source, 0644); err != nil {
						return err
					}
					if err := save(filepath.Join(dir, "plan.json"), doc); err != nil {
						return err
					}
					planRaw, err := read(filepath.Join(dir, "plan.json"), 128<<10)
					if err != nil {
						return err
					}
					args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--activity", "Combined", "input.gooo"}
					if arm != "offline" {
						modelPath, err := filepath.Abs(filepath.Join(modelRoot, arm, "model.json"))
						if err != nil {
							return err
						}
						args = append(args, "--path-model", modelPath)
					}
					stdout, stderr, metrics, execErr := child(dir, compiler, args...)
					if err := os.WriteFile(filepath.Join(dir, "native-stdout.json"), stdout, 0644); err != nil {
						return err
					}
					if len(stderr) > 0 {
						if err := os.WriteFile(filepath.Join(dir, "native-stderr.txt"), stderr, 0644); err != nil {
							return err
						}
					}
					if execErr != nil {
						return errors.New("native execution failed; captures retained")
					}
					var result reply
					if err := json.Unmarshal(stdout, &result); err != nil {
						return err
					}
					expectedCalls := 0
					if arm != "offline" {
						expectedCalls = 3
					}
					search := result.Report.Paths.Search
					if result.Report.Compiler != nativeRevision || result.Report.Decision != "PASS" || !result.Report.Typecheck || !result.Report.Replay || result.Report.Writes != 0 || result.Report.ID != "sample://activity/combined" || !result.Report.Paths.SourceMatched || search.Selection.ModelCalls != expectedCalls || search.Selection.ExternalCalls != 0 || !search.Selection.ExternalCallsKnown || len(search.Attempts) > budget || search.DeclaredCombinations != 8 {
						return errors.New("native receipt contract failed")
					}
					if arm != "offline" && (search.Selection.MetadataSHA256 != models[arm]["metadata_sha256"] || search.Selection.WeightsSHA256 != models[arm]["weights_sha256"]) {
						return errors.New("native model binding differs")
					}
					selected, err := pathplan.Compile(doc.Plan, search.Selection.Choices)
					if err != nil {
						return err
					}
					generated := []byte(result.Source)
					if err := os.WriteFile(filepath.Join(dir, "generated.go.txt"), generated, 0644); err != nil {
						return err
					}
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
					fmt.Fprintf(&test, "package %s\nimport \"testing\"\nfunc TestIndependentObservation(t *testing.T){\n", parsed.Name.Name)
					for _, input := range inputs {
						selectedValue, err := selected.Evaluate(input)
						if err != nil {
							return err
						}
						fmt.Fprintf(&test, "{input:=int64(%d);got:=Combined(input);if got!=int64(%d){t.Fatalf(\"selected arena/native mismatch input=%%d got=%%d\",input,got)}; expected:=int64(%d); t.Logf(\"GOOO_OBSERVATION {\\\"cell\\\":\\\"%s\\\",\\\"input\\\":%%d,\\\"expected\\\":%%d,\\\"actual\\\":%%d,\\\"passed\\\":%%t}\",input,expected,got,got==expected)}\n", input, selectedValue.Int, gold(input), id)
					}
					test.WriteString("}\n")
					if err := os.WriteFile(filepath.Join(pkg, "generated_test.go"), test.Bytes(), 0644); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(dir, "independent_test.go.txt"), test.Bytes(), 0644); err != nil {
						return err
					}
					cells = append(cells, cell{ID: id, Arm: arm, Language: language, Contract: contract, Budget: budget, NativeSourceSHA: hash(source), DocumentSHA: hash(planRaw), ReceiptSHA: hash(stdout), GeneratedSHA: hash(generated), Process: metrics, Search: search, Functional: result.Report.Paths.Functional, TotalMS: result.Report.Paths.Timing.Total, LoadMS: result.Report.Paths.Timing.Load, SearchMS: result.Report.Paths.Timing.Search, HoldoutTotal: len(inputs)})
				}
			}
		}
	}
	stdout, stderr, goMetrics, err := child(testRoot, goBinary, "test", "-count=1", "-json", "./...")
	if writeErr := os.WriteFile(filepath.Join(output, "independent-go-tests.jsonl"), stdout, 0644); writeErr != nil {
		return writeErr
	}
	if len(stderr) > 0 {
		if writeErr := os.WriteFile(filepath.Join(output, "independent-go-stderr.txt"), stderr, 0644); writeErr != nil {
			return writeErr
		}
	}
	if err != nil {
		return errors.New("independent exact Go execution failed; captures retained")
	}
	byID := map[string]*cell{}
	for i := range cells {
		byID[cells[i].ID] = &cells[i]
	}
	observed := map[string]map[int64]bool{}
	caseTotal := 0
	parityTests := 0
	packages := 0
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	scanner.Buffer(make([]byte, 8192), 1<<20)
	for scanner.Scan() {
		var event struct {
			Action string
			Test   string
			Output string
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return err
		}
		if event.Action == "pass" {
			if event.Test == "" {
				packages++
			} else if event.Test == "TestIndependentObservation" {
				parityTests++
			}
		}
		index := strings.Index(event.Output, "GOOO_OBSERVATION ")
		if index < 0 {
			continue
		}
		var item observation
		if err := json.Unmarshal([]byte(strings.TrimSpace(event.Output[index+len("GOOO_OBSERVATION "):])), &item); err != nil {
			return err
		}
		row := byID[item.Cell]
		if row == nil || item.Expected != gold(item.Input) || item.Passed != (item.Actual == item.Expected) {
			return errors.New("invalid independent observation")
		}
		if observed[item.Cell] == nil {
			observed[item.Cell] = map[int64]bool{}
		}
		if observed[item.Cell][item.Input] {
			return errors.New("duplicate observation")
		}
		observed[item.Cell][item.Input] = true
		if item.Passed {
			row.HoldoutPassed++
		}
		caseTotal++
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	if packages != 32 || parityTests != 32 || caseTotal != 320 {
		return errors.New("exact Go execution accounting failed")
	}
	var summaries []map[string]any
	modelCalls := 0
	for _, arm := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
		var totals, loads, searches, predictions, cpu, rss []float64
		passed, total, attempts, typeRejected := 0, 0, 0, 0
		for _, row := range cells {
			if row.Arm != arm {
				continue
			}
			if len(observed[row.ID]) != 10 {
				return errors.New("missing observed inputs")
			}
			modelCalls += row.Search.Selection.ModelCalls
			totals = append(totals, row.TotalMS)
			loads = append(loads, row.LoadMS)
			searches = append(searches, row.SearchMS)
			cpu = append(cpu, row.Process.CPUPercent)
			if row.Process.RSSKnown {
				rss = append(rss, float64(row.Process.PeakRSS))
			}
			for _, receipt := range row.Search.Selection.Receipts {
				if receipt.PredictNS > 0 {
					predictions = append(predictions, float64(receipt.PredictNS))
				}
			}
			if row.Budget == 8 {
				passed += row.HoldoutPassed
				total += row.HoldoutTotal
				attempts += len(row.Search.Attempts)
				typeRejected += row.Search.TypeRejected
				if row.HoldoutPassed != 10 {
					return errors.New("full-budget compound selection did not meet independent gold")
				}
			}
		}
		summaries = append(summaries, map[string]any{"arm": arm, "cells": 8, "median_prediction_ns": median(predictions), "median_native_total_ms": median(totals), "median_model_load_ms": median(loads), "median_bounded_search_ms": median(searches), "median_native_child_cpu_percent_one_core": median(cpu), "median_native_child_peak_rss_bytes": median(rss), "full_budget_gold_passed": passed, "full_budget_gold_total": total, "full_budget_attempts": attempts, "full_budget_type_rejected": typeRejected})
	}
	if modelCalls != 72 {
		return errors.New("model call accounting failed")
	}
	if err := save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/native-typed-path-observation/v1", "runner_revision": revision, "native_revision": nativeRevision, "native_calls": len(cells), "local_model_predictions": modelCalls, "external_calls": 0, "external_calls_known": true, "independent_cases": caseTotal, "independent_package_passes": packages, "independent_parity_test_passes": parityTests, "go_test_process": goMetrics, "arms": summaries, "cells": cells, "scope": "one compound intent probe; independent arithmetic gold on 10 inputs per cell, exact emitted Go/selected arena parity; partial contracts retain wrong expectation; no claims of unseen NL accuracy, all-input proof or host CPU increase", "execution_order": "en,ko; offline,fp32,ptq,qat; attempts4,8; full,partial; single fixed order without matched repetition", "model_training_steps": 0, "extra_runner_predictions": 0, "gpu_inference": false}); err != nil {
		return err
	}
	fmt.Println("OBSERVED 32 native calls, 72 fresh in-process predictions, 320 independent arithmetic cases; 0 external calls/optimizer steps")
	return nil
}
func main() {
	compiler := flag.String("compiler", "", "native binary")
	goBinary := flag.String("go", "", "Go binary")
	fixtureRoot := flag.String("fixtures", "", "fixed native fixtures")
	modelRoot := flag.String("models", "runs/typed-path-positioned-random-20261001/models", "frozen model bundles")
	output := flag.String("output", "", "fresh evidence directory")
	revision := flag.String("source-revision", "", "actual clean runner HEAD")
	compilerPin := flag.String("compiler-sha", compilerSHA, "expected platform binary SHA256")
	goPin := flag.String("go-sha", goSHA, "expected Go binary SHA256")
	flag.Parse()
	if err := run(*compiler, *goBinary, *fixtureRoot, *modelRoot, *output, *revision, *compilerPin, *goPin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
