package main

import "testing"

func TestUnequalTargetsWithCommonExecutableChoice(t *testing.T) {
	groups := map[string][]sample{"input": {
		{Target: [4]float32{0.5, 0, 0, 0.5}, Support: 9},
		{Target: [4]float32{1, 0, 0, 0}, Support: 1},
	}}
	r := summarize(groups)
	if r.DifferentTargets != 1 || r.EmptyCommon != 0 || r.FixedPass != 2 || r.UnavoidableFailed != 0 || r.Collisions[0].CommonSupport != 1 {
		t.Fatalf("distribution variation incorrectly classified as impossibility: %+v", r)
	}
}

func TestPairwiseOverlapDoesNotImplyGlobalCommonChoice(t *testing.T) {
	groups := map[string][]sample{"input": {
		{Target: [4]float32{0.5, 0.5, 0, 0}, Support: 3},
		{Target: [4]float32{0, 0.5, 0.5, 0}, Support: 6},
		{Target: [4]float32{0.5, 0, 0.5, 0}, Support: 5},
	}}
	r := summarize(groups)
	if r.EmptyCommon != 1 || r.FixedPass != 2 || r.UnavoidableFailed != 1 || r.Collisions[0].CommonSupport != 0 {
		t.Fatalf("incompatible executable supports were lost: %+v", r)
	}
}

func TestRepeatedViewsCountTowardsCoverageNotIntentCount(t *testing.T) {
	groups := map[string][]sample{"input": {
		{Target: [4]float32{1, 0, 0, 0}, Support: 1},
		{Target: [4]float32{1, 0, 0, 0}, Support: 1},
		{Target: [4]float32{0, 1, 0, 0}, Support: 2},
	}}
	r := summarize(groups)
	if r.Observations != 3 || r.Distinct != 1 || r.FixedPass != 2 || r.UnavoidableFailed != 1 {
		t.Fatalf("wrong frequency-aware fixed-choice upper bound: %+v", r)
	}
}
