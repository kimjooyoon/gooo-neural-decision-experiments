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

func validateTailPre(pre tailPreexecution, original map[string]threestudent.Pin, missing []missingIdentity, pins map[string]pin, audit threestudent.Pin, phase string) error {
	if pre.Schema != "gooo/own-three-sdk-tail-preexecution/v1" || len(pre.Source) != 40 || pre.Go != "go1.27.1" || pre.Protocol != threefeedback.ProtocolSHA || pre.Amendment != continuationSHA || pre.Dataset != threecohort.DatasetSHA || pre.OriginalPhase != phase || !reflect.DeepEqual(pre.OriginalFiles, original) || pre.OriginalAudit != audit || !reflect.DeepEqual(pre.Models, pins) || pre.Selected != "set-feedback/fp32" || pre.RawCap != continuationCap || pre.Available < 4<<30 || !reflect.DeepEqual(pre.Missing, missing) || pre.Budget != 8 || pre.Step != 1 || pre.Rounds != 7 || pre.Seed != "" || pre.CI {
		return errors.New("separate continuation changed original identities, models, bounds or selection")
	}
	return nil
}
func auditContinuation(dataset, modelRoot, prefix, output, destination string) error {
	if err := amendment(); err != nil {
		return err
	}
	original, err := originalPins(prefix)
	if err != nil {
		return err
	}
	// Replay the original independently again. Temporary audit output never
	// modifies either phase or supplies feedback to an operational model.
	temp, err := os.MkdirTemp("", "gooo-three-prefix-replay-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	auditName := filepath.Join(temp, "original-audit.json")
	if err = auditStudy(dataset, modelRoot, prefix, auditName); err != nil {
		return err
	}
	originalAudit, err := threestudent.FilePin(auditName)
	if err != nil {
		return err
	}
	retainedAudit, err := threestudent.FilePin(filepath.Join(output, "original-prefix-audit.json"))
	if err != nil || originalAudit != retainedAudit {
		return errors.New("original independent prefix replay differs")
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	models, ids, err := loadModels(modelRoot)
	if err != nil {
		return err
	}
	pins := map[string]pin{}
	for id, m := range models {
		if m != nil {
			pins[id] = m.Pin
		}
	}
	missing, err := originalMissing(prefix, views, ids, models)
	if err != nil {
		return err
	}
	var pre tailPreexecution
	if err = strict(filepath.Join(output, "preexecution.json"), &pre); err != nil {
		return err
	}
	if err = validateTailPre(pre, original, missing, pins, originalAudit, filepath.Base(prefix)); err != nil {
		return err
	}
	priorRoot := filepath.Dir(filepath.Dir(dataset))
	var priorBytes int64
	for name, want := range pre.PriorFiles {
		if !strings.HasPrefix(name, "runs/own-three-") || filepath.Clean(name) != name {
			return errors.New("closed prior raw identity required")
		}
		actual, e := threestudent.FilePin(filepath.Join(priorRoot, strings.TrimPrefix(name, "runs/")))
		if e != nil || actual != want {
			return errors.New("prior original raw bytes changed")
		}
		priorBytes += actual.Bytes
	}
	if priorBytes != pre.PriorBytes || len(pre.PriorFiles) == 0 {
		return errors.New("continuation prior raw denominator differs")
	}
	for name, p := range original {
		if pre.PriorFiles["runs/"+pre.OriginalPhase+"/"+name] != p {
			return errors.New("whole original failed phase must be counted in amended raw budget")
		}
	}
	var attempt tailAttempt
	if err = strict(filepath.Join(output, "collection-attempt.json"), &attempt); err != nil {
		return err
	}
	full := attempt.Status == "SDK_TAIL_CAPTURED_PENDING_EXTERNAL_REPLAY"
	if attempt.Schema != "gooo/own-three-sdk-tail-attempt/v1" || (!full && attempt.Status != "FAILED_TAIL_PREFIX_RETAINED") || (full && attempt.Error != "") || (!full && attempt.Error == "") || attempt.Prior != pre.PriorBytes || attempt.Updates != 0 || attempt.Native != 0 || attempt.Promoted || attempt.WallNS <= 0 || attempt.CPUNS < 0 || attempt.CPUPercent != 100*float64(attempt.CPUNS)/float64(attempt.WallNS) || attempt.RSS <= 0 || len(attempt.Setup) != 10 {
		return errors.New("retained tail terminal status and resource arithmetic required")
	}
	for id := range pins {
		if attempt.Setup[id] <= 0 {
			return errors.New("tail actual setup measurement missing")
		}
	}
	files, newBytes, err := closedFiles(output)
	if err != nil {
		return err
	}
	if len(files) != len(attempt.Files)+1 || newBytes-files["collection-attempt.json"].Bytes != attempt.New || newBytes+pre.PriorBytes > continuationCap {
		return errors.New("closed tail inventory or amended whole-study raw cap differs")
	}
	allowed := map[string]bool{"collection-attempt.json": true, "preexecution.json": true, "original-prefix-audit.json": true}
	for _, item := range missing {
		allowed[filename("development", item.Policy)] = true
	}
	for name, p := range files {
		if !allowed[name] || (name != "collection-attempt.json" && attempt.Files[name] != p) {
			return errors.New("unknown or changed retained tail member")
		}
	}
	all := map[string]map[string]counts{"calibration": {}, "development": {}}
	originalSessions, originalPredictions, tailSessions, tailPredictions, totalValues := 0, 0, 0, 0, 0
	tailStopped := false
	for _, split := range []string{"calibration", "development"} {
		cell := splitViews(views, split)
		for _, id := range ids {
			a := accumulator{}
			n, e := cellRows(filepath.Join(prefix, filename(split, id)), id, cell, 0, &a, models[id], true)
			if e != nil {
				return e
			}
			originalSessions += n
			originalPredictions += a.Counts.Predictions
			if split == "development" && n < 512 {
				before := a.Counts.Predictions
				name := filename(split, id)
				if tailStopped {
					if _, ok := files[name]; ok {
						return errors.New("retained tail is not an ordered missing-only prefix")
					}
				}
				added, e := cellRows(filepath.Join(output, name), id, cell, n, &a, models[id], true)
				if e != nil {
					return e
				}
				tailSessions += added
				tailPredictions += a.Counts.Predictions - before
				if n+added < 512 {
					tailStopped = true
				}
			}
			totalValues += a.Counts.ActualValues
			if a.Counts.Views == 512 {
				complete, e := a.finish()
				if e != nil {
					return e
				}
				all[split][id] = complete
			} else if full {
				return errors.New("captured full continuation has missing original identities")
			}
		}
	}
	if originalSessions != 8779 || originalPredictions != 33388 || tailSessions != attempt.Sessions || tailPredictions != attempt.Predictions {
		return errors.New("actual original/tail denominators differ from immutable journals")
	}
	selected := choose(all["calibration"], models)
	if len(all["calibration"]) != 11 || selected != pre.Selected {
		return errors.New("original calibration selector changed")
	}
	status := "COMBINED_PREFIX_VERIFIED_INCOMPLETE"
	if full {
		if tailSessions != continuationSessions || originalSessions+tailSessions != 11264 || len(all["development"]) != 11 {
			return errors.New("complete amended eleven-policy comparison denominator differs")
		}
		status = "PASS_WITH_SEPARATE_STORAGE_AMENDMENT"
	}
	return (&storage{}).save(destination, map[string]any{
		"schema": "gooo/own-three-sdk-combined-independent-audit/v1", "status": status,
		"original_producer_source_revision": originalProducer, "tail_producer_source_revision": pre.Source,
		"original_status": "FAILED_PREFIX_RETAINED", "original_raw_cap_bytes": threestudent.RawCap,
		"storage_continuation_sha256": continuationSHA, "amended_whole_study_raw_cap_bytes": continuationCap,
		"original_captured_sdk_sessions": originalSessions, "tail_captured_sdk_sessions": tailSessions,
		"actual_unique_sdk_sessions": originalSessions + tailSessions, "original_model_predictions": originalPredictions,
		"tail_model_predictions": tailPredictions, "actual_captured_model_predictions": originalPredictions + tailPredictions,
		"actual_independently_verified_ordered_values": totalValues,
		"calibration_selected_candidate":               selected, "complete_calibration_cells": len(all["calibration"]),
		"complete_development_cells": len(all["development"]), "calibration": all["calibration"], "development": all["development"],
		"original_collector_resource_metrics_available": false, "tail_collector_wall_ns": attempt.WallNS,
		"tail_collector_cpu_ns": attempt.CPUNS, "tail_cpu_percent_of_one_core": attempt.CPUPercent,
		"tail_process_lifetime_peak_rss_bytes": attempt.RSS, "prior_retained_raw_bytes": pre.PriorBytes,
		"retained_tail_bytes": newBytes, "total_retained_raw_bytes": pre.PriorBytes + newBytes,
		"tail_phase_files": files, "original_phase_files": original, "unchanged_model_pins": pins,
		"new_model_predictions": 0, "new_optimizer_updates": 0, "native_calls": 0, "default_model_promoted": false,
		"scope": "All original rows and only exact missing development rows reconciled in original policy/view order, with independent full source/arithmetic/progress/context/frontier verification. No repeated operational original calls. Original 768-MiB failure remains a failure; original CPU/RSS remain unavailable. Full amended finite known-cohort completeness is not open-language generalization or compiler default promotion.",
	})
}
