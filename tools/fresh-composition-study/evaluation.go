package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type counts struct {
	Views           int    `json:"function_views"`
	Cases           int    `json:"ordered_finite_cases"`
	InitialPassed   int    `json:"initial_cases_passed"`
	InitialComplete int    `json:"initial_complete_functions"`
	FinalComplete   int    `json:"complete_functions_by_budget_four"`
	Extras          int    `json:"additional_candidate_attempts"`
	Calls           int    `json:"actual_model_predictions"`
	FeedbackCalls   int    `json:"actual_feedback_predictions"`
	Declines        int    `json:"zero_call_feedback_context_declines"`
	TypeRejected    int    `json:"type_rejections"`
	Unattempted     int    `json:"unattempted_masks_after_completion"`
	Different       int    `json:"bilingual_initial_mask_disagreements"`
	Pairs           int    `json:"bilingual_function_pairs"`
	MinimumBudget   [4]int `json:"minimum_complete_budget_histogram_1_to_4"`
	CurveComplete   [4]int `json:"complete_function_curve_budgets_1_to_4"`
	CurvePassed     [4]int `json:"best_passing_case_curve_budgets_1_to_4"`
	PredictionNS    int64  `json:"summed_observed_prediction_ns"`
}
type score struct {
	Total    counts            `json:"total"`
	Families map[string]counts `json:"family_language"`
}
type observation struct {
	ID       string                     `json:"function_view_id"`
	Inputs   [2]string                  `json:"input_sha256"`
	Search   pathplan.SearchResult      `json:"search"`
	Progress []pathplan.SessionProgress `json:"session_progress"`
	Feedback []pathplan.FeedbackReceipt `json:"feedback_judgments"`
	WallNS   int64                      `json:"whole_session_wall_ns"`
}

