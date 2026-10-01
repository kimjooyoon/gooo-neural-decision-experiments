package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

func TestIncrementalCLIBindsInitializationAndDeterministicPartialSteps(t *testing.T) {
	plan, err := pathstudy.Fixture(pathplan.BranchLayout, 51, "Swap the branches.")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := pathstudy.Oracle(pathplan.BranchLayout, true, 51, 0)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	testsFile := filepath.Join(dir, "tests.json")
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(map[string]any{"schema": "gooo/typed-path-finite-tests/v1", "cases": []pathplan.TestCase{{Input: 0, Expected: expected}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testsFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--plan", planFile, "--tests", testsFile, "--max-attempts", "2", "--step-attempts", "1"}
	var first, second bytes.Buffer
	if err := run(args, &first); err != nil {
		t.Fatal(err)
	}
	if err := run(args, &second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("model-free incremental replay differs")
	}
	decoder := json.NewDecoder(&first)
	var rows []sessionComposeResult
	for {
		var row sessionComposeResult
		err := decoder.Decode(&row)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 3 || rows[0].GoSource != "" || rows[0].Progress.Attempted != 0 || !rows[0].Progress.Initialized {
		t.Fatal("initialization executed or emitted a candidate")
	}
	seen := map[uint16]bool{}
	previous := ""
	for index, row := range rows {
		if row.Schema != "gooo/typed-path-session-compose-result/v1" || row.Progress.Sequence != index+1 || row.Progress.PreviousSHA != previous || row.Progress.Selection.ModelCalls != 0 || row.Progress.PredictionsThisAdvance != 0 {
			t.Fatal("stream observation accounting differs")
		}
		previous = row.Progress.SHA
		for _, attempt := range row.Progress.NewAttempts {
			if seen[attempt.Mask] {
				t.Fatal("incremental CLI repeated a path")
			}
			seen[attempt.Mask] = true
		}
	}
	if rows[1].Progress.Status != "PARTIAL" || rows[1].GoSource == "" || rows[2].Progress.Status != "TRAINING_COMPLETE" || rows[2].Progress.Attempted != 2 || rows[2].Progress.SelectedPassed != 1 {
		t.Fatal("stream lost partial or completed outcomes")
	}
	for _, invalid := range [][]string{{"--plan", planFile, "--step-attempts", "1"}, {"--plan", planFile, "--tests", testsFile, "--step-attempts", "65"}, {"--plan", planFile, "--tests", testsFile, "--step-attempts", "-1"}} {
		if err := run(invalid, &bytes.Buffer{}); err == nil {
			t.Fatal("invalid incremental CLI contract accepted")
		}
	}
}
