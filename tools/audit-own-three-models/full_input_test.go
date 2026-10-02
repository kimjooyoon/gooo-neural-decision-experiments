package main

import (
	"bytes"
	"math"
	"testing"
)

func TestFullInputStablePassingNLL(t *testing.T) {
	var logits [8]float32
	var passed [8]int
	passed[0] = 16
	if got := fullStableNLL(logits, passed, 1); math.Abs(got-math.Log(8)) > 1e-12 {
		t.Fatal("uniform passing-set loss", got)
	}
	logits[0], logits[1] = -1000, 1000
	if got := fullStableNLL(logits, passed, 1); !finite(got) || got != 2000 {
		t.Fatal("underflow-safe loss", got)
	}
	for i := range passed {
		passed[i] = 16
	}
	if got := fullStableNLL(logits, passed, .5); got != 0 {
		t.Fatal("whole passing set", got)
	}
}

func TestFullInputJournalBound(t *testing.T) {
	var b bytes.Buffer
	w := fullBoundedWriter{writer: &b, remaining: 3}
	if n, e := w.Write([]byte("ok")); n != 2 || e != nil {
		t.Fatal(n, e)
	}
	if n, e := w.Write([]byte("no")); n != 0 || e == nil || b.String() != "ok" {
		t.Fatal("prefix exceeded")
	}
	if n, e := w.Write([]byte("!")); n != 1 || e != nil || b.String() != "ok!" {
		t.Fatal(n, e)
	}
}
