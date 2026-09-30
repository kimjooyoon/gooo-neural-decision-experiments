package main

import (
	"os"
	"strings"
	"testing"
)

func TestSavedBilingualSampleAccounting(t *testing.T) {
	t.Chdir("../..")
	for _, arm := range []string{"offline", "fp32"} {
		raw, err := os.ReadFile("runs/typed-path-reserved-probe-tdd-20261001/" + arm + "-rows.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		rows, err := samples(raw)
		if err != nil || len(rows) != 10 {
			t.Fatalf("samples: %d %v", len(rows), err)
		}
	}
	if _, err := samples([]byte("{}\n")); err == nil {
		t.Fatal("missing samples accepted")
	}
	if err := run("", "", nativeBinarySHA, goBinarySHA, "", "", strings.Repeat("0", 40)); err == nil || !strings.Contains(err.Error(), "declared source") {
		t.Fatal("incorrect declared source reached execution preflight")
	}
}
