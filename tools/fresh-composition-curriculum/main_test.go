package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

// Synthetic receipt construction tests the consumer's contract only. It is
// never counted as an actual native export or a learned-model observation.
func syntheticExport(t *testing.T) (export, document, []byte) {
	t.Helper()
	doc, source, _, err := fixture("branch_schedule", 40, 3, "ko")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		t.Fatal(err)
	}
	docRaw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var value export
	value.Schema, value.Source, value.Document = "gooo/compiler-path-input-export/v2", "sha256:"+hash(source), "sha256:"+hash(docRaw)
	caseRaw, err := json.Marshal(doc.Cases)
	if err != nil {
		t.Fatal(err)
	}
	value.TestsSHA = "sha256:" + hash(caseRaw)
	value.Binding.Equivalent, value.Binding.Semantic = true, "sha256:"+strings.Repeat("b", 64)
	value.Context.Schema, value.Context.Status = "gooo/compiler-typed-path-context/v3", "ENCODED"
	value.Context.Feature, value.Context.Original, value.Context.Semantic = decision.SemanticContextIntentFeatureVersion, prepared.PlanSHA256(), value.Binding.Semantic
	for _, choice := range doc.Plan.Decisions {
		fields, err := prepared.SourceFeatures(choice.ID)
		if err != nil {
			t.Fatal(err)
		}
		natural := choice.Intent[strings.LastIndex(choice.Intent, "intent: ")+8:]
		text, err := decision.EncodeSemanticContextInput(fields, natural)
		if err != nil {
			t.Fatal(err)
		}
		in := input{choice.ID, text, "sha256:" + hash([]byte(text)), "sha256:" + hash([]byte(natural)), "sha256:" + hash([]byte(choice.Intent)), "sha256:" + hash(fields[:]), len(text), true}
		value.Inputs = append(value.Inputs, in)
		in.Text = ""
		value.Context.Inputs = append(value.Context.Inputs, in)
	}
	return value, doc, source
}

func TestSourceBoundExportConsumer(t *testing.T) {
	value, doc, source := syntheticExport(t)
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = inspect(raw, doc, source, decision.SemanticContextIntentFeatureVersion); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*export){
		"nonzero predictions":     func(v *export) { v.Predictions = 1 },
		"candidate tests":         func(v *export) { v.Tests = 1 },
		"repository write":        func(v *export) { v.Writes = 1 },
		"selected emission":       func(v *export) { v.Emission = true },
		"unbound source":          func(v *export) { v.Binding.Equivalent = false },
		"source mismatch":         func(v *export) { v.Source = "different" },
		"finite suite mismatch":   func(v *export) { v.TestsSHA = "different" },
		"semantic mismatch":       func(v *export) { v.Context.Semantic = "different" },
		"input overflow":          func(v *export) { v.Inputs[0].Bytes = 513 },
		"source array mismatch":   func(v *export) { v.Inputs[0].SourceFeatures = "different" },
		"model metadata":          func(v *export) { v.Context.Metadata = "model" },
		"atomic decline":          func(v *export) { v.Context.Status = "DECLINED"; v.Inputs = nil },
		"caller context retained": func(v *export) { v.Inputs[0].Replaced = false },
	} {
		t.Run(name, func(t *testing.T) {
			altered, _, _ := syntheticExport(t)
			change(&altered)
			raw, _ := json.Marshal(altered)
			if _, err := inspect(raw, doc, source, decision.SemanticContextIntentFeatureVersion); err == nil {
				t.Fatal("altered receipt accepted")
			}
		})
	}
	var missing map[string]json.RawMessage
	if err = json.Unmarshal(raw, &missing); err != nil {
		t.Fatal(err)
	}
	delete(missing, "model_predictions")
	raw, _ = json.Marshal(missing)
	if _, err = inspect(raw, doc, source, decision.SemanticContextIntentFeatureVersion); err == nil {
		t.Fatal("absent prediction counter accepted as zero")
	}
}

func TestBoundedChildCaptureAndIndependentSource(t *testing.T) {
	var buffer limitedBuffer
	if n, err := buffer.Write([]byte("prior")); err != nil || n != 5 {
		t.Fatal("initial buffer")
	}
	if n, err := buffer.Write(make([]byte, 1<<20)); err == nil || n != 0 || buffer.String() != "prior" {
		t.Fatal("overflow mutated existing capture")
	}
	doc, source, target, err := fixture("assignment_schedule", 47, 2, "en")
	if err != nil || len(doc.Cases) != 16 || target.Passed[2] != 16 || doc.Max != 4 || privatePattern.Match(source) {
		t.Fatalf("public full fixture: %v", err)
	}
	if !strings.Contains(string(source), "activity ChoosePath(Integer) -> Integer computes") || len(doc.Plan.Decisions) != 2 {
		t.Fatal("Gooo source/body declaration missing")
	}
}

func TestStreamingDigestAndIncompleteAuditDecline(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dataset.jsonl")
	raw := []byte("deliberately incomplete public fixture\n")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sha, size, err := fileDigest(file)
	if err != nil || sha != hash(raw) || size != int64(len(raw)) {
		t.Fatal("streaming file digest differs")
	}
	if err = save(filepath.Join(dir, "manifest.json"), map[string]any{"status": "FAILED_PARTIAL_COLLECTION_RETAINED", "decision_rows": 9216, "native_export_calls": 4608}); err != nil {
		t.Fatal(err)
	}
	if auditCollection(dir, filepath.Join(dir, "audit.json")) == nil {
		t.Fatal("failed partial collection accepted")
	}
	if err = save(filepath.Join(dir, "manifest.json"), map[string]any{"status": "SOURCE_BOUND_EXPORTED", "decision_rows": 9216, "native_export_calls": 4608, "dataset_sha256": "wrong", "captures_sha256": "wrong"}); err != nil {
		t.Fatal(err)
	}
	if auditCollection(dir, filepath.Join(dir, "audit.json")) == nil {
		t.Fatal("mismatched collection digest accepted")
	}
}

func TestActualNativeByteCaptureRoundTrip(t *testing.T) {
	value, _, _ := syntheticExport(t)
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	// RawMessage removes the native encoder's newline. This prior format is a
	// retained failed collection, not proof of original native bytes.
	old, err := json.Marshal(map[string]any{"native_receipt": json.RawMessage(raw)})
	if err != nil {
		t.Fatal(err)
	}
	var previous struct {
		Raw json.RawMessage `json:"native_receipt"`
	}
	if json.Unmarshal(old, &previous) != nil || bytes.Equal(previous.Raw, raw) {
		t.Fatal("prior formatting-loss fixture did not reproduce")
	}
	encoded, err := json.Marshal(map[string]any{"schema": "gooo/fresh-composition-native-capture/v2", "native_receipt": raw})
	if err != nil {
		t.Fatal(err)
	}
	var lossless struct {
		Raw []byte `json:"native_receipt"`
	}
	if json.Unmarshal(encoded, &lossless) != nil || !bytes.Equal(lossless.Raw, raw) || hash(lossless.Raw) != hash(raw) {
		t.Fatal("actual native-byte capture changed")
	}
	if json.Unmarshal(old, &lossless) == nil {
		t.Fatal("lossy old object accepted as native byte capture")
	}
}
