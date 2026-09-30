// compiler-dogfood runs the real optional model path and independently executes
// emitted Go. PROV-O traces bind the model, source, generation and verification.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
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
	"strings"
	"time"
)

type testCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}
type plan struct {
	Intent string     `json:"intent"`
	Cases  []testCase `json:"test_cases"`
}
type payload struct {
	Source string `json:"source"`
	Report struct {
		Compiler string `json:"compiler_source_sha"`
		Fill     struct {
			Decision struct {
				Provider    string `json:"provider"`
				Operation   string `json:"tiny_go_predicted_operation"`
				Applied     *bool  `json:"tiny_go_prediction_applied"`
				MetadataSHA string `json:"tiny_go_metadata_sha256"`
				WeightsSHA  string `json:"tiny_go_weights_sha256"`
			} `json:"decision"`
			Passed        int  `json:"test_cases_passed"`
			Total         int  `json:"test_cases_total"`
			Predictions   int  `json:"local_model_predictions"`
			External      int  `json:"external_provider_calls"`
			ExternalKnown bool `json:"external_provider_calls_known"`
			Timing        struct {
				Load     float64 `json:"tiny_model_load_ms"`
				Decision float64 `json:"tiny_decision_ms"`
			} `json:"timing"`
		} `json:"body_fill"`
	} `json:"report"`
}

type cell struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	RawOperation      string  `json:"raw_operation"`
	RawCorrect        bool    `json:"raw_operation_correct"`
	Applied           bool    `json:"model_applied"`
	ModelPredictions  int     `json:"model_predictions"`
	Passed            int     `json:"tdd_cases_passed"`
	Total             int     `json:"tdd_cases_total"`
	IndependentPassed int     `json:"independent_cases_passed"`
	WallMS            float64 `json:"native_wall_ms"`
	UserMS            float64 `json:"native_user_ms"`
	SystemMS          float64 `json:"native_system_ms"`
	PeakRSSBytes      int64   `json:"native_lifetime_peak_rss_bytes"`
	DecisionMS        float64 `json:"tiny_decision_ms"`
	LoadMS            float64 `json:"tiny_load_ms"`
	SourceSHA         string  `json:"source_sha256"`
	PlanSHA           string  `json:"plan_sha256"`
	MetadataSHA       string  `json:"model_metadata_sha256,omitempty"`
	WeightsSHA        string  `json:"weights_sha256,omitempty"`
	RawSHA            string  `json:"raw_receipt_sha256"`
	GeneratedSHA      string  `json:"generated_source_sha256,omitempty"`
	VerificationSHA   string  `json:"verification_sha256,omitempty"`
	Error             string  `json:"error,omitempty"`
}

type boundedBuffer struct {
	bytes.Buffer
	Truncated bool
}

func (b *boundedBuffer) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 128*1024 {
		b.Truncated = true
		return 0, errors.New("child output exceeds 128 KiB")
	}
	return b.Buffer.Write(raw)
}
func hash(raw []byte) string               { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func readHash(path string) (string, error) { raw, err := os.ReadFile(path); return hash(raw), err }
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
func child(ctx context.Context, directory, program string, args ...string) (*exec.Cmd, []byte, []byte, bool, float64, error) {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir, cmd.WaitDelay = directory, 2*time.Second
	cmd.Env = append(os.Environ(), "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY=", "GOTOOLCHAIN=local", "GOWORK=off")
	configureProcess(cmd)
	var stdout, stderr boundedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	started := time.Now()
	err := cmd.Run()
	return cmd, stdout.Bytes(), stderr.Bytes(), stdout.Truncated || stderr.Truncated, float64(time.Since(started).Nanoseconds()) / 1e6, err
}

func executeGenerated(ctx context.Context, source, activity string, cases []testCase) ([]byte, error) {
	directory, err := os.MkdirTemp("", "gooo-dogfood-replay-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	var probe strings.Builder
	fmt.Fprintf(&probe, "package bodycodegen\nimport \"testing\"\nfunc TestBehavior(t *testing.T) {\n")
	for _, item := range cases {
		fmt.Fprintf(&probe, "if got := %s(%d); got != %d { t.Errorf(\"input=%d got=%%d\", got) }\n", activity, item.Input, item.Expected, item.Input)
	}
	probe.WriteString("}\n")
	for name, content := range map[string]string{"go.mod": "module gooo.dogfood.replay\n\ngo 1.27.1\n", "generated.go": source, "generated_test.go": probe.String()} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o644); err != nil {
			return nil, err
		}
	}
	_, stdout, stderr, truncated, _, err := child(ctx, directory, filepath.Join(runtime.GOROOT(), "bin", "go"), "test", "-count=1", "./...")
	raw := append(append([]byte{}, stdout...), stderr...)
	if truncated {
		return raw, errors.New("replay output truncated")
	}
	return raw, err
}

