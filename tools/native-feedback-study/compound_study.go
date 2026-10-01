package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const compoundNative = "ef63060ed1aebd9d92a2fe4cf24ce9c5929b8726"
const compoundRoot = "studies/compound-path-v1"

type compoundObservation struct {
	ID                  string  `json:"id"`
	CaseID              string  `json:"case_id"`
	Template            string  `json:"template"`
	Arm                 string  `json:"arm"`
	Feedback            bool    `json:"feedback_enabled"`
	Capture             string  `json:"capture_sha256"`
	GoSHA               string  `json:"generated_go_sha256"`
	Selected            uint16  `json:"selected_mask"`
	IntentAgreement     bool    `json:"original_intention_mask_agreement"`
	FiniteBest          bool    `json:"finite_best_set_selected"`
	Passed              int     `json:"finite_passed"`
	Cases               int     `json:"finite_cases"`
	SeparatePassed      int     `json:"separate_cases_passed"`
	SeparateCases       int     `json:"separate_cases"`
	Attempts            int     `json:"candidate_attempts"`
	Predictions         int     `json:"actual_model_predictions"`
	FeedbackPredictions int     `json:"feedback_predictions"`
	NoChoice            int     `json:"zero_call_ranking_unnecessary_receipts"`
	ChangedJudgments    int     `json:"feedback_judgments_different_from_initial_proposal"`
	Metrics             metrics `json:"process_metrics"`
}

func writeCompoundCohort(dir string) error {
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		return errors.New("fresh compound cohort directory required")
	}
	rows, err := compoundstudy.Cohort()
	if err != nil {
		return err
	}
	var raw bytes.Buffer
	encoder := json.NewEncoder(&raw)
	ambiguous := 0
	for _, r := range rows {
		if err = encoder.Encode(r); err != nil {
			return err
		}
		if len(r.FiniteBestMasks) > 1 {
			ambiguous++
		}
	}
	sources := map[string]string{}
	for _, p := range []string{"internal/compoundstudy/cohort.go", "internal/compoundstudy/cohort_test.go"} {
		data, err := read(p)
		if err != nil {
			return err
		}
		sources[p] = hash(data)
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "cohort.jsonl"), raw.Bytes(), 0644); err != nil {
		return err
	}
	return save(filepath.Join(dir, "manifest.json"), map[string]any{"schema": "gooo/compound-path-cohort/v1", "cohort_sha256": hash(raw.Bytes()), "source_sha256": sources, "views": 72, "templates": compoundstudy.Templates, "configuration": compoundstudy.Configuration, "source_bodies": 3, "declared_masks_per_body": 4, "structural_intention_groups": 12, "bilingual_program_instructions": 24, "ambiguous_finite_views": ambiguous, "languages": []string{"en", "ko"}, "contracts": []string{"complete", "sparse", "contradictory"}, "planned_native_matrix_calls": 648, "planned_pilot_calls": 3, "scope": "Three newly authored interacting body templates using five existing binary decision kinds. Two decisions yield four whole-body paths; each decision still has two options. One synthetic numeric configuration. Views/languages/contracts/arms are repeated policy observations, not 72 independent experiments or an untouched language benchmark. reference_schedule masks 1 and 3 commute and are functionally equivalent even when structural intention differs."})
}
func compoundRows() ([]compoundstudy.Case, error) {
	raw, err := read(compoundRoot + "/cohort.jsonl")
	if err != nil {
		return nil, err
	}
	var rows []compoundstudy.Case
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var r compoundstudy.Case
		if err = json.Unmarshal(scanner.Bytes(), &r); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	fresh, err := compoundstudy.Cohort()
	if err != nil || !reflect.DeepEqual(rows, fresh) {
		return nil, errors.New("authored compound cohort differs")
	}
	manifest, err := read(compoundRoot + "/manifest.json")
	if err != nil {
		return nil, err
	}
	var pin struct {
		SHA string `json:"cohort_sha256"`
	}
	if json.Unmarshal(manifest, &pin) != nil || pin.SHA != hash(raw) {
		return nil, errors.New("compound cohort digest differs")
	}
	return rows, nil
}

func inspectCompound(v nativeResult, r compoundstudy.Case, a familyArm) (compoundObservation, error) {
	return inspectCompoundFor(v, r, a, compoundNative)
}

