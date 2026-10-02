package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

var arms = [3]string{"uniform-initial", "set-initial", "set-feedback"}
var variants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}

type pin struct {
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
	Packed   int    `json:"packed_weights_bytes"`
}
type model struct {
	Three       *jointdecision.ThreeModel
	Independent *decision.Model
	Pin         pin
	SetupNS     int64
	Variant     string
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
func loadModels(root string) (map[string]*model, []string, error) {
	var report struct {
		Status  string                    `json:"status"`
		Steps   int                       `json:"optimizer_steps"`
		Exports map[string]map[string]pin `json:"exports"`
	}
	if err := read(filepath.Join(root, "report.json"), &report); err != nil {
		return nil, nil, err
	}
	if report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 4800 || len(report.Exports) != 3 {
		return nil, nil, errors.New("all nine frozen 4800-update own three-choice students required")
	}
	all := map[string]*model{"offline": nil}
	for _, arm := range arms {
		if len(report.Exports[arm]) != 3 {
			return nil, nil, errors.New("three retained variants per arm required")
		}
		for _, variant := range variants {
			start := time.Now()
			m, err := jointdecision.LoadThree(filepath.Join(root, arm, "models", variant, "model.json"))
			if err != nil {
				return nil, nil, err
			}
			p := report.Exports[arm][variant]
			if m.MetadataSHA256() != p.Metadata || m.WeightsSHA256() != p.Weights || m.PackedFileBytes() != p.Packed || m.Variant() != variant {
				return nil, nil, errors.New("frozen student export pin differs")
			}
			all[arm+"/"+variant] = &model{Three: m, Pin: p, SetupNS: time.Since(start).Nanoseconds(), Variant: variant}
		}
	}
	start := time.Now()
	m, err := decision.LoadPath("models/joint-composition-v1/independent/models/fp32/model.json")
	if err != nil {
		return nil, nil, err
	}
	if m.MetadataSHA256() != threefeedback.TeacherMetadata || m.WeightsSHA256() != threefeedback.TeacherWeights {
		return nil, nil, errors.New("frozen independent reference differs")
	}
	all["reference-v1-independent"] = &model{Independent: m, Pin: pin{m.MetadataSHA256(), m.WeightsSHA256(), m.PackedFileBytes()}, SetupNS: time.Since(start).Nanoseconds(), Variant: "fp32"}
	ids := make([]string, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	if len(ids) != 11 {
		return nil, nil, errors.New("exact eleven policies required")
	}
	return all, ids, nil
}
