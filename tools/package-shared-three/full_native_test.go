package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFullNativeClosedInventory(t *testing.T) {
	files := fullNativeFiles("capture")
	if len(files) != 3614 {
		t.Fatalf("inventory count = %d", len(files))
	}
	counts := map[string]int{}
	for name := range files {
		for _, suffix := range []string{"generation.json", "runtime.json", "generation-process.json", "runtime-process.json"} {
			if strings.HasSuffix(name, "/"+suffix) {
				counts[suffix]++
			}
		}
	}
	for name, n := range counts {
		if n != 400 {
			t.Fatalf("%s count %d", name, n)
		}
	}
}

func TestPrivacyInspectsEncodedDiagnostics(t *testing.T) {
	raw, err := json.Marshal(map[string]any{"stderr_bytes": []byte("/Users/private/source")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("encoded diagnostic path accepted")
		}
	}()
	privacy("native/sample/generation-process.json", raw)
}
