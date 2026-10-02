package threestudent

import (
	"bytes"
	"math"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

func TestEqualLanguageAndGroupWeights(t *testing.T) {
	var states []threefeedback.State
	for _, lang := range []string{"en", "ko"} {
		states = append(states, threefeedback.State{Group: "group", Language: lang, Phase: "initial", Split: "train"})
		n := 3
		if lang == "ko" {
			n = 1
		}
		for range n {
			states = append(states, threefeedback.State{Group: "group", Language: lang, Phase: "feedback", Split: "train"})
		}
	}
	rows, err := Rows(states)
	if err != nil {
		t.Fatal(err)
	}
	var a, b float64
	for _, r := range rows {
		a += r.InitialWeight
		b += r.FeedbackWeight
	}
	if a != 1 || math.Abs(b-1) > 1e-12 || rows[0].FeedbackWeight != .25 || rows[1].FeedbackWeight != .25/3 {
		t.Fatal(rows)
	}
	states[1].Split = "development"
	if _, err = Rows(states); err == nil {
		t.Fatal("holdout feedback accepted")
	}
}

func TestNoFeedbackAndMissingLanguage(t *testing.T) {
	states := []threefeedback.State{{Group: "group", Language: "en", Phase: "initial", Split: "calibration"}, {Group: "group", Language: "ko", Phase: "initial", Split: "calibration"}}
	rows, err := Rows(states)
	if err != nil || rows[0].FeedbackWeight != .5 {
		t.Fatal(rows, err)
	}
	if _, err = Rows(states[:1]); err == nil {
		t.Fatal("unpaired group accepted")
	}
	if _, err = Rows(append(states, states[0])); err == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestFreshInitializationIsDeterministicAndBounded(t *testing.T) {
	a, b := InitialWeights(), InitialWeights()
	if len(a) != 74624 || !bytes.Equal(a, b) || threecohort.SHA(a) != InitialSHA {
		t.Fatal("fresh common initialization differs")
	}
	t.Logf("initialization SHA: %s", threecohort.SHA(a))
}
