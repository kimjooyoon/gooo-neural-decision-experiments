package decision_test

import (
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/decision"
)

func TestPublicModelLoadsEachVariantAndPredicts(t *testing.T) {
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		t.Run(variant, func(t *testing.T) {
			modelPath := filepath.Join("..", "runs", "pilot-mps-20260930-v1", "models", variant, "model.json")
			model, err := decision.Load(modelPath)
			if err != nil {
				t.Fatal(err)
			}
			if model.Variant() != variant || model.WeightsSHA256() == "" || model.PackedFileBytes() <= 0 {
				t.Fatalf("loaded model provenance is incomplete: variant=%q digest=%q packed=%d", model.Variant(), model.WeightsSHA256(), model.PackedFileBytes())
			}
			wantResident, wantMatrix, wantBias, wantScale := 50_912, 50_688, 224, 0
			if variant != "fp32" {
				wantResident, wantMatrix, wantScale = 12_896, 12_672, 8
			}
			if model.ResidentTensorBytes() != wantResident || model.MatrixTensorBytes() != wantMatrix ||
				model.BiasTensorBytes() != wantBias || model.MatrixScaleBytes() != wantScale {
				t.Fatalf("loaded model tensor sizes = %d/%d/%d/%d, want %d/%d/%d/%d",
					model.ResidentTensorBytes(), model.MatrixTensorBytes(), model.BiasTensorBytes(), model.MatrixScaleBytes(),
					wantResident, wantMatrix, wantBias, wantScale)
			}

			var workspace decision.Workspace
			var prediction decision.Prediction
			if err := model.PredictInto("Add the quantity and fee.", &workspace, &prediction); err != nil {
				t.Fatal(err)
			}
			if !contains(decision.Labels(), model.PredictLabel(&prediction)) {
				t.Fatalf("prediction label %q is outside the public closed label set", model.PredictLabel(&prediction))
			}
			if model.ConfidenceThreshold() <= 0 || model.ConfidenceThreshold() > 1 {
				t.Fatalf("invalid confidence threshold %f", model.ConfidenceThreshold())
			}
		})
	}
	if decision.InputMaxBytes != 512 || decision.WorkspaceBytes() != 1248 ||
		decision.PredictionValueArrayBytes() != 64 || decision.PredictionBytes() < 64 {
		t.Fatalf("public fixed bounds changed unexpectedly: input=%d workspace=%d arrays=%d prediction=%d",
			decision.InputMaxBytes, decision.WorkspaceBytes(), decision.PredictionValueArrayBytes(), decision.PredictionBytes())
	}
}

func TestPublicLabelsAreCopied(t *testing.T) {
	first := decision.Labels()
	if len(first) != 8 || first[0] == "" {
		t.Fatalf("unexpected closed labels: %#v", first)
	}
	wantFirst := first[0]
	first[0] = "arbitrary"
	second := decision.Labels()
	if second[0] != wantFirst {
		t.Fatalf("mutating returned labels changed later result: first=%q", second[0])
	}
}

