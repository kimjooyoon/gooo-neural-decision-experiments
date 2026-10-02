package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func auditNativeStudy(dataset, modelsRoot, prefix, tail, output, destination string) error {
	if err := amendment(); err != nil {
		return err
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	selected, err := nativeViews(views)
	if err != nil {
		return err
	}
	models, _, err := loadModels(modelsRoot)
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "gooo-three-native-prior-audit-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	combined := filepath.Join(temp, "combined.json")
	if err = auditContinuation(dataset, modelsRoot, prefix, tail, combined); err != nil {
		return err
	}
	combinedPin, err := threestudent.FilePin(combined)
	if err != nil {
		return err
	}
	savedCombined, err := threestudent.FilePin(filepath.Join(output, "combined-sdk-audit.json"))
	if err != nil || savedCombined != combinedPin {
		return errors.New("complete combined SDK prerequisite bytes differ")
	}
	var prerequisite struct {
		Status   string `json:"status"`
		Sessions int    `json:"actual_unique_sdk_sessions"`
		Selected string `json:"calibration_selected_candidate"`
	}
	if err = read(combined, &prerequisite); err != nil {
		return err
	}
	if prerequisite.Status != "PASS_WITH_SEPARATE_STORAGE_AMENDMENT" || prerequisite.Sessions != 11264 || prerequisite.Selected != nativePolicies[0].Candidate {
		return errors.New("full combined SDK prerequisite required")
	}
	ids := make([]string, 0, 128)
	for _, v := range selected {
		ids = append(ids, v.ID)
	}
	var pre nativePre
	if err = strict(filepath.Join(output, "preexecution.json"), &pre); err != nil {
		return err
	}
	if pre.Schema != "gooo/own-three-native-preexecution/v1" || len(pre.Source) != 40 || pre.Native != nativeRevision || pre.SDK != nativeSDK || pre.Go != "go1.27.1" || pre.Protocol != threefeedback.ProtocolSHA || pre.Amendment != continuationSHA || pre.Dataset != threecohort.DatasetSHA || pre.CombinedAudit != combinedPin || pre.Selected != prerequisite.Selected || !reflect.DeepEqual(pre.Models, nativeModelPins(models)) || pre.Policies != nativePolicies || !reflect.DeepEqual(pre.IDs, ids) || pre.Calls != 640 || pre.Executions != 640 || pre.Invocations != 10240 || pre.Cap != continuationCap || pre.Available < 4<<30 || pre.CI {
		return errors.New("immutable native source/model/policy/view/budget preexecution differs")
	}
	for _, p := range []threestudent.Pin{pre.Binary, pre.Worker, pre.GoBinary} {
		if len(p.SHA) != 64 || strings.Trim(p.SHA, "0123456789abcdef") != "" || p.Bytes <= 0 || p.Bytes > 64<<20 {
			return errors.New("native/worker/Go executable byte pins required")
		}
	}
	priorRoot := filepath.Dir(filepath.Dir(dataset))
	var priorBytes int64
	for name, want := range pre.Prior {
		if !strings.HasPrefix(name, "runs/own-three-") || filepath.Clean(name) != name {
			return errors.New("closed native prior raw identity required")
		}
		actual, e := threestudent.FilePin(filepath.Join(priorRoot, strings.TrimPrefix(name, "runs/")))
		if e != nil || actual != want {
			return errors.New("native prior source/teacher/training/SDK bytes changed")
		}
		priorBytes += actual.Bytes
	}
	if priorBytes != pre.PriorBytes || len(pre.Prior) == 0 {
		return errors.New("native whole-study prior raw denominator differs")
	}
	var attempt nativeAttempt
	if err = strict(filepath.Join(output, "collection-attempt.json"), &attempt); err != nil {
		return err
	}
	full := attempt.Status == "NATIVE_CAPTURED_PENDING_EXTERNAL_REPLAY"
	if attempt.Schema != "gooo/own-three-native-collection-attempt/v1" || (!full && attempt.Status != "FAILED_NATIVE_PREFIX_RETAINED") || (full && attempt.Error != "") || (!full && attempt.Error == "") || attempt.Prior != pre.PriorBytes || attempt.Wall <= 0 || attempt.CPU < 0 || attempt.CPUPercent != 100*float64(attempt.CPU)/float64(attempt.Wall) || attempt.RSS <= 0 || attempt.Updates != 0 || attempt.Promoted {
		return errors.New("native retained terminal/resource boundary differs")
	}
	files, newBytes, err := closedFiles(output)
	if err != nil {
		return err
	}
	if len(files) != len(attempt.Files)+1 || newBytes-files["collection-attempt.json"].Bytes != attempt.New || newBytes+pre.PriorBytes > continuationCap {
		return errors.New("native closed inventory/amended raw cap differs")
	}
	allowed := map[string]bool{"preexecution.json": true, "combined-sdk-audit.json": true, "report.json": true, "collection-attempt.json": true}
	for _, v := range selected {
		for _, p := range nativePolicies {
			name := nativeName(v, p)
			allowed[name+"-native.json"], allowed[name+"-execution.json"] = true, true
		}
	}
	for name, p := range files {
		if !allowed[name] || (name != "collection-attempt.json" && attempt.Files[name] != p) {
			return errors.New("unknown or changed native member")
		}
	}
	priors, err := sdkNativePriors(prefix, tail, selected)
	if err != nil {
		return err
	}
	accumulators := map[string]*nativeAccumulator{}
	for _, p := range nativePolicies {
		accumulators[p.Name] = &nativeAccumulator{}
	}
	calls, executions, complete, predictions := 0, 0, 0, 0
	predictionsKnown, stopped := true, false
	for _, v := range selected {
		doc, src, err := nativeOriginal(v)
		if err != nil {
			return err
		}
		for _, p := range nativePolicies {
			name := nativeName(v, p)
			_, hasRaw := files[name+"-native.json"]
			_, hasExecution := files[name+"-execution.json"]
			if !hasRaw && !hasExecution {
				stopped = true
				continue
			}
			if stopped || !hasRaw || !hasExecution {
				return errors.New("native rows must form complete captured pairs and an ordered retained prefix")
			}
			raw, err := os.ReadFile(filepath.Join(output, name+"-native.json"))
			if err != nil || len(raw) > lineCap {
				return errors.New("bounded complete native output required")
			}
			var x nativeExecution
			if err = strict(filepath.Join(output, name+"-execution.json"), &x); err != nil {
				return err
			}
			if x.Schema != "gooo/own-three-native-execution/v1" || x.View != v.ID || x.Policy != p || x.Capture != threecohort.SHA(raw) || x.NativeCalled != x.Codegen.Started || x.GoCalled != x.Execution.Started {
				return errors.New("native process/identity/capture binding differs")
			}
			if x.NativeCalled {
				calls++
			}
			if x.GoCalled {
				executions++
			}
			n, c, auditView, inspectErr := inspectNative(raw, v, doc, src, models[p.Candidate], x.Codegen.Wall)
			if c.Schema != "" {
				predictions += c.Search.Selection.ModelCalls
			} else if x.NativeCalled {
				predictionsKnown = false
			}
			if x.Error != "" || !x.Executed {
				if full {
					return errors.New("full native report contains a failed pair")
				}
				stopped = true
				continue
			}
			if inspectErr != nil {
				return inspectErr
			}
			if err = compareSDK(c, priors[p.Candidate][v.ID]); err != nil {
				return err
			}
			if err = auditNativeExecution(x, raw, n, v, p); err != nil {
				return err
			}
			if err = accumulators[p.Name].add(auditView, c, n, x); err != nil {
				return err
			}
			complete++
		}
	}
	if calls != attempt.Generations || executions != attempt.Executions || complete != attempt.Complete || predictions != attempt.Predictions || predictionsKnown != attempt.PredictionsKnown {
		return errors.New("actual retained native/Go/prediction denominators differ")
	}
	status := "NATIVE_PREFIX_VERIFIED_INCOMPLETE"
	totals := map[string]nativeTotals{}
	if full {
		if calls != 640 || executions != 640 || complete != 640 || !predictionsKnown {
			return errors.New("full native six-hundred-forty pair denominator differs")
		}
		for _, p := range nativePolicies {
			t, err := accumulators[p.Name].finish()
			if err != nil {
				return err
			}
			totals[p.Name] = t
		}
		var r nativeReport
		if err = strict(filepath.Join(output, "report.json"), &r); err != nil {
			return err
		}
		if r.Schema != "gooo/own-three-native-report/v1" || r.Status != "PASS" || r.Source != pre.Source || r.Native != nativeRevision || r.Calls != calls || r.Executions != executions || r.Invocations != 10240 || r.Predictions != predictions || !reflect.DeepEqual(r.Totals, totals) || r.Updates != 0 || r.Promoted || r.CI {
			return errors.New("native actual complete summary differs from independent reconstruction")
		}
		status = "PASS"
	} else if _, ok := files["report.json"]; ok {
		return errors.New("failed native phase cannot carry a full report")
	}
	return (&storage{}).save(destination, map[string]any{"schema": "gooo/own-three-native-independent-audit/v1", "status": status, "producer_source_revision": pre.Source, "native_revision": nativeRevision, "sdk": nativeSDK, "source_dataset_sha256": threecohort.DatasetSHA, "actual_native_generations_audited": calls, "actual_compile_and_run_calls_audited": executions, "actual_verified_compiled_go_executions": complete, "actual_independently_verified_ordered_invocations": complete * 16, "recorded_actual_model_predictions": predictions, "all_native_model_prediction_counts_known": predictionsKnown, "policies": totals, "native_phase_files": files, "prior_raw_bytes": pre.PriorBytes, "native_phase_bytes": newBytes, "total_retained_raw_bytes": pre.PriorBytes + newBytes, "amended_raw_cap_bytes": continuationCap, "collector_wall_ns": attempt.Wall, "collector_cpu_ns": attempt.CPU, "collector_cpu_percent_of_one_core": attempt.CPUPercent, "collector_lifetime_peak_rss_bytes": attempt.RSS, "new_native_calls": 0, "new_go_execution_calls": 0, "new_model_predictions": 0, "new_optimizer_updates": 0, "default_model_promoted": false, "scope": "Every completed native/Go pair is source/model/document/input/case bound, independently checks full progress/frontier/failure arithmetic, matches frozen SDK paths and records actual generated-Go stdout. Original SDK storage failure remains preserved. Missing/failed native prefixes are not full completeness. Child RSS is measured usage, not simultaneous tree RAM; compile/run costs retain ordinary Go cache behavior. All replay arithmetic and summaries use recorded observations, with zero native, Go or model operational calls."})
}
