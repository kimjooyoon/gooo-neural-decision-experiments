package main

import "testing"

func TestAuditComparisonKeepsCountsExact(t *testing.T) {
	if compareValue("/views", float64(512), float64(511)) == nil {
		t.Fatal("count drift accepted")
	}
	if err := compareValue("/summed_passing_set_nll", 123.0, 123.000001); err != nil {
		t.Fatal(err)
	}
	if compareValue("/summed_passing_set_nll", 123.0, 123.01) == nil {
		t.Fatal("large numerical drift accepted")
	}
	if compareValue("/maximum_absolute_export_parity_error", 1e-6, 2e-5) == nil {
		t.Fatal("parity bound ignored")
	}
	if compareValue("", map[string]any{"a": float64(1)}, map[string]any{"b": float64(1)}) == nil {
		t.Fatal("key drift accepted")
	}
}
