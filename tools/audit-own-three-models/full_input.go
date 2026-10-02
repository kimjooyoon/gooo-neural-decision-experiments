package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const fullTrainingSource = "d23bd7e9e9fa3f197ffb5ee8d10872ad982e29ca"
const fullProtocolSHA = "798df9e2b42b425e40ca003455b9bf0b6934a31a476cd052fa40c2c596c080b0"

var fullArms = [4]string{"positioned-original", "positioned-varied", "bag-original", "bag-varied"}

type fullParity struct {
	parityRow
	Feature string `json:"feature_version"`
	Form    string `json:"form"`
	Input   string `json:"input_sha256"`
}

type fullExport struct {
	exportPin
	Feature string `json:"feature_version"`
}

type fullModelObservation struct {
	ExpandedMetadata string                 `json:"expanded_metadata_sha256"`
	CompactMetadata  string                 `json:"compact_metadata_sha256"`
	CompactWeights   string                 `json:"compact_weights_sha256"`
	Feature          string                 `json:"feature_version"`
	ParityCalls      int                    `json:"expanded_export_parity_predictions"`
	CompactCalls     int                    `json:"compact_parity_predictions"`
	MaxError         float64                `json:"maximum_absolute_export_parity_error"`
	Conditions       map[string]development `json:"development_conditions"`
	PairValidity     map[string][3]int      `json:"both_valid_different_valid_same_wrong_by_condition"`
	StableNLL        map[string]float64     `json:"stable_float64_passing_set_nll_sum_by_condition"`
}

func fullVersion(arm string) string {
	if strings.HasPrefix(arm, "bag-") {
		return jointdecision.ThreeBagFeatureVersion
	}
	return jointdecision.ThreeFeatureVersion
}

func fullExpectedParity(states []threefeedback.State) ([]fullParity, error) {
	var result []fullParity
	for _, language := range []string{"en", "ko"} {
		for _, form := range []string{"development", "original", "request-prefix", "please-prefix", "request-suffix", "please-suffix"} {
			split, phase, limit, name := "train", "feedback", 2, form
			if form == "development" {
				split, phase, limit, name = "development", "initial", 16, "original"
			}
			count := 0
			for _, s := range states {
				if s.Split != split || s.Phase != phase || s.Language != language || count >= limit {
					continue
				}
				input, err := fullinputstudy.Apply(s.Text, language, name)
				if err != nil {
					return nil, err
				}
				if input.Declined {
					continue
				}
				result = append(result, fullParity{parityRow: parityRow{ID: s.ID + "/form/" + name, Phase: phase, Text: input.Text}, Form: name, Input: threecohort.SHA([]byte(input.Text))})
				count++
			}
			if count != limit {
				return nil, errors.New("full-input parity partition incomplete")
			}
		}
	}
	return result, nil
}

func fullParityRows(path string) ([]fullParity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), 1<<20)
	var result []fullParity
	for scan.Scan() {
		var r fullParity
		if err = threecohort.Decode(scan.Bytes(), &r); err != nil {
			return nil, err
		}
		if len(result) >= 156 {
			return nil, errors.New("extra full-input parity row")
		}
		result = append(result, r)
	}
	if scan.Err() != nil || len(result) != 156 {
		return nil, errors.New("complete full-input parity journal required")
	}
	return result, nil
}

