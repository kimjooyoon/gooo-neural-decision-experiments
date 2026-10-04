package main

import "testing"

func TestNamedChoicesRetainSourceOrderAndOrientation(t *testing.T) {
	choices := []sourceChoice{
		{ID: "role-a", Field: "message", Intent: "제목", First: "a0", Second: "a1"},
		{ID: "role-b", Field: "flag", Intent: "state", First: "b0", Second: "b1"},
		{ID: "role-c", Field: "context", Intent: "사유", First: "c0", Second: "c1"},
	}
	p := [3][2]float64{{.2, .8}, {.7, .3}, {.4, .6}}
	for mask := uint16(0); mask < 8; mask++ {
		got, e := describeChoices(choices, p, mask)
		if e != nil {
			t.Fatal(e)
		}
		for i := range got {
			want := choices[i].First
			if mask&(1<<i) != 0 {
				want = choices[i].Second
			}
			if got[i].sourceChoice != choices[i] || got[i].Probabilities != p[i] || got[i].Proposed != want {
				t.Fatalf("mask%d choice%d: %+v", mask, i, got[i])
			}
		}
	}
}

func TestNamedChoicesRequireCompleteDeclaredShape(t *testing.T) {
	for _, n := range []int{0, 2, 4} {
		if _, e := describeChoices(make([]sourceChoice, n), [3][2]float64{}, 0); e == nil {
			t.Fatalf("accepted%dchoices", n)
		}
	}
	if _, e := describeChoices(make([]sourceChoice, 3), [3][2]float64{}, 8); e == nil {
		t.Fatal("accepted mask outside three choices")
	}
}
