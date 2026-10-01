package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type unfixedCapture struct {
	CaseID   string                     `json:"case_id"`
	Arm      string                     `json:"arm"`
	Unfixed  bool                       `json:"unfixed_only"`
	Progress []pathplan.SessionProgress `json:"progress"`
	Feedback []pathplan.FeedbackReceipt `json:"feedback"`
	Search   pathplan.SearchResult      `json:"search"`
	Source   string                     `json:"generated_go"`
	WallNS   int64                      `json:"prepared_session_wall_ns"`
}

func captureUnfixed(ctx context.Context, prepared *pathplan.PreparedPlan, model *decision.Model,
	r compoundstudy.Case, arm familyArm, unfixed bool) (unfixedCapture, error) {
	c := unfixedCapture{CaseID: r.ID, Arm: arm.Name, Unfixed: unfixed}
	started := time.Now()
	session, err := prepared.NewSession(ctx, model, r.Document.Cases, "")
	if err != nil {
		return c, err
	}
	initial, err := session.Observe()
	if err != nil {
		return c, err
	}
	c.Progress = append(c.Progress, initial)
	var attempts []pathplan.SearchAttempt
	var body *bodyplan.Program
	for len(attempts) < 4 {
		progress, selected, failure := session.Advance(ctx, 1)
		if failure != nil {
			return c, failure
		}
		body = selected
		attempts = append(attempts, progress.NewAttempts...)
		c.Progress = append(c.Progress, progress)
		if progress.Status == "TRAINING_COMPLETE" || progress.Exhausted {
			break
		}
		hint := &pathplan.CIHint{SourceSHA: compoundNative, Status: "PASS"}
		var receipt pathplan.FeedbackReceipt
		if unfixed {
			receipt, err = session.ReconsiderUnfixed(ctx, model, hint)
		} else {
			receipt, err = session.Reconsider(ctx, model, hint)
		}
		if err != nil {
			return c, err
		}
		c.Feedback = append(c.Feedback, receipt)
		observed, err := session.Observe()
		if err != nil {
			return c, err
		}
		c.Progress = append(c.Progress, observed)
	}
	last := c.Progress[len(c.Progress)-1]
	c.Search = pathplan.SearchResult{Schema: "gooo/typed-path-tdd-search/v1", Status: last.Status,
		Selection: last.Selection, DeclaredCombinations: last.Declared, Unattempted: last.Unattempted,
		Evaluated: last.Evaluated, TypeRejected: last.TypeRejected, SelectedTrainingPassed: last.SelectedPassed,
		TrainingTotal: last.Cases, Attempts: attempts, ModelAbstentionsObserved: last.ModelAbstentions,
		EligibleProbabilities: last.EligibleProbabilities, InitialProposals: last.InitialProposals}
	c.Source, c.WallNS = body.GoSource(), time.Since(started).Nanoseconds()
	return c, nil
}

// verifyUnfixed interprets recorded candidates and verifies digest/call/skip
// lineage. It performs no predictions and starts no subprocess.
func verifyUnfixed(c unfixedCapture, r compoundstudy.Case, arm familyArm) (int, error) {
	return verifyUnfixedFor(c, r, arm, compoundNative)
}

func verifyUnfixedFor(c unfixedCapture, r compoundstudy.Case, arm familyArm, ciSource string) (int, error) {
	return verifyUnfixedWithCI(c, r, arm, pathplan.CIHint{SourceSHA: ciSource, Status: "PASS"})
}

func verifyUnfixedWithCI(c unfixedCapture, r compoundstudy.Case, arm familyArm, ci pathplan.CIHint) (int, error) {
	return verifyBoundedTrace(c, r, arm, ci, true)
}

