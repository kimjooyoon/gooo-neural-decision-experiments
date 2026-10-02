package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type nativeAttempt struct {
	Schema           string                      `json:"schema"`
	Status           string                      `json:"status"`
	Error            string                      `json:"error"`
	Generations      int                         `json:"actual_native_process_calls"`
	Executions       int                         `json:"actual_compile_and_run_calls"`
	Complete         int                         `json:"complete_verified_pairs"`
	Predictions      int                         `json:"recorded_model_predictions"`
	PredictionsKnown bool                        `json:"all_native_model_prediction_counts_known"`
	Prior            int64                       `json:"prior_raw_bytes"`
	New              int64                       `json:"retained_new_bytes_before_attempt"`
	Files            map[string]threestudent.Pin `json:"retained_files_before_attempt"`
	Wall             int64                       `json:"collector_wall_ns"`
	CPU              int64                       `json:"collector_cpu_ns"`
	CPUPercent       float64                     `json:"collector_cpu_percent_of_one_core"`
	RSS              int64                       `json:"collector_lifetime_peak_rss_bytes"`
	Updates          int                         `json:"new_optimizer_updates"`
	Promoted         bool                        `json:"default_model_promoted"`
}
type nativeReport struct {
	Schema      string                  `json:"schema"`
	Status      string                  `json:"status"`
	Source      string                  `json:"source_revision"`
	Native      string                  `json:"native_revision"`
	Calls       int                     `json:"actual_native_generations"`
	Executions  int                     `json:"actual_compiled_go_executions"`
	Invocations int                     `json:"actual_ordered_function_invocations"`
	Predictions int                     `json:"actual_model_predictions"`
	Totals      map[string]nativeTotals `json:"policies"`
	Updates     int                     `json:"new_optimizer_updates"`
	Promoted    bool                    `json:"default_model_promoted"`
	CI          bool                    `json:"ci_hint_supplied"`
	Scope       string                  `json:"scope"`
}

