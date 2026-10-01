package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

// Stage publishes the verified SDK phase before native adoption completes.
// Its manifest is explicitly incomplete and cannot pass the full verifier.
func stageSDK(models, study, output, revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean stage source required")
	}
	var audit struct {
		Status  string `json:"status"`
		Dataset string `json:"dataset_sha256"`
		Views   int    `json:"frozen_sdk_function_observations"`
		Calls   int    `json:"actual_model_predictions_all_stages"`
	}
	raw, err := os.ReadFile(filepath.Join(study, "independent-audit.json"))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &audit); err != nil || audit.Status != "PASS" || audit.Dataset != datasetSHA || audit.Views != 5376 || audit.Calls != 25471 {
		return errors.New("actual SDK audit required")
	}
	if _, err = os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh stage required")
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	files := map[string]string{"LICENSE": "LICENSE", "protocol.md": "docs/joint-path-composition-preregistration-20261002.md", "prefixture-amendment.md": "docs/joint-path-composition-prefixture-amendment-20261002.md", "results.md": "docs/joint-composition-model-results-20261002.md", "training-report.json": filepath.Join(models, "report.json"), "training-preexecution.json": filepath.Join(models, "preexecution.json"), "study-report.json": filepath.Join(study, "report.json"), "study-preexecution.json": filepath.Join(study, "preexecution.json"), "calibration-selection.json": filepath.Join(study, "selection.json"), "independent-audit.json": filepath.Join(study, "independent-audit.json"), "curriculum-manifest.json": "publication/joint-composition-curriculum-manifest-20261002.json", "curriculum-audit.json": "publication/joint-composition-curriculum-audit-20261002.json", "sdk-source-provenance.json": filepath.Join(sdkRoot, "source-provenance.json")}
	for _, arm := range []string{"independent", "joint"} {
		files[arm+"/go-parity.json"] = filepath.Join(models, arm, "go-parity.json")
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			name := arm + "/models/" + variant + "/model.json"
			if arm == "joint" {
				if _, e := jointdecision.Load(filepath.Join(models, name)); e != nil {
					return e
				}
			} else {
				if _, e := decision.LoadPath(filepath.Join(models, name)); e != nil {
					return e
				}
			}
			files[name] = filepath.Join(models, name)
			name = arm + "/models/" + variant + "/weights.bin"
			files[name] = filepath.Join(models, name)
		}
	}
	for name, source := range files {
		if _, err = hashFile(source); err != nil {
			return err
		}
		if strings.HasSuffix(name, "weights.bin") {
			target := filepath.Join(output, name)
			if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			raw, err = os.ReadFile(source)
			if err != nil {
				return err
			}
			if err = os.WriteFile(target, raw, 0644); err != nil {
				return err
			}
		} else if err = copyFile(source, filepath.Join(output, name)); err != nil {
			return err
		}
	}
	card := `---
license: mit
language: [en, ko]
tags: [gooo, metaprogramming, ternary, from-scratch, go]
---
# Own Gooo joint path tiny v1 — intermediate SDK edition

Six own random-initialized models, no Laya/pretrained/earlier own-model weights.
Independent architecture 256/48/8 has 12,728 parameters; joint 512/24/4 has
12,412. The models rank compiler-owned legal paths from typed Gooo source and
complete Korean/English intent. They are bounded judgments, not text generators.

All 480 actual optimizer updates, six FP32/PTQ/QAT exports, parity and negative
comparisons are retained. Calibration selected independent FP32. Development
joint FP32 makes 777 predictions versus 1,447, with 455 versus 448 extra assembly
attempts. All seven policies, including offline, finish 384 finite function
contracts by four candidates. This does not establish universal correctness.
Joint ternary weights occupy 2,590 bytes on disk and decode to int8 for execution.

This revision publishes the verified Go SDK phase: 5,376 captured sessions,
25,471 actual predictions across evaluation/parity/probes, and an independent
Go audit adding no model calls. No native-main execution is claimed here.
SDK v0.2.12 is released and native feature PR #1137 has merged to dev. Main
promotion PR #1138 and the separate actual native execution stage are pending.
No default model is promoted. Model-disconnected continuation is deterministic.

Read results.md, both frozen protocol documents and the intermediate manifest.
Native captures, PROV-O chains and the complete raw archive will follow adoption.
Only deliberate public synthetic artifacts are included; all text is scanned
for credentials and host paths. See GitHub for runtime and trainer source:
https://github.com/kimjooyoon/gooo-neural-decision-experiments
https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.12-experimental
`
	if err = os.WriteFile(filepath.Join(output, "README.md"), []byte(card), 0644); err != nil {
		return err
	}
	files["README.md"] = ""
	var entries []artifact
	for name := range files {
		entry, e := hashFile(filepath.Join(output, name))
		if e != nil {
			return e
		}
		entry.Path = name
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return save(filepath.Join(output, "publication-manifest.json"), map[string]any{"schema": "gooo/own-joint-composition-intermediate-publication/v1", "status": "SDK_VERIFIED_NATIVE_PENDING", "repository": repository, "source_revision": revision, "files": entries, "dataset_sha256": datasetSHA, "model_exports": 6, "optimizer_updates": 480, "actual_sdk_predictions_all_stages": 25471, "frozen_sdk_function_observations": 5376, "native_main_execution_claimed": false, "default_model_promoted": false, "credentials_and_host_paths_scanned": true})
}
