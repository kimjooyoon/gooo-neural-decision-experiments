// Collect paired legacy/separate arithmetic with frozen source, text and weights.
package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const bundle = "publication/full-input-initial-study-20261003"
const manifestSHA = "45f78a838c0a7cc7255b26d596db3871a283cf6f1a8504a74f9cb211329ad32a"
const protocol = "docs/full-input-separate-arithmetic-protocol-20261003.md"
const outputCap int64 = 64 << 20

var arms = []string{"positioned-original", "positioned-varied", "bag-original", "bag-varied"}
var variants = []string{"fp32", "ptq_ternary", "qat_ternary"}
var forms = []string{"original", "task-prefix", "complete-suffix"}
var lanes = [4]string{"legacy_expanded", "legacy_compact", "separate_expanded", "separate_compact"}

type original struct {
	View       string                        `json:"view_id"`
	Group      string                        `json:"program_contract_group"`
	Language   string                        `json:"language"`
	Input      string                        `json:"input_sha256"`
	Source     string                        `json:"source_sha256"`
	Prediction jointdecision.ThreePrediction `json:"prediction"`
	Passed     [8]int                        `json:"passed_cases_by_mask"`
	Order      [8]int                        `json:"static_ranked_mask_order"`
	Text       string                        `json:"complete_input"`
}

type observation struct {
	Features   string                        `json:"feature_bits_sha256"`
	HiddenSHA  string                        `json:"hidden_bits_sha256"`
	LogitsSHA  string                        `json:"logit_bits_sha256"`
	ProbsSHA   string                        `json:"probability_bits_sha256"`
	Hidden     [24]float32                   `json:"hidden"`
	Prediction jointdecision.ThreePrediction `json:"prediction"`
	Order      [8]int                        `json:"ranked_mask_order"`
	Partial    [8]int                        `json:"best_passing_cases_by_budget"`
	CompleteAt int                           `json:"first_complete_budget"`
}

type pairedRow struct {
	Original original       `json:"original"`
	Lanes    [4]observation `json:"lanes"`
}

type totals struct {
	Views    int    `json:"views"`
	First    int    `json:"first_complete"`
	Extra    int    `json:"extra_ranked_attempts"`
	Complete [8]int `json:"complete_by_budget"`
	Partial  [8]int `json:"best_passing_cases_by_budget"`
}

type changes struct {
	Rows           int     `json:"rows"`
	HiddenRows     int     `json:"rows_with_both_hidden_observations"`
	Hidden         int     `json:"different_hidden_rows"`
	Logits         int     `json:"different_logit_rows"`
	Probabilities  int     `json:"different_probability_rows"`
	Masks          int     `json:"different_first_masks"`
	Orders         int     `json:"different_complete_rankings"`
	Finite         int     `json:"different_finite_outcomes"`
	MaxLogit       float64 `json:"maximum_absolute_logit_delta"`
	MaxProbability float64 `json:"maximum_absolute_probability_delta"`
}

type report struct {
	Schema       string                      `json:"schema"`
	Status       string                      `json:"status"`
	Source       string                      `json:"source_revision"`
	GoVersion    string                      `json:"go_version"`
	GOOS         string                      `json:"goos"`
	GOARCH       string                      `json:"goarch"`
	Manifest     string                      `json:"original_manifest_sha256"`
	Protocol     threestudent.Pin            `json:"protocol"`
	Sources      map[string]threestudent.Pin `json:"computational_sources"`
	Models       map[string]threestudent.Pin `json:"model_files"`
	Journals     map[string]threestudent.Pin `json:"journals"`
	Lanes        [4]string                   `json:"lane_order"`
	Rows         int                         `json:"paired_input_rows"`
	Calls        int                         `json:"actual_model_predictions"`
	Updates      int                         `json:"optimizer_updates"`
	Counts       map[string][4]totals        `json:"conditions"`
	Changes      map[string]changes          `json:"legacy_to_separate"`
	RetainedDiff map[string]changes          `json:"retained_to_current_legacy"`
	Bytes        int64                       `json:"output_bytes_before_report"`
	WallSeconds  float64                     `json:"collection_wall_seconds"`
}

