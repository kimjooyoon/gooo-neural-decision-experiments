package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
)

type executionCapture struct {
	Capture  string  `json:"capture_sha256"`
	Source   string  `json:"emitted_go_sha256"`
	Executed bool    `json:"actual_compiled_go_execution"`
	Values   []int64 `json:"ordered_actual_values"`
	Cases    int     `json:"case_denominator"`
	Metrics  metrics `json:"process_metrics"`
}

func auditExecution(x executionCapture, raw []byte, n nativeResult, doc document) error {
	if x.Capture != hash(raw) || x.Source != hash([]byte(n.Source)) || !x.Executed || x.Cases != 16 || len(x.Values) != 16 || len(doc.Cases) != 16 {
		return errors.New("actual Go execution binding differs")
	}
	for i, value := range x.Values {
		if value != doc.Cases[i].Expected {
			return errors.New("actual Go independent oracle differs")
		}
	}
	if x.Metrics.Wall <= 0 || x.Metrics.User < 0 || x.Metrics.System < 0 || x.Metrics.RSS <= 0 || x.Metrics.CPU != 100*float64(x.Metrics.User+x.Metrics.System)/float64(x.Metrics.Wall) {
		return errors.New("process CPU/RSS measurement differs")
	}
	return nil
}

func auditNative(curriculum, models, study, native, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh native audit output required")
	}
	views, err := loadViews(curriculum)
	if err != nil {
		return err
	}
	loaded, pins, err := loadModels(models)
	if err != nil {
		return err
	}
	var sdk struct {
		Status      string           `json:"status"`
		Selected    string           `json:"selected_candidate"`
		Calibration map[string]score `json:"calibration"`
	}
	if err = decodeFile(filepath.Join(study, "report.json"), &sdk); err != nil || sdk.Status != "PASS" {
		return errors.New("frozen SDK evidence required")
	}
	chosen, err := choose(sdk.Calibration, pins)
	if err != nil || chosen != sdk.Selected {
		return errors.New("calibration-only native selector differs")
	}
	var pre struct {
		Native      string              `json:"native_revision"`
		Dataset     string              `json:"dataset_sha256"`
		Selected    string              `json:"selected_candidate"`
		SDK         string              `json:"sdk"`
		Pins        map[string]modelPin `json:"model_pins"`
		Selection   string              `json:"selection_sha256"`
		Calls       int                 `json:"planned_native_calls"`
		Executions  int                 `json:"planned_independently_compiled_go_executions"`
		Invocations int                 `json:"planned_ordered_go_function_invocations"`
	}
	if err = decodeFile(filepath.Join(native, "preexecution.json"), &pre); err != nil {
		return err
	}
	selectionSHA, err := fileHash(filepath.Join(study, "selection.json"))
	if err != nil || pre.Native != nativeDeployed || pre.Native == "" || pre.SDK != "v0.2.12-experimental" || pre.Dataset != datasetSHA || pre.Selected != chosen || !reflect.DeepEqual(pre.Pins, pins) || pre.Selection != selectionSHA || pre.Calls != 192 || pre.Executions != 192 || pre.Invocations != 3072 {
		return errors.New("native preexecution source/model/denominators differ")
	}
	var report struct {
		Status      string               `json:"status"`
		Native      string               `json:"native_revision"`
		Calls       int                  `json:"actual_native_calls"`
		Predictions int                  `json:"actual_model_predictions"`
		Executions  int                  `json:"independently_compiled_go_executions"`
		Invocations int                  `json:"ordered_go_function_invocations"`
		Totals      map[string]counts    `json:"totals"`
		Metrics     map[string][]metrics `json:"native_process_measurements"`
	}
	if err = decodeFile(filepath.Join(native, "report.json"), &report); err != nil || report.Status != "PASS" || report.Native != nativeDeployed || report.Calls != 192 || report.Executions != 192 || report.Invocations != 3072 {
		return errors.New("completed native evidence required")
	}
	ids := [4]string{chosen, "independent-fp32", "joint-fp32", "offline"}
	policies := [4]string{"selected", "independent", "joint", "offline"}
	priors := map[string]map[string]observation{}
	for _, id := range ids {
		priors[id], err = priorObservations(study, "development", id)
		if err != nil {
			return err
		}
	}
	totals := map[string]counts{}
	pairs := map[string]map[string]map[string]uint16{}
	calls, predictions := 0, 0
	for _, family := range jointcompositionstudy.Families {
		for goal := 0; goal < 4; goal++ {
			for _, language := range []string{"en", "ko"} {
				for i, id := range ids {
					var v view
					for _, candidate := range views {
						r := candidate.Rows[0]
						if r.Family == family && r.Config == 40 && r.Desired == goal && r.Language == language {
							v = candidate
							break
						}
					}
					if v.ID == "" {
						return errors.New("frozen native subset missing")
					}
					doc, source, e := originalDocument(v)
					if e != nil {
						return e
					}
					name := fmt.Sprintf("%s-goal%d-%s-%s", family, goal, language, policies[i])
					raw, e := os.ReadFile(filepath.Join(native, name+".json"))
					if e != nil {
						return e
					}
					n, o, e := inspectNative(raw, v, doc, source, loaded[id])
					if e != nil {
						return e
					}
					if e = auditModelBinding(v, o, loaded[id], pins[id]); e != nil {
						return e
					}
					prior := priors[id][v.ID]
					if !reflect.DeepEqual(o.Search.Attempts, prior.Search.Attempts) || o.Search.Selection.ModelCalls != prior.Search.Selection.ModelCalls || !reflect.DeepEqual(o.Search.InitialProposals, prior.Search.InitialProposals) {
						return errors.New("native/SDK candidate path differs")
					}
					var x executionCapture
					if e = decodeFile(filepath.Join(native, name+"-execution.json"), &x); e != nil {
						return e
					}
					if e = auditExecution(x, raw, n, doc); e != nil {
						return e
					}
					policy := policies[i]
					c := totals[policy]
					if c.Views >= len(report.Metrics[policy]) || x.Metrics != report.Metrics[policy][c.Views] {
						return errors.New("native metric capture differs")
					}
					add(&c, o)
					if e = addTarget(&c, o, v.Rows[0].Finite.Joint); e != nil {
						return e
					}
					totals[policy] = c
					if pairs[policy] == nil {
						pairs[policy] = map[string]map[string]uint16{}
					}
					key := fmt.Sprintf("%s/%d", family, goal)
					if pairs[policy][key] == nil {
						pairs[policy][key] = map[string]uint16{}
					}
					pairs[policy][key][language] = o.Search.Attempts[0].Mask
					calls++
					predictions += o.Search.Selection.ModelCalls
				}
			}
		}
	}
	for policy, p := range pairs {
		c := totals[policy]
		for _, lang := range p {
			if len(lang) != 2 {
				return errors.New("language pair missing")
			}
			c.Pairs++
			if lang["en"] != lang["ko"] {
				c.Different++
			}
		}
		totals[policy] = c
		if c.Views != 48 || c.Pairs != 24 || len(report.Metrics[policy]) != 48 {
			return errors.New("native policy denominator differs")
		}
	}
	if calls != 192 || predictions != report.Predictions || !sameCountMaps(totals, report.Totals) {
		return errors.New("native aggregate reconstruction differs")
	}
	var rounding []roundingDifference
	for _, policy := range policies {
		rounding = appendRounding(rounding, "native", policy, totals[policy], report.Totals[policy])
	}
	return save(output, map[string]any{"schema": "gooo/joint-composition-native-audit/v1", "status": "PASS", "dataset_sha256": datasetSHA, "native_revision": nativeDeployed, "native_captures": calls, "independently_executed_go_captures": calls, "ordered_go_function_invocations": calls * 16, "recorded_native_model_predictions": predictions, "selected_candidate": chosen, "reconstructed_native_totals": totals, "new_native_calls": 0, "new_model_predictions": 0, "new_optimizer_updates": 0, "floating_aggregate_rounding_differences": rounding, "floating_aggregate_tolerance": "1e-10 absolute + 1e-12 relative for rederived target NLL/mass only", "scope": "Source/model/intent hashes, independent int64 arithmetic, all ordered cases, actual compiled Go values, progress/feedback chains, four policies and bilingual pairs. Selected and independent FP32 are explicit replays of the same model; they are not separate intentions."})
}
