package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func budgetGroup(rows []budgetObservation) map[string]any {
	passed, cases, separate, hidden, calls, feedback, attempts, matches, maxima := 0, 0, 0, 0, 0, 0, 0, 0, 0
	area, attainment := 0.0, 0.0
	var latency, native []float64
	for _, o := range rows {
		passed += o.Passed
		cases += o.Cases
		separate += o.SeparatePassed
		hidden += o.SeparateCases
		calls += o.Predictions
		feedback += o.FeedbackPredictions
		attempts += len(o.Sequence)
		area += o.BudgetArea
		attainment += o.Attainment
		if o.Selected == o.Intended {
			matches++
		}
		if o.Passed == o.LegalMaximum {
			maxima++
		}
		latency = append(latency, float64(o.Latency)/1e6)
		native = append(native, o.NativeMS)
	}
	sort.Float64s(latency)
	return map[string]any{"observations": len(rows), "finite_passed": passed, "finite_cases": cases,
		"declared_completion_percent": 100 * float64(passed) / float64(cases), "actual_separate_passed": separate, "actual_separate_cases": hidden,
		"separate_completion_percent": 100 * float64(separate) / float64(hidden), "intention_mask_matches": matches,
		"finite_maximum_reached": maxima, "mean_finite_attainment_percent": attainment / float64(len(rows)),
		"mean_completion_over_budget_positions_percent": area / float64(len(rows)), "actual_model_predictions": calls,
		"feedback_predictions": feedback, "committed_candidates": attempts, "request_p50_ms": compoundMedian(latency),
		"request_p95_ms": latency[(95*len(rows)+99)/100-1], "native_total_p50_ms": compoundMedian(native)}
}

func validateBudgetPrefixes(c budgetCollection) error {
	byID := map[string]budgetObservation{}
	for _, o := range c.observations {
		if _, ok := byID[o.ID]; ok {
			return fmt.Errorf("duplicate observation: %s", o.ID)
		}
		byID[o.ID] = o
	}
	for _, o := range c.observations {
		for _, b := range []int{1, 2, 4} {
			other, ok := byID[fmt.Sprintf("%s-budget-%d-%s", o.Arm, b, o.Case)]
			if !ok {
				return fmt.Errorf("budget pair absent")
			}
			n := min(len(o.Sequence), len(other.Sequence))
			if !reflect.DeepEqual(o.Sequence[:n], other.Sequence[:n]) || !reflect.DeepEqual(o.Curve[:n], other.Curve[:n]) {
				return fmt.Errorf("same-arm prefix differs across budgets: %s", o.ID)
			}
		}
		if o.Budget == 1 && strings.HasSuffix(o.Arm, "-feedback") {
			other := byID[strings.TrimSuffix(o.Arm, "-feedback")+"-initial-budget-1-"+o.Case]
			if o.SHA != other.SHA || !reflect.DeepEqual(o.Sequence, other.Sequence) || o.Passed != other.Passed || o.FeedbackPredictions != 0 {
				return fmt.Errorf("feedback before the first committed attempt")
			}
		}
	}
	return nil
}

