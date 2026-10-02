package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func TestSharedFreshInitializerAndTopology(t *testing.T) {
	raw := sharedInitial()
	if threecohort.SHA(raw) != "e2ccb2bd0de4df48ecd1537acf2e54d44d4a3e81ea97c52ca4da928df54bd8fd" {
		t.Fatal("independent Go initializer differs from frozen recipe")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "model.json")
	meta := jointdecision.Metadata{WeightsFile: "weights.bin"}
	at := int64(0)
	for _, count := range []int{18432, 24, 192, 8} {
		meta.Tensors = append(meta.Tensors, decision.TensorMetadata{Count: count, Offset: at, Encoding: "float32_le", Scale: 1})
		at += int64(4 * count)
	}
	write := func(data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "weights.bin"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(raw)
	if err := verifyTopology(path, meta); err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{256 * 4, (8*768 + 256) * 4, 18432*4 + 8*4, (18432 + 24 + 2*24) * 4, (18656 - 1) * 4} {
		modified := append([]byte(nil), raw...)
		modified[offset+3] ^= 0x3f
		write(modified)
		if err := verifyTopology(path, meta); err == nil {
			t.Fatalf("accepted mutated shared topology at byte %d", offset)
		}
	}
}

func TestSharedPackedTernaryTopology(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.json")
	meta := jointdecision.Metadata{WeightsFile: "weights.bin", Tensors: []decision.TensorMetadata{
		{Count: 18432, Offset: 0, Encoding: "ternary_base3_5", Scale: .5},
		{Count: 24, Offset: 3687, Encoding: "float32_le", Scale: 1},
		{Count: 192, Offset: 3783, Encoding: "ternary_base3_5", Scale: .25},
		{Count: 8, Offset: 3822, Encoding: "float32_le", Scale: 1},
	}}
	raw := make([]byte, 3854)
	for i := range 3687 {
		raw[i] = 121
	}
	for i := range 39 {
		raw[3783+i] = 121
	}
	if err := os.WriteFile(filepath.Join(dir, "weights.bin"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyTopology(path, meta); err != nil {
		t.Fatal(err)
	}
	raw[256/5] += 3 // One off-diagonal trit becomes nonzero.
	if err := os.WriteFile(filepath.Join(dir, "weights.bin"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyTopology(path, meta); err == nil {
		t.Fatal("accepted nonzero packed off-diagonal weight")
	}
}

func TestStaticRankingKeepsPartialAndCompleteSeparate(t *testing.T) {
	v := threecohort.View{}
	v.Target.Cases = 16
	v.Target.Passed = [8]int{0, 8, 16, 0, 0, 0, 0, 0}
	v.Target.Joint[2] = 1
	p := jointdecision.ThreePrediction{Mask: 0, Probabilities: [8]float32{.5, .25, .25}}
	var counts developmentCounts
	counts.add(v, p, [8]int{0, 1, 2, 3, 4, 5, 6, 7})
	if counts.Views != 1 || counts.Cases != 16 || counts.InitialComplete != 0 || counts.InitialCases != 0 || counts.Extra != 2 || counts.Curve[1] != 0 || counts.Curve[2] != 1 || counts.Partial[1] != 8 || counts.PassingMass != .25 {
		t.Fatalf("wrong completeness accounting: %+v", counts)
	}
}
