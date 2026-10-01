package bilingualstudy

import (
	"reflect"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
)

func TestFrozenPairDenominatorsAndSourceBinding(t *testing.T) {
	pairs, err := Load("../../data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	counts, groups, splits := map[string]int{}, map[string]bool{}, map[string]string{}
	for _, p := range pairs {
		counts[p.Split]++
		groups[p.Program] = true
		for _, key := range []string{p.Program, p.Template, p.Rows[0].ConfigurationID} {
			if old, ok := splits[key]; ok && old != p.Split {
				t.Fatal("split leakage", key)
			}
			splits[key] = p.Split
		}
		if p.Targets[0]+p.Targets[1] != 1 || p.Hashes[0] != p.Rows[0].InputSHA ||
			p.Hashes[1] != p.Rows[1].InputSHA || p.Rows[0].Language != "en" || p.Rows[1].Language != "ko" {
			t.Fatal("pair binding lost", p.ID)
		}
	}
	if len(groups) != 640 || !reflect.DeepEqual(counts, map[string]int{"train": 800, "calibration": 80, "test": 160}) {
		t.Fatal("reused denominator differs", counts, len(groups))
	}
}

func TestRejectDuplicateMissingAndConflictingPairs(t *testing.T) {
	pairs, err := Load("../../data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	a, b := pairs[0].Rows[0], pairs[0].Rows[1]
	if _, err := Group([]feedbackstudy.Row{a}); err == nil {
		t.Fatal("missing Korean row accepted")
	}
	if _, err := Group([]feedbackstudy.Row{a, b, b}); err == nil {
		t.Fatal("duplicate row accepted")
	}
	for _, mutate := range []func(*feedbackstudy.Row){
		func(r *feedbackstudy.Row) { r.Split = "test" },
		func(r *feedbackstudy.Row) { r.IntentionLabel = "different" },
		func(r *feedbackstudy.Row) { r.Cases = nil },
		func(r *feedbackstudy.Row) { r.OriginalText = "different Gooo body\nintent: 한글" },
	} {
		other := b
		mutate(&other)
		if _, err := Group([]feedbackstudy.Row{a, other}); err == nil {
			t.Fatal("conflicting pair accepted")
		}
	}
}
