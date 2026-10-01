package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func inspectDeclines(value nativeResult, d document, revision string) error {
	feedback := value.Report.Paths.Feedback
	value.Report.Paths.Feedback = nil
	if err := inspect(value, false, 6, revision); err != nil {
		return err
	}
	p := value.Report.Paths
	if value.Source == "" || value.Report.ActivityID == "" || p.Search.Evaluated != 64 ||
		p.Search.SelectedTrainingPassed != 6 || len(feedback) != 2 {
		return errors.New("continued partial body differs")
	}
	progress := map[string]pathplan.SessionProgress{}
	for _, observation := range p.Progress {
		sha := observation.SHA
		observation.SHA = ""
		raw, err := json.Marshal(observation)
		if err != nil || hash(raw) != sha || observation.Interrupted {
			return errors.New("continued progress digest differs")
		}
		observation.SHA = sha
		progress[sha] = observation
	}
	previous := ""
	for i, f := range feedback {
		prior, exists := progress[f.FromProgressSHA]
		sha := f.SHA
		f.SHA = ""
		raw, err := json.Marshal(f)
		if err != nil || hash(raw) != sha || !exists || prior.Attempted != f.Attempted ||
			f.Round != i+1 || f.PreviousSHA != previous || !f.ContextDeclined || f.Applied || f.AddedMask ||
			f.ModelCalls != 0 || f.CumulativeCalls != 6 || len(f.Judgments) != 0 || f.CIIsAuthority || f.CI != nil ||
			f.MetadataSHA != modelPins["fp32"] || f.DeclinedDecision != d.Plan.Decisions[0].ID ||
			f.DeclinedIntentSHA != hash([]byte(d.Plan.Decisions[0].Intent)) || f.FirstFailure == nil {
			return errors.New("zero-call decline lost its observation binding")
		}
		prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d", f.Attempted,
			f.Passed, f.Cases, f.TypeRejected, 64-f.Attempted)
		prefix += fmt.Sprintf(" mismatch=%d:%d:%d", f.FirstFailure.Input, f.FirstFailure.Actual, f.FirstFailure.Expected)
		input := prefix + " selected=" + prior.Selection.Choices[f.DeclinedDecision] + "\nintent: " + d.Plan.Decisions[0].Intent
		if len(input) <= 512 || f.DeclinedBytes != len(input) || f.DeclinedInputSHA != hash([]byte(input)) {
			return errors.New("declined input does not preserve the original intention")
		}
		previous = sha
	}
	last := p.Progress[len(p.Progress)-1]
	if last.FeedbackRounds != 2 || last.FeedbackPredictions != 0 || last.LatestFeedbackSHA != previous ||
		last.Attempted != 64 || !last.Exhausted {
		return errors.New("context decline interrupted remaining paths")
	}
	return nil
}

