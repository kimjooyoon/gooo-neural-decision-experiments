package main

import (
	"encoding/json"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

func TestStorageReportCountsItsOwnBytes(t *testing.T) {
	for _, before := range []int64{0, 9, 99, 999, 123456789} {
		raw, err := storageReportBytes(map[string]any{"existing_full_input_bytes": before}, before)
		if err != nil {
			t.Fatal(err)
		}
		var value struct {
			Bytes int64 `json:"existing_full_input_bytes"`
		}
		if err = json.Unmarshal(raw, &value); err != nil || value.Bytes != before+int64(len(raw)) {
			t.Fatal("self-inclusive inventory", err)
		}
	}
}

func TestIndependentFormReconstruction(t *testing.T) {
	var fields [64]byte
	fields[0], fields[5] = 128, 128
	part, err := decision.EncodeSemanticContextInput(fields, "첫 변수를 보존한다.\nfeedback: failed=3")
	if err != nil {
		t.Fatal(err)
	}
	text, err := jointdecision.EncodeThree([3]string{part, part, part})
	if err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"en", "ko"} {
		for _, form := range fullinputstudy.TrainingForms {
			got, err := fullinputstudy.Apply(text, language, form)
			if err != nil {
				t.Fatal(err)
			}
			want, err := expectedText(text, language, form)
			if err != nil || want != got.Text {
				t.Fatal("independent full form differs", language, form, err)
			}
		}
	}
}
