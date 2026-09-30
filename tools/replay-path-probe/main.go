// replay-path-probe sends saved model/TDD assembled Gooo to the real compiler.
// It makes no new model calls and independently executes exact emitted Go bytes.
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
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

const nativeSource = "68361e64def5457f0d0e6de972570a9885cceb96"
const nativeBinarySHA = "3f3e632affb55c35449fcdcd0c69fcd8dcd84ec42dcaa488641eb03dd2f2cb50"
const goBinarySHA = "132b69336a1f809932a8a20b0201dbbb980e86e3a323ae32e893639d83d71598"

type saved struct {
	Instruction struct {
		ID            string `json:"id"`
		Family        string `json:"family"`
		Configuration int    `json:"configuration"`
		Reverse       bool   `json:"reverse"`
		Language      string `json:"language"`
		View          string `json:"view"`
		Text          string `json:"text"`
	} `json:"instruction"`
	Search pathplan.SearchResult `json:"search"`
	GoSHA  string                `json:"go_source_sha256"`
}
type cell struct {
	ID               string             `json:"id"`
	Arm              string             `json:"arm"`
	InputSHA         string             `json:"gooo_input_sha256"`
	GeneratedSHA     string             `json:"generated_go_sha256"`
	NativeReceiptSHA string             `json:"native_receipt_sha256"`
	Selection        pathplan.Selection `json:"saved_selection"`
	FiniteCases      int                `json:"independent_finite_cases"`
	NativeWallNS     int64              `json:"native_wall_ns"`
}
type bounded struct{ bytes.Buffer }

