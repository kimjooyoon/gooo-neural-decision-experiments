package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodydecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

func TestRunRejectsDuplicateInvalidUTF8UnknownAndOversizedPlansWithoutOutput(t *testing.T) {
	valid, err := json.Marshal(cliPlan("safe operation"))
	if err != nil {
		t.Fatal(err)
	}
	duplicate := bytes.Replace(valid, []byte(`"id":"cli-plan"`), []byte(`"id":"cli-plan","id":"other"`), 1)
	unknown := bytes.Replace(valid, []byte(`"schema":`), []byte(`"open_source":"bad","schema":`), 1)
	invalidUTF8 := append([]byte(nil), valid...)
	textOffset := bytes.Index(invalidUTF8, []byte("safe operation"))
	if textOffset < 0 {
		t.Fatal("test plan text not found")
	}
	invalidUTF8[textOffset] = 0xff
	oversized := bytes.Repeat([]byte(" "), 65_537)
	for name, data := range map[string][]byte{
		"duplicate nested key": duplicate,
		"unknown field":        unknown,
		"raw invalid UTF-8":    invalidUTF8,
		"oversized plan":       oversized,
		"trailing JSON":        append(append([]byte(nil), valid...), []byte(` {}`)...),
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			err := run(nil, bytes.NewReader(data), &output)
			if err == nil {
				t.Fatal("invalid plan was accepted")
			}
			if output.Len() != 0 {
				t.Fatalf("invalid plan wrote a partial result: %s", output.String())
			}
		})
	}
}

func TestRunStrictlyDecodesTrainingCases(t *testing.T) {
	plan, err := json.Marshal(cliPlan("training only"))
	if err != nil {
		t.Fatal(err)
	}
	for name, training := range map[string]string{
		"duplicate case key":    `[{"input":1,"input":2,"expected":{"type":"Int","int":0}}]`,
		"unknown case key":      `[{"input":1,"unexpected":2,"expected":{"type":"Int","int":0}}]`,
		"trailing cases":        `[] []`,
		"missing input":         `[{"expected":{"type":"Int"}}]`,
		"null input":            `[{"input":null,"expected":{"type":"Int"}}]`,
		"missing expected":      `[{"input":1}]`,
		"null expected":         `[{"input":1,"expected":null}]`,
		"missing expected type": `[{"input":1,"expected":{}}]`,
		"null expected scalar":  `[{"input":1,"expected":{"type":"Int","int":null}}]`,
		"oversized cases":       strings.Repeat(" ", 1_048_577),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "training.json")
			if err := os.WriteFile(path, []byte(training), 0o600); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			err := run([]string{"--training-cases", path}, bytes.NewReader(plan), &output)
			if err == nil {
				t.Fatal("invalid training file was accepted")
			}
			if output.Len() != 0 {
				t.Fatalf("invalid training file wrote output: %s", output.String())
			}
		})
	}
}

func TestRunSeparatesFallbackProposalFromTrainingSearchAndDoesNotExecuteText(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "must-not-exist")
	planValue := cliPlan("`); os.WriteFile(" + strconv.Quote(marker) + ", []byte(`ran`), 0600); //")
	planBytes, err := json.Marshal(planValue)
	if err != nil {
		t.Fatal(err)
	}
	trainingRaw := []byte(`[{"input":1,"expected":{"type":"Int","int":0}},{"input":2,"expected":{"type":"Int","int":1}}]`)
	trainingPath := filepath.Join(t.TempDir(), "training.json")
	if err := os.WriteFile(trainingPath, trainingRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"--training-cases", trainingPath, "--attempt-limit=8"}, bytes.NewReader(planBytes), &output); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Schema         string                     `json:"schema"`
		PlanSHA256     string                     `json:"plan_sha256"`
		Selection      bodydecision.Selection     `json:"initial_selection"`
		Search         *bodydecision.SearchResult `json:"training_search"`
		TrainingSHA256 string                     `json:"training_cases_sha256"`
		Choices        map[string]string          `json:"emitted_choices"`
		GoooSource     string                     `json:"gooo_source"`
		GoSource       string                     `json:"go_source"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Schema != "gooo/typed-body-composition-result/v1" || result.Selection.Choices["op"] != "multiply" {
		t.Fatalf("unexpected schema or initial fallback proposal: %+v", result)
	}
	if result.Selection.ModelCalls != 0 || result.Selection.ProviderCalls != 0 {
		t.Fatalf("model-free run claims inference/provider calls: %+v", result.Selection)
	}
	if result.Search == nil || result.Search.Choices["op"] != "subtract" || result.Choices["op"] != "subtract" {
		t.Fatalf("training-tested final choice was not kept separate from initial selection: %+v", result)
	}
	if result.Search.Final.Total != 2 || result.Search.Final.Correct != 2 || result.Search.Initial.Total != 2 {
		t.Fatalf("search denominator does not equal the supplied training set: %+v", result.Search)
	}
	wantTrainingDigest := sha256.Sum256(trainingRaw)
	if result.TrainingSHA256 != hex.EncodeToString(wantTrainingDigest[:]) {
		t.Fatalf("training file digest mismatch: got=%s want=%x", result.TrainingSHA256, wantTrainingDigest)
	}
	if strings.Contains(result.GoSource, marker) || strings.Contains(result.GoooSource, marker) {
		t.Fatalf("hole text crossed the closed source-generation boundary:\n%s\n%s", result.GoSource, result.GoooSource)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("CLI executed model-provided/hole text or created unexpected file: stat error %v", err)
	}
}

func TestRunRejectsSeedWithoutModel(t *testing.T) {
	plan, err := json.Marshal(cliPlan("seed test"))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = run([]string{"--sample-seed=fixed"}, bytes.NewReader(plan), &output)
	if err == nil || !strings.Contains(err.Error(), "sampling requires") {
		t.Fatalf("seed without model error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid seeded run wrote output: %s", output.String())
	}
}

func cliPlan(text string) bodyplan.Plan {
	return bodyplan.Plan{
		Schema: bodyplan.Schema, ID: "cli-plan", Name: "Compose", ResultType: decision.TypeInt,
		Expressions: []bodyplan.Expr{
			{Kind: bodyplan.ExprInput, Name: "input"},
			{Kind: bodyplan.ExprInt, Int: 1},
			{Kind: bodyplan.ExprHole, Left: 0, Right: 1, HoleID: "op", Text: text, Allowed: []string{"add", "subtract", "multiply"}, Fallback: "multiply"},
		},
		Statements: []bodyplan.Stmt{{Kind: bodyplan.StmtReturn, Expr: 2}}, Root: []int{0},
	}
}
