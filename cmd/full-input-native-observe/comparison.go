package main

import (
	"encoding/json"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type runtimePolicy struct{ Name, Model string }

var compactVariants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}

type compactBinding struct {
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
	Schema   string `json:"schema"`
}
type pairedGeneration struct {
	Source string `json:"source"`
	Report struct {
		Paths struct {
			Search   pathplan.SearchResult      `json:"search"`
			Feedback []pathplan.FeedbackReceipt `json:"feedback_judgments"`
		} `json:"body_paths"`
	} `json:"report"`
}

func normalizedThree(p *pathplan.ThreeReceipt, schema string) *pathplan.ThreeReceipt {
	if p == nil {
		return nil
	}
	if p.Schema != schema {
		panic("wrong runtime model schema")
	}
	copy := *p
	copy.Schema, copy.PredictNS = "", 0
	return &copy
}

// Normalize the same artifact/timing fields as the frozen compact study. Full
// inputs, versioned features/arithmetic, probabilities and search stay intact.
func normalizeGeneration(raw []byte, binding compactBinding) []byte {
	var g pairedGeneration
	must(json.Unmarshal(raw, &g))
	s := &g.Report.Paths.Search.Selection
	if s.MetadataSHA256 != binding.Metadata || s.WeightsSHA256 != binding.Weights || s.SeedSHA256 != "" || s.Three == nil || !s.ExternalCallsKnown || s.ExternalCalls != 0 || s.ModelCalls < 1 {
		panic("bound unseeded local selection required")
	}
	s.MetadataSHA256, s.WeightsSHA256 = "", ""
	s.Three = normalizedThree(s.Three, binding.Schema)
	for i := range s.Receipts {
		s.Receipts[i].PredictNS = 0
	}
	for i := range g.Report.Paths.Feedback {
		f := &g.Report.Paths.Feedback[i]
		if f.MetadataSHA != binding.Metadata || f.WeightsSHA != binding.Weights {
			panic("feedback model repinning differs")
		}
		f.MetadataSHA, f.WeightsSHA, f.SHA, f.PreviousSHA, f.FromProgressSHA = "", "", "", "", ""
		f.Three = normalizedThree(f.Three, binding.Schema)
	}
	b, err := json.Marshal(g)
	must(err)
	return b
}
