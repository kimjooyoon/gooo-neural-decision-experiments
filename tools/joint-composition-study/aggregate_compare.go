package main

import "math"

// Only derived floating aggregates tolerate platform math-library rounding.
// Every integer count, budget curve and duration remains exact.
func sameAggregateFloat(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) {
		return false
	}
	return math.Abs(a-b) <= 1e-10+1e-12*math.Max(math.Abs(a), math.Abs(b))
}
func sameCounts(a, b counts) bool {
	outa, outb, nlla, nllb := a.InitialOutside, b.InitialOutside, a.InitialNLL, b.InitialNLL
	a.InitialOutside, b.InitialOutside, a.InitialNLL, b.InitialNLL = 0, 0, 0, 0
	return a == b && sameAggregateFloat(outa, outb) && sameAggregateFloat(nlla, nllb)
}
func sameCountMaps(a, b map[string]counts) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		other, ok := b[key]
		if !ok || !sameCounts(value, other) {
			return false
		}
	}
	return true
}
func sameScore(a, b score) bool {
	return sameCounts(a.Total, b.Total) && sameCountMaps(a.Families, b.Families)
}
func sameScoreMaps(a, b map[string]score) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		other, ok := b[key]
		if !ok || !sameScore(value, other) {
			return false
		}
	}
	return true
}

type roundingDifference struct {
	Cell       string  `json:"cell"`
	Scope      string  `json:"scope"`
	Metric     string  `json:"metric"`
	Captured   float64 `json:"captured"`
	Recomputed float64 `json:"recomputed"`
}

func appendRounding(out []roundingDifference, cell, scope string, a, b counts) []roundingDifference {
	if a.InitialOutside != b.InitialOutside {
		out = append(out, roundingDifference{cell, scope, "initial_probability_mass_outside_full_target", b.InitialOutside, a.InitialOutside})
	}
	if a.InitialNLL != b.InitialNLL {
		out = append(out, roundingDifference{cell, scope, "initial_full_target_nll", b.InitialNLL, a.InitialNLL})
	}
	return out
}