// This audit records the initial development comparison. The complete protocol
// additionally requires all-split wording studies, resource probes and native runs.
func auditFullInput(root, dataset, teacher, output, revision string) (failure error) {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact audit source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean audit source required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh full-input audit directory required")
	}
	if err = fullAuditStorage(output); err != nil {
		return err
	}
	protocol, err := threestudent.FilePin(fullinputstudy.Protocol)
	if err != nil || protocol.SHA != fullProtocolSHA {
		return errors.New("published full-input protocol differs")
	}
	var pre struct {
		Schema, Device string
		Source         string            `json:"source_revision"`
		Protocol       string            `json:"protocol_sha256"`
		Sources        map[string]string `json:"sources_sha256"`
		Initial        string            `json:"initial_state_sha256"`
		Steps          int               `json:"planned_optimizer_updates"`
		Parameters     int               `json:"trainable_parameters_per_arm"`
		Arms           []string          `json:"arms"`
		Promoted       bool              `json:"default_model_promoted"`
	}
	if err = read(filepath.Join(root, "preexecution.json"), &pre); err != nil {
		return err
	}
	if pre.Schema != "gooo/full-input-judgment-training/v1" || pre.Source != fullTrainingSource || pre.Protocol != fullProtocolSHA || pre.Device != "mps" || pre.Initial != threecohort.SHA(sharedInitial()) || pre.Steps != 6400 || pre.Parameters != 2072 || !sameJSON(pre.Arms, fullArms[:]) || pre.Promoted || len(pre.Sources) != 6 {
		return errors.New("frozen full-input training preexecution differs")
	}
	for _, name := range []string{"train_full_input_judgment_v1.py", "full_input_evidence_v1.py", "train_shared_three_judgment_v1.py", "shared_evidence_v1.py", "train_own_three_feedback_v1.py", "train_pilot_v2.py"} {
		pin, e := threestudent.FilePin(filepath.Join("training", name))
		if e != nil || pin.SHA != pre.Sources[name] {
			return errors.New("offline optimizer source differs")
		}
	}
	var training struct {
		Schema   string                            `json:"schema"`
		Status   string                            `json:"status"`
		Updates  int                               `json:"optimizer_updates"`
		Pre      string                            `json:"preexecution_sha256"`
		Training map[string]map[string]stageReport `json:"training"`
		Exports  map[string]map[string]fullExport  `json:"exports"`
		Promoted bool                              `json:"default_model_promoted"`
	}
	if err = read(filepath.Join(root, "report.json"), &training); err != nil {
		return err
	}
	prePin, err := threestudent.FilePin(filepath.Join(root, "preexecution.json"))
	if err != nil || training.Schema != "gooo/full-input-judgment-training-result/v1" || training.Status != "TRAINED_AND_EXPORTED" || training.Updates != 6400 || training.Pre != prePin.SHA || len(training.Training) != 4 || len(training.Exports) != 4 || training.Promoted {
		return errors.New("complete four-arm training report required")
	}
	states, err := threestudent.Load(dataset, teacher, "publication/own-three-choice-teacher-audit-20261002.json")
	if err != nil {
		return err
	}
	expected, err := fullExpectedParity(states)
	if err != nil {
		return err
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	defer func() {
		if failure != nil {
			_ = fullWriteJSON(filepath.Join(output, "failure.json"), map[string]any{"status": "FAILED_PREFIX_RETAINED", "error": failure.Error(), "new_optimizer_updates": 0})
		}
	}()
	observed := map[string]fullModelObservation{}
	for _, arm := range fullArms {
		if len(training.Training[arm]) != 2 || len(training.Exports[arm]) != 3 {
			return errors.New("missing full-input stage or export")
		}
		fp, qat := training.Training[arm]["fp32"], training.Training[arm]["qat_ternary"]
		if fp.InitialSHA != pre.Initial || qat.InitialSHA != fp.SelectedSHA {
			return errors.New("fresh shared or QAT initializer differs")
		}
		var duplicate map[string]stageReport
		if err = read(filepath.Join(root, arm, "training-report.json"), &duplicate); err != nil || !sameJSON(duplicate, training.Training[arm]) {
			return errors.New("full-input stage duplicate differs")
		}
		count := 9715
		if strings.HasSuffix(arm, "-varied") {
			count *= 5 // This actual prepared bank retained every full form.
		}
		for name, stage := range map[string]stageReport{"fp32": fp, "qat": qat} {
			if err = verifyStageRows(filepath.Join(root, arm, name), arm, name, stage, count); err != nil {
				return err
			}
		}
		rows, err := fullParityRows(filepath.Join(root, arm, "go-parity.jsonl"))
		if err != nil {
			return err
		}
		for vi, variant := range variants {
			path := filepath.Join(root, arm, "models", variant, "model.json")
			load := jointdecision.LoadThree
			if fullVersion(arm) == jointdecision.ThreeBagFeatureVersion {
				load = jointdecision.LoadThreeBag
			}
			model, err := load(path)
			if err != nil {
				return err
			}
			pin := training.Exports[arm][variant]
			var meta jointdecision.Metadata
			if err = read(path, &meta); err != nil {
				return err
			}
			if model.MetadataSHA256() != pin.Metadata || model.WeightsSHA256() != pin.Weights || model.Variant() != variant || model.FeatureVersion() != pin.Feature || pin.Feature != fullVersion(arm) || model.PackedFileBytes() != pin.Packed || meta.Temperature != pin.Temperature || !validTemperature(pin.Temperature) || !finite(pin.NLL) || variant == "fp32" && (pin.Weights != fp.SelectedSHA || pin.Temperature != fp.Temperature) || variant == "qat_ternary" && pin.Temperature != qat.Temperature {
				return errors.New("full-input model identity or calibration differs")
			}
			metadata, weights, err := jointdecision.CompactThree(model)
			if err != nil {
				return err
			}
			compactDir := filepath.Join(output, "compact", arm, variant)
			if err = os.MkdirAll(compactDir, 0755); err != nil {
				return err
			}
			metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				return err
			}
			if err = os.WriteFile(filepath.Join(compactDir, "model.json"), append(metadataBytes, '\n'), 0644); err != nil {
				return err
			}
			if err = os.WriteFile(filepath.Join(compactDir, "weights.bin"), weights, 0644); err != nil {
				return err
			}
			compact, err := jointdecision.LoadSharedThree(filepath.Join(compactDir, "model.json"))
			if err != nil {
				return err
			}
			o := fullModelObservation{ExpandedMetadata: model.MetadataSHA256(), CompactMetadata: compact.MetadataSHA256(), CompactWeights: compact.WeightsSHA256(), Feature: model.FeatureVersion(), Conditions: map[string]development{}, PairValidity: map[string][3]int{}, StableNLL: map[string]float64{}}
			var w, cw jointdecision.ThreeWorkspace
			var p, cp jointdecision.ThreePrediction
			for i, want := range expected {
				r := rows[vi*52+i]
				if r.Variant != variant || r.Feature != fullVersion(arm) || r.ID != want.ID || r.Phase != want.Phase || r.Text != want.Text || r.Form != want.Form || r.Input != want.Input {
					return errors.New("source-bound full-input parity row differs")
				}
				if err = model.PredictInto(r.Text, &w, &p); err != nil {
					return err
				}
				o.ParityCalls++
				if err = compact.PredictInto(r.Text, &cw, &cp); err != nil {
					return err
				}
				o.CompactCalls++
				if w != cw || p != cp || p.Mask != r.Mask {
					return errors.New("compact parity or exported mask differs")
				}
				for k, value := range w.Features {
					if value != r.Features[k] {
						return errors.New("source features differ from exported vector")
					}
				}
				for k := range p.Logits {
					for _, d := range []float64{float64(p.Logits[k]) - float64(r.Logits[k]), float64(p.Probabilities[k]) - float64(r.Probabilities[k])} {
						if !finite(d) {
							return errors.New("nonfinite parity value")
						}
						o.MaxError = math.Max(o.MaxError, math.Abs(d))
					}
				}
			}
			if o.MaxError > 1e-5 {
				return errors.New("export numerical tolerance exceeded")
			}
			calibration := map[float64]float64{.5: 0, 1: 0, 2: 0, 4: 0}
			calibrationCalls := 0
			for _, v := range views {
				if v.Split != "calibration" {
					continue
				}
				if err = compact.PredictInto(v.Text, &cw, &cp); err != nil {
					return err
				}
				calibrationCalls++
				for temperature := range calibration {
					calibration[temperature] += fullStableNLL(cp.Logits, v.Target.Passed, temperature) / 512
				}
			}
			if calibrationCalls != 512 || math.Abs(calibration[pin.Temperature]-pin.NLL) > 1e-5 {
				return errors.New("replayed export calibration differs")
			}
			if variant == "ptq_ternary" {
				best := .5
				for _, t := range []float64{1, 2, 4} {
					if calibration[t] < calibration[best] {
						best = t
					}
				}
				if best != pin.Temperature {
					return errors.New("replayed PTQ temperature choice differs")
				}
			}
			for _, form := range []string{"original", "task-prefix", "complete-suffix"} {
				changed := append([]threecohort.View(nil), views...)
				texts := map[string]string{}
				for i, v := range changed {
					if v.Split != "development" {
						continue
					}
					input, e := fullinputstudy.Apply(v.Text, v.Language, form)
					if e != nil || input.Declined {
						return errors.New("full development form cannot be represented; preserve prefix")
					}
					changed[i].Text, texts[v.ID] = input.Text, input.Text
				}
				dev, err := evaluateDevelopment(compact, changed)
				if err != nil {
					return err
				}
				var pairs = map[string][]developmentRow{}
				family := map[string]string{}
				for _, v := range changed {
					family[v.ID] = v.Family
				}
				dev.Counts.NLL = 0
				for _, counts := range dev.Families {
					counts.NLL = 0
				}
				for _, counts := range dev.Languages {
					counts.NLL = 0
				}
				for _, row := range dev.Rows {
					pairs[row.Group] = append(pairs[row.Group], row)
					nll := fullStableNLL(row.Prediction.Logits, row.Passed, pin.Temperature)
					o.StableNLL[form] += nll
					dev.Counts.NLL += nll
					dev.Families[family[row.View]].NLL += nll
					dev.Languages[row.Language].NLL += nll
				}
				var validity [3]int
				for _, pair := range pairs {
					a, b := pair[0], pair[1]
					av, bv := a.Passed[a.Prediction.Mask] == 16, b.Passed[b.Prediction.Mask] == 16
					if av && bv {
						validity[0]++
						if a.Prediction.Mask != b.Prediction.Mask {
							validity[1]++
						}
					} else if !av && !bv && a.Prediction.Mask == b.Prediction.Mask {
						validity[2]++
					}
				}
				o.PairValidity[form] = validity
				if err = fullDevelopmentJournal(filepath.Join(output, arm+"--"+variant+"--"+form+".jsonl.gz"), dev.Rows, texts); err != nil {
					return err
				}
				dev.Rows = nil
				o.Conditions[form] = dev
			}
			observed[arm+"/"+variant] = o
		}
	}
	return fullWriteJSON(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/full-input-initial-go-audit/v1", "status": "PASS", "auditor_source_revision": revision, "training_source_revision": fullTrainingSource, "protocol_sha256": fullProtocolSHA, "optimizer_updates_reconciled": 6400, "model_exports": 12, "actual_expanded_parity_predictions": 624, "actual_compact_parity_predictions": 624, "actual_compact_calibration_predictions": 6144, "actual_compact_development_predictions": 18432, "models": observed, "new_optimizer_updates": 0, "native_executions": 0, "scope": "Initial audit: all optimizer/epoch journals and calibration selectors, source-bound export parity, tied compact conversion, original development wording and two previously frozen evaluation-only forms. Complete all-split/form study, allocation/timing probes and actual native adoption remain pending."})
}

