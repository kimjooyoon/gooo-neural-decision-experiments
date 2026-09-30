// compare-compiler-bodies derives matched metrics from the two captured runs.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodydecision"
)

type cell struct {
	ID           string                 `json:"id"`
	Family       string                 `json:"family"`
	Arm          string                 `json:"arm"`
	Initial      bodydecision.Selection `json:"initial_selection"`
	Train        bodydecision.Score     `json:"compiled_go_training"`
	Test         bodydecision.Score     `json:"compiled_go_heldout"`
	Holes        int                    `json:"total_holes"`
	CorrectHoles int                    `json:"oracle_correct_holes"`
	Native       bool                   `json:"native_gooo_generation_pass"`
	Cases        int                    `json:"compiled_go_cases_observed"`
}
type capture struct {
	Status        string `json:"status"`
	Cohort        string `json:"cohort_sha256"`
	Source        string `json:"source_revision"`
	Runner        string `json:"runner_binary_sha256"`
	Compiler      string `json:"gooo_binary_sha256"`
	Go            string `json:"go_tool_sha256"`
	Planned       int    `json:"planned_cells"`
	Observed      int    `json:"observed_selected_cells"`
	ObservedCases int    `json:"compiled_go_cases_observed"`
	UnknownCases  int    `json:"compiled_go_cases_unknown"`
	Predictions   int    `json:"tiny_predictions_observed"`
	LayaCalls     int    `json:"laya_posts_attempted"`
	Stable        bool   `json:"executable_pins_stable_after_execution"`
	Cells         []cell `json:"cells"`
}
type metric struct {
	Programs     int                `json:"programs"`
	FullyCorrect int                `json:"all_heldout_cases_correct_programs"`
	Test         bodydecision.Score `json:"heldout_cases"`
	Train        bodydecision.Score `json:"training_cases"`
	Holes        int                `json:"selected_holes_total"`
	CorrectHoles int                `json:"selected_holes_correct"`
	RawKnown     int                `json:"raw_model_holes_observed"`
	RawCorrect   int                `json:"raw_model_holes_correct"`
	Applied      int                `json:"model_applied_holes"`
	Fallback     int                `json:"fallback_holes"`
}
type row struct {
	ID     string            `json:"id"`
	Oracle map[string]string `json:"oracle_operations"`
}
type artifact struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}

func digest(raw []byte) string { value := sha256.Sum256(raw); return hex.EncodeToString(value[:]) }
func readJSON(name string, value any) ([]byte, error) {
	raw, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	if len(raw) > 8<<20 {
		return nil, errors.New("report too large")
	}
	return raw, json.Unmarshal(raw, value)
}
func writeJSON(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}

