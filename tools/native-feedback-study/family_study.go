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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/familystudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

const familyCohortSHA = "0e9d5c2cb9a816ca4d05c2e6e2d94ceccb9a0d02912edd4cc1063c0adb648a70"
const familyNativeSHA = "4dced73b26dde567cb7129f3e4ba5733850196d0"

type familySpec struct {
	Revision, SDK string
	NoChoice      bool
}

var legacyFamily = familySpec{Revision: familyNativeSHA, SDK: "v0.2.4-experimental"}

type familyArm struct {
	Name, Variant, Path, Metadata, Weights string
	Feedback                               bool
}
type familyObservation struct {
	ID                  string  `json:"id"`
	CaseID              string  `json:"case_id"`
	Arm                 string  `json:"arm"`
	Feedback            bool    `json:"feedback_enabled"`
	Capture             string  `json:"capture_sha256"`
	GoSHA               string  `json:"generated_go_sha256"`
	Selected            string  `json:"selected_label"`
	IntentionAgreement  bool    `json:"original_intention_label_agreement"`
	FiniteBest          bool    `json:"finite_best_set_selected"`
	Passed              int     `json:"finite_passed"`
	Cases               int     `json:"finite_cases"`
	SeparatePassed      int     `json:"separate_cases_passed"`
	SeparateCases       int     `json:"separate_cases"`
	Attempts            int     `json:"candidate_attempts"`
	Predictions         int     `json:"actual_model_predictions"`
	FeedbackPredictions int     `json:"feedback_predictions"`
	NoChoice            int     `json:"zero_call_ranking_unnecessary_receipts,omitempty"`
	Metrics             metrics `json:"process_metrics"`
}

