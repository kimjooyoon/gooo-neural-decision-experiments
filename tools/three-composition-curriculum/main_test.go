package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

func exportedFixture(t *testing.T, family string) (document, []byte, export) {
	t.Helper()
	doc, source, _, err := fixture(family, 20, 7, "ko")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		t.Fatal(err)
	}
	docRaw, _ := json.Marshal(doc)
	casesRaw, _ := json.Marshal(doc.Cases)
	zero, no := 0, false
	value := export{Schema: "gooo/compiler-path-input-export/v2", Source: "sha256:" + hash(source),
		Document: "sha256:" + hash(docRaw), TestsSHA: "sha256:" + hash(casesRaw),
		Predictions: &zero, Tests: &zero, Writes: &zero, Emission: &no}
	value.Binding.Equivalent, value.Binding.Semantic = true, "sha256:"+strings.Repeat("a", 64)
	value.Context.Schema, value.Context.Status = "gooo/compiler-typed-path-context/v3", "ENCODED"
	value.Context.Feature, value.Context.Semantic = decision.SemanticContextIntentFeatureVersion, value.Binding.Semantic
	value.Context.Original = prepared.PlanSHA256()
	for _, choice := range doc.Plan.Decisions {
		_, natural, _ := strings.Cut(choice.Intent, "intent: ")
		fields, err := prepared.SourceFeatures(choice.ID)
		if err != nil {
			t.Fatal(err)
		}
		text, err := decision.EncodeSemanticContextInput(fields, natural)
		if err != nil {
			t.Fatal(err)
		}
		in := input{choice.ID, text, "sha256:" + hash([]byte(text)), "sha256:" + hash([]byte(natural)),
			"sha256:" + hash([]byte(choice.Intent)), "sha256:" + hash(fields[:]), len(text), true}
		value.Inputs = append(value.Inputs, in)
		in.Text = ""
		value.Context.Inputs = append(value.Context.Inputs, in)
	}
	return doc, source, value
}

func TestCompleteExportsAndNegativeInputIdentity(t *testing.T) {
	for _, family := range threecompositionstudy.Families {
		doc, source, value := exportedFixture(t, family)
		raw, _ := json.Marshal(value)
		_, text, err := inspect(raw, doc, source)
		if err != nil || text == "" || len(doc.Cases) != 16 || doc.Max != 8 {
			t.Fatal("complete fixture/export", family, err)
		}
		original := string(raw)
		for _, mutate := range []func(*export){
			func(v *export) { v.Inputs = v.Inputs[:2]; v.Context.Inputs = v.Context.Inputs[:2] },
			func(v *export) { v.Inputs[2].Text += "x" },
			func(v *export) { v.Context.Inputs[2].Text = "unexpected" },
			func(v *export) { v.Tests = nil },
			func(v *export) { v.Binding.Equivalent = false },
			func(v *export) { v.Context.Metadata = strings.Repeat("a", 64) },
			func(v *export) { v.Source = strings.Repeat("b", 64) },
		} {
			var changed export
			if err := json.Unmarshal([]byte(original), &changed); err != nil {
				t.Fatal(err)
			}
			mutate(&changed)
			bad, _ := json.Marshal(changed)
			if _, _, err := inspect(bad, doc, source); err == nil {
				t.Fatal("invalid export accepted", family)
			}
		}
		duplicate := strings.Replace(original, `"model_predictions":0`, `"model_predictions":0,"model_predictions":0`, 1)
		if _, _, err := inspect([]byte(duplicate), doc, source); err == nil {
			t.Fatal("duplicate JSON accepted")
		}
	}
}

func TestFirstActualNativeExportRegression(t *testing.T) {
	doc, source, _, err := fixture("chained_operands", 0, 0, "en")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("testdata/first-native-export.json")
	if err != nil {
		t.Fatal(err)
	}
	value, text, err := inspect(raw, doc, source)
	if err != nil || text == "" || len(value.Inputs) != 3 || *value.Predictions != 0 {
		t.Fatal("actual native export regression", err)
	}
}

func TestJournalCapsPreserveBytesAndHashes(t *testing.T) {
	var used int64
	name := filepath.Join(t.TempDir(), "journal.jsonl")
	j, err := openJournal(name, &used)
	if err != nil {
		t.Fatal(err)
	}
	defer j.file.Close()
	if err := j.append(map[string]int{"value": 1}); err != nil {
		t.Fatal(err)
	}
	prefix, err := os.ReadFile(name)
	if err != nil || hash(prefix) != j.pin()["sha256"] || int64(len(prefix)) != used {
		t.Fatal("actual journal pin", err)
	}
	for _, value := range []string{strings.Repeat("x", lineCap), "/" + "Users/" + "synthetic/project"} {
		if err := j.append(value); err == nil {
			t.Fatal("oversized/private journal accepted")
		}
	}
	used = rawCap - 1
	if err := j.append("ok"); err == nil || used != rawCap-1 || j.lines != 1 || j.bytes != int64(len(prefix)) {
		t.Fatal("shared raw cap mutated prefix")
	}
	actual, _ := os.ReadFile(name)
	if string(actual) != string(prefix) || hash(actual) != j.pin()["sha256"] {
		t.Fatal("decline changed committed journal")
	}
	if _, err := openJournal(name, &used); err == nil {
		t.Fatal("journal overwrite accepted")
	}
}

func TestProtocolAndNativePinBounds(t *testing.T) {
	t.Chdir("../..")
	raw, err := os.ReadFile(protocol)
	if err != nil || hash(raw) != protocolSHA {
		t.Fatal("protocol differs", err)
	}
	if _, _, err := pin("missing", "missing", "missing"); err == nil {
		t.Fatal("missing pins accepted")
	}
	var b limitedBuffer
	if _, err := b.Write(make([]byte, lineCap)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte{1}); err == nil || b.Len() != lineCap {
		t.Fatal("child bound changed prefix")
	}
}
