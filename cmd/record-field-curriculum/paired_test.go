package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestPairedRecountComparesValuesAcrossNestedStructAndMapOrder(t *testing.T) {
	want := map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": pairedNativeCases(6, 7)}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if !sameRecountJSON(raw, want) {
		t.Fatal("equivalent cases rejected due to JSON key order")
	}
	want["cases"] = pairedNativeCases(6, 0)
	if sameRecountJSON(raw, want) {
		t.Fatal("different intended values accepted")
	}
}

func TestBaseCurriculumPreservesPublishedSourceBytes(t *testing.T) {
	z, err := zip.OpenReader("../../publication/record-field-learning-20261005/evidence.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	byName := map[string]*zip.File{}
	for _, f := range z.File {
		byName[f.Name] = f
	}
	for family := range 8 {
		for p, order := range orders {
			for mask := uint16(0); mask < 8; mask++ {
				for _, lang := range []string{"ko", "en"} {
					name := fmt.Sprintf("prepared/sources/f%d-p%d-m%d-%s.gooo.fixture", family, p, mask, lang)
					f := byName[name]
					if f == nil {
						t.Fatal("published base missing", name)
					}
					in, err := f.Open()
					if err != nil {
						t.Fatal(err)
					}
					want, err := io.ReadAll(in)
					in.Close()
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(want, fixture(family, order, mask, lang)) {
						t.Fatal("base byte contract changed", name)
					}
				}
			}
		}
	}
}

func TestPairedPlanHasBalancedGoalsAndSeparateWordingAxes(t *testing.T) {
	seen := map[string]bool{}
	counts := map[string]int{}
	goals := map[string][8]int{}
	for _, s := range pairedPlan() {
		if seen[s.ID()] {
			t.Fatal("duplicate view")
		}
		seen[s.ID()] = true
		counts[s.Split]++
		g := goals[s.Split]
		g[s.Goal]++
		goals[s.Split] = g
		if (s.Split == "train" && s.Family >= 4) || (s.Split == "calibration" && (s.Family < 4 || s.Family > 5)) || (s.Split == "test_source" && s.Family < 6) || (strings.Contains(s.Split, "wording") && s.Style == 0) {
			t.Fatal("incorrect evaluation split", s)
		}
	}
	want := map[string]int{"train": 3072, "calibration": 1536, "test_source": 1536, "test_wording_seen": 512, "test_wording_new": 512}
	if len(seen) != pairedTotal {
		t.Fatal("inventory", len(seen))
	}
	for part, n := range want {
		if counts[part] != n {
			t.Fatal(part, counts[part])
		}
		for _, g := range goals[part] {
			if g != n/8 {
				t.Fatal("unbalanced intent goals", part, goals[part])
			}
		}
	}
}

func TestPairedNativeInventoryAndFreshChangedFields(t *testing.T) {
	registered := map[string]bool{}
	for _, s := range pairedPlan() {
		registered[s.ID()] = true
	}
	seen := map[string]bool{}
	for _, s := range pairedNativePlan() {
		if !registered[s.ID()] || seen[s.ID()] {
			t.Fatal("native view outside fixed bank or duplicate", s)
		}
		seen[s.ID()] = true
		cases := pairedNativeCases(s.Family, s.Goal)
		if len(cases) != 4 {
			t.Fatal("fresh runtime inventory")
		}
		for i, c := range cases {
			input := c.Inputs["Select.input0"].(map[string]string)
			want := c.Expected["Select"].(map[string]string)
			if i >= 2 && fmt.Sprint(input) != fmt.Sprint(want) {
				t.Fatal("unchanged guard expected differently")
			}
			if i < 2 && input[names[s.Family][1]] == want[names[s.Family][1]] {
				t.Fatal("active state must change")
			}
		}
	}
	if len(seen) != 24 {
		t.Fatal("native views changed")
	}
}

func TestGoalMaskFollowsSourcePermutation(t *testing.T) {
	for _, order := range orders {
		for orientation := uint16(0); orientation < 8; orientation++ {
			for goal := range 8 {
				got := goalMask(order, orientation, goal)
				for position, role := range order {
					usesAlternate := ((got >> position) & 1) ^ ((orientation >> position) & 1)
					if usesAlternate != uint16((goal>>role)&1) {
						t.Fatal("goal changes wrong source position", order, orientation, goal)
					}
				}
			}
		}
	}
}

func TestGoalWordsAndOraclesDescribeTheActualAlternateExpression(t *testing.T) {
	for family := range 8 {
		for goal := range 8 {
			values := goalValues(family, goal, "input title", "queued", "reason")
			if goal&1 != 0 {
				want := "draft"
				if titleCopiesState(family) {
					want = "queued"
				}
				if values[names[family][0]] != want {
					t.Fatal("title alternate")
				}
			}
			if goal&4 != 0 {
				want := "deferred"
				if family >= 3 {
					want = ":acceptedreason"
				}
				if values[names[family][2]] != want {
					t.Fatal("reason alternate")
				}
			}
			base := string(goalFixture(family, orders[0], 3, "ko", 0, 0))
			body := strings.SplitN(strings.SplitN(base, " computes `", 2)[1], "` assembling", 2)[0]
			for style := range 3 {
				for _, lang := range []string{"ko", "en"} {
					source := string(goalFixture(family, orders[0], 3, lang, goal, style))
					gotBody := strings.SplitN(strings.SplitN(source, " computes `", 2)[1], "` assembling", 2)[0]
					if gotBody != body || strings.Count(source, "field_value at") != 3 || strings.Count(source, "value_case") != 5 {
						t.Fatal("intent changed candidates/body", family, goal, style)
					}
				}
			}
		}
	}
	for family := range 8 {
		for role := range 3 {
			for _, lang := range []string{"ko", "en"} {
				for _, opposite := range []bool{false, true} {
					a, b, c := goalIntent(family, role, opposite, lang, 0), goalIntent(family, role, opposite, lang, 1), goalIntent(family, role, opposite, lang, 2)
					if a == b || a == c || b == c {
						t.Fatal("wording holdout duplicated training")
					}
				}
			}
		}
	}
}
