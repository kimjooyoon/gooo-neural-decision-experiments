// Package bodydecision selects only predeclared, typed operations in a body
// plan. The model supplies operation labels; bodyplan owns scope and emission.
package bodydecision

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

type HoleReceipt struct {
	ID            string                      `json:"id"`
	TextSHA256    string                      `json:"text_sha256"`
	Mode          string                      `json:"mode"`
	Proposed      string                      `json:"proposed,omitempty"`
	Selected      string                      `json:"selected"`
	Confidence    float32                     `json:"global_confidence"`
	Probabilities []decision.LabelProbability `json:"raw_probabilities,omitempty"`
	PredictNS     int64                       `json:"predict_ns"`
}

type Selection struct {
	Choices       map[string]string `json:"choices"`
	Holes         []HoleReceipt     `json:"holes"`
	ModelVariant  string            `json:"model_variant,omitempty"`
	WeightsSHA256 string            `json:"weights_sha256,omitempty"`
	SeedSHA256    string            `json:"seed_sha256,omitempty"`
	ModelCalls    int               `json:"model_prediction_calls"`
	ProviderCalls int               `json:"external_provider_calls"`
}

// Validate checks the complete candidate set before any inference. Every
// individual allowed replacement must preserve the declared typed structure.
func Validate(plan bodyplan.Plan) (map[string]string, error) {
	choices := make(map[string]string)
	for _, expr := range plan.Expressions {
		if expr.Kind != "hole" {
			continue
		}
		if _, exists := choices[expr.HoleID]; exists {
			return nil, fmt.Errorf("duplicate hole %q", expr.HoleID)
		}
		choices[expr.HoleID] = expr.Fallback
	}
	if _, err := bodyplan.Compile(plan, choices); err != nil {
		return nil, err
	}
	for _, expr := range plan.Expressions {
		if expr.Kind != "hole" {
			continue
		}
		for _, operation := range expr.Allowed {
			trial := CloneChoices(choices)
			trial[expr.HoleID] = operation
			if _, err := bodyplan.Compile(plan, trial); err != nil {
				return nil, fmt.Errorf("hole %s candidate %s: %w", expr.HoleID, operation, err)
			}
		}
	}
	return choices, nil
}

// Choose never reads gold labels or test cases. With no model it replays each
// declared fallback. Sampling is explicit and replayable from its seed/weights.
func Choose(plan bodyplan.Plan, model *decision.Model, seed string) (Selection, error) {
	choices, err := Validate(plan)
	if err != nil {
		return Selection{}, err
	}
	if seed != "" && model == nil {
		return Selection{}, errors.New("sampling requires an explicit loaded model")
	}
	result := Selection{Choices: choices, Holes: make([]HoleReceipt, 0, len(choices))}
	if model != nil {
		result.ModelVariant, result.WeightsSHA256 = model.Variant(), model.WeightsSHA256()
	}
	if seed != "" {
		digest := sha256.Sum256([]byte(seed))
		result.SeedSHA256 = hex.EncodeToString(digest[:])
	}
	var workspace decision.Workspace
	labels := decision.Labels()
	for _, expr := range plan.Expressions {
		if expr.Kind != "hole" {
			continue
		}
		digest := sha256.Sum256([]byte(expr.Text))
		receipt := HoleReceipt{ID: expr.HoleID, TextSHA256: hex.EncodeToString(digest[:]), Mode: "declared_fallback", Selected: expr.Fallback}
		if model != nil {
			var prediction decision.Prediction
			started := time.Now()
			err := model.PredictInto(expr.Text, &workspace, &prediction)
			receipt.PredictNS = time.Since(started).Nanoseconds()
			result.ModelCalls++
			if err != nil {
				receipt.Mode = "fallback_prediction_error"
			} else {
				receipt.Confidence = prediction.Confidence
				receipt.Proposed = model.PredictLabel(&prediction)
				for i, label := range labels {
					receipt.Probabilities = append(receipt.Probabilities, decision.LabelProbability{Label: label, Probability: prediction.Probabilities[i]})
				}
				switch {
				case prediction.Abstained:
					receipt.Mode = "fallback_low_global_confidence"
				case seed != "":
					selected, sampleErr := sample(plan.ID, expr.HoleID, seed, expr.Allowed, receipt.Probabilities)
					if sampleErr != nil {
						receipt.Mode = "fallback_invalid_distribution"
					} else {
						receipt.Selected, receipt.Mode = selected, "seeded_allowed_probability_sample"
					}
				case contains(expr.Allowed, receipt.Proposed):
					receipt.Selected, receipt.Mode = receipt.Proposed, "model_global_argmax"
				default:
					receipt.Mode = "fallback_global_label_outside_allowed"
				}
			}
		}
		result.Choices[expr.HoleID] = receipt.Selected
		result.Holes = append(result.Holes, receipt)
	}
	if _, err := bodyplan.Compile(plan, result.Choices); err != nil {
		return Selection{}, fmt.Errorf("selected body: %w", err)
	}
	return result, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func CloneChoices(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func sample(planID, holeID, seed string, allowed []string, probabilities []decision.LabelProbability) (string, error) {
	weights := make([]float64, len(allowed))
	var total float64
	for i, label := range allowed {
		for _, value := range probabilities {
			if value.Label == label {
				weights[i] = float64(value.Probability)
			}
		}
		if weights[i] < 0 || math.IsNaN(weights[i]) || math.IsInf(weights[i], 0) {
			return "", errors.New("invalid probability")
		}
		total += weights[i]
	}
	if total <= 0 || math.IsInf(total, 0) {
		return "", errors.New("empty probability mass")
	}
	bound := []byte(seed + "\x00" + planID + "\x00" + holeID + "\x00")
	for i, label := range allowed {
		bound = append(bound, label...)
		bound = append(bound, 0)
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], math.Float64bits(weights[i]/total))
		bound = append(bound, encoded[:]...)
	}
	digest := sha256.Sum256(bound)
	draw := float64(binary.BigEndian.Uint64(digest[:8])>>11) / (1 << 53)
	var cumulative float64
	for i, weight := range weights {
		cumulative += weight / total
		if draw < cumulative {
			return allowed[i], nil
		}
	}
	return allowed[len(allowed)-1], nil
}