func main() {
	mode := flag.String("mode", "collect", "collect or compare")
	output := flag.String("output", "", "fresh output directory/report")
	source := flag.String("source-revision", "", "exact clean collection source")
	local := flag.String("local", "", "first complete collection")
	remote := flag.String("remote", "", "second complete collection")
	sourceDelta := flag.Bool("source-delta", false, "compare arithmetic across separately Git-verified source inventories")
	flag.Parse()
	var err error
	switch *mode {
	case "collect":
		err = collect(*output, *source)
	case "compare":
		err = compare(*local, *remote, *output, *sourceDelta)
	default:
		err = errors.New("unknown mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func bitsSHA(values []float32) string {
	var raw [768 * 4]byte
	for i, value := range values {
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(value))
	}
	return threecohort.SHA(raw[:len(values)*4])
}

func rank(p jointdecision.ThreePrediction) [8]int {
	order := [8]int{0, 1, 2, 3, 4, 5, 6, 7}
	sort.Slice(order[:], func(i, j int) bool {
		a, b := order[i], order[j]
		return p.Probabilities[a] > p.Probabilities[b] || p.Probabilities[a] == p.Probabilities[b] && a < b
	})
	return order
}

func observe(m *jointdecision.ThreeModel, row original) (observation, error) {
	var w jointdecision.ThreeWorkspace
	var p jointdecision.ThreePrediction
	if err := m.PredictInto(row.Text, &w, &p); err != nil {
		return observation{}, err
	}
	r := observation{Features: bitsSHA(w.Features[:]), HiddenSHA: bitsSHA(w.Hidden[:]), LogitsSHA: bitsSHA(p.Logits[:]), ProbsSHA: bitsSHA(p.Probabilities[:]), Hidden: w.Hidden, Prediction: p, Order: rank(p)}
	if r.Order[0] != int(p.Mask) {
		return r, errors.New("mask/ranking mismatch")
	}
	r.Partial, r.CompleteAt = finiteOutcomes(r.Order, row.Passed)
	if r.CompleteAt == 0 {
		return r, errors.New("frozen finite target lacks complete path")
	}
	return r, nil
}

func finiteOutcomes(order, passed [8]int) (partial [8]int, completeAt int) {
	best := 0
	for i, mask := range order {
		best = max(best, passed[mask])
		partial[i] = best
		if best == 16 && completeAt == 0 {
			completeAt = i + 1
		}
	}
	return
}

func (t *totals) add(o observation) {
	t.Views++
	if o.CompleteAt == 1 {
		t.First++
	}
	t.Extra += o.CompleteAt - 1
	for i, v := range o.Partial {
		t.Partial[i] += v
		if o.CompleteAt <= i+1 {
			t.Complete[i]++
		}
	}
}

func (c *changes) add(a, b observation) {
	c.Rows++
	if a.HiddenSHA != "" && b.HiddenSHA != "" {
		c.HiddenRows++
		if a.HiddenSHA != b.HiddenSHA {
			c.Hidden++
		}
	}
	if a.LogitsSHA != b.LogitsSHA {
		c.Logits++
	}
	if a.ProbsSHA != b.ProbsSHA {
		c.Probabilities++
	}
	if a.Prediction.Mask != b.Prediction.Mask {
		c.Masks++
	}
	if a.Order != b.Order {
		c.Orders++
	}
	if a.Partial != b.Partial || a.CompleteAt != b.CompleteAt {
		c.Finite++
	}
	for i := range a.Prediction.Logits {
		c.MaxLogit = math.Max(c.MaxLogit, math.Abs(float64(a.Prediction.Logits[i])-float64(b.Prediction.Logits[i])))
		c.MaxProbability = math.Max(c.MaxProbability, math.Abs(float64(a.Prediction.Probabilities[i])-float64(b.Prediction.Probabilities[i])))
	}
}
