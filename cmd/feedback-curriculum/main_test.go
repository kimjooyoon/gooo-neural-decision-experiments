package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
)

func TestCurriculumRetainsGroupsAndReplay(t *testing.T) {
	t.Chdir("../..")
	a, counts, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := generate()
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("curriculum is not deterministic", err)
	}
	for split, total := range map[string]int{"train": 4800, "calibration": 480, "test": 960} {
		if counts[split]["rows"] != total || counts[split]["multiple_best_finite_labels"] == 0 {
			t.Fatal("split or finite ambiguity count changed", counts)
		}
	}
	groups := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(a))
	for scanner.Scan() {
		var row feedbackstudy.Row
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		for _, group := range []string{row.InstructionID, row.ProgramID, row.TemplateID, row.ConfigurationID} {
			if groups[group] != "" && groups[group] != row.Split {
				t.Fatal("source or intention group crossed splits")
			}
			groups[group] = row.Split
		}
		if len(row.Accepted) < 1 || len(row.Accepted) > 2 || row.BestPassed > len(row.Cases) || row.FunctionalIsIntent {
			t.Fatal("finite targets were presented as general intent correctness")
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}
