package main

import (
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
)

func TestEligiblePairRankingIgnoresOtherLabelsAndRetainsTies(t *testing.T) {
	model, err := decision.LoadPath("../../runs/typed-path-positioned-random-20261001/models/fp32/model.json")
	if err != nil {
		t.Fatal(err)
	}
	p := decision.Prediction{Logits: [8]float32{1e20, 0, 0, 0, -1000, 1000}}
	options := [2]string{"layout_forward", "layout_reverse"}
	if result := probability(model, p, options); result != [2]float64{0, 1} {
		t.Fatal("eligible pair was replaced by global label or underflow", result)
	}
	p.Logits[4] = 1000
	if result := probability(model, p, options); result != [2]float64{0.5, 0.5} {
		t.Fatal("equal eligible logits lost their tie", result)
	}
}

func TestOfflineFiniteSetObservationMakesNoPredictions(t *testing.T) {
	row, err := feedbackstudy.Derive(feedbackstudy.Original{ID: "one", InstructionID: "intent", ProgramID: "program",
		TemplateID: "template", ConfigurationID: "config", ConfigurationIndex: 32,
		Family: "operand_order", Language: "en", Split: "test", View: "gooo",
		Text: "Subtract the offset from the input.", Label: "layout_forward"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := observe([]feedbackstudy.Row{row}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := result["scores"].(counters)
	if c.Calls != 0 || c.Best != 1 || c.Passed != 1 || c.Total != 1 || c.Ambiguous != 1 || c.Mass != 1 {
		t.Fatal("offline observation lost ambiguity, denominator or zero-call accounting", c)
	}
}
