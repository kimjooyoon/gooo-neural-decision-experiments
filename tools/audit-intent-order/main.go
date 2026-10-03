package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

type task struct {
	ID, Language string
	Clauses      [2]string
	Bodies       [2]string
	Expected     [2][3]int64
}

type view struct {
	Intent       string                        `json:"intent"`
	GoooBody     string                        `json:"reference_gooo_body"`
	Expected     [3]int64                      `json:"reference_outputs"`
	InputSHA     string                        `json:"input_sha256"`
	Order        decision.IntentOrderSketch    `json:"experimental_order_sketch"`
	Prediction   jointdecision.ThreePrediction `json:"frozen_bag_model_prediction"`
	PredictionNS int64                         `json:"prediction_ns"`
}

type record struct {
	ID, Language, Wrapper                                  string
	Views                                                  [2]view
	BagEqual, PositionedEqual, OrderEqual, PredictionEqual bool
}

func main() {
	output := flag.String("output", "", "new publication directory")
	revision := flag.String("source-revision", "", "collector source revision")
	modelPath := flag.String("model", "publication/full-input-separate-arithmetic-20261003/models/compact/bag-original/fp32/model.json", "frozen model")
	flag.Parse()
	if *output == "" || len(*revision) != 40 {
		panic("new output and exact source revision required")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		panic("output must not exist")
	}
	model, err := jointdecision.LoadSharedThree(*modelPath)
	must(err)
	if model.FeatureVersion() != jointdecision.ThreeBagFeatureVersion {
		panic("bag control required")
	}
	var records []record
	separated, aliases, identicalPredictions := 0, 0, 0
	for _, t := range tasks() {
		for _, wrapper := range []string{"bare", "prefix", "suffix", "both"} {
			r := collect(t, wrapper, model)
			records = append(records, r)
			if r.BagEqual {
				aliases++
			}
			if !r.OrderEqual {
				separated++
			}
			if r.PredictionEqual {
				identicalPredictions++
			}
		}
	}
	must(os.MkdirAll(*output, 0755))
	write(filepath.Join(*output, "records.json"), records)
	write(filepath.Join(*output, "report.json"), map[string]any{
		"schema": "gooo/intent-order-feature-preflight/v1", "collector_source": *revision,
		"feature_version": decision.IntentOrderSketchVersion, "model_metadata_sha256": model.MetadataSHA256(),
		"model_weights_sha256": model.WeightsSHA256(), "authored_operator_families": 4,
		"language_views": 2, "wrappers": 4, "permutation_pairs": len(records),
		"bag_identical_pairs": aliases, "order_sketch_separated_pairs": separated,
		"model_identical_prediction_pairs": identicalPredictions, "actual_model_predictions": len(records) * 2,
		"training_updates": 0, "new_feature_model_predictions": 0, "native_executions": 0,
		"sketch_storage_bytes": 128, "reference_inputs": [3]int64{-2, 0, 3},
		"remaining_collision": remainingCollision(),
		"scope":               "Feature distinguishability on four authored arithmetic order pairs, Korean/English views and four wrappers. Fixed synthetic valid semantic headers, not native source-derived context. Frozen model predictions expose V4 aliasing; new sketch has no trained head and no accuracy score. References are authored arithmetic observations, not compiled Gooo execution.",
	})
	fmt.Printf("pairs=%d bag_aliases=%d sketch_separated=%d actual_predictions=%d\n", len(records), aliases, separated, len(records)*2)
}

