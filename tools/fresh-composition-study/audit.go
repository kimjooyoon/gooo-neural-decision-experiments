package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

func receiptLinks(o observation) error {
	previous := ""
	for i, p := range o.Progress {
		if p.Sequence != i+1 || p.PreviousSHA != previous || p.SHA == "" {
			return errors.New("progress chain differs")
		}
		previous = p.SHA
		p.SHA = ""
		raw, err := json.Marshal(p)
		if err != nil || hash(raw) != previous {
			return errors.New("progress receipt digest differs")
		}
	}
	previous = ""
	for i, f := range o.Feedback {
		if f.Round != i+1 || f.PreviousSHA != previous || f.SHA == "" {
			return errors.New("feedback chain differs")
		}
		previous = f.SHA
		f.SHA = ""
		raw, err := json.Marshal(f)
		if err != nil || hash(raw) != previous {
			return errors.New("feedback receipt digest differs")
		}
	}
	return nil
}

func summarizeFrozen(views []view, split, id, study string, model *decision.Model) (score, int, error) {
	observations, err := priorObservations(study, split, id)
	if err != nil {
		return score{}, 0, err
	}
	feature := decision.SplitContextIntentFeatureVersion
	if model != nil {
		feature = model.FeatureVersion()
	}
	result := score{Families: map[string]counts{}}
	initial := map[string]map[string]uint16{}
	for _, v := range views {
		r := v.Rows[0]
		if r.Split != split || r.Feature != feature {
			continue
		}
		o, ok := observations[v.ID]
		if !ok || o.Inputs != [2]string{v.Rows[0].Input.SHA, v.Rows[1].Input.SHA} {
			return result, 0, errors.New("frozen input binding differs")
		}
		if err = validateObservation(v, o, model != nil); err != nil {
			return result, 0, err
		}
		if err = receiptLinks(o); err != nil {
			return result, 0, err
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
			return result, 0, errors.New("frozen bilingual pair missing")
		}
		result.Total.Pairs++
		if pair["en"] != pair["ko"] {
			result.Total.Different++
		}
	}
	if result.Total.Views != 384 || result.Total.Pairs != 192 {
		return result, 0, errors.New("frozen view count differs")
	}
	return result, len(observations), nil
}

