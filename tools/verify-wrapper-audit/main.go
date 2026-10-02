// verify-wrapper-audit independently replays each archived input and prediction.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

var forms = []string{"original", "bare", "calibration_prefix", "development_prefix", "development_suffix"}
var models = []string{"dense/fp32", "dense/ptq_ternary", "dense/qat_ternary", "shared/fp32", "shared/ptq_ternary", "shared/qat_ternary"}

type input struct {
	ID        string  `json:"id"`
	View      string  `json:"view_id"`
	Form      string  `json:"form"`
	Split     string  `json:"split"`
	Language  string  `json:"language"`
	Source    string  `json:"source_sha256"`
	Text      string  `json:"input"`
	SHA       string  `json:"input_sha256"`
	Feature   string  `json:"feature_sha256"`
	Distance  float64 `json:"feature_l2_from_original"`
	Passing   uint8   `json:"passing_masks"`
	Preserved bool    `json:"source_channel_unchanged"`
}
type counts struct {
	Views    int    `json:"views"`
	Cases    int    `json:"cases"`
	Passed   int    `json:"initial_passed_cases"`
	Complete int    `json:"initial_complete"`
	Extra    int    `json:"static_ranked_extra_attempts"`
	Curve    [8]int `json:"complete_by_budget"`
	Masks    [8]int `json:"initial_mask_histogram"`
}
type pairs struct {
	Pairs     int `json:"pairs"`
	Disagree  int `json:"mask_disagreements"`
	Both      int `json:"both_valid"`
	Same      int `json:"same_mask_both_valid"`
	Different int `json:"different_masks_both_valid"`
	Wrong     int `json:"same_mask_both_wrong"`
}
type result struct {
	Counts map[string]*counts `json:"counts"`
	Pairs  map[string]*pairs  `json:"pairs"`
}
type pin struct {
	Schema   string `json:"schema"`
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
}

