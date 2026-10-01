package main

import (
	"testing"
)

func TestCompoundFrozenCohortRetainsFourPathSemantics(t *testing.T) {
	t.Chdir("../..")
	rows, err := compoundRows()
	if err != nil || len(rows) != 72 {
		t.Fatal("frozen authored compound cohort differs", err)
	}
	for _, r := range rows {
		if _, err := inspectCompound(nativeResult{}, r, familyArm{}); err == nil {
			t.Fatal("unverified native result accepted")
		}
	}
}
