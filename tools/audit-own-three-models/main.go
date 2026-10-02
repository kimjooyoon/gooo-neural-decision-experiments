// audit-own-three-models independently loads all fresh exports and exercises
// their actual Go kernels, while preserving optimization versus behavior scope.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

var arms = [3]string{"uniform-initial", "set-initial", "set-feedback"}
var variants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}

type exportPin struct {
	Metadata    string  `json:"metadata_sha256"`
	Weights     string  `json:"weights_sha256"`
	Packed      int     `json:"packed_weights_bytes"`
	Temperature float64 `json:"temperature"`
	NLL         float64 `json:"calibration_export_passing_set_nll"`
}
type parityRow struct {
	Variant       string       `json:"variant"`
	ID            string       `json:"state_id"`
	Phase         string       `json:"phase"`
	Text          string       `json:"text"`
	Features      [768]float32 `json:"features"`
	Logits        [8]float32   `json:"logits"`
	Probabilities [8]float32   `json:"probabilities"`
	Mask          uint16       `json:"selected_mask"`
}
type measure struct {
	Metadata        string  `json:"metadata_sha256"`
	Weights         string  `json:"weights_sha256"`
	Packed          int     `json:"packed_weights_bytes"`
	Resident        int     `json:"resident_tensor_bytes"`
	Scales          int     `json:"matrix_scale_bytes"`
	Workspace       uintptr `json:"caller_workspace_bytes"`
	ParityCalls     int     `json:"actual_parity_predictions"`
	Initial         int     `json:"development_initial_parity_rows"`
	Feedback        int     `json:"training_feedback_parity_rows"`
	WarmCalls       int     `json:"actual_warm_probe_predictions"`
	AllocationCalls int     `json:"actual_allocation_probe_predictions"`
	NegativeAPI     int     `json:"invalid_input_api_probes_with_zero_kernel_predictions"`
	NS              float64 `json:"warm_ns_per_prediction"`
	Allocations     float64 `json:"heap_allocations_per_valid_predict_into"`
	MaxError        float64 `json:"maximum_absolute_parity_error"`
}