func aggregate(value capture, gold map[string]map[string]string) (map[string]metric, error) {
	if value.Status != "completed" || value.Planned != 1024 || value.Observed != 1024 || len(value.Cells) != 1024 || value.ObservedCases != 36160 || value.UnknownCases != 0 || value.Predictions != 666 || value.LayaCalls != 0 || !value.Stable {
		return nil, errors.New("capture is incomplete or differs from fixed denominator")
	}
	metrics := map[string]metric{}
	seen := map[string]bool{}
	observedCases := 0
	arms := map[string]bool{"deterministic": true, "deterministic_search": true, "fp32": true, "fp32_search": true, "ptq_ternary": true, "ptq_ternary_search": true, "qat_ternary": true, "qat_ternary_search": true}
	for _, c := range value.Cells {
		key := c.ID + "/" + c.Arm
		if seen[key] || !arms[c.Arm] || gold[c.ID] == nil || !c.Native || c.Cases != c.Train.Total+c.Test.Total || c.Test.Total != 10 || c.CorrectHoles < 0 || c.CorrectHoles > c.Holes || c.Holes != len(gold[c.ID]) || c.Test.Correct < 0 || c.Test.Correct > c.Test.Total || c.Train.Correct < 0 || c.Train.Correct > c.Train.Total {
			return nil, errors.New("invalid cell identity or denominator")
		}
		seen[key] = true
		observedCases += c.Cases
		for _, key := range []string{c.Arm, c.Family + "/" + c.Arm} {
			m := metrics[key]
			m.Programs++
			m.Test.Correct += c.Test.Correct
			m.Test.Total += c.Test.Total
			m.Train.Correct += c.Train.Correct
			m.Train.Total += c.Train.Total
			m.Holes += c.Holes
			m.CorrectHoles += c.CorrectHoles
			if c.Test.Correct == c.Test.Total {
				m.FullyCorrect++
			}
			for _, h := range c.Initial.Holes {
				if h.Proposed != "" {
					m.RawKnown++
					if h.Proposed == gold[c.ID][h.ID] {
						m.RawCorrect++
					}
				}
				if h.Mode == "model_global_argmax" {
					m.Applied++
				} else if strings.HasPrefix(h.Mode, "fallback_") || h.Mode == "declared_fallback" {
					m.Fallback++
				}
			}
			metrics[key] = m
		}
	}
	if observedCases != value.ObservedCases {
		return nil, errors.New("case totals differ from capture")
	}
	for arm := range arms {
		m := metrics[arm]
		if m.Programs != 128 || m.Holes != 222 || m.Test.Total != 1280 || m.Train.Total != 3240 {
			return nil, errors.New("arm denominator differs")
		}
	}
	return metrics, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	root := flag.String("study", "runs/compiler-prov-v3-bodyplan-20261001", "fixed captured study")
	cohort := flag.String("cohort", "studies/body-plan-v1/cohort.jsonl", "frozen intents and oracle operations")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	output := filepath.Join(*root, "comparison.json")
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("comparison must be fresh")
	}
	corpus, err := os.ReadFile(*cohort)
	if err != nil {
		return err
	}
	gold := map[string]map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(corpus)), "\n") {
		var r row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return err
		}
		if r.ID == "" || gold[r.ID] != nil {
			return errors.New("invalid corpus identity")
		}
		gold[r.ID] = r.Oracle
	}
	if len(gold) != 128 {
		return errors.New("wrong intent denominator")
	}
	var parent, improved capture
	praw, err := readJSON(filepath.Join(*root, "parent/report.json"), &parent)
	if err != nil {
		return err
	}
	inraw, err := readJSON(filepath.Join(*root, "improved/report.json"), &improved)
	if err != nil {
		return err
	}
	if parent.Cohort != digest(corpus) || improved.Cohort != parent.Cohort || parent.Source != improved.Source || parent.Runner != improved.Runner || parent.Compiler != improved.Compiler || parent.Go != improved.Go {
		return errors.New("comparison controls differ")
	}
	before, err := aggregate(parent, gold)
	if err != nil {
		return err
	}
	after, err := aggregate(improved, gold)
	if err != nil {
		return err
	}
	for _, arm := range []string{"deterministic", "deterministic_search"} {
		if before[arm] != after[arm] {
			return errors.New("deterministic controls differ")
		}
	}
	var keys []string
	for key := range before {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var comparisons []map[string]any
	for _, key := range keys {
		b, a := before[key], after[key]
		if b.Programs != a.Programs || b.Test.Total != a.Test.Total || b.Train.Total != a.Train.Total || b.Holes != a.Holes {
			return errors.New("matched denominators differ")
		}
		comparisons = append(comparisons, map[string]any{"scope": key, "parent": b, "improved": a, "heldout_accuracy_delta_percentage_points": 100 * float64(a.Test.Correct-b.Test.Correct) / float64(b.Test.Total), "all_heldout_correct_program_delta": a.FullyCorrect - b.FullyCorrect})
	}
	source, err := os.ReadFile("tools/compare-compiler-bodies/main.go")
	if err != nil {
		return err
	}
	if err := writeJSON(output, map[string]any{"schema": "gooo/compiler-bodyplan-model-comparison/v1", "status": "PASS", "parent_report_sha256": digest(praw), "improved_report_sha256": digest(inraw), "analyzer_source_sha256": digest(source), "cohort_sha256": parent.Cohort, "comparisons": comparisons, "native_calls_across_cohorts": 2048, "local_tiny_predictions_across_cohorts": 1332, "external_model_calls": 0, "generated_go_case_executions_across_cohorts": 72320, "unique_intents_across_cohorts": 128, "limitations": []string{"Heldout means disjoint function inputs; synthetic program intents are not an independent natural-language workload.", "Initial decisions are made in the Go body assembler; native compiler --tiny-model is not used in this study.", "Search consumes only training cases and makes no additional model predictions.", "Runs execute concurrently; wall times are not a controlled latency or throughput comparison.", "FP32 improves; PTQ and QAT regress in direct functional selection. Synthetic instruction-view accuracy does not establish body-level performance."}}); err != nil {
		return err
	}
	var files []artifact
	err = filepath.WalkDir(*root, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(*root, name)
		if err != nil {
			return err
		}
		if rel == "artifact-manifest.json" {
			return errors.New("artifact manifest must be fresh")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("nonregular evidence file")
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files = append(files, artifact{filepath.ToSlash(rel), digest(raw), len(raw)})
		return nil
	})
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(*root, "artifact-manifest.json"), map[string]any{"schema": "gooo/compiler-bodyplan-evidence-manifest/v1", "files": files, "self_excluded": true}); err != nil {
		return err
	}
	fmt.Printf("PASS: matched 128 intents, 2048 native generations, 72320 finite executions; manifest binds %d files\n", len(files))
	return nil
}
