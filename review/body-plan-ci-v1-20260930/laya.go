package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodydecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

const expectedLayaRevision = "55cf4c4ebb4ebe31b2550e8bdf3bd21b99753851"

type layaServerReply struct {
	Model   string `json:"model"`
	Answers map[string]struct {
		Type          string             `json:"type"`
		Choice        string             `json:"choice"`
		Probabilities map[string]float64 `json:"probabilities"`
	} `json:"answers"`
	Routing struct {
		Model string `json:"model"`
	} `json:"routing"`
}

type layaRequest struct {
	Model     string            `json:"model"`
	State     map[string]string `json:"state"`
	Questions map[string]any    `json:"questions"`
}

func auditLayaCapture(repo, capture string, ci evaluation, ciStats map[string]armStats) (map[string]any, error) {
	bytesReport, err := readFile(filepath.Join(capture, "report.json"), 32<<20)
	if err != nil {
		return nil, err
	}
	var report evaluation
	if err := strictjson.Decode(bytesReport, &report); err != nil {
		return nil, err
	}
	rowsBytes, err := readFile(filepath.Join(repo, "studies/body-plan-v1/cohort.jsonl"), 16<<20)
	if err != nil {
		return nil, err
	}
	rows, err := readRows(rowsBytes)
	if err != nil {
		return nil, err
	}
	if err := validateLayaReport(report, ci, rows, digest(rowsBytes)); err != nil {
		return nil, err
	}
	holesByPlan := make(map[string]map[string]bodyplan.Expr, len(rows))
	for _, item := range rows {
		holesByPlan[item.ID] = make(map[string]bodyplan.Expr)
		for _, expr := range item.Plan.Expressions {
			if expr.Kind == "hole" {
				holesByPlan[item.ID][expr.HoleID] = expr
			}
		}
	}
	byPlan, exchangeDigest, latency, fileCount, err := readLayaExchanges(capture, holesByPlan)
	if err != nil {
		return nil, err
	}
	reportByKey, selectedByKey, err := readLayaCells(capture, report)
	if err != nil {
		return nil, err
	}
	maxSharedProbabilityDelta, err := compareSharedCells(reportByKey, ci.Cells)
	if err != nil {
		return nil, err
	}
	localArms := []string{"deterministic", "deterministic_search", "fp32", "fp32_search", "laya_multilingual", "laya_multilingual_search", "ptq_ternary", "ptq_ternary_search", "qat_ternary", "qat_ternary_search"}
	localStats := make(map[string]armStats)
	markerCount := 0
	for _, arm := range localArms {
		if err := checkArmReplay(repo, capture, arm, rows, reportByKey, selectedByKey, report.ModelArtifactSHA256, byPlan, &localStats, &markerCount); err != nil {
			return nil, fmt.Errorf("local arm %s: %w", arm, err)
		}
	}
	if markerCount != 45200 || report.ObservedGoCases != markerCount || report.PlannedGoCases != markerCount || report.UnknownGoCases != 0 {
		return nil, fmt.Errorf("local Go markers/report denominator mismatch: markers=%d observed=%d planned=%d unknown=%d", markerCount, report.ObservedGoCases, report.PlannedGoCases, report.UnknownGoCases)
	}
	if err := checkArmScoreExpectations(localStats); err != nil {
		return nil, err
	}
	if err := compareSharedArms(localStats, ciStats); err != nil {
		return nil, err
	}
	layaStats := localStats["laya_multilingual"]
	layaSearch := localStats["laya_multilingual_search"]
	if layaStats.HeldoutCorrect != 945 || layaStats.HeldoutTotal != 1280 || layaStats.InitialOracleCorrectHole != 144 || layaStats.TotalOracleHoles != 222 {
		return nil, fmt.Errorf("Laya first-choice denominators changed: %+v", layaStats)
	}
	if layaSearch.HeldoutCorrect != 1280 || layaSearch.HeldoutTotal != 1280 || layaSearch.SearchAttempts != 404 || layaSearch.SearchCells != 128 || layaSearch.EmittedOracleCorrectHole != 222 {
		return nil, fmt.Errorf("Laya training-only search summary changed: %+v", layaSearch)
	}
	resource, err := auditLayaResources(capture)
	if err != nil {
		return nil, err
	}
	latencySummary := summarizeLatency(latency)
	controlAttempts := localStats["deterministic_search"].SearchAttempts
	modelSavings := controlAttempts - layaSearch.SearchAttempts
	return map[string]any{
		"source_revision":               report.SourceRevision,
		"laya_revision":                 expectedLayaRevision,
		"capture_report_sha256":         digest(bytesReport),
		"final_selected_cells_sha256":   digest(mustRead(filepath.Join(capture, "selected-cells.jsonl"), 32<<20)),
		"initial_preexecution_snapshot": preexecutionSummary(capture),
		"progress_snapshot":             progressSummary(capture),
		"capture": map[string]any{
			"intents_reused_across_arms": 128, "planned_cells": report.PlannedCells, "observed_cells": report.ObservedCells,
			"report_wall_ms":             report.WallMS,
			"planned_and_reported_posts": report.PlannedLayaPOSTs, "attempted_posts": report.LayaPOSTs, "captured_exchange_files": fileCount,
			"captured_exchanges": len(latency), "tiny_model_predictions": report.TinyPredictions,
			"planned_go_cases": report.PlannedGoCases, "observed_unique_go_case_markers": markerCount, "unknown_go_cases": report.UnknownGoCases,
			"native_generated_cells": report.NativePassed, "native_failed_or_unknown_cells": report.NativeFailedOrUnknown,
			"runner_binary_sha256": report.RunnerSHA, "gooo_binary_sha256": report.GoooSHA, "go_tool_sha256": report.GoSHA,
			"executable_pins_stable_after_execution": report.ExecutablePinsStable,
			"transient_binary_bytes_in_capture":      false,
			"binary_digest_interpretation":           "These digests and the stable-after-execution flag are recorded in the final report; the transient runner/compiler/toolchain bytes are not included here for independent rehashing.",
		},
		"exchange_binding": map[string]any{
			"all_requests_exactly_match_frozen_hole_text_and_allowed_operation_descriptions": true,
			"requests_include_no_training_or_heldout_cases":                                  true,
			"each_raw_reply_choice_and_probabilities_match_selection_receipt":                true,
			"exchange_bundle_sha256": exchangeDigest, "one_exchange_per_declared_hole": true,
			"retry_or_extra_exchange_records": 0,
		},
		"laya_first_choice": map[string]any{
			"heldout_correct": layaStats.HeldoutCorrect, "heldout_total": layaStats.HeldoutTotal,
			"gold_holes_correct": layaStats.InitialOracleCorrectHole, "holes_total": layaStats.TotalOracleHoles,
		},
		"finite_training_case_search": map[string]any{
			"attempt_limit_per_cell": 64, "laya_search_attempts_including_initial": layaSearch.SearchAttempts,
			"deterministic_control_attempts_including_initial": controlAttempts, "attempts_saved_vs_control": modelSavings,
			"attempt_reduction_fraction_vs_control": float64(modelSavings) / float64(controlAttempts),
			"laya_search_heldout_correct":           layaSearch.HeldoutCorrect, "laya_search_heldout_total": layaSearch.HeldoutTotal,
			"interpretation": "The search enumerates the same finite candidate space using training cases only. It begins from Laya's choice; the final 100% heldout score is attainable without a model and is not attributable to Laya alone.",
		},
		"request_latency_ms":        latencySummary,
		"process_observation":       resource,
		"cross_arm_limit":           "Laya receives operation descriptions with the hole text; tiny-model arms classify the text against a global label set. Their first-choice scores compare these frozen protocols on the same plans, not model quality in isolation.",
		"shared_ci_cell_comparison": map[string]any{"eight_shared_arms_matched_selections_scores_and_source_hashes": true, "maximum_absolute_probability_delta_across_platform_runs": maxSharedProbabilityDelta, "comparison_tolerance": 0.000001, "timing_fields_compared": false},
	}, nil
}

