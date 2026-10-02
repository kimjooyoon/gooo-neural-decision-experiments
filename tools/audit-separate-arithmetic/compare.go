package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func readReport(root string) (report, error) {
	var r report
	b, err := os.ReadFile(filepath.Join(root, "report.json"))
	if err != nil {
		return r, err
	}
	if len(b) > 1<<20 {
		return r, errors.New("bounded collection report required")
	}
	if err = threecohort.Decode(b, &r); err != nil {
		return r, err
	}
	if r.Schema != "gooo/separate-arithmetic-collection/v1" || r.Status != "COLLECTED" || r.Rows != 18432 || r.Calls != 73728 || r.Updates != 0 || r.Manifest != manifestSHA || r.Lanes != lanes || r.GoVersion != "go1.27.1" || len(r.Source) != 40 || len(r.Models) != 96 || len(r.Journals) != 36 || len(r.Counts) != 36 || len(r.Sources) < 20 {
		return r, errors.New("complete pinned collection required")
	}
	if err = pinned(protocol, r.Protocol); err != nil {
		return r, err
	}
	for name, pin := range r.Sources {
		if !filepath.IsLocal(name) {
			return r, errors.New("nonlocal source pin")
		}
		if err = pinned(name, pin); err != nil {
			return r, err
		}
	}
	for _, arm := range arms {
		for _, variant := range variants {
			for _, layout := range []string{"expanded", "compact"} {
				for _, version := range []string{"legacy", "separate"} {
					for _, file := range []string{"model.json", "weights.bin"} {
						if _, ok := r.Models[version+"/"+layout+"/"+arm+"/"+variant+"/"+file]; !ok {
							return r, errors.New("expected model absent")
						}
					}
				}
			}
			for _, form := range forms {
				name := arm + "--" + variant + "--" + form + ".jsonl.gz"
				if _, ok := r.Journals[name]; !ok {
					return r, errors.New("expected journal absent")
				}
			}
		}
	}
	for name, pin := range r.Models {
		if !filepath.IsLocal(name) {
			return r, errors.New("nonlocal model pin")
		}
		if strings.HasPrefix(name, "separate/") {
			err = pinned(filepath.Join(root, "models", strings.TrimPrefix(name, "separate/")), pin)
		} else if strings.HasPrefix(name, "legacy/") {
			err = pinned(filepath.Join(bundle, "models", strings.TrimPrefix(name, "legacy/")), pin)
		} else {
			return r, errors.New("unknown model lane")
		}
		if err != nil {
			return r, err
		}
	}
	for name, pin := range r.Journals {
		if filepath.Base(name) != name {
			return r, errors.New("nonlocal journal pin")
		}
		if err = pinned(filepath.Join(root, name), pin); err != nil {
			return r, err
		}
	}
	return r, nil
}

func readPairs(path string) ([]pairedRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []pairedRow
	err = scanGzip(f, func(b []byte) error {
		var row pairedRow
		if err := threecohort.Decode(b, &row); err != nil {
			return err
		}
		if row.Original.Input != threecohort.SHA([]byte(row.Original.Text)) {
			return errors.New("input digest differs")
		}
		var features [jointdecision.ThreeFeatureDim]float32
		if strings.HasPrefix(filepath.Base(path), "bag-") {
			err = jointdecision.FeaturesIntoThreeBag(row.Original.Text, &features)
		} else {
			err = jointdecision.FeaturesIntoThree(row.Original.Text, &features)
		}
		if err != nil {
			return err
		}
		for _, o := range row.Lanes {
			if o.Features != bitsSHA(features[:]) {
				return errors.New("replayed features differ")
			}
			if err := validateObservation(o, row.Original.Passed); err != nil {
				return err
			}
		}
		if row.Lanes[0] != row.Lanes[1] || row.Lanes[2] != row.Lanes[3] {
			return errors.New("journal layout parity differs")
		}
		rows = append(rows, row)
		return nil
	})
	return rows, err
}