func trace(item cell, compilerSHA string) []byte {
	// Every IRI component below is a generated SHA, fixed identifier, or enum.
	entity := func(kind, digest string) string { return "<urn:gooo:" + kind + ":" + digest + ">" }
	activity := entity("generation", item.RawSHA)
	verification := entity("verification", item.RawSHA)
	generated := entity("generated", item.RawSHA+":"+item.GeneratedSHA)
	evidence := entity("evidence", item.RawSHA+":"+item.VerificationSHA)
	agent := entity("compiler", compilerSHA)
	var output strings.Builder
	output.WriteString("@prefix prov: <http://www.w3.org/ns/prov#> .\n")
	fmt.Fprintf(&output, "%s a prov:SoftwareAgent .\n%s a prov:Activity ; prov:wasAssociatedWith %s .\n", agent, activity, agent)
	for _, input := range []string{entity("source", item.SourceSHA), entity("plan", item.PlanSHA), entity("metadata", item.MetadataSHA), entity("weights", item.WeightsSHA)} {
		if strings.HasSuffix(input, ":>") {
			continue
		}
		fmt.Fprintf(&output, "%s a prov:Entity .\n%s prov:used %s .\n%s prov:wasDerivedFrom %s .\n", input, activity, input, generated, input)
	}
	fmt.Fprintf(&output, "%s a prov:Entity ; prov:wasGeneratedBy %s .\n", generated, activity)
	fmt.Fprintf(&output, "%s a prov:Activity ; prov:used %s ; prov:wasInformedBy %s ; prov:wasAssociatedWith %s .\n", verification, generated, activity, agent)
	fmt.Fprintf(&output, "%s a prov:Entity ; prov:wasGeneratedBy %s ; prov:wasDerivedFrom %s .\n", evidence, verification, generated)
	return []byte(output.String())
}