func validateLayaReport(report, ci evaluation, rows []row, cohortSHA string) error {
	if report.Schema != "gooo/multi-node-body-plan-evaluation/v1" || report.Status != "completed" || report.SourceRevision != expectedRevision || report.LayaRevision != expectedLayaRevision || report.CohortSHA != cohortSHA || report.CohortSHA != ci.CohortSHA {
		return errors.New("local Laya report schema/status/source/cohort binding invalid")
	}
	if report.PlannedIntents != len(rows) || report.PlannedCells != 1280 || report.ObservedCells != 1280 || report.UnstartedCells != 0 || report.NativePassed != 1280 || report.NativeFailedOrUnknown != 0 || report.UnknownGoCases != 0 || !report.ExecutablePinsStable {
		return errors.New("local Laya report has incomplete plan/cell/native evidence")
	}
	if report.PlannedGoCases != 45200 || report.ObservedGoCases != 45200 || report.AttemptLimit != 64 || report.PlannedLayaPOSTs != 222 || report.LayaPOSTs != 222 || report.TinyPredictions != 666 || report.WallMS <= 0 {
		return errors.New("local Laya report count, POST, prediction or timing totals invalid")
	}
	if report.RunnerSHA == "" || report.GoooSHA != "7329b8d255b083bacfd7d44c7271caa4e3c8254bd48cc085068c92665a02591a" || report.GoSHA != "a19a71df81715c12d9a7e81bab036c12696fec1ddbd4258b48a2131a9080b267" {
		return errors.New("local Laya binary/tool pins differ from declared capture")
	}
	for variant, values := range ci.ModelArtifactSHA256 {
		if !reflect.DeepEqual(report.ModelArtifactSHA256[variant], values) {
			return fmt.Errorf("local Laya model pins differ from CI for %s", variant)
		}
	}
	if len(report.Cells) != 1280 {
		return errors.New("local Laya report cell vector has wrong size")
	}
	return nil
}