func inspectCompoundFor(v nativeResult, r compoundstudy.Case, a familyArm, native string) (compoundObservation, error) {
	var result compoundObservation
	p := v.Report.Paths
	if v.Report.Decision != "PASS" || !v.Report.Types || !v.Report.Replay || v.Report.Writes != 0 || v.Report.Compiler != native || !p.Bound || v.Report.ActivityID != "compound-study://activity/compose-paths" || v.Source == "" || len(p.Cases) != len(r.Document.Cases) || p.Search.TrainingTotal != len(r.Document.Cases) || p.Search.DeclaredCombinations != 4 || len(p.Search.Attempts) < 1 || len(p.Search.Attempts) > 4 || p.Search.TypeRejected != 0 || p.Search.Selection.ExternalCalls != 0 || !p.Search.Selection.ExternalCallsKnown {
		return result, errors.New("compound native verification differs")
	}
	prepared, err := pathplan.Prepare(r.Document.Plan)
	if err != nil {
		return result, err
	}
	if p.Search.Selection.PlanSHA256 != prepared.PlanSHA256() || p.Search.Selection.MetadataSHA256 != a.Metadata || p.Search.Selection.WeightsSHA256 != a.Weights {
		return result, errors.New("compound source/model binding differs")
	}
	seen := map[uint16]bool{}
	for _, attempt := range p.Search.Attempts {
		mask, err := compoundstudy.Mask(r.Document.Plan, attempt.Choices)
		if err != nil || mask != attempt.Mask || mask > 3 || seen[mask] || attempt.Status != "EVALUATED" || attempt.Total != len(r.Document.Cases) || len(attempt.Results) != attempt.Total {
			return result, errors.New("invalid or repeated compound candidate")
		}
		seen[mask] = true
		body, err := prepared.Compile(attempt.Choices)
		if err != nil || attempt.GoooSHA != hash([]byte(body.GoooSource())) {
			return result, errors.New("candidate typed body differs")
		}
		passed := 0
		for i, c := range r.Document.Cases {
			expected, _ := compoundstudy.Oracle(r.Template, mask, c.Input)
			actual, e := body.Evaluate(c.Input)
			observed := attempt.Results[i]
			if e != nil || actual.Int != expected || observed.Input != c.Input || observed.Expected != c.Expected || observed.Actual != expected || observed.Passed != (expected == c.Expected) {
				return result, errors.New("candidate finite/oracle outcome differs")
			}
			if observed.Passed {
				passed++
			}
		}
		if passed != attempt.Passed {
			return result, errors.New("candidate finite count differs")
		}
	}
	var recorded []pathplan.SearchAttempt
	progresses := map[string]pathplan.SessionProgress{}
	prior := ""
	for i, progress := range p.Progress {
		sha := progress.SHA
		progress.SHA = ""
		raw, err := json.Marshal(progress)
		if err != nil || hash(raw) != sha || progress.Sequence != i+1 || progress.PreviousSHA != prior || progress.Interrupted || progress.PredictionsThisAdvance != 0 || progress.Declared != 4 || progress.Cases != len(r.Document.Cases) || progress.Selection.PlanSHA256 != prepared.PlanSHA256() {
			return result, errors.New("compound progress lineage differs")
		}
		progress.SHA = sha
		progresses[sha] = progress
		prior = sha
		recorded = append(recorded, progress.NewAttempts...)
		if progress.Attempted != len(recorded) || progress.Unattempted != 4-len(recorded) {
			return result, errors.New("progress attempt denominator differs")
		}
	}
	if !reflect.DeepEqual(recorded, p.Search.Attempts) || len(p.Progress) == 0 || p.Search.Evaluated != len(recorded) {
		return result, errors.New("progress lost compound candidates")
	}
	initial := 0
	if a.Path != "" {
		initial = 2
	}
	feedbackCalls, noChoice, changed := 0, 0, 0
	prior = ""
	for i, f := range p.Feedback {
		sha := f.SHA
		f.SHA = ""
		raw, err := json.Marshal(f)
		from, ok := progresses[f.FromProgressSHA]
		if err != nil || hash(raw) != sha || !ok || !a.Feedback || f.PreviousSHA != prior || f.Round != i+1 || f.Attempted != from.Attempted || f.Passed != from.SelectedPassed || f.Cases != from.Cases || f.TypeRejected != from.TypeRejected || f.CaseSHA != from.CaseSHA || f.PlanSHA != prepared.PlanSHA256() || f.MetadataSHA != a.Metadata || f.WeightsSHA != a.Weights || f.CIIsAuthority || f.CI == nil || *f.CI != (pathplan.CIHint{SourceSHA: native, Status: "PASS"}) || f.ContextDeclined || f.Error != "" {
			return result, errors.New("compound feedback lineage differs")
		}
		var first *pathplan.TestResult
		for _, c := range from.BestCases {
			if !c.Passed {
				copy := c
				first = &copy
				break
			}
		}
		if first == nil || !reflect.DeepEqual(first, f.FirstFailure) {
			return result, errors.New("feedback lost original failure")
		}
		if from.Unattempted == 1 {
			if !f.RankingUnnecessary || f.ModelCalls != 0 || f.Applied || f.AddedMask || len(f.Judgments) != 0 {
				return result, errors.New("sole compound path re-ranked")
			}
			noChoice++
		} else {
			if f.RankingUnnecessary || !f.Applied || f.ModelCalls != 2 || len(f.Judgments) != 2 {
				return result, errors.New("multi-path feedback predictions differ")
			}
			prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d mismatch=%d:%d:%d ci=PASS", f.Attempted, f.Passed, f.Cases, f.TypeRejected, from.Unattempted, first.Input, first.Actual, first.Expected)
			for j, judgment := range f.Judgments {
				choice := r.Document.Plan.Decisions[j]
				input := prefix + " selected=" + from.Selection.Choices[choice.ID] + "\nintent: " + choice.Intent
				if judgment.DecisionID != choice.ID || judgment.Input != input || judgment.InputSHA != hash([]byte(input)) || judgment.Prediction.IntentSHA256 != hash([]byte(choice.Intent)) || judgment.Prediction.ID != choice.ID || judgment.Prediction.Kind != choice.Kind {
					return result, errors.New("feedback altered original intention or observed context")
				}
				if judgment.Prediction.Selected != p.Search.InitialProposals[choice.ID] {
					changed++
				}
			}
			feedbackCalls += 2
		}
		if f.CumulativeCalls != initial+feedbackCalls {
			return result, errors.New("feedback cumulative calls differ")
		}
		prior = sha
	}
	if a.Feedback && len(p.Feedback) != len(recorded)-1 || !a.Feedback && len(p.Feedback) != 0 || p.Search.Selection.ModelCalls != initial+feedbackCalls {
		return result, errors.New("compound call accounting differs")
	}
	last := p.Progress[len(p.Progress)-1]
	if last.FeedbackRounds != len(p.Feedback) || last.FeedbackPredictions != feedbackCalls || last.LatestFeedbackSHA != prior || last.Selection.ModelCalls != initial+feedbackCalls {
		return result, errors.New("final compound feedback accounting differs")
	}
	mask, err := compoundstudy.Mask(r.Document.Plan, p.Search.Selection.Choices)
	if err != nil {
		return result, err
	}
	compiled, err := prepared.Compile(p.Search.Selection.Choices)
	if err != nil {
		return result, err
	}
	actualFunction, err := compoundFunction(v.Source)
	if err != nil {
		return result, err
	}
	expectedFunction, err := compoundFunction(compiled.GoSource())
	if err != nil || actualFunction != expectedFunction {
		return result, errors.New("emitted function differs from selected typed body")
	}
	passed, separate := 0, 0
	for i, c := range r.Document.Cases {
		expected, _ := compoundstudy.Oracle(r.Template, mask, c.Input)
		observed := p.Cases[i]
		if observed.Input != c.Input || observed.Expected != c.Expected || observed.Actual != expected || observed.Passed != (expected == c.Expected) {
			return result, errors.New("native selected outcome differs")
		}
		if observed.Passed {
			passed++
		}
	}
	for _, c := range r.Separate {
		actual, _ := compoundstudy.Oracle(r.Template, mask, c.Input)
		if actual == c.Expected {
			separate++
		}
	}
	best := false
	for _, m := range r.FiniteBestMasks {
		best = best || m == mask
	}
	if !best || passed != r.FiniteBestPassed || passed != p.Search.SelectedTrainingPassed || p.Completeness != 100*float64(passed)/float64(len(r.Document.Cases)) {
		return result, errors.New("finite construction did not reach declared best completion")
	}
	return compoundObservation{CaseID: r.ID, Template: r.Template, Arm: a.Name, Feedback: a.Feedback, GoSHA: hash([]byte(v.Source)), Selected: mask, IntentAgreement: mask == r.IntendedMask, FiniteBest: best, Passed: passed, Cases: len(r.Document.Cases), SeparatePassed: separate, SeparateCases: len(r.Separate), Attempts: len(recorded), Predictions: initial + feedbackCalls, FeedbackPredictions: feedbackCalls, NoChoice: noChoice, ChangedJudgments: changed}, nil
}