func run() error {
	binary := flag.String("compiler", "", "native Gooo executable")
	compilerDir := flag.String("compiler-dir", "", "clean native module checkout")
	modelRoot := flag.String("models", "", "model bundle root; empty means deterministic baseline")
	fixtures := flag.String("fixtures", "publication/native-tiny-body-fill-e7dc-v2/fixtures", "two disclosed regression fixtures")
	output := flag.String("output", "", "fresh append-only output directory")
	flag.Parse()
	if *binary == "" || *compilerDir == "" || *output == "" || flag.NArg() != 0 {
		return errors.New("compiler, compiler-dir and output are required")
	}
	info, err := buildinfo.ReadFile(*binary)
	if err != nil {
		return err
	}
	if info.GoVersion != "go1.27.1" {
		return errors.New("native binary must use Go 1.27.1")
	}
	revision := ""
	for _, item := range info.Settings {
		if item.Key == "vcs.revision" {
			revision = item.Value
		}
		if item.Key == "vcs.modified" && item.Value != "false" {
			return errors.New("native source is modified")
		}
	}
	if len(revision) != 40 {
		return errors.New("native source revision missing")
	}
	nativeSHA, err := readHash(*binary)
	if err != nil {
		return err
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return errors.New("output must be fresh")
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		return err
	}
	variants := []string{"fp32", "ptq_ternary", "qat_ternary"}
	if *modelRoot == "" {
		variants = []string{"offline"}
	}
	var cells []cell
	for _, fixture := range []struct{ Name, Activity, Gold string }{{"arithmetic", "ArithmeticHole", "add"}, {"boolean", "BooleanHole", "less_equal"}} {
		for _, variant := range variants {
			item := cell{ID: fixture.Name + "-" + variant, Status: "FAIL"}
			sourcePath, _ := filepath.Abs(filepath.Join(*fixtures, fixture.Name+".gooo"))
			planPath, _ := filepath.Abs(filepath.Join(*fixtures, fixture.Name+".plan.json"))
			sourceRaw, readErr := os.ReadFile(sourcePath)
			if readErr != nil {
				return readErr
			}
			planRaw, readErr := os.ReadFile(planPath)
			if readErr != nil {
				return readErr
			}
			item.SourceSHA, item.PlanSHA = hash(sourceRaw), hash(planRaw)
			var expected plan
			if err := json.Unmarshal(planRaw, &expected); err != nil {
				return err
			}
			args := []string{"body-codegen", "--json", "--fill-plan", planPath, "--activity", fixture.Activity}
			if variant != "offline" {
				metadata, _ := filepath.Abs(filepath.Join(*modelRoot, variant, "model.json"))
				item.MetadataSHA, err = readHash(metadata)
				if err != nil {
					return err
				}
				item.WeightsSHA, err = readHash(filepath.Join(filepath.Dir(metadata), "weights.bin"))
				if err != nil {
					return err
				}
				args = append(args, "--tiny-model", metadata)
			}
			args = append(args, sourcePath)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cmd, stdout, stderr, truncated, wall, runErr := child(ctx, *compilerDir, *binary, args...)
			cancel()
			item.WallMS, item.RawSHA = wall, hash(stdout)
			item.PeakRSSBytes = peakRSS(cmd)
			if cmd.ProcessState != nil {
				item.UserMS = float64(cmd.ProcessState.UserTime().Nanoseconds()) / 1e6
				item.SystemMS = float64(cmd.ProcessState.SystemTime().Nanoseconds()) / 1e6
			}
			if err := os.WriteFile(filepath.Join(*output, item.ID+".stdout.json"), stdout, 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(*output, item.ID+".stderr.txt"), stderr, 0o644); err != nil {
				return err
			}
			var decoded payload
			if runErr != nil || truncated || json.Unmarshal(stdout, &decoded) != nil {
				item.Error = "native failed, truncated or invalid JSON"
				cells = append(cells, item)
				continue
			}
			fill := decoded.Report.Fill
			item.RawOperation = fill.Decision.Operation
			item.RawCorrect = item.RawOperation == fixture.Gold
			item.ModelPredictions = fill.Predictions
			item.Applied = fill.Decision.Applied != nil && *fill.Decision.Applied
			item.Passed, item.Total = fill.Passed, fill.Total
			item.LoadMS, item.DecisionMS = fill.Timing.Load, fill.Timing.Decision
			if decoded.Report.Compiler != revision || !fill.ExternalKnown || fill.External != 0 || item.Total != len(expected.Cases) || decoded.Source == "" {
				item.Error = "native provenance/count binding mismatch"
				cells = append(cells, item)
				continue
			}
			if variant != "offline" && (fill.Decision.MetadataSHA != item.MetadataSHA || fill.Decision.WeightsSHA != item.WeightsSHA || fill.Predictions != 1) {
				item.Error = "model binding mismatch"
				cells = append(cells, item)
				continue
			}
			if variant == "offline" && (fill.Predictions != 0 || fill.Decision.Provider != "deterministic") {
				item.Error = "offline provider mismatch"
				cells = append(cells, item)
				continue
			}
			item.GeneratedSHA = hash([]byte(decoded.Source))
			if err := os.WriteFile(filepath.Join(*output, item.ID+".generated.go"), []byte(decoded.Source), 0o644); err != nil {
				return err
			}
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			verification, verifyErr := executeGenerated(ctx, decoded.Source, fixture.Activity, expected.Cases)
			cancel()
			item.VerificationSHA = hash(verification)
			if err := os.WriteFile(filepath.Join(*output, item.ID+".verification.txt"), verification, 0o644); err != nil {
				return err
			}
			if verifyErr == nil {
				item.IndependentPassed = len(expected.Cases)
			} else {
				item.Error = "independent Go execution failed"
			}
			if verifyErr == nil && item.Passed == item.Total {
				item.Status = "PASS"
			}
			if err := os.WriteFile(filepath.Join(*output, item.ID+".prov.ttl"), trace(item, revision), 0o644); err != nil {
				return err
			}
			cells = append(cells, item)
		}
	}
	finalSHA, err := readHash(*binary)
	if err != nil {
		return err
	}
	if finalSHA != nativeSHA {
		return errors.New("native binary changed during run")
	}
	status := "PASS"
	for _, item := range cells {
		if item.Status != "PASS" {
			status = "FAIL"
		}
	}
	report := map[string]any{"schema": "gooo/compiler-dogfood/v1", "status": status, "compiler_revision": revision, "compiler_sha256": nativeSHA, "go_version": runtime.Version(), "cells": cells,
		"scope": "Two disclosed repair-training intents; not holdout or full-domain proof", "warmups": 0, "retries": 0, "external_provider_calls": 0,
		"resource_scope":  "Native child process wall/CPU and OS lifetime peak RSS; excludes Go replay processes, not host utilization delta",
		"provenance_spec": "https://www.w3.org/TR/prov-o/"}
	if err := save(filepath.Join(*output, "report.json"), report); err != nil {
		return err
	}
	fmt.Printf("%s: %d native cells, PROV-O traces and independent Go executions\n", status, len(cells))
	if status != "PASS" {
		return errors.New("dogfood evidence contains failures")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var _ io.Writer = (*boundedBuffer)(nil)
