package main

import "testing"

func TestFrozenMetricsAndEvenMedian(t *testing.T) {
	t.Chdir("../..")
	if median([]float64{9, 1, 7, 3}) != 5 {
		t.Fatal("median differs")
	}
	value, err := generate()
	if err != nil || len(value["summaries"].([]summary)) != 18 || value["new_model_predictions"] != 0 {
		t.Fatal("frozen metrics failed", err)
	}
	if _, err = summarize("invalid", "invalid", nil); err == nil {
		t.Fatal("missing cohort accepted")
	}
}