func audit(curriculum, models, study, native, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh audit report required")
	}
	views, err := loadViews(curriculum)
	if err != nil {
		return err
	}
	loaded, pins, err := loadModels(models)
	if err != nil {
		return err
	}
	var report struct {
		Status      string           `json:"status"`
		Selected    string           `json:"selected_candidate"`
		Calibration map[string]score `json:"calibration"`
		Development map[string]score `json:"development"`
	}
	if err = decodeFile(filepath.Join(study, "report.json"), &report); err != nil {
		return err
	}
	if report.Status != "PASS" {
		return errors.New("completed frozen SDK study required")
	}
	records := 0
	for _, split := range []string{"calibration", "development"} {
		scores := report.Calibration
		if split == "development" {
			scores = report.Development
		}
		for _, id := range []string{"v2-fp32", "v2-ptq_ternary", "v2-qat_ternary", "v3-fp32", "v3-ptq_ternary", "v3-qat_ternary", "offline"} {
			actual, n, e := summarizeFrozen(views, split, id, study, loaded[id])
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(actual, scores[id]) {
				return errors.New("independently accumulated SDK score differs")
			}
			records += n
		}
	}
	chosen, err := choose(report.Calibration, pins)
	if err != nil || chosen != report.Selected {
		return errors.New("calibration selector reconstruction differs")
	}
	var nativeReport struct {
		Status      string               `json:"status"`
		Calls       int                  `json:"actual_native_calls"`
		Predictions int                  `json:"actual_model_predictions"`
		Executions  int                  `json:"independently_compiled_go_executions"`
		Totals      map[string]counts    `json:"totals"`
		Metrics     map[string][]metrics `json:"native_process_measurements"`
	}
	if err = decodeFile(filepath.Join(native, "report.json"), &nativeReport); err != nil {
		return err
	}
	if nativeReport.Status != "PASS" || nativeReport.Calls != 144 || nativeReport.Executions != 144 {
		return errors.New("completed frozen native study required")
	}
	ids := [3]string{chosen, "v2-fp32", "offline"}
	policies := [3]string{"selected", "reference", "offline"}
	corrected := map[string]counts{}
	pairs := map[string]map[string]map[string]uint16{}
	calls, predictions, executions := 0, 0, 0
	for _, family := range []string{"reference_operand", "assignment_branch", "predicate_branch", "assignment_schedule", "reference_schedule", "branch_schedule"} {
		for goal := 0; goal < 4; goal++ {
			for _, language := range []string{"en", "ko"} {
				for i, id := range ids {
					feature := decision.SplitContextIntentFeatureVersion
					if loaded[id] != nil {
						feature = loaded[id].FeatureVersion()
					}
					var v view
					for _, candidate := range views {
						r := candidate.Rows[0]
						if r.Family == family && r.Config == 40 && r.Desired == goal && r.Language == language && r.Feature == feature {
							v = candidate
							break
						}
					}
					if v.ID == "" {
						return errors.New("frozen native view missing")
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
					if e = receiptLinks(o); e != nil {
						return e
					}
					var execution struct {
						Capture  string  `json:"capture_sha256"`
						Source   string  `json:"emitted_go_sha256"`
						Executed bool    `json:"actual_compiled_go_execution"`
						Values   []int64 `json:"ordered_actual_values"`
						Cases    int     `json:"case_denominator"`
						Metrics  metrics `json:"process_metrics"`
					}
					if e = decodeFile(filepath.Join(native, name+"-execution.json"), &execution); e != nil {
						return e
					}
					if execution.Capture != hash(raw) || execution.Source != hash([]byte(n.Source)) || !execution.Executed || execution.Cases != 16 || len(execution.Values) != 16 {
						return errors.New("actual execution capture binding differs")
					}
					for j, x := range execution.Values {
						if x != doc.Cases[j].Expected {
							return errors.New("captured actual-Go oracle differs")
						}
					}
					if execution.Metrics != nativeReport.Metrics[policies[i]][corrected[policies[i]].Views] {
						return errors.New("native process metric capture differs")
					}
					c := corrected[policies[i]]
					add(&c, o)
					corrected[policies[i]] = c
					if pairs[policies[i]] == nil {
						pairs[policies[i]] = map[string]map[string]uint16{}
					}
					key := fmt.Sprintf("%s/%d", family, goal)
					if pairs[policies[i]][key] == nil {
						pairs[policies[i]][key] = map[string]uint16{}
					}
					pairs[policies[i]][key][language] = o.Search.Attempts[0].Mask
					calls++
					predictions += o.Search.Selection.ModelCalls
					executions++
				}
			}
		}
	}
	for policy, p := range pairs {
		c := corrected[policy]
		original := nativeReport.Totals[policy]
		original.Pairs, original.Different = 0, 0
		if c != original {
			return errors.New("native actual attempt/case/prediction totals differ")
		}
		for _, languages := range p {
			if len(languages) != 2 {
				return errors.New("native language pair missing")
			}
			c.Pairs++
			if languages["en"] != languages["ko"] {
				c.Different++
			}
		}
		corrected[policy] = c
		if nativeReport.Totals[policy].Pairs != 0 && c != nativeReport.Totals[policy] {
			return errors.New("populated native bilingual totals differ")
		}
	}
	if calls != 144 || executions != 144 || predictions != nativeReport.Predictions {
		return errors.New("actual native evidence denominator differs")
	}
	return save(output, map[string]any{"schema": "gooo/fresh-composition-independent-audit/v1", "status": "PASS", "dataset_sha256": datasetSHA,
		"frozen_sdk_function_observations": records, "native_captures": calls, "independently_executed_go_captures": executions,
		"recorded_native_model_predictions": predictions, "selected_candidate": chosen, "native_totals_with_reconstructed_bilingual_pairs": corrected,
		"native_original_bilingual_fields_populated": nativeReport.Totals["selected"].Pairs != 0, "native_bilingual_counter_correction": "The first native aggregator emitted zero in unpopulated pair fields. Retained raw captures reconstruct 24 pairs per policy; this audit appends corrected bilingual statistics without rerunning or changing captures. Future native aggregation populates and verifies these counters.",
		"new_native_calls": 0, "new_model_predictions": 0, "new_optimizer_updates": 0,
		"scope": "Independent arithmetic, ordered duplicate cases, complete masks, input/model/source hashes, receipt chains, calibration selector and captured actual-Go values reconciled without new inference or execution."})
}
