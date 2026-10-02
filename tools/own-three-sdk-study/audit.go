package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func strict(name string, value any) error {
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	return threecohort.Decode(raw, value)
}
func auditStudy(dataset, root, output, destination string) error {
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	models, ids, err := loadModels(root)
	if err != nil {
		return err
	}
	pins := map[string]pin{}
	for id, m := range models {
		if m != nil {
			pins[id] = m.Pin
		}
	}
	var pre preexecution
	if err = strict(filepath.Join(output, "preexecution.json"), &pre); err != nil {
		return err
	}
	if pre.Schema != "gooo/own-three-sdk-preexecution/v1" || len(pre.Source) != 40 || pre.Go != "go1.27.1" || pre.Protocol != threefeedback.ProtocolSHA || pre.Dataset != threecohort.DatasetSHA || !reflect.DeepEqual(pre.Models, pins) || !reflect.DeepEqual(pre.Policies, ids) || pre.RawCap != threestudent.RawCap || pre.Sessions != 11264 || pre.Views != 512 || pre.Budget != 8 || pre.Step != 1 || pre.Rounds != 7 || pre.Seed != "" || pre.CI || pre.Selector != "calibration extra candidates, actual predictions, bilingual disagreement, packed bytes, candidate name; offline excluded; no development selection" {
		return errors.New("frozen SDK preexecution differs")
	}
	var attempt struct {
		Schema      string                      `json:"schema"`
		Status      string                      `json:"status"`
		Error       string                      `json:"error"`
		Sessions    int                         `json:"actual_sdk_sessions"`
		Predictions int                         `json:"actual_model_predictions"`
		Prior       int64                       `json:"prior_raw_bytes"`
		New         int64                       `json:"retained_new_bytes_before_attempt"`
		Files       map[string]threestudent.Pin `json:"retained_files_before_attempt"`
		Updates     int                         `json:"new_optimizer_updates"`
		Native      int                         `json:"native_calls"`
	}
	if err = strict(filepath.Join(output, "collection-attempt.json"), &attempt); err != nil {
		return err
	}
	full := attempt.Status == "SDK_CAPTURED_PENDING_EXTERNAL_REPLAY"
	if attempt.Schema != "gooo/own-three-sdk-collection-attempt/v1" || (!full && attempt.Status != "FAILED_PREFIX_RETAINED") || (full && attempt.Error != "") || (!full && attempt.Error == "") || attempt.Prior != pre.PriorBytes || attempt.Updates != 0 || attempt.Native != 0 {
		return errors.New("terminal retained collection attempt required")
	}
	allowed := map[string]bool{"preexecution.json": true, "collection-attempt.json": true, "selection.json": true, "report.json": true}
	for _, split := range []string{"calibration", "development"} {
		for _, id := range ids {
			allowed[filename(split, id)] = true
		}
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		return err
	}
	var newBytes int64
	var attemptBytes int64
	if len(entries) != len(attempt.Files)+1 {
		return errors.New("closed retained phase inventory differs")
	}
	for _, d := range entries {
		if d.IsDir() || !allowed[d.Name()] {
			return errors.New("unknown or nonregular phase member")
		}
		p, e := threestudent.FilePin(filepath.Join(output, d.Name()))
		if e != nil {
			return e
		}
		newBytes += p.Bytes
		if d.Name() == "collection-attempt.json" {
			attemptBytes = p.Bytes
		}
		if d.Name() != "collection-attempt.json" && p != attempt.Files[d.Name()] {
			return errors.New("retained actual evidence bytes differ")
		}
	}
	if newBytes-attemptBytes != attempt.New || newBytes+pre.PriorBytes > threestudent.RawCap {
		return errors.New("whole-study raw cap violated")
	}
	var priorBytes int64
	priorRoot := filepath.Dir(filepath.Dir(dataset))
	for name, want := range pre.PriorFiles {
		if !strings.HasPrefix(name, "runs/own-three-") || filepath.Clean(name) != name {
			return errors.New("closed prior raw identity required")
		}
		p, e := threestudent.FilePin(filepath.Join(priorRoot, strings.TrimPrefix(name, "runs/")))
		if e != nil || p != want {
			return errors.New("prior retained raw evidence changed")
		}
		priorBytes += p.Bytes
	}
	if priorBytes != pre.PriorBytes || len(pre.PriorFiles) == 0 {
		return errors.New("prior raw denominator differs")
	}
	all := map[string]map[string]counts{"calibration": {}, "development": {}}
	sessions, predictions, values := 0, 0, 0
	stopped := false
	for _, split := range []string{"calibration", "development"} {
		for _, id := range ids {
			name := filepath.Join(output, filename(split, id))
			f, e := os.Open(name)
			if os.IsNotExist(e) {
				stopped = true
				continue
			}
			if e != nil {
				return e
			}
			if stopped {
				f.Close()
				return errors.New("retained session cells are not an ordered prefix")
			}
			a := accumulator{}
			scan := bufio.NewScanner(f)
			scan.Buffer(make([]byte, 32768), lineCap)
			for _, v := range views {
				if v.Split != split {
					continue
				}
				if !scan.Scan() {
					break
				}
				var o observation
				if e = threecohort.Decode(scan.Bytes(), &o); e != nil {
					f.Close()
					return e
				}
				if o.Policy != id {
					f.Close()
					return errors.New("ordered actual policy differs")
				}
				if e = verify(v, o.Capture, models[id]); e != nil {
					f.Close()
					return fmt.Errorf("%s %s %s: %w", split, id, v.ID, e)
				}
				if e = a.add(v, o.Capture); e != nil {
					f.Close()
					return e
				}
				sessions++
				predictions += o.Capture.Search.Selection.ModelCalls
				values += 16 * len(o.Capture.Search.Attempts)
			}
			if scan.Scan() {
				f.Close()
				return errors.New("extra captured view outside closed cell")
			}
			if e = scan.Err(); e != nil {
				f.Close()
				return e
			}
			if e = f.Close(); e != nil {
				return e
			}
			if a.Counts.Views == 512 {
				cell, e := a.finish()
				if e != nil {
					return e
				}
				all[split][id] = cell
			} else {
				if full {
					return errors.New("complete study contains incomplete cell")
				}
				stopped = true
			}
		}
	}
	if sessions != attempt.Sessions || predictions != attempt.Predictions {
		return errors.New("actual prefix counts differ from immutable journal")
	}
	selected := ""
	if len(all["calibration"]) == 11 {
		selected = choose(all["calibration"], models)
		var s selection
		if err = strict(filepath.Join(output, "selection.json"), &s); err != nil {
			return err
		}
		if s.Schema != "gooo/own-three-sdk-calibration-selection/v1" || s.Selected != selected || !reflect.DeepEqual(s.Calibration, all["calibration"]) || !reflect.DeepEqual(s.Models, pins) || s.DevelopmentSeen {
			return errors.New("calibration-only selector differs")
		}
	} else if _, ok := attempt.Files["selection.json"]; ok {
		return errors.New("selector written before calibration complete")
	}
	status := "PREFIX_VERIFIED_INCOMPLETE"
	if full {
		var r report
		if err = strict(filepath.Join(output, "report.json"), &r); err != nil {
			return err
		}
		if sessions != 11264 || len(all["development"]) != 11 || r.Schema != "gooo/own-three-sdk-report/v1" || r.Status != "PASS" || r.Selected != selected || r.Sessions != sessions || r.Predictions != predictions || !reflect.DeepEqual(r.Calibration, all["calibration"]) || !reflect.DeepEqual(r.Development, all["development"]) || r.NewUpdates != 0 || r.Native != 0 || r.Promoted || r.WallNS <= 0 || r.CPUNS < 0 || r.CPUPercent != 100*float64(r.CPUNS)/float64(r.WallNS) || r.RSS < 0 || len(r.ModelSetup) != 10 {
			return errors.New("complete captured summary differs from independently audited observations")
		}
		for id := range pins {
			if r.ModelSetup[id] <= 0 {
				return errors.New("actual model setup time missing")
			}
		}
		status = "PASS"
	} else if _, ok := attempt.Files["report.json"]; ok {
		return errors.New("failed prefix cannot carry a full report")
	}
	return (&storage{}).save(destination, map[string]any{"schema": "gooo/own-three-sdk-independent-audit/v1", "status": status, "producer_source_revision": pre.Source, "actual_captured_sdk_sessions": sessions, "actual_captured_model_predictions": predictions, "actual_independently_verified_ordered_values": values, "calibration_selected_candidate": selected, "complete_calibration_cells": len(all["calibration"]), "complete_development_cells": len(all["development"]), "prior_retained_raw_bytes": pre.PriorBytes, "retained_phase_bytes": newBytes, "raw_cap_bytes": threestudent.RawCap, "new_model_predictions": 0, "new_optimizer_updates": 0, "native_calls": 0, "default_model_promoted": false, "scope": "All retained rows are decoded strictly; every actual source, ordered value, full failed context, fixed/unnecessary/declined ranking, observed probability and committed frontier/progress chain is independently reconciled without running a model. A verified incomplete prefix is not full eleven-policy comparison."})
}