func main() {
	root := flag.String("bundle", "", "verified extracted archive")
	out := flag.String("output", "", "fresh verification JSON")
	flag.Parse()
	r, e := verify(*root)
	if e == nil {
		var b []byte
		b, e = json.MarshalIndent(r, "", "  ")
		if e == nil {
			var f *os.File
			f, e = os.OpenFile(*out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e == nil {
				_, e = f.Write(append(b, '\n'))
				closed := f.Close()
				if e == nil {
					e = closed
				}
			}
		}
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("PASS: complete input, feature, model and finite-target replay")
}

func scan(path string, fn func(int, []byte) error) error {
	s, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !s.Mode().IsRegular() || s.Size() > 32<<20 {
		return errors.New("bounded regular journal required")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	n := 0
	for scanner.Scan() {
		if e = fn(n, scanner.Bytes()); e != nil {
			return e
		}
		n++
	}
	return scanner.Err()
}
func read(path string, v any) error {
	s, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !s.Mode().IsRegular() || s.Size() > 2<<20 {
		return errors.New("bounded regular metadata required")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return threecohort.Decode(b, v)
}

func expected(v threecohort.View, form string) (string, error) {
	if form == "original" {
		return v.Text, nil
	}
	cal, dev := "Decision request: ", "While composing, "
	if v.Language == "ko" {
		cal, dev = "구성 요청: ", "함수를 구성할 때, "
	}
	parts := v.Parts
	for i, part := range parts {
		before, body, ok := strings.Cut(part, ";intent: ")
		if !ok {
			return "", errors.New("missing intent")
		}
		prefix := ""
		if v.Split == "calibration" {
			prefix = cal
		}
		if v.Split == "development" {
			prefix = dev
		}
		if !strings.HasPrefix(body, prefix) {
			return "", errors.New("wrong original wrapper")
		}
		body = body[len(prefix):]
		switch form {
		case "bare":
		case "calibration_prefix":
			body = cal + body
		case "development_prefix":
			body = dev + body
		case "development_suffix":
			body += " " + strings.TrimSuffix(dev, ", ") + "."
		default:
			return "", errors.New("unknown wrapper")
		}
		parts[i] = before + ";intent: " + body
	}
	return jointdecision.EncodeThree(parts)
}

func verifyInput(v threecohort.View, form string, in input) error {
	want, e := expected(v, form)
	if e != nil {
		return e
	}
	var valid uint8
	for mask, passed := range v.Target.Passed {
		if passed == v.Target.Cases {
			valid |= 1 << mask
		}
	}
	if in.ID != v.ID+"/"+form || in.View != v.ID || in.Form != form || in.Split != v.Split || in.Language != v.Language || in.Source != v.SourceSHA || in.Text != want || in.SHA != threecohort.SHA([]byte(want)) || in.Passing != valid || !in.Preserved {
		return errors.New("complete input identity/target differs")
	}
	var before, after [768]float32
	if e = jointdecision.FeaturesIntoThree(v.Text, &before); e != nil {
		return e
	}
	if e = jointdecision.FeaturesIntoThree(in.Text, &after); e != nil {
		return e
	}
	var raw [3072]byte
	var squared float64
	for i, x := range after {
		if i%256 < 64 && x != before[i] {
			return errors.New("source channel differs")
		}
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(x))
		d := float64(x) - float64(before[i])
		squared += d * d
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw[:])) != in.Feature || math.Abs(math.Sqrt(squared)-in.Distance) > 1e-12 {
		return errors.New("feature identity/distance differs")
	}
	return nil
}

func increment(c *counts, v threecohort.View, p jointdecision.ThreePrediction) error {
	if p.Mask >= 8 {
		return errors.New("invalid mask")
	}
	order := []int{0, 1, 2, 3, 4, 5, 6, 7}
	sort.SliceStable(order, func(i, j int) bool { return p.Probabilities[order[i]] > p.Probabilities[order[j]] })
	if int(p.Mask) != order[0] {
		return errors.New("recorded tie decision differs")
	}
	first := 8
	for rank, mask := range order {
		if v.Target.Passed[mask] == v.Target.Cases {
			first = rank
			break
		}
	}
	if first == 8 {
		return errors.New("missing passing candidate")
	}
	c.Views++
	c.Cases += v.Target.Cases
	c.Passed += v.Target.Passed[p.Mask]
	c.Masks[p.Mask]++
	c.Extra += first
	if first == 0 {
		c.Complete++
	}
	for budget := first; budget < 8; budget++ {
		c.Curve[budget]++
	}
	return nil
}

func verify(root string) (any, error) {
	views, e := threecohort.Load(filepath.Join(root, "dataset.jsonl"))
	if e != nil {
		return nil, e
	}
	var report struct {
		Schema     string            `json:"schema"`
		Status     string            `json:"status"`
		Source     string            `json:"source_revision"`
		Calls      int               `json:"actual_model_predictions"`
		Inputs     int               `json:"input_forms"`
		Bytes      int               `json:"raw_journal_bytes"`
		Models     map[string]pin    `json:"models"`
		Results    map[string]result `json:"results"`
		Duplicates map[string]int    `json:"duplicate_feature_buckets"`
		Conflicts  int               `json:"disjoint_passing_set_feature_buckets"`
		Pairs      int               `json:"bilingual_source_target_pairs_verified"`
		Updates    int               `json:"optimizer_updates"`
		Native     int               `json:"native_calls"`
		Scope      string            `json:"scope"`
	}
	if e = read(filepath.Join(root, "raw/report.json"), &report); e != nil {
		return nil, e
	}
	if report.Schema != "gooo/bilingual-wrapper-audit/v1" || report.Status != "COMPLETE" || report.Source != "75da63a0171a60180eacc3d997af4d7e5e8e35ba" || report.Calls != 92160 || report.Inputs != 15360 || report.Updates != 0 || report.Native != 0 || report.Pairs != 1536 {
		return nil, errors.New("complete frozen report required")
	}
	var journalBytes int64
	for _, name := range []string{"inputs.jsonl", "predictions.jsonl"} {
		s, err := os.Lstat(filepath.Join(root, "raw", name))
		if err != nil {
			return nil, err
		}
		journalBytes += s.Size()
	}
	if journalBytes != int64(report.Bytes) {
		return nil, errors.New("journal byte denominator differs")
	}
	for i := 0; i < len(views); i += 2 {
		if views[i].Group != views[i+1].Group || views[i].SourceSHA != views[i+1].SourceSHA || !reflect.DeepEqual(views[i].Target, views[i+1].Target) {
			return nil, errors.New("bilingual source/targets differ")
		}
	}
	inputs := make([]input, 0, 15360)
	type collision struct {
		Valid uint8    `json:"common_passing_masks"`
		IDs   []string `json:"input_ids"`
	}
	collisions := map[string]*collision{}
	e = scan(filepath.Join(root, "raw/inputs.jsonl"), func(n int, b []byte) error {
		if n >= 15360 {
			return errors.New("extra input")
		}
		var in input
		if e := threecohort.Decode(b, &in); e != nil {
			return e
		}
		v, form := views[n/5], forms[n%5]
		if e := verifyInput(v, form, in); e != nil {
			return e
		}
		key := form + "/" + in.Feature
		if collisions[key] == nil {
			collisions[key] = &collision{Valid: 255}
		}
		c := collisions[key]
		c.Valid &= in.Passing
		c.IDs = append(c.IDs, in.ID)
		inputs = append(inputs, in)
		return nil
	})
	if e != nil {
		return nil, e
	}
	if len(inputs) != 15360 {
		return nil, errors.New("incomplete inputs")
	}
	conflicts := map[string]*collision{}
	duplicates := map[string]int{}
	for key, c := range collisions {
		if len(c.IDs) > 1 {
			duplicates[strings.SplitN(key, "/", 2)[0]]++
		}
		if c.Valid == 0 {
			conflicts[key] = c
		}
	}
	var recordedConflicts map[string]*collision
	if e = read(filepath.Join(root, "raw/feature-conflicts.json"), &recordedConflicts); e != nil {
		return nil, e
	}
	if !reflect.DeepEqual(conflicts, recordedConflicts) || len(conflicts) != report.Conflicts || !reflect.DeepEqual(duplicates, report.Duplicates) {
		return nil, errors.New("feature collision audit differs")
	}
	loaded := map[string]*jointdecision.ThreeModel{}
	for _, name := range models {
		load := jointdecision.LoadThree
		if strings.HasPrefix(name, "shared/") {
			load = jointdecision.LoadSharedThree
		}
		m, err := load(filepath.Join(root, "models", name, "model.json"))
		if err != nil {
			return nil, err
		}
		want, ok := report.Models[name]
		if !ok || want != (pin{m.Schema(), m.MetadataSHA256(), m.WeightsSHA256()}) {
			return nil, errors.New("model identities differ")
		}
		loaded[name] = m
	}
	if len(report.Models) != 6 {
		return nil, errors.New("extra model identity")
	}
	observed := map[string]result{}
	calls := 0
	maxError := 0.
	var previous uint16
	e = scan(filepath.Join(root, "raw/predictions.jsonl"), func(n int, b []byte) error {
		if n >= 92160 {
			return errors.New("extra prediction")
		}
		model, fi, vi := models[n/(5*3072)], (n/3072)%5, n%3072
		form, v := forms[fi], views[vi]
		in := inputs[vi*5+fi]
		var row struct {
			Model      string                        `json:"model"`
			ID         string                        `json:"input_id"`
			Prediction jointdecision.ThreePrediction `json:"prediction"`
		}
		if e := threecohort.Decode(b, &row); e != nil {
			return e
		}
		if row.Model != model || row.ID != in.ID {
			return errors.New("prediction order/identity differs")
		}
		var workspace jointdecision.ThreeWorkspace
		var actual jointdecision.ThreePrediction
		if e := loaded[model].PredictInto(in.Text, &workspace, &actual); e != nil {
			return e
		}
		calls++
		if actual.Mask != row.Prediction.Mask {
			return errors.New("actual kernel mask differs")
		}
		for i := 0; i < 8; i++ {
			for _, difference := range []float64{float64(actual.Logits[i]) - float64(row.Prediction.Logits[i]), float64(actual.Probabilities[i]) - float64(row.Prediction.Probabilities[i])} {
				if math.IsNaN(difference) || math.IsInf(difference, 0) || math.Abs(difference) > 1e-5 {
					return errors.New("actual kernel numerical mismatch")
				}
				maxError = math.Max(maxError, math.Abs(difference))
			}
		}
		key := model + "/" + form
		r, ok := observed[key]
		if !ok {
			r = result{map[string]*counts{}, map[string]*pairs{}}
			observed[key] = r
		}
		for _, part := range []string{v.Split + "/all", v.Split + "/language/" + v.Language, v.Split + "/family/" + v.Family} {
			if r.Counts[part] == nil {
				r.Counts[part] = &counts{}
			}
			if e := increment(r.Counts[part], v, row.Prediction); e != nil {
				return e
			}
		}
		if vi%2 == 0 {
			previous = row.Prediction.Mask
		} else {
			if r.Pairs[v.Split] == nil {
				r.Pairs[v.Split] = &pairs{}
			}
			p := r.Pairs[v.Split]
			p.Pairs++
			a, b := previous, row.Prediction.Mask
			av, bv := v.Target.Passed[a] == v.Target.Cases, v.Target.Passed[b] == v.Target.Cases
			if a != b {
				p.Disagree++
			}
			if av && bv {
				p.Both++
				if a == b {
					p.Same++
				} else {
					p.Different++
				}
			}
			if a == b && !av && !bv {
				p.Wrong++
			}
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	if calls != 92160 || !reflect.DeepEqual(observed, report.Results) {
		return nil, errors.New("recomputed complete statistics differ")
	}
	return map[string]any{"schema": "gooo/bilingual-wrapper-verification/v1", "status": "PASS", "collector_source_revision": report.Source, "inputs_verified": len(inputs), "actual_replay_model_predictions": calls, "maximum_absolute_kernel_error": maxError, "result_groups_recomputed": len(observed), "source_target_pairs_verified": 1536, "conflicting_feature_buckets_verified": len(conflicts), "optimizer_updates": 0, "native_calls": 0, "scope": "Independent reader reconstructed every complete input/feature, replayed every Go kernel, and recomputed all finite and bilingual summaries. Frozen known cohort; no training or default model promotion."}, nil
}
