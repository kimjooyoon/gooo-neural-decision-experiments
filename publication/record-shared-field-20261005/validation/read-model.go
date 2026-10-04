// Read the public shared model through the Go SDK and report field choices.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	if len(os.Args) != 3 {
		panic("model.json and source-only context export required")
	}
	model, err := jointdecision.LoadRecordSharedThree(os.Args[1])
	must(err)
	raw, err := os.ReadFile(os.Args[2])
	must(err)
	var input struct {
		Source      string `json:"original_source_sha256"`
		Predictions int    `json:"model_predictions"`
		Tests       int    `json:"candidate_tests"`
		Context     struct {
			Text    string `json:"text"`
			Status  string `json:"status"`
			Version string `json:"feature_version"`
		} `json:"context"`
	}
	must(json.Unmarshal(raw, &input))
	if input.Predictions != 0 || input.Tests != 0 || input.Context.Status != "ENCODED" || input.Context.Version != jointdecision.RecordSharedFeatureVersion {
		panic("complete source-only shared context required")
	}
	var w jointdecision.ThreeWorkspace
	var p jointdecision.ThreePrediction
	started := time.Now()
	must(model.PredictRecordSharedInto(input.Context.Text, &w, &p))
	elapsed := time.Since(started).Nanoseconds()
	fields, err := jointdecision.RecordChoiceMarginals(p)
	must(err)
	output := map[string]any{"schema": "gooo/public-shared-field-read/v1", "source_sha256": input.Source, "metadata_sha256": model.MetadataSHA256(), "weights_sha256": model.WeightsSHA256(), "weight_file_bytes": model.PackedFileBytes(), "resident_tensor_bytes": model.ResidentTensorBytes(), "scale_bytes": model.MatrixScaleBytes(), "prediction": p, "field_probabilities": fields, "first_text_prediction_ns": elapsed, "model_calls": 1, "scope": "First-call source parsing and ranking; finite completeness is measured separately by Gooo body-compose."}
	result, err := json.MarshalIndent(output, "", "  ")
	must(err)
	fmt.Println(string(result))
}
