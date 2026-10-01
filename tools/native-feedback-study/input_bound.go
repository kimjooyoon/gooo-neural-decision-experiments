package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func probeInputBound(binary, root, output, revision, nativeRevision string) error {
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("pinned native binary required")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if settings["vcs.revision"] != nativeRevision || settings["vcs.modified"] != "false" {
		return errors.New("clean native source required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("runner source mismatch")
	}
	if err = exec.Command("git", "diff", "--quiet", "HEAD", "--", "tools/native-feedback-study").Run(); err != nil {
		return errors.New("dirty probe source")
	}
	if filepath.IsAbs(output) || filepath.Clean(output) != output || !strings.HasPrefix(output, "runs/") {
		return errors.New("fresh relative output required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh output required")
	}
	modelPath, err := filepath.Abs("runs/typed-path-positioned-random-20261001/models/fp32/model.json")
	if err != nil {
		return err
	}
	model, err := decision.LoadPath(modelPath)
	if err != nil || model.MetadataSHA256() != modelPins["fp32"] {
		return errors.New("fixed model required")
	}
	source, err := read(filepath.Join(root, "examples/body-codegen/typed-path-conditional-assignment.gooo.fixture"))
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "gooo-input-bound-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err = os.WriteFile(filepath.Join(dir, "fixture.gooo"), source, 0600); err != nil {
		return err
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/feedback-input-bound-probe-preexecution/v1", "runner_revision": revision, "native_revision": nativeRevision, "model_metadata_sha256": model.MetadataSHA256(), "model_weights_sha256": model.WeightsSHA256(), "languages": 2, "calls": 4, "source_sha256": hash(source), "scope": "Synthetic legal long intentions under 512 bytes; observe optional feedback overflow without changing compiler checks."}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var records []map[string]any
	for _, lang := range []string{"en", "ko"} {
		name := "typed-path-conditional-assignment-plan.json"
		if lang == "ko" {
			name = "typed-path-conditional-assignment-ko-plan.json"
		}
		raw, err := read(filepath.Join(root, "examples/body-codegen", name))
		if err != nil {
			return err
		}
		var d document
		if err = json.Unmarshal(raw, &d); err != nil {
			return err
		}
		d.Cases[6].Expected = 999
		padding := " detail"
		if lang == "ko" {
			padding = " 설명"
		}
		for len(d.Plan.Decisions[0].Intent)+len(padding) <= 480 {
			d.Plan.Decisions[0].Intent += padding
		}
		if err = save(filepath.Join(output, lang+"-plan.json"), d); err != nil {
			return err
		}
		if err = save(filepath.Join(dir, "plan.json"), d); err != nil {
			return err
		}
		for _, feedback := range []bool{false, true} {
			mode := "baseline"
			if feedback {
				mode = "feedback"
			}
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-model", modelPath, "--path-step-attempts", "8", "--activity", "ConditionalAssign", "fixture.gooo"}
			if feedback {
				args = append(args, "--path-feedback-rounds", "2")
			}
			capture, m, callErr := child(ctx, dir, binary, args...)
			if err = os.WriteFile(filepath.Join(output, lang+"-"+mode+".json"), capture, 0644); err != nil {
				return err
			}
			var result nativeResult
			if !feedback {
				if callErr != nil {
					return callErr
				}
				if err = json.Unmarshal(capture, &result); err != nil {
					return err
				}
				if err = inspect(result, false, 6, nativeRevision); err != nil {
					return err
				}
				if result.Report.Paths.Search.SelectedTrainingPassed != 6 {
					return errors.New("long-intent baseline lost partial result")
				}
				records = append(records, map[string]any{"language": lang, "mode": mode, "intent_bytes": len(d.Plan.Decisions[0].Intent), "capture_sha256": hash(capture), "decision": result.Report.Decision, "model_predictions": result.Report.Paths.Search.Selection.ModelCalls, "candidate_attempts": len(result.Report.Paths.Search.Attempts), "passed": 6, "finite_cases": 7, "emitted_go": true, "process_metrics": m})
			} else {
				var failure struct {
					Decision string `json:"decision"`
					Error    string `json:"error"`
					Paths    struct {
						Search   pathplan.SearchResult      `json:"search"`
						Progress []pathplan.SessionProgress `json:"session_progress"`
					} `json:"body_paths"`
				}
				if err = json.Unmarshal(capture, &failure); err != nil {
					return err
				}
				if callErr == nil || failure.Decision != "FAIL_CLOSED" || !strings.Contains(failure.Error, "feedback context exceeds") || failure.Paths.Search.Selection.ModelCalls != 6 || len(failure.Paths.Search.Attempts) != 8 {
					return errors.New("expected explicit optional feedback input-bound failure differs")
				}
				records = append(records, map[string]any{"language": lang, "mode": mode, "intent_bytes": len(d.Plan.Decisions[0].Intent), "capture_sha256": hash(capture), "decision": failure.Decision, "error": failure.Error, "model_predictions": 6, "feedback_predictions": 0, "candidate_attempts": 8, "passed": failure.Paths.Search.SelectedTrainingPassed, "finite_cases": 7, "emitted_go": false, "process_metrics": m})
			}
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/feedback-input-bound-probe/v1", "runner_revision": revision, "native_revision": nativeRevision, "native_calls": 4, "actual_model_predictions": 24, "feedback_predictions": 0, "records": records, "scope": "Two synthetic long-intent variants of the same existing compound source, not new independent tasks or model training. Baseline retains 6/7 and emits Go; explicit feedback currently stops after its first 8 candidates before new feedback predictions. This reproduces a recoverable representation-bound veto; it does not modify frozen primary measurements."})
}
