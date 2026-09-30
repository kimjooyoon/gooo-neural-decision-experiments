package decisionstream

import (
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

// BenchmarkEvaluateJSONAndDecision includes envelope validation, JSON decode,
// typed-request validation, prediction, and response construction. Compare it
// with decision.BenchmarkPredictInto, which measures the allocation-free hot
// model path only.
func BenchmarkEvaluateJSONAndDecision(b *testing.B) {
	model, err := decision.Load(writeStreamFixtureModel(b))
	if err != nil {
		b.Fatal(err)
	}
	item := record{sequence: 1, raw: requestLine(b, "bench-1", "Add the quantity and fee.")}
	var workspace decision.Workspace
	b.ReportAllocs()
	b.SetBytes(int64(len(item.raw)))
	b.ResetTimer()
	for range b.N {
		result := evaluate(model, item, &workspace)
		if result.Status != "completed" {
			b.Fatalf("benchmark request rejected: %s", result.Error)
		}
	}
}
