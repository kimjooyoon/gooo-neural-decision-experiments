package main

import (
	"encoding/json"
	"testing"
)

func TestGenerationMetricsUseCompilerJSONFields(t *testing.T) {
	raw := []byte(`{"gooo_source":"body","source":"go","report":{"body_paths":{
    "selected_source_sha256":"source-hash","document_sha256":"plan-hash",
    "search":{"selection":{"local_model_predictions":1,
    "three_choice_prediction":{"input_sha256":"input-hash","predict_ns":20300}}},
    "native_case_results":[{"passed":true},{"passed":false}]}}}`)
	var g generation
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	p := g.Report.BodyPaths
	if g.GoooSource != "body" || g.Source != "go" || p.SelectedSHA != "source-hash" ||
		p.DocumentSHA != "plan-hash" || p.Search.Selection.Calls != 1 ||
		p.Search.Selection.Prediction == nil || p.Search.Selection.Prediction.NS != 20300 ||
		p.Search.Selection.Prediction.SHA != "input-hash" || len(p.NativeCases) != 2 ||
		!p.NativeCases[0].Passed || p.NativeCases[1].Passed {
		t.Fatal("compiler metrics silently decoded to defaults", g)
	}
}

func TestMacOSResourceLinesKeepTheirUnits(t *testing.T) {
	text := "        0.03 real         0.01 user         0.02 sys\n 21430272  maximum resident set size\n"
	clock, rss := clockLine.FindStringSubmatch(text), rssLine.FindStringSubmatch(text)
	if len(clock) != 4 || clock[2] != "0.01" || clock[3] != "0.02" ||
		len(rss) != 2 || rss[1] != "21430272" {
		t.Fatal("resource units lost", clock, rss)
	}
}
