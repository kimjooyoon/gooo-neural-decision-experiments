package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func nativeUnfixedRows() ([]compoundstudy.Case, error) {
	all, err := compoundRows()
	if err != nil {
		return nil, err
	}
	var rows []compoundstudy.Case
	for _, row := range all {
		if row.IntendedMask == 0 && row.Contract == "contradictory" {
			rows = append(rows, row)
		}
	}
	if len(rows) != 6 {
		return nil, errors.New("six reused contradictory pilot views required")
	}
	return rows, nil
}

// This checks retained observations without starting a compiler, Go process or
// prediction. Resource sidecars are observed measurements, not replayed clocks.
func auditNativeUnfixed(dir, revision, native string) (map[string]any, error) {
	sha40 := regexp.MustCompile(`^[a-f0-9]{40}$`)
	sha64 := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if !sha40.MatchString(revision) || !sha40.MatchString(native) {
		return nil, errors.New("explicit source pins required")
	}
	pre, err := read(filepath.Join(dir, "preexecution.json"))
	var binding struct {
		Schema    string `json:"schema"`
		Runner    string `json:"runner_revision"`
		Native    string `json:"native_revision"`
		Binary    string `json:"native_binary_sha256"`
		GoBinary  string `json:"go_binary_sha256"`
		Go        string `json:"go"`
		SDK       string `json:"sdk"`
		Calls     int    `json:"planned_native_calls"`
		Views     int    `json:"views"`
		Authority bool   `json:"caller_ci_hint_is_authority"`
		Hint      struct {
			Source string `json:"source_sha"`
			Status string `json:"status"`
		} `json:"caller_ci_hint"`
	}
	if err != nil || json.Unmarshal(pre, &binding) != nil || binding.Schema != "gooo/native-unfixed-preexecution/v1" ||
		binding.Runner != revision || binding.Native != native || !sha64.MatchString(binding.Binary) ||
		!sha64.MatchString(binding.GoBinary) || binding.Go != "1.27.1" || binding.SDK != "v0.2.7-experimental" ||
		binding.Calls != 48 || binding.Views != 6 || binding.Authority || binding.Hint.Source != native {
		return nil, errors.New("native unfixed preexecution differs")
	}
	ci := pathplan.CIHint{SourceSHA: binding.Hint.Source, Status: binding.Hint.Status}
	if err := ci.Validate(); err != nil {
		return nil, err
	}
	rows, err := nativeUnfixedRows()
	if err != nil {
		return nil, err
	}
	arms, err := compoundArms()
	if err != nil {
		return nil, err
	}
	inputSet := map[int64]bool{}
	for _, row := range rows {
		for _, test := range row.Document.Cases {
			inputSet[test.Input] = true
		}
		for _, test := range row.Separate {
			inputSet[test.Input] = true
		}
	}
	var inputs []int64
	for input := range inputSet {
		inputs = append(inputs, input)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i] < inputs[j] })
	files := map[string]string{}
	executions := map[string]bool{}
	var observations []map[string]any
	pairs, changedSequences, changedBodies, legacyCalls, optCalls, skips := 0, 0, 0, 0, 0, 0
	for _, row := range rows {
		for _, arm := range arms {
			if !arm.Feedback {
				continue
			}
			var previous nativeResult
			for _, unfixed := range []bool{false, true} {
				id := fmt.Sprintf("%s-%s-%t", row.ID, arm.Name, unfixed)
				path := "captures/" + id + ".json"
				raw, err := read(filepath.Join(dir, path))
				var value nativeResult
				if err != nil || json.Unmarshal(raw, &value) != nil {
					return nil, errors.New("native pilot capture missing")
				}
				files[path] = hash(raw)
				metricPath := "metrics/" + id + ".json"
				metricRaw, err := read(filepath.Join(dir, metricPath))
				var m metrics
				if err != nil || json.Unmarshal(metricRaw, &m) != nil || m.Wall <= 0 || m.User < 0 || m.System < 0 || m.RSS <= 0 ||
					m.CPU != 100*float64(m.User+m.System)/float64(m.Wall) {
					return nil, errors.New("native resource observation invalid")
				}
				files[metricPath] = hash(metricRaw)
				skipped, err := inspectNativeUnfixedFor(value, row, arm, ci, unfixed, m.Wall)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", id, err)
				}
				sha := hash([]byte(value.Source))
				executionPath := "executions/" + sha + ".json"
				executionRaw, err := read(filepath.Join(dir, executionPath))
				var values []int64
				if err != nil || json.Unmarshal(executionRaw, &values) != nil || len(values) != len(inputs) {
					return nil, errors.New("actual pilot Go values missing")
				}
				files[executionPath], executions[sha] = hash(executionRaw), true
				mask, err := compoundstudy.Mask(row.Document.Plan, value.Report.Paths.Search.Selection.Choices)
				if err != nil {
					return nil, err
				}
				for i, input := range inputs {
					want, err := compoundstudy.Oracle(row.Template, mask, input)
					if err != nil || values[i] != want {
						return nil, errors.New("actual generated Go differs from independent state oracle")
					}
				}
				if !unfixed {
					legacyCalls += value.Report.Paths.Search.Selection.ModelCalls
					previous = value
				} else {
					pairs++
					optCalls += value.Report.Paths.Search.Selection.ModelCalls
					skips += skipped
					if !reflect.DeepEqual(previous.Report.Paths.Search.Attempts, value.Report.Paths.Search.Attempts) {
						changedSequences++
					}
					if previous.Source != value.Source || previous.Report.Paths.Completeness != value.Report.Paths.Completeness {
						changedBodies++
					}
				}
				observations = append(observations, map[string]any{"id": id, "case_id": row.ID, "arm": arm.Name,
					"unfixed": unfixed, "capture_sha256": hash(raw), "generated_go_sha256": sha, "process_metrics": m,
					"model_predictions": value.Report.Paths.Search.Selection.ModelCalls, "prediction_skips": skipped,
					"candidate_attempts": len(value.Report.Paths.Search.Attempts), "finite_passed": value.Report.Paths.Search.SelectedTrainingPassed,
					"finite_cases": len(row.Document.Cases)})
			}
		}
	}
	return map[string]any{"schema": "gooo/native-unfixed-pilot/v1", "decision": "PASS",
		"runner_revision": revision, "native_revision": native, "preexecution_sha256": hash(pre), "files_sha256": files,
		"observations": observations, "native_calls": len(observations), "legacy_predictions": legacyCalls,
		"opt_in_predictions": optCalls, "actual_model_predictions": legacyCalls + optCalls, "observed_prediction_skips": skips,
		"paired_views": pairs, "changed_candidate_sequences": changedSequences, "changed_final_body_or_finite_results": changedBodies,
		"actual_go_processes": len(executions), "actual_function_evaluations": len(executions) * len(inputs), "execution_inputs": inputs,
		"scope": "Actual clean native SDK 2.7 codegen and own models; six reused bilingual/contradictory views and 24 legacy-first pairs, not independent new tasks. Native bodies are source-bound, type/replay checked and matched to selected typed function AST and independent integer-state oracle. Fixed arm order, whole compiler child resources, no causal speedup or host CPU utilization claim. Go execution deduplicates identical emitted source. Resource sidecars are retained observations. No training, GPU work, upstream Laya calls or source writes; offline audit performs zero inference or subprocess calls."}, nil
}