func observe(v view, model *decision.Model) (observation, error) {
	prepared, cases, err := prepareView(v)
	if err != nil {
		return observation{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	o := observation{ID: v.ID, Inputs: [2]string{v.Rows[0].Input.SHA, v.Rows[1].Input.SHA}}
	start := time.Now()
	if model == nil {
		o.Search, _, o.Progress, err = prepared.SearchBatches(ctx, nil, cases, 4, 1, "")
	} else {
		hint := &pathplan.CIHint{SourceSHA: nativeMain, Status: "PASS"}
		o.Search, _, o.Progress, o.Feedback, err = prepared.SearchFeedbackBatchesUnfixed(ctx, model, cases, 4, 1, "", 3, hint)
	}
	o.WallNS = time.Since(start).Nanoseconds()
	if err != nil {
		return o, err
	}
	if err = validateObservation(v, o, model != nil); err != nil {
		return o, err
	}
	return o, nil
}

func validateObservation(v view, o observation, hasModel bool) error {
	r := o.Search
	if r.TrainingTotal != 16 || r.SelectedTrainingPassed != 16 || r.TypeRejected != 0 || r.DeclaredCombinations != 4 ||
		len(r.Attempts) < 1 || len(r.Attempts) > 4 || r.Evaluated != len(r.Attempts) || r.Selection.ExternalCalls != 0 {
		return errors.New("bounded finite assembly differs")
	}
	seen := map[uint16]bool{}
	for _, a := range r.Attempts {
		if a.Mask > 3 || seen[a.Mask] || a.Total != 16 || a.Passed != v.Rows[0].Finite.Passed[a.Mask] || len(a.Results) != 16 {
			return errors.New("independent complete-mask target differs")
		}
		seen[a.Mask] = true
		passed := 0
		cases, err := compositionstudy.Cases(v.Rows[0].Family, v.Rows[0].Config, v.Rows[0].Desired)
		if err != nil {
			return err
		}
		for i, c := range a.Results {
			actual, err := compositionstudy.Oracle(v.Rows[0].Family, v.Rows[0].Config, int(a.Mask), cases[i].Input)
			if err != nil || c.Input != cases[i].Input || c.Expected != cases[i].Expected || c.Actual != actual {
				return errors.New("independent ordered arithmetic actual differs")
			}
			if c.Passed {
				passed++
			}
			if c.Passed != (c.Actual == c.Expected) {
				return errors.New("case flag differs")
			}
		}
		if passed != a.Passed {
			return errors.New("complete case count differs")
		}
	}
	feedbackCalls := 0
	for _, f := range o.Feedback {
		feedbackCalls += f.ModelCalls
		if f.CI == nil || f.CI.SourceSHA != nativeMain || f.CI.Status != "PASS" || f.CIIsAuthority {
			return errors.New("observed CI hint binding differs")
		}
		if f.ContextDeclined && f.ModelCalls != 0 {
			return errors.New("decline invoked predictor")
		}
	}
	expected := 0
	if hasModel {
		expected = 2 + feedbackCalls
	}
	if r.Selection.ModelCalls != expected {
		return errors.New("actual prediction accounting differs")
	}
	return nil
}

func add(c *counts, o observation) {
	r := o.Search
	c.Views++
	c.Cases += 16
	c.InitialPassed += r.Attempts[0].Passed
	if r.Attempts[0].Passed == 16 {
		c.InitialComplete++
	}
	c.FinalComplete++
	c.Extras += len(r.Attempts) - 1
	c.Calls += r.Selection.ModelCalls
	c.TypeRejected += r.TypeRejected
	c.Unattempted += r.Unattempted
	c.MinimumBudget[len(r.Attempts)-1]++
	best := 0
	for budget := 0; budget < 4; budget++ {
		if budget < len(r.Attempts) {
			best = max(best, r.Attempts[budget].Passed)
		}
		c.CurvePassed[budget] += best
		if best == 16 {
			c.CurveComplete[budget]++
		}
	}
	for _, r := range r.Selection.Receipts {
		c.PredictionNS += r.PredictNS
	}
	for _, f := range o.Feedback {
		c.FeedbackCalls += f.ModelCalls
		if f.ContextDeclined {
			c.Declines++
		}
		for _, j := range f.Judgments {
			c.PredictionNS += j.Prediction.PredictNS
		}
	}
}

func evaluate(views []view, split, id string, model *decision.Model, output string) (score, error) {
	feature := decision.SplitContextIntentFeatureVersion
	if model != nil {
		feature = model.FeatureVersion()
	}
	f, err := os.OpenFile(filepath.Join(output, split+"-"+id+".jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return score{}, err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	result := score{Families: map[string]counts{}}
	initial := map[string]map[string]uint16{}
	for _, v := range views {
		r := v.Rows[0]
		if r.Split != split || r.Feature != feature {
			continue
		}
		o, e := observe(v, model)
		if o.Search.Schema != "" {
			if err = encoder.Encode(o); err != nil {
				return result, err
			}
		}
		if e != nil {
			return result, e
		}
		add(&result.Total, o)
		key := r.Family + "/" + r.Language
		family := result.Families[key]
		add(&family, o)
		result.Families[key] = family
		if initial[r.Group] == nil {
			initial[r.Group] = map[string]uint16{}
		}
		initial[r.Group][r.Language] = o.Search.Attempts[0].Mask
	}
	for _, pair := range initial {
		if len(pair) != 2 {
			return result, errors.New("bilingual pair missing")
		}
		result.Total.Pairs++
		if pair["en"] != pair["ko"] {
			result.Total.Different++
		}
	}
	if result.Total.Views != 384 || result.Total.Pairs != 192 {
		return result, errors.New("fixed function-view denominator differs")
	}
	return result, nil
}

func choose(scores map[string]score, pins map[string]modelPin) (string, error) {
	ids := make([]string, 0, len(pins))
	for id := range pins {
		if _, ok := scores[id]; !ok {
			return "", errors.New("calibration candidate missing")
		}
		ids = append(ids, id)
	}
	if len(ids) != 6 {
		return "", errors.New("six candidates required")
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := scores[ids[i]].Total, scores[ids[j]].Total
		if a.Extras != b.Extras {
			return a.Extras < b.Extras
		}
		if a.Calls != b.Calls {
			return a.Calls < b.Calls
		}
		if a.Different != b.Different {
			return a.Different < b.Different
		}
		if pins[ids[i]].Packed != pins[ids[j]].Packed {
			return pins[ids[i]].Packed < pins[ids[j]].Packed
		}
		return ids[i] < ids[j]
	})
	return ids[0], nil
}

func run(curriculum, models, output, revision string) error {
	if err := preflight(output, revision); err != nil {
		return err
	}
	views, err := loadViews(curriculum)
	if err != nil {
		return err
	}
	loaded, pins, err := loadModels(models)
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/fresh-composition-study-preexecution/v1",
		"source_revision": revision, "dataset_sha256": datasetSHA, "protocol_sha256": protocolSHA, "models": pins,
		"planned_initial_session_predictions": 9216, "planned_parity_predictions": 192, "planned_warm_and_allocation_predictions": 12012,
		"calibration_function_views_per_arm": 384, "development_function_views_per_arm": 384, "candidate_budget": 4, "step": 1,
		"feedback_round_bound": 3, "caller_ci_hint": pathplan.CIHint{SourceSHA: nativeMain, Status: "PASS"}, "new_optimizer_updates": 0}); err != nil {
		return err
	}
	parities, runtime := map[string]float64{}, map[string]runtimeObservation{}
	for _, arm := range arms {
		parities[arm], err = parity(models, arm, loaded, views)
		if err != nil {
			return err
		}
	}
	calibration := map[string]score{}
	for _, arm := range arms {
		for _, variant := range variants {
			id := arm + "-" + variant
			calibration[id], err = evaluate(views, "calibration", id, loaded[id], output)
			if err != nil {
				return err
			}
		}
	}
	calibration["offline"], err = evaluate(views, "calibration", "offline", nil, output)
	if err != nil {
		return err
	}
	selected, err := choose(calibration, pins)
	if err != nil {
		return err
	}
	selection := map[string]any{"schema": "gooo/fresh-composition-calibration-selection/v1", "selected": selected,
		"model": pins[selected], "dataset_sha256": datasetSHA, "source_revision": revision, "calibration": calibration,
		"rule": "calibration only: extra attempts, actual predictions, bilingual initial mask disagreements, packed bytes, arm/variant names",
		"frozen_before_development_session_predictions": true}
	if err = save(filepath.Join(output, "selection.json"), selection); err != nil {
		return err
	}
	development := map[string]score{}
	for _, arm := range arms {
		for _, variant := range variants {
			id := arm + "-" + variant
			development[id], err = evaluate(views, "development", id, loaded[id], output)
			if err != nil {
				return err
			}
			for _, v := range views {
				if v.Rows[0].Feature == loaded[id].FeatureVersion() {
					runtime[id], err = measureModel(loaded[id], v.Rows[0].Input.Text)
					break
				}
			}
			if err != nil {
				return err
			}
		}
	}
	development["offline"], err = evaluate(views, "development", "offline", nil, output)
	if err != nil {
		return err
	}
	calls := 192 + 12012
	for _, scores := range []map[string]score{calibration, development} {
		for _, s := range scores {
			calls += s.Total.Calls
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/fresh-composition-study/v1", "status": "PASS",
		"source_revision": revision, "dataset_sha256": datasetSHA, "selected_candidate": selected, "calibration": calibration,
		"development": development, "go_python_parity_max_absolute_error": parities, "parity_predictions": 192,
		"runtime": runtime, "benchmark_predictions_including_allocation_probe": 12012, "actual_model_predictions_all_stages": calls,
		"native_calls": 0, "new_optimizer_updates": 0, "default_model_promoted": false,
		"scope": "Six compositions, four goals with bilingual/parameter variants; all 16 ordered independent cases retained. Model ranks legal compiler-owned fragments and observed feedback only. Calibration selector frozen before development sessions. Four-path completeness is finite enumerative evidence, not general language understanding. Warm inference allocations exclude loading/context/session and decoded ternary matrices remain int8."})
}
