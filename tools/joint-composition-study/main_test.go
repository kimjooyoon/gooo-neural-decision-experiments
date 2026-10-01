package main

import (
	"encoding/json"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func testView(t *testing.T, family string, desired int, language string) view {
	t.Helper()
	plan, err := jointcompositionstudy.Fixture(family, 40, desired, language)
	if err != nil {
		t.Fatal(err)
	}
	base, err := pathplan.Prepare(plan)
	if err != nil {
		t.Fatal(err)
	}
	finite, err := jointcompositionstudy.Target(plan, family, 40, desired)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(base.Fallback().GoooBody())
	source := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	v := view{ID: family + language}
	var parts [2]string
	for i, c := range plan.Decisions {
		r := row{ID: c.ID, Group: family, Family: family, Config: 40, Desired: desired, Language: language, Feature: decision.SemanticContextIntentFeatureVersion, Coordinate: i, Finite: finite, SourceSHA: hash(source), Options: [2]string{c.Options[0].Label, c.Options[1].Label}, Targets: finite.Marginals[i]}
		fields, err := base.SourceFeatures(c.ID)
		if err != nil {
			t.Fatal(err)
		}
		natural := strings.SplitN(c.Intent, "intent: ", 2)[1]
		r.Input.ID, r.Input.Natural = c.ID, "sha256:"+hash([]byte(natural))
		r.Input.Text, err = decision.EncodeSemanticContextInput(fields, natural)
		if err != nil {
			t.Fatal(err)
		}
		r.Input.SHA = "sha256:" + hash([]byte(r.Input.Text))
		v.Rows[i] = r
		parts[i] = r.Input.Text
	}
	joint, err := jointdecision.Encode(parts)
	if err != nil {
		t.Fatal(err)
	}
	for i := range v.Rows {
		v.Rows[i].JointText, v.Rows[i].JointSHA = joint, hash([]byte(joint))
	}
	return v
}
func TestNewNativeSubsetOfflineAndSourceTargetTamperRejections(t *testing.T) {
	for _, family := range jointcompositionstudy.Families {
		for goal := 0; goal < 4; goal++ {
			for _, lang := range [2]string{"en", "ko"} {
				v := testView(t, family, goal, lang)
				o, err := observe(v, nil)
				if err != nil || o.Search.SelectedTrainingPassed != 16 || o.Search.Selection.ModelCalls != 0 {
					t.Fatal(family, goal, lang, err)
				}
				o.Search.Attempts[0].Results[0].Actual++
				if err = validateObservation(v, o, "offline"); err == nil {
					t.Fatal("ordinary oracle mismatch accepted")
				}
			}
		}
	}
	v := testView(t, jointcompositionstudy.Families[0], 0, "en")
	v.Rows[0].JointText += "x"
	if _, _, err := prepareView(v); err == nil {
		t.Fatal("partial/reordered joint input accepted")
	}
}
func TestAllPublishedExportsGoPythonParity(t *testing.T) {
	root := filepath.Join("..", "..", "models", "joint-composition-v1")
	loaded, _, err := loadModels(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, arm := range arms {
		var document struct {
			Rows []struct {
				ID   string `json:"row_id"`
				Text string `json:"text"`
			} `json:"rows"`
		}
		if err = decodeFile(filepath.Join(root, arm, "go-parity.json"), &document); err != nil {
			t.Fatal(err)
		}
		var views []view
		for _, r := range document.Rows {
			v := view{}
			v.Rows[0].ID, v.Rows[0].Split, v.Rows[0].Input.Text, v.Rows[0].JointText = r.ID, "development", r.Text, r.Text
			views = append(views, v)
		}
		if _, err = parity(root, arm, loaded, views); err != nil {
			t.Fatal(arm, err)
		}
	}
}
func TestFrozenSelectionOrder(t *testing.T) {
	pins, scores := map[string]modelPin{}, map[string]score{}
	for _, arm := range arms {
		for _, variant := range variants {
			id := arm + "-" + variant
			pins[id] = modelPin{Packed: 100}
			scores[id] = score{Total: counts{Extras: 10, Calls: 12, Different: 2}}
		}
	}
	s := scores["joint-qat_ternary"]
	s.Total.Calls = 11
	scores["joint-qat_ternary"] = s
	chosen, err := choose(scores, pins)
	if err != nil || chosen != "joint-qat_ternary" {
		t.Fatal("frozen call tie break", chosen, err)
	}
}

func TestDescriptiveDistributionPreservesInputsAndRejectsNonfiniteValues(t *testing.T) {
	values := []float64{4, 1, 3, 2}
	d, err := describe(values)
	if err != nil || d.Mean != 2.5 || d.Median != 2.5 || d.P95 != 4 || d.Maximum != 4 || values[0] != 4 {
		t.Fatal(d, err, values)
	}
	for _, invalid := range [][]float64{nil, {-1}, {math.NaN()}, {math.Inf(1)}} {
		if _, err = describe(invalid); err == nil {
			t.Fatal("invalid observations accepted")
		}
	}
}

func TestOnlyDerivedFloatingAggregatesToleratePlatformRounding(t *testing.T) {
	a := counts{Views: 384, Cases: 6144, Calls: 777, PredictionNS: 123, InitialOutside: 271.007082, InitialNLL: 530.123456}
	b := a
	b.InitialOutside = math.Nextafter(b.InitialOutside, math.Inf(1))
	b.InitialNLL = math.Nextafter(b.InitialNLL, math.Inf(-1))
	if !sameCounts(a, b) {
		t.Fatal("one-ulp derived aggregate rejected")
	}
	for _, change := range []func(*counts){func(c *counts) { c.Calls++ }, func(c *counts) { c.CurveComplete[2]++ }, func(c *counts) { c.PredictionNS++ }, func(c *counts) { c.InitialOutside += 1e-6 }, func(c *counts) { c.InitialNLL = math.NaN() }} {
		b = a
		change(&b)
		if sameCounts(a, b) {
			t.Fatal("non-rounding change accepted", b)
		}
	}
}