func main() {
	models := flag.String("models", "", "actual new training output or complete published copy")
	dataset := flag.String("dataset", "", "exact native source dataset")
	teacher := flag.String("teacher-curriculum", "", "frozen actual teacher states")
	output := flag.String("output", "", "fresh audit output")
	revision := flag.String("source-revision", "", "clean exact Go audit source")
	shared := flag.Bool("shared-experiment", false, "audit the fixed dense/shared-local experiment")
	flag.Parse()
	auditor := audit
	if *shared {
		auditor = auditShared
	}
	if err := auditor(*models, *dataset, *teacher, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func read(name string, value any) error {
	p, err := threestudent.FilePin(name)
	if err != nil || p.Bytes > 4<<20 {
		return errors.New("bounded regular audit JSON required")
	}
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func audit(root, dataset, teacher, output, revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact Go audit source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean Go audit source required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh model audit output required")
	}
	states, err := threestudent.Load(dataset, teacher, "publication/own-three-choice-teacher-audit-20261002.json")
	if err != nil {
		return err
	}
	byID := map[string]threefeedback.State{}
	for _, s := range states {
		byID[s.ID] = s
	}
	var report struct {
		Schema   string                            `json:"schema"`
		Status   string                            `json:"status"`
		Steps    int                               `json:"optimizer_steps"`
		PreSHA   string                            `json:"preexecution_sha256"`
		Training map[string]map[string]stageReport `json:"training"`
		Exports  map[string]map[string]exportPin   `json:"exports"`
		Promoted bool                              `json:"default_model_promoted"`
	}
	if err = read(filepath.Join(root, "report.json"), &report); err != nil {
		return err
	}
	prePin, err := threestudent.FilePin(filepath.Join(root, "preexecution.json"))
	if err != nil || report.Schema != "gooo/own-three-choice-training-report/v1" || report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 4800 || report.PreSHA != prePin.SHA || report.Promoted || len(report.Training) != 3 || len(report.Exports) != 3 {
		return errors.New("complete nine-export fixed optimization report required")
	}
	if err = verifyPre(filepath.Join(root, "preexecution.json")); err != nil {
		return err
	}
	observed := map[string]measure{}
	calls := 0
	for _, arm := range arms {
		if len(report.Training[arm]) != 2 || len(report.Exports[arm]) != 3 {
			return errors.New("all declared arm stages/variants required")
		}
		fp, qat := report.Training[arm]["fp32"], report.Training[arm]["qat_ternary"]
		if fp.InitialSHA != threestudent.InitialSHA || qat.InitialSHA != fp.SelectedSHA {
			return errors.New("fresh shared initializer or arm-specific QAT checkpoint differs")
		}
		var duplicate map[string]stageReport
		if err = read(filepath.Join(root, arm, "training-report.json"), &duplicate); err != nil || !sameJSON(duplicate, report.Training[arm]) {
			return errors.New("arm-stage report differs")
		}
		for name, stage := range map[string]stageReport{"fp32": fp, "qat": qat} {
			if err = verifyStage(filepath.Join(root, arm, name), arm, name, stage); err != nil {
				return err
			}
		}
		rows, err := parityRows(filepath.Join(root, arm, "go-parity.jsonl"))
		if err != nil {
			return err
		}
		for _, variant := range variants {
			m, err := jointdecision.LoadThree(filepath.Join(root, arm, "models", variant, "model.json"))
			if err != nil {
				return err
			}
			p := report.Exports[arm][variant]
			if m.Variant() != variant || m.MetadataSHA256() != p.Metadata || m.WeightsSHA256() != p.Weights || m.PackedFileBytes() != p.Packed || !finite(p.NLL) || !validTemperature(p.Temperature) {
				return errors.New("actual model export pin differs")
			}
			var metadata struct {
				Temperature float64 `json:"temperature"`
			}
			if err = read(filepath.Join(root, arm, "models", variant, "model.json"), &metadata); err != nil || metadata.Temperature != p.Temperature {
				return errors.New("actual calibration temperature differs")
			}
			if variant == "fp32" && p.Temperature != fp.Temperature || variant == "qat_ternary" && p.Temperature != qat.Temperature {
				return errors.New("joint epoch/temperature selection differs")
			}
			r, err := observe(m, variant, rows, byID)
			if err != nil {
				return fmt.Errorf("%s/%s: %w", arm, variant, err)
			}
			observed[arm+"/"+variant] = r
			calls += r.ParityCalls + r.WarmCalls + r.AllocationCalls
		}
	}
	raw, err := json.MarshalIndent(map[string]any{"schema": "gooo/own-three-choice-model-go-audit/v1", "status": "PASS", "auditor_source_revision": revision, "models": observed, "model_exports": 9, "recorded_optimizer_updates_reconciled": 4800, "actual_go_model_predictions": calls, "actual_parity_predictions": 432, "invalid_input_api_probes_with_zero_kernel_predictions": 18, "maximum_allowed_absolute_parity_error": 1e-5, "new_optimizer_updates": 0, "new_native_calls": 0, "default_model_promoted": false, "scope": "All nine strict three-choice Go exports, 32 development initial and 16 actual-training-failure parity rows per model, zero-allocation valid warmed kernels, unchanged outputs on invalid arity/overflow and all 4800 recorded stage updates plus calibration checkpoint selection. Recorded update receipts are reconciled, not a second optimization. Kernel timing includes feature projection and is not source-to-codegen speed or SDK/native completeness."}, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func parityRows(name string) ([]parityRow, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 32768), 1<<20)
	rows := make([]parityRow, 0, 144)
	for scanner.Scan() {
		var r parityRow
		if err = threecohort.Decode(scanner.Bytes(), &r); err != nil {
			return nil, err
		}
		if len(rows) >= 144 {
			return nil, errors.New("extra parity rows")
		}
		rows = append(rows, r)
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	if len(rows) != 144 {
		return nil, errors.New("144 complete parity rows required")
	}
	for i, r := range rows {
		if r.Variant != variants[i/48] {
			return nil, errors.New("frozen variant parity order differs")
		}
	}
	return rows, nil
}

func observe(m *jointdecision.ThreeModel, variant string, rows []parityRow, states map[string]threefeedback.State) (measure, error) {
	r := measure{Metadata: m.MetadataSHA256(), Weights: m.WeightsSHA256(), Packed: m.PackedFileBytes(), Resident: m.ResidentTensorBytes(), Scales: m.MatrixScaleBytes(), Workspace: unsafe.Sizeof(jointdecision.ThreeWorkspace{})}
	expectedPacked, expectedResident, expectedScales := 74624, 74624, 0
	if variant != "fp32" {
		expectedPacked, expectedResident, expectedScales = 3854, 18752, 8
	}
	if r.Packed != expectedPacked || r.Resident != expectedResident || r.Scales != expectedScales || r.Workspace != 3200 {
		return r, errors.New("tensor/storage/workspace scopes differ")
	}
	var workspace jointdecision.ThreeWorkspace
	var prediction jointdecision.ThreePrediction
	first := ""
	seen := map[string]bool{}
	expectedIDs := expectedParityIDs(states)
	for _, row := range rows {
		if row.Variant != variant {
			continue
		}
		s, ok := states[row.ID]
		if !ok || seen[row.ID] || r.ParityCalls >= len(expectedIDs) || row.ID != expectedIDs[r.ParityCalls] || row.Text != s.Text || row.Phase != s.Phase {
			return r, errors.New("parity full input not bound to frozen distinct actual state")
		}
		seen[row.ID] = true
		if first == "" {
			first = row.Text
		}
		if s.Phase == "feedback" && s.Split == "train" {
			r.Feedback++
		} else if s.Phase == "initial" && s.Split == "development" {
			r.Initial++
		} else {
			return r, errors.New("parity source partition differs")
		}
		if err := m.PredictInto(row.Text, &workspace, &prediction); err != nil {
			return r, err
		}
		r.ParityCalls++
		if prediction.Mask != row.Mask {
			return r, errors.New("Go/export mask differs")
		}
		for i, v := range workspace.Features {
			r.MaxError = math.Max(r.MaxError, math.Abs(float64(v-row.Features[i])))
		}
		for i, v := range prediction.Logits {
			r.MaxError = math.Max(r.MaxError, math.Abs(float64(v-row.Logits[i])))
			r.MaxError = math.Max(r.MaxError, math.Abs(float64(prediction.Probabilities[i]-row.Probabilities[i])))
		}
	}
	if r.ParityCalls != 48 || r.Initial != 32 || r.Feedback != 16 || r.MaxError > 1e-5 {
		return r, errors.New("all parity rows or numerical tolerance differ")
	}
	for _, invalid := range []string{"gooo;joint2|1:x1:y", strings.Repeat("x", jointdecision.ThreeInputMaxBytes+1)} {
		beforeWorkspace, beforePrediction := workspace, prediction
		if err := m.PredictInto(invalid, &workspace, &prediction); err == nil || workspace != beforeWorkspace || prediction != beforePrediction {
			return r, errors.New("invalid input committed model output")
		}
		r.NegativeAPI++
	}
	if err := m.PredictInto(first, &workspace, &prediction); err != nil {
		return r, err
	}
	r.WarmCalls++
	started := time.Now()
	for range 1000 {
		if err := m.PredictInto(first, &workspace, &prediction); err != nil {
			return r, err
		}
		r.WarmCalls++
	}
	r.NS = float64(time.Since(started).Nanoseconds()) / 1000
	var failure error
	r.Allocations = testing.AllocsPerRun(1000, func() { failure = m.PredictInto(first, &workspace, &prediction); r.AllocationCalls++ })
	if failure != nil || r.Allocations != 0 || r.AllocationCalls != 1001 {
		return r, errors.New("valid warmed three-choice prediction allocated")
	}
	return r, nil
}

func finite(v float64) bool           { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func validTemperature(v float64) bool { return v == .5 || v == 1 || v == 2 || v == 4 }
func sameJSON(a, b any) bool {
	x, e := json.Marshal(a)
	if e != nil {
		return false
	}
	y, e := json.Marshal(b)
	return e == nil && string(x) == string(y)
}

func expectedParityIDs(states map[string]threefeedback.State) []string {
	result := make([]string, 0, 48)
	for _, selection := range []struct {
		split, phase string
		count        int
	}{{"development", "initial", 32}, {"train", "feedback", 16}} {
		ids := []string{}
		for id, s := range states {
			if s.Split == selection.split && s.Phase == selection.phase {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		if len(ids) < selection.count {
			return nil
		}
		for i := range selection.count {
			result = append(result, ids[i*(len(ids)-1)/(selection.count-1)])
		}
	}
	return result
}