// Bind signature and statements modulo in-range int64 literal wrappers. The SDK
// emits int64(4), while the native compiler emits 4 in an int64 expression.
func compoundFunction(source string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "projection.go", source, parser.SkipObjectResolution)
	if err != nil || len(file.Decls) != 1 {
		return "", errors.New("one declared compound function required")
	}
	function, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || function.Name.Name != "ComposePaths" || function.Recv != nil || function.Body == nil {
		return "", errors.New("declared compound function signature required")
	}
	normalizeCompoundBlock(function.Body)
	var signature, body bytes.Buffer
	if err = format.Node(&signature, fset, function.Type); err != nil {
		return "", err
	}
	if err = format.Node(&body, fset, function.Body); err != nil {
		return "", err
	}
	return signature.String() + body.String(), nil
}

func normalizeCompoundExpression(expression ast.Expr) ast.Expr {
	switch value := expression.(type) {
	case *ast.CallExpr:
		name, ok := value.Fun.(*ast.Ident)
		if ok && name.Name == "int64" && len(value.Args) == 1 && value.Ellipsis == token.NoPos {
			literal, ok := value.Args[0].(*ast.BasicLit)
			if ok && literal.Kind == token.INT {
				if _, err := strconv.ParseInt(literal.Value, 0, 64); err == nil {
					return literal
				}
			}
		}
	case *ast.BinaryExpr:
		value.X, value.Y = normalizeCompoundExpression(value.X), normalizeCompoundExpression(value.Y)
	case *ast.ParenExpr:
		value.X = normalizeCompoundExpression(value.X)
	case *ast.UnaryExpr:
		value.X = normalizeCompoundExpression(value.X)
	}
	return expression
}