// requireBest preserves the original full-budget study check. Partial-budget
// studies verify actual best-so-far results without requiring the global maximum.
func verifyBoundedTrace(c unfixedCapture, r compoundstudy.Case, arm familyArm, ci pathplan.CIHint, requireBest bool) (int, error) {
	if err := ci.Validate(); err != nil {
		return 0, err
	}
	if c.CaseID != r.ID || c.Arm != arm.Name || c.WallNS <= 0 || len(c.Progress) < 2 || len(c.Search.Attempts) < 1 || len(c.Search.Attempts) > 4 {
		return 0, errors.New("unfixed capture identity/bounds differ")
	}
	prepared, err := pathplan.Prepare(r.Document.Plan)
	if err != nil {
		return 0, err
	}
	casesRaw, _ := json.Marshal(r.Document.Cases)
	caseSHA := hash(casesRaw)
	var attempts []pathplan.SearchAttempt
	bySHA := map[string]pathplan.SessionProgress{}
	prior := ""
	for i, p := range c.Progress {
		sha := p.SHA
		p.SHA = ""
		raw, _ := json.Marshal(p)
		if hash(raw) != sha || p.PreviousSHA != prior || p.Sequence != i+1 || p.Interrupted || p.Declared != 4 || p.Cases != len(r.Document.Cases) || p.CaseSHA != caseSHA || p.Selection.PlanSHA256 != prepared.PlanSHA256() || p.Selection.MetadataSHA256 != arm.Metadata || p.Selection.WeightsSHA256 != arm.Weights || p.PredictionsThisAdvance != 0 {
			return 0, errors.New("unfixed progress lineage differs")
		}
		p.SHA = sha
		bySHA[sha], prior = p, sha
		attempts = append(attempts, p.NewAttempts...)
		if p.Attempted != len(attempts) || p.Unattempted != 4-len(attempts) {
			return 0, errors.New("unfixed attempt count differs")
		}
	}
	expectedFeedback := 0
	if arm.Feedback {
		expectedFeedback = len(attempts) - 1
	}
	if !reflect.DeepEqual(attempts, c.Search.Attempts) || len(c.Feedback) != expectedFeedback {
		return 0, errors.New("unfixed attempts lost")
	}
	var seen [4]bool
	best := -1
	for _, a := range attempts {
		mask, e := compoundstudy.Mask(r.Document.Plan, a.Choices)
		if e != nil || mask > 3 || mask != a.Mask || seen[mask] || a.Status != "EVALUATED" || a.Total != len(r.Document.Cases) || len(a.Results) != a.Total {
			return 0, errors.New("unfixed candidate invalid/repeated")
		}
		seen[mask] = true
		body, e := prepared.Compile(a.Choices)
		if e != nil || a.GoooSHA != hash([]byte(body.GoooSource())) {
			return 0, errors.New("unfixed typed candidate differs")
		}
		passed := 0
		for i, test := range r.Document.Cases {
			actual, _ := compoundstudy.Oracle(r.Template, mask, test.Input)
			value, e := body.Evaluate(test.Input)
			want := pathplan.TestResult{Input: test.Input, Expected: test.Expected, Actual: actual, Passed: actual == test.Expected}
			if e != nil || value.Int != actual || a.Results[i] != want {
				return 0, errors.New("unfixed candidate oracle differs")
			}
			if want.Passed {
				passed++
			}
		}
		if passed != a.Passed {
			return 0, errors.New("unfixed finite count differs")
		}
		best = max(best, passed)
	}
	prior = ""
	initialCalls := 0
	if arm.Path != "" {
		initialCalls = 2
	}
	calls, skipped := initialCalls, 0
	for i, f := range c.Feedback {
		sha := f.SHA
		f.SHA = ""
		raw, _ := json.Marshal(f)
		from, ok := bySHA[f.FromProgressSHA]
		if !ok || hash(raw) != sha || f.PreviousSHA != prior || f.Round != i+1 || f.Attempted != from.Attempted || f.Cases != from.Cases || f.Passed != from.SelectedPassed || f.PlanSHA != prepared.PlanSHA256() || f.CaseSHA != from.CaseSHA || f.MetadataSHA != arm.Metadata || f.WeightsSHA != arm.Weights || f.Error != "" || f.ContextDeclined || f.CIIsAuthority || f.CI == nil || *f.CI != ci {
			return 0, errors.New("unfixed feedback binding differs")
		}
		var first *pathplan.TestResult
		for _, result := range from.BestCases {
			if !result.Passed {
				copy := result
				first = &copy
				break
			}
		}
		if first == nil || !reflect.DeepEqual(first, f.FirstFailure) {
			return 0, errors.New("unfixed original failure lost")
		}
		if from.Unattempted == 1 {
			if !f.RankingUnnecessary || f.ModelCalls != 0 || len(f.Judgments) != 0 || len(f.FixedCoordinates) != 0 || f.Applied || f.AddedMask {
				return 0, errors.New("unfixed sole-path receipt differs")
			}
		} else {
			var fixed []pathplan.FixedCoordinate
			var tried [4]bool
			for _, a := range attempts[:f.Attempted] {
				tried[a.Mask] = true
			}
			for bit, choice := range r.Document.Plan.Decisions {
				values := 0
				for mask := range 4 {
					if !tried[mask] {
						values |= 1 << (mask >> bit & 1)
					}
				}
				if c.Unfixed && (values == 1 || values == 2) {
					option := values - 1
					fixed = append(fixed, pathplan.FixedCoordinate{DecisionID: choice.ID, Option: option, Selected: choice.Options[option].Label, OtherTried: 2, Remaining: from.Unattempted})
				}
			}
			if !reflect.DeepEqual(fixed, f.FixedCoordinates) || !f.Applied || f.RankingUnnecessary || f.ModelCalls != 2-len(fixed) || len(f.Judgments) != f.ModelCalls {
				return 0, errors.New("unfixed skip/call evidence differs")
			}
			prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d mismatch=%d:%d:%d ci=%s", f.Attempted, f.Passed, f.Cases, f.TypeRejected, from.Unattempted, first.Input, first.Actual, first.Expected, ci.Status)
			j := 0
			for _, choice := range r.Document.Plan.Decisions {
				isFixed := false
				for _, value := range fixed {
					isFixed = isFixed || choice.ID == value.DecisionID
				}
				if isFixed {
					continue
				}
				judgment := f.Judgments[j]
				j++
				input := prefix + " selected=" + from.Selection.Choices[choice.ID] + "\nintent: " + choice.Intent
				if judgment.DecisionID != choice.ID || judgment.Input != input || judgment.InputSHA != hash([]byte(input)) || judgment.Prediction.ID != choice.ID || judgment.Prediction.Kind != choice.Kind || judgment.Prediction.IntentSHA256 != hash([]byte(choice.Intent)) {
					return 0, errors.New("unfixed active input changed")
				}
			}
			skipped += len(fixed)
		}
		calls += f.ModelCalls
		if f.CumulativeCalls != calls {
			return 0, errors.New("unfixed cumulative calls differ")
		}
		prior = sha
	}
	last := c.Progress[len(c.Progress)-1]
	if c.Search.DeclaredCombinations != 4 || c.Search.TrainingTotal != len(r.Document.Cases) || c.Search.TypeRejected != 0 || c.Search.Evaluated != len(attempts) || c.Search.Unattempted != 4-len(attempts) || last.FeedbackRounds != len(c.Feedback) || c.Search.Selection.ModelCalls != calls || last.Selection.ModelCalls != calls || last.FeedbackPredictions != calls-initialCalls || last.LatestFeedbackSHA != prior || last.SelectedPassed != best || c.Search.SelectedTrainingPassed != best || requireBest && best != r.FiniteBestPassed {
		return 0, errors.New("unfixed final completeness/accounting differs")
	}
	body, err := prepared.Compile(c.Search.Selection.Choices)
	if err != nil || body.GoSource() != c.Source {
		return 0, errors.New("unfixed selected Go differs")
	}
	return skipped, nil
}

