package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func modelCard() string {
	return `---
license: apache-2.0
language: [en, ko]
tags: [gooo, metaprogramming, provenance, small-model, ternary, experimental]
---
# Own Gooo joint feedback structural models v2

Nine own random-initialized 12,412-parameter models judge four legal Gooo body
paths from complete source-bound paired inputs and observed TDD failures.
They are bounded structural judgment models; they do not generate arbitrary
Korean/English text or guarantee general unseen program correctness.

Three matched arms compare uniform-initial loss, passing-set loss, and the
passing-set objective with actual own-teacher failure continuations. No Laya
or other pretrained weights initialize them. All nine FP32/PTQ/QAT exports
and negative results are retained. Calibration selected the older independent
reference; this publication does not implicitly change the compiler default.

Actual evidence: 3,600 offline MPS optimizer updates, 6,144 own-teacher SDK
sessions, 9,216 student/control SDK sessions, and 240 adopted-main Gooo
generations independently compiled/executed against 3,840 ordered Go cases.
See results.md for finite denominators, regressions and generalization limits.
The first rejected native audit attempt is also preserved in the raw archive.

Runtime, orchestration, tests, audits and publication packaging are Go 1.27.1.
Python is used only for offline MPS optimization/export. Each valid Go kernel
call allocates zero heap objects with a caller-owned 2,160-byte workspace.
FP32 weights occupy 49,648 bytes. Five-trit packed weights occupy 2,590 bytes;
runtime tensors use 12,384 int8 matrix bytes and 112 FP32 bias bytes, plus
8 matrix-scale bytes. This is 1.6 stored bits
per trit weight, separate from total process RAM and theoretical log2(3).

Load an arm's models/fp32/model.json with the public gooo-decision-runtime
v0.2.12-experimental jointdecision API. The compiler requires an explicit
--path-model file and uses the actual source/intent context automatically.
Disconnected operation remains deterministic. Models rank declared paths;
type checks and independently written finite tests decide executable acceptance.

Source and reproducible Go runners:
https://github.com/kimjooyoon/gooo-neural-decision-experiments
Compiler:
https://github.com/kimjooyoon/meta-ontology-go

raw-evidence.zip contains the frozen original curriculum, real teacher and
student captures, actual Go execution records, both reference models, and
runtime/optimizer source. publication-manifest.json binds every payload and
archive member by SHA-256 and byte count. Go packaging scans fixed synthetic
inputs for credentials and private host paths. PROV-O files separate training,
calibration, teacher observations, own initialization and native execution.
`
}

func writeProvenance(out, revision string) ([]string, error) {
	var pre struct {
		Initial string `json:"initial_state_sha256"`
		Source  string `json:"source_revision"`
		States  string `json:"student_states_sha256"`
	}
	if err := read(models+"/preexecution.json", &pre); err != nil {
		return nil, err
	}
	if len(pre.Initial) != 64 || len(pre.Source) != 40 || len(pre.States) != 64 {
		return nil, fmt.Errorf("own initialization and training source pins required")
	}
	names := []string{}
	if err := os.MkdirAll(filepath.Join(out, "provenance"), 0755); err != nil {
		return nil, err
	}
	for _, arm := range arms {
		fp, err := hashFile(models + "/" + arm + "/models/fp32/weights.bin")
		if err != nil {
			return nil, err
		}
		for _, variant := range variants {
			w, err := hashFile(models + "/" + arm + "/models/" + variant + "/weights.bin")
			if err != nil {
				return nil, err
			}
			m, err := hashFile(models + "/" + arm + "/models/" + variant + "/model.json")
			if err != nil {
				return nil, err
			}
			used, updates := pre.Initial, 600
			if variant != "fp32" {
				used = fp.SHA
			}
			if variant == "ptq_ternary" {
				updates = 0
			}
			role := "initial inputs only"
			if arm == "set-feedback" {
				role = "initial inputs plus independently verified actual train-only failure continuations"
			}
			text := fmt.Sprintf("@prefix prov: <http://www.w3.org/ns/prov#> .\n@prefix gooo: <urn:gooo:property:> .\n@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .\n\n<urn:sha256:%s> a prov:Entity ; gooo:role \"own random initialization or this arm's own FP32 checkpoint; no inherited pretrained weights\" .\n<urn:gooo:feedback-v2:%s:%s> a prov:Activity ; gooo:optimizerUpdates \"%d\"^^xsd:integer ; gooo:trainingDataRole \"%s\" ; prov:used <urn:sha256:%s>, <https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/%s> .\n<urn:sha256:%s> a prov:Entity ; prov:wasGeneratedBy <urn:gooo:feedback-v2:%s:%s> ; prov:wasDerivedFrom <urn:sha256:%s> .\n<urn:sha256:%s> a prov:Entity ; gooo:role \"strict Go joint model metadata\" ; prov:wasDerivedFrom <urn:sha256:%s> .\n<https://github.com/kimjooyoon/meta-ontology-go/commit/363a3d8aa365c35dd634c241248b444de0050973> a prov:Entity ; gooo:role \"actual adopted-main source-bound execution compiler\" .\n<https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/%s> a prov:Entity ; gooo:role \"publication and audit source; development does not select checkpoints or deployment\" .\n", used, arm, variant, updates, role, used, pre.Source, w.SHA, arm, variant, used, m.SHA, w.SHA, revision)
			p := "provenance/" + arm + "-" + variant + ".ttl"
			if err = os.WriteFile(filepath.Join(out, p), []byte(text), 0644); err != nil {
				return nil, err
			}
			names = append(names, p)
		}
	}
	lineage := fmt.Sprintf("@prefix prov: <http://www.w3.org/ns/prov#> .\n@prefix gooo: <urn:gooo:property:> .\n<urn:sha256:2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383> a prov:Entity ; gooo:role \"frozen bilingual original source curriculum\" .\n<urn:gooo:feedback-v2:observed-training-failures> a prov:Activity ; prov:used <urn:sha256:e1576dd7e7f206e60d1523eff3bb0af2b33c969c7021a502133d980de4985cf6> ; gooo:role \"old own teacher ranks training paths only; no student weights inherited\" .\n<urn:sha256:%s> a prov:Entity ; prov:wasGeneratedBy <urn:gooo:feedback-v2:observed-training-failures> ; prov:wasDerivedFrom <urn:sha256:2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383> .\n", pre.States)
	if err := os.WriteFile(filepath.Join(out, "provenance/experiment.ttl"), []byte(lineage), 0644); err != nil {
		return nil, err
	}
	names = append(names, "provenance/experiment.ttl")
	return names, nil
}
