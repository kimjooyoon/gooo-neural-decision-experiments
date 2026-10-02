package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func auditStudy(dataset, root, directory, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh SDK audit required")
	}
	views, err := jointcohort.Load(dataset)
	if err != nil {
		return err
	}
	models, ids, err := loadModels(root)
	if err != nil {
		return err
	}
	var captured struct {
		Status      string            `json:"status"`
		Selected    string            `json:"selected_candidate"`
		Sessions    int               `json:"actual_sdk_sessions"`
		Predictions int               `json:"actual_model_predictions"`
		Calibration map[string]counts `json:"calibration"`
		Development map[string]counts `json:"development"`
	}
	if err = read(filepath.Join(directory, "report.json"), &captured); err != nil {
		return err
	}
	if captured.Status != "PASS" || captured.Sessions != 9216 || len(captured.Calibration) != 12 || len(captured.Development) != 12 {
		return errors.New("completed all-policy SDK study required")
	}
	all, sessions, predictions := map[string]map[string]counts{}, 0, 0
	differences := []map[string]any{}
	for _, split := range []string{"calibration", "development"} {
		cells := map[string]counts{}
		for _, id := range ids {
			c, err := auditCell(directory, split, id, views, models[id])
			if err != nil {
				return err
			}
			original := captured.Calibration[id]
			if split == "development" {
				original = captured.Development[id]
			}
			if !equalCounts(c, original) {
				return errors.New("SDK integer counts, curves, durations or derived statistics differ")
			}
			for metric, pair := range map[string][2]float64{"initial_outside_mass": {original.Outside, c.Outside}, "initial_passing_set_nll": {original.SetNLL, c.SetNLL}, "initial_uniform_target_cross_entropy": {original.UniformCE, c.UniformCE}} {
				if pair[0] != pair[1] {
					differences = append(differences, map[string]any{"cell": split + "/" + id, "metric": metric, "captured": pair[0], "recomputed": pair[1]})
				}
			}
			cells[id] = c
			sessions += c.Views
			predictions += c.Predictions
		}
		all[split] = cells
	}
	selected := choose(all["calibration"], models)
	var selection struct {
		Selected    string            `json:"selected_candidate"`
		Development bool              `json:"development_seen"`
		Calibration map[string]counts `json:"calibration"`
		Pins        map[string]pin    `json:"model_pins"`
	}
	if err = read(filepath.Join(directory, "selection.json"), &selection); err != nil {
		return err
	}
	if selected != captured.Selected || selected != selection.Selected || selection.Development || sessions != 9216 || predictions != captured.Predictions || len(selection.Pins) != 11 {
		return errors.New("calibration-only selector or actual totals differ")
	}
	for id, m := range models {
		if !equalCounts(all["calibration"][id], selection.Calibration[id]) {
			return errors.New("selector calibration differs")
		}
		if m != nil && selection.Pins[id] != m.Pin {
			return errors.New("selector model pin differs")
		}
	}
	return save(output, map[string]any{"schema": "gooo/own-joint-feedback-sdk-audit/v2", "status": "PASS", "sdk_sessions_audited": sessions,
		"recorded_model_predictions": predictions, "selected_candidate": selected, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_updates": 0,
		"floating_aggregate_rounding_differences": differences, "floating_aggregate_tolerance": "1e-10 absolute + 1e-12 relative only for rederived NLL/mass; counts, curves and captured durations exact",
		"scope": "Independent source/oracle reconstruction, actual candidate values, source/model/input bindings, progress and feedback receipts, all twelve policies and the calibration-only selector. Frozen observations are reconstructed without operational inference."})
}

