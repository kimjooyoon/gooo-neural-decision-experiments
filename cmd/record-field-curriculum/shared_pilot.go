package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func sharedInitializer(prepared, compiler, revision, output string) {
	rows := readPaired(prepared)
	if output == "" || compiler == "" || len(revision) != 40 {
		panic("shared initializer inputs required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh shared initializer required")
	}
	check(os.MkdirAll(output, 0755))
	raw := invoke(compiler, "version", "--build", "--json")
	var identity struct {
		Source string `json:"compiler_source_sha"`
		State  string `json:"source_status"`
		Go     string `json:"go_version"`
	}
	check(json.Unmarshal(raw, &identity))
	if identity.Source != revision || identity.State != "CLEAN_VCS" || identity.Go != "go1.27.1" {
		panic("exact shared compiler required")
	}
	check(os.WriteFile(filepath.Join(output, "compiler-build.json"), raw, 0644))
	byID := map[string]PairedRow{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	for _, spec := range pairedNativePlan() {
		row := byID[spec.ID()]
		raw = invoke(compiler, "body-context", "--activity", "Select", "--feature-version", jointdecision.RecordSharedFeatureVersion, filepath.Join(prepared, row.SourceFile))
		var captured struct {
			Source      string `json:"original_source_sha256"`
			Predictions int    `json:"model_predictions"`
			Tests       int    `json:"candidate_tests"`
			Context     struct {
				Text    string `json:"text"`
				Version string `json:"feature_version"`
				Status  string `json:"status"`
			} `json:"context"`
		}
		check(json.Unmarshal(raw, &captured))
		if captured.Source != "sha256:"+row.SourceSHA || captured.Predictions != 0 || captured.Tests != 0 ||
			captured.Context.Text != row.Context || captured.Context.Version != jointdecision.RecordSharedFeatureVersion || captured.Context.Status != "ENCODED" {
			panic("shared source projection differs")
		}
		check(os.WriteFile(filepath.Join(output, spec.ID()+"-context.json"), raw, 0644))
	}
	blob := make([]byte, 8288)
	rng, at := rand.New(rand.NewSource(20261055)), 0
	for _, tensor := range [3][2]int{{8 * 256, 256}, {8, 256}, {2 * 8, 8}} {
		bound := 1 / math.Sqrt(float64(tensor[1]))
		for range tensor[0] {
			binary.LittleEndian.PutUint32(blob[at:], math.Float32bits(float32((2*rng.Float64()-1)*bound)))
			at += 4
		}
	}
	check(os.WriteFile(filepath.Join(output, "initial-fp32.bin"), blob, 0644))
	pins := map[string]any{}
	for _, name := range []string{"manifest.json", "rows.jsonl", "features-fp32.bin"} {
		raw, err := os.ReadFile(filepath.Join(prepared, name))
		check(err)
		pins[name] = map[string]any{"sha256": hash(raw), "bytes": len(raw)}
	}
	save(filepath.Join(output, "manifest.json"), map[string]any{
		"schema": "gooo/record-shared-field-initializer/v1", "initial_seed": 20261055, "initial_sha256": hash(blob),
		"initial_bytes": len(blob), "parameters": 2072, "rows": len(rows), "compiler_source": revision,
		"shared_feature_version": jointdecision.RecordSharedFeatureVersion, "prepared_pins": pins, "fresh_shared_contexts": 24,
		"scope": "Full frozen source/feature bank replay; same complete record projection recaptured at 24 registered native views. Fresh Go weights; no fitting or candidate outcomes in shared context exports."})
}

func loadRecordPilot(name string) *jointdecision.ThreeModel {
	raw, err := os.ReadFile(name)
	check(err)
	var meta jointdecision.Metadata
	check(json.Unmarshal(raw, &meta))
	var model *jointdecision.ThreeModel
	if meta.Feature == jointdecision.RecordSharedFeatureVersion {
		model, err = jointdecision.LoadRecordSharedThree(name)
	} else {
		model, err = jointdecision.LoadRecordThree(name)
	}
	check(err)
	return model
}

func predictRecordPilot(model *jointdecision.ThreeModel, text string, w *jointdecision.ThreeWorkspace, p *jointdecision.ThreePrediction) {
	if model.FeatureVersion() == jointdecision.RecordSharedFeatureVersion {
		check(model.PredictRecordSharedInto(text, w, p))
	} else {
		check(model.PredictRecordInto(text, w, p))
	}
}

func auditShared(prepared, models, frozen, output string) {
	if output == "" || models == "" || frozen == "" {
		panic("shared audit inputs required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh shared audit required")
	}
	check(os.MkdirAll(output, 0755))
	rows := readPaired(prepared)
	loaded := map[string]*jointdecision.ThreeModel{}
	for _, variant := range []string{"frozen_field_v2", "fp32", "ptq_ternary", "qat_ternary"} {
		name := filepath.Join(models, variant, "model.json")
		if variant == "frozen_field_v2" {
			name = frozen
		}
		loaded[variant] = loadRecordPilot(name)
		if (variant == "frozen_field_v2") != (loaded[variant].FeatureVersion() == jointdecision.RecordFieldFeatureVersion) {
			panic("shared comparison contract differs")
		}
	}
	parityFile, err := os.Open(filepath.Join(filepath.Dir(models), "go-parity.jsonl"))
	check(err)
	scanner := bufio.NewScanner(parityFile)
	scanner.Buffer(make([]byte, 4096), 8192)
	byID := map[string]PairedRow{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	seen := map[string]bool{}
	maximum := 0.
	for scanner.Scan() {
		var ref struct {
			Variant string    `json:"variant"`
			ID      string    `json:"id"`
			Text    string    `json:"text"`
			Logits  []float64 `json:"logits"`
		}
		check(json.Unmarshal(scanner.Bytes(), &ref))
		row, ok := byID[ref.ID]
		key := ref.Variant + "/" + ref.ID
		if !ok || row.Context != ref.Text || len(ref.Logits) != 8 || seen[key] || !slices.Contains([]string{"fp32", "ptq_ternary", "qat_ternary"}, ref.Variant) {
			panic("shared parity identity differs")
		}
		seen[key] = true
		var w jointdecision.ThreeWorkspace
		var p jointdecision.ThreePrediction
		predictRecordPilot(loaded[ref.Variant], ref.Text, &w, &p)
		for i, want := range ref.Logits {
			diff := math.Abs(float64(p.Logits[i]) - want)
			maximum = math.Max(maximum, diff)
			if math.IsNaN(diff) || math.IsInf(diff, 0) || diff > 5e-5 {
				panic("shared Go/export logit parity differs")
			}
		}
	}
	check(scanner.Err())
	check(parityFile.Close())
	if len(seen) != 120 {
		panic("complete 120-row shared numerical comparison required")
	}
	all := map[string]any{}
	started := time.Now()
	var before, after syscall.Rusage
	check(syscall.Getrusage(syscall.RUSAGE_SELF, &before))
	for _, variant := range []string{"frozen_field_v2", "fp32", "ptq_ternary", "qat_ternary"} {
		model := loaded[variant]
		counts := map[string]map[string]int{}
		times := map[string][]int64{}
		nll := map[string]float64{}
		brier := map[string]float64{}
		f, err := os.OpenFile(filepath.Join(output, variant+"-predictions.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		check(err)
		enc := json.NewEncoder(f)
		var w jointdecision.ThreeWorkspace
		for _, row := range rows {
			var p jointdecision.ThreePrediction
			begin := time.Now()
			predictRecordPilot(model, row.Context, &w, &p)
			elapsed := time.Since(begin).Nanoseconds()
			ranking := rankedMasks(p)
			rank := slices.Index(ranking, row.Target) + 1
			marginals, err := jointdecision.RecordChoiceMarginals(p)
			check(err)
			check(enc.Encode(map[string]any{"id": row.ID, "split": row.Split, "language": row.Language, "target": row.Target, "prediction": p, "ranking": ranking, "target_rank": rank, "field_probabilities": marginals, "text_prediction_ns": elapsed}))
			for _, key := range []string{row.Split, row.Split + "/" + row.Language} {
				if counts[key] == nil {
					counts[key] = map[string]int{}
				}
				c := counts[key]
				c["total"]++
				c["field_total"] += 3
				for _, budget := range []int{1, 2, 4, 8} {
					if rank <= budget {
						c[fmt.Sprintf("target_in_first_%d", budget)]++
					}
				}
				for part := range 3 {
					bit := (row.Target >> part) & 1
					if (ranking[0]>>part)&1 == bit {
						c["fields_correct"]++
					}
					delta := marginals[part][1] - float64(bit)
					brier[key] += delta * delta
				}
				nll[key] -= math.Log(math.Max(float64(p.Probabilities[row.Target]), 1e-30))
			}
			if strings.HasPrefix(row.Split, "test_") {
				times[row.Split] = append(times[row.Split], elapsed)
			}
		}
		check(f.Close())
		medians := map[string]int64{}
		for key, values := range times {
			slices.Sort(values)
			medians[key] = (values[(len(values)-1)/2] + values[len(values)/2]) / 2
		}
		for key := range counts {
			nll[key] /= float64(counts[key]["total"])
			brier[key] /= float64(counts[key]["field_total"])
		}
		kernel := map[string]any{}
		if variant != "frozen_field_v2" {
			var features [768]float32
			check(jointdecision.FeaturesIntoRecordThree(rows[0].Context, &features))
			var scratch jointdecision.ThreeWorkspace
			var p jointdecision.ThreePrediction
			allocations := testing.AllocsPerRun(1000, func() { check(model.PredictRecordSharedFeaturesInto(&features, &scratch, &p)) })
			durations := make([]int64, 4096)
			for i := range durations {
				begin := time.Now()
				check(model.PredictRecordSharedFeaturesInto(&features, &scratch, &p))
				durations[i] = time.Since(begin).Nanoseconds()
			}
			slices.Sort(durations)
			kernel = map[string]any{"fixed_row": rows[0].ID, "timed_predictions": 4096, "allocation_probe_predictions": 1001, "heap_allocations_per_call": allocations, "median_ns": (durations[2047] + durations[2048]) / 2, "p95_ns": durations[3891], "scope": "Prepared-array compute on one fixed row; text parsing excluded. Sequential controls, no throughput speedup claim."}
		}
		all[variant] = map[string]any{"counts": counts, "mask_nll": nll, "field_brier": brier, "text_predict_ns_median": medians, "prepared_kernel": kernel, "metadata_sha256": model.MetadataSHA256(), "weights_sha256": model.WeightsSHA256(), "weight_file_bytes": model.PackedFileBytes(), "resident_tensor_bytes": model.ResidentTensorBytes(), "scale_bytes": model.MatrixScaleBytes()}
	}
	check(syscall.Getrusage(syscall.RUSAGE_SELF, &after))
	wall := time.Since(started).Seconds()
	cpu := float64(after.Utime.Sec-before.Utime.Sec+after.Stime.Sec-before.Stime.Sec) + float64(after.Utime.Usec-before.Utime.Usec+after.Stime.Usec-before.Stime.Usec)/1e6
	rss := after.Maxrss
	if runtime.GOOS != "darwin" {
		rss *= 1024
	}
	save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/record-shared-field-audit/v1", "rows": len(rows), "parity_records": len(seen), "max_logit_error": maximum, "models": all, "wall_seconds": wall, "cpu_seconds": cpu, "process_cpu_percent_one_core": 100 * cpu / wall, "lifetime_peak_rss_bytes": rss, "scope": "Frozen 7,168 reused benchmark views; diagnostics remain separate. All eight goals seen during training. Per-field marginals are ranking probabilities and do not prove arbitrary natural-language meaning. Complete source/Go feature replay before numerical and quality observations."})
}