func fullStableNLL(logits [8]float32, passed [8]int, temperature float64) float64 {
	all, valid := math.Inf(-1), math.Inf(-1)
	for i, v := range logits {
		all = math.Max(all, float64(v)/temperature)
		if passed[i] == 16 {
			valid = math.Max(valid, float64(v)/temperature)
		}
	}
	allSum, validSum := 0., 0.
	for i, v := range logits {
		allSum += math.Exp(float64(v)/temperature - all)
		if passed[i] == 16 {
			validSum += math.Exp(float64(v)/temperature - valid)
		}
	}
	return all + math.Log(allSum) - valid - math.Log(validSum)
}

func fullWriteJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil || len(raw) > 4<<20 {
		return errors.New("bounded audit metadata required")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	return errors.Join(err, f.Close())
}

func fullDevelopmentJournal(path string, rows []developmentRow, texts map[string]string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	z := gzip.NewWriter(&fullBoundedWriter{writer: f, remaining: 1 << 20})
	encoder := json.NewEncoder(z)
	for _, row := range rows {
		if err = encoder.Encode(struct {
			developmentRow
			Text string `json:"complete_input"`
		}{row, texts[row.View]}); err != nil {
			break
		}
	}
	return errors.Join(err, z.Close(), f.Close())
}

