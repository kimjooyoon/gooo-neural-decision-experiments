package feedbackstudy

import (
	"strings"
	"testing"
)

func fixture(view string) Original {
	return Original{ID: "test", InstructionID: "instruction", ProgramID: "program", TemplateID: "template",
		ConfigurationID: "config", ConfigurationIndex: 32, Family: "operand_order", Language: "en",
		Split: "train", View: view, Text: "Subtract the offset from the input.", Label: "layout_forward"}
}

func TestSparseCasesRetainMultipleBestLabels(t *testing.T) {
	r, err := Derive(fixture("gooo"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Cases) != 1 || len(r.Accepted) != 2 || r.BestPassed != 1 ||
		r.OptionPassed != [2]int{1, 1} || r.FunctionalIsIntent || r.CIIsAuthority {
		t.Fatal("finite ambiguity was turned into a unique semantic answer")
	}
	full, err := Derive(fixture("plain"))
	if err != nil || len(full.Accepted) != 1 || full.Accepted[0] != "layout_forward" || full.BestPassed != len(full.Cases) {
		t.Fatal("complete finite witnesses lost the original behavior", err)
	}
}

func TestContradictoryCasesRetainPartialDenominator(t *testing.T) {
	r, err := Derive(fixture("prov"))
	if err != nil || r.Contract != "inconsistent_finite" || r.BestPassed >= len(r.Cases) {
		t.Fatal("contradiction was hidden as complete", err)
	}
	first, last := r.Cases[0], r.Cases[len(r.Cases)-1]
	if first.Input != last.Input || first.Expected == last.Expected || r.IntentionLabel != "layout_forward" {
		t.Fatal("partial curriculum changed the source intention or contradictory witness")
	}
}

func TestLongContextsRetainOriginalTextWithoutInferenceEligibility(t *testing.T) {
	source := fixture("plain")
	source.Text = strings.Repeat("detail ", 70)
	r, err := Derive(source)
	if err != nil || r.Eligible || r.Representation != "context_declined" || r.InputBytes <= 512 ||
		r.OriginalText != source.Text || r.OriginalSHA != Hash([]byte(source.Text)) ||
		!strings.HasSuffix(r.Text, "\nintent: "+source.Text) || r.InputSHA != Hash([]byte(r.Text)) {
		t.Fatal("over-bound training context was truncated or admitted", err)
	}
}
