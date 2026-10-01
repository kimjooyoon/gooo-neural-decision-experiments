// audit-feedback-pilot checks captured finite evidence without new predictions.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func hash(raw []byte) string { v := sha256.Sum256(raw); return hex.EncodeToString(v[:]) }
func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 4<<20 {
		return nil, errors.New("bounded regular evidence required")
	}
	return os.ReadFile(path)
}
func decode(path string, value any) ([]byte, error) {
	raw, err := read(path)
	if err != nil {
		return nil, err
	}
	return raw, json.Unmarshal(raw, value)
}

type policy struct {
	ID           string                     `json:"id"`
	Arm          string                     `json:"arm"`
	Language     string                     `json:"language"`
	Contract     string                     `json:"contract"`
	Enabled      bool                       `json:"feedback_enabled"`
	WallNS       int64                      `json:"search_wall_ns"`
	Progress     []pathplan.SessionProgress `json:"progress"`
	Feedback     []pathplan.FeedbackReceipt `json:"feedback"`
	NativeSHA    string                     `json:"native_capture_sha256"`
	GoSHA        string                     `json:"generated_go_sha256"`
	ReplaySHA    string                     `json:"executed_go_capture_sha256"`
	FinitePassed int                        `json:"executed_finite_passed"`
	FiniteTotal  int                        `json:"executed_finite_cases"`
	EvalPassed   int                        `json:"executed_disjoint_input_passed"`
	EvalTotal    int                        `json:"executed_disjoint_input_cases"`
}

var metadataPins = map[string]string{
	"fp32":        "1ea3bada068f2487418f17270db4ba6785a98c3ad60e0ad7eb92359400a01bb5",
	"ptq_ternary": "7c4eb4068d76e83620a9b8e7ce7f8d948b5e2436d44c7b79cf241da609dab894",
	"qat_ternary": "368e37e7899cecba03874b69e933525a2f8c90ecc53d787d8e698de6aa21c087",
}