func summarizeBudget(c budgetCollection) map[string]any {
	groups := map[string][]budgetObservation{}
	subgroups := map[string][]budgetObservation{}
	byID := map[string]budgetObservation{}
	totalCalls, totalFeedback, totalAttempts, totalSkips := 0, 0, 0, 0
	for _, o := range c.observations {
		key := fmt.Sprintf("%s-budget-%d", o.Arm, o.Budget)
		groups[key] = append(groups[key], o)
		subkey := key + "-" + o.Contract + "-" + o.Language
		subgroups[subkey] = append(subgroups[subkey], o)
		byID[o.ID] = o
		totalCalls += o.Predictions
		totalFeedback += o.FeedbackPredictions
		totalAttempts += len(o.Sequence)
		totalSkips += o.Skipped
	}
	summaries := map[string]any{}
	strata := map[string]any{}
	costs := map[string]any{}
	for key, rows := range groups {
		summaries[key] = budgetGroup(rows)
	}
	for key, rows := range subgroups {
		strata[key] = budgetGroup(rows)
	}
	for key := range groups {
		cpu, wall := int64(0), int64(0)
		var rss, startup []float64
		for id, p := range c.processes {
			if !strings.HasPrefix(id, key+"-chunk-") {
				continue
			}
			cpu += p.Metrics.User + p.Metrics.System
			wall += p.Metrics.Wall
			rss = append(rss, float64(p.Metrics.RSS)/(1<<20))
			startup = append(startup, float64(p.StartupNS)/1e6)
		}
		medianRSS := compoundMedian(rss)
		medianStartup := compoundMedian(startup)
		costs[key] = map[string]any{"native_processes": len(rss), "valid_constructions": len(groups[key]),
			"whole_child_cpu_ms_per_valid_construction": float64(cpu) / 1e6 / float64(len(groups[key])),
			"whole_child_wall_ms":                       float64(wall) / 1e6, "aggregate_child_cpu_percent_one_core": 100 * float64(cpu) / float64(wall),
			"child_peak_rss_p50_mib": medianRSS, "child_peak_rss_max_mib": rss[len(rss)-1],
			"startup_until_constructor_p50_ms": medianStartup, "startup_until_constructor_max_ms": startup[len(startup)-1]}
	}
	var pairs []map[string]any
	for _, o := range c.observations {
		if !strings.HasSuffix(o.Arm, "-feedback") {
			continue
		}
		initial := byID[strings.TrimSuffix(o.Arm, "-feedback")+"-initial-budget-"+fmtInt(o.Budget)+"-"+o.Case]
		pairs = append(pairs, map[string]any{"feedback": o.ID, "initial": initial.ID, "same_candidate_sequence": reflect.DeepEqual(o.Sequence, initial.Sequence),
			"same_selected_body": o.SHA == initial.SHA, "finite_pass_delta": o.Passed - initial.Passed, "separate_pass_delta": o.SeparatePassed - initial.SeparatePassed,
			"attempt_delta": len(o.Sequence) - len(initial.Sequence), "prediction_delta": o.Predictions - initial.Predictions,
			"budget_position_completion_delta_percent": o.BudgetArea - initial.BudgetArea})
	}
	var bilingual []map[string]any
	for _, o := range c.observations {
		if o.Language != "en" {
			continue
		}
		other := byID[fmt.Sprintf("%s-budget-%d-%s", o.Arm, o.Budget, strings.Replace(o.Case, "-en-", "-ko-", 1))]
		bilingual = append(bilingual, map[string]any{"english": o.ID, "korean": other.ID, "same_selected_mask": o.Selected == other.Selected,
			"same_selected_body": o.SHA == other.SHA, "korean_minus_english_finite_passed": other.Passed - o.Passed,
			"korean_minus_english_separate_passed":           other.SeparatePassed - o.SeparatePassed,
			"korean_minus_english_budget_completion_percent": other.BudgetArea - o.BudgetArea})
	}
	return map[string]any{"schema": "gooo/native-budget-study/v1", "decision": "PASS", "runner_revision": c.pre.Runner, "native_revision": c.pre.Native,
		"files_sha256": c.files, "observations": c.observations, "summaries": summaries, "contract_language_subgroups": strata, "whole_process_costs": costs,
		"feedback_initial_pairs": pairs, "bilingual_pairs": bilingual, "valid_native_constructions": len(c.observations), "source_rejections": len(c.processes),
		"native_processes": len(c.processes), "actual_model_predictions": totalCalls, "feedback_predictions": totalFeedback,
		"committed_candidates": totalAttempts, "fixed_coordinate_prediction_skips": totalSkips, "actual_generated_go_processes": len(c.sources),
		"actual_generated_function_invocations": len(c.sources) * len(c.inputs), "execution_inputs": c.inputs,
		"new_training_steps": 0, "new_gpu_work": 0, "new_upstream_laya_calls": 0,
		"scope": "Actual clean deployed main worker, 72 reused bilingual/contract views over 12 existing intent groups and three templates. Nine arms and budgets 1/2/4 produce repeated observations, not new independent experiments. Only declared finite cases reach selection; separate inputs and authored mask remain evaluator-only. Independent arithmetic oracle establishes the restricted four-path finite maximum after construction, including for partial native search. This is not all-input or language completeness. Curves record actual committed attempts; budget-position means carry the terminal value forward after early completion and do not invent extra attempts. Same-arm prefixes match across budgets; first-budget feedback/initial bodies match. Generated Go is independently compiled and executed once per distinct source, then reconciled for every selected mask. Request timing excludes constructor startup/setup; whole-child costs include setup and one rejection plus 12 valid requests. Controller costs, host CPU growth and maximum-size/long-stream RAM are unmeasured. Original UNKNOWN caller CI hint is unchanged and is not authority. Frozen own models are reused without promotion. First-shot correctness is not acceptance. Offline audit/summary performs zero new model/native/generated-Go calls."}
}
