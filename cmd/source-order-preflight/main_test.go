package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/orderfacts"
)

func TestPinnedCompilerExportDigestConventions(t *testing.T) {
	raw, err := os.ReadFile("testdata/add-multiply-context.json")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../publication/native-order-20261003/add-multiply.gooo")
	if err != nil {
		t.Fatal(err)
	}
	var e exported
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if err := validateExport(e, source); err != nil {
		t.Fatal(err)
	}
	if _, err := orderfacts.Alternatives(e.Plan, "order"); err != nil {
		t.Fatal(err)
	}
	for _, input := range e.Inputs {
		if input.InputSHA != "sha256:"+hash([]byte(input.Text)) || len(input.FeatureSHA) != 71 {
			t.Fatal("compiler input digest convention changed")
		}
	}
	e.SourceSHA = strings.TrimPrefix(e.SourceSHA, "sha256:")
	if validateExport(e, source) == nil {
		t.Fatal("unprefixed source digest accepted")
	}
	e.SourceSHA = "sha256:" + hash(source)
	e.Plan.Base.Expressions[2].Int++
	if validateExport(e, source) == nil {
		t.Fatal("plan mutation retained claimed identity")
	}
}
