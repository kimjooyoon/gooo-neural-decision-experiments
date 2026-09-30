package main

import "testing"

func TestReservedProbeAndInputPartitions(t *testing.T) {
	rows, err := probe()
	if err != nil || len(rows) != 1920 {
		t.Fatalf("probe: %d %v", len(rows), err)
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.ID] || row.Configuration < 64 || row.Configuration >= 96 || row.Text == "" {
			t.Fatal("reserved probe identity or group is invalid")
		}
		seen[row.ID] = true
		training, holdout, err := finiteCases(row)
		if err != nil || len(training) != 4 || len(holdout) != 8 {
			t.Fatalf("partition: %s %d %d %v", row.ID, len(training), len(holdout), err)
		}
		for _, test := range training {
			for _, input := range holdout {
				if test.Input == input {
					t.Fatal("training and unseen-input partitions overlap")
				}
			}
		}
	}
}