func median(values []int64) float64 {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	n := len(values)
	if n%2 == 1 {
		return float64(values[n/2])
	}
	return float64(values[n/2-1]+values[n/2]) / 2
}
func audit(dir string) (map[string]any, error) {
	var report struct {
		Schema      string   `json:"schema"`
		Revision    string   `json:"runner_revision"`
		NativeCalls int      `json:"native_calls"`
		GoCalls     int      `json:"unique_generated_go_input_executions"`
		Policies    []policy `json:"policies"`
	}
	reportRaw, err := decode(filepath.Join(dir, "report.json"), &report)
	if err != nil {
		return nil, err
	}
	if report.Schema != "gooo/feedback-path-pilot/v1" || report.Revision != "63c9967505355ccc5c56780f901f4ce583008f17" || len(report.Policies) != 32 || report.NativeCalls != 32 || report.GoCalls != 2 {
		return nil, errors.New("fixed pilot shape differs")
	}
	var evaluation []pathplan.TestCase
	if _, err = decode("studies/conditional-paths-v1/cohort/holdout-cases.json", &evaluation); err != nil {
		return nil, err
	}
	if len(evaluation) != 9 {
		return nil, errors.New("evaluation denominator differs")
	}
	byPair := map[string][]policy{}
	unique := map[string]bool{}
	actualCalls, feedbackCalls, rounds, totalAttempts, finitePassed, evalPassed, checkedCandidates := 0, 0, 0, 0, 0, 0, 0
	baseTimes, feedbackTimes := []int64{}, []int64{}
	for _, p := range report.Policies {
		if p.Arm != "offline" && p.Arm != "fp32" && p.Arm != "ptq_ternary" && p.Arm != "qat_ternary" {
			return nil, errors.New("unknown model arm")
		}
		if p.ID != p.Language+"-"+p.Contract+"-"+p.Arm+"-"+strconv.FormatBool(p.Enabled) {
			return nil, errors.New("policy identity differs")
		}
		if unique[p.ID] || len(p.Progress) < 2 || p.FiniteTotal != 7 || p.EvalTotal != 9 {
			return nil, errors.New("duplicate policy or finite denominators")
		}
		unique[p.ID] = true
		var doc struct {
			Plan  pathplan.Plan       `json:"path_plan"`
			Cases []pathplan.TestCase `json:"test_cases"`
		}
		if p.Language != "en" && p.Language != "ko" {
			return nil, errors.New("language outside fixed pilot")
		}
		if _, err = decode("studies/conditional-paths-v1/cohort/"+p.Language+"-budget-64.json", &doc); err != nil {
			return nil, err
		}
		if len(doc.Cases) != 7 || len(doc.Plan.Decisions) != 6 {
			return nil, errors.New("fixed plan bounds differ")
		}
		if p.Contract == "inconsistent" {
			doc.Cases[6].Expected = 999
		} else if p.Contract != "complete" {
			return nil, errors.New("unknown contract")
		}
		caseRaw, _ := json.Marshal(doc.Cases)
		previous := ""
		seen := map[uint16]bool{}
		progressBySHA := map[string]pathplan.SessionProgress{}
		for i, progress := range p.Progress {
			copy := progress
			copy.SHA = ""
			raw, _ := json.Marshal(copy)
			if hash(raw) != progress.SHA || progress.PreviousSHA != previous || progress.Sequence != i+1 || progress.CaseSHA != hash(caseRaw) || progress.Cases != 7 || progress.PredictionsThisAdvance != 0 {
				return nil, errors.New("broken progress hash, fixed cases or call accounting")
			}
			previous = progress.SHA
			progressBySHA[progress.SHA] = progress
			for _, attempt := range progress.NewAttempts {
				if seen[attempt.Mask] || attempt.Mask >= 64 || attempt.Total != 7 {
					return nil, errors.New("attempt mask or cases differ")
				}
				seen[attempt.Mask] = true
				body, compileErr := pathplan.Compile(doc.Plan, attempt.Choices)
				if compileErr != nil {
					if attempt.Status != "TYPE_REJECTED" {
						return nil, errors.New("hidden type failure")
					}
					continue
				}
				checkedCandidates++
				if attempt.Status != "EVALUATED" || len(attempt.Results) != 7 || hash([]byte(body.GoooSource())) != attempt.GoooSHA {
					return nil, errors.New("candidate source or finite result shape differs")
				}
				passed := 0
				for j, test := range doc.Cases {
					value, err := body.Evaluate(test.Input)
					if err != nil {
						return nil, err
					}
					result := pathplan.TestResult{Input: test.Input, Expected: test.Expected, Actual: value.Int, Passed: value.Int == test.Expected}
					if result.Passed {
						passed++
					}
					if result != attempt.Results[j] {
						return nil, errors.New("candidate finite interpretation differs")
					}
				}
				if passed != attempt.Passed {
					return nil, errors.New("candidate finite numerator differs")
				}
			}
			if progress.Attempted != len(seen) || progress.Unattempted != 64-len(seen) {
				return nil, errors.New("cumulative denominator differs")
			}
		}
		previousFeedback := ""
		calls := 0
		for i, f := range p.Feedback {
			copy := f
			copy.SHA = ""
			raw, _ := json.Marshal(copy)
			prior, ok := progressBySHA[f.FromProgressSHA]
			if !ok || hash(raw) != f.SHA || f.PreviousSHA != previousFeedback || f.Round != i+1 || f.CIIsAuthority || f.CI == nil || f.CI.Status != "PASS" || f.CI.SourceSHA != "307159f041644a3aa56dfd325c345f5325aec902" || !f.Applied || f.Error != "" || f.Attempted != prior.Attempted || f.Passed != prior.SelectedPassed || f.CaseSHA != prior.CaseSHA || f.ModelCalls != 6 || len(f.Judgments) != 6 {
				return nil, errors.New("feedback lineage or observed calls differ")
			}
			for j, judgment := range f.Judgments {
				choice := doc.Plan.Decisions[j]
				prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d", prior.Attempted, prior.SelectedPassed, prior.Cases, prior.TypeRejected, prior.Unattempted)
				var firstFailure *pathplan.TestResult
				for _, result := range prior.BestCases {
					if !result.Passed {
						copy := result
						firstFailure = &copy
						break
					}
				}
				if !reflect.DeepEqual(firstFailure, f.FirstFailure) {
					return nil, errors.New("feedback failure differs from prior selected observation")
				}
				if firstFailure != nil {
					prefix += fmt.Sprintf(" mismatch=%d:%d:%d", firstFailure.Input, firstFailure.Actual, firstFailure.Expected)
				}
				expectedInput := prefix + " ci=PASS selected=" + prior.Selection.Choices[choice.ID] + "\nintent: " + choice.Intent
				if judgment.DecisionID != choice.ID || judgment.Input != expectedInput || judgment.InputSHA != hash([]byte(judgment.Input)) || judgment.Prediction.IntentSHA256 != hash([]byte(choice.Intent)) || len(judgment.Input) > 512 || judgment.Eligible[0] < 0 || judgment.Eligible[0] > 1 || judgment.Eligible[1] < 0 || judgment.Eligible[1] > 1 || math.Abs(judgment.Eligible[0]+judgment.Eligible[1]-1) > 1e-8 {
					return nil, errors.New("feedback input or eligible pair differs")
				}
			}
			calls += f.ModelCalls
			previousFeedback = f.SHA
			rounds++
		}
		last := p.Progress[len(p.Progress)-1]
		if p.Arm != "offline" && last.Selection.MetadataSHA256 != metadataPins[p.Arm] {
			return nil, errors.New("frozen model identity differs")
		}
		for _, f := range p.Feedback {
			if f.MetadataSHA != last.Selection.MetadataSHA256 || f.WeightsSHA != last.Selection.WeightsSHA256 {
				return nil, errors.New("feedback model changed")
			}
		}
		initialCalls := 6
		if p.Arm == "offline" {
			initialCalls = 0
		}
		if last.Selection.ModelCalls != initialCalls+calls || last.FeedbackPredictions != calls || last.FeedbackRounds != len(p.Feedback) || last.LatestFeedbackSHA != previousFeedback || last.SelectedPassed != p.FinitePassed || last.Selection.ExternalCalls != 0 {
			return nil, errors.New("final prediction or finite accounting differs")
		}
		fixtureRaw, err := read(filepath.Join(dir, "captures", p.ID+".gooo.fixture"))
		if err != nil {
			return nil, err
		}
		selected, err := pathplan.Compile(doc.Plan, last.Selection.Choices)
		if err != nil || !reflect.DeepEqual(fixtureRaw, []byte(selected.GoooSource())) {
			return nil, errors.New("native fixture differs from selected typed body")
		}
		var native struct {
			Source string `json:"source"`
			Report struct {
				CompilerSHA string `json:"compiler_source_sha"`
				Decision    string `json:"decision"`
				Typecheck   bool   `json:"typecheck_passed"`
				Replay      bool   `json:"deterministic_replay"`
				Writes      int    `json:"repository_writes"`
			} `json:"report"`
		}
		nativeRaw, err := decode(filepath.Join(dir, "captures", p.ID+".native.json"), &native)
		if err != nil || hash(nativeRaw) != p.NativeSHA || hash([]byte(native.Source)) != p.GoSHA || native.Report.CompilerSHA != "307159f041644a3aa56dfd325c345f5325aec902" || native.Report.Decision != "PASS" || !native.Report.Typecheck || !native.Report.Replay || native.Report.Writes != 0 {
			return nil, errors.New("native capture source/type/replay differs")
		}
		var actual []int64
		replayRaw, err := decode(filepath.Join(dir, "captures", p.ID+".executed.json"), &actual)
		if err != nil || hash(replayRaw) != p.ReplaySHA || len(actual) != 16 {
			return nil, errors.New("generated Go execution capture differs")
		}
		passed, evaluated := 0, 0
		for i, test := range doc.Cases {
			if actual[i] == test.Expected {
				passed++
			}
		}
		for i, test := range evaluation {
			if actual[7+i] == test.Expected {
				evaluated++
			}
		}
		if passed != p.FinitePassed || evaluated != p.EvalPassed {
			return nil, errors.New("executed Go numerator differs")
		}
		actualCalls += last.Selection.ModelCalls
		feedbackCalls += calls
		totalAttempts += last.Attempted
		finitePassed += passed
		evalPassed += evaluated
		byPair[p.Language+"/"+p.Contract+"/"+p.Arm] = append(byPair[p.Language+"/"+p.Contract+"/"+p.Arm], p)
		if p.Arm != "offline" {
			if p.Enabled {
				feedbackTimes = append(feedbackTimes, p.WallNS)
			} else {
				baseTimes = append(baseTimes, p.WallNS)
			}
		}
	}
	sameGo, sameFunctional, improved, worsened, unchanged := 0, 0, 0, 0, 0
	for _, pair := range byPair {
		if len(pair) != 2 || pair[0].Enabled || !pair[1].Enabled {
			return nil, errors.New("paired controls differ")
		}
		if pair[0].GoSHA == pair[1].GoSHA {
			sameGo++
		}
		if pair[0].FinitePassed == pair[1].FinitePassed && pair[0].EvalPassed == pair[1].EvalPassed {
			sameFunctional++
		}
		a, b := pair[0].Progress[len(pair[0].Progress)-1].Attempted, pair[1].Progress[len(pair[1].Progress)-1].Attempted
		if b < a {
			improved++
		} else if b > a {
			worsened++
		} else {
			unchanged++
		}
	}
	return map[string]any{"schema": "gooo/feedback-path-pilot-audit/v1", "decision": "PASS", "report_sha256": hash(reportRaw), "policies": 32, "pairs": len(byPair), "same_emitted_go_pairs": sameGo, "same_finite_and_evaluation_pairs": sameFunctional, "candidate_count_reduced_pairs": improved, "candidate_count_increased_pairs": worsened, "candidate_count_unchanged_pairs": unchanged, "actual_model_predictions": actualCalls, "feedback_model_predictions": feedbackCalls, "feedback_rounds": rounds, "candidate_attempts": totalAttempts, "independently_interpreted_candidate_bodies": checkedCandidates, "executed_finite_passed": finitePassed, "executed_finite_cases": 224, "executed_disjoint_input_passed": evalPassed, "executed_disjoint_input_observations": 288, "distinct_evaluation_inputs": 9, "native_calls": report.NativeCalls, "unique_generated_go_input_executions": report.GoCalls, "new_predictions_by_audit": 0, "new_native_calls_by_audit": 0, "model_policy_median_base_search_ms": median(baseTimes) / 1e6, "model_policy_median_feedback_search_ms": median(feedbackTimes) / 1e6, "cpu_or_rss_measured": false, "scope": "Fixed one-intent pilot, one baseline-first ordering and no repetitions. Finite selection cases and existing disjoint function inputs are separate; repeated observations are not 288 independent tasks. No generic NL improvement or causal speedup claim. Audit re-interprets candidates and hashes captured native/Go outputs; it does not re-execute native or generated Go or retrain the model."}, nil
}
func main() {
	dir := flag.String("study", "runs/feedback-path-pilot-fixed-20261001", "fixed local pilot")
	output := flag.String("output", "", "audit JSON output")
	flag.Parse()
	value, err := audit(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, _ := json.MarshalIndent(value, "", "  ")
	raw = append(raw, '\n')
	if *output != "" {
		if err = os.WriteFile(*output, raw, 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		os.Stdout.Write(raw)
	}
}
