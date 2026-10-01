package jointcompositionstudy

import (
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"reflect"
	"strings"
	"testing"
)

func TestFrozenCohortEveryCombinationAndCompleteNaturalInputs(t *testing.T) {
	views := 0
	splits := map[string]int{}
	scales := map[int64]bool{}
	for _, family := range Families {
		for c := 0; c < 48; c++ {
			_, s, _ := Parameters(c)
			scales[s] = true
			for desired := 0; desired < 4; desired++ {
				var previous FiniteTarget
				for _, lang := range [2]string{"en", "ko"} {
					plan, err := Fixture(family, c, desired, lang)
					if err != nil {
						t.Fatal(err)
					}
					target, err := Target(plan, family, c, desired)
					if err != nil {
						t.Fatal(err)
					}
					if target.Cases != 16 || target.Passed[desired] != 16 {
						t.Fatal("desired full target")
					}
					if lang == "ko" && !reflect.DeepEqual(previous, target) {
						t.Fatal("bilingual arithmetic changed")
					}
					previous = target
					var total float32
					for _, mass := range target.Joint {
						total += mass
					}
					if total != 1 {
						t.Fatal("joint target mass")
					}
					prepared, err := pathplan.Prepare(plan)
					if err != nil {
						t.Fatal(err)
					}
					var parts [2]string
					for i, choice := range plan.Decisions {
						if choice.Fallback != choice.Options[(FallbackMask(c)>>i)&1].Label {
							t.Fatal("source fallback mask changed")
						}
						fields, err := prepared.SourceFeatures(choice.ID)
						if err != nil {
							t.Fatal(err)
						}
						natural := strings.SplitN(choice.Intent, "intent: ", 2)[1]
						if strings.Contains(natural, "mask_") || strings.Contains(natural, "reference_") || strings.Contains(natural, "assign_") || strings.Contains(natural, "goal") {
							t.Fatal("target metadata exposed")
						}
						parts[i], err = decision.EncodeSemanticContextInput(fields, natural)
						if err != nil {
							t.Fatal(err)
						}
					}
					joint, err := jointdecision.Encode(parts)
					if err != nil {
						t.Fatal(err)
					}
					var features [jointdecision.FeatureDim]float32
					if err = jointdecision.FeaturesInto(joint, &features); err != nil {
						t.Fatal(err)
					}
					views++
					splits[Split(c)]++
				}
			}
		}
	}
	if views != 2304 || splits["train"] != 1536 || splits["calibration"] != 384 || splits["development"] != 384 || len(scales) != 5 {
		t.Fatal("frozen denominators", views, splits, scales)
	}
}
func TestFiniteEqualityTiesAndLiteralAliasIndependence(t *testing.T) {
	plan, _ := Fixture("predicate_assignment", 1, 0, "en")
	target, err := Target(plan, "predicate_assignment", 1, 0)
	if err != nil || len(target.BestMasks) < 2 || target.Marginals[0] != [2]float32{.5, .5} {
		t.Fatal("equal predicate ties lost", target, err)
	}
	for _, family := range Families {
		first, _ := Fixture(family, 0, 0, "en")
		second, _ := Fixture(family, 12, 3, "ko")
		left, err := pathplan.Prepare(first)
		if err != nil {
			t.Fatal(err)
		}
		right, err := pathplan.Prepare(second)
		if err != nil {
			t.Fatal(err)
		}
		if left.Fallback().GoooBody() == right.Fallback().GoooBody() {
			t.Fatal("literal/alias fixture invariant")
		}
		for _, choice := range first.Decisions {
			a, err := left.SourceFeatures(choice.ID)
			if err != nil {
				t.Fatal(err)
			}
			b, err := right.SourceFeatures(choice.ID)
			if err != nil || a != b {
				t.Fatal("source feature leaked literal, alias, goal or language", family, err)
			}
		}
	}
	for _, c := range []int{-1, 48} {
		if _, err := Fixture(Families[0], c, 0, "en"); err == nil {
			t.Fatal("fixture bound")
		}
	}
	for _, kind := range []string{"assignment_target", "root_order", "operand_order"} {
		if TemplateID(kind, 0, "en") == TemplateID(kind, 32, "en") || TemplateID(kind, 32, "en") == TemplateID(kind, 40, "en") {
			t.Fatal("template identifiers overlap")
		}
	}
}
