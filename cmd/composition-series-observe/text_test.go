package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTextShowsCurrentPartialValues(t *testing.T) {
	raw := readCapture(t)
	m, err := measure(raw, candidateSHA, false)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = writeText(&out, m, raw); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"current model calls: 0", "outputs 1/2 (50.00%)", "record fields 3/3 (100.00%)", "actual=\"Third:ready:unchanged\"", "expected=\"a deliberately different expectation\""} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %s", want, out.String())
		}
	}
	if strings.Count(out.String(), "; reuse;") != 5 || strings.Count(out.String(), "; build;") != 1 {
		t.Fatal(out.String())
	}
}
