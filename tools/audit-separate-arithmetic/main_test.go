package main

import (
	"bytes"
	"compress/gzip"
	"math"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

func TestRankingAndFiniteCoverageKeepEveryBudget(t *testing.T) {
	p := jointdecision.ThreePrediction{Probabilities: [8]float32{.25, .25, .25, .25}, Mask: 0}
	order := rank(p)
	if order != [8]int{0, 1, 2, 3, 4, 5, 6, 7} {
		t.Fatal("tie order changed", order)
	}
	partial, at := finiteOutcomes(order, [8]int{2, 7, 5, 16, 0, 0, 0, 0})
	if partial != [8]int{2, 7, 7, 16, 16, 16, 16, 16} || at != 4 {
		t.Fatal("finite coverage", partial, at)
	}
	o := observation{Prediction: p, Order: order, Partial: partial, CompleteAt: at}
	o.HiddenSHA, o.LogitsSHA, o.ProbsSHA = bitsSHA(o.Hidden[:]), bitsSHA(p.Logits[:]), bitsSHA(p.Probabilities[:])
	if err := validateObservation(o, [8]int{2, 7, 5, 16}); err != nil {
		t.Fatal(err)
	}
	bad := o
	bad.Hidden[0] = 1
	if validateObservation(bad, [8]int{2, 7, 5, 16}) == nil {
		t.Fatal("changed hidden values accepted")
	}
	bad = o
	bad.Partial[2] = 16
	if validateObservation(bad, [8]int{2, 7, 5, 16}) == nil {
		t.Fatal("changed partial outcome accepted")
	}
	var counts totals
	counts.add(o)
	if counts.Extra != 3 || counts.First != 0 || counts.Complete != [8]int{0, 0, 0, 1, 1, 1, 1, 1} {
		t.Fatal(counts)
	}
}

func TestBitDigestsRetainNegativeZero(t *testing.T) {
	var zero float32
	negative := math.Float32frombits(0x80000000)
	if zero != negative || bitsSHA([]float32{zero}) == bitsSHA([]float32{negative}) {
		t.Fatal("signed zero identity lost")
	}
}

func TestJournalRequiresEveryRowAndValidTrailer(t *testing.T) {
	encode := func(rows int) []byte {
		var b bytes.Buffer
		z := gzip.NewWriter(&b)
		for range rows {
			if _, err := z.Write([]byte("{}\n")); err != nil {
				t.Fatal(err)
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	valid := encode(512)
	if err := scanGzip(bytes.NewReader(valid), func([]byte) error { return nil }); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{encode(511), encode(513), valid[:len(valid)-1]} {
		if scanGzip(bytes.NewReader(data), func([]byte) error { return nil }) == nil {
			t.Fatal("incomplete journal accepted")
		}
	}
}
