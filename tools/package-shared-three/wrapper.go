package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
)

const wrapperSchema = "gooo/bilingual-wrapper-public-evidence/v1"

// Public source contains the privacy filter itself and one synthetic negative
// test. Exempt those exact literals only; actual evidence has no exemptions.
func publicationPrivacy(name string, raw []byte, schema string) {
	checked := raw
	if schema == wrapperSchema {
		if name == "source/auditor/io.go" {
			checked = bytes.ReplaceAll(checked, []byte("`/Users/|/home/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_`"), []byte("`PUBLIC_PRIVACY_PATTERN`"))
		}
		if name == "source/auditor/main_test.go" {
			checked = bytes.ReplaceAll(checked, []byte("\"/Users/example/private\""), []byte("\"SYNTHETIC_PRIVATE_PATH_TEST\""))
		}
	}
	privacy(name, checked)
}

func wrapperFiles(raw, dataset, dense, shared string) map[string]string {
	m := map[string]string{"README.md": "docs/bilingual-wrapper-audit-results-20261003.md", "LICENSE": "LICENSE", "protocol.md": "docs/bilingual-wrapper-audit-protocol-20261003.md", "dataset.jsonl": dataset, "process-metrics.json": "publication/bilingual-wrapper-process-metrics-20261003.json"}
	for _, name := range []string{"preexecution.json", "inputs.jsonl", "predictions.jsonl", "feature-conflicts.json", "report.json", "manifest.json"} {
		m["raw/"+name] = filepath.Join(raw, name)
	}
	for _, arm := range []string{"dense", "shared"} {
		root := dense
		if arm == "shared" {
			root = shared
		}
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, name := range []string{"model.json", "weights.bin"} {
				m["models/"+arm+"/"+variant+"/"+name] = filepath.Join(root, variant, name)
			}
		}
	}
	for _, name := range []string{"main.go", "counts.go", "inputs.go", "io.go", "main_test.go"} {
		m["source/auditor/"+name] = filepath.Join("tools/audit-bilingual-wrappers", name)
	}
	for _, name := range []string{"main.go", "main_test.go"} {
		m["source/verifier/"+name] = filepath.Join("tools/verify-wrapper-audit", name)
	}
	return m
}

func packWrappers(raw, dataset, dense, shared, out string) {
	var report struct {
		Schema string `json:"schema"`
		Status string `json:"status"`
		Calls  int    `json:"actual_model_predictions"`
		Inputs int    `json:"input_forms"`
	}
	must(json.Unmarshal(read(filepath.Join(raw, "report.json")), &report))
	if report.Schema != "gooo/bilingual-wrapper-audit/v1" || report.Status != "COMPLETE" || report.Calls != 92160 || report.Inputs != 15360 {
		panic("complete closed wrapper audit required")
	}
	packInventory(wrapperFiles(raw, dataset, dense, shared), out, wrapperSchema, "All five authored input forms, six frozen models, complete source dataset, 92,160 recorded predictions, source and independent reader. Existing weights; zero optimizer updates and zero native executions. Known-cohort input sensitivity with complete inputs and all conditions retained.", 96<<20)
}
