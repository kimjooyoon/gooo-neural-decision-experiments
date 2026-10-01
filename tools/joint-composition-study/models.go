package main

import (
	"errors"
	"fmt"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"math"
	"path/filepath"
	"testing"
	"time"
	"unsafe"
)

type modelPin struct {
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
	Packed   int    `json:"packed_weights_bytes"`
}
type loadedModel struct {
	kind        string
	independent *decision.Model
	joint       *jointdecision.Model
}

func (m *loadedModel) MetadataSHA256() string {
	if m.joint != nil {
		return m.joint.MetadataSHA256()
	}
	return m.independent.MetadataSHA256()
}
func (m *loadedModel) WeightsSHA256() string {
	if m.joint != nil {
		return m.joint.WeightsSHA256()
	}
	return m.independent.WeightsSHA256()
}
func (m *loadedModel) RuntimeFeatureVersion() string {
	if m.joint != nil {
		return m.joint.FeatureVersion()
	}
	return m.independent.FeatureVersion()
}

func (m *loadedModel) FeatureVersion() string { return decision.SemanticContextIntentFeatureVersion }
func loadModels(root string) (map[string]*loadedModel, map[string]modelPin, error) {
	var report struct {
		Status  string                         `json:"status"`
		Steps   int                            `json:"optimizer_steps"`
		Exports map[string]map[string]modelPin `json:"exports"`
	}
	if err := decodeFile(filepath.Join(root, "report.json"), &report); err != nil {
		return nil, nil, err
	}
	if report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 480 || len(report.Exports) != 2 {
		return nil, nil, errors.New("six fresh frozen exports required")
	}
	loaded, pins := map[string]*loadedModel{}, map[string]modelPin{}
	for _, arm := range arms {
		for _, variant := range variants {
			name := filepath.Join(root, arm, "models", variant, "model.json")
			m := &loadedModel{kind: arm}
			var metadata, weights string
			var size int
			var err error
			if arm == "independent" {
				m.independent, err = decision.LoadPath(name)
				if err != nil {
					return nil, nil, err
				}
				if m.independent.FeatureVersion() != decision.SemanticContextIntentFeatureVersion {
					return nil, nil, errors.New("independent feature differs")
				}
				metadata, weights, size = m.independent.MetadataSHA256(), m.independent.WeightsSHA256(), m.independent.PackedFileBytes()
			} else {
				m.joint, err = jointdecision.Load(name)
				if err != nil {
					return nil, nil, err
				}
				metadata, weights, size = m.joint.MetadataSHA256(), m.joint.WeightsSHA256(), m.joint.PackedFileBytes()
			}
			pin := report.Exports[arm][variant]
			if metadata != pin.Metadata || weights != pin.Weights || size != pin.Packed {
				return nil, nil, errors.New("actual export pin differs")
			}
			id := arm + "-" + variant
			loaded[id], pins[id] = m, pin
		}
	}
	return loaded, pins, nil
}

type runtimeObservation struct {
	WarmPredictions            int     `json:"warm_benchmark_predictions"`
	NSPerPrediction            float64 `json:"warm_ns_per_prediction"`
	HeapAllocs                 float64 `json:"predict_into_heap_allocations_per_call"`
	AllocationProbePredictions int     `json:"allocation_probe_predictions"`
	Resident                   int     `json:"resident_tensor_bytes"`
	Scales                     int     `json:"matrix_scale_bytes"`
	Workspace                  uintptr `json:"workspace_bytes"`
}