func probeContinuedBound(binary, goBinary, output, revision, nativeRevision string) error {
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("pinned native binary required")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	sdk := ""
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = d.Version
		}
	}
	if settings["vcs.revision"] != nativeRevision || settings["vcs.modified"] != "false" || sdk != "v0.2.4-experimental" {
		return errors.New("clean pinned native source and repaired SDK required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("runner source mismatch")
	}
	if err = exec.Command("git", "diff", "--quiet", "HEAD", "--", "tools/native-feedback-study",
		"runs/feedback-input-bound-20261001", "studies/native-feedback-v1", "runs/typed-path-positioned-random-20261001/models").Run(); err != nil {
		return errors.New("dirty probe source")
	}
	untracked, err := exec.Command("git", "ls-files", "--others", "--exclude-standard", "--", "tools/native-feedback-study").Output()
	if err != nil || len(untracked) != 0 {
		return errors.New("untracked probe source")
	}
	goInfo, err := buildinfo.ReadFile(goBinary)
	if err != nil || goInfo.GoVersion != "go1.27.1" {
		return errors.New("exact Go execution toolchain required")
	}
	loaded, err := decision.LoadPath("runs/typed-path-positioned-random-20261001/models/fp32/model.json")
	if err != nil || loaded.MetadataSHA256() != modelPins["fp32"] {
		return errors.New("fixed structural model required")
	}
	if filepath.IsAbs(output) || filepath.Clean(output) != output || !strings.HasPrefix(output, "runs/") {
		return errors.New("fresh relative output required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh output required")
	}
	binaryRaw, err := read(binary)
	if err != nil {
		// Native binaries are larger than the regular evidence-file limit.
		stat, statErr := os.Lstat(binary)
		if statErr != nil || !stat.Mode().IsRegular() || stat.Size() > 64<<20 {
			return errors.New("bounded native executable required")
		}
		binaryRaw, err = os.ReadFile(binary)
		if err != nil {
			return err
		}
	}
	source, err := read("studies/native-feedback-v1/examples/body-codegen/typed-path-conditional-assignment.gooo.fixture")
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "gooo-continued-bound-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err = os.WriteFile(filepath.Join(dir, "fixture.gooo"), source, 0600); err != nil {
		return err
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/continued-bound-preexecution/v1",
		"runner_revision": revision, "native_revision": nativeRevision, "native_binary_sha256": hash(binaryRaw),
		"sdk": sdk, "languages": 2, "native_calls_planned": 4, "source_sha256": hash(source),
		"source_plan_directory": "runs/feedback-input-bound-20261001", "model_metadata_sha256": modelPins["fp32"],
		"scope": "Repeat the two frozen long-intent variants using the repaired integration; no new intention family or training."}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	model, err := filepath.Abs("runs/typed-path-positioned-random-20261001/models/fp32/model.json")
	if err != nil {
		return err
	}
	var observations []map[string]any
	var generated string
	for _, lang := range []string{"en", "ko"} {
		plan, err := read("runs/feedback-input-bound-20261001/" + lang + "-plan.json")
		if err != nil {
			return err
		}
		var d document
		if err = json.Unmarshal(plan, &d); err != nil || len(d.Cases) != 7 || d.Cases[6].Expected != 999 {
			return errors.New("frozen long-intent document differs")
		}
		if err = os.WriteFile(filepath.Join(dir, "plan.json"), plan, 0600); err != nil {
			return err
		}
		var baseline nativeResult
		for _, feedback := range []bool{false, true} {
			mode := "baseline"
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-model", model,
				"--path-step-attempts", "8", "--activity", "ConditionalAssign", "fixture.gooo"}
			if feedback {
				mode = "feedback"
				args = append(args, "--path-feedback-rounds", "2")
			}
			raw, m, callErr := child(ctx, dir, binary, args...)
			if err = os.WriteFile(filepath.Join(output, lang+"-"+mode+".json"), raw, 0644); err != nil {
				return err
			}
			if callErr != nil {
				return callErr
			}
			var value nativeResult
			if err = json.Unmarshal(raw, &value); err != nil {
				return err
			}
			if feedback {
				if err = inspectDeclines(value, d, nativeRevision); err != nil {
					return err
				}
				if value.Source != baseline.Source || value.Report.ActivityID != baseline.Report.ActivityID ||
					!reflect.DeepEqual(value.Report.Paths.Search.Attempts, baseline.Report.Paths.Search.Attempts) {
					return errors.New("decline changed original ranking, candidate order or stable body")
				}
			} else {
				if err = inspect(value, false, 6, nativeRevision); err != nil {
					return err
				}
				baseline = value
			}
			if generated != "" && generated != value.Source {
				return errors.New("bilingual continued bodies unexpectedly differ")
			}
			generated = value.Source
			observations = append(observations, map[string]any{"language": lang, "mode": mode,
				"intent_bytes": len(d.Plan.Decisions[0].Intent), "capture_sha256": hash(raw), "plan_sha256": hash(plan),
				"generated_go_sha256": hash([]byte(value.Source)), "finite_passed": 6, "finite_cases": 7,
				"actual_local_predictions": value.Report.Paths.Search.Selection.ModelCalls, "candidate_attempts": 64,
				"declined_feedback_rounds": len(value.Report.Paths.Feedback), "feedback_predictions": 0,
				"native_typecheck": value.Report.Types, "deterministic_replay": value.Report.Replay, "process_metrics": m})
		}
	}
	inputs := []int64{-7, -1, 0, 1, 5, 6, 7}
	expected := []int64{-4, 8, 10, -18, -10, 22, 24}
	executed, err := executeGenerated(ctx, goBinary, generated, inputs)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(output, "go-execution.json"), executed, 0644); err != nil {
		return err
	}
	var actual []int64
	if err = json.Unmarshal(executed, &actual); err != nil || !reflect.DeepEqual(actual, expected) {
		return errors.New("actual emitted Go did not preserve the fixed source behavior")
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/continued-bound-probe/v1", "decision": "PASS",
		"runner_revision": revision, "native_revision": nativeRevision, "native_calls": 4, "actual_local_predictions": 24,
		"feedback_predictions": 0, "zero_call_context_declines": 4, "candidate_attempts": 256,
		"repeated_finite_passes": 24, "repeated_finite_cases": 28, "observations": observations,
		"actual_generated_go_processes": 1, "actual_generated_function_evaluations": 7, "execution_sha256": hash(executed),
		"execution_inputs": inputs, "execution_actual": actual,
		"scope": "Two existing compound long-intent variants; four actual native calls and one deduplicated emitted-Go execution. No first-shot accuracy claim, extra feedback predictions, new training, GPU or host utilization sampling. The deliberately inconsistent final expectation remains unmet (6/7)."})
}
