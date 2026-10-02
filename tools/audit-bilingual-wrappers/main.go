// audit-bilingual-wrappers measures complete-input sensitivity of frozen models.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

const protocol = "docs/bilingual-wrapper-audit-protocol-20261003.md"

type loaded struct {
	name  string
	model *jointdecision.ThreeModel
}
type prepared struct {
	view  threecohort.View
	texts [5]string
}
type collision struct {
	Valid uint8    `json:"common_passing_masks"`
	IDs   []string `json:"input_ids"`
}

func main() {
	dataset := flag.String("dataset", "", "frozen source curriculum")
	dense := flag.String("dense-models", "", "existing dense model directory")
	shared := flag.String("shared-models", "", "existing compact shared model directory")
	out := flag.String("output", "", "fresh raw output directory")
	revision := flag.String("source-revision", "", "clean committed audit revision")
	flag.Parse()
	if err := run(*dataset, *dense, *shared, *out, *revision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(dataset, dense, shared, out, revision string) (resultErr error) {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return fmt.Errorf("exact committed source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) > 0 {
		return fmt.Errorf("clean audit source required")
	}
	if out == "" {
		return fmt.Errorf("fresh output required")
	}
	if _, err = os.Lstat(out); !os.IsNotExist(err) {
		return fmt.Errorf("output must not exist; retain prior attempts")
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	for i := 0; i < len(views); i += 2 {
		a, b := views[i], views[i+1]
		if a.Group != b.Group || a.Language != "en" || b.Language != "ko" || a.SourceSHA != b.SourceSHA || !reflect.DeepEqual(a.Target, b.Target) {
			return fmt.Errorf("bilingual source/target mismatch")
		}
	}
	models := []loaded{}
	pins := map[string]any{}
	for _, arm := range []string{"dense", "shared"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			root, load := dense, jointdecision.LoadThree
			if arm == "shared" {
				root, load = shared, jointdecision.LoadSharedThree
			}
			m, e := load(filepath.Join(root, variant, "model.json"))
			if e != nil {
				return e
			}
			if m.Variant() != variant {
				return fmt.Errorf("variant mismatch")
			}
			name := arm + "/" + variant
			models = append(models, loaded{name, m})
			pins[name] = map[string]any{"schema": m.Schema(), "metadata_sha256": m.MetadataSHA256(), "weights_sha256": m.WeightsSHA256()}
		}
	}
	protocolPin, err := pin(protocol)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return err
	}
	if err = save(out, "preexecution.json", map[string]any{"schema": "gooo/bilingual-wrapper-preexecution/v1", "source_revision": revision, "protocol": protocolPin, "dataset_sha256": threecohort.DatasetSHA, "models": pins, "forms": forms, "planned_model_predictions": 92160, "raw_cap_bytes": capBytes, "optimizer_updates": 0, "native_calls": 0}); err != nil {
		return err
	}
	var used int64
	calls := 0
	inputs, err := openJournal(out, "inputs.jsonl", &used)
	if err != nil {
		return err
	}
	rows, err := openJournal(out, "predictions.jsonl", &used)
	if err != nil {
		inputs.file.Close()
		return err
	}
	defer func() {
		for _, j := range []*journal{inputs, rows} {
			if e := j.file.Close(); resultErr == nil && e != nil {
				resultErr = e
			}
		}
		if resultErr != nil {
			_ = save(out, "failure.json", map[string]any{"status": "FAILED_PREFIX_RETAINED", "actual_model_predictions": calls, "raw_journal_bytes": used})
		}
	}()
	all := make([]prepared, 0, len(views))
	collisions := map[string]*collision{}
	for _, v := range views {
		p := prepared{view: v}
		var original [768]float32
		if err = jointdecision.FeaturesIntoThree(v.Text, &original); err != nil {
			return err
		}
		for i, form := range forms {
			text, e := transformed(v, form)
			if e != nil {
				return e
			}
			p.texts[i] = text
			featureSHA, distance, e := featurePin(text, &original)
			if e != nil {
				return e
			}
			key := form + "/" + featureSHA
			if collisions[key] == nil {
				collisions[key] = &collision{Valid: 255}
			}
			c := collisions[key]
			c.Valid &= passingSet(v)
			c.IDs = append(c.IDs, v.ID+"/"+form)
			if e = inputs.append(map[string]any{"id": v.ID + "/" + form, "view_id": v.ID, "form": form, "split": v.Split, "language": v.Language, "source_sha256": v.SourceSHA, "input": text, "input_sha256": threecohort.SHA([]byte(text)), "feature_sha256": featureSHA, "feature_l2_from_original": distance, "passing_masks": passingSet(v), "source_channel_unchanged": true}); e != nil {
				return e
			}
		}
		all = append(all, p)
	}
	results := map[string]*result{}
	for _, m := range models {
		for fi, form := range forms {
			r := &result{Counts: map[string]*counts{}, Pairs: map[string]*pairCounts{}}
			results[m.name+"/"+form] = r
			var workspace jointdecision.ThreeWorkspace
			var prediction jointdecision.ThreePrediction
			var previous uint16
			for i, p := range all {
				if err = m.model.PredictInto(p.texts[fi], &workspace, &prediction); err != nil {
					return err
				}
				calls++
				if err = r.add(p.view, prediction); err != nil {
					return err
				}
				if i%2 == 0 {
					previous = prediction.Mask
				} else {
					key := p.view.Split
					if r.Pairs[key] == nil {
						r.Pairs[key] = &pairCounts{}
					}
					r.Pairs[key].add(previous, prediction.Mask, passingSet(p.view))
				}
				if err = rows.append(map[string]any{"model": m.name, "input_id": p.view.ID + "/" + form, "prediction": prediction}); err != nil {
					return err
				}
			}
		}
	}
	if calls != 92160 {
		return fmt.Errorf("incomplete prediction denominator")
	}
	baseline := map[string][4]int{"dense/fp32": {95, 1572, 256, 0}, "shared/fp32": {113, 1469, 255, 0}, "dense/ptq_ternary": {74, 1595, 45, 180}, "shared/ptq_ternary": {80, 1439, 136, 98}, "dense/qat_ternary": {82, 1584, 240, 12}, "shared/qat_ternary": {89, 1512, 255, 1}}
	for name, want := range baseline {
		r := results[name+"/original"]
		c, p := r.Counts["development/all"], r.Pairs["development"]
		if [4]int{c.Complete, c.Extra, p.Disagree, p.SameWrong} != want {
			return fmt.Errorf("published original development observation changed: %s", name)
		}
	}
	conflicts := map[string]*collision{}
	duplicateBuckets := map[string]int{}
	for key, c := range collisions {
		if len(c.IDs) > 1 {
			duplicateBuckets[strings.SplitN(key, "/", 2)[0]]++
		}
		if c.Valid == 0 {
			conflicts[key] = c
		}
	}
	if err = save(out, "feature-conflicts.json", conflicts); err != nil {
		return err
	}
	if err = save(out, "report.json", map[string]any{"schema": "gooo/bilingual-wrapper-audit/v1", "status": "COMPLETE", "source_revision": revision, "actual_model_predictions": calls, "input_forms": 15360, "raw_journal_bytes": used, "models": pins, "results": results, "duplicate_feature_buckets": duplicateBuckets, "disjoint_passing_set_feature_buckets": len(conflicts), "bilingual_source_target_pairs_verified": 1536, "optimizer_updates": 0, "native_calls": 0, "scope": "Authored wrapper interventions on the previously observed fixed source cohort. Static rankings, complete finite targets, no adaptive feedback or model promotion."}); err != nil {
		return err
	}
	names := []string{"preexecution.json", "inputs.jsonl", "predictions.jsonl", "feature-conflicts.json", "report.json"}
	sort.Strings(names)
	files := map[string]any{}
	var total int64
	for _, name := range names {
		p, e := pin(filepath.Join(out, name))
		if e != nil {
			return e
		}
		files[name] = p
		total += p["bytes"].(int64)
	}
	if total > capBytes-(64<<10) {
		return fmt.Errorf("raw output cap exceeded")
	}
	return save(out, "manifest.json", map[string]any{"schema": "gooo/bilingual-wrapper-files/v1", "files": files})
}
