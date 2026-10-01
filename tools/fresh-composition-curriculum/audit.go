package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compositionstudy"
)

// Streaming digests bound collection/audit memory independently of evidence size.
func fileDigest(name string) (string, int64, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	digest := sha256.New()
	var buffer [32 << 10]byte
	count, err := io.CopyBuffer(digest, file, buffer[:])
	if err != nil {
		return "", count, err
	}
	return hex.EncodeToString(digest.Sum(nil)), count, nil
}

func scanner(file *os.File) *bufio.Scanner {
	s := bufio.NewScanner(file)
	s.Buffer(make([]byte, 64<<10), 1<<20)
	return s
}

func auditCollection(directory, reportPath string) error {
	manifestRaw, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Status   string `json:"status"`
		Dataset  string `json:"dataset_sha256"`
		Captures string `json:"captures_sha256"`
		Rows     int    `json:"decision_rows"`
		Calls    int    `json:"native_export_calls"`
		Source   string `json:"runner_revision"`
	}
	if json.Unmarshal(manifestRaw, &manifest) != nil || manifest.Status != "SOURCE_BOUND_EXPORTED" || manifest.Rows != 9216 || manifest.Calls != 4608 {
		return errors.New("complete fixed manifest required")
	}
	for name, expected := range map[string]string{"dataset.jsonl": manifest.Dataset, "exports.jsonl": manifest.Captures} {
		actual, _, err := fileDigest(filepath.Join(directory, name))
		if err != nil || actual != expected {
			return errors.New("collection file digest mismatch")
		}
	}
	data, err := os.Open(filepath.Join(directory, "dataset.jsonl"))
	if err != nil {
		return err
	}
	defer data.Close()
	capture, err := os.Open(filepath.Join(directory, "exports.jsonl"))
	if err != nil {
		return err
	}
	defer capture.Close()
	rows, captures := scanner(data), scanner(capture)
	count, calls, ambiguous, maximum := 0, 0, 0, 0
	var times []int64
	splits := map[string]int{}
	// Exact byte-identical inputs with different finite marginals are diagnostic
	// representation limits. Keep every row and tie; do not relabel or drop them.
	inputTargets := map[string][2]float32{}
	conflicting := map[string]bool{}
	for _, family := range compositionstudy.Families {
		for config := 0; config < 48; config++ {
			for desired := 0; desired < 4; desired++ {
				for _, language := range [2]string{"en", "ko"} {
					doc, source, target, err := fixture(family, config, desired, language)
					if err != nil {
						return err
					}
					for _, feature := range featureArms {
						if !captures.Scan() {
							return errors.New("native captures missing")
						}
						var c struct {
							Schema    string `json:"schema"`
							ID        string `json:"id"`
							Raw       []byte `json:"native_receipt"`
							Wall      int64  `json:"wall_ns"`
							Succeeded bool   `json:"child_succeeded"`
						}
						if json.Unmarshal(captures.Bytes(), &c) != nil || c.Schema != "gooo/fresh-composition-native-capture/v2" || !c.Succeeded || c.Wall <= 0 {
							return errors.New("actual native capture envelope differs")
						}
						group := fmt.Sprintf("%s-c%02d-goal%d", family, config, desired)
						if c.ID != group+"-"+language+"-"+feature {
							return errors.New("exact native view identity differs")
						}
						value, err := inspect(c.Raw, doc, source, feature)
						if err != nil {
							return err
						}
						calls++
						times = append(times, c.Wall)
						for i, in := range value.Inputs {
							if !rows.Scan() {
								return errors.New("decision rows missing")
							}
							var r row
							if json.Unmarshal(rows.Bytes(), &r) != nil {
								return errors.New("decision row JSON differs")
							}
							choice := doc.Plan.Decisions[i]
							if r.Group != group || r.Pair != fmt.Sprintf("%s-%s-choice%d", group, feature, i) {
								return errors.New("program/bilingual group identity differs")
							}
							if r.ID != c.ID+"-"+choice.ID || r.Family != family || r.Config != config || r.Desired != desired || r.Language != language || r.Split != compositionstudy.Split(config) || r.Template != compositionstudy.TemplateID(choice.Kind, config, language) || r.Feature != feature || r.Coordinate != i || r.Options != [2]string{choice.Options[0].Label, choice.Options[1].Label} || r.Targets != target.Marginals[i] || !reflect.DeepEqual(r.Finite, target) || r.SourceSHA != hash(source) || r.DocumentSHA != value.Document || r.CaptureSHA != hash(c.Raw) || r.Input != in {
								return errors.New("row/source/target/native capture binding differs")
							}
							key := feature + "/" + r.Input.SHA
							if prior, ok := inputTargets[key]; ok && prior != r.Targets {
								conflicting[key] = true
							}
							inputTargets[key] = r.Targets
							if len(target.BestMasks) > 1 {
								ambiguous++
							}
							maximum = max(maximum, len(in.Text))
							splits[feature+"/"+r.Split]++
							count++
						}
					}
				}
			}
		}
	}
	if rows.Scan() || captures.Scan() || rows.Err() != nil || captures.Err() != nil || count != 9216 || calls != 4608 {
		return errors.New("exact complete audit denominator differs")
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	unique, conflicts := map[string]int{}, map[string]int{}
	for key := range inputTargets {
		for _, feature := range featureArms {
			if len(key) > len(feature) && key[:len(feature)+1] == feature+"/" {
				unique[feature]++
			}
		}
	}
	for key := range conflicting {
		for _, feature := range featureArms {
			if len(key) > len(feature) && key[:len(feature)+1] == feature+"/" {
				conflicts[feature]++
			}
		}
	}
	return save(reportPath, map[string]any{"schema": "gooo/fresh-composition-collection-audit/v1", "status": "PASS", "collection_runner_revision": manifest.Source, "dataset_sha256": manifest.Dataset, "captures_sha256": manifest.Captures, "manifest_sha256": hash(manifestRaw), "native_captures_audited": calls, "decision_rows_audited": count, "ambiguous_finite_target_decision_rows": ambiguous, "maximum_complete_input_bytes": maximum, "split_decision_rows": splits, "distinct_complete_input_texts_by_feature_arm": unique, "distinct_byte_identical_inputs_with_conflicting_finite_marginals": conflicts, "native_export_child_wall_median_ns": times[len(times)/2], "native_export_child_wall_p95_ns": times[len(times)*95/100], "new_native_calls_by_audit": 0, "new_model_predictions": 0, "new_optimizer_updates": 0, "cpu_rss_measured": false, "scope": "streaming captured zero-prediction native export audit and independent finite arithmetic reconstruction; latency is source-binding/export child wall time, not model inference or host CPU increase; identical-input finite target conflicts remain evidence"})
}
