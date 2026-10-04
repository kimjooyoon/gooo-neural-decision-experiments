package main

import "fmt"

const pairedTotal = 7168

type PairedSpec struct {
	Family, Permutation, Goal, Style int
	Orientation                      uint16
	Language, Split                  string
}

func (s PairedSpec) ID() string {
	return fmt.Sprintf("f%d-p%d-m%d-g%d-w%d-%s", s.Family, s.Permutation, s.Orientation, s.Goal, s.Style, s.Language)
}

type PairedRow struct {
	Row
	Goal        int    `json:"intent_goal"`
	Style       int    `json:"wording_variant"`
	Permutation int    `json:"choice_permutation"`
	Orientation uint16 `json:"choice_orientation"`
}

func pairedPlan() []PairedSpec {
	plan := make([]PairedSpec, 0, pairedTotal)
	for family := range 8 {
		part := split(family)
		if part == "test" {
			part = "test_source"
		}
		for permutation := range orders {
			for orientation := uint16(0); orientation < 8; orientation++ {
				for goal := range 8 {
					for _, language := range []string{"ko", "en"} {
						plan = append(plan, PairedSpec{family, permutation, goal, 0, orientation, language, part})
					}
				}
			}
		}
	}
	for _, family := range []int{0, 3, 6, 7} {
		part := "test_wording_seen"
		if family >= 6 {
			part = "test_wording_new"
		}
		for _, permutation := range []int{0, 4} {
			for _, orientation := range []uint16{0, 1, 6, 7} {
				for goal := range 8 {
					for style := 1; style <= 2; style++ {
						for _, language := range []string{"ko", "en"} {
							plan = append(plan, PairedSpec{family, permutation, goal, style, orientation, language, part})
						}
					}
				}
			}
		}
	}
	return plan
}
