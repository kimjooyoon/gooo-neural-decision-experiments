// Read the public shared model through the Go SDK and report field choices.
package main

import (
	"encoding/json"
	"flag"
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
	asText := flag.Bool("text", false, "show field names, expressions and ordering hints")
	flag.Parse()
	if flag.NArg() != 2 {
		panic("model.json and source-only context export required")
	}
	model, err := jointdecision.LoadRecordSharedThree(flag.Arg(0))
	must(err)
	raw, err := os.ReadFile(flag.Arg(1))
	must(err)
	var input struct {
		Source      string         `json:"original_source_sha256"`
		Predictions int            `json:"model_predictions"`
		Tests       int            `json:"candidate_tests"`
		Choices     []sourceChoice `json:"choices"`
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
	named, err := describeChoices(input.Choices, fields, p.Mask)
	must(err)
	if *asText {
		fmt.Printf("Source %s; proposed mask %d; model calls: 1\n", input.Source, p.Mask)
		for _, choice := range named {
			fmt.Printf("%s [%s]\n  intent: %s\n  first  %.2f%%: %s\n  second %.2f%%: %s\n  proposed: %s\n", choice.Field, choice.ID, choice.Intent, 100*choice.Probabilities[0], choice.First, 100*choice.Probabilities[1], choice.Second, choice.Proposed)
		}
		fmt.Println("These probabilities order attempts. Run Gooo body-compose to measure finite functional completeness.")
		return
	}
	output := map[string]any{"schema": "gooo/public-shared-field-read/v1", "source_sha256": input.Source, "metadata_sha256": model.MetadataSHA256(), "weights_sha256": model.WeightsSHA256(), "weight_file_bytes": model.PackedFileBytes(), "resident_tensor_bytes": model.ResidentTensorBytes(), "scale_bytes": model.MatrixScaleBytes(), "prediction": p, "field_probabilities": fields, "field_choices": named, "first_text_prediction_ns": elapsed, "model_calls": 1, "scope": "First-call source parsing and ranking; finite completeness is measured separately by Gooo body-compose."}
	result, err := json.MarshalIndent(output, "", "  ")
	must(err)
	fmt.Println(string(result))
}

type sourceChoice struct {
	ID     string `json:"id"`
	Field  string `json:"field"`
	Intent string `json:"intent"`
	First  string `json:"first"`
	Second string `json:"second"`
}
type namedChoice struct {
	sourceChoice
	Probabilities [2]float64 `json:"probabilities"`
	Proposed      string     `json:"proposed"`
}

func describeChoices(choices []sourceChoice, probabilities [3][2]float64, mask uint16) ([3]namedChoice, error) {
	var result [3]namedChoice
	if len(choices) != len(result) || mask >= 8 {
		return result, fmt.Errorf("three source choices and a three-bit mask required")
	}
	for i, choice := range choices {
		proposed := choice.First
		if mask&(1<<i) != 0 {
			proposed = choice.Second
		}
		result[i] = namedChoice{choice, probabilities[i], proposed}
	}
	return result, nil
}
