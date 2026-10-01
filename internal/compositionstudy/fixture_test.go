package compositionstudy

import (
	"math"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestEveryFreshCombinationAgainstIndependentOracle(t *testing.T) {
	groups, views, pairs := 0, 0, 0
	splits := map[string]int{}
	for _, family := range Families {
		for config := 0; config < 48; config++ {
			for desired := 0; desired < 4; desired++ {
				var previous FiniteTarget
				for _, language := range [2]string{"en", "ko"} {
					plan, err := Fixture(family, config, desired, language)
					if err != nil {
						t.Fatal(err)
					}
					target, err := Target(plan, family, config, desired)
					if err != nil {
						t.Fatal(err)
					}
					if target.Cases != 16 || target.Passed[desired] != 16 {
						t.Fatal("full desired contract denominator")
					}
					for i := range target.Marginals {
						if target.Marginals[i][0]+target.Marginals[i][1] != 1 {
							t.Fatal("finite target normalization")
						}
					}
					if language == "ko" {
						for mask := range target.Passed {
							if target.Passed[mask] != previous.Passed[mask] {
								t.Fatal("bilingual oracle changed")
							}
						}
					}
					previous = target
					prepared, err := pathplan.Prepare(plan)
					if err != nil {
						t.Fatal(err)
					}
					for i, choice := range plan.Decisions {
						if choice.Fallback != choice.Options[(config%4>>i)&1].Label {
							t.Fatal("source fallback mask changed")
						}
						fields, err := prepared.SourceFeatures(choice.ID)
						if err != nil {
							t.Fatal(err)
						}
						intent := choice.Intent[strings.LastIndex(choice.Intent, "intent: ")+8:]
						text, err := decision.EncodeSemanticContextInput(fields, intent)
						if err != nil || len(text) > 512 || !strings.HasSuffix(text, intent) {
							t.Fatalf("complete semantic input: %v", err)
						}
					}
					views++
					splits[Split(config)]++
				}
				groups++
				pairs += 2
			}
		}
	}
	if groups != 1152 || views != 2304 || pairs != 2304 || splits["train"] != 1536 || splits["calibration"] != 384 || splits["development"] != 384 {
		t.Fatal("frozen cohort counts changed")
	}
}

func TestFreshLiteralAliasAndGoalDoNotEnterSourceArray(t *testing.T) {
	for _, family := range Families {
		original, err := Fixture(family, 0, 0, "en")
		if err != nil {
			t.Fatal(err)
		}
		renamed, err := Fixture(family, 12, 3, "ko")
		if err != nil {
			t.Fatal(err)
		}
		left, err := pathplan.Prepare(original)
		if err != nil {
			t.Fatal(err)
		}
		right, err := pathplan.Prepare(renamed)
		if err != nil {
			t.Fatal(err)
		}
		if left.Fallback().GoooBody() == right.Fallback().GoooBody() {
			t.Fatal("fixture names/literals did not change")
		}
		for _, choice := range original.Decisions {
			a, err := left.SourceFeatures(choice.ID)
			if err != nil {
				t.Fatal(err)
			}
			b, err := right.SourceFeatures(choice.ID)
			if err != nil || a != b {
				t.Fatalf("source array leaked spelling, literal, goal or language for %s: %v", family, err)
			}
		}
	}
}

func TestOracleWrapAndFiniteTies(t *testing.T) {
	// config=6 gives delta zero. The forward reference/subtraction path
	// returns x*3; ordinary signed wrap at MaxInt64 is intentional.
	x := int64(math.MaxInt64)
	value, err := Oracle("reference_operand", 6, 0, x)
	want := x * 3
	if err != nil || value != want {
		t.Fatalf("wrap: %d, %v", value, err)
	}
	plan, err := Fixture("predicate_branch", 1, 0, "en")
	if err != nil {
		t.Fatal(err)
	}
	target, err := Target(plan, "predicate_branch", 1, 0)
	if err != nil || len(target.BestMasks) < 2 || target.Marginals[0] != [2]float32{0.5, 0.5} {
		t.Fatalf("equality operand tie was lost: %#v, %v", target, err)
	}
	cases, err := Cases("reference_operand", 6, 0)
	if err != nil || len(cases) != 16 || cases[14].Input != 0 || cases[15].Input != 0 {
		t.Fatal("ordered duplicate inputs discarded")
	}
}

func TestFrozenTemplatesAndBounds(t *testing.T) {
	for _, language := range [2]string{"en", "ko"} {
		a, b, c := TemplateID("branch_layout", 0, language), TemplateID("branch_layout", 32, language), TemplateID("branch_layout", 40, language)
		if a == b || a == c || b == c {
			t.Fatal("template identifiers overlap splits")
		}
	}
	for _, config := range []int{-1, 48} {
		if _, err := Fixture("reference_operand", config, 0, "en"); err == nil {
			t.Fatal("unbounded fixture")
		}
		if _, err := Oracle("reference_operand", config, 0, 1); err == nil {
			t.Fatal("unbounded oracle")
		}
	}
	if _, err := Fixture("unknown", 0, 0, "en"); err == nil {
		t.Fatal("unknown family")
	}
	if _, err := Fixture("reference_operand", 0, 4, "en"); err == nil {
		t.Fatal("unknown mask")
	}
	if _, err := Fixture("reference_operand", 0, 0, "ja"); err == nil {
		t.Fatal("unknown language")
	}
}
