package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

var arms = [3]string{"uniform-initial", "set-initial", "set-feedback"}
var variants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}

type pin struct {
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
	Packed   int    `json:"packed_weights_bytes"`
}

type model struct {
	Joint       *jointdecision.Model
	Independent *decision.Model
	Pin         pin
}

func read(name string, value any) error {
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func save(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}

func loadModels(root string) (map[string]*model, []string, error) {
	var report struct {
		Status  string                    `json:"status"`
		Steps   int                       `json:"optimizer_steps"`
		Exports map[string]map[string]pin `json:"exports"`
	}
	if err := read(filepath.Join(root, "report.json"), &report); err != nil {
		return nil, nil, err
	}
	if report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 3600 || len(report.Exports) != 3 {
		return nil, nil, errors.New("nine frozen own students required")
	}
	all := map[string]*model{"offline": nil}
	for _, arm := range arms {
		if len(report.Exports[arm]) != 3 {
			return nil, nil, errors.New("three variants per arm required")
		}
		for _, variant := range variants {
			m, err := jointdecision.Load(filepath.Join(root, arm, "models", variant, "model.json"))
			if err != nil {
				return nil, nil, err
			}
			p := report.Exports[arm][variant]
			if m.MetadataSHA256() != p.Metadata || m.WeightsSHA256() != p.Weights || m.PackedFileBytes() != p.Packed || m.Variant() != variant {
				return nil, nil, errors.New("student export pin differs")
			}
			all[arm+"/"+variant] = &model{Joint: m, Pin: p}
		}
	}
	for _, arm := range []string{"joint", "independent"} {
		name := filepath.Join("models/joint-composition-v1", arm, "models/fp32/model.json")
		m := &model{}
		var err error
		if arm == "joint" {
			m.Joint, err = jointdecision.Load(name)
			if err == nil {
				m.Pin = pin{m.Joint.MetadataSHA256(), m.Joint.WeightsSHA256(), m.Joint.PackedFileBytes()}
			}
		} else {
			m.Independent, err = decision.LoadPath(name)
			if err == nil {
				m.Pin = pin{m.Independent.MetadataSHA256(), m.Independent.WeightsSHA256(), m.Independent.PackedFileBytes()}
			}
		}
		if err != nil {
			return nil, nil, err
		}
		var old struct {
			Exports map[string]map[string]pin `json:"exports"`
		}
		if err = read("models/joint-composition-v1/report.json", &old); err != nil {
			return nil, nil, err
		}
		if m.Pin != old.Exports[arm]["fp32"] {
			return nil, nil, errors.New("frozen v1 reference pin differs")
		}
		all["reference-v1-"+arm] = m
	}
	ids := make([]string, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return all, ids, nil
}
