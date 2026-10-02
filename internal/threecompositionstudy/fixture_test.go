package threecompositionstudy

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestAuthoredThreeChoiceCohortAllMasksAndLanguages(t *testing.T) {
	views, comparisons, ties := 0, 0, 0
	splits := map[string]int{}
	for _, family := range Families {
		for config := range 24 {
			for goal := range 8 {
				var previous FiniteTarget
				for _, language := range [2]string{"en", "ko"} {
					plan, err := Fixture(family, config, goal, language)
					if err != nil {
						t.Fatal(err)
					}
					target, err := Target(plan, family, config, goal)
					if err != nil {
						t.Fatal(err)
					}
					if target.Cases != 16 || target.Passed[goal] != 16 || len(target.BestMasks) == 0 {
						t.Fatal("desired complete finite target", target)
					}
					if language == "ko" && !reflect.DeepEqual(previous, target) {
						t.Fatal("bilingual arithmetic changed")
					}
					previous = target
					if len(target.BestMasks) > 1 {
						ties++
					}
					var mass float64
					for mask, value := range target.Joint {
						mass += float64(value)
						if (value > 0) != (target.Passed[mask] == 16) {
							t.Fatal("passing-mask set forced into a single label")
						}
					}
					if math.Abs(mass-1) > 1e-6 {
						t.Fatal("joint target mass", mass)
					}
					assertCompleteInputs(t, plan, config)
					views++
					comparisons += 8 * 16
					splits[Split(config)]++
				}
			}
		}
	}
	if views != 3072 || comparisons != 393216 || splits["train"] != 2048 ||
		splits["calibration"] != 512 || splits["development"] != 512 || ties == 0 {
		t.Fatal("preregistered denominators", views, comparisons, splits, ties)
	}
	t.Logf("authored ways=8 views=%d ordered typed/oracle comparisons=%d tied views=%d", views, comparisons, ties)
}

func assertCompleteInputs(t *testing.T, plan pathplan.Plan, config int) {
	t.Helper()
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		t.Fatal(err)
	}
	var parts [3]string
	for coordinate, choice := range plan.Decisions {
		if choice.Fallback != choice.Options[(FallbackMask(config)>>coordinate)&1].Label {
			t.Fatal("source fallback mask changed")
		}
		fields, err := prepared.SourceFeatures(choice.ID)
		if err != nil {
			t.Fatal(err)
		}
		_, natural, found := strings.Cut(choice.Intent, "intent: ")
		if !found || natural == "" {
			t.Fatal("complete natural instruction missing")
		}
		for _, forbidden := range []string{"mask_", "reference_", "assign_", "goal", "layout_", "schedule_"} {
			if strings.Contains(natural, forbidden) {
				t.Fatal("target metadata exposed")
			}
		}
		parts[coordinate], err = decision.EncodeSemanticContextInput(fields, natural)
		if err != nil {
			t.Fatal(err)
		}
	}
	text, err := jointdecision.EncodeThree(parts)
	if err != nil {
		t.Fatal(err)
	}
	var features [jointdecision.ThreeFeatureDim]float32
	if err := jointdecision.FeaturesIntoThree(text, &features); err != nil {
		t.Fatal(err)
	}
	decoded, err := jointdecision.ThreeParts(text)
	if err != nil || decoded != parts {
		t.Fatal("ordered complete context changed")
	}
}

func TestIndependentHandCalculatedCases(t *testing.T) {
	// Config 0, input 3: delta=-6, scale=3, threshold=-7, locals=(-3,9).
	cases := []struct {
		family string
		mask   int
		want   int64
	}{
		{"chained_operands", 0, -3}, {"chained_operands", 7, 15},
		{"successive_assignments_reference", 0, 9}, {"successive_assignments_reference", 7, -18},
		{"branch_assignment_reference", 0, -30}, {"branch_assignment_reference", 7, -3},
		{"assignment_reference_operand", 0, -3}, {"assignment_reference_operand", 7, 0},
		{"schedule_assignment_operand", 0, 0}, {"schedule_assignment_operand", 7, 3},
		{"nested_branches_reference", 0, -3}, {"nested_branches_reference", 7, 6},
		{"comparison_branch_assignment", 0, -18}, {"comparison_branch_assignment", 5, 0},
		{"boolean_reference_branch_operand", 0, 15}, {"boolean_reference_branch_operand", 7, -15},
	}
	for _, test := range cases {
		got, err := Oracle(test.family, 0, test.mask, 3)
		if err != nil || got != test.want {
			t.Fatalf("%s mask=%d got=%d want=%d error=%v", test.family, test.mask, got, test.want, err)
		}
	}
}

func TestTiesBoundariesAndSourceFactIndependence(t *testing.T) {
	plan, _ := Fixture("comparison_branch_assignment", 1, 0, "en")
	target, err := Target(plan, "comparison_branch_assignment", 1, 0)
	if err != nil || len(target.BestMasks) < 2 || target.Marginals[0] != [2]float32{.5, .5} {
		t.Fatal("equality comparison tie lost", target, err)
	}
	for _, family := range Families {
		first, _ := Fixture(family, 0, 0, "en")
		second, _ := Fixture(family, 8, 7, "ko")
		left, err := pathplan.Prepare(first)
		if err != nil {
			t.Fatal(err)
		}
		right, err := pathplan.Prepare(second)
		if err != nil {
			t.Fatal(err)
		}
		if left.Fallback().GoooBody() == right.Fallback().GoooBody() {
			t.Fatal("literal/alias fixtures unexpectedly equal")
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
	for _, config := range []int{-1, 24} {
		if _, err := Fixture(Families[0], config, 0, "en"); err == nil {
			t.Fatal("fixture bound")
		}
	}
	for _, goal := range []int{-1, 8} {
		if _, err := Fixture(Families[0], 0, goal, "en"); err == nil {
			t.Fatal("goal bound")
		}
	}
	if _, err := Fixture("unknown", 0, 0, "en"); err == nil {
		t.Fatal("unknown family")
	}
	if _, err := Fixture(Families[0], 0, 0, "fr"); err == nil {
		t.Fatal("unsupported language")
	}
	if _, err := Choices(plan, 8); err == nil {
		t.Fatal("mask bound")
	}
	for _, kind := range []string{pathplan.LocalReference, pathplan.OperandOrder, pathplan.BranchLayout} {
		if TemplateID(kind, 0, "en") == TemplateID(kind, 16, "en") ||
			TemplateID(kind, 16, "en") == TemplateID(kind, 20, "en") {
			t.Fatal("split template identifiers overlap")
		}
	}
}
