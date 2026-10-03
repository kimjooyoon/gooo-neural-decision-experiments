package main

import "testing"

func TestAuthoredOrderReferences(t *testing.T) {
	want := map[string][2][3]int64{
		"add-multiply":      {{-2, 2, 8}, {-3, 1, 7}},
		"subtract-multiply": {{-10, -6, 0}, {-7, -3, 3}},
		"negate-add":        {{6, 4, 1}, {-2, -4, -7}},
		"square-add":        {{5, 1, 10}, {1, 1, 16}},
	}
	all := tasks()
	if len(all) != 8 {
		t.Fatal("four families in two languages required")
	}
	for _, task := range all {
		if task.Expected != want[task.ID] || task.Clauses[0] == task.Clauses[1] || task.Bodies[0] == task.Bodies[1] {
			t.Fatal("authored reference differs", task.ID, task.Language)
		}
	}
	if remainingCollision()["sketches_equal"] != true {
		t.Fatal("known directed-edge collision omitted")
	}
}
