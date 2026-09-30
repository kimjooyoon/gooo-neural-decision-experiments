package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func validateDirectEvidence(raw []byte) error {
	var report struct {
		Schema      string `json:"schema"`
		Runner      string `json:"runner_revision"`
		Native      string `json:"native_revision"`
		Calls       int    `json:"native_calls"`
		Predictions int    `json:"local_model_predictions"`
		External    int    `json:"external_calls"`
		Known       bool   `json:"external_calls_known"`
		Cases       int    `json:"independent_cases"`
		Packages    int    `json:"independent_package_passes"`
		Tests       int    `json:"independent_parity_test_passes"`
		Steps       int    `json:"model_training_steps"`
		Cells       []struct {
			ID           string                `json:"id"`
			Arm          string                `json:"arm"`
			Budget       int                   `json:"max_attempts"`
			SourceSHA    string                `json:"gooo_input_sha256"`
			DocumentSHA  string                `json:"input_document_sha256"`
			ReceiptSHA   string                `json:"native_receipt_sha256"`
			GeneratedSHA string                `json:"generated_go_sha256"`
			Search       pathplan.SearchResult `json:"search"`
			Passed       int                   `json:"independent_gold_cases_passed"`
			Total        int                   `json:"independent_gold_cases_total"`
		} `json:"cells"`
	}
	if json.Unmarshal(raw, &report) != nil || report.Schema != "gooo/native-typed-path-observation/v1" ||
		report.Runner != "b4d4ebe08335ab96427e2add3b012126d4687a3f" || report.Native != "d21ce275ec0832e38ad4962ad03f4546c1b46e9f" ||
		report.Calls != 32 || report.Predictions != 72 || report.External != 0 || !report.Known || report.Cases != 320 ||
		report.Packages != 32 || report.Tests != 32 || report.Steps != 0 || len(report.Cells) != 32 {
		return errors.New("native direct evidence accounting is invalid")
	}
	root := "runs/native-typed-path-compound-20261001"
	seen := map[string]bool{}
	passedByID := map[string]int{}
	totalGold, fullBudgetGold, calls := 0, 0, 0
	for _, row := range report.Cells {
		if seen[row.ID] || row.Total != 10 || row.Passed < 0 || row.Passed > 10 || row.Budget != 4 && row.Budget != 8 || row.Search.DeclaredCombinations != 8 || row.Search.Unattempted != 8-len(row.Search.Attempts) {
			return errors.New("invalid direct cell")
		}
		seen[row.ID] = true
		bindings := map[string]string{"input.gooo": row.SourceSHA, "plan.json": row.DocumentSHA, "native-stdout.json": row.ReceiptSHA, "generated.go.txt": row.GeneratedSHA}
		for name, pin := range bindings {
			data, err := read(filepath.Join(root, row.ID, name))
			if err != nil || digest(data) != pin {
				return errors.New("direct capture hash mismatch")
			}
		}
		calls += row.Search.Selection.ModelCalls
		if row.Search.Selection.ExternalCalls != 0 || !row.Search.Selection.ExternalCallsKnown {
			return errors.New("direct capture claims external execution")
		}
		if row.Arm == "offline" {
			if row.Search.Selection.ModelCalls != 0 {
				return errors.New("offline cell has predictions")
			}
		} else if row.Search.Selection.ModelCalls != 3 {
			return errors.New("native prediction count differs")
		}
		capture, err := read(filepath.Join(root, row.ID, "native-stdout.json"))
		if err != nil {
			return err
		}
		var payload struct {
			Source string `json:"source"`
			Report struct {
				Compiler string `json:"compiler_source_sha"`
				Paths    struct {
					Search pathplan.SearchResult `json:"search"`
				} `json:"body_paths"`
			} `json:"report"`
		}
		if json.Unmarshal(capture, &payload) != nil || payload.Report.Compiler != report.Native || digest([]byte(payload.Source)) != row.GeneratedSHA {
			return errors.New("direct captured payload mismatch")
		}
		left, _ := json.Marshal(row.Search)
		right, _ := json.Marshal(payload.Report.Paths.Search)
		if !bytes.Equal(left, right) {
			return errors.New("direct search differs from native capture")
		}
		passedByID[row.ID] = row.Passed
		totalGold += row.Passed
		if row.Budget == 8 {
			fullBudgetGold += row.Passed
			if row.Passed != 10 {
				return errors.New("full-budget observation differs")
			}
		}
	}
	if totalGold != 226 || fullBudgetGold != 160 || calls != 72 {
		return errors.New("direct gold/prediction totals differ")
	}
	data, err := read(filepath.Join(root, "independent-go-tests.jsonl"))
	if err != nil {
		return err
	}
	observed := map[string]map[int64]bool{}
	passed := map[string]int{}
	cases, packages, tests := 0, 0, 0
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 8192), 1<<20)
	for scanner.Scan() {
		var event struct {
			Action string
			Test   string
			Output string
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			return errors.New("invalid Go event")
		}
		if event.Action == "pass" {
			if event.Test == "" {
				packages++
			} else if event.Test == "TestIndependentObservation" {
				tests++
			}
		}
		index := strings.Index(event.Output, "GOOO_OBSERVATION ")
		if index < 0 {
			continue
		}
		var item struct {
			Cell     string `json:"cell"`
			Input    int64  `json:"input"`
			Expected int64  `json:"expected"`
			Actual   int64  `json:"actual"`
			Passed   bool   `json:"passed"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(event.Output[index+len("GOOO_OBSERVATION "):])), &item) != nil || !seen[item.Cell] {
			return errors.New("invalid direct observation")
		}
		gold := 5*item.Input + 2
		if item.Input < 0 {
			gold -= 3
		} else {
			gold += 3
		}
		if item.Expected != gold || item.Passed != (item.Actual == gold) {
			return errors.New("incorrect arithmetic observation")
		}
		if observed[item.Cell] == nil {
			observed[item.Cell] = map[int64]bool{}
		}
		if observed[item.Cell][item.Input] {
			return errors.New("duplicate direct observation")
		}
		observed[item.Cell][item.Input] = true
		if item.Passed {
			passed[item.Cell]++
		}
		cases++
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	if cases != 320 || packages != 32 || tests != 32 {
		return errors.New("independent Go event count differs")
	}
	for id, want := range passedByID {
		if len(observed[id]) != 10 || passed[id] != want {
			return errors.New("per-cell Go observation differs")
		}
	}
	return nil
}
