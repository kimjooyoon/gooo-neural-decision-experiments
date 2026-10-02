package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompactInventoryContainsEveryCapture(t *testing.T) {
	files := compactFiles("captures")
	if len(files) != 684 {
		t.Fatalf("inventory count = %d", len(files))
	}
	counts := map[string]int{}
	for name := range files {
		if strings.HasSuffix(name, "/runtime.json") {
			counts["runtime"]++
		}
		if strings.HasSuffix(name, "/generation.json") {
			counts["generation"]++
		}
	}
	if counts["runtime"] != 96 || counts["generation"] != 96 {
		t.Fatalf("capture count: %v", counts)
	}
}

func TestPrivacyInspectsEmbeddedParentBytes(t *testing.T) {
	raw, err := json.Marshal(map[string]any{"parent_receipt_bytes": []byte(`{"path":"/Users/private/source"}`)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("encoded private path accepted")
		}
	}()
	privacy("native/sample/runtime.json", raw)
}
