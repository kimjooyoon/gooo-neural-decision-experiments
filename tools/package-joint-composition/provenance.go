package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func modelProvenance(models, native, output string) ([]string, error) {
	var pre struct {
		Source  string            `json:"source_revision"`
		Initial map[string]string `json:"initial_state_sha256"`
	}
	raw, err := os.ReadFile(filepath.Join(models, "preexecution.json"))
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &pre); err != nil {
		return nil, err
	}
	var compiler struct {
		Native string `json:"native_revision"`
		Status string `json:"status"`
	}
	raw, err = os.ReadFile(filepath.Join(native, "independent-audit.json"))
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &compiler); err != nil || compiler.Native != deployedCompiler || deployedCompiler == "" || compiler.Status != "PASS" {
		return nil, errors.New("adopted native compiler audit required")
	}
	if err = os.MkdirAll(filepath.Join(output, "provenance"), 0755); err != nil {
		return nil, err
	}
	var result []string
	for _, arm := range []string{"independent", "joint"} {
		fp, err := hashFile(filepath.Join(models, arm, "models/fp32/weights.bin"))
		if err != nil {
			return nil, err
		}
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			weights, e := hashFile(filepath.Join(models, arm, "models", variant, "weights.bin"))
			if e != nil {
				return nil, e
			}
			metadata, e := hashFile(filepath.Join(models, arm, "models", variant, "model.json"))
			if e != nil {
				return nil, e
			}
			steps := 120
			kind := "offline MPS optimizer and calibration checkpoint selection"
			used := pre.Initial[arm]
			if len(used) != 64 {
				return nil, errors.New("own architecture initial-state pin required")
			}
			if variant != "fp32" {
				used = fp.SHA
			}
			if variant == "ptq_ternary" {
				steps = 0
				kind = "post-training five-trit export from own FP32 checkpoint"
			}
			dataUses := fmt.Sprintf("<urn:gooo:dataset:%s:train>, <urn:gooo:dataset:%s:calibration>", datasetSHA, datasetSHA)
			if variant == "ptq_ternary" {
				dataUses = fmt.Sprintf("<urn:gooo:dataset:%s:calibration>", datasetSHA)
			}
			activity := fmt.Sprintf("urn:gooo:joint-composition:20261002:%s:%s", arm, variant)
			ttl := fmt.Sprintf(`@prefix prov: <http://www.w3.org/ns/prov#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix gooo: <urn:gooo:property:> .

<urn:gooo:dataset:%s> a prov:Entity ; gooo:sha256 "%s" .
<urn:gooo:dataset:%s:train> a prov:Entity ; prov:wasDerivedFrom <urn:gooo:dataset:%s> ; gooo:role "optimization split only" .
<urn:gooo:dataset:%s:calibration> a prov:Entity ; prov:wasDerivedFrom <urn:gooo:dataset:%s> ; gooo:role "checkpoint and temperature selection only" .
<urn:gooo:dataset:%s:development> a prov:Entity ; prov:wasDerivedFrom <urn:gooo:dataset:%s> ; gooo:role "subsequent evaluation; excluded from training and selector" .
<urn:sha256:%s> a prov:Entity ; gooo:role "own random initial state or own FP32 checkpoint; no inherited Laya weights" .
<%s> a prov:Activity ; gooo:kind "%s" ; gooo:optimizerUpdates "%d"^^xsd:integer ;
    prov:used <urn:sha256:%s>, %s ;
    prov:used <https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/%s> .
<urn:sha256:%s> a prov:Entity ; gooo:role "exported model weights" ; prov:wasGeneratedBy <%s> ; prov:wasDerivedFrom <urn:sha256:%s> .
<urn:sha256:%s> a prov:Entity ; gooo:role "bounded Go model metadata" ; prov:wasGeneratedBy <%s> ; prov:wasDerivedFrom <urn:sha256:%s> .
<https://github.com/kimjooyoon/meta-ontology-go/commit/f4813dc6251037767c8cff7295ccfdab2b044ff2> a prov:Entity ; gooo:role "source-bound canonical curriculum export compiler; no predictions" .
<https://github.com/kimjooyoon/meta-ontology-go/commit/%s> a prov:Entity ; gooo:role "independently verified adopted joint ABI native execution compiler" .
`, datasetSHA, datasetSHA, datasetSHA, datasetSHA, datasetSHA, datasetSHA, datasetSHA, datasetSHA, used, activity, kind, steps, used, dataUses, pre.Source, weights.SHA, activity, used, metadata.SHA, activity, weights.SHA, compiler.Native)
			name := "provenance/" + arm + "-" + variant + ".ttl"
			if err = os.WriteFile(filepath.Join(output, name), []byte(ttl), 0644); err != nil {
				return nil, err
			}
			result = append(result, name)
		}
	}
	return result, nil
}
