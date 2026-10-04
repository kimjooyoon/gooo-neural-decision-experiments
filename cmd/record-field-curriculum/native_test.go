package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestCapturedNativeValuesAreIndependentlyRecounted(t *testing.T) {
	for _, tc := range []struct {
		file          string
		counter       bool
		named, fields int
	}{
		{"full.json", false, 8, 12}, {"counter-partial.json", true, 4, 6},
	} {
		raw, err := os.ReadFile("testdata/" + tc.file)
		if err != nil {
			t.Fatal(err)
		}
		got := summarizeNative(raw, nativeCases(6, tc.counter))
		if got.RuntimePassed != tc.named || got.RuntimeTotal != 8 || got.FieldsPassed != tc.fields || got.FieldsTotal != 12 || got.Calls != 1 || got.Attempts != 1 || got.BuildRSS <= 0 || got.RunRSS <= 0 {
			t.Fatalf("unexpected observation: %+v", got)
		}
	}
}

func TestCapturedNativeInputsDeliveryAndCompletionAreChecked(t *testing.T) {
	raw, err := os.ReadFile("testdata/full.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"different-root", func(c map[string]any) { root(c)["inputs"].([]any)[1].(map[string]any)["value"] = false }},
		{"different-delivery", func(c map[string]any) { child(c)["input"] = map[string]any{"flag": "invented"} }},
		{"different-producer", func(c map[string]any) { child(c)["producer_id"] = "other" }},
		{"unfinished-run", func(c map[string]any) {
			c["runtime"].(map[string]any)["runs"].([]any)[0].(map[string]any)["completed"] = false
		}},
		{"different-count", func(c map[string]any) { c["runtime"].(map[string]any)["finite_passed"] = 7 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var c map[string]any
			if err := json.Unmarshal(raw, &c); err != nil {
				t.Fatal(err)
			}
			tc.edit(c)
			changed, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if recover() == nil {
					t.Fatal("inconsistent trace accepted")
				}
			}()
			summarizeNative(changed, nativeCases(6, false))
		})
	}
}

func root(c map[string]any) map[string]any {
	return c["runtime"].(map[string]any)["traces"].([]any)[0].(map[string]any)["deliveries"].([]any)[0].(map[string]any)
}
func child(c map[string]any) map[string]any {
	return c["runtime"].(map[string]any)["traces"].([]any)[0].(map[string]any)["deliveries"].([]any)[1].(map[string]any)
}

func TestCounterIntentsPreserveActualAlternatives(t *testing.T) {
	for _, language := range []string{"ko", "en"} {
		before := fixture(6, orders[0], 3, language)
		after := counterSource(Row{Family: 6, Language: language}, before)
		if bytes.Equal(before, after) || bytes.Count(after, []byte("field_value at")) != 3 || bytes.Count(after, []byte("value_case")) != 5 {
			t.Fatal("counter intent source shape differs")
		}
		// All first/second expression text remains byte-identical.
		for _, token := range []string{`first "input0.message"`, `second "input0.flag"`, `"wait"`, `:accepted`} {
			if bytes.Count(before, []byte(token)) != bytes.Count(after, []byte(token)) {
				t.Fatal("counter source rewrote a candidate", token)
			}
		}
	}
}