func measureModel(m *loadedModel, text string) (runtimeObservation, error) {
	var iw decision.Workspace
	var ip decision.Prediction
	var jw jointdecision.Workspace
	var jp jointdecision.Prediction
	predict := func() error {
		if m.joint != nil {
			return m.joint.PredictInto(text, &jw, &jp)
		}
		return m.independent.PredictInto(text, &iw, &ip)
	}
	if err := predict(); err != nil {
		return runtimeObservation{}, err
	}
	start := time.Now()
	for range 1000 {
		if err := predict(); err != nil {
			return runtimeObservation{}, err
		}
	}
	r := runtimeObservation{WarmPredictions: 1001, NSPerPrediction: float64(time.Since(start).Nanoseconds()) / 1000}
	if m.joint != nil {
		r.Resident, r.Scales, r.Workspace = m.joint.ResidentTensorBytes(), m.joint.MatrixScaleBytes(), unsafe.Sizeof(jw)
	} else {
		r.Resident, r.Scales, r.Workspace = m.independent.ResidentTensorBytes(), m.independent.MatrixScaleBytes(), unsafe.Sizeof(iw)
	}
	var failure error
	r.HeapAllocs = testing.AllocsPerRun(1000, func() { failure = predict() })
	r.AllocationProbePredictions = 1001
	return r, failure
}
func parity(root, arm string, loaded map[string]*loadedModel, views []view) (float64, error) {
	var document struct {
		Rows []struct {
			Variant       string    `json:"variant"`
			ID            string    `json:"row_id"`
			Text          string    `json:"text"`
			Features      []float32 `json:"features"`
			Logits        []float64 `json:"logits"`
			Probabilities []float64 `json:"probabilities"`
			Label         string    `json:"selected_label"`
		} `json:"rows"`
	}
	if err := decodeFile(filepath.Join(root, arm, "go-parity.json"), &document); err != nil {
		return 0, err
	}
	if len(document.Rows) != 96 {
		return 0, errors.New("96 parity rows per architecture required")
	}
	bound := map[string]string{}
	for _, v := range views {
		for _, r := range v.Rows {
			if r.Split == "development" {
				bound[r.ID] = r.Input.Text
				if arm == "joint" {
					bound[r.ID] = r.JointText
				}
			}
		}
	}
	counts := map[string]int{}
	maximum := 0.0
	for _, r := range document.Rows {
		m := loaded[arm+"-"+r.Variant]
		if m == nil || bound[r.ID] != r.Text {
			return maximum, errors.New("heldout source input/variant differs")
		}
		counts[r.Variant]++
		var features, logits, probabilities []float32
		var label string
		if m.joint != nil {
			var w jointdecision.Workspace
			var p jointdecision.Prediction
			if err := m.joint.PredictInto(r.Text, &w, &p); err != nil {
				return maximum, err
			}
			features, logits, probabilities = w.Features[:], p.Logits[:], p.Probabilities[:]
			label = fmt.Sprintf("mask_%d", p.Mask)
		} else {
			var w decision.Workspace
			var p decision.Prediction
			if err := m.independent.PredictInto(r.Text, &w, &p); err != nil {
				return maximum, err
			}
			var f [decision.FeatureDim]float32
			if err := m.independent.FeaturesInto(r.Text, &f); err != nil {
				return maximum, err
			}
			features, logits, probabilities = f[:], p.Logits[:], p.Probabilities[:]
			label = m.independent.PredictLabel(&p)
		}
		if len(features) != len(r.Features) || len(logits) != len(r.Logits) || len(probabilities) != len(r.Probabilities) || label != r.Label {
			return maximum, errors.New("parity dimensions/selection differ")
		}
		for i, v := range features {
			maximum = math.Max(maximum, math.Abs(float64(v-r.Features[i])))
		}
		for i, v := range logits {
			maximum = math.Max(maximum, math.Abs(float64(v)-r.Logits[i]))
			maximum = math.Max(maximum, math.Abs(float64(probabilities[i])-r.Probabilities[i]))
		}
	}
	for _, variant := range variants {
		if counts[variant] != 32 {
			return maximum, errors.New("32 parity judgments per variant required")
		}
	}
	if math.IsNaN(maximum) || math.IsInf(maximum, 0) || maximum > 1e-6 {
		return maximum, errors.New("Go Python numerical parity exceeds tolerance")
	}
	return maximum, nil
}