func runUnfixed(goBinary, output, revision string) error {
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh unfixed output required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || string(head) != revision+"\n" {
		return errors.New("committed runner revision required")
	}
	if err = exec.Command("git", "diff", "--quiet", "HEAD", "--", "internal/pathplan", "tools/native-feedback-study").Run(); err != nil {
		return errors.New("unfixed runtime/runner source dirty")
	}
	baselineAudit, err := auditCompound("runs/compound-path-main-20261001")
	if err != nil {
		return err
	}
	rows, err := compoundRows()
	if err != nil {
		return err
	}
	arms, err := compoundArms()
	if err != nil {
		return err
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	version, err := exec.Command(goBinary, "version").Output()
	if err != nil || !bytes.Contains(version, []byte("go1.27.1 ")) {
		return errors.New("exact Go 1.27.1 executable required")
	}
	if err = os.MkdirAll(filepath.Join(output, "captures"), 0755); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/unfixed-feedback-preexecution/v1", "runner_revision": revision, "baseline_report_sha256": baselineAudit["report_sha256"], "go": "1.27.1", "go_binary_sha256": goSHA, "planned_sdk_sessions": 576, "planned_native_calls": 0, "planned_optimizer_steps": 0, "coordinate_counter_bytes": 64, "arm_order": "per view/model: legacy then unfixed; reused loaded models and prepared plan; not randomized", "entrypoint": "Go SDK methods Reconsider and ReconsiderUnfixed; compiler main stays SDK 2.5"}); err != nil {
		return err
	}
	models := map[string]*decision.Model{}
	for _, arm := range arms {
		if arm.Feedback {
			model, e := decision.LoadPath(arm.Path)
			if e != nil {
				return e
			}
			models[arm.Name] = model
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for _, r := range rows {
		prepared, err := pathplan.Prepare(r.Document.Plan)
		if err != nil {
			return err
		}
		for _, arm := range arms {
			if !arm.Feedback {
				continue
			}
			for _, unfixed := range []bool{false, true} {
				c, failure := captureUnfixed(ctx, prepared, models[arm.Name], r, arm, unfixed)
				id := fmt.Sprintf("%s-%s-%t", r.ID, arm.Name, unfixed)
				if err = save(filepath.Join(output, "captures", id+".json"), c); err != nil {
					return err
				}
				if failure != nil {
					return failure
				}
				if _, err = verifyUnfixed(c, r, arm); err != nil {
					return fmt.Errorf("%s: %w", id, err)
				}
			}
		}
	}
	if err = os.Mkdir(filepath.Join(output, "executions"), 0755); err != nil {
		return err
	}
	// All selected functions from both arms are executed once, with shared
	// recorded values reused for equivalent source. Keep the old input union.
	var baseline struct {
		Inputs []int64 `json:"execution_inputs"`
	}
	raw, _ := read("runs/compound-path-main-20261001/report.json")
	if json.Unmarshal(raw, &baseline) != nil {
		return errors.New("baseline input union differs")
	}
	sources := map[string]string{}
	entries, _ := os.ReadDir(filepath.Join(output, "captures"))
	for _, entry := range entries {
		raw, _ := read(filepath.Join(output, "captures", entry.Name()))
		var c unfixedCapture
		if json.Unmarshal(raw, &c) != nil {
			return errors.New("unfixed capture decode failed")
		}
		sources[hash([]byte(c.Source))] = c.Source
	}
	var keys []string
	for sha := range sources {
		keys = append(keys, sha)
	}
	sort.Strings(keys)
	for _, sha := range keys {
		raw, err := executeFunction(ctx, goBinary, sources[sha], "ComposePaths", baseline.Inputs)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "executions", sha+".json"), raw, 0644); err != nil {
			return err
		}
	}
	result, err := auditUnfixed(output, revision)
	if err != nil {
		return err
	}
	return save(filepath.Join(output, "report.json"), result)
}