func nativeName(v threecohort.View, p nativePolicy) string {
	return fmt.Sprintf("%s-goal%d-%s-%s", v.Family, v.Goal, v.Language, p.Name)
}
func nativeModelPins(models map[string]*model) map[string]pin {
	pins := map[string]pin{}
	for _, p := range nativePolicies {
		if m := models[p.Candidate]; m != nil {
			pins[p.Candidate] = m.Pin
		}
	}
	return pins
}
func nativeStudy(dataset, modelRoot, prefix, tail, output, revision, binary, goBinary, audit string) (failure error) {
	if audit != "" {
		return auditNativeStudy(dataset, modelRoot, prefix, tail, output, audit)
	}
	if err := source(revision); err != nil {
		return err
	}
	if err := amendment(); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(output)
	if err != nil || filepath.Dir(absolute) != filepath.Join(cwd, "runs") || !strings.HasPrefix(filepath.Base(absolute), "own-three-") {
		return errors.New("fresh own-three native phase directly under runs required")
	}
	if _, err = os.Lstat(absolute); !os.IsNotExist(err) {
		return errors.New("never overwrite or restart retained native phase")
	}
	b, w, g, err := compilerPins(binary, goBinary)
	if err != nil {
		return err
	}
	prior, used, err := inventory()
	if err != nil {
		return err
	}
	store := &storage{Used: used, Prior: used, Cap: continuationCap, FreePath: cwd}
	if err = beforeNativePair(store); err != nil {
		return err
	}
	available, err := freeBytes(cwd)
	if err != nil {
		return err
	}
	start, cpuStart := time.Now(), cpuNS()
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	selected, err := nativeViews(views)
	if err != nil {
		return err
	}
	models, _, err := loadModels(modelRoot)
	if err != nil {
		return err
	}
	if err = os.Mkdir(absolute, 0755); err != nil {
		return err
	}
	calls, executions, complete, predictions := 0, 0, 0, 0
	predictionsKnown := true
	defer func() {
		wall, cpu := time.Since(start).Nanoseconds(), cpuNS()-cpuStart
		status, message := "NATIVE_CAPTURED_PENDING_EXTERNAL_REPLAY", ""
		if failure != nil {
			status, message = "FAILED_NATIVE_PREFIX_RETAINED", failure.Error()
		}
		files, _, e := closedFiles(absolute)
		if e != nil {
			if failure == nil {
				failure = e
			}
			return
		}
		r := nativeAttempt{Schema: "gooo/own-three-native-collection-attempt/v1", Status: status, Error: message, Generations: calls, Executions: executions, Complete: complete, Predictions: predictions, PredictionsKnown: predictionsKnown, Prior: used, New: store.Used - used, Files: files, Wall: wall, CPU: cpu, CPUPercent: 100 * float64(cpu) / float64(wall), RSS: peakRSS()}
		if e = store.save(filepath.Join(absolute, "collection-attempt.json"), r); e != nil && failure == nil {
			failure = e
		}
	}()
	combinedName := filepath.Join(absolute, "combined-sdk-audit.json")
	if err = auditContinuation(dataset, modelRoot, prefix, tail, combinedName); err != nil {
		return err
	}
	combinedPin, err := threestudent.FilePin(combinedName)
	if err != nil {
		return err
	}
	store.Used += combinedPin.Bytes
	var combined struct {
		Status   string `json:"status"`
		Selected string `json:"calibration_selected_candidate"`
		Sessions int    `json:"actual_unique_sdk_sessions"`
	}
	if err = read(combinedName, &combined); err != nil {
		return err
	}
	if combined.Status != "PASS_WITH_SEPARATE_STORAGE_AMENDMENT" || combined.Sessions != 11264 || combined.Selected != nativePolicies[0].Candidate {
		return errors.New("full frozen combined SDK audit and calibration selection required")
	}
	priors, err := sdkNativePriors(prefix, tail, selected)
	if err != nil {
		return err
	}
	ids := make([]string, 0, 128)
	for _, v := range selected {
		ids = append(ids, v.ID)
	}
	pre := nativePre{Schema: "gooo/own-three-native-preexecution/v1", Source: revision, Native: nativeRevision, SDK: nativeSDK, Go: runtime.Version(), Protocol: threefeedback.ProtocolSHA, Amendment: continuationSHA, Dataset: threecohort.DatasetSHA, CombinedAudit: combinedPin, Selected: combined.Selected, Models: nativeModelPins(models), Policies: nativePolicies, IDs: ids, Binary: b, Worker: w, GoBinary: g, Calls: 640, Executions: 640, Invocations: 10240, Prior: prior, PriorBytes: used, Cap: continuationCap, Available: available}
	if err = store.save(filepath.Join(absolute, "preexecution.json"), pre); err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "gooo-three-native-pair-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	accumulators := map[string]*nativeAccumulator{}
	for _, p := range nativePolicies {
		accumulators[p.Name] = &nativeAccumulator{}
	}
	for _, v := range selected {
		doc, src, err := nativeOriginal(v)
		if err != nil {
			return err
		}
		for _, p := range nativePolicies {
			if err = beforeNativePair(store); err != nil {
				return err
			}
			x, raw, n, c, auditView, runErr := runNativePair(store, absolute, workspace, binary, goBinary, modelRoot, v, p, doc, src, models[p.Candidate], priors[p.Candidate][v.ID])
			if x.NativeCalled {
				calls++
			}
			if x.GoCalled {
				executions++
			}
			if c.Schema != "" {
				predictions += c.Search.Selection.ModelCalls
			} else if x.NativeCalled {
				predictionsKnown = false
			}
			if runErr != nil {
				return fmt.Errorf("%s: %w", nativeName(v, p), runErr)
			}
			if err = auditNativeExecution(x, raw, n, v, p); err != nil {
				return err
			}
			if err = accumulators[p.Name].add(auditView, c, n, x); err != nil {
				return err
			}
			complete++
			if complete%40 == 0 {
				fmt.Printf("native verified generation/execution pairs=%d/640 predictions=%d raw=%d\n", complete, predictions, store.Used)
			}
		}
	}
	if calls != 640 || executions != 640 || complete != 640 || !predictionsKnown {
		return errors.New("full native generation/execution denominator differs")
	}
	totals := map[string]nativeTotals{}
	for _, p := range nativePolicies {
		t, err := accumulators[p.Name].finish()
		if err != nil {
			return err
		}
		totals[p.Name] = t
	}
	return store.save(filepath.Join(absolute, "report.json"), nativeReport{Schema: "gooo/own-three-native-report/v1", Status: "PASS", Source: revision, Native: nativeRevision, Calls: calls, Executions: executions, Invocations: complete * 16, Predictions: predictions, Totals: totals, Scope: "Five original policies on all 128 config-20 bilingual source views. Every actual native source/model/progress/frontier and compiled Go output is independently bound to frozen authored contracts and SDK observations. Codegen, internal projection/model setup and compilation/run costs are separate, with fixed numeric arrays. Models rank legal authored alternatives; finite completeness does not establish general natural-language judgment, universal equivalence, simultaneous process-tree RSS, causal host utilization or compiler default promotion."})
}
func beforeNativePair(s *storage) error {
	if err := s.beforeCall(); err != nil {
		return err
	}
	if s.Used+receiptReserve+2*lineCap > s.limit() {
		return errors.New("native capture/execution and terminal space must be reserved before a pair")
	}
	return nil
}
