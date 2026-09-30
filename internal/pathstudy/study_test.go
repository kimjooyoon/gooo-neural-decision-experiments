package pathstudy

import (
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestAllStructuralProgramsAgainstIndependentOracle(t *testing.T) {
	for _, family := range Families {
		for config := 0; config < 64; config++ {
			for _, reverse := range []bool{false, true} {
				text, _, err := Instruction(family, reverse, config, 0)
				if err != nil {
					t.Fatal(err)
				}
				plan, err := Fixture(family, config, text)
				if err != nil {
					t.Fatal(err)
				}
				program, err := pathplan.Compile(plan, map[string]string{"structure": GoldLabel(family, reverse)})
				if err != nil {
					t.Fatalf("%s/%d/%t: %v", family, config, reverse, err)
				}
				for _, input := range Inputs(config) {
					want, err := Oracle(family, reverse, config, input)
					if err != nil {
						t.Fatal(err)
					}
					got, err := program.Evaluate(input)
					if err != nil || got.Int != want {
						t.Fatalf("%s/%d/%t input=%d: got=%v err=%v want=%d", family, config, reverse, input, got, err, want)
					}
				}
				for _, view := range []string{"plain", "gooo", "prov"} {
					for template := 0; template < 8; template++ {
						plain, _, err := Instruction(family, reverse, config, template)
						if err != nil {
							t.Fatal(err)
						}
						text, err := View(plain, family, view, config)
						if err != nil || len(text) > 512 {
							t.Fatalf("%s/%d/%t view=%s template=%d bytes=%d err=%v", family, config, reverse, view, template, len(text), err)
						}
					}
				}
			}
		}
	}
}

func TestDeterministicPathSelectionAndStructuralRejection(t *testing.T) {
	for _, family := range Families {
		plan, err := Fixture(family, 51, "Preserve the declared path.")
		if err != nil {
			t.Fatal(err)
		}
		selection, _, err := pathplan.Choose(plan, nil, "")
		if err != nil || selection.ModelCalls != 0 || !selection.ExternalCallsKnown || selection.Choices["structure"] != plan.Decisions[0].Fallback {
			t.Fatalf("offline: %+v %v", selection, err)
		}
		if _, _, err := pathplan.Choose(plan, nil, "seed"); err == nil {
			t.Fatal("offline sampling accepted")
		}
		plan.Decisions[0].Options[1].Label = "add"
		if _, err := pathplan.Validate(plan); err == nil {
			t.Fatal("operation label accepted as structural choice")
		}
	}
	plan, err := Fixture(pathplan.LocalReference, 51, "Choose a local.")
	if err != nil {
		t.Fatal(err)
	}
	plan.Decisions[0].Options[1].Name = "outside_scope"
	if _, err := pathplan.Validate(plan); err == nil {
		t.Fatal("undeclared local accepted")
	}
	plan, err = Fixture(pathplan.RootOrder, 51, "Choose an order.")
	if err != nil {
		t.Fatal(err)
	}
	plan.Decisions[0].Options[1].Order = []int{0, 1, 2, 2, 4}
	if _, err := pathplan.Validate(plan); err == nil {
		t.Fatal("duplicate statement accepted")
	}
}
