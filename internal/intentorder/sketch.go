package intentorder

import (
	"errors"
	"math/bits"
	"strings"
	"unicode/utf8"
)

const IntentOrderSketchVersion = "semantic_intent_clause_edges_experiment_v1"
const IntentOrderSketchDim = 64
const InputMaxBytes = 512

// IntentOrderSketch is an experimental auxiliary channel. Existing model input
// dimensions, features and loaders are unchanged. Counts preserve some ordered
// clause transitions; hashing and clause boundaries can still lose distinctions.
type IntentOrderSketch [IntentOrderSketchDim]uint16

// Into consumes complete bounded intention text and commits caller storage only
// after validation. Callers separately bind source context; this feature module
// has no model or source-framing dependency. Clause edges include both boundaries.
func Into(intent string, output *IntentOrderSketch) error {
	if output == nil || len(intent) == 0 || len(intent) > InputMaxBytes || !utf8.ValidString(intent) {
		return errors.New("bounded valid order-sketch input and output required")
	}
	var candidate IntentOrderSketch
	previous := uint64(0x243f6a8885a308d3)
	start, clauses := 0, 0
	appendClause := func(end int) {
		clause := strings.TrimSpace(intent[start:end])
		if clause == "" {
			return
		}
		current := clauseOrderHash(clause)
		candidate[orderEdgeBucket(previous, current)]++
		previous = current
		clauses++
	}
	for at, r := range intent {
		switch r {
		case '.', ';', '!', '?', '\n', '。', '；', '！', '？':
			appendClause(at)
			start = at + utf8.RuneLen(r)
		}
	}
	appendClause(len(intent))
	if clauses != 0 {
		candidate[orderEdgeBucket(previous, 0x13198a2e03707344)]++
	}
	*output = candidate
	return nil
}

// This hash consumes all clause runes. Whitespace and case remain observable;
// only clause-edge whitespace is trimmed, with no interpretation of vocabulary.
func clauseOrderHash(text string) uint64 {
	h := uint64(14695981039346656037)
	for _, r := range text {
		// Encode the rune as fixed-width integers to keep byte boundaries explicit.
		for shift := 0; shift < 32; shift += 8 {
			h ^= uint64(uint32(r) >> shift & 255)
			h *= 1099511628211
		}
	}
	return h
}

func orderEdgeBucket(before, after uint64) uint64 {
	h := before ^ bits.RotateLeft64(after, 23) ^ 0x9e3779b97f4a7c15
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	h *= 0x94d049bb133111eb
	h ^= h >> 31
	return h % IntentOrderSketchDim
}
