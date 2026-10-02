package main

import (
	"encoding/json"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

const fullNativeSchema = "gooo/full-input-native-public-evidence/v1"

func fullNativeFiles(native string) map[string]string {
	m := map[string]string{
		"README.md": "docs/full-input-native-results-20261003.md", "LICENSE": "LICENSE",
		"protocol.md":                    "docs/full-input-sdk-native-protocol-20261003.md",
		"source/independent-consumer.go": "tools/consume-full-input-native/main.go.txt",
	}
	for _, name := range []string{"preexecution.json", "report.json", "progress.json", "independent-consumption.json"} {
		m["native/"+name] = filepath.Join(native, name)
	}
	for _, name := range []string{"comparison.go", "full_input.go", "full_input_bounds.go", "full_input_test.go", "main.go", "runtime_test.go"} {
		m["source/native-runner/"+name] = filepath.Join("cmd/full-input-native-observe", name)
	}
	modes := []string{"offline"}
	for _, arm := range []string{"positioned-original", "positioned-varied", "bag-original", "bag-varied"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, layout := range []string{"expanded", "compact"} {
				modes = append(modes, arm+"-"+variant+"-"+layout)
			}
		}
	}
	for _, family := range threecompositionstudy.Families {
		for _, language := range []string{"en", "ko"} {
			for _, mode := range modes {
				name := family + "-" + language + "-" + mode
				for _, file := range []string{"input.gooo", "plan.json", "generation.json", "generation-process.json", "generated.go", "runtime-cases.json", "runtime.json", "runtime-process.json", "observation.json"} {
					m["native/"+name+"/"+file] = filepath.Join(native, name, file)
				}
			}
		}
	}
	return m
}

func packFullNative(native, out string) {
	var r struct {
		Status    string `json:"status"`
		Source    string `json:"compiler_source_sha"`
		Collector string `json:"collector_source_sha"`
		Consumer  string `json:"consumer_sha256"`
		Receipts  int    `json:"receipts"`
		Runs      int    `json:"compiled_runs"`
		Cases     int    `json:"finite_expectations"`
	}
	must(json.Unmarshal(read(filepath.Join(native, "independent-consumption.json")), &r))
	if r.Status != "PASS" || r.Source != "e461c1defddd4bb35500c7f5dfd67d2c23836156" || r.Collector != "07afc044810dca126cf8890dd086efe4adb56dd8" || r.Consumer != "sha256:e0df1e4fe16b1ab903eb1051b32b89e660c75204048953b17ffd70d2c7ab8b42" || r.Receipts != 400 || r.Runs != 800 || r.Cases != 9600 {
		panic("source-bound complete native consumption required")
	}
	packInventory(fullNativeFiles(native), out, fullNativeSchema,
		"All 400 native generations, 800 compiled runs and 192 representation pairs over 16 known bilingual tasks; original source, plan, process costs, progress/feedback and runtime bytes. Independent consumption checks 9600 finite expectations. Text, decoded JSON and embedded parent/diagnostic bytes are privacy-scanned. Frozen model artifacts remain in the separately pinned arithmetic bundle.", 128<<20)
}
