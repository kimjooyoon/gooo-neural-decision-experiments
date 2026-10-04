package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReadRowsUsesObservedJSONNames(t *testing.T) {
	private, public := t.TempDir(), t.TempDir()
	response := []byte(`{"status":"completed","response":{"report":{"generated_digest":"fixture-sha","body_paths":{
"declared_test_cases":3,"finite_functional_completeness_percent":100,"search":{"evaluated_candidates":2,
"attempts":[{"training_cases_passed":1,"training_cases_total":3}],"selection":{"local_model_predictions":1,
"model_metadata_sha256":"metadata","model_weights_sha256":"weights","three_choice_prediction":{"input_sha256":"input","predict_ns":22000}}}}}},
"execution":{"observation":{"runtime_replayed":true,"declared_cases":1,"cases":[{"passed":true}],
"runs":[{"started":true},{"started":true}],"artifact":{"reused":true}}}}`)
	timing := []byte(`{"response_ns":4000000,"wall":{"phases":[{"name":"generation","start_ns":100,"end_ns":2000100}]}}`)
	for i := 1; i <= 4; i++ {
		for name, data := range map[string][]byte{"response": response, "timing": timing, "generation": []byte(`{"fixture":"public"}`)} {
			path := filepath.Join(private, fmt.Sprintf("run-%d-%s.json", i, name))
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	rows, err := readRows(private, "fixture", "source", "model", public)
	if err != nil || len(rows) != 4 {
		t.Fatal(err, rows)
	}
	for _, row := range rows {
		if row.ModelCalls != 1 || row.GenerationMS != 2 || row.PredictNS != 22000 || row.ModelInputSHA != "input" ||
			row.Evaluated != 2 || row.FirstPassed != 1 || row.FirstTotal != 3 || row.RuntimePassed != 1 {
			t.Fatal("metric extraction lost observed fields", row)
		}
	}
}

func TestCommandResourceScopeAndMissingFields(t *testing.T) {
	row, err := parseResource("0.42 real 0.23 user 0.11 sys\n86491136 maximum resident set size\n15942184 peak memory footprint\n")
	if err != nil || row.MaxRSS != 86491136 || row.Footprint != 15942184 || row.CPUPercent < 80 || row.CPUPercent > 82 {
		t.Fatal(err, row)
	}
	if _, err := parseResource("0.42 real 0.23 user 0.11 sys"); err == nil {
		t.Fatal("absent memory was treated as zero")
	}
}
