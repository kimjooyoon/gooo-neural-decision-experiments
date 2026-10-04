package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func readPaired(prepared string) []PairedRow {
	raw, err := os.ReadFile(filepath.Join(prepared, "manifest.json"))
	check(err)
	var manifest struct {
		Schema   string `json:"schema"`
		Complete bool   `json:"complete"`
		Total    int    `json:"total"`
		Feature  string `json:"feature_version"`
		Files    map[string]struct {
			SHA   string `json:"sha256"`
			Bytes int    `json:"bytes"`
		} `json:"files"`
	}
	check(json.Unmarshal(raw, &manifest))
	if manifest.Schema != "gooo/record-field-paired-training-inputs/v2" || !manifest.Complete || manifest.Total != pairedTotal || manifest.Feature != jointdecision.RecordFieldFeatureVersion || len(manifest.Files) != 4 {
		panic("complete paired preparation required")
	}
	for _, name := range []string{"rows.jsonl", "features-fp32.bin", "initial-fp32.bin", "compiler-build.json"} {
		data, err := os.ReadFile(filepath.Join(prepared, name))
		check(err)
		pin, ok := manifest.Files[name]
		if !ok || pin.SHA != hash(data) || pin.Bytes != len(data) {
			panic("paired file pin differs")
		}
	}
	raw, err = os.ReadFile(filepath.Join(prepared, "rows.jsonl"))
	check(err)
	features, err := os.ReadFile(filepath.Join(prepared, "features-fp32.bin"))
	check(err)
	plan := pairedPlan()
	rows := make([]PairedRow, 0, pairedTotal)
	scan := bufio.NewScanner(bytes.NewReader(raw))
	scan.Buffer(make([]byte, 4096), 8192)
	for scan.Scan() {
		if len(rows) >= len(plan) {
			panic("extra paired row")
		}
		s := plan[len(rows)]
		var r PairedRow
		check(json.Unmarshal(scan.Bytes(), &r))
		if r.ID != s.ID() || r.Family != s.Family || r.Language != s.Language || r.Split != s.Split || r.Goal != s.Goal || r.Style != s.Style || r.Permutation != s.Permutation || r.Orientation != s.Orientation || r.Target != goalMask(orders[s.Permutation], s.Orientation, s.Goal) || r.SourceFile != filepath.Join("sources", s.ID()+".gooo.fixture") {
			panic("paired row differs from registered inventory")
		}
		source, err := os.ReadFile(filepath.Join(prepared, r.SourceFile))
		check(err)
		if !bytes.Equal(source, goalFixture(s.Family, orders[s.Permutation], s.Orientation, s.Language, s.Goal, s.Style)) || hash(source) != r.SourceSHA || hash([]byte(r.Context)) != r.ContextSHA || r.Passed != 5 || r.Total != 5 || r.FieldsPassed != 15 || r.FieldsTotal != 15 {
			panic("paired source or finite observation differs")
		}
		var projected [768]float32
		check(jointdecision.FeaturesIntoRecordThree(r.Context, &projected))
		var blob [768 * 4]byte
		for i, v := range projected {
			binary.LittleEndian.PutUint32(blob[4*i:], math.Float32bits(v))
		}
		at := len(rows) * len(blob)
		if at+len(blob) > len(features) || hash(blob[:]) != r.FeatureSHA || !bytes.Equal(blob[:], features[at:at+len(blob)]) {
			panic("paired Go feature replay differs")
		}
		capture, err := os.ReadFile(filepath.Join(prepared, "context-captures", s.ID()+".json"))
		check(err)
		var context struct {
			Source      string `json:"original_source_sha256"`
			Predictions int    `json:"model_predictions"`
			Tests       int    `json:"candidate_tests"`
			Context     struct {
				Text    string `json:"text"`
				Status  string `json:"status"`
				Version string `json:"feature_version"`
			} `json:"context"`
		}
		check(json.Unmarshal(capture, &context))
		if context.Source != "sha256:"+r.SourceSHA || context.Predictions != 0 || context.Tests != 0 || context.Context.Text != r.Context || context.Context.Status != "ENCODED" || context.Context.Version != manifest.Feature {
			panic("paired source-only context replay differs")
		}
		rows = append(rows, r)
	}
	check(scan.Err())
	if len(rows) != pairedTotal || len(features) != pairedTotal*768*4 {
		panic("complete paired rows required")
	}
	return rows
}

func rankedMasks(p jointdecision.ThreePrediction) []uint16 {
	ranking := []uint16{0, 1, 2, 3, 4, 5, 6, 7}
	slices.SortFunc(ranking, func(a, b uint16) int {
		if p.Probabilities[a] > p.Probabilities[b] {
			return -1
		}
		if p.Probabilities[a] < p.Probabilities[b] {
			return 1
		}
		return int(a) - int(b)
	})
	return ranking
}