func readLayaExchanges(capture string, holes map[string]map[string]bodyplan.Expr) (map[string][]layaExchange, string, []int64, int, error) {
	entries, err := os.ReadDir(capture)
	if err != nil {
		return nil, "", nil, 0, err
	}
	var filenames []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "body-") && strings.HasSuffix(entry.Name(), "-laya-exchanges.json") {
			filenames = append(filenames, entry.Name())
		}
	}
	sort.Strings(filenames)
	if len(filenames) != len(holes) {
		return nil, "", nil, 0, fmt.Errorf("exchange file count %d differs from intent count %d", len(filenames), len(holes))
	}
	byPlan := make(map[string][]layaExchange, len(holes))
	seen := make(map[string]bool)
	var latency []int64
	var tree strings.Builder
	for _, filename := range filenames {
		bodyID := strings.TrimSuffix(strings.TrimPrefix(filename, "body-"), "-laya-exchanges.json")
		planID := "body-" + bodyID
		declared, ok := holes[planID]
		if !ok {
			return nil, "", nil, 0, fmt.Errorf("exchange file names undeclared plan %s", planID)
		}
		data, err := readFile(filepath.Join(capture, filename), 1<<20)
		if err != nil {
			return nil, "", nil, 0, err
		}
		var exchanges []bodydecision.Exchange
		if err := strictjson.Decode(data, &exchanges); err != nil {
			return nil, "", nil, 0, err
		}
		if len(exchanges) != len(declared) {
			return nil, "", nil, 0, fmt.Errorf("%s has %d exchanges for %d holes", planID, len(exchanges), len(declared))
		}
		for _, exchange := range exchanges {
			key := planID + "\x00" + exchange.HoleID
			expr, ok := declared[exchange.HoleID]
			if !ok || seen[key] {
				return nil, "", nil, 0, fmt.Errorf("unknown or duplicate exchange %s/%s", planID, exchange.HoleID)
			}
			if exchange.Status != "completed_validated" || exchange.WallNS <= 0 || len(exchange.Response) == 0 || len(exchange.Response) > 65536 {
				return nil, "", nil, 0, fmt.Errorf("incomplete or oversized exchange %s/%s", planID, exchange.HoleID)
			}
			if err := checkLayaRequest(exchange.Request, expr); err != nil {
				return nil, "", nil, 0, fmt.Errorf("request %s/%s: %w", planID, exchange.HoleID, err)
			}
			reply, err := decodeLayaReply(exchange.Response)
			if err != nil {
				return nil, "", nil, 0, fmt.Errorf("reply %s/%s: %w", planID, exchange.HoleID, err)
			}
			answer, ok := reply.Answers["operation"]
			if reply.Routing.Model != "multilingual" || answer.Type != "choice" || !contains(expr.Allowed, answer.Choice) || len(answer.Probabilities) != len(expr.Allowed) {
				return nil, "", nil, 0, fmt.Errorf("reply route, type, choice or labels invalid for %s/%s", planID, exchange.HoleID)
			}
			for _, operation := range expr.Allowed {
				probability, ok := answer.Probabilities[operation]
				if !ok || math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 || probability > 1 {
					return nil, "", nil, 0, fmt.Errorf("reply probability invalid for %s/%s/%s", planID, exchange.HoleID, operation)
				}
			}
			seen[key] = true
			byPlan[planID] = append(byPlan[planID], layaExchange{Exchange: exchange, Choice: answer.Choice, Probabilities: answer.Probabilities})
			latency = append(latency, exchange.WallNS)
			fmt.Fprintf(&tree, "%s\x00%s\x00%s\x00%s\x00%d\n", planID, exchange.HoleID, digest(exchange.Request), digest(exchange.Response), exchange.WallNS)
		}
		fmt.Fprintf(&tree, "file\x00%s\x00%s\n", filename, digest(data))
	}
	if len(seen) != 222 {
		return nil, "", nil, 0, fmt.Errorf("captured %d distinct hole exchanges, expected 222", len(seen))
	}
	return byPlan, digest([]byte(tree.String())), latency, len(filenames), nil
}

