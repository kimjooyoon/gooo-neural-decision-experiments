package main

import "testing"

func TestArithmeticReferencesRetainOrderAndConstants(t *testing.T) {
	for _, c := range []struct {
		family           string
		fresh            bool
		forward, reverse int64
	}{
		{"add-multiply", false, 8, 7}, {"add-multiply", true, 15, 11},
		{"subtract-multiply", false, 0, 3}, {"subtract-multiply", true, 8, 11},
		{"negate-add", false, 1, -7}, {"negate-add", true, -1, -5},
		{"square-add", false, 10, 16}, {"square-add", true, 12, 36},
	} {
		v := task{Family: c.family, NewConstants: c.fresh}
		if expected(v, 3) != c.forward {
			t.Fatal("forward arithmetic differs", c)
		}
		v.WantedOrder = 1
		if expected(v, 3) != c.reverse {
			t.Fatal("reverse arithmetic differs", c)
		}
	}
}
