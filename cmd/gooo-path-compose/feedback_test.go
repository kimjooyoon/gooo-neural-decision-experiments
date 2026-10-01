package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestFeedbackCLIEmitsBoundedModelJudgmentsAndKeepsPartialCases(t *testing.T) {
	raw, err := os.ReadFile("../../studies/conditional-paths-v1/cohort/ko-budget-64.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Plan  pathplan.Plan       `json:"path_plan"`
		Cases []pathplan.TestCase `json:"test_cases"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	fixture.Cases[6].Expected = 999
	dir := t.TempDir()
	plan := filepath.Join(dir, "plan.json")
	tests := filepath.Join(dir, "tests.json")
	ci := filepath.Join(dir, "ci.json")
	raw, _ = json.Marshal(fixture.Plan)
	if err = os.WriteFile(plan, raw, 0600); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(map[string]any{"schema": "gooo/typed-path-finite-tests/v1", "cases": fixture.Cases})
	if err = os.WriteFile(tests, raw, 0600); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(pathplan.CIHint{SourceSHA: strings.Repeat("a", 40), Status: "UNKNOWN"})
	if err = os.WriteFile(ci, raw, 0600); err != nil {
		t.Fatal(err)
	}
	model := "../../runs/typed-path-positioned-random-20261001/models/fp32/model.json"
	args := []string{"--plan", plan, "--tests", tests, "--model", model, "--max-attempts", "64", "--step-attempts", "8", "--feedback-rounds", "2", "--feedback-ci", ci}
	var output bytes.Buffer
	if err = run(args, &output); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	var last sessionComposeResult
	var feedbackSHA string
	rounds := 0
	seen := map[uint16]bool{}
	for {
		var row json.RawMessage
		if err = decoder.Decode(&row); err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		var header struct {
			Schema string `json:"schema"`
		}
		if err = json.Unmarshal(row, &header); err != nil {
			t.Fatal(err)
		}
		if header.Schema == "gooo/typed-path-feedback-judgment/v1" {
			var receipt pathplan.FeedbackReceipt
			if err = json.Unmarshal(row, &receipt); err != nil {
				t.Fatal(err)
			}
			rounds++
			if receipt.Round != rounds || receipt.ModelCalls != 6 || !receipt.Applied || receipt.CIIsAuthority || receipt.CI == nil || receipt.CI.Status != "UNKNOWN" || len(receipt.Judgments) != 6 {
				t.Fatal("feedback JSON-line evidence differs")
			}
			feedbackSHA = receipt.SHA
			continue
		}
		if header.Schema != "gooo/typed-path-session-compose-result/v1" {
			t.Fatal("unknown streamed schema")
		}
		if err = json.Unmarshal(row, &last); err != nil {
			t.Fatal(err)
		}
		for _, attempt := range last.Progress.NewAttempts {
			if seen[attempt.Mask] {
				t.Fatal("feedback repeated a tested path")
			}
			seen[attempt.Mask] = true
		}
	}
	if rounds != 2 || len(seen) != 64 || last.Progress.Status != "PARTIAL" || last.Progress.SelectedPassed != 6 || last.Progress.Cases != 7 || last.Progress.FeedbackPredictions != 12 || last.Progress.Selection.ModelCalls != 18 || last.Progress.LatestFeedbackSHA != feedbackSHA || !last.Progress.Exhausted {
		t.Fatal("CLI hid partial cases, additional model calls or unattempted alternatives")
	}
	for _, invalid := range [][]string{
		{"--plan", plan, "--feedback-rounds", "1"},
		{"--plan", plan, "--tests", tests, "--step-attempts", "8", "--feedback-rounds", "1"},
		{"--plan", plan, "--tests", tests, "--model", model, "--feedback-rounds", "1"},
		{"--plan", plan, "--feedback-rounds", "17"},
		{"--plan", plan, "--feedback-ci", ci},
	} {
		if err = run(invalid, &bytes.Buffer{}); err == nil {
			t.Fatal("invalid feedback flags accepted")
		}
	}
	if err = os.WriteFile(ci, []byte(`{"source_sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","status":"PASS","status":"FAIL"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var invalidOutput bytes.Buffer
	if err = run(args, &invalidOutput); err == nil || invalidOutput.Len() != 0 {
		t.Fatal("duplicate CI hint keys reached generation")
	}
}