type layaExchange struct {
	bodydecision.Exchange
	Choice        string
	Probabilities map[string]float64
}

func checkLayaRequest(raw json.RawMessage, expr bodyplan.Expr) error {
	criteria := map[string]string{
		"add":        "Add the left and right values: left + right.",
		"subtract":   "Subtract the right value from the left: left - right.",
		"multiply":   "Multiply the two values: left * right.",
		"less_than":  "Test whether left is strictly smaller than right: left < right.",
		"less_equal": "Test whether left is smaller than or equal to right: left <= right.",
		"equal":      "Test whether both values are equal: left == right.",
		"and":        "Require both Boolean values to be true: left && right.",
		"or":         "Require at least one Boolean value to be true: left || right.",
	}
	allowed := make(map[string]string, len(expr.Allowed))
	for _, operation := range expr.Allowed {
		allowed[operation] = criteria[operation]
	}
	want, err := json.Marshal(layaRequest{
		Model: "multilingual", State: map[string]string{"request": expr.Text},
		Questions: map[string]any{"operation": map[string]any{
			"type": "choice", "instructions": "Choose the declared binary operation matching the instruction. Return one listed operation.", "criteria": allowed,
		}},
	})
	if err != nil {
		return err
	}
	var gotValue any
	if err := strictjson.Decode(raw, &gotValue); err != nil {
		return err
	}
	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		return err
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		return errors.New("request does not equal the bounded text-and-choice payload")
	}
	return nil
}

func decodeLayaReply(raw []byte) (layaServerReply, error) {
	var fields map[string]json.RawMessage
	if err := strictjson.Decode(raw, &fields); err != nil {
		return layaServerReply{}, err
	}
	var reply layaServerReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return layaServerReply{}, err
	}
	if reply.Model == "" || reply.Answers == nil || reply.Routing.Model == "" {
		return layaServerReply{}, errors.New("missing server reply fields")
	}
	return reply, nil
}

func checkLayaSelection(item row, entry cell, exchanges []layaExchange) error {
	if entry.Arm != "laya_multilingual" && entry.Arm != "laya_multilingual_search" {
		return errors.New("unexpected external-provider arm")
	}
	if entry.Initial.ModelVariant != "laya_multilingual" || entry.Initial.ModelCalls != 0 || entry.Initial.ProviderCalls != countHoles(item.Plan) {
		return errors.New("Laya selection route or call count mismatch")
	}
	if len(entry.Initial.Holes) != countHoles(item.Plan) || len(entry.Initial.Choices) != countHoles(item.Plan) || len(exchanges) != countHoles(item.Plan) {
		return errors.New("Laya selection does not cover every declared hole")
	}
	exchangeByHole := make(map[string]layaExchange, len(exchanges))
	for _, exchange := range exchanges {
		exchangeByHole[exchange.HoleID] = exchange
	}
	for _, expr := range item.Plan.Expressions {
		if expr.Kind != "hole" {
			continue
		}
		exchange, ok := exchangeByHole[expr.HoleID]
		if !ok {
			return fmt.Errorf("missing raw exchange for %s", expr.HoleID)
		}
		var receipt *bodydecision.HoleReceipt
		for i := range entry.Initial.Holes {
			if entry.Initial.Holes[i].ID == expr.HoleID {
				receipt = &entry.Initial.Holes[i]
				break
			}
		}
		if receipt == nil || entry.Initial.Choices[expr.HoleID] != exchange.Choice || receipt.Selected != exchange.Choice || receipt.Proposed != exchange.Choice {
			return fmt.Errorf("raw provider choice does not bind selection for %s", expr.HoleID)
		}
		if receipt.Mode != "laya_closed_choice" || receipt.TextSHA256 != digest([]byte(expr.Text)) || receipt.PredictNS != exchange.WallNS || receipt.Confidence != 0 {
			return fmt.Errorf("Laya receipt text, mode or timing differs for %s", expr.HoleID)
		}
		if len(receipt.Probabilities) != len(expr.Allowed) {
			return fmt.Errorf("probability vector size differs for %s", expr.HoleID)
		}
		for i, operation := range expr.Allowed {
			probability, ok := exchange.Probabilities[operation]
			if !ok || receipt.Probabilities[i].Label != operation || receipt.Probabilities[i].Probability != float32(probability) {
				return fmt.Errorf("probability receipt differs from raw response for %s", expr.HoleID)
			}
		}
	}
	return nil
}

