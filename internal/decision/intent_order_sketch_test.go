package decision

import (
	"strings"
	"testing"
	"unsafe"
)

func TestOrderSketchSeparatesFrozenBagCounterexamples(t *testing.T) {
	for _, pair := range [][2]string{
		{"Step: add one. Step: multiply by two. Step: end.", "Step: multiply by two. Step: add one. Step: end."},
		{"단계: 1을 더한다. 단계: 2를 곱한다. 단계: 끝.", "단계: 2를 곱한다. 단계: 1을 더한다. 단계: 끝."},
	} {
		var bag [2][FeatureDim]float32
		var order [2]IntentOrderSketch
		for i, intent := range pair {
			text, err := EncodeSemanticContextInput(semanticTestFields(), intent)
			if err != nil {
				t.Fatal(err)
			}
			if err := SemanticContextBagFeaturesInto(text, &bag[i]); err != nil {
				t.Fatal(err)
			}
			if err := SemanticIntentOrderSketchInto(text, &order[i]); err != nil {
				t.Fatal(err)
			}
		}
		if bag[0] != bag[1] || order[0] == order[1] {
			t.Fatal("frozen alias not separated", pair)
		}
	}
}

func TestOrderSketchAtomicBoundedAndAllocationFree(t *testing.T) {
	text, err := EncodeSemanticContextInput(semanticTestFields(), "단계: 더한다. Step: multiply. End.")
	if err != nil {
		t.Fatal(err)
	}
	var output IntentOrderSketch
	if err := SemanticIntentOrderSketchInto(text, &output); err != nil {
		t.Fatal(err)
	}
	if size := unsafe.Sizeof(output); size != 128 {
		t.Fatal("unexpected workspace", size)
	}
	if n := testing.AllocsPerRun(100, func() {
		if err := SemanticIntentOrderSketchInto(text, &output); err != nil {
			panic(err)
		}
	}); n != 0 {
		t.Fatal("order sketch allocates", n)
	}
	before := output
	for _, bad := range []string{"", "plain text", "\xff", text[:semanticContextHeaderBytes], text + strings.Repeat("x", 512)} {
		if SemanticIntentOrderSketchInto(bad, &output) == nil || output != before {
			t.Fatal("invalid input changed output")
		}
	}
	if SemanticIntentOrderSketchInto(text, nil) == nil {
		t.Fatal("nil output accepted")
	}
	boundary, err := EncodeSemanticContextInput(semanticTestFields(), strings.Repeat("x", InputMaxBytes-semanticContextHeaderBytes))
	if err != nil || SemanticIntentOrderSketchInto(boundary, &output) != nil {
		t.Fatal("boundary input rejected", err)
	}
}

func TestOrderSketchDirectedEdgesAndBoundaries(t *testing.T) {
	first, second := clauseOrderHash("add"), clauseOrderHash("multiply")
	if orderEdgeBucket(first, second) == orderEdgeBucket(second, first) {
		t.Fatal("example directed edge collided")
	}
	var a, b IntentOrderSketch
	for i, intent := range []string{"add\n\nmultiply。end!", " add ; multiply ; end "} {
		text, _ := EncodeSemanticContextInput(semanticTestFields(), intent)
		out := &a
		if i == 1 {
			out = &b
		}
		if err := SemanticIntentOrderSketchInto(text, out); err != nil {
			t.Fatal(err)
		}
	}
	if a != b {
		t.Fatal("empty clauses or edge whitespace changed edges")
	}
	count := 0
	for _, v := range a {
		count += int(v)
	}
	if count != 4 {
		t.Fatal("three clauses need four boundary/transition edges", count)
	}
}

func BenchmarkSemanticIntentOrderSketch(b *testing.B) {
	text, _ := EncodeSemanticContextInput(semanticTestFields(), "단계: 1을 더한다. 단계: 2를 곱한다. 단계: 끝.")
	var output IntentOrderSketch
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := SemanticIntentOrderSketchInto(text, &output); err != nil {
			b.Fatal(err)
		}
	}
}

func TestOrderSketchRetainsRepeatedExcursionCounterexample(t *testing.T) {
	var out [2]IntentOrderSketch
	for i, intent := range []string{"A. B. A. C. A. D.", "A. C. A. B. A. D."} {
		text, err := EncodeSemanticContextInput(semanticTestFields(), intent)
		if err != nil {
			t.Fatal(err)
		}
		if err := SemanticIntentOrderSketchInto(text, &out[i]); err != nil {
			t.Fatal(err)
		}
	}
	if out[0] != out[1] {
		t.Fatal("directed edge multiset limitation changed")
	}
}
