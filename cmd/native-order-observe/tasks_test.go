package main

import (
	"strings"
	"testing"
)

func TestOrderTaskFiniteReferences(t *testing.T) {
	expected := [][2][3]int64{{{-2, 2, 8}, {-3, 1, 7}}, {{-10, -6, 0}, {-7, -3, 3}}, {{6, 4, 1}, {-2, -4, -7}}, {{5, 1, 10}, {1, 1, 16}}}
	for i, task := range orderTasks() {
		for order := range 2 {
			for j, input := range []int64{-2, 0, 3} {
				if got := task.Expected(input, order); got != expected[i][order][j] {
					t.Fatalf("%s order%d input%d: %d", task.ID, order, input, got)
				}
			}
		}
		if !strings.Contains(task.source(), "value = "+task.Assignments[0]+"; value = "+task.Assignments[1]) {
			t.Fatal("source assignment order")
		}
		for language := range 2 {
			if task.intent(language, 0) == task.intent(language, 1) {
				t.Fatal("intent order collapsed")
			}
		}
	}
}
