package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func provenance(name string) error {
	if _, err := os.Lstat(name); !os.IsNotExist(err) {
		return errors.New("fresh PROV-O artifact required")
	}
	raw, err := os.ReadFile("runs/" + trained + "/report.json")
	if err != nil {
		return err
	}
	var report struct {
		Status  string `json:"status"`
		Steps   int    `json:"optimizer_steps"`
		Exports map[string]map[string]struct {
			Metadata string `json:"metadata_sha256"`
			Weights  string `json:"weights_sha256"`
		} `json:"exports"`
	}
	if err = json.Unmarshal(raw, &report); err != nil || report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 4800 || len(report.Exports) != 3 {
		return errors.New("actual complete training report required")
	}
	graph := []any{
		map[string]any{"@id": "#go", "@type": "prov:SoftwareAgent", "gooo:runtime": "Go 1.27.1"},
		map[string]any{"@id": "#mps-optimizer", "@type": "prov:SoftwareAgent", "gooo:runtime": "Offline local Torch MPS; optimization/export only"},
		map[string]any{"@id": "#native-source", "@type": "prov:Entity", "gooo:sha256": threecohort.DatasetSHA},
		map[string]any{"@id": "#actual-teacher-states", "@type": "prov:Entity", "gooo:sha256": threestudent.StatesSHA, "gooo:actual_sessions": 4096, "gooo:actual_teacher_predictions": 37394, "gooo:initialization_use": false},
		map[string]any{"@id": "#fresh-initializer", "@type": "prov:Entity", "gooo:sha256": threestudent.InitialSHA, "gooo:seed": threestudent.InitialSeed, "gooo:inherited_model_weights": false},
		map[string]any{"@id": "#feature-preparation", "@type": "prov:Activity", "prov:used": []string{"#native-source", "#actual-teacher-states"}, "prov:wasAssociatedWith": "#go", "gooo:source_revision": "e0b256a9e561260672c5956834a9da10a39d11a0", "gooo:new_predictions": 0},
		map[string]any{"@id": "#prepared-features", "@type": "prov:Entity", "prov:wasGeneratedBy": "#feature-preparation", "gooo:sha256": "4143e4f57936427faeee5b46b91374511efdf20dec5c4c15f70fd9af6d893167"},
		map[string]any{"@id": "#recorded-training", "@type": "prov:Entity", "gooo:sha256": sha(raw), "gooo:actual_updates": 4800},
		map[string]any{"@id": "#go-kernel-audit", "@type": "prov:Activity", "prov:wasAssociatedWith": "#go", "gooo:source_revision": "08e15cfa03d82a02a35081fc0fe504d508a55690", "gooo:actual_parity_predictions": 432, "gooo:actual_warm_allocation_predictions": 18018, "gooo:new_optimizer_updates": 0},
	}
	for _, arm := range []string{"uniform-initial", "set-initial", "set-feedback"} {
		graph = append(graph, map[string]any{"@id": "#" + arm + "-fp", "@type": "prov:Activity", "prov:used": []string{"#prepared-features", "#fresh-initializer"}, "prov:wasAssociatedWith": "#mps-optimizer", "gooo:actual_updates": 800, "gooo:calibration_only_selection": true})
		graph = append(graph, map[string]any{"@id": "#" + arm + "-qat", "@type": "prov:Activity", "prov:used": []string{"#prepared-features", "#" + arm + "-fp32"}, "prov:wasAssociatedWith": "#mps-optimizer", "gooo:actual_updates": 800, "gooo:calibration_only_selection": true})
		graph = append(graph, map[string]any{"@id": "#" + arm + "-ptq", "@type": "prov:Activity", "prov:used": "#" + arm + "-fp32", "gooo:actual_updates": 0, "gooo:calibration_only_selection": true})
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			p, ok := report.Exports[arm][variant]
			if !ok {
				return errors.New("missing retained model variant")
			}
			activity := arm + "-fp"
			if variant == "ptq_ternary" {
				activity = arm + "-ptq"
			}
			if variant == "qat_ternary" {
				activity = arm + "-qat"
			}
			graph = append(graph, map[string]any{"@id": "#" + arm + "-" + variant, "@type": "prov:Entity", "prov:wasGeneratedBy": "#" + activity, "gooo:metadata_sha256": p.Metadata, "gooo:weights_sha256": p.Weights, "gooo:retained_if_negative": true})
		}
	}
	value := map[string]any{"@context": map[string]any{"prov": "http://www.w3.org/ns/prov#", "gooo": "https://github.com/kimjooyoon/gooo-neural-decision-experiments/terms/", "prov:used": map[string]string{"@type": "@id"}, "prov:wasAssociatedWith": map[string]string{"@type": "@id"}, "prov:wasGeneratedBy": map[string]string{"@type": "@id"}}, "@graph": graph, "gooo:scope": "Separate immutable source, actual own-teacher observation, new independent initialization, balanced feature preparation, matched MPS optimization, all nine negative/positive exports and actual Go inference. No new full SDK/native study or default promotion is claimed."}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(encoded, '\n'))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
