package main

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestCalibrationOrderingKeepsAllControlDimensions(t *testing.T) {
	models := map[string]*model{"a": {Pin: pin{Packed: 2}}, "b": {Pin: pin{Packed: 1}}, "offline": nil}
	cal := map[string]counts{"a": {}, "b": {}, "offline": {}}
	if choose(cal, models) != "b" {
		t.Fatal("packed-byte tie breaker ignored")
	}
	for _, field := range []string{"Extra", "Predictions", "Disagreement"} {
		a, b := counts{}, counts{}
		switch field {
		case "Extra":
			b.Extra = 1
		case "Predictions":
			b.Predictions = 1
		case "Disagreement":
			b.Disagreement = 1
		}
		cal["a"], cal["b"] = a, b
		if choose(cal, models) != "a" {
			t.Fatal(field + " did not precede packed bytes")
		}
	}
	models["a"].Pin.Packed = 1
	cal["a"], cal["b"] = counts{}, counts{}
	if choose(cal, models) != "a" {
		t.Fatal("stable candidate tie breaker ignored")
	}
}
func TestAggregateToleranceNeverAcceptsChangedCountsOrNonfinite(t *testing.T) {
	a := counts{Views: 384, Predictions: 775, SetNLL: 40}
	b := a
	b.SetNLL = math.Nextafter(a.SetNLL, math.Inf(1))
	if !equalCounts(a, b) {
		t.Fatal("one-ULP rederivation rejected")
	}
	b = a
	b.Predictions++
	if equalCounts(a, b) {
		t.Fatal("changed prediction count accepted")
	}
	b = a
	b.Curve[2]++
	if equalCounts(a, b) {
		t.Fatal("changed completeness curve accepted")
	}
	for _, x := range []float64{math.NaN(), math.Inf(1), 40.1} {
		b = a
		b.SetNLL = x
		if equalCounts(a, b) {
			t.Fatal("invalid derived value accepted")
		}
	}
}
func testView(t *testing.T) jointcohort.View {
	t.Helper()
	p, e := jointcompositionstudy.Fixture("schedule_operand", 0, 1, "en")
	if e != nil {
		t.Fatal(e)
	}
	original, e := pathplan.Prepare(p)
	if e != nil {
		t.Fatal(e)
	}
	body, e := json.Marshal(original.Fallback().GoooBody())
	if e != nil {
		t.Fatal(e)
	}
	originalSource := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	for i, c := range p.Decisions {
		fields, e := original.SourceFeatures(c.ID)
		if e != nil {
			t.Fatal(e)
		}
		text, e := decision.EncodeSemanticContextInput(fields, c.Intent[strings.LastIndex(c.Intent, "intent: ")+8:])
		if e != nil {
			t.Fatal(e)
		}
		p.Decisions[i].Intent = text
	}
	prepared, e := pathplan.Prepare(p)
	if e != nil {
		t.Fatal(e)
	}
	text, e := prepared.JointInput()
	if e != nil {
		t.Fatal(e)
	}
	cases, e := jointcompositionstudy.Cases("schedule_operand", 0, 1)
	if e != nil {
		t.Fatal(e)
	}
	target, e := jointcompositionstudy.Target(p, "schedule_operand", 0, 1)
	if e != nil {
		t.Fatal(e)
	}
	return jointcohort.View{ID: "fixture/en", Group: "fixture", Family: "schedule_operand", Config: 0, Goal: 1, Language: "en", SourceSHA: jointcohort.SHA(originalSource), JointInput: text, Prepared: prepared, Cases: cases, Target: target}
}
func TestActualOfflineAndReferenceReceiptAudits(t *testing.T) {
	v := testView(t)
	baseline, e := decision.LoadPath("../../models/joint-composition-v1/independent/models/fp32/model.json")
	if e != nil {
		t.Fatal(e)
	}
	m := &model{Independent: baseline, Pin: pin{baseline.MetadataSHA256(), baseline.WeightsSHA256(), baseline.PackedFileBytes()}}
	for _, selected := range []*model{nil, m} {
		c, e := one(v, selected)
		if e != nil {
			t.Fatal(e)
		}
		if e = verifyObservation(v, c, selected); e != nil {
			t.Fatal(e)
		}
		for _, mutate := range []func(*jointfeedback.Capture){
			func(c *jointfeedback.Capture) { c.SourceSHA = "changed" },
			func(c *jointfeedback.Capture) { c.Search.Attempts[0].Results[0].Actual++ },
			func(c *jointfeedback.Capture) { c.Progress[1].BestCases[0].Actual++ },
			func(c *jointfeedback.Capture) { c.Progress[0].Selection.Receipts[0].IntentSHA256 = "changed" },
			func(c *jointfeedback.Capture) { c.Search.Selection.ModelCalls++ },
		} {
			raw, _ := json.Marshal(c)
			var copy jointfeedback.Capture
			if e = json.Unmarshal(raw, &copy); e != nil {
				t.Fatal(e)
			}
			mutate(&copy)
			if verifyObservation(v, copy, selected) == nil {
				t.Fatal("tampered reference observation accepted")
			}
		}
	}
}