func familyRows() ([]familystudy.Case, error) {
	raw, err := read("studies/feedback-family-v1/cohort.jsonl")
	if err != nil || hash(raw) != familyCohortSHA {
		return nil, errors.New("fixed family cohort required")
	}
	var rows []familystudy.Case
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var row familystudy.Case
		if err = json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	fresh, err := familystudy.Cohort()
	if err != nil || !reflect.DeepEqual(rows, fresh) {
		return nil, errors.New("independent authored cohort changed")
	}
	return rows, nil
}
func executableHash(path string) (string, error) {
	i, err := os.Lstat(path)
	if err != nil || !i.Mode().IsRegular() || i.Size() > 64<<20 {
		return "", errors.New("bounded regular executable required")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, 64<<20)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func familyPreflight(binary, goBinary, output, revision string) ([]familystudy.Case, []familyArm, error) {
	return familyPreflightFor(binary, goBinary, output, revision, legacyFamily)
}
func familyPreflightFor(binary, goBinary, output, revision string, spec familySpec) ([]familystudy.Case, []familyArm, error) {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(spec.Revision) || spec.NoChoice && spec.SDK != "v0.2.5-experimental" {
		return nil, nil, errors.New("explicit source-pinned family SDK contract required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return nil, nil, errors.New("family runner source differs")
	}
	if err = exec.Command("git", "diff", "--quiet", "HEAD", "--", "tools/native-feedback-study", "internal/familystudy", "studies/feedback-family-v1").Run(); err != nil {
		return nil, nil, errors.New("family source dirty")
	}
	u, err := exec.Command("git", "ls-files", "--others", "--exclude-standard", "--", "tools/native-feedback-study", "internal/familystudy", "studies/feedback-family-v1").Output()
	if err != nil || len(u) != 0 {
		return nil, nil, errors.New("family source untracked")
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return nil, nil, errors.New("pinned native Go toolchain required")
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
	if settings["vcs.revision"] != spec.Revision || settings["vcs.modified"] != "false" || sdk != spec.SDK {
		return nil, nil, errors.New("clean main native source and SDK required")
	}
	g, err := buildinfo.ReadFile(goBinary)
	if err != nil || g.GoVersion != "go1.27.1" {
		return nil, nil, errors.New("Go 1.27.1 execution required")
	}
	if filepath.IsAbs(output) || filepath.Clean(output) != output || !strings.HasPrefix(output, "runs/") {
		return nil, nil, errors.New("fresh relative run directory required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return nil, nil, errors.New("existing run cannot be overwritten")
	}
	rows, err := familyRows()
	if err != nil {
		return nil, nil, err
	}
	arms := []familyArm{{Name: "offline"}}
	for _, choice := range []struct{ Name, Root, Variant, Pin string }{
		{"parent_fp32", "runs/typed-path-positioned-random-20261001/models", "fp32", modelPins["fp32"]},
		{"feedback_fp32", "runs/feedback-path-soft-target-mps-20261001/models", "fp32", trainedPins["fp32"]},
		{"feedback_ptq", "runs/feedback-path-soft-target-mps-20261001/models", "ptq_ternary", trainedPins["ptq_ternary"]},
		{"feedback_qat", "runs/feedback-path-soft-target-mps-20261001/models", "qat_ternary", trainedPins["qat_ternary"]},
	} {
		path, err := filepath.Abs(filepath.Join(choice.Root, choice.Variant, "model.json"))
		if err != nil {
			return nil, nil, err
		}
		model, err := decision.LoadPath(path)
		if err != nil || model.MetadataSHA256() != choice.Pin {
			return nil, nil, errors.New("family model pin differs")
		}
		for _, feedback := range []bool{false, true} {
			arms = append(arms, familyArm{Name: choice.Name, Variant: choice.Variant, Path: path, Metadata: choice.Pin, Weights: model.WeightsSHA256(), Feedback: feedback})
		}
	}
	return rows, arms, nil
}
func inspectFamily(v nativeResult, r familystudy.Case, a familyArm) (familyObservation, error) {
	return inspectFamilyFor(v, r, a, legacyFamily)
}
func inspectFamilyFor(v nativeResult, r familystudy.Case, a familyArm, spec familySpec) (familyObservation, error) {
	p := v.Report.Paths
	if v.Report.Decision != "PASS" || !v.Report.Types || !v.Report.Replay || v.Report.Writes != 0 || v.Report.Compiler != spec.Revision || !p.Bound || v.Source == "" || len(p.Cases) != len(r.Document.Cases) || p.Search.DeclaredCombinations != 2 || len(p.Search.Attempts) < 1 || len(p.Search.Attempts) > 2 || p.Search.Selection.ExternalCalls != 0 {
		return familyObservation{}, errors.New("family native verification differs")
	}
	prepared, err := pathplan.Prepare(r.Document.Plan)
	if err != nil {
		return familyObservation{}, err
	}
	if p.Search.Selection.PlanSHA256 != prepared.PlanSHA256() || p.Search.Selection.MetadataSHA256 != a.Metadata || p.Search.Selection.WeightsSHA256 != a.Weights {
		return familyObservation{}, errors.New("family source/model binding differs")
	}
	seen := map[uint16]bool{}
	evaluated := 0
	for _, candidate := range p.Search.Attempts {
		if seen[candidate.Mask] {
			return familyObservation{}, errors.New("repeated family candidate")
		}
		seen[candidate.Mask] = true
		body, err := prepared.Compile(candidate.Choices)
		if err != nil || candidate.GoooSHA != hash([]byte(body.GoooSource())) || len(candidate.Results) != len(r.Document.Cases) {
			return familyObservation{}, errors.New("family candidate body differs")
		}
		passed := 0
		for i, c := range r.Document.Cases {
			value, err := body.Evaluate(c.Input)
			if err != nil || candidate.Results[i].Input != c.Input || candidate.Results[i].Expected != c.Expected || candidate.Results[i].Actual != value.Int || candidate.Results[i].Passed != (value.Int == c.Expected) {
				return familyObservation{}, errors.New("family finite outcome differs")
			}
			if value.Int == c.Expected {
				passed++
			}
		}
		if passed != candidate.Passed {
			return familyObservation{}, errors.New("family finite count differs")
		}
		evaluated++
	}
	var recorded []pathplan.SearchAttempt
	prior := ""
	progressBySHA := map[string]pathplan.SessionProgress{}
	for i, progress := range p.Progress {
		sha := progress.SHA
		progress.SHA = ""
		encoded, err := json.Marshal(progress)
		if err != nil || hash(encoded) != sha || progress.Sequence != i+1 || progress.PreviousSHA != prior || progress.Interrupted || progress.PredictionsThisAdvance != 0 {
			return familyObservation{}, errors.New("family progress binding differs")
		}
		prior = sha
		progress.SHA = sha
		progressBySHA[sha] = progress
		recorded = append(recorded, progress.NewAttempts...)
	}
	if !reflect.DeepEqual(recorded, p.Search.Attempts) {
		return familyObservation{}, errors.New("family progress lost candidates")
	}
	initial := 0
	if a.Path != "" {
		initial = 1
	}
	feedbackCalls := 0
	noChoice := 0
	prior = ""
	for i, f := range p.Feedback {
		sha := f.SHA
		f.SHA = ""
		encoded, err := json.Marshal(f)
		from, ok := progressBySHA[f.FromProgressSHA]
		if err != nil || hash(encoded) != sha || !ok || f.PreviousSHA != prior || f.Round != i+1 || !a.Feedback || f.CIIsAuthority || f.CI == nil || f.CI.SourceSHA != spec.Revision || f.CI.Status != "PASS" || f.MetadataSHA != a.Metadata || f.WeightsSHA != a.Weights || from.Attempted != f.Attempted || f.PlanSHA != prepared.PlanSHA256() || f.CaseSHA != from.CaseSHA || f.Passed != from.SelectedPassed || f.Cases != from.Cases || f.TypeRejected != from.TypeRejected {
			return familyObservation{}, errors.New("family feedback binding differs")
		}
		var first *pathplan.TestResult
		for _, result := range from.BestCases {
			if !result.Passed {
				copy := result
				first = &copy
				break
			}
		}
		if !reflect.DeepEqual(first, f.FirstFailure) {
			return familyObservation{}, errors.New("family failure binding differs")
		}
		if spec.NoChoice {
			if !f.RankingUnnecessary || f.ModelCalls != 0 || f.CumulativeCalls != initial+feedbackCalls || f.Applied || f.AddedMask || f.ContextDeclined || len(f.Judgments) != 0 || f.Error != "" || from.Declared-from.Attempted != 1 {
				return familyObservation{}, errors.New("sole remaining path performed a ranking or lost its zero-call receipt")
			}
			noChoice++
			prior = sha
			continue
		}
		if f.RankingUnnecessary || f.ModelCalls != 1 || f.CumulativeCalls != initial+feedbackCalls+1 || !f.Applied || f.ContextDeclined || len(f.Judgments) != 1 {
			return familyObservation{}, errors.New("legacy family ranking differs")
		}
		j := f.Judgments[0]
		intent := r.Document.Plan.Decisions[0].Intent
		if j.InputSHA != hash([]byte(j.Input)) || !strings.HasSuffix(j.Input, "\nintent: "+intent) || j.Prediction.IntentSHA256 != hash([]byte(intent)) {
			return familyObservation{}, errors.New("family feedback changed original intent")
		}
		feedbackCalls++
		prior = sha
	}
	if p.Search.Selection.ModelCalls != initial+feedbackCalls || len(p.Feedback) > 1 || evaluated != p.Search.Evaluated || p.Search.TypeRejected != 0 {
		return familyObservation{}, errors.New("family prediction accounting differs")
	}
	if a.Feedback && evaluated == 2 && len(p.Feedback) != 1 || !a.Feedback && len(p.Feedback) != 0 || len(p.Progress) == 0 {
		return familyObservation{}, errors.New("family feedback observation missing or unexpected")
	}
	last := p.Progress[len(p.Progress)-1]
	if last.FeedbackRounds != len(p.Feedback) || last.FeedbackPredictions != feedbackCalls || last.LatestFeedbackSHA != prior || last.Selection.ModelCalls != initial+feedbackCalls {
		return familyObservation{}, errors.New("family final feedback accounting differs")
	}
	selected := p.Search.Selection.Choices["structure"]
	compiled, err := prepared.Compile(p.Search.Selection.Choices)
	if err != nil {
		return familyObservation{}, err
	}
	passed := 0
	for i, c := range r.Document.Cases {
		value, err := compiled.Evaluate(c.Input)
		if err != nil || p.Cases[i].Input != c.Input || p.Cases[i].Expected != c.Expected || p.Cases[i].Actual != value.Int || p.Cases[i].Passed != (value.Int == c.Expected) {
			return familyObservation{}, errors.New("family native selected cases differ")
		}
		if value.Int == c.Expected {
			passed++
		}
	}
	separate := 0
	for _, c := range r.Separate {
		value, err := compiled.Evaluate(c.Input)
		if err != nil {
			return familyObservation{}, err
		}
		if value.Int == c.Expected {
			separate++
		}
	}
	best := false
	for _, label := range r.FiniteBestLabels {
		best = best || selected == label
	}
	if passed != p.Search.SelectedTrainingPassed || p.Completeness != 100*float64(passed)/float64(len(r.Document.Cases)) {
		return familyObservation{}, errors.New("family native completeness differs")
	}
	return familyObservation{CaseID: r.ID, Arm: a.Name, Feedback: a.Feedback, GoSHA: hash([]byte(v.Source)), Selected: selected, IntentionAgreement: selected == r.IntentionLabel, FiniteBest: best, Passed: passed, Cases: len(r.Document.Cases), SeparatePassed: separate, SeparateCases: len(r.Separate), Attempts: evaluated, Predictions: initial + feedbackCalls, FeedbackPredictions: feedbackCalls, NoChoice: noChoice}, nil
}
func runFamily(binary, goBinary, output, revision string, pilot bool) error {
	return runFamilyFor(binary, goBinary, output, revision, pilot, legacyFamily)
}
func runFamilyFor(binary, goBinary, output, revision string, pilot bool, spec familySpec) error {
	rows, arms, err := familyPreflightFor(binary, goBinary, output, revision, spec)
	if err != nil {
		return err
	}
	if pilot {
		var subset []familystudy.Case
		for _, r := range rows {
			if r.Language == "en" && !r.Reverse && r.Contract == "complete" {
				subset = append(subset, r)
			}
		}
		rows = subset
		arms = arms[:1]
	}
	binSHA, err := executableHash(binary)
	if err != nil {
		return err
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(output, "captures"), 0755); err != nil {
		return err
	}
	pins := map[string]map[string]string{}
	for _, a := range arms {
		pins[a.Name] = map[string]string{"metadata": a.Metadata, "weights": a.Weights}
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/native-feedback-family-preexecution/v1", "runner_revision": revision, "native_revision": spec.Revision, "binary_sha256": binSHA, "go_binary_sha256": goSHA, "go": "1.27.1", "sdk": spec.SDK, "cohort_sha256": familyCohortSHA, "planned_native_calls": len(rows) * len(arms), "models": pins, "caller_ci_hint": pathplan.CIHint{SourceSHA: spec.Revision, Status: "PASS"}, "caller_ci_hint_is_authority": false, "pilot": pilot}); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "gooo-family-study-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err = save(filepath.Join(dir, "ci.json"), pathplan.CIHint{SourceSHA: spec.Revision, Status: "PASS"}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var observations []familyObservation
	generated := map[string]string{}
	inputs := []int64{}
	inputSeen := map[int64]bool{}
	for _, r := range rows {
		for _, c := range append(append([]pathplan.TestCase(nil), r.Document.Cases...), r.Separate...) {
			if !inputSeen[c.Input] {
				inputs = append(inputs, c.Input)
				inputSeen[c.Input] = true
			}
		}
	}
	for _, r := range rows {
		if err = os.WriteFile(filepath.Join(dir, "source.gooo"), []byte(r.Source), 0600); err != nil {
			return err
		}
		if err = save(filepath.Join(dir, "plan.json"), r.Document); err != nil {
			return err
		}
		for _, a := range arms {
			id := fmt.Sprintf("%s-%s-%t", r.ID, a.Name, a.Feedback)
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ChoosePath"}
			if a.Path != "" {
				args = append(args, "--path-model", a.Path)
			}
			if a.Feedback {
				args = append(args, "--path-feedback-rounds", "1", "--path-feedback-ci", "ci.json")
			}
			args = append(args, "source.gooo")
			raw, m, childErr := child(ctx, dir, binary, args...)
			if err = os.WriteFile(filepath.Join(output, "captures", id+".json"), raw, 0644); err != nil {
				return err
			}
			if childErr != nil {
				return fmt.Errorf("family native failure retained at %s", id)
			}
			var v nativeResult
			if err = json.Unmarshal(raw, &v); err != nil {
				return err
			}
			o, err := inspectFamilyFor(v, r, a, spec)
			if err != nil {
				return fmt.Errorf("%s: %w", id, err)
			}
			o.ID, o.Capture, o.Metrics = id, hash(raw), m
			observations = append(observations, o)
			generated[o.GoSHA] = v.Source
		}
	}
	if err = os.Mkdir(filepath.Join(output, "executions"), 0755); err != nil {
		return err
	}
	executions := map[string][]int64{}
	for sha, source := range generated {
		raw, err := executeFunction(ctx, goBinary, source, "ChoosePath", inputs)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "executions", sha+".json"), raw, 0644); err != nil {
			return err
		}
		var actual []int64
		if err = json.Unmarshal(raw, &actual); err != nil || len(actual) != len(inputs) {
			return errors.New("family Go execution shape differs")
		}
		executions[sha] = actual
	}
	byID := map[string]familystudy.Case{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	predictions, feedbackPredictions, attempts, passed, total, separatePassed, separateTotal := 0, 0, 0, 0, 0, 0, 0
	for _, o := range observations {
		r := byID[o.CaseID]
		reverse := o.Selected == pathstudy.GoldLabel(r.Family, true)
		for i, input := range inputs {
			expected, err := pathstudy.Oracle(r.Family, reverse, r.Configuration, input)
			if err != nil || executions[o.GoSHA][i] != expected {
				return errors.New("actual family generated Go differs from independent selected-path oracle")
			}
		}
		predictions += o.Predictions
		feedbackPredictions += o.FeedbackPredictions
		attempts += o.Attempts
		passed += o.Passed
		total += o.Cases
		separatePassed += o.SeparatePassed
		separateTotal += o.SeparateCases
	}
	scope := "Fixed baseline-first development matrix across five existing families and ten source bodies. Repeated contracts/languages/arms are not independent tasks. Actual Go is deduplicated per emitted source and checked against an independent arithmetic oracle; reuse is counted separately. No host utilization or causal performance improvement is inferred. SDK v0.2.4 raw receipt wording describes a non-feedback-trained model even for the newly tuned models; that description is a known metadata-scope defect, not an actual training fact."
	if spec.NoChoice {
		scope = "SDK v0.2.5 sole-remaining-path continuation; same frozen development cohort and models. Zero-call receipts retain failures and lineage. Compare source, selected labels, finite/separate outcomes and attempts to the frozen v0.2.4 matrix. Repeated policy observations are not independent tasks or new Go executions. One fixed ordering does not establish a causal wall-time or host-utilization improvement."
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/native-feedback-family-study/v1", "decision": "PASS", "runner_revision": revision, "native_revision": spec.Revision, "pilot": pilot, "observations": observations, "native_calls": len(observations), "actual_model_predictions": predictions, "feedback_predictions": feedbackPredictions, "candidate_attempts": attempts, "repeated_finite_passes": passed, "repeated_finite_cases": total, "repeated_separate_input_passes": separatePassed, "repeated_separate_input_cases": separateTotal, "actual_generated_go_processes": len(executions), "actual_generated_function_evaluations": len(executions) * len(inputs), "execution_inputs": inputs, "scope": scope})
}
