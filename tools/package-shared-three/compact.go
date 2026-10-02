package main

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

const compactNativeSchema = "gooo/compact-shared-native-public-evidence/v1"

func standaloneFile(name, schema string) bool {
	if schema == fullNativeSchema {
		return name == "README.md" || name == "LICENSE" || name == "native/report.json" || name == "native/preexecution.json" || name == "native/independent-consumption.json"
	}
	if schema == wrapperSchema {
		return name == "README.md" || name == "LICENSE" || name == "raw/report.json" || name == "process-metrics.json"
	}
	return strings.HasPrefix(name, "models/") || name == "README.md" || name == "LICENSE" ||
		name == "training/go-audit.json" || name == "native/independent-consumption.json" ||
		(schema == compactNativeSchema && (name == "native/paired-report.json" || name == "native/preexecution.json"))
}

func compactFiles(native string) map[string]string {
	m := map[string]string{
		"README.md": "docs/compact-shared-native-results-20261003.md", "LICENSE": "LICENSE",
		"protocol.json":     "preexecution/shared-three-compact-runtime-preregistration-20261003.json",
		"kernel-audit.json": "runs/compact-shared-runtime-20261003/report.json",
	}
	for _, name := range []string{"preexecution.json", "report.json", "paired-report.json", "independent-consumption.json", "independent-consumer.go"} {
		m["native/"+name] = filepath.Join(native, name)
	}
	for _, name := range []string{"main.go", "compact.go", "shared.go"} {
		m["source/native-runner/"+name] = filepath.Join("cmd/native-runtime-observe", name)
	}
	for _, family := range threecompositionstudy.Families {
		for _, language := range []string{"en", "ko"} {
			for _, format := range []string{"expanded", "compact"} {
				for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
					name := family + "-" + language + "-" + format + "-" + variant
					for _, file := range []string{"input.gooo", "plan.json", "generation.json", "generated.go", "runtime-cases.json", "runtime.json", "observation.json"} {
						m["native/"+name+"/"+file] = filepath.Join(native, name, file)
					}
				}
			}
		}
	}
	return m
}

func packCompact(native, out string) {
	var report struct {
		Status   string `json:"status"`
		Source   string `json:"compiler_source_sha"`
		Receipts int    `json:"receipts"`
		Progress int    `json:"verified_session_progress_records"`
		Feedback int    `json:"verified_feedback_records"`
	}
	must(json.Unmarshal(read(filepath.Join(native, "independent-consumption.json")), &report))
	if report.Status != "PASS" || report.Source != "7db19b6bc9a2909aa059f1265d39a539c3573a57" || report.Receipts != 96 || report.Progress != 596 || report.Feedback != 202 {
		panic("independently consumed complete compact native evidence required")
	}
	packInventory(compactFiles(native), out, compactNativeSchema,
		"All 96 native generations and 192 compiled runs, independent receipt/chain consumption, all 48 representation pairs and exact collector source. Embedded parent JSON is decoded for privacy inspection. Finite observed equivalence, not a new holdout or accuracy improvement. No duplicate training dataset or model weights.", 32<<20)
}