func auditUnfixed(dir, revision string) (map[string]any, error) {
	if _, err := auditCompound("runs/compound-path-main-20261001"); err != nil {
		return nil, err
	}
	rows, err := compoundRows()
	if err != nil {
		return nil, err
	}
	arms, err := compoundArms()
	if err != nil {
		return nil, err
	}
	var baseline struct {
		Inputs []int64 `json:"execution_inputs"`
	}
	raw, _ := read("runs/compound-path-main-20261001/report.json")
	if hash(raw) != "dec30461109eafcea587e7d78a533087df21879fb23bb7edc68c5645aefca66e" || json.Unmarshal(raw, &baseline) != nil {
		return nil, errors.New("frozen native baseline differs")
	}
	legacyCalls, optimizedCalls, skipped, pairs, sequenceChanged, finalChanged, nativeMismatch := 0, 0, 0, 0, 0, 0, 0
	var observations []map[string]any
	files := map[string]string{}
	executions := map[string]bool{}
	for _, r := range rows {
		for _, arm := range arms {
			if !arm.Feedback {
				continue
			}
			var previous unfixedCapture
			for _, unfixed := range []bool{false, true} {
				id := fmt.Sprintf("%s-%s-%t", r.ID, arm.Name, unfixed)
				path := "captures/" + id + ".json"
				raw, err := read(filepath.Join(dir, path))
				var c unfixedCapture
				if err != nil || json.Unmarshal(raw, &c) != nil || c.Unfixed != unfixed {
					return nil, errors.New("captured unfixed arm differs")
				}
				files[path] = hash(raw)
				fixed, err := verifyUnfixed(c, r, arm)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", id, err)
				}
				sha := hash([]byte(c.Source))
				execution := "executions/" + sha + ".json"
				raw, err = read(filepath.Join(dir, execution))
				var values []int64
				if err != nil || json.Unmarshal(raw, &values) != nil || len(values) != len(baseline.Inputs) {
					return nil, errors.New("actual Go values missing")
				}
				files[execution], executions[sha] = hash(raw), true
				mask, _ := compoundstudy.Mask(r.Document.Plan, c.Search.Selection.Choices)
				for i, input := range baseline.Inputs {
					want, _ := compoundstudy.Oracle(r.Template, mask, input)
					if values[i] != want {
						return nil, errors.New("actual Go differs from state oracle")
					}
				}
				if !unfixed {
					legacyCalls += c.Search.Selection.ModelCalls
					previous = c
					oldRaw, e := read("runs/compound-path-main-20261001/captures/" + fmt.Sprintf("%s-%s-true.json", r.ID, arm.Name))
					var old nativeResult
					if e != nil || json.Unmarshal(oldRaw, &old) != nil {
						return nil, errors.New("native reference missing")
					}
					actual, _ := compoundFunction(c.Source)
					expected, _ := compoundFunction(old.Source)
					if actual != expected || !reflect.DeepEqual(c.Search.Attempts, old.Report.Paths.Search.Attempts) || c.Search.Selection.ModelCalls != old.Report.Paths.Search.Selection.ModelCalls {
						nativeMismatch++
					}
				} else {
					pairs++
					optimizedCalls += c.Search.Selection.ModelCalls
					skipped += fixed
					if !reflect.DeepEqual(previous.Search.Attempts, c.Search.Attempts) {
						sequenceChanged++
					}
					if previous.Source != c.Source || previous.Search.SelectedTrainingPassed != c.Search.SelectedTrainingPassed {
						finalChanged++
					}
				}
				observations = append(observations, map[string]any{"id": id, "unfixed": unfixed, "model_calls": c.Search.Selection.ModelCalls, "candidate_attempts": len(c.Search.Attempts), "fixed_coordinate_receipts": fixed, "wall_ns": c.WallNS, "finite_passed": c.Search.SelectedTrainingPassed, "finite_cases": len(r.Document.Cases), "selected_mask": mask})
			}
		}
	}
	pre, err := read(filepath.Join(dir, "preexecution.json"))
	var pin struct {
		Revision string `json:"runner_revision"`
		Baseline string `json:"baseline_report_sha256"`
	}
	if err != nil || json.Unmarshal(pre, &pin) != nil || pin.Revision != revision || pin.Baseline != "dec30461109eafcea587e7d78a533087df21879fb23bb7edc68c5645aefca66e" {
		return nil, errors.New("unfixed preexecution binding differs")
	}
	return map[string]any{"schema": "gooo/unfixed-feedback-study/v1", "decision": "PASS", "runner_revision": revision, "preexecution_sha256": hash(pre), "files_sha256": files, "observations": observations, "actual_sdk_sessions": pairs * 2, "actual_local_model_predictions": legacyCalls + optimizedCalls, "legacy_predictions": legacyCalls, "unfixed_predictions": optimizedCalls, "observed_prediction_reduction": legacyCalls - optimizedCalls, "fixed_coordinate_prediction_skips": skipped, "paired_views": pairs, "changed_candidate_sequences": sequenceChanged, "changed_final_source_or_finite_results": finalChanged, "legacy_sdk_native_reference_mismatches": nativeMismatch, "actual_generated_go_processes": len(executions), "actual_generated_function_evaluations": len(executions) * len(baseline.Inputs), "new_native_calls": 0, "new_optimizer_steps": 0, "new_upstream_laya_calls": 0, "scope": "Actual Go SDK sessions and own frozen tiny models, legacy-first fixed order with shared prepared plans/models. New opt-in skip API; native main still SDK 2.5 and uses the default API. The 72 bilingual/contract views are reused development data, not independent tasks. Digest/call/constant-coordinate proofs and independent integer-state oracle validated; audit mode performs no predictions or subprocess. Removed common log factors can change floating-point near ties on other data; no all-input ordering equivalence or causal timing/host CPU improvement is claimed."}, nil
}
