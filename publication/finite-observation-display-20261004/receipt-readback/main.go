// Read saved runtime accuracy states without model or native execution.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type object = map[string]any

func main() {
	if len(os.Args) != 2 {
		panic("usage receipt-readback saved-controls-directory")
	}
	var rows []object
	for _, name := range []string{"missing-tool", "fifo-tool", "observed-zero-deterministic", "observed-zero-model"} {
		b, err := os.ReadFile(filepath.Join(os.Args[1], name, "run-1-runtime.json"))
		if err != nil {
			panic(err)
		}
		var value object
		if err := json.Unmarshal(b, &value); err != nil {
			panic(err)
		}
		observation := value["observation"].(object)
		receipt := value["completeness_receipt"].(object)
		observed := len(observation["cases"].([]any))
		status, profile, count, replay := "UNKNOWN", "gooo/typed-path-runtime-v1", 0, false
		decision := "FAIL_CLOSED"
		if name == "observed-zero-deterministic" || name == "observed-zero-model" {
			status, profile, count, replay = "PROGRESS", "gooo/typed-path-runtime-v3", 128, true
			decision = "PROGRESS_WITHIN_DECLARED_RUNTIME_SCOPE"
		}
		aggregate, present := receipt["aggregate_completeness_score"]
		if observed != count || observation["runtime_replayed"] != replay || receipt["profile_id"] != profile ||
			!present || aggregate != nil || receipt["decision"] != decision {
			panic("scope/state changed")
		}
		found := 0
		for _, value := range receipt["dimensions"].([]any) {
			dimension := value.(object)
			if dimension["id"] != "runtime_finite_accuracy" {
				continue
			}
			found++
			if dimension["status"] != status || dimension["numerator"] != float64(0) || dimension["denominator"] != float64(128) ||
				dimension["unit"] != "ordered finite caller expectations matched" {
				panic("finite state changed")
			}
			rows = append(rows, object{"condition": name, "profile": profile, "finite_status": status, "unit": dimension["unit"],
				"numerator": 0, "denominator": 128, "observed_cases": observed, "runtime_replayed": replay,
				"original_decision": receipt["decision"], "aggregate_score": nil})
		}
		if found != 1 {
			panic("axis count")
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(object{"status": "PASS", "scope": "four saved runtime receipts only; original profile/status/unit retained",
		"model_predictions": 0, "native_executions": 0, "rows": rows}); err != nil {
		panic(err)
	}
}
