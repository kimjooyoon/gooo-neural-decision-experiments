// compare-session-native measures repeated restart budgets against one continued
// finite search. It uses frozen synthetic intentions and real public tiny models.
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
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

var modelPins = map[string]string{
	"fp32":        "1ea3bada068f2487418f17270db4ba6785a98c3ad60e0ad7eb92359400a01bb5",
	"ptq_ternary": "7c4eb4068d76e83620a9b8e7ce7f8d948b5e2436d44c7b79cf241da609dab894",
	"qat_ternary": "368e37e7899cecba03874b69e933525a2f8c90ecc53d787d8e698de6aa21c087",
}

type pathDocument struct {
	Schema      string              `json:"schema"`
	Plan        pathplan.Plan       `json:"path_plan"`
	TestCases   []pathplan.TestCase `json:"test_cases"`
	MaxAttempts int                 `json:"max_attempts"`
}

type reply struct {
	Source string `json:"source"`
	Report struct {
		Decision  string `json:"decision"`
		ID        string `json:"activity_id"`
		Typecheck bool   `json:"typecheck_passed"`
		Replay    bool   `json:"deterministic_replay"`
		Writes    int    `json:"repository_writes"`
		Paths     struct {
			Matched    bool                       `json:"source_base_matched"`
			Search     pathplan.SearchResult      `json:"search"`
			Progress   []pathplan.SessionProgress `json:"session_progress"`
			Functional float64                    `json:"finite_functional_completeness_percent"`
			Timing     struct {
				Total float64 `json:"total_ms"`
			} `json:"timing"`
		} `json:"body_paths"`
	} `json:"report"`
}
type process struct {
	WallNS   int64 `json:"wall_ns"`
	UserNS   int64 `json:"user_cpu_ns"`
	SystemNS int64 `json:"system_cpu_ns"`
	RSS      int64 `json:"lifetime_peak_rss_bytes"`
	RSSKnown bool  `json:"rss_known"`
}
type observation struct {
	Budget    int     `json:"total_budget"`
	ReplySHA  string  `json:"stdout_sha256"`
	SourceSHA string  `json:"go_source_sha256"`
	Calls     int     `json:"local_model_predictions"`
	Attempts  int     `json:"attempted_candidates"`
	Passed    int     `json:"selected_finite_passed"`
	Cases     int     `json:"finite_cases"`
	StageMS   float64 `json:"native_stage_ms"`
	Process   process `json:"process"`
}
type cell struct {
	ID                      string                     `json:"id"`
	Arm                     string                     `json:"arm"`
	Language                string                     `json:"language"`
	Contract                string                     `json:"contract"`
	Repeat                  int                        `json:"repeat"`
	BatchFirst              bool                       `json:"batch_first"`
	Restart                 []observation              `json:"restart_observations"`
	Batched                 observation                `json:"batched_observation"`
	Progress                []pathplan.SessionProgress `json:"session_progress"`
	SameFinalBodyAndSearch  bool                       `json:"same_final_body_and_search"`
	MatchedProgressPrefixes int                        `json:"matched_progress_prefixes"`
}
type limited struct{ bytes.Buffer }

