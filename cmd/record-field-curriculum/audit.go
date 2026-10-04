package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

type MaskPrediction struct {
	ID         string                        `json:"id"`
	Split      string                        `json:"split"`
	Language   string                        `json:"language"`
	Target     uint16                        `json:"observed_target_mask"`
	Prediction jointdecision.ThreePrediction `json:"prediction"`
	Ranking    []uint16                      `json:"ranking"`
	Rank       int                           `json:"observed_target_rank"`
	PredictNS  int64                         `json:"predict_ns"`
}

func readRows(prepared string) []Row {
	manifestRaw, err := os.ReadFile(filepath.Join(prepared, "manifest.json"))
	check(err)
	var manifest struct {
		Complete bool   `json:"complete"`
		Total    int    `json:"total"`
		Feature  string `json:"feature_version"`
		Files    map[string]struct {
			SHA   string `json:"sha256"`
			Bytes int    `json:"bytes"`
		} `json:"files"`
	}
	check(json.Unmarshal(manifestRaw, &manifest))
	if !manifest.Complete || manifest.Total != 768 || manifest.Feature != jointdecision.RecordFieldFeatureVersion || len(manifest.Files) != 4 {
		panic("complete preparation manifest required")
	}
	for _, name := range []string{"rows.jsonl", "features-fp32.bin", "initial-fp32.bin", "compiler-build.json"} {
		data, err := os.ReadFile(filepath.Join(prepared, name))
		check(err)
		pin, ok := manifest.Files[name]
		if !ok || pin.Bytes != len(data) || pin.SHA != hash(data) {
			panic("prepared file differs from manifest")
		}
	}
	raw, err := os.ReadFile(filepath.Join(prepared, "rows.jsonl"))
	check(err)
	features, err := os.ReadFile(filepath.Join(prepared, "features-fp32.bin"))
	check(err)
	var rows []Row
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 8192)
	for scanner.Scan() {
		var row Row
		check(json.Unmarshal(scanner.Bytes(), &row))
		source, err := os.ReadFile(filepath.Join(prepared, row.SourceFile))
		check(err)
		if hash(source) != row.SourceSHA || hash([]byte(row.Context)) != row.ContextSHA || row.Passed != 5 || row.Total != 5 || row.FieldsPassed != 15 || row.FieldsTotal != 15 {
			panic("source/observation identity differs")
		}
		var projected [jointdecision.ThreeFeatureDim]float32
		check(jointdecision.FeaturesIntoRecordThree(row.Context, &projected))
		var blob [jointdecision.ThreeFeatureDim * 4]byte
		for i, value := range projected {
			binary.LittleEndian.PutUint32(blob[4*i:], math.Float32bits(value))
		}
		at := len(rows) * len(blob)
		if hash(blob[:]) != row.FeatureSHA || at+len(blob) > len(features) || !bytes.Equal(blob[:], features[at:at+len(blob)]) {
			panic("Go source feature bank does not replay")
		}
		rows = append(rows, row)
	}
	check(scanner.Err())
	if len(rows) != 768 || len(features) != len(rows)*768*4 {
		panic("complete feature inventory required")
	}
	return rows
}

func audit(prepared, models, output string) {
	if prepared == "" || models == "" || output == "" {
		panic("prepared/models/output required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh audit directory required")
	}
	check(os.MkdirAll(output, 0755))
	rows := readRows(prepared)
	summaries := map[string]any{}
	loaded := map[string]*jointdecision.ThreeModel{}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		model, err := jointdecision.LoadRecordThree(filepath.Join(models, variant, "model.json"))
		check(err)
		loaded[variant] = model
		file, err := os.OpenFile(filepath.Join(output, variant+"-predictions.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		check(err)
		encoder := json.NewEncoder(file)
		counts := map[string]map[string]int{}
		durations := []int64{}
		var workspace jointdecision.ThreeWorkspace
		for _, row := range rows {
			var prediction jointdecision.ThreePrediction
			started := time.Now()
			check(model.PredictRecordInto(row.Context, &workspace, &prediction))
			elapsed := time.Since(started).Nanoseconds()
			ranking := []uint16{0, 1, 2, 3, 4, 5, 6, 7}
			slices.SortFunc(ranking, func(a, b uint16) int {
				if prediction.Probabilities[a] > prediction.Probabilities[b] {
					return -1
				}
				if prediction.Probabilities[a] < prediction.Probabilities[b] {
					return 1
				}
				return int(a) - int(b)
			})
			rank := slices.Index(ranking, row.Target) + 1
			check(encoder.Encode(MaskPrediction{row.ID, row.Split, row.Language, row.Target, prediction, ranking, rank, elapsed}))
			for _, key := range []string{row.Split, row.Split + "/" + row.Language} {
				if counts[key] == nil {
					counts[key] = map[string]int{}
				}
				counts[key]["total"]++
				if rank == 1 {
					counts[key]["top_mask_matches"]++
				}
				counts[key]["target_rank_sum"] += rank
				for _, budget := range []int{1, 2, 4, 8} {
					if rank <= budget {
						counts[key]["target_in_first_"+strconv.Itoa(budget)]++
					}
				}
			}
			if row.Split == "test" {
				durations = append(durations, elapsed)
			}
		}
		check(file.Close())
		slices.Sort(durations)
		summaries[variant] = map[string]any{"counts": counts, "test_predict_ns_median": (durations[95] + durations[96]) / 2, "metadata_sha256": model.MetadataSHA256(), "weights_sha256": model.WeightsSHA256(), "weight_file_bytes": model.PackedFileBytes(), "resident_tensor_bytes": model.ResidentTensorBytes(), "scale_bytes": model.MatrixScaleBytes(), "arithmetic_version": model.ArithmeticVersion()}
	}
	parityFile, err := os.Open(filepath.Join(filepath.Dir(models), "go-parity.jsonl"))
	check(err)
	defer parityFile.Close()
	scanner := bufio.NewScanner(parityFile)
	scanner.Buffer(make([]byte, 4096), 8192)
	maximum := 0.
	total := 0
	for scanner.Scan() {
		var record struct {
			Variant string    `json:"variant"`
			Text    string    `json:"text"`
			Logits  []float64 `json:"logits"`
		}
		check(json.Unmarshal(scanner.Bytes(), &record))
		if len(record.Logits) != 8 || loaded[record.Variant] == nil {
			panic("parity identity differs")
		}
		var workspace jointdecision.ThreeWorkspace
		var prediction jointdecision.ThreePrediction
		check(loaded[record.Variant].PredictRecordInto(record.Text, &workspace, &prediction))
		for i, want := range record.Logits {
			difference := math.Abs(float64(prediction.Logits[i]) - want)
			maximum = math.Max(maximum, difference)
			if difference > 5e-5 {
				panic("exported/Go logit parity differs")
			}
		}
		total++
	}
	check(scanner.Err())
	if total != 192 {
		panic("complete numerical parity inventory required")
	}
	save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/record-field-model-audit/v1", "feature_bank_replayed": true, "rows": len(rows), "parity_records": total, "maximum_logit_absolute_error": maximum, "models": summaries, "scope": "Complete source/feature byte replay, numerical export comparison and source-mask ordering. Native case/field completeness is measured separately. Test prediction medians follow train/calibration warmup; all row timings retained."})
}