func readLayaCells(capture string, report evaluation) (map[string]cell, map[string]cell, error) {
	reportByKey := make(map[string]cell, len(report.Cells))
	for _, item := range report.Cells {
		key := item.ID + "\x00" + item.Arm
		if _, exists := reportByKey[key]; exists {
			return nil, nil, errors.New("duplicate local final report cell")
		}
		reportByKey[key] = item
	}
	selectedBytes, err := readFile(filepath.Join(capture, "selected-cells.jsonl"), 32<<20)
	if err != nil {
		return nil, nil, err
	}
	selected, err := readCells(selectedBytes)
	if err != nil {
		return nil, nil, err
	}
	selectedByKey := make(map[string]cell, len(selected))
	for _, item := range selected {
		key := item.ID + "\x00" + item.Arm
		if _, exists := selectedByKey[key]; exists {
			return nil, nil, errors.New("duplicate local selected journal cell")
		}
		selectedByKey[key] = item
	}
	if len(reportByKey) != 1280 || len(selectedByKey) != 1280 {
		return nil, nil, errors.New("local report and selection journal do not cover 1280 cells")
	}
	return reportByKey, selectedByKey, nil
}

func compareSharedArms(local, ci map[string]armStats) error {
	for _, arm := range []string{"deterministic", "deterministic_search", "fp32", "fp32_search", "ptq_ternary", "ptq_ternary_search", "qat_ternary", "qat_ternary_search"} {
		a, b := local[arm], ci[arm]
		if a.Cells != b.Cells || a.TrainingCorrect != b.TrainingCorrect || a.TrainingTotal != b.TrainingTotal || a.HeldoutCorrect != b.HeldoutCorrect || a.HeldoutTotal != b.HeldoutTotal || a.GoCases != b.GoCases || a.InitialOracleCorrectHole != b.InitialOracleCorrectHole || a.TotalOracleHoles != b.TotalOracleHoles || a.EmittedOracleCorrectHole != b.EmittedOracleCorrectHole || a.SearchAttempts != b.SearchAttempts {
			return fmt.Errorf("shared arm %s differs between local Laya run and frozen CI audit", arm)
		}
	}
	return nil
}

func compareSharedCells(local map[string]cell, ciCells []cell) (float64, error) {
	ci := make(map[string]cell, len(ciCells))
	for _, item := range ciCells {
		ci[item.ID+"\x00"+item.Arm] = item
	}
	maximumProbabilityDelta := 0.0
	for key, item := range local {
		if !strings.Contains(key, "\x00") || !strings.Contains(key, "deterministic") && !strings.Contains(key, "fp32") && !strings.Contains(key, "ptq_ternary") && !strings.Contains(key, "qat_ternary") {
			continue
		}
		if strings.Contains(key, "laya_multilingual") {
			continue
		}
		other, ok := ci[key]
		if !ok || item.ID != other.ID || item.Family != other.Family || item.Arm != other.Arm || !reflect.DeepEqual(item.Choices, other.Choices) || item.Training != other.Training || item.Heldout != other.Heldout || item.OracleHoles != other.OracleHoles || item.TotalHoles != other.TotalHoles || item.GoooSHA != other.GoooSHA || item.GoSHA != other.GoSHA || item.NativePass != other.NativePass || item.GoTests != other.GoTests || item.CompiledTraining != other.CompiledTraining || item.CompiledHeldout != other.CompiledHeldout {
			return 0, fmt.Errorf("shared cell %q differs in selection, score, or emitted-source digest", key)
		}
		delta, same := selectionMeaningAndProbabilityDelta(item.Initial, other.Initial)
		if !same || !reflect.DeepEqual(item.Search, other.Search) {
			return 0, fmt.Errorf("shared cell %q differs in initial selection semantics or search receipt", key)
		}
		if delta > maximumProbabilityDelta {
			maximumProbabilityDelta = delta
		}
	}
	return maximumProbabilityDelta, nil
}