func equalCounts(a, b counts) bool {
	for _, p := range [][2]float64{{a.Outside, b.Outside}, {a.SetNLL, b.SetNLL}, {a.UniformCE, b.UniformCE}} {
		if math.IsNaN(p[0]) || math.IsNaN(p[1]) || math.IsInf(p[0], 0) || math.IsInf(p[1], 0) || math.Abs(p[0]-p[1]) > 1e-10+1e-12*math.Max(math.Abs(p[0]), math.Abs(p[1])) {
			return false
		}
	}
	a.Outside, a.SetNLL, a.UniformCE, b.Outside, b.SetNLL, b.UniformCE = 0, 0, 0, 0, 0, 0
	return a == b
}

func auditCell(directory, split, id string, views []jointcohort.View, m *model) (counts, error) {
	byID := map[string]jointcohort.View{}
	for _, v := range views {
		if v.Split == split {
			byID[v.ID] = v
		}
	}
	f, err := os.Open(filepath.Join(directory, split+"-"+strings.ReplaceAll(id, "/", "-")+".jsonl"))
	if err != nil {
		return counts{}, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 32768), 1<<20)
	c, seen, pairs := counts{}, map[string]bool{}, map[string]map[string]uint16{}
	for scanner.Scan() {
		var o observation
		if err = decision.RejectDuplicateJSONKeys(scanner.Bytes()); err != nil {
			return c, err
		}
		if err = json.Unmarshal(scanner.Bytes(), &o); err != nil {
			return c, err
		}
		v, ok := byID[o.Capture.ViewID]
		if !ok || seen[v.ID] || o.Policy != id {
			return c, errors.New("distinct source-bound policy view required")
		}
		seen[v.ID] = true
		if err = verifyObservation(v, o.Capture, m); err != nil {
			return c, err
		}
		if err = c.add(v, o.Capture); err != nil {
			return c, err
		}
		if pairs[v.Group] == nil {
			pairs[v.Group] = map[string]uint16{}
		}
		pairs[v.Group][v.Language] = o.Capture.Search.Attempts[0].Mask
	}
	if err = scanner.Err(); err != nil {
		return c, err
	}
	for _, languages := range pairs {
		if len(languages) != 2 {
			return c, errors.New("bilingual observation pair missing")
		}
		c.Pairs++
		if languages["en"] != languages["ko"] {
			c.Disagreement++
		}
	}
	if c.Views != 384 || c.Curve[3] != 384 {
		return c, errors.New("SDK view/finite completeness count differs")
	}
	return c, nil
}

func verifyObservation(v jointcohort.View, c jointfeedback.Capture, m *model) error {
	if c.Schema != "gooo/own-joint-feedback-student-capture/v2" || c.ViewID != v.ID || c.SourceSHA != v.SourceSHA || c.JointSHA != jointcohort.SHA([]byte(v.JointInput)) || c.WallNS <= 0 || c.Seed != "" {
		return errors.New("original student input/source binding differs")
	}
	if m != nil && m.Joint != nil {
		return jointfeedback.VerifyJointObservation(v, c, m.Pin.Metadata, m.Pin.Weights)
	}
	if err := jointfeedback.VerifyFiniteAttempts(v, c.Search); err != nil {
		return err
	}
	if c.Search.Selection.PlanSHA256 != v.Prepared.PlanSHA256() || c.Search.Selection.SeedSHA256 != "" {
		return errors.New("source plan changed during reference search")
	}
	if m == nil {
		if c.Search.Selection.ModelCalls != 0 || c.Search.Selection.MetadataSHA256 != "" || c.Search.Selection.WeightsSHA256 != "" || len(c.Feedback) != 0 {
			return errors.New("disconnected path performed model work")
		}
	} else if c.Search.Selection.MetadataSHA256 != m.Pin.Metadata || c.Search.Selection.WeightsSHA256 != m.Pin.Weights {
		return errors.New("reference model pin differs")
	}
	return verifyReferenceLinks(v, c, m)
}

