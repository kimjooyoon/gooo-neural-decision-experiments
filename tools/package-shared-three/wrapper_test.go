package main

import (
	"strings"
	"testing"
)

func TestWrapperInventoryAndStandaloneScope(t *testing.T) {
	files := wrapperFiles("raw", "dataset", "dense", "shared")
	if len(files) != 30 {
		t.Fatalf("closed inventory: %d", len(files))
	}
	weights := 0
	for name := range files {
		if strings.HasSuffix(name, "weights.bin") {
			weights++
			if standaloneFile(name, wrapperSchema) {
				t.Fatal("unnecessary standalone model copy")
			}
		}
	}
	if weights != 6 || !standaloneFile("raw/report.json", wrapperSchema) {
		t.Fatal("model/report inventory")
	}
}

func TestWrapperSourceExceptionDoesNotApplyToEvidence(t *testing.T) {
	publicationPrivacy("source/auditor/main_test.go", []byte("\"/Users/example/private\""), wrapperSchema)
	defer func() {
		if recover() == nil {
			t.Fatal("runtime evidence privacy exemption")
		}
	}()
	publicationPrivacy("raw/inputs.jsonl", []byte("\"/Users/example/private\""), wrapperSchema)
}