func selectionMeaningAndProbabilityDelta(a, b bodydecision.Selection) (float64, bool) {
	if !reflect.DeepEqual(a.Choices, b.Choices) || a.ModelVariant != b.ModelVariant || a.WeightsSHA256 != b.WeightsSHA256 || a.SeedSHA256 != b.SeedSHA256 || a.ModelCalls != b.ModelCalls || a.ProviderCalls != b.ProviderCalls || len(a.Holes) != len(b.Holes) {
		return 0, false
	}
	maximum := 0.0
	for index := range a.Holes {
		x, y := a.Holes[index], b.Holes[index]
		if x.ID != y.ID || x.TextSHA256 != y.TextSHA256 || x.Mode != y.Mode || x.Proposed != y.Proposed || x.Selected != y.Selected || math.Abs(float64(x.Confidence-y.Confidence)) > 0.000001 || len(x.Probabilities) != len(y.Probabilities) {
			return 0, false
		}
		for i := range x.Probabilities {
			if x.Probabilities[i].Label != y.Probabilities[i].Label {
				return 0, false
			}
			delta := math.Abs(float64(x.Probabilities[i].Probability - y.Probabilities[i].Probability))
			if delta > 0.000001 {
				return 0, false
			}
			if delta > maximum {
				maximum = delta
			}
		}
	}
	return maximum, true
}

func auditLayaResources(capture string) (map[string]any, error) {
	observations, err := readFile(filepath.Join(capture, "laya-process-observations.jsonl"), 2<<20)
	if err != nil {
		return nil, err
	}
	monitorSource, err := readFile(filepath.Join(capture, "resource-monitor.go.txt"), 1<<20)
	if err != nil {
		return nil, err
	}
	if digest(observations) != "cd428927b9963d08a4355e200510deb7cc235a319b5e139b92bebcafd4395cef" || digest(monitorSource) != "cfed251abeb1f1caadd23e81a36e981f2d120f15acf37b0c017dd8ca96b924b5" {
		return nil, errors.New("resource observations or monitor source differ from the captured receipt pins")
	}
	type sample struct {
		Sequence    int     `json:"sequence"`
		Status      string  `json:"status"`
		ElapsedMS   float64 `json:"elapsed_ms"`
		CPUSeconds  float64 `json:"cumulative_cpu_seconds"`
		RSSKiB      int64   `json:"rss_kib"`
		RollingPCPU float64 `json:"rolling_pcpu_one_core_percent"`
	}
	var samples []sample
	for _, line := range bytes.Split(bytes.TrimSpace(observations), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var item sample
		if err := strictjson.Decode(line, &item); err != nil {
			return nil, err
		}
		samples = append(samples, item)
	}
	if len(samples) != 762 || samples[0].Sequence != 0 || samples[0].Status != "observed" || samples[len(samples)-1].Sequence != 761 || samples[len(samples)-1].Status != "process_observation_ended" {
		return nil, errors.New("resource monitor samples are incomplete or out of order")
	}
	observed := samples[:len(samples)-1]
	peakRSS, peakPCPU := int64(0), 0.0
	for i, item := range observed {
		if item.Status != "observed" || item.Sequence != i || item.RSSKiB < 0 || item.CPUSeconds < 0 || item.ElapsedMS < 0 {
			return nil, errors.New("resource monitor sample has invalid fields/order")
		}
		if item.RSSKiB > peakRSS {
			peakRSS = item.RSSKiB
		}
		if item.RollingPCPU > peakPCPU {
			peakPCPU = item.RollingPCPU
		}
	}
	first, last := observed[0], observed[len(observed)-1]
	cpuDelta := last.CPUSeconds - first.CPUSeconds
	if cpuDelta < 0 {
		return nil, errors.New("resource monitor cumulative CPU counter regressed")
	}
	endMarker := samples[len(samples)-1]
	return map[string]any{
		"observations_sha256": digest(observations), "monitor_source_sha256": digest(monitorSource),
		"sample_count": len(observed), "end_marker_count": 1, "last_sample_elapsed_ms": last.ElapsedMS, "end_marker_elapsed_ms": endMarker.ElapsedMS,
		"sampled_process_cpu_seconds_delta": cpuDelta, "first_sample_rss_kib": first.RSSKiB,
		"sampled_process_peak_rss_kib": peakRSS, "rolling_one_core_pcpu_peak_percent": peakPCPU,
		"interpretation": "This process monitor spans substantially longer than the 47.123-second report wall window and includes idle time before and after the Laya run. CPU delta, RSS and rolling CPU are process observations across the whole monitor window, not per-inference CPU, peak live request memory, or whole-host utilization.",
	}, nil
}

