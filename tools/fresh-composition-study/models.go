package main

import (
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

type modelPin struct {
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
	Packed   int    `json:"packed_weights_bytes"`
}

func loadModels(root string) (map[string]*decision.Model, map[string]modelPin, error) {
	var report struct {
		Status  string                         `json:"status"`
		Steps   int                            `json:"optimizer_steps"`
		Exports map[string]map[string]modelPin `json:"exports"`
	}
	if err := decodeFile(filepath.Join(root, "report.json"), &report); err != nil {
		return nil, nil, err
	}
	if report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 960 || len(report.Exports) != 2 {
		return nil, nil, errors.New("six frozen exports required")
	}
	loaded, pins := map[string]*decision.Model{}, map[string]modelPin{}
	for _, arm := range arms {
		feature := decision.SplitContextIntentFeatureVersion
		if arm == "v3" {
			feature = decision.SemanticContextIntentFeatureVersion
		}
		for _, variant := range variants {
			model, err := decision.LoadPath(filepath.Join(root, arm, "models", variant, "model.json"))
			if err != nil {
				return nil, nil, err
			}
			pin := report.Exports[arm][variant]
			if model.MetadataSHA256() != pin.Metadata || model.WeightsSHA256() != pin.Weights || model.FeatureVersion() != feature {
				return nil, nil, errors.New("model export pin differs")
			}
			id := arm + "-" + variant
			loaded[id], pins[id] = model, pin
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

func measureModel(model *decision.Model, text string) (runtimeObservation, error) {
	var workspace decision.Workspace
	var prediction decision.Prediction
	if err := model.PredictInto(text, &workspace, &prediction); err != nil {
		return runtimeObservation{}, err
	}
	start := time.Now()
	for range 1000 {
		if err := model.PredictInto(text, &workspace, &prediction); err != nil {
			return runtimeObservation{}, err
		}
	}
	result := runtimeObservation{WarmPredictions: 1001, NSPerPrediction: float64(time.Since(start).Nanoseconds()) / 1000,
		Resident: model.ResidentTensorBytes(), Scales: model.MatrixScaleBytes(), Workspace: unsafe.Sizeof(workspace)}
	var probeError error
	result.HeapAllocs = testing.AllocsPerRun(1000, func() { probeError = model.PredictInto(text, &workspace, &prediction) })
	result.AllocationProbePredictions = 1001
	return result, probeError
}

func parity(root, arm string, loaded map[string]*decision.Model, views []view) (float64, error) {
	var document struct {
		Rows []struct {
			Variant       string       `json:"variant"`
			ID            string       `json:"row_id"`
			Text          string       `json:"text"`
			Features      [256]float32 `json:"features"`
			Logits        [8]float64   `json:"logits"`
			Probabilities [8]float64   `json:"probabilities"`
			Label         string       `json:"selected_label"`
		} `json:"rows"`
	}
	if err := decodeFile(filepath.Join(root, arm, "go-parity.json"), &document); err != nil {
		return 0, err
	}
	if len(document.Rows) != 96 {
		return 0, errors.New("96 held-out parity rows required")
	}
	bound := map[string]string{}
	for _, v := range views {
		for _, r := range v.Rows {
			if r.Split == "development" {
				bound[r.ID] = r.Input.Text
			}
		}
	}
	counts, maxError := map[string]int{}, 0.0
	for _, r := range document.Rows {
		model := loaded[arm+"-"+r.Variant]
		if model == nil || bound[r.ID] != r.Text {
			return 0, errors.New("held-out parity input/variant differs")
		}
		counts[r.Variant]++
		var workspace decision.Workspace
		var prediction decision.Prediction
		var features [256]float32
		if err := model.FeaturesInto(r.Text, &features); err != nil {
			return 0, err
		}
		if err := model.PredictInto(r.Text, &workspace, &prediction); err != nil {
			return 0, err
		}
		if model.PredictLabel(&prediction) != r.Label {
			return 0, errors.New("parity label differs")
		}
		for i, v := range features {
			maxError = math.Max(maxError, math.Abs(float64(v-r.Features[i])))
		}
		for i, v := range r.Logits {
			maxError = math.Max(maxError, math.Abs(float64(prediction.Logits[i])-v))
			maxError = math.Max(maxError, math.Abs(float64(prediction.Probabilities[i])-r.Probabilities[i]))
		}
	}
	for _, v := range variants {
		if counts[v] != 32 {
			return 0, errors.New("parity variant count differs")
		}
	}
	if math.IsNaN(maxError) || math.IsInf(maxError, 0) || maxError > 1e-6 {
		return maxError, errors.New("Go/Python parity tolerance exceeded")
	}
	return maxError, nil
}
