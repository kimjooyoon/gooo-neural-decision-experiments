package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var trainedPins = map[string]string{
	"fp32":        "47bd3ed2037c8ba0af31ec4cad3a47fe1182af171845406aba01c5107f4b24b7",
	"ptq_ternary": "a7b66676af2eb5a7e6952f3c65c812f231865ac4974a3c3c1f329b125e55124c",
	"qat_ternary": "17dd98b2c05c7a9c203523a2e8e70ba0cbb523ae47715fdc0dc42acef81b0536",
}

func trainedDogfood(binary, goBinary, output, revision, nativeRevision string) error {
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 native binary required")
	}
	settings := map[string]string{}
	for _, v := range info.Settings {
		settings[v.Key] = v.Value
	}
	sdk := ""
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = d.Version
		}
	}
	if settings["vcs.revision"] != nativeRevision || settings["vcs.modified"] != "false" || sdk != "v0.2.4-experimental" {
		return errors.New("clean source-pinned native SDK v0.2.4 required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("runner revision differs")
	}
	if err = exec.Command("git", "diff", "--quiet", "HEAD", "--", "tools/native-feedback-study", "runs/feedback-path-soft-target-mps-20261001/models").Run(); err != nil {
		return errors.New("runner or model sources dirty")
	}
	untracked, err := exec.Command("git", "ls-files", "--others", "--exclude-standard", "--", "tools/native-feedback-study").Output()
	if err != nil || len(untracked) != 0 {
		return errors.New("runner source untracked")
	}
	if filepath.IsAbs(output) || filepath.Clean(output) != output || !strings.HasPrefix(output, "runs/") {
		return errors.New("fresh relative output required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("existing capture cannot be overwritten")
	}
	models, weights := map[string]string{}, map[string]string{}
	for variant, pin := range trainedPins {
		path, err := filepath.Abs(filepath.Join("runs/feedback-path-soft-target-mps-20261001/models", variant, "model.json"))
		if err != nil {
			return err
		}
		model, err := decision.LoadPath(path)
		if err != nil || model.MetadataSHA256() != pin {
			return errors.New("trained structural model differs")
		}
		models[variant], weights[variant] = path, model.WeightsSHA256()
	}
	source, err := read("studies/native-feedback-v1/examples/body-codegen/typed-path-conditional-assignment.gooo.fixture")
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "gooo-trained-dogfood-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err = os.WriteFile(filepath.Join(dir, "fixture.gooo"), source, 0600); err != nil {
		return err
	}
	ci := pathplan.CIHint{SourceSHA: "3b2c11b418479d4a5c4df845425678fd25b0a695", Status: "PASS"}
	if err = save(filepath.Join(dir, "hint.json"), ci); err != nil {
		return err
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/trained-feedback-dogfood-preexecution/v1",
		"runner_revision": revision, "native_revision": nativeRevision, "go": "1.27.1", "sdk": sdk,
		"models": trainedPins, "weights": weights, "source_sha256": hash(source), "planned_native_calls": 6,
		"caller_ci_hint": ci, "ci_hint_is_authority": false, "scope": "Three new small models, two languages, one existing compound source and deliberately inconsistent seven-case contract; finite partial construction, not six independent tasks."}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var observations []map[string]any
	generated := map[string]string{}
	var inputs, expected []int64
	calls := 0
	for _, language := range []string{"en", "ko"} {
		name := "typed-path-conditional-assignment-plan.json"
		if language == "ko" {
			name = "typed-path-conditional-assignment-ko-plan.json"
		}
		originalRaw, err := read(filepath.Join("studies/native-feedback-v1/examples/body-codegen", name))
		if err != nil {
			return err
		}
		var original document
		if err = json.Unmarshal(originalRaw, &original); err != nil || len(original.Cases) != 7 {
			return errors.New("fixed native document required")
		}
		if language == "en" {
			for _, c := range original.Cases {
				inputs, expected = append(inputs, c.Input), append(expected, c.Expected)
			}
			separate, err := read("studies/conditional-paths-v1/cohort/holdout-cases.json")
			if err != nil {
				return err
			}
			var cases []pathplan.TestCase
			if err = json.Unmarshal(separate, &cases); err != nil || len(cases) != 9 {
				return errors.New("nine fixed disjoint inputs required")
			}
			for _, c := range cases {
				inputs, expected = append(inputs, c.Input), append(expected, c.Expected)
			}
		}
		d := original
		d.Cases = append([]pathplan.TestCase(nil), original.Cases...)
		d.Cases[6].Expected = 999
		if err = save(filepath.Join(dir, "plan.json"), d); err != nil {
			return err
		}
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			raw, m, callErr := child(ctx, dir, binary, "body-codegen", "--json", "--path-plan", "plan.json", "--path-model", models[variant],
				"--path-step-attempts", "8", "--path-feedback-rounds", "2", "--path-feedback-ci", "hint.json", "--activity", "ConditionalAssign", "fixture.gooo")
			if err = os.WriteFile(filepath.Join(output, language+"-"+variant+".json"), raw, 0644); err != nil {
				return err
			}
			if callErr != nil {
				return callErr
			}
			var value nativeResult
			if err = json.Unmarshal(raw, &value); err != nil {
				return err
			}
			if err = inspect(value, true, 6, nativeRevision); err != nil {
				return err
			}
			p := value.Report.Paths
			if p.Search.Evaluated != 64 || p.Search.SelectedTrainingPassed != 6 || len(p.Feedback) != 2 ||
				p.Search.Selection.ModelCalls != 18 || p.Search.Selection.MetadataSHA256 != trainedPins[variant] {
				return errors.New("trained model lost continued partial construction or actual call accounting")
			}
			calls += p.Search.Selection.ModelCalls
			sha := hash([]byte(value.Source))
			generated[sha] = value.Source
			observations = append(observations, map[string]any{"language": language, "variant": variant, "capture_sha256": hash(raw),
				"generated_go_sha256": sha, "candidate_attempts": 64, "actual_local_predictions": 18,
				"feedback_predictions": 12, "finite_passed": 6, "finite_cases": 7, "process_metrics": m})
		}
	}
	if err = os.Mkdir(filepath.Join(output, "executions"), 0755); err != nil {
		return err
	}
	for sha, source := range generated {
		raw, err := executeGenerated(ctx, goBinary, source, inputs)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "executions", sha+".json"), raw, 0644); err != nil {
			return err
		}
		var actual []int64
		if err = json.Unmarshal(raw, &actual); err != nil || !reflect.DeepEqual(actual, expected) {
			return errors.New("actual trained-model emitted Go differs from authored finite and independent inputs")
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/trained-feedback-dogfood/v1", "decision": "PASS",
		"runner_revision": revision, "native_revision": nativeRevision, "native_calls": 6, "actual_local_predictions": calls,
		"feedback_predictions": 72, "candidate_attempts": 384, "repeated_finite_passes": 36, "repeated_finite_cases": 42,
		"actual_generated_go_processes": len(generated), "actual_generated_function_evaluations": len(generated) * len(inputs),
		"actual_separate_input_evaluations": len(generated) * 9, "distinct_separate_inputs": 9,
		"repeated_separate_input_policy_observations": 54, "execution_inputs": inputs, "execution_expected": expected,
		"observations": observations, "scope": "Actual new-model native dogfood on one existing compound source, with six policy/language views and preserved 6/7 partial contracts. Exact emitted Go is deduplicated and executed on seven selection inputs plus nine old disjoint inputs; reused policy observations are not independent tasks. Host utilization is not sampled."})
}
