package main

import (
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"testing"
)

func TestFrozenProtocolAmendmentAndEverySourceFixture(t *testing.T) {
	t.Chdir("../..")
	pins, err := frozenDocuments()
	if err != nil || len(pins) != 2 {
		t.Fatal("both documents must be frozen", err)
	}
	for _, family := range jointcompositionstudy.Families {
		doc, source, target, err := fixture(family, 40, 3, "ko")
		if err != nil || len(source) == 0 || len(doc.Cases) != 16 || target.Cases != 16 || len(doc.Plan.Decisions) != 2 {
			t.Fatal(family, err)
		}
	}
}
