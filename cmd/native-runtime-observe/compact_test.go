package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func comparisonFixture(t *testing.T, schema string) (pairedGeneration, compactBinding) {
	t.Helper()
	b := compactBinding{Metadata: strings.Repeat("a", 64), Weights: strings.Repeat("b", 64), Schema: schema}
	var g pairedGeneration
	g.Source = "package sample\nfunc F(x int) int { return x+1 }\n"
	g.Report.Paths.Search = pathplan.SearchResult{Schema: "gooo/typed-path-tdd-search/v1", Status: "PARTIAL",
		Selection: pathplan.Selection{MetadataSHA256: b.Metadata, WeightsSHA256: b.Weights, ModelCalls: 2, ExternalCallsKnown: true,
			Three: &pathplan.ThreeReceipt{Schema: schema, Input: "complete original input", InputSHA: "original", Calls: 1, PredictionValid: true}},
		Attempts: []pathplan.SearchAttempt{{Mask: 1, Passed: 1, Total: 2}}}
	g.Report.Paths.Feedback = []pathplan.FeedbackReceipt{{MetadataSHA: b.Metadata, WeightsSHA: b.Weights, SHA: "receipt-chain",
		PreviousSHA: "previous", FromProgressSHA: "progress", Round: 1, Attempted: 1, Passed: 1, Cases: 2, ModelCalls: 1, CumulativeCalls: 2,
		Three: &pathplan.ThreeReceipt{Schema: schema, Input: "complete feedback input", InputSHA: "feedback", Calls: 1, PredictionValid: true}}}
	return g, b
}

func encodeComparison(t *testing.T, g pairedGeneration) []byte {
	t.Helper()
	b, e := json.Marshal(g)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestCompactComparisonExcludesOnlyValidatedIdentityAndTiming(t *testing.T) {
	a, ap := comparisonFixture(t, jointdecision.ThreeSchema)
	b, bp := comparisonFixture(t, jointdecision.SharedThreeSchema)
	bp.Metadata, bp.Weights = strings.Repeat("c", 64), strings.Repeat("d", 64)
	b.Report.Paths.Search.Selection.MetadataSHA256, b.Report.Paths.Search.Selection.WeightsSHA256 = bp.Metadata, bp.Weights
	b.Report.Paths.Search.Selection.Three.PredictNS = 999
	b.Report.Paths.Feedback[0].MetadataSHA, b.Report.Paths.Feedback[0].WeightsSHA = bp.Metadata, bp.Weights
	b.Report.Paths.Feedback[0].SHA, b.Report.Paths.Feedback[0].PreviousSHA, b.Report.Paths.Feedback[0].FromProgressSHA = "new-chain", "new-previous", "new-progress"
	b.Report.Paths.Feedback[0].Three.PredictNS = 333
	want := normalizeGeneration(encodeComparison(t, a), ap)
	if !bytes.Equal(want, normalizeGeneration(encodeComparison(t, b), bp)) {
		t.Fatal("allowed identities/timing affect semantic equality")
	}
	for _, mutation := range []func(*pairedGeneration){
		func(g *pairedGeneration) { g.Source += "// different body\n" },
		func(g *pairedGeneration) { g.Report.Paths.Search.Attempts[0].Mask = 2 },
		func(g *pairedGeneration) { g.Report.Paths.Search.Selection.Three.Input += "lost parity" },
		func(g *pairedGeneration) { g.Report.Paths.Search.Selection.Three.Probabilities[0] = .5 },
		func(g *pairedGeneration) { g.Report.Paths.Search.Selection.ModelCalls++ },
		func(g *pairedGeneration) { g.Report.Paths.Feedback[0].Three.Input += "changed observation" },
		func(g *pairedGeneration) { g.Report.Paths.Feedback[0].Passed++ },
	} {
		fresh, pin := comparisonFixture(t, jointdecision.ThreeSchema)
		mutation(&fresh)
		if bytes.Equal(want, normalizeGeneration(encodeComparison(t, fresh), pin)) {
			t.Fatal("semantic change discarded")
		}
	}
}

func TestCompactComparisonRejectsWrongArtifactAndSeed(t *testing.T) {
	for _, kind := range []string{"metadata", "weights", "schema", "seed", "feedback"} {
		t.Run(kind, func(t *testing.T) {
			g, pin := comparisonFixture(t, jointdecision.ThreeSchema)
			switch kind {
			case "metadata":
				g.Report.Paths.Search.Selection.MetadataSHA256 = "wrong"
			case "weights":
				g.Report.Paths.Search.Selection.WeightsSHA256 = "wrong"
			case "schema":
				g.Report.Paths.Search.Selection.Three.Schema = "wrong"
			case "seed":
				g.Report.Paths.Search.Selection.SeedSHA256 = "seeded"
			case "feedback":
				g.Report.Paths.Feedback[0].MetadataSHA = "wrong"
			}
			defer func() {
				if recover() == nil {
					t.Fatal("invalid artifact accepted")
				}
			}()
			normalizeGeneration(encodeComparison(t, g), pin)
		})
	}
}

func TestCompactCaptureBoundCannotBeBypassedByCopy(t *testing.T) {
	c := &capture{maximum: 1024}
	if _, e := io.Copy(c, strings.NewReader(strings.Repeat("x", 1024))); e != nil {
		t.Fatal(e)
	}
	if _, e := io.Copy(c, strings.NewReader("x")); e == nil || c.data.Len() != 1024 {
		t.Fatal("capture grew beyond bound")
	}
	if compactPairReserve < 3*(1<<20)+(128<<10) {
		t.Fatal("generation/source/runtime plus auxiliary files not reserved")
	}
}
