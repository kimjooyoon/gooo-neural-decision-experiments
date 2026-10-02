package fullinputstudy

import (
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

func TestFullFormsPreserveOriginalAndOverflow(t *testing.T) {
	var fields [64]byte
	fields[0], fields[5] = 128, 128
	unit, _ := decision.EncodeSemanticContextInput(fields, "x")
	for _, intent := range []string{"Keep the first variable. 원문.", strings.Repeat("x", 512-len(unit)+1)} {
		part, err := decision.EncodeSemanticContextInput(fields, intent)
		if err != nil {
			t.Fatal(err)
		}
		text, err := jointdecision.EncodeThree([3]string{part, part, part})
		if err != nil {
			t.Fatal(err)
		}
		for _, language := range []string{"en", "ko"} {
			for _, form := range append(TrainingForms[:], EvaluationForms[:]...) {
				got, err := Apply(text, language, form)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(got.Text, intent) != 3 || form == "original" && got.Text != text {
					t.Fatal("full text changed or truncated")
				}
				if len(intent) > 300 && form != "original" && !got.Declined {
					t.Fatal("overflow was inferred")
				}
			}
		}
	}
	if _, err := Apply("bad", "en", "original"); err == nil {
		t.Fatal("invalid framing accepted")
	}
}
