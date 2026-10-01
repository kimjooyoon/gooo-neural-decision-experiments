package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func testView(t *testing.T, family string, desired int, language string) view {
	t.Helper()
	plan, err := compositionstudy.Fixture(family, 40, desired, language)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		t.Fatal(err)
	}
	finite, err := compositionstudy.Target(plan, family, 40, desired)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(prepared.Fallback().GoooBody())
	if err != nil {
		t.Fatal(err)
	}
	source := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	v := view{ID: family + language}
	for i, c := range plan.Decisions {
		r := row{ID: c.ID, Group: family, Family: family, Config: 40, Desired: desired, Language: language,
			Feature: decision.SemanticContextIntentFeatureVersion, Coordinate: i, Finite: finite, SourceSHA: hash(source),
			Options: [2]string{c.Options[0].Label, c.Options[1].Label}, Targets: finite.Marginals[i]}
		fields, e := prepared.SourceFeatures(c.ID)
		if e != nil {
			t.Fatal(e)
		}
		natural := c.Intent[strings.LastIndex(c.Intent, "intent: ")+8:]
		r.Input.ID, r.Input.Natural = c.ID, "sha256:"+hash([]byte(natural))
		r.Input.Text, e = decision.EncodeSemanticContextInput(fields, natural)
		if e != nil {
			t.Fatal(e)
		}
		r.Input.SHA = "sha256:" + hash([]byte(r.Input.Text))
		v.Rows[i] = r
	}
	return v
}

func TestAllNativeSubsetViewsCompleteWithoutModel(t *testing.T) {
	for _, family := range compositionstudy.Families {
		for desired := 0; desired < 4; desired++ {
			for _, language := range []string{"en", "ko"} {
				v := testView(t, family, desired, language)
				o, err := observe(v, nil)
				if err != nil || o.Search.Selection.ModelCalls != 0 || o.Search.SelectedTrainingPassed != 16 {
					t.Fatal(v.ID, err)
				}
				if err = receiptLinks(o); err != nil {
					t.Fatal("actual progress hash chain", err)
				}
				if err = validateObservation(v, o, true); err == nil {
					t.Fatal("missing initial model calls accepted")
				}
				o.Search.Attempts[0].Results[0].Actual++
				if err = validateObservation(v, o, false); err == nil {
					t.Fatal("independent actual mismatch accepted")
				}
			}
		}
	}
}

func TestPublishedOwnModelsNumericalParity(t *testing.T) {
	root := filepath.Join("..", "..", "models", "fresh-composition-v1")
	loaded, _, err := loadModels(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, arm := range arms {
		var rows struct {
			Rows []struct {
				ID   string `json:"row_id"`
				Text string `json:"text"`
			} `json:"rows"`
		}
		if err = decodeFile(filepath.Join(root, arm, "go-parity.json"), &rows); err != nil {
			t.Fatal(err)
		}
		var views []view
		for _, r := range rows.Rows {
			v := view{}
			v.Rows[0].ID, v.Rows[0].Split, v.Rows[0].Input.Text = r.ID, "development", r.Text
			views = append(views, v)
		}
		if _, err = parity(root, arm, loaded, views); err != nil {
			t.Fatal(arm, err)
		}
	}
}

func TestSourceAndFiniteTargetTamperingRejected(t *testing.T) {
	v := testView(t, "reference_operand", 0, "ko")
	v.Rows[0].SourceSHA = "bad"
	if _, _, err := prepareView(v); err == nil {
		t.Fatal("bad original source accepted")
	}
	v = testView(t, "assignment_branch", 1, "en")
	v.Rows[0].Finite.Passed[0]++
	if _, _, err := prepareView(v); err == nil {
		t.Fatal("bad finite oracle target accepted")
	}
}

func TestFrozenSelectorOrder(t *testing.T) {
	pins := map[string]modelPin{}
	scores := map[string]score{}
	for _, a := range arms {
		for _, v := range variants {
			id := a + "-" + v
			pins[id] = modelPin{Packed: 100}
			scores[id] = score{Total: counts{Extras: 10, Calls: 20, Different: 3}}
		}
	}
	x := scores["v3-qat_ternary"]
	x.Total.Extras = 9
	scores["v3-qat_ternary"] = x
	id, err := choose(scores, pins)
	if err != nil || id != "v3-qat_ternary" {
		t.Fatal(id, err)
	}
	x.Total.Extras = 10
	x.Total.Calls = 19
	scores["v3-qat_ternary"] = x
	id, err = choose(scores, pins)
	if err != nil || id != "v3-qat_ternary" {
		t.Fatal(id, err)
	}
	x.Total.Calls = 20
	x.Total.Different = 2
	scores["v3-qat_ternary"] = x
	id, err = choose(scores, pins)
	if err != nil || id != "v3-qat_ternary" {
		t.Fatal(id, err)
	}
	x.Total.Different = 3
	scores["v3-qat_ternary"] = x
	pins["v3-qat_ternary"] = modelPin{Packed: 99}
	id, err = choose(scores, pins)
	if err != nil || id != "v3-qat_ternary" {
		t.Fatal(id, err)
	}
	pins["v3-qat_ternary"] = modelPin{Packed: 100}
	id, err = choose(scores, pins)
	if err != nil || id != "v2-fp32" {
		t.Fatal(id, err)
	}
}

func TestCompletenessDenominatorCarriesAcrossBudgets(t *testing.T) {
	o := observation{Search: pathplan.SearchResult{Attempts: []pathplan.SearchAttempt{{Passed: 3}, {Passed: 11}, {Passed: 16}}, Selection: pathplan.Selection{ModelCalls: 4}}}
	var c counts
	add(&c, o)
	if c.Cases != 16 || c.InitialPassed != 3 || c.Extras != 2 || c.MinimumBudget != [4]int{0, 0, 1, 0} ||
		c.CurveComplete != [4]int{0, 0, 1, 1} || c.CurvePassed != [4]int{3, 11, 16, 16} {
		t.Fatalf("%+v", c)
	}
}
