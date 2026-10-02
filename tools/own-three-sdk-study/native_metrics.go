package main

import (
	"errors"
	"math"
	"slices"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

type childSummary struct {
	Children   int     `json:"children"`
	Wall       int64   `json:"summed_wall_ns"`
	CPU        int64   `json:"summed_cpu_ns"`
	CPUPercent float64 `json:"aggregate_cpu_percent_of_one_core"`
	Median     int64   `json:"median_wall_ns"`
	P95        int64   `json:"nearest_rank_p95_wall_ns"`
	RSS        int64   `json:"median_child_peak_rss_bytes"`
	MaxRSS     int64   `json:"max_child_peak_rss_bytes"`
}

func summarizeChildren(values [128]childMetrics) (childSummary, error) {
	var walls, resident [128]int64
	s := childSummary{Children: 128}
	for i, m := range values {
		if !validChild(m) || m.User > math.MaxInt64-m.System || s.Wall > math.MaxInt64-m.Wall || s.CPU > math.MaxInt64-m.User-m.System {
			return childSummary{}, errors.New("invalid or overflowing actual child metrics")
		}
		walls[i], resident[i] = m.Wall, m.RSS
		s.Wall += m.Wall
		s.CPU += m.User + m.System
	}
	slices.Sort(walls[:])
	slices.Sort(resident[:])
	s.Median, s.P95 = walls[63]+(walls[64]-walls[63])/2, walls[121]
	s.RSS, s.MaxRSS = resident[63]+(resident[64]-resident[63])/2, resident[127]
	s.CPUPercent = 100 * float64(s.CPU) / float64(s.Wall)
	return s, nil
}

type timingSummary struct {
	Sum    float64 `json:"sum_ms"`
	Median float64 `json:"median_ms"`
	P95    float64 `json:"nearest_rank_p95_ms"`
}
type nativeTotals struct {
	Counts    counts                   `json:"finite_counts"`
	Codegen   childSummary             `json:"actual_codegen_children"`
	Execution childSummary             `json:"actual_compile_and_run_children"`
	Timing    map[string]timingSummary `json:"native_internal_phase_timing"`
	Bilingual bilingualCounts          `json:"initial_bilingual_functional_diagnostic"`
}
type nativeAccumulator struct {
	Counts    counts
	Codegen   [128]childMetrics
	Execution [128]childMetrics
	Phases    [7][128]float64
	Pairs     map[string]map[string]firstOutput
}

func (a *nativeAccumulator) add(v threecohort.View, c threefeedback.Capture, n nativeResult, x nativeExecution) error {
	i := a.Counts.Views
	if i >= 128 {
		return errors.New("native policy exceeds 128 frozen views")
	}
	if err := a.Counts.add(v, c); err != nil {
		return err
	}
	a.Codegen[i], a.Execution[i] = x.Codegen, x.Execution
	t := n.Report.Paths.Timing
	for j, value := range [7]float64{t.Plan, t.Binding, t.Load, t.Context, t.Search, t.Emission, t.Total} {
		a.Phases[j][i] = value
	}
	if a.Pairs == nil {
		a.Pairs = map[string]map[string]firstOutput{}
	}
	if a.Pairs[v.Group] == nil {
		a.Pairs[v.Group] = map[string]firstOutput{}
	}
	if _, ok := a.Pairs[v.Group][v.Language]; ok {
		return errors.New("duplicate native bilingual view")
	}
	f, err := first(v, c)
	if err != nil {
		return err
	}
	a.Pairs[v.Group][v.Language] = f
	return nil
}
func (a *nativeAccumulator) finish() (nativeTotals, error) {
	if a.Counts.Views != 128 || a.Counts.Curve[7] != 128 || len(a.Pairs) != 64 {
		return nativeTotals{}, errors.New("complete native 128-view policy required")
	}
	c, err := summarizeChildren(a.Codegen)
	if err != nil {
		return nativeTotals{}, err
	}
	e, err := summarizeChildren(a.Execution)
	if err != nil {
		return nativeTotals{}, err
	}
	b := bilingualCounts{}
	for _, pair := range a.Pairs {
		en, enOK := pair["en"]
		ko, koOK := pair["ko"]
		if len(pair) != 2 || !enOK || !koOK {
			return nativeTotals{}, errors.New("exact native Korean/English pair required")
		}
		if err = b.add(en, ko); err != nil {
			return nativeTotals{}, err
		}
	}
	a.Counts.Pairs, a.Counts.Disagreement = b.Pairs, b.MaskDiff
	a.Counts.MedianNS, a.Counts.P95NS = c.Median, c.P95
	t := map[string]timingSummary{}
	for i, name := range [7]string{"plan_prepare", "source_binding", "model_load", "context_projection", "bounded_search", "final_emission", "total"} {
		values := a.Phases[i]
		s := timingSummary{}
		for _, v := range values {
			s.Sum += v
		}
		slices.Sort(values[:])
		s.Median, s.P95 = values[63]+(values[64]-values[63])/2, values[121]
		t[name] = s
	}
	return nativeTotals{a.Counts, c, e, t, b}, nil
}