type fullBoundedWriter struct {
	writer    io.Writer
	remaining int
}

func (w *fullBoundedWriter) Write(raw []byte) (int, error) {
	if len(raw) > w.remaining {
		return 0, errors.New("bounded full-input journal exhausted")
	}
	n, err := w.writer.Write(raw)
	w.remaining -= n
	return n, err
}

// Reserve more than every possible bounded journal/model/report byte before
// starting this one audit. Output files are fresh and old prefixes are retained.
func fullAuditStorage(output string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(output)
	if err != nil || filepath.Dir(abs) != filepath.Join(root, "runs") || !strings.HasPrefix(filepath.Base(abs), "own-three-full-input-") {
		return errors.New("bounded full-input audit phase required")
	}
	entries, err := os.ReadDir(filepath.Join(root, "runs"))
	if err != nil {
		return err
	}
	var all, study int64
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "own-three-") {
			continue
		}
		var size int64
		err = filepath.Walk(filepath.Join(root, "runs", entry.Name()), func(path string, info os.FileInfo, e error) error {
			if e != nil {
				return e
			}
			if info.IsDir() {
				return nil
			}
			if !info.Mode().IsRegular() {
				return errors.New("regular retained evidence required")
			}
			size += info.Size()
			return nil
		})
		if err != nil {
			return err
		}
		all += size
		if strings.HasPrefix(entry.Name(), "own-three-full-input-") {
			study += size
		}
	}
	var disk syscall.Statfs_t
	if err = syscall.Statfs(root, &disk); err != nil {
		return err
	}
	if disk.Bavail*uint64(disk.Bsize) < 4<<30 || study+(48<<20) > (768<<20)-(16<<20) || all+(48<<20) > (3<<30)-(16<<20) {
		return errors.New("full-input audit storage reserve exhausted")
	}
	return nil
}
