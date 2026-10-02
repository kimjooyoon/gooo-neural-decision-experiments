package main

import (
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func TestReplayWrapperRetainsBodyAndHeader(t *testing.T) {
	var fields [64]byte
	fields[0], fields[5] = 128, 128
	part, e := decision.EncodeSemanticContextInput(fields, "함수를 구성할 때, 먼저 선언한 정수 변수를 읽어라.")
	if e != nil {
		t.Fatal(e)
	}
	v := threecohort.View{Language: "ko", Split: "development", Parts: [3]string{part, part, part}}
	v.Text, e = jointdecision.EncodeThree(v.Parts)
	if e != nil {
		t.Fatal(e)
	}
	for _, form := range forms {
		text, e := expected(v, form)
		if e != nil {
			t.Fatal(e)
		}
		parts, e := jointdecision.ThreeParts(text)
		if e != nil {
			t.Fatal(e)
		}
		for _, p := range parts {
			if !strings.Contains(p, "먼저 선언한 정수 변수를 읽어라.") {
				t.Fatal("body lost")
			}
		}
	}
	v.Split = "calibration"
	if _, e = expected(v, "bare"); e == nil {
		t.Fatal("wrong authored wrapper accepted")
	}
}

func TestIndependentCountersKeepTiesAndFiniteDenominators(t *testing.T) {
	var v threecohort.View
	v.Target.Cases = 2
	v.Target.Passed = [8]int{2, 1, 0, 2, 0, 0, 0, 0}
	p := jointdecision.ThreePrediction{Mask: 1, Probabilities: [8]float32{.2, .5, 0, .3}}
	var c counts
	if e := increment(&c, v, p); e != nil {
		t.Fatal(e)
	}
	if c.Views != 1 || c.Cases != 2 || c.Passed != 1 || c.Complete != 0 || c.Extra != 1 || c.Curve != [8]int{0, 1, 1, 1, 1, 1, 1, 1} {
		t.Fatalf("wrong finite counters: %+v", c)
	}
	p = jointdecision.ThreePrediction{Mask: 0, Probabilities: [8]float32{.5, 0, 0, .5}}
	if e := increment(&c, v, p); e != nil {
		t.Fatal(e)
	}
	p.Mask = 3
	if e := increment(&c, v, p); e == nil {
		t.Fatal("noncanonical tie accepted")
	}
}