func verifyReferenceLinks(v jointcohort.View, c jointfeedback.Capture, m *model) error {
	if err := verifyReferenceProgress(v, c, m); err != nil {
		return err
	}
	previous := ""
	bySHA := map[string]pathplan.SessionProgress{}
	for i, p := range c.Progress {
		sha := p.SHA
		p.SHA = ""
		raw, err := json.Marshal(p)
		if err != nil || sha == "" || sha != jointcohort.SHA(raw) || p.PreviousSHA != previous || p.Sequence != i+1 || p.Selection.PlanSHA256 != v.Prepared.PlanSHA256() {
			return errors.New("reference progress chain differs")
		}
		p.SHA = sha
		bySHA[sha] = p
		previous = sha
	}
	if len(c.Progress) < 2 || !reflect.DeepEqual(c.Progress[len(c.Progress)-1].Selection, c.Search.Selection) {
		return errors.New("reference final observation differs")
	}
	previous, calls := "", c.Progress[0].Selection.ModelCalls
	if m == nil && calls != 0 || m != nil && calls != 2 {
		return errors.New("reference initial actual calls differ")
	}
	parts, err := jointParts(v)
	if err != nil {
		return err
	}
	plan, err := jointcompositionstudy.Fixture(v.Family, v.Config, v.Goal, v.Language)
	if err != nil {
		return err
	}
	initial := c.Progress[0].Selection.Receipts
	if len(initial) != 2 {
		return errors.New("complete original reference judgments required")
	}
	for i, r := range initial {
		if r.ID != plan.Decisions[i].ID || r.Kind != plan.Decisions[i].Kind || r.IntentSHA256 != jointcohort.SHA([]byte(parts[i])) {
			return errors.New("original reference intent binding differs")
		}
	}
	for i, f := range c.Feedback {
		sha := f.SHA
		f.SHA = ""
		raw, err := json.Marshal(f)
		p, ok := bySHA[f.FromProgressSHA]
		if err != nil || !ok || sha == "" || sha != jointcohort.SHA(raw) || f.PreviousSHA != previous || f.Round != i+1 || f.PlanSHA != v.Prepared.PlanSHA256() || f.CaseSHA != p.CaseSHA || f.Attempted != p.Attempted || f.Passed != p.SelectedPassed || f.CI != nil || f.CIIsAuthority {
			return errors.New("reference feedback cause differs")
		}
		if m == nil || f.MetadataSHA != m.Pin.Metadata || f.WeightsSHA != m.Pin.Weights {
			return errors.New("reference feedback model differs")
		}
		var mismatch *pathplan.TestResult
		for _, x := range p.BestCases {
			if !x.Passed {
				copy := x
				mismatch = &copy
				break
			}
		}
		if mismatch == nil || !reflect.DeepEqual(mismatch, f.FirstFailure) {
			return errors.New("reference feedback not caused by actual failure")
		}
		for _, j := range f.Judgments {
			coordinate := -1
			for i, choice := range plan.Decisions {
				if choice.ID == j.DecisionID {
					coordinate = i
				}
			}
			if coordinate < 0 || j.InputSHA != jointcohort.SHA([]byte(j.Input)) {
				return errors.New("reference full source intent was not retained")
			}
			x := f.FirstFailure
			prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d mismatch=%d:%d:%d selected=%s", f.Attempted, f.Passed, f.Cases, f.TypeRejected, 4-f.Attempted, x.Input, x.Actual, x.Expected, p.Selection.Choices[j.DecisionID])
			input, err := decision.SemanticContextFeedbackInput(parts[coordinate], prefix)
			if err != nil || input != j.Input || j.Prediction.ID != j.DecisionID || j.Prediction.Kind != plan.Decisions[coordinate].Kind || j.Prediction.IntentSHA256 != jointcohort.SHA([]byte(parts[coordinate])) {
				return errors.New("reference feedback text differs from the observed failure")
			}
		}
		calls += f.ModelCalls
		if calls != f.CumulativeCalls {
			return errors.New("reference actual call count differs")
		}
		previous = sha
	}
	if calls != c.Search.Selection.ModelCalls {
		return errors.New("reference total actual calls differ")
	}
	return nil
}