func auditPaired(prepared, models, frozen, output string) {
	if prepared == "" || models == "" || frozen == "" || output == "" {
		panic("paired audit inputs required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh paired audit required")
	}
	check(os.MkdirAll(output, 0755))
	rows := readPaired(prepared)
	loaded := map[string]*jointdecision.ThreeModel{}
	summaries := map[string]any{}
	for _, variant := range []string{"frozen_field_v1", "fp32", "ptq_ternary", "qat_ternary"} {
		path := filepath.Join(models, variant, "model.json")
		if variant == "frozen_field_v1" {
			path = frozen
		}
		model, err := jointdecision.LoadRecordThree(path)
		check(err)
		loaded[variant] = model
	}
	total, maximum := pairedParity(models, rows, loaded)
	for _, variant := range []string{"frozen_field_v1", "fp32", "ptq_ternary", "qat_ternary"} {
		model := loaded[variant]
		file, err := os.OpenFile(filepath.Join(output, variant+"-predictions.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		check(err)
		encoder := json.NewEncoder(file)
		counts := map[string]map[string]int{}
		durations := map[string][]int64{}
		entropy := map[string]float64{}
		var workspace jointdecision.ThreeWorkspace
		for _, r := range rows {
			var p jointdecision.ThreePrediction
			started := time.Now()
			check(model.PredictRecordInto(r.Context, &workspace, &p))
			elapsed := time.Since(started).Nanoseconds()
			ranking := rankedMasks(p)
			rank := slices.Index(ranking, r.Target) + 1
			check(encoder.Encode(MaskPrediction{r.ID, r.Split, r.Language, r.Target, p, ranking, rank, elapsed}))
			for _, key := range []string{r.Split, r.Split + "/" + r.Language, fmt.Sprintf("%s/goal_%d", r.Split, r.Goal), fmt.Sprintf("%s/family_%d", r.Split, r.Family)} {
				if counts[key] == nil {
					counts[key] = map[string]int{}
				}
				c := counts[key]
				c["total"]++
				c["target_rank_sum"] += rank
				for _, budget := range []int{1, 2, 4, 8} {
					if rank <= budget {
						c["target_in_first_"+strconv.Itoa(budget)]++
					}
				}
				c["field_requirements_total"] += 3
				for pos, role := range orders[r.Permutation] {
					if (ranking[0]>>pos)&1 == (r.Target>>pos)&1 {
						c["field_requirements_correct"]++
						c["role_"+strconv.Itoa(role)+"_correct"]++
					}
				}
				for _, prob := range p.Probabilities {
					if prob > 0 {
						entropy[key] -= float64(prob) * math.Log2(float64(prob))
					}
				}
			}
			if strings.HasPrefix(r.Split, "test_") {
				durations[r.Split] = append(durations[r.Split], elapsed)
			}
		}
		check(file.Close())
		medians := map[string]int64{}
		for k, v := range durations {
			slices.Sort(v)
			medians[k] = (v[(len(v)-1)/2] + v[len(v)/2]) / 2
		}
		for key := range entropy {
			entropy[key] /= float64(counts[key]["total"])
		}
		summaries[variant] = map[string]any{"counts": counts, "predict_ns_median": medians, "mean_entropy_bits": entropy, "metadata_sha256": model.MetadataSHA256(), "weights_sha256": model.WeightsSHA256(), "weight_file_bytes": model.PackedFileBytes(), "resident_tensor_bytes": model.ResidentTensorBytes(), "scale_bytes": model.MatrixScaleBytes()}
	}
	save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/record-paired-model-audit/v2", "rows": len(rows), "source_and_feature_bank_replayed": true, "parity_records": total, "maximum_logit_absolute_error": maximum, "models": summaries, "scope": "Complete registered source/context/feature replay and numerical parity before quality evaluation. Distinct source/wording diagnostic axes; seen-body wording shares skeletons intentionally. Known v1 families are reused benchmarks. Warm row medians are not an end-to-end speedup; frozen v1 field model is evaluated on the same goal contexts."})
}

func pairedParity(models string, rows []PairedRow, loaded map[string]*jointdecision.ThreeModel) (int, float64) {
	file, err := os.Open(filepath.Join(filepath.Dir(models), "go-parity.jsonl"))
	check(err)
	defer file.Close()
	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 4096), 8192)
	total, maximum := 0, 0.
	seen := map[string]bool{}
	byID := map[string]PairedRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	for scan.Scan() {
		var record struct {
			Variant string    `json:"variant"`
			ID      string    `json:"id"`
			Text    string    `json:"text"`
			Logits  []float64 `json:"logits"`
		}
		check(json.Unmarshal(scan.Bytes(), &record))
		r, ok := byID[record.ID]
		key := record.Variant + "/" + record.ID
		if !ok || record.Text != r.Context || seen[key] || len(record.Logits) != 8 || loaded[record.Variant] == nil || record.Variant == "frozen_field_v1" {
			panic("paired parity identity differs")
		}
		seen[key] = true
		var w jointdecision.ThreeWorkspace
		var p jointdecision.ThreePrediction
		check(loaded[record.Variant].PredictRecordInto(record.Text, &w, &p))
		for i, want := range record.Logits {
			diff := math.Abs(float64(p.Logits[i]) - want)
			maximum = math.Max(maximum, diff)
			if math.IsNaN(diff) || math.IsInf(diff, 0) || diff > 5e-5 {
				panic("paired exported/Go parity differs")
			}
		}
		total++
	}
	check(scan.Err())
	if total != 120 {
		panic("complete paired parity inventory required")
	}
	return total, maximum
}
