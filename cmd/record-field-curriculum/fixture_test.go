package main

import (
	"strings"
	"testing"
)

func TestFixtureCurriculumIsBalancedAndSourceOwned(t *testing.T) {
	seen := map[string]bool{}
	counts := map[string]int{}
	for family := range 8 {
		for _, order := range orders {
			for mask := uint16(0); mask < 8; mask++ {
				for _, language := range []string{"ko", "en"} {
					source := fixture(family, order, mask, language)
					if seen[hash(source)] || strings.Count(string(source), "field_value at") != 3 || strings.Count(string(source), "value_case") != 5 ||
						!strings.Contains(string(source), "bind Select.result -> Label.input") {
						t.Fatal("duplicate or incomplete source-owned fixture")
					}
					seen[hash(source)] = true
					counts[split(family)]++
				}
			}
		}
	}
	if len(seen) != 768 || counts["train"] != 384 || counts["calibration"] != 192 || counts["test"] != 192 {
		t.Fatal("source family split differs", counts)
	}
}
