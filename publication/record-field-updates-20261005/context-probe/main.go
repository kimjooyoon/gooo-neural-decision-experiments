package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
	"os"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func main() {
	model, e := jointdecision.LoadRecordSharedThree(os.Args[1])
	must(e)
	var rows []any
	for _, pair := range [][2]string{{"copy.state", "saved.state"}, {"copy.title + copy.state", "saved.title + saved.state"}, {"copy.title", "saved"}} {
		var contexts [2]string
		var features [2][jointdecision.ThreeFeatureDim]float32
		var predictions [2]jointdecision.ThreePrediction
		for i, expr := range pair {
			choices := [3]jointdecision.RecordChoice{{Field: "title", First: "\"draft\"", Second: "input0.title", Intent: "Keep the original title."}, {Field: "state", First: "\"wait\"", Second: "\"ready\"", Intent: "Set state to ready."}, {Field: "reason", First: "\"deferred\"", Second: expr, Intent: "Use the declared expression with current and saved values."}}
			contexts[i], e = jointdecision.EncodeRecordThree(choices)
			must(e)
			must(jointdecision.FeaturesIntoRecordThree(contexts[i], &features[i]))
			var w jointdecision.ThreeWorkspace
			must(model.PredictRecordSharedFeaturesInto(&features[i], &w, &predictions[i]))
		}
		rows = append(rows, map[string]any{"expressions": pair, "context_sha256": [2]string{fmt.Sprintf("%x", sha256.Sum256([]byte(contexts[0]))), fmt.Sprintf("%x", sha256.Sum256([]byte(contexts[1])))}, "features_identical": features[0] == features[1], "predictions_identical": predictions[0] == predictions[1], "predictions": predictions})
	}
	b, e := json.MarshalIndent(map[string]any{"schema": "gooo/local-binding-feature-probe/v1", "pairs": rows, "model_updates": 0, "scope": "same intents/other choices; current features omit selector receiver provenance; text identity retained"}, "", "  ")
	must(e)
	must(os.WriteFile(os.Args[2], append(b, '\n'), 0644))
	fmt.Println(string(b))
}