func TestPublicDecisionReturnsOnlyValidatedTypedIR(t *testing.T) {
	model, err := decision.Load(filepath.Join("..", "runs", "pilot-mps-20260930-v1", "models", "fp32", "model.json"))
	if err != nil {
		t.Fatal(err)
	}
	request := decision.DecisionRequest{
		Schema: decision.DecisionRequestSchema,
		Text:   "Add the quantity and fee.",
		Left:   decision.Identifier{Name: "quantity", Type: decision.TypeInt},
		Right:  decision.Identifier{Name: "fee", Type: decision.TypeInt},
	}
	var workspace decision.Workspace
	response, err := model.Decide(request, &workspace)
	if err != nil {
		t.Fatal(err)
	}
	if response.Schema != decision.DecisionResponseSchema || response.ModelVariant != "fp32" ||
		response.WeightsSHA256 != model.WeightsSHA256() || len(response.Probabilities) != len(decision.Labels()) {
		t.Fatalf("public response omitted model binding or closed probabilities: %#v", response)
	}
	labels := decision.Labels()
	for index, probability := range response.Probabilities {
		if probability.Label != labels[index] {
			t.Fatalf("response label %d = %q, want closed label %q", index, probability.Label, labels[index])
		}
	}
	switch response.Status {
	case "decision":
		if response.TypedBinaryIR == nil {
			t.Fatal("decision response omitted typed IR")
		}
	case "abstained":
		if response.TypedBinaryIR != nil || response.AbstainReason == "" {
			t.Fatalf("abstention carried IR or omitted its reason: %#v", response)
		}
	default:
		t.Fatalf("unexpected public decision status %q", response.Status)
	}
	if response.TypedBinaryIR != nil {
		if _, err := decision.BuildTypedBinary(response.TypedBinaryIR.Operation, response.TypedBinaryIR.Left, response.TypedBinaryIR.Right); err != nil {
			t.Fatalf("model returned invalid typed IR: %v", err)
		}
		if response.TypedBinaryIR.Left.Name != request.Left.Name || response.TypedBinaryIR.Right.Name != request.Right.Name {
			t.Fatalf("model changed declared operands: %#v", response.TypedBinaryIR)
		}
	}
	if _, err := decision.BuildTypedBinary("panic(1)", request.Left, request.Right); err == nil {
		t.Fatal("arbitrary operation string was accepted")
	}
	if _, err := decision.BuildTypedBinary("and", request.Left, request.Right); err == nil {
		t.Fatal("ill-typed operation was accepted")
	}
	if _, err := decision.AssembleGoExpression(decision.TypedBinaryIR{
		Schema: decision.TypedBinaryIRSchema, Kind: "binary", Operation: "add",
		Left: request.Left, Right: request.Right, ResultType: decision.TypeBool,
	}); err == nil {
		t.Fatal("inconsistent typed IR was assembled")
	}
}

func TestPublicPredictionEnforcesBoundedInput(t *testing.T) {
	model, err := decision.Load(filepath.Join("..", "runs", "pilot-mps-20260930-v1", "models", "fp32", "model.json"))
	if err != nil {
		t.Fatal(err)
	}
	var workspace decision.Workspace
	var prediction decision.Prediction
	if err := model.PredictInto(string(make([]byte, decision.InputMaxBytes+1)), &workspace, &prediction); err == nil {
		t.Fatal("oversized instruction was accepted")
	}
	if err := model.PredictInto(string([]byte{0xff, 'a'}), &workspace, &prediction); err == nil {
		t.Fatal("invalid UTF-8 instruction was accepted")
	}
	if err := model.PredictInto("Add two values", nil, &prediction); err == nil {
		t.Fatal("nil workspace was accepted")
	}
	if err := model.PredictInto("Add two values", &workspace, nil); err == nil {
		t.Fatal("nil prediction output was accepted")
	}
}

func TestPublicModelRejectsNilAndZeroValueModels(t *testing.T) {
	var workspace decision.Workspace
	var prediction decision.Prediction
	var nilModel *decision.Model
	if err := nilModel.PredictInto("Add two values", &workspace, &prediction); err == nil {
		t.Fatal("nil model prediction succeeded")
	}
	if _, err := nilModel.Decide(decision.DecisionRequest{}, &workspace); err == nil {
		t.Fatal("nil model decision succeeded")
	}
	var empty decision.Model
	if err := empty.PredictInto("Add two values", &workspace, &prediction); err == nil {
		t.Fatal("zero-value model prediction succeeded")
	}
	if _, err := empty.Decide(decision.DecisionRequest{
		Schema: decision.DecisionRequestSchema,
		Text:   "Add two values",
		Left:   decision.Identifier{Name: "left", Type: decision.TypeInt},
		Right:  decision.Identifier{Name: "right", Type: decision.TypeInt},
	}, &workspace); err == nil {
		t.Fatal("zero-value model decision succeeded")
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