func (b *bounded) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 1<<20 {
		return 0, errors.New("bounded process output exceeded")
	}
	return b.Buffer.Write(raw)
}
func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
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
func samples(raw []byte) ([]saved, error) {
	var rows []saved
	seen := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 8192), 128<<10)
	for scanner.Scan() {
		var row saved
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		if row.Instruction.Configuration != 64 || !row.Instruction.Reverse || row.Instruction.View != "plain" {
			continue
		}
		if row.Instruction.Language != "en" && row.Instruction.Language != "ko" {
			return nil, errors.New("invalid language")
		}
		key := row.Instruction.Family + "-" + row.Instruction.Language
		if seen[key] || row.Search.Status != "TRAINING_COMPLETE" {
			return nil, errors.New("duplicate or incomplete saved sample")
		}
		seen[key] = true
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(rows) != 10 {
		return nil, errors.New("expected five bilingual structural samples")
	}
	return rows, nil
}
func child(directory, binary string, args ...string) ([]byte, []byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = directory
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "GOTOOLCHAIN=local", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOENV=off"}
	for _, name := range []string{"HOME", "TMPDIR"} {
		if value := os.Getenv(name); value != "" {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	var stdout, stderr bounded
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), time.Since(start).Nanoseconds(), err
}
func run(compiler, goBinary, compilerSHA, goSHA, probeRoot, output, revision string) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("source revision required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	head, headErr := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	diffErr := exec.CommandContext(ctx, "git", "diff", "--quiet", "HEAD", "--", "tools/replay-path-probe", "internal/pathplan", "internal/pathstudy", "internal/bodyplan").Run()
	cancel()
	if headErr != nil || strings.TrimSpace(string(head)) != revision || diffErr != nil {
		return errors.New("declared source must match the clean tracked runner checkout before execution")
	}
	for _, pin := range []string{compilerSHA, goSHA} {
		if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(pin) {
			return errors.New("explicit binary SHA-256 required")
		}
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh output required")
	}
	for _, binding := range [][2]string{{compiler, compilerSHA}, {goBinary, goSHA}} {
		raw, err := read(binding[0], 128<<20)
		if err != nil || digest(raw) != binding[1] {
			return errors.New("fixed compiler/Go binary mismatch")
		}
		info, err := buildinfo.ReadFile(binding[0])
		if err != nil || info.GoVersion != "go1.27.1" {
			return errors.New("Go toolchain version mismatch")
		}
	}
	sampled := map[string][]saved{}
	inputBindings := map[string]string{}
	for _, arm := range []string{"offline", "fp32"} {
		raw, err := read(filepath.Join(probeRoot, arm+"-rows.jsonl"), 32<<20)
		if err != nil {
			return err
		}
		rows, err := samples(raw)
		if err != nil {
			return err
		}
		sampled[arm] = rows
		inputBindings[arm] = digest(raw)
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	runnerRaw, err := read(executable, 128<<20)
	if err != nil {
		return err
	}
	if err := save(filepath.Join(output, "runner-binding.json"), map[string]any{"runner_source_revision": revision, "runner_binary_sha256": digest(runnerRaw), "tracked_source_clean": true}); err != nil {
		return err
	}
	if err := save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/typed-path-native-replay-preexecution/v1", "runner_source_revision": revision, "native_source_revision": nativeSource, "native_binary_sha256": compilerSHA, "go_binary_sha256": goSHA, "saved_rows_sha256": inputBindings, "planned_native_calls": 20, "planned_local_model_predictions": 0, "planned_external_calls": 0, "scope": "Saved offline/FP32 selections, five families × two languages × two arms; each pair has the same finite-test selected function. This is structural assembly before native codegen, not native model inference."}); err != nil {
		return err
	}
	replay, err := os.MkdirTemp("", "gooo-path-native-replay-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(replay)
	if err := os.WriteFile(filepath.Join(replay, "go.mod"), []byte("module gooo.path.replay\n\ngo 1.27.1\n"), 0644); err != nil {
		return err
	}
	var cells []cell
	unique := map[string]bool{}
	for _, arm := range []string{"offline", "fp32"} {
		for _, row := range sampled[arm] {
			plan, err := pathstudy.Fixture(row.Instruction.Family, 64, row.Instruction.Text)
			if err != nil {
				return err
			}
			program, err := pathplan.Compile(plan, row.Search.Selection.Choices)
			if err != nil {
				return err
			}
			if digest([]byte(program.GoSource())) != row.GoSHA {
				return errors.New("saved selection no longer reproduces its exact Go projection")
			}
			id := arm + "-" + row.Instruction.Family + "-" + row.Instruction.Language
			dir := filepath.Join(output, id)
			if err := os.Mkdir(dir, 0755); err != nil {
				return err
			}
			source := []byte(program.GoooSource())
			if err := os.WriteFile(filepath.Join(dir, "input.gooo"), source, 0644); err != nil {
				return err
			}
			stdout, stderr, wall, execErr := child(dir, compiler, "body-codegen", "--json", "--activity", "ChoosePath", "input.gooo")
			if err := os.WriteFile(filepath.Join(dir, "native-stdout.json"), stdout, 0644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "native-stderr.txt"), stderr, 0644); err != nil {
				return err
			}
			if execErr != nil {
				return errors.New("native execution failed; raw capture retained")
			}
			var reply struct {
				Source string `json:"source"`
				Report struct {
					Decision  string `json:"decision"`
					Typecheck bool   `json:"typecheck_passed"`
					Replay    bool   `json:"deterministic_replay"`
					Compiler  string `json:"compiler_source_sha"`
				} `json:"report"`
			}
			if err := json.Unmarshal(stdout, &reply); err != nil {
				return err
			}
			if reply.Report.Decision != "PASS" || !reply.Report.Typecheck || !reply.Report.Replay || reply.Report.Compiler != nativeSource || reply.Source == "" {
				return errors.New("native report did not pass fixed source verification")
			}
			generated := []byte(reply.Source)
			if err := os.WriteFile(filepath.Join(dir, "generated.go.txt"), generated, 0644); err != nil {
				return err
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, parser.PackageClauseOnly)
			if err != nil {
				return err
			}
			pkg := filepath.Join(replay, id)
			if err := os.Mkdir(pkg, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(pkg, "generated.go"), generated, 0644); err != nil {
				return err
			}
			var tests bytes.Buffer
			fmt.Fprintf(&tests, "package %s\nimport \"testing\"\nfunc TestIndependentBehavior(t *testing.T){\n", parsed.Name.Name)
			for _, input := range pathstudy.Inputs(64) {
				want, err := pathstudy.Oracle(row.Instruction.Family, true, 64, input)
				if err != nil {
					return err
				}
				fmt.Fprintf(&tests, "if got:=ChoosePath(%d);got!=%d{t.Errorf(\"input=%d got=%%d\",got)}\n", input, want, input)
			}
			tests.WriteString("}\n")
			if err := os.WriteFile(filepath.Join(pkg, "generated_test.go"), tests.Bytes(), 0644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "independent_test.go.txt"), tests.Bytes(), 0644); err != nil {
				return err
			}
			cells = append(cells, cell{id, arm, digest(source), digest(generated), digest(stdout), row.Search.Selection, len(pathstudy.Inputs(64)), wall})
			unique[digest(generated)] = true
		}
	}
	stdout, stderr, wall, err := child(replay, goBinary, "test", "-count=1", "-json", "./...")
	if writeErr := os.WriteFile(filepath.Join(output, "go-replay-stdout.jsonl"), stdout, 0644); writeErr != nil {
		return writeErr
	}
	if writeErr := os.WriteFile(filepath.Join(output, "go-replay-stderr.txt"), stderr, 0644); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return errors.New("independent generated-Go replay failed; exact captures retained")
	}
	passedPackages, passedTests := 0, 0
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	scanner.Buffer(make([]byte, 8192), 128<<10)
	for scanner.Scan() {
		var event struct {
			Action string
			Test   string
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			return errors.New("invalid Go test event")
		}
		if event.Action == "pass" {
			if event.Test == "TestIndependentBehavior" {
				passedTests++
			} else if event.Test == "" {
				passedPackages++
			}
		}
	}
	if scanner.Err() != nil || passedPackages != 20 || passedTests != 20 {
		return errors.New("Go test events did not account for all twenty independent tests")
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/typed-path-native-replay-report/v1", "status": "PASS", "runner_source_revision": revision, "native_source_revision": nativeSource, "native_compiler_calls": 20, "additional_local_model_predictions": 0, "external_provider_calls": 0, "independent_cases_passed": 240, "independent_cases_total": 240, "unique_generated_functions": len(unique), "passed_packages": passedPackages, "passed_tests": passedTests, "replay_wall_ns": wall, "replay_stdout_sha256": digest(stdout), "cells": cells, "scope": "Exact native emitted Go bytes tested in isolated packages. Twenty executions repeat five unique functions across two languages and offline/FP32 arms. Finite cases do not prove full natural-language intent."})
}
func main() {
	compiler := flag.String("compiler", "", "fixed native compiler")
	goBinary := flag.String("go", "", "fixed Go 1.27.1")
	compilerSHA := flag.String("compiler-sha", nativeBinarySHA, "explicit platform compiler binary SHA-256")
	goSHA := flag.String("go-sha", goBinarySHA, "explicit platform Go binary SHA-256")
	probe := flag.String("probe", "runs/typed-path-reserved-probe-tdd-20261001", "saved probe")
	output := flag.String("output", "", "fresh evidence directory")
	revision := flag.String("source-revision", "", "full source commit")
	flag.Parse()
	if *compiler == "" || *goBinary == "" || *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "replay-path-probe: compiler, go, output and source-revision required")
		os.Exit(2)
	}
	if err := run(*compiler, *goBinary, *compilerSHA, *goSHA, *probe, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, "replay-path-probe:", err)
		os.Exit(1)
	}
	fmt.Println(`{"status":"PASS","native_calls":20,"additional_model_predictions":0,"independent_cases":240}`)
}