func normalizeCompoundBlock(block *ast.BlockStmt) {
	for _, statement := range block.List {
		switch value := statement.(type) {
		case *ast.DeclStmt:
			declaration, ok := value.Decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range declaration.Specs {
				if binding, ok := spec.(*ast.ValueSpec); ok {
					for i, expression := range binding.Values {
						binding.Values[i] = normalizeCompoundExpression(expression)
					}
				}
			}
		case *ast.AssignStmt:
			for i, expression := range value.Rhs {
				value.Rhs[i] = normalizeCompoundExpression(expression)
			}
		case *ast.ReturnStmt:
			for i, expression := range value.Results {
				value.Results[i] = normalizeCompoundExpression(expression)
			}
		case *ast.IfStmt:
			value.Cond = normalizeCompoundExpression(value.Cond)
			normalizeCompoundBlock(value.Body)
			if alternative, ok := value.Else.(*ast.BlockStmt); ok {
				normalizeCompoundBlock(alternative)
			}
		}
	}
}

func runCompound(binary, goBinary, output, revision string, pilot bool) error {
	_, arms, err := familyPreflightFor(binary, goBinary, output, revision, familySpec{Revision: compoundNative, SDK: "v0.2.5-experimental", NoChoice: true, CIStatus: "PASS"})
	if err != nil {
		return err
	}
	paths := []string{"tools/native-feedback-study", "internal/compoundstudy", compoundRoot}
	if err = exec.Command("git", append([]string{"diff", "--quiet", "HEAD", "--"}, paths...)...).Run(); err != nil {
		return errors.New("compound source dirty")
	}
	untracked, err := exec.Command("git", append([]string{"ls-files", "--others", "--exclude-standard", "--"}, paths...)...).Output()
	if err != nil || len(untracked) != 0 {
		return errors.New("compound source untracked")
	}
	rows, err := compoundRows()
	if err != nil {
		return err
	}
	if pilot {
		subset := rows[:0:0]
		for _, r := range rows {
			if r.Language == "en" && r.Contract == "complete" && r.IntendedMask == 0 {
				subset = append(subset, r)
			}
		}
		rows = subset
		arms = arms[:1]
	}
	cohort, err := read(compoundRoot + "/cohort.jsonl")
	if err != nil {
		return err
	}
	binarySHA, err := executableHash(binary)
	if err != nil {
		return err
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	pins := map[string]map[string]string{}
	for _, a := range arms {
		if a.Path != "" {
			pins[a.Name] = map[string]string{"metadata": a.Metadata, "weights": a.Weights}
		}
	}
	inputsSet := map[int64]bool{}
	for _, r := range rows {
		for _, c := range append(append([]pathplan.TestCase(nil), r.Document.Cases...), r.Separate...) {
			inputsSet[c.Input] = true
		}
	}
	var inputs []int64
	for x := range inputsSet {
		inputs = append(inputs, x)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i] < inputs[j] })
	if err = os.MkdirAll(filepath.Join(output, "captures"), 0755); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/compound-path-preexecution/v1", "runner_revision": revision, "native_revision": compoundNative, "binary_sha256": binarySHA, "go_binary_sha256": goSHA, "go": "1.27.1", "sdk": "v0.2.5-experimental", "pilot": pilot, "cohort_sha256": hash(cohort), "models": pins, "planned_native_calls": len(rows) * len(arms), "caller_ci_hint": pathplan.CIHint{SourceSHA: compoundNative, Status: "PASS"}, "caller_ci_hint_is_authority": false, "maximum_attempts": 4, "step_attempts": 1, "feedback_round_budget": 3, "training_steps": 0}); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "gooo-compound-study-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err = save(filepath.Join(dir, "ci.json"), pathplan.CIHint{SourceSHA: compoundNative, Status: "PASS"}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var observations []compoundObservation
	generated := map[string]string{}
	for _, r := range rows {
		if err = os.WriteFile(filepath.Join(dir, "source.gooo"), []byte(r.Source), 0600); err != nil {
			return err
		}
		if err = save(filepath.Join(dir, "plan.json"), r.Document); err != nil {
			return err
		}
		for _, a := range arms {
			id := fmt.Sprintf("%s-%s-%t", r.ID, a.Name, a.Feedback)
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ComposePaths"}
			if a.Path != "" {
				args = append(args, "--path-model", a.Path)
			}
			if a.Feedback {
				args = append(args, "--path-feedback-rounds", "3", "--path-feedback-ci", "ci.json")
			}
			args = append(args, "source.gooo")
			raw, m, callErr := child(ctx, dir, binary, args...)
			if err = os.WriteFile(filepath.Join(output, "captures", id+".json"), raw, 0644); err != nil {
				return err
			}
			if callErr != nil {
				return fmt.Errorf("compound native failure retained at %s", id)
			}
			var v nativeResult
			if err = json.Unmarshal(raw, &v); err != nil {
				return err
			}
			o, err := inspectCompound(v, r, a)
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
	var sorted []string
	for sha := range generated {
		sorted = append(sorted, sha)
	}
	sort.Strings(sorted)
	for _, sha := range sorted {
		raw, err := executeFunction(ctx, goBinary, generated[sha], "ComposePaths", inputs)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "executions", sha+".json"), raw, 0644); err != nil {
			return err
		}
		var values []int64
		if json.Unmarshal(raw, &values) != nil || len(values) != len(inputs) {
			return errors.New("actual compound Go execution shape differs")
		}
		executions[sha] = values
	}
	byID := map[string]compoundstudy.Case{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	predictions, feedback, attempts, passed, total, separatePassed, separateTotal, noChoice, changed, intent := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	for _, o := range observations {
		r := byID[o.CaseID]
		for i, x := range inputs {
			expected, _ := compoundstudy.Oracle(r.Template, o.Selected, x)
			if executions[o.GoSHA][i] != expected {
				return errors.New("actual emitted Go differs from independent compound oracle")
			}
		}
		predictions += o.Predictions
		feedback += o.FeedbackPredictions
		attempts += o.Attempts
		passed += o.Passed
		total += o.Cases
		separatePassed += o.SeparatePassed
		separateTotal += o.SeparateCases
		noChoice += o.NoChoice
		changed += o.ChangedJudgments
		if o.IntentAgreement {
			intent++
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/compound-path-study/v1", "decision": "PASS", "runner_revision": revision, "native_revision": compoundNative, "pilot": pilot, "observations": observations, "native_calls": len(observations), "actual_model_predictions": predictions, "feedback_predictions": feedback, "candidate_attempts": attempts, "repeated_finite_passes": passed, "repeated_finite_cases": total, "repeated_separate_input_passes": separatePassed, "repeated_separate_input_cases": separateTotal, "original_intention_mask_matches": intent, "zero_call_ranking_unnecessary_receipts": noChoice, "feedback_judgments_different_from_initial_proposal": changed, "actual_generated_go_processes": len(executions), "actual_generated_function_evaluations": len(executions) * len(inputs), "execution_inputs": inputs, "scope": "Three interacting body templates and twelve structural intention groups using five existing binary kinds. Whole-body four-path construction, finite TDD continuation and ambiguous/contradictory contracts. Repeated language/contract/model views are not independent tasks. Some structural paths commute and are behaviorally equivalent. Fixed offline/parent/new-model order; no causal timing or host utilization claim. Actual Go execution is deduplicated by emitted source and independently checked against arithmetic state transitions. No training or upstream Laya calls."})
}