func validateObservation(o observation, passed [8]int) error {
	if o.Prediction.Mask >= 8 || o.HiddenSHA != bitsSHA(o.Hidden[:]) || o.LogitsSHA != bitsSHA(o.Prediction.Logits[:]) || o.ProbsSHA != bitsSHA(o.Prediction.Probabilities[:]) || o.Order != rank(o.Prediction) || o.Order[0] != int(o.Prediction.Mask) {
		return errors.New("observation arithmetic/ranking differs")
	}
	for _, v := range passed {
		if v < 0 || v > 16 {
			return errors.New("finite target extent differs")
		}
	}
	partial, at := finiteOutcomes(o.Order, passed)
	if at == 0 || o.Partial != partial || o.CompleteAt != at {
		return errors.New("finite outcome differs")
	}
	var total float64
	for _, v := range o.Prediction.Probabilities {
		if v < 0 || v > 1 || math.IsNaN(float64(v)) {
			return errors.New("invalid probability")
		}
		total += float64(v)
	}
	if math.Abs(total-1) > 1e-6 {
		return errors.New("probability mass differs")
	}
	for _, values := range [][]float32{o.Hidden[:], o.Prediction.Logits[:]} {
		for _, v := range values {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return errors.New("nonfinite observation")
			}
		}
	}
	return nil
}

func compare(local, remote, output string) error {
	if output == "" {
		return errors.New("fresh comparison path required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh comparison path required")
	}
	a, err := readReport(local)
	if err != nil {
		return err
	}
	b, err := readReport(remote)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(a.Sources, b.Sources) || !reflect.DeepEqual(a.Models, b.Models) || a.Protocol != b.Protocol {
		return errors.New("computational source/model/protocol differs")
	}
	var legacy, separate changes
	var changed []map[string]any
	countsA, countsB := map[string][4]totals{}, map[string][4]totals{}
	for _, arm := range arms {
		for _, variant := range variants {
			for _, form := range forms {
				name := arm + "--" + variant + "--" + form + ".jsonl.gz"
				x, err := readPairs(filepath.Join(local, name))
				if err != nil {
					return err
				}
				y, err := readPairs(filepath.Join(remote, name))
				if err != nil {
					return err
				}
				var ca, cb [4]totals
				for i, row := range x {
					other := y[i]
					if row.Original != other.Original {
						return errors.New("paired original input/source/target differs")
					}
					for lane := range lanes {
						ca[lane].add(row.Lanes[lane])
						cb[lane].add(other.Lanes[lane])
					}
					legacy.add(row.Lanes[0], other.Lanes[0])
					separate.add(row.Lanes[2], other.Lanes[2])
					if row.Lanes[2].Features != other.Lanes[2].Features {
						return errors.New("platform feature arrays differ")
					}
					if row.Lanes[2] != other.Lanes[2] {
						changed = append(changed, map[string]any{"journal": name, "view_id": row.Original.View, "input_sha256": row.Original.Input, "observations": [2]observation{row.Lanes[2], other.Lanes[2]}})
					}
				}
				countsA[name], countsB[name] = ca, cb
			}
		}
	}
	if !reflect.DeepEqual(countsA, a.Counts) || !reflect.DeepEqual(countsB, b.Counts) {
		return errors.New("collection counts differ from complete journals")
	}
	status := "PASS"
	if separate.Hidden != 0 || separate.Logits != 0 || separate.Masks != 0 || separate.Orders != 0 || separate.Finite != 0 || separate.MaxProbability > 1e-5 {
		status = "FAIL"
	}
	pa, err := threestudent.FilePin(filepath.Join(local, "report.json"))
	if err != nil {
		return err
	}
	pb, err := threestudent.FilePin(filepath.Join(remote, "report.json"))
	if err != nil {
		return err
	}
	value := map[string]any{"schema": "gooo/separate-arithmetic-comparison/v1", "status": status, "source_revisions": [2]string{a.Source, b.Source}, "computational_sources_equal": true, "platforms": [2]string{a.GOOS + "/" + a.GOARCH, b.GOOS + "/" + b.GOARCH}, "collection_reports": [2]threestudent.Pin{pa, pb}, "legacy": legacy, "separate": separate, "changed_explicit_rows": changed, "new_model_predictions": 0, "new_optimizer_updates": 0, "feature_replays": 36864, "scope": "All paired complete journals, exact explicit feature/hidden/logit bits and discrete outcomes. Probability differences remain measured with a 1e-5 bound; legacy observations keep their own differences."}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	var used int64
	if err = writeFresh(output, append(raw, '\n'), &used); err != nil {
		return err
	}
	fmt.Printf("{\"status\":%q,\"rows\":%d,\"legacy_rank_differences\":%d,\"explicit_rank_differences\":%d,\"explicit_logit_differences\":%d}\n", status, separate.Rows, legacy.Orders, separate.Orders, separate.Logits)
	if status != "PASS" {
		return errors.New("explicit arithmetic comparison failed; complete differences retained")
	}
	return nil
}