func (b *limited) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 2<<20 {
		return 0, errors.New("bounded process output exceeded")
	}
	return b.Buffer.Write(p)
}
func digest(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func read(path string) ([]byte, error) {
	i, err := os.Lstat(path)
	if err != nil || !i.Mode().IsRegular() || i.Size() <= 0 || i.Size() > 4<<20 {
		return nil, errors.New("bounded regular input required")
	}
	return os.ReadFile(path)
}
func normalized(s pathplan.SearchResult) pathplan.SearchResult {
	s.Selection.Receipts = append([]pathplan.Receipt(nil), s.Selection.Receipts...)
	for i := range s.Selection.Receipts {
		s.Selection.Receipts[i].PredictNS = 0
	}
	return s
}
func checkProgress(progress []pathplan.SessionProgress, search pathplan.SearchResult, calls int) error {
	if len(progress) < 2 || progress[0].Attempted != 0 || !progress[0].Initialized {
		return errors.New("missing initialization observation")
	}
	previous := ""
	attempted := 0
	seen := map[string]bool{}
	for i, p := range progress {
		if p.Sequence != i+1 || p.PreviousSHA != previous || p.Selection.ModelCalls != calls ||
			p.PredictionsThisAdvance != 0 || p.Selection.ExternalCalls != 0 || !p.Selection.ExternalCallsKnown ||
			len(p.NewAttempts) > 8 || p.ScheduledBytes != 8 {
			return errors.New("invalid continued observation accounting")
		}
		stored := p.SHA
		p.SHA = ""
		raw, err := json.Marshal(p)
		if err != nil || stored != digest(raw) {
			return errors.New("observation digest mismatch")
		}
		previous = stored
		for _, a := range p.NewAttempts {
			raw, err := json.Marshal(a.Choices)
			if err != nil || seen[string(raw)] {
				return errors.New("repeated continued candidate")
			}
			seen[string(raw)] = true
			attempted++
		}
		if p.Attempted != attempted || p.Unattempted != p.Declared-attempted {
			return errors.New("candidate denominator mismatch")
		}
	}
	last := progress[len(progress)-1]
	if attempted != len(search.Attempts) || last.SelectedPassed != search.SelectedTrainingPassed ||
		last.Evaluated != search.Evaluated || last.TypeRejected != search.TypeRejected ||
		!reflect.DeepEqual(last.Selection.Choices, search.Selection.Choices) {
		return errors.New("final observation mismatch")
	}
	return nil
}
func child(ctx context.Context, binary string, args []string) ([]byte, process, error) {
	bounded, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, binary, args...)
	configureProcess(cmd)
	cmd.WaitDelay = time.Second
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "LANG=C", "GOTOOLCHAIN=go1.27.1", "GOWORK=off"}
	var out, errout limited
	cmd.Stdout = &out
	cmd.Stderr = &errout
	start := time.Now()
	err := cmd.Run()
	p := process{WallNS: time.Since(start).Nanoseconds()}
	if cmd.ProcessState != nil {
		p.UserNS = cmd.ProcessState.UserTime().Nanoseconds()
		p.SystemNS = cmd.ProcessState.SystemTime().Nanoseconds()
		p.RSS, p.RSSKnown = peakRSS(cmd.ProcessState)
	}
	if err != nil || errout.Len() != 0 {
		return out.Bytes(), p, fmt.Errorf("bounded native execution failed: %v", err)
	}
	return out.Bytes(), p, nil
}
func invoke(ctx context.Context, binary, output, id, plan, model string, step, budget int) (reply, observation, error) {
	args := []string{"body-codegen", "--json", "--path-plan", plan, "--activity", "ConditionalAssign", "studies/conditional-paths-v1/cohort/conditional-assignment.gooo.fixture"}
	if model != "" {
		args = append(args, "--path-model", model)
	}
	if step > 0 {
		args = append(args, "--path-step-attempts", fmt.Sprint(step))
	}
	raw, p, err := child(ctx, binary, args)
	if writeErr := os.WriteFile(filepath.Join(output, "captures", id+".json"), raw, 0644); writeErr != nil {
		return reply{}, observation{}, writeErr
	}
	if err != nil {
		return reply{}, observation{}, err
	}
	var r reply
	if err = json.Unmarshal(raw, &r); err != nil {
		return r, observation{}, err
	}
	s := r.Report.Paths.Search
	if r.Report.Decision != "PASS" || !r.Report.Typecheck || !r.Report.Replay || r.Report.Writes != 0 ||
		!r.Report.Paths.Matched || s.TrainingTotal != 7 || s.Selection.ExternalCalls != 0 ||
		!s.Selection.ExternalCallsKnown || s.DeclaredCombinations != 64 || len(s.Attempts) > budget {
		return r, observation{}, errors.New("native source/type/replay/finite boundary failed")
	}
	o := observation{Budget: budget, ReplySHA: digest(raw), SourceSHA: digest([]byte(r.Source)), Calls: s.Selection.ModelCalls,
		Attempts: len(s.Attempts), Passed: s.SelectedTrainingPassed, Cases: s.TrainingTotal, StageMS: r.Report.Paths.Timing.Total, Process: p}
	return r, o, nil
}
func run(binary, revision, output string) error {
	if binary == "" || len(revision) != 40 || filepath.IsAbs(output) {
		return errors.New("explicit binary/revision and relative output required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("output must be fresh")
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("exact Go build required")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if settings["vcs.revision"] != revision || settings["vcs.modified"] != "false" {
		return errors.New("clean compiler identity required")
	}
	sdk := ""
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" && d.Replace == nil {
			sdk = d.Version
		}
	}
	if sdk != "v0.2.2-experimental" {
		return errors.New("public incremental SDK required")
	}
	modelEvidence := map[string]string{}
	modelPaths := map[string]string{}
	for arm, pin := range modelPins {
		path := filepath.Join("runs/typed-path-positioned-random-20261001/models", arm, "model.json")
		raw, err := read(path)
		if err != nil || digest(raw) != pin {
			return errors.New("frozen model metadata mismatch")
		}
		model, err := decision.LoadPath(path)
		if err != nil || model.Schema() != decision.PathMetadataSchema {
			return errors.New("frozen structural model invalid")
		}
		modelEvidence[arm] = model.WeightsSHA256()
		modelPaths[arm] = path
	}
	if err := os.MkdirAll(filepath.Join(output, "captures"), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(output, "inputs"), 0755); err != nil {
		return err
	}
	binaryRaw, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	runnerPath, err := os.Executable()
	if err != nil {
		return err
	}
	runnerRaw, err := os.ReadFile(runnerPath)
	if err != nil {
		return err
	}
	pre := map[string]any{"schema": "gooo/incremental-native-preexecution/v1", "compiler_revision": revision,
		"compiler_binary_sha256": digest(binaryRaw), "runner_binary_sha256": digest(runnerRaw), "sdk": sdk, "go_version": info.GoVersion,
		"model_weights_sha256": modelEvidence, "planned_cells": 32, "planned_native_calls": 288, "planned_fresh_model_predictions": 1296,
		"optimizer_steps": 0, "gpu_work": false, "external_provider_calls": 0, "repetitions": 2, "step_new_attempts": 8,
		"baseline_budgets": []int{8, 16, 24, 32, 40, 48, 56, 64}, "scope": "One frozen compound intent, two languages, four model arms, complete and deliberately inconsistent contracts. These views and repeats are not distinct ideas. Baseline makes eight fresh native requests; continuation makes one. This compares client policies, not equal-call kernel throughput. Models rank once before tests, without feedback reranking or new training."}
	if err := save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var cells []cell
	nativeCalls, predictions := 0, 0
	for _, language := range []string{"en", "ko"} {
		for _, contract := range []string{"complete", "inconsistent"} {
			raw, err := read("studies/conditional-paths-v1/cohort/" + language + "-budget-64.json")
			if err != nil {
				return err
			}
			var document pathDocument
			if err := strictjson.Decode(raw, &document); err != nil {
				return err
			}
			if document.Schema != "gooo/body-codegen-typed-path-plan/v1" || len(document.TestCases) != 7 || document.MaxAttempts != 64 {
				return errors.New("frozen finite document envelope mismatch")
			}
			if _, err := pathplan.Prepare(document.Plan); err != nil {
				return err
			}
			if contract == "inconsistent" {
				document.TestCases[6].Expected = 999
			}
			plans := map[int]string{}
			for budget := 8; budget <= 64; budget += 8 {
				document.MaxAttempts = budget
				path := filepath.Join(output, "inputs", fmt.Sprintf("%s-%s-%d.json", language, contract, budget))
				if err := save(path, document); err != nil {
					return err
				}
				plans[budget] = path
			}
			for _, arm := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
				for repeat := 0; repeat < 2; repeat++ {
					c := cell{ID: fmt.Sprintf("%s-%s-%s-%d", language, contract, arm, repeat), Arm: arm, Language: language, Contract: contract, Repeat: repeat, BatchFirst: len(cells)%2 == 0}
					var batch reply
					restarts := map[int]reply{}
					modes := []string{"restart", "batch"}
					if c.BatchFirst {
						modes = []string{"batch", "restart"}
					}
					for _, mode := range modes {
						budgets := []int{64}
						step := 8
						if mode == "restart" {
							budgets = []int{8, 16, 24, 32, 40, 48, 56, 64}
							step = 0
						}
						for _, budget := range budgets {
							r, o, err := invoke(ctx, binary, output, fmt.Sprintf("%s-%s-%d", c.ID, mode, budget), plans[budget], modelPaths[arm], step, budget)
							nativeCalls++
							predictions += o.Calls
							if err != nil {
								_ = save(filepath.Join(output, "failure.json"), map[string]any{"status": "FAILED", "cell": c.ID, "mode": mode, "budget": budget, "native_calls_started": nativeCalls, "successful_prediction_receipts": predictions, "error": err.Error()})
								return err
							}
							if mode == "batch" {
								batch = r
								c.Batched = o
								c.Progress = r.Report.Paths.Progress
							} else {
								restarts[budget] = r
								c.Restart = append(c.Restart, o)
							}
						}
					}
					if err := checkProgress(c.Progress, batch.Report.Paths.Search, c.Batched.Calls); err != nil {
						return err
					}
					c.SameFinalBodyAndSearch = batch.Source == restarts[64].Source && batch.Report.ID == restarts[64].Report.ID && reflect.DeepEqual(normalized(batch.Report.Paths.Search), normalized(restarts[64].Report.Paths.Search))
					if !c.SameFinalBodyAndSearch {
						return errors.New("final native restart/session mismatch")
					}
					var prefix []pathplan.SearchAttempt
					for i, p := range c.Progress {
						if i == 0 {
							continue
						}
						prefix = append(prefix, p.NewAttempts...)
						r := restarts[min(i*8, 64)].Report.Paths.Search
						if !reflect.DeepEqual(prefix, r.Attempts) || p.SelectedPassed != r.SelectedTrainingPassed || !reflect.DeepEqual(p.Selection.Choices, r.Selection.Choices) {
							return errors.New("continued prefix differs from same-budget restart")
						}
						c.MatchedProgressPrefixes++
					}
					cells = append(cells, c)
				}
			}
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/incremental-native-comparison/v1", "status": "PASS",
		"preexecution": pre, "native_calls": nativeCalls, "fresh_local_model_predictions": predictions, "cells": cells,
		"scope": "Declared seven cases, no new heldout set or training. Raw 288 native captures are separate. CPU is child CPU time, not host utilization delta; RSS is child lifetime high-water, not additive restart memory or model memory. Progress bitset excludes model, arena, frontier, runtime and retained receipts."})
}
func main() {
	binary := flag.String("binary", "", "clean native compiler")
	revision := flag.String("revision", "", "exact native source SHA")
	output := flag.String("output", "runs/incremental-native-20261001", "fresh relative output")
	flag.Parse()
	if err := run(*binary, *revision, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
