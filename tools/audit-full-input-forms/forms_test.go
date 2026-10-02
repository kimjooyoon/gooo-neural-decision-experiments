package main

import (
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func formView(t *testing.T, split, language string) threecohort.View {
	t.Helper()
	prefix, err := authoredWrapper(split, language)
	if err != nil {
		t.Fatal(err)
	}
	intent := "Keep the source subtraction operands."
	if language == "ko" {
		intent = "원본 빼기 피연산자의 순서를 유지하세요."
	}
	var fields [64]byte
	fields[0], fields[5] = 128, 128
	part, err := decision.EncodeSemanticContextInput(fields, prefix+intent)
	if err != nil {
		t.Fatal(err)
	}
	text, err := jointdecision.EncodeThree([3]string{part, part, part})
	if err != nil {
		t.Fatal(err)
	}
	return threecohort.View{Split: split, Language: language, Text: text}
}

func TestAllDeclaredFormsRetainSourceAndExactOriginal(t *testing.T) {
	for _, split := range []string{"train", "calibration", "development"} {
		for _, language := range []string{"en", "ko"} {
			v := formView(t, split, language)
			original, err := features(v.Text, false)
			if err != nil {
				t.Fatal(err)
			}
			if original[0] == 0 {
				t.Fatal("feature result copied before encoding")
			}
			for _, form := range forms {
				input, err := applyForm(v, form)
				if err != nil || input.Declined {
					t.Fatal(split, language, form, err)
				}
				if form == "original" && input.Text != v.Text {
					t.Fatal("original input altered")
				}
				for _, bag := range []bool{false, true} {
					candidate, err := features(input.Text, bag)
					if err != nil || !preservedSource(&original, &candidate) {
						t.Fatal("source coordinates changed", form, err)
					}
				}
				if form == "bare" {
					part, err := jointdecision.ThreeParts(input.Text)
					if err != nil {
						t.Fatal(err)
					}
					if strings.Contains(part[0], "While composing,") || strings.Contains(part[0], "함수를 구성할 때,") || strings.Contains(part[0], "Decision request:") || strings.Contains(part[0], "구성 요청:") {
						t.Fatal("diagnostic prefix transformation differs")
					}
				}
			}
		}
	}
	v := formView(t, "development", "en")
	if _, err := applyForm(v, "unregistered"); err == nil {
		t.Fatal("unknown form accepted")
	}
	v.Text = strings.ReplaceAll(v.Text, "While composing, ", "Another wording! ")
	if _, err := applyForm(v, "bare"); err == nil {
		t.Fatal("unknown original wrapper replaced")
	}
}
