package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func fullFixture(t *testing.T, schema string) (map[string]any, fullBinding) {
	t.Helper()
	pin := fullBinding{compactBinding{strings.Repeat("a", 64), strings.Repeat("b", 64), schema}, jointdecision.ThreeBagFeatureVersion, jointdecision.SeparateArithmeticVersion}
	r := &pathplan.ThreeReceipt{Schema: schema, Feature: pin.Feature, Arithmetic: pin.Arithmetic, Input: "원본 complete input", InputSHA: "input", Calls: 1, PredictionValid: true}
	g := pairedGeneration{Source: "package sample\nfunc F(x int)int{return x+1}\n"}
	g.Report.Paths.Search = pathplan.SearchResult{Schema: "gooo/typed-path-tdd-search/v1", Status: "PARTIAL", Selection: pathplan.Selection{MetadataSHA256: pin.Metadata, WeightsSHA256: pin.Weights, ModelCalls: 2, ExternalCallsKnown: true, Three: r}, Attempts: []pathplan.SearchAttempt{{Mask: 1, Passed: 1, Total: 2}}}
	g.Report.Paths.Feedback = []pathplan.FeedbackReceipt{{MetadataSHA: pin.Metadata, WeightsSHA: pin.Weights, ModelCalls: 1, Three: r}}
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err = json.Unmarshal(b, &value); err != nil {
		t.Fatal(err)
	}
	paths := value["report"].(map[string]any)["body_paths"].(map[string]any)
	paths["model_context"] = map[string]any{"status": "ENCODED", "feature_version": pin.Feature, "arithmetic_version": pin.Arithmetic, "model_metadata_sha256": pin.Metadata}
	return value, pin
}

func fullBytes(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func expectFullPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("invalid observation accepted")
		}
	}()
	f()
}

func TestFullModelContextAndPredictionIdentity(t *testing.T) {
	a, ap := fullFixture(t, jointdecision.ThreeSchema)
	b, bp := fullFixture(t, jointdecision.SharedThreeSchema)
	want := validateFullGeneration(fullBytes(t, a), ap)
	if !bytes.Equal(want, validateFullGeneration(fullBytes(t, b), bp)) {
		t.Fatal("equivalent representations differ")
	}
	for _, key := range []string{"status", "feature_version", "arithmetic_version", "model_metadata_sha256"} {
		t.Run("context-"+key, func(t *testing.T) {
			g, pin := fullFixture(t, jointdecision.ThreeSchema)
			paths := g["report"].(map[string]any)["body_paths"].(map[string]any)
			paths["model_context"].(map[string]any)[key] = "wrong"
			expectFullPanic(t, func() { validateFullGeneration(fullBytes(t, g), pin) })
		})
	}
	for _, key := range []string{"feature_version", "arithmetic_version"} {
		t.Run("prediction-"+key, func(t *testing.T) {
			g, pin := fullFixture(t, jointdecision.ThreeSchema)
			paths := g["report"].(map[string]any)["body_paths"].(map[string]any)
			search := paths["search"].(map[string]any)
			selection := search["selection"].(map[string]any)
			var receiptKey string
			for k, v := range selection {
				if record, ok := v.(map[string]any); ok && record["schema"] == jointdecision.ThreeSchema {
					receiptKey = k
				}
			}
			if receiptKey == "" {
				t.Fatal("three receipt missing")
			}
			selection[receiptKey].(map[string]any)[key] = "wrong"
			expectFullPanic(t, func() { validateFullGeneration(fullBytes(t, g), pin) })
		})
	}
	b, bp = fullFixture(t, jointdecision.SharedThreeSchema)
	b["source"] = "changed Go body"
	if bytes.Equal(want, validateFullGeneration(fullBytes(t, b), bp)) {
		t.Fatal("source change normalized away")
	}
}

func TestFullStorageAndCaptureBounds(t *testing.T) {
	base := fullStorage{Available: 4 << 30, Whole: 2 << 30, Study: 500 << 20, Phase: 16 << 20}
	if !fullStorageFits(base, fullInvocationReserve) {
		t.Fatal("valid storage declined")
	}
	for _, change := range []func(*fullStorage){func(s *fullStorage) { s.Available-- }, func(s *fullStorage) { s.Phase = fullPhaseCap - fullFailureReserve }, func(s *fullStorage) { s.Study = (768 << 20) - fullFailureReserve }, func(s *fullStorage) { s.Whole = (3 << 30) - fullFailureReserve }} {
		s := base
		change(&s)
		if fullStorageFits(s, fullInvocationReserve) {
			t.Fatal("storage bound bypassed")
		}
	}
	if fullStorageFits(base, -1) || fullStorageFits(base, 1<<62) {
		t.Fatal("invalid reserve accepted")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "input")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	if string(boundedFullFile(path, 3)) != "abc" {
		t.Fatal("bounded content changed")
	}
	expectFullPanic(t, func() { boundedFullFile(path, 2) })
	if err := os.Symlink(path, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := fullDirectorySize(dir); err == nil {
		t.Fatal("symlink counted as retained evidence")
	}
	expectFullPanic(t, func() { boundedFullFile(filepath.Join(dir, "link"), 3) })
	c := &capture{maximum: 3}
	if _, err := io.Copy(c, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(c, strings.NewReader("d")); err == nil || c.data.Len() != 3 {
		t.Fatal("capture grew beyond limit")
	}
}
