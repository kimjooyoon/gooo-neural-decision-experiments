package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
			return errors.New("progress digest differs")
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
			return errors.New("feedback digest differs")
		}
	}
	return nil
}
func priorObservations(study, split, id string) (map[string]observation, error) {
	f, err := os.Open(filepath.Join(study, split+"-"+id+".jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 32768), 1<<20)
	result := map[string]observation{}
	for scanner.Scan() {
		var o observation
		if err = json.Unmarshal(scanner.Bytes(), &o); err != nil {
			return nil, err
		}
		if _, ok := result[o.ID]; ok || o.ID == "" {
			return nil, errors.New("duplicate/empty observed view")
		}
		result[o.ID] = o
	}
	return result, scanner.Err()
}
func auditModelBinding(v view, o observation, m *loadedModel, pin modelPin) error {
	kind := "offline"
	if m != nil {
		kind = m.kind
		if o.Search.Selection.MetadataSHA256 != pin.Metadata || o.Search.Selection.WeightsSHA256 != pin.Weights {
			return errors.New("frozen model pin differs")
		}
	}
	if err := validateObservation(v, o, kind); err != nil {
		return err
	}
	if err := receiptLinks(o); err != nil {
		return err
	}
	if o.Inputs != [2]string{v.Rows[0].Input.SHA, v.Rows[1].Input.SHA} {
		return errors.New("original input binding differs")
	}
	if kind == "joint" {
		r := o.Progress[0].Selection.Joint
		if r == nil || r.Calls != 1 || r.Schema != "gooo/tiny-joint-path-model/v1" || r.Input != v.Rows[0].JointText || r.InputSHA != v.Rows[0].JointSHA || r.Bytes != len(r.Input) || r.Declined {
			return errors.New("initial joint receipt differs")
		}
		for i, row := range v.Rows {
			if r.PartSHA[i] != hash([]byte(row.Input.Text)) {
				return errors.New("joint part source digest differs")
			}
		}
	}
	for _, f := range o.Feedback {
		if f.Joint != nil {
			r := f.Joint
			if r.InputSHA != hash([]byte(r.Input)) || r.Bytes != len(r.Input) || r.Calls != f.ModelCalls {
				return errors.New("feedback joint input/call facts differ")
			}
		}
	}
	return nil
}
func summarizeFrozen(views []view, split, id, study string, m *loadedModel, pin modelPin) (score, error) {
	observations, err := priorObservations(study, split, id)
	if err != nil {
		return score{}, err
	}
	result := score{Families: map[string]counts{}}
	initial := map[string]map[string]uint16{}
	for _, v := range views {
		r := v.Rows[0]
		if r.Split != split {
			continue
		}
		o, ok := observations[v.ID]
		if !ok {
			return result, errors.New("frozen observation missing")
		}
		if err = auditModelBinding(v, o, m, pin); err != nil {
			return result, err
		}
		add(&result.Total, o)
		if err = addTarget(&result.Total, o, r.Finite.Joint); err != nil {
			return result, err
		}
		key := r.Family + "/" + r.Language
		c := result.Families[key]
		add(&c, o)
		if err = addTarget(&c, o, r.Finite.Joint); err != nil {
			return result, err
		}
		result.Families[key] = c
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
	if result.Total.Views != 384 || result.Total.Pairs != 192 || len(observations) != 384 {
		return result, errors.New("frozen audit denominator differs")
	}
	return result, nil
}
func auditSDK(curriculum, models, study, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh audit file required")
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
		Calls       int              `json:"actual_model_predictions_all_stages"`
	}
	if err = decodeFile(filepath.Join(study, "report.json"), &report); err != nil || report.Status != "PASS" {
		return errors.New("complete study report required")
	}
	var selection struct {
		Selected string           `json:"selected"`
		Scores   map[string]score `json:"calibration"`
		Frozen   bool             `json:"frozen_before_development_session_predictions"`
	}
	if err = decodeFile(filepath.Join(study, "selection.json"), &selection); err != nil || !selection.Frozen {
		return errors.New("frozen calibration selection required")
	}
	records, calls := 0, 192+12012
	calibration := map[string]score{}
	ids := []string{"offline"}
	for _, arm := range arms {
		for _, variant := range variants {
			ids = append(ids, arm+"-"+variant)
		}
	}
	for _, split := range []string{"calibration", "development"} {
		for _, id := range ids {
			s, err := summarizeFrozen(views, split, id, study, loaded[id], pins[id])
			if err != nil {
				return err
			}
			expected := report.Calibration[id]
			if split == "development" {
				expected = report.Development[id]
			} else {
				calibration[id] = s
			}
			if !reflect.DeepEqual(s, expected) {
				return errors.New("frozen independent summary differs")
			}
			calls += s.Total.Calls
			records += s.Total.Views
		}
	}
	chosen, err := choose(calibration, pins)
	if err != nil || chosen != report.Selected || chosen != selection.Selected || !reflect.DeepEqual(calibration, selection.Scores) || calls != report.Calls || records != 5376 {
		return errors.New("calibration choice/actual-call totals differ")
	}
	return save(output, map[string]any{"schema": "gooo/joint-composition-sdk-audit/v1", "status": "PASS", "dataset_sha256": datasetSHA, "frozen_sdk_function_observations": records, "actual_model_predictions_all_stages": calls, "selected_candidate": chosen, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_updates": 0, "native_execution_verified": false, "scope": "Independent Go arithmetic reconstruction, full probabilities, source hashes, progress/feedback chains, model pins, language pairs and calibration selector; four-path finite construction only. Native integration remains a separate stage."})
}