func collect(t task, wrapper string, model *jointdecision.ThreeModel) record {
	r := record{ID: t.ID, Language: t.Language, Wrapper: wrapper}
	var bag, positioned [2][decision.FeatureDim]float32
	for i := range 2 {
		prefix, end, intro, suffix := "Step: ", "Step: end.", "Request: ", " Please finish."
		if t.Language == "ko" {
			prefix, end, intro, suffix = "단계: ", "단계: 끝.", "요청: ", " 완료해 주세요."
		}
		intent := prefix + t.Clauses[i] + ". " + prefix + t.Clauses[1-i] + ". " + end
		if wrapper == "prefix" || wrapper == "both" {
			intent = intro + intent
		}
		if wrapper == "suffix" || wrapper == "both" {
			intent += suffix
		}
		var fields [decision.SplitContextDim]byte
		fields[4], fields[5] = 128, 128
		text, err := decision.EncodeSemanticContextInput(fields, intent)
		must(err)
		v := view{Intent: intent, GoooBody: t.Bodies[i], Expected: t.Expected[i], InputSHA: hash([]byte(text))}
		must(decision.SemanticContextBagFeaturesInto(text, &bag[i]))
		must(decision.SemanticContextFeaturesInto(text, &positioned[i]))
		must(decision.SemanticIntentOrderSketchInto(text, &v.Order))
		joint, err := jointdecision.EncodeThree([3]string{text, text, text})
		must(err)
		var work jointdecision.ThreeWorkspace
		start := time.Now()
		must(model.PredictInto(joint, &work, &v.Prediction))
		v.PredictionNS = time.Since(start).Nanoseconds()
		r.Views[i] = v
	}
	r.BagEqual, r.PositionedEqual, r.OrderEqual = bag[0] == bag[1], positioned[0] == positioned[1], r.Views[0].Order == r.Views[1].Order
	r.PredictionEqual = reflect.DeepEqual(r.Views[0].Prediction, r.Views[1].Prediction)
	return r
}

func tasks() []task {
	definitions := []struct {
		id           string
		en, ko, body [2]string
		eval         func(int64) [2]int64
	}{
		{"add-multiply", [2]string{"add one", "multiply by two"}, [2]string{"1을 더한다", "2를 곱한다"}, [2]string{"return ((input + 1) * 2)", "return ((input * 2) + 1)"}, func(x int64) [2]int64 { return [2]int64{(x + 1) * 2, x*2 + 1} }},
		{"subtract-multiply", [2]string{"subtract three", "multiply by two"}, [2]string{"3을 뺀다", "2를 곱한다"}, [2]string{"return ((input - 3) * 2)", "return ((input * 2) - 3)"}, func(x int64) [2]int64 { return [2]int64{(x - 3) * 2, x*2 - 3} }},
		{"negate-add", [2]string{"negate the value", "add four"}, [2]string{"부호를 반대로 바꾼다", "4를 더한다"}, [2]string{"return ((0 - input) + 4)", "return (0 - (input + 4))"}, func(x int64) [2]int64 { return [2]int64{-x + 4, -(x + 4)} }},
		{"square-add", [2]string{"square the value", "add one"}, [2]string{"값을 제곱한다", "1을 더한다"}, [2]string{"return ((input * input) + 1)", "return ((input + 1) * (input + 1))"}, func(x int64) [2]int64 { return [2]int64{x*x + 1, (x + 1) * (x + 1)} }},
	}
	var out []task
	for _, d := range definitions {
		var expected [2][3]int64
		for i, x := range [3]int64{-2, 0, 3} {
			y := d.eval(x)
			expected[0][i], expected[1][i] = y[0], y[1]
		}
		for _, language := range []string{"en", "ko"} {
			clauses := d.en
			if language == "ko" {
				clauses = d.ko
			}
			out = append(out, task{d.id, language, clauses, d.body, expected})
		}
	}
	return out
}

func remainingCollision() map[string]any {
	texts := [2]string{"A. B. A. C. A. D.", "A. C. A. B. A. D."}
	var fields [decision.SplitContextDim]byte
	fields[4], fields[5] = 128, 128
	var sketches [2]decision.IntentOrderSketch
	for i, intent := range texts {
		text, err := decision.EncodeSemanticContextInput(fields, intent)
		must(err)
		must(decision.SemanticIntentOrderSketchInto(text, &sketches[i]))
	}
	return map[string]any{"intents": texts, "sketches_equal": sketches[0] == sketches[1], "reason": "Directed edge multisets can lose the order of repeated excursions; finite hashes add other possible collisions."}
}

func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func write(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	must(err)
	must(os.WriteFile(path, append(b, '\n'), 0644))
}
