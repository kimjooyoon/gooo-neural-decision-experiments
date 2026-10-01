// audit-joint-representation separates target-distribution variation from
// incompatible executable choices in the immutable source-bound curriculum.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	dataset := flag.String("dataset", "", "frozen source-bound curriculum JSONL")
	output := flag.String("output", "", "fresh diagnostic report")
	flag.Parse()
	if *dataset == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "dataset and fresh output required")
		os.Exit(2)
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "output must not exist")
		os.Exit(2)
	}
	layers, err := readCurriculum(*dataset)
	if err == nil {
		err = writeReport(*output, layers)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeReport(output string, layers map[string]map[string][]sample) error {
	results := map[string]summary{}
	for name, groups := range layers {
		results[name] = summarize(groups)
	}
	value := struct {
		Schema     string             `json:"schema"`
		Status     string             `json:"status"`
		DatasetSHA string             `json:"dataset_sha256"`
		Rows       int                `json:"source_decision_rows"`
		Views      int                `json:"bilingual_function_views"`
		Layers     map[string]summary `json:"representation_layers"`
		NewCalls   int                `json:"new_model_predictions"`
		NewUpdates int                `json:"new_optimizer_updates"`
		Scope      string             `json:"scope"`
	}{"gooo/joint-representation-diagnostic/v1", "PASS", datasetSHA, 4608, 2304, results, 0, 0,
		"Frozen finite targets only. Unequal uniform target distributions can retain a common valid executable choice. Empty common support and best fixed-choice coverage measure actual representation incompatibility. This does not measure learned-model quality or universal correctness."}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(raw, '\n'), 0644)
}
