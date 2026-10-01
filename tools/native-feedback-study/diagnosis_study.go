package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const diagnosisPrereg = "docs/path-diagnosis-preregistration.md"
const diagnosisModel = "runs/feedback-path-soft-target-mps-20261001/models/fp32/model.json"
const diagnosisModelPin = "47bd3ed2037c8ba0af31ec4cad3a47fe1182af171845406aba01c5107f4b24b7"
const diagnosisWeightsPin = "73eedef1d60f49129190683a41cbf82229c1e3603fd44cdd17792d015f499954"

func diagnosisProbes() []int64 {
	return []int64{math.MinInt64, math.MinInt64 + 1, -107, -33, -15, -14, -13, -9, -4, -3, -2, -1, 0,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 13, 14, 15, 17, 31, 33, 107, math.MaxInt64 - 1, math.MaxInt64}
}

type diagnosisCapture struct {
	ID          string                `json:"id"`
	Arm         string                `json:"arm"`
	Search      pathplan.SearchResult `json:"search"`
	Diagnosis   pathplan.Diagnosis    `json:"diagnosis"`
	SearchNS    int64                 `json:"search_ns"`
	DiagnosisNS int64                 `json:"diagnosis_ns"`
}

type diagnosisExecution struct {
	Schema   string  `json:"schema"`
	Template string  `json:"template"`
	Mask     uint16  `json:"mask"`
	Source   string  `json:"generated_go"`
	SHA      string  `json:"generated_go_sha256"`
	Inputs   []int64 `json:"inputs"`
	Values   []int64 `json:"actual_go_values"`
}

func diagnosisPreflight(output, revision string) error {
	if !strings.HasPrefix(output, "runs/") || filepath.Clean(output) != output || strings.Contains(output, "..") {
		return errors.New("fresh relative runs directory required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh diagnosis output required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision || len(revision) != 40 {
		return errors.New("exact committed diagnosis source required")
	}
	status, err := exec.Command("git", "status", "--porcelain", "--untracked-files=normal").Output()
	if err != nil || len(status) != 0 {
		return errors.New("clean diagnosis runner required")
	}
	return nil
}

func runDiagnosisStudy(goBinary, output, revision string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(goBinary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("exact Go 1.27.1 executable required")
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	rows, err := compoundRows()
	if err != nil || len(rows) != 72 {
		return errors.New("frozen 72-view compound cohort required")
	}
	model, err := decision.LoadPath(diagnosisModel)
	if err != nil || model.MetadataSHA256() != diagnosisModelPin || model.WeightsSHA256() != diagnosisWeightsPin {
		return errors.New("frozen own FP32 model required")
	}
	prereg, err := read(diagnosisPrereg)
	if err != nil {
		return err
	}
	cohort, err := read(compoundRoot + "/cohort.jsonl")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(output, "executions"), 0755); err != nil {
		return err
	}
	pre := map[string]any{"schema": "gooo/path-diagnosis-preexecution/v1", "runner_revision": revision,
		"cohort_sha256": hash(cohort), "preregistration_sha256": hash(prereg), "go_binary_sha256": goSHA,
		"model_metadata_sha256": model.MetadataSHA256(), "model_weights_sha256": model.WeightsSHA256(),
		"search_budget": 2, "diagnosis_budget": 4, "probe_inputs": diagnosisProbes(), "planned_searches": 144,
		"planned_diagnoses": 144, "planned_candidate_observations": 576, "planned_initial_predictions": 144,
		"scope": "SDK study using existing 72 bilingual/contract views over 12 intention groups; no native call, tuning, GPU or upstream Laya. Probes go only to post-selection deterministic diagnosis. Witness outputs are observations, not authoritative expected answers. Search/diagnosis timings exclude model loading and plan preparation; no causal speedup or host utilization claim."}
	if err = save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(output, "observations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	executions := map[string]diagnosisExecution{}
	for index, row := range rows {
		arms := []string{"offline", "fp32"}
		if index%2 != 0 {
			arms[0], arms[1] = arms[1], arms[0]
		}
		for _, arm := range arms {
			prepared, err := pathplan.Prepare(row.Document.Plan)
			if err != nil {
				return err
			}
			var selector *decision.Model
			if arm == "fp32" {
				selector = model
			}
			started := time.Now()
			search, _, err := prepared.Search(ctx, selector, row.Document.Cases, 2, "")
			searchNS := time.Since(started).Nanoseconds()
			if err != nil {
				return err
			}
			started = time.Now()
			diagnosis, err := prepared.Diagnose(ctx, search.Selection.Choices, row.Document.Cases, diagnosisProbes(), 4)
			diagnosisNS := time.Since(started).Nanoseconds()
			if err != nil {
				return err
			}
			capture := diagnosisCapture{row.ID, arm, search, diagnosis, searchNS, diagnosisNS}
			if err = encoder.Encode(capture); err != nil {
				return err
			}
			if err = validateDiagnosisCapture(row, capture); err != nil {
				return err
			}
			for mask := range uint16(4) {
				program, err := prepared.Compile(compoundstudy.Choices(row.Document.Plan, mask))
				if err != nil {
					return err
				}
				source := program.GoSource()
				sha := hash([]byte(source))
				executions[sha] = diagnosisExecution{Schema: "gooo/path-diagnosis-go-execution/v1", Template: row.Template, Mask: mask,
					Source: source, SHA: sha, Inputs: diagnosisProbes()}
			}
		}
	}
	if err = file.Close(); err != nil {
		return err
	}
	var keys []string
	for sha := range executions {
		keys = append(keys, sha)
	}
	sort.Strings(keys)
	for _, sha := range keys {
		value := executions[sha]
		raw, err := executeFunction(ctx, goBinary, value.Source, "ComposePaths", value.Inputs)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &value.Values); err != nil {
			return err
		}
		if err = save(filepath.Join(output, "executions", sha+".json"), value); err != nil {
			return err
		}
	}
	report, err := auditDiagnosisStudy(output, revision)
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "report.json"), report); err != nil {
		return err
	}
	return save(filepath.Join(output, "audit.json"), report)
}
