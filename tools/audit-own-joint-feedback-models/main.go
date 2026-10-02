// audit-own-joint-feedback-models exercises every new export through strict Go.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
)

var arms = [3]string{"uniform-initial", "set-initial", "set-feedback"}
var variants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}

type pin struct {
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
	Packed   int    `json:"packed_weights_bytes"`
}

type row struct {
	Variant       string       `json:"variant"`
	ID            string       `json:"state_id"`
	Phase         string       `json:"phase"`
	Text          string       `json:"text"`
	Features      [512]float32 `json:"features"`
	Logits        [4]float32   `json:"logits"`
	Probabilities [4]float32   `json:"probabilities"`
	Mask          uint16       `json:"selected_mask"`
}

type measure struct {
	Metadata        string  `json:"metadata_sha256"`
	Weights         string  `json:"weights_sha256"`
	Packed          int     `json:"packed_weights_bytes"`
	Resident        int     `json:"resident_tensor_bytes"`
	Scales          int     `json:"matrix_scale_bytes"`
	Workspace       uintptr `json:"caller_workspace_bytes"`
	WarmCalls       int     `json:"actual_warm_probe_predictions"`
	AllocationCalls int     `json:"actual_allocation_probe_predictions"`
	NS              float64 `json:"warm_ns_per_prediction"`
	Allocations     float64 `json:"heap_allocations_per_valid_predict_into"`
	ParityCalls     int     `json:"actual_parity_predictions"`
	InitialParity   int     `json:"initial_parity_rows"`
	FeedbackParity  int     `json:"feedback_parity_rows"`
	MaxError        float64 `json:"maximum_absolute_parity_error"`
}

func main() {
	models := flag.String("models", "", "nine committed own exports")
	curriculum := flag.String("curriculum", "", "audited student state directory")
	output := flag.String("output", "", "fresh independent Go inference audit")
	revision := flag.String("source-revision", "", "clean exact Go auditor source")
	flag.Parse()
	if err := audit(*models, *curriculum, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func read(name string, value any) error {
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func audit(root, curriculum, output, revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact committed Go auditor required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean Go auditor required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh Go inference audit required")
	}
	states, err := loadStates(curriculum)
	if err != nil {
		return err
	}
	var report struct {
		Status  string                    `json:"status"`
		Updates int                       `json:"optimizer_steps"`
		PreSHA  string                    `json:"preexecution_sha256"`
		Exports map[string]map[string]pin `json:"exports"`
	}
	if err = read(filepath.Join(root, "report.json"), &report); err != nil {
		return err
	}
	preRaw, err := os.ReadFile(filepath.Join(root, "preexecution.json"))
	if err != nil || jointcohort.SHA(preRaw) != report.PreSHA || report.Status != "TRAINED_AND_EXPORTED" || report.Updates != 3600 || len(report.Exports) != 3 {
		return errors.New("nine frozen fresh exports and optimizer budget required")
	}
	if err = verifyPre(preRaw); err != nil {
		return err
	}
	observed, calls := map[string]measure{}, 0
	for _, arm := range arms {
		var parity struct {
			Rows []row `json:"rows"`
		}
		if err = read(filepath.Join(root, arm, "go-parity.json"), &parity); err != nil {
			return err
		}
		if len(parity.Rows) != 144 || len(report.Exports[arm]) != 3 {
			return errors.New("48 source-bound parity rows per variant required")
		}
		for _, variant := range variants {
			m, e := jointdecision.Load(filepath.Join(root, arm, "models", variant, "model.json"))
			if e != nil {
				return e
			}
			p := report.Exports[arm][variant]
			if m.Variant() != variant || m.MetadataSHA256() != p.Metadata || m.WeightsSHA256() != p.Weights || m.PackedFileBytes() != p.Packed {
				return errors.New("actual export pin differs")
			}
			r, e := observe(m, variant, parity.Rows, states)
			if e != nil {
				return e
			}
			observed[arm+"/"+variant] = r
			calls += r.ParityCalls + r.WarmCalls + r.AllocationCalls
		}
	}
	raw, err := json.MarshalIndent(map[string]any{"schema": "gooo/own-joint-feedback-model-go-audit/v2", "status": "PASS",
		"auditor_source_revision": revision, "maximum_allowed_absolute_parity_error": 1e-5,
		"models": observed, "actual_model_predictions": calls, "actual_parity_predictions": 432, "model_exports": 9,
		"recorded_optimizer_updates": 3600, "new_optimizer_updates": 0, "new_native_calls": 0,
		"scope": "Strict Go loading and real PredictInto numerical parity on 32 development initial inputs and 16 training actual-feedback inputs per variant. Warm timing/allocation probes are real separately counted calls. This is not student completeness or native compiled execution evidence."}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(raw, '\n'), 0644)
}

func observe(m *jointdecision.Model, variant string, rows []row, states map[string]jointfeedback.State) (measure, error) {
	r := measure{Metadata: m.MetadataSHA256(), Weights: m.WeightsSHA256(), Packed: m.PackedFileBytes(), Resident: m.ResidentTensorBytes(), Scales: m.MatrixScaleBytes(), Workspace: unsafe.Sizeof(jointdecision.Workspace{})}
	var workspace jointdecision.Workspace
	var prediction jointdecision.Prediction
	first := ""
	for _, row := range rows {
		if row.Variant != variant {
			continue
		}
		state, ok := states[row.ID]
		if !ok || state.Text != row.Text || state.Phase != row.Phase {
			return r, errors.New("parity input not bound to actual student state")
		}
		if first == "" {
			first = row.Text
		}
		if err := m.PredictInto(row.Text, &workspace, &prediction); err != nil {
			return r, err
		}
		r.ParityCalls++
		if state.Phase == "feedback" {
			r.FeedbackParity++
		} else if state.Split == "development" {
			r.InitialParity++
		} else {
			return r, errors.New("initial parity must use development")
		}
		if prediction.Mask != row.Mask {
			return r, errors.New("Go/export selected mask differs")
		}
		for i, x := range workspace.Features {
			r.MaxError = math.Max(r.MaxError, math.Abs(float64(x-row.Features[i])))
		}
		for i, x := range prediction.Logits {
			r.MaxError = math.Max(r.MaxError, math.Abs(float64(x-row.Logits[i])))
			r.MaxError = math.Max(r.MaxError, math.Abs(float64(prediction.Probabilities[i]-row.Probabilities[i])))
		}
	}
	if r.ParityCalls != 48 || r.InitialParity != 32 || r.FeedbackParity != 16 || r.MaxError > 1e-5 {
		return r, errors.New("parity row counts or numerical bounds differ")
	}
	if err := m.PredictInto(first, &workspace, &prediction); err != nil {
		return r, err
	}
	started := time.Now()
	for range 1000 {
		if err := m.PredictInto(first, &workspace, &prediction); err != nil {
			return r, err
		}
	}
	r.NS, r.WarmCalls = float64(time.Since(started).Nanoseconds())/1000, 1001
	var failure error
	r.Allocations = testing.AllocsPerRun(1000, func() { failure = m.PredictInto(first, &workspace, &prediction) })
	r.AllocationCalls = 1001
	if failure != nil || r.Allocations != 0 {
		return r, errors.New("valid Go inference must allocate zero heap objects")
	}
	return r, nil
}