func preexecutionSummary(capture string) map[string]any {
	data, err := readFile(filepath.Join(capture, "preexecution.json"), 2<<20)
	if err != nil {
		return map[string]any{"available": false}
	}
	var snapshot struct {
		Status     string `json:"status"`
		Observed   int    `json:"observed_selected_cells"`
		Posts      int    `json:"laya_posts_attempted"`
		GoCases    int    `json:"compiled_go_cases_observed"`
		PinsStable bool   `json:"executable_pins_stable_after_execution"`
	}
	if json.Unmarshal(data, &snapshot) != nil {
		return map[string]any{"available": true, "sha256": digest(data), "parseable": false}
	}
	return map[string]any{"sha256": digest(data), "status": snapshot.Status, "observed_cells_at_snapshot": snapshot.Observed, "posts_at_snapshot": snapshot.Posts, "go_cases_at_snapshot": snapshot.GoCases, "pins_stable_at_snapshot": snapshot.PinsStable, "interpretation": "This was an initial running-state snapshot preserved before capture; the completed final report and selected-cell journal are the outcome evidence."}
}

func progressSummary(capture string) map[string]any {
	data, err := readFile(filepath.Join(capture, "progress.json"), 2<<20)
	if err != nil {
		return map[string]any{"available": false}
	}
	var snapshot struct {
		Status   string `json:"status"`
		Observed int    `json:"observed_selected_cells"`
		Posts    int    `json:"laya_posts_attempted"`
		GoCases  int    `json:"compiled_go_cases_observed"`
	}
	if json.Unmarshal(data, &snapshot) != nil {
		return map[string]any{"available": true, "sha256": digest(data), "parseable": false}
	}
	return map[string]any{"sha256": digest(data), "status": snapshot.Status, "observed_cells_at_snapshot": snapshot.Observed, "posts_at_snapshot": snapshot.Posts, "go_cases_at_snapshot": snapshot.GoCases, "interpretation": "This is a preserved progress snapshot; the completed final report and selected-cell journal carry the final outcome."}
}

func summarizeLatency(values []int64) map[string]any {
	ordered := append([]int64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	if len(ordered) == 0 {
		return map[string]any{"count": 0}
	}
	median := (float64(ordered[(len(ordered)-1)/2]) + float64(ordered[len(ordered)/2])) / 2
	upperMedian := float64(ordered[len(ordered)/2])
	nearestRankIndex := int(math.Ceil(float64(len(ordered))*0.95)) - 1
	floorRankIndex := int(math.Floor(float64(len(ordered))*0.95)) - 1
	if nearestRankIndex < 0 {
		nearestRankIndex = 0
	}
	if floorRankIndex < 0 {
		floorRankIndex = 0
	}
	return map[string]any{
		"count": len(ordered), "median_ms_average_middle_pair": median / 1e6,
		"upper_middle_observation_ms": upperMedian / 1e6,
		"p95_floor_n_rank_ms":         float64(ordered[floorRankIndex]) / 1e6,
		"p95_nearest_rank_ms":         float64(ordered[nearestRankIndex]) / 1e6,
		"maximum_ms":                  float64(ordered[len(ordered)-1]) / 1e6,
	}
}

func mustRead(path string, limit int64) []byte {
	data, err := readFile(path, limit)
	if err != nil {
		panic(err)
	}
	return data
}
