package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTraceUsesEntitiesActivitiesAndCompilerAgent(t *testing.T) {
	value := strings.Repeat("a", 64)
	item := cell{RawSHA: value, SourceSHA: value, PlanSHA: value, MetadataSHA: value, WeightsSHA: value, GeneratedSHA: value, VerificationSHA: value}
	raw := string(trace(item, value))
	for _, term := range []string{"prov:SoftwareAgent", "prov:Activity", "prov:Entity", "prov:used", "prov:wasDerivedFrom", "prov:wasGeneratedBy", "prov:wasInformedBy", "prov:wasAssociatedWith"} {
		if !strings.Contains(raw, term) {
			t.Fatalf("missing %s", term)
		}
	}
	if strings.Contains(raw, "<urn:gooo:weights:>") {
		t.Fatal("empty model entity")
	}
}

func TestOfflineTraceHasNoInventedModelUsage(t *testing.T) {
	value := strings.Repeat("b", 64)
	raw := trace(cell{RawSHA: value, SourceSHA: value, PlanSHA: value, GeneratedSHA: value, VerificationSHA: value}, value)
	if bytes.Contains(raw, []byte("urn:gooo:weights")) || bytes.Contains(raw, []byte("urn:gooo:metadata")) {
		t.Fatal("invented offline model entity")
	}
}

func TestChildOutputLimit(t *testing.T) {
	var output boundedBuffer
	if _, err := output.Write(make([]byte, 128*1024)); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte{1}); err == nil || !output.Truncated || output.Len() != 128*1024 {
		t.Fatal("output cap not enforced")
	}
}
