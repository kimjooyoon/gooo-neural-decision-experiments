package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

var sharedArms = [2]string{"dense", "shared-local"}

type sharedPre struct {
	Schema      string            `json:"schema"`
	Source      string            `json:"source_revision"`
	Sources     map[string]string `json:"sources_sha256"`
	Protocol    string            `json:"protocol_sha256"`
	Amendment   string            `json:"storage_amendment_sha256"`
	Storage     string            `json:"storage_preflight_sha256"`
	Prepared    string            `json:"prepared_manifest_sha256"`
	Preparation string            `json:"preparation_audit_sha256"`
	Initial     map[string]string `json:"initial_state_sha256"`
	Parameters  map[string]int    `json:"trainable_parameters"`
	Expanded    int               `json:"expanded_parameters"`
	Steps       int               `json:"planned_optimizer_steps"`
	Arms        []string          `json:"arms"`
	Device      string            `json:"device"`
	Prior       int64             `json:"prior_raw_bytes"`
	NewCap      int64             `json:"new_raw_cap_bytes"`
	WholeCap    int64             `json:"amended_whole_cap_bytes"`
	Promoted    bool              `json:"default_model_promoted"`
}

func sharedInitial() []byte {
	dense, out := threestudent.InitialWeights(), make([]byte, 74624)
	copyValue := func(to, from int) { copy(out[4*to:4*to+4], dense[4*from:4*from+4]) }
	for part := range 3 {
		for row := range 8 {
			for col := range 256 {
				copyValue((part*8+row)*768+part*256+col, row*768+col)
			}
			copyValue(768*24+part*8+row, 768*24+row)
		}
	}
	for mask := range 8 {
		for part := range 3 {
			for col := range 8 {
				copyValue(768*24+24+mask*24+part*8+col, 768*24+24+((mask>>part)&1)*24+col)
			}
		}
	}
	return out
}

func verifySharedPre(root string) (sharedPre, error) {
	var p sharedPre
	if err := read(filepath.Join(root, "preexecution.json"), &p); err != nil {
		return p, err
	}
	if p.Schema != "gooo/shared-three-judgment-training/v1" || p.Source != "0f3249596b4dacdb34241e3b5e8b4cafac58b0df" || p.Protocol != "96227c0c84c61cd48d2c696e9d858f1e6f149b081f32a76e9e86383c8472927b" || p.Amendment != "46157fecbf1bca42d2c86eb74e372bcbd84f8222b5ac21aa25300456a511beca" || p.Storage != "0d163043df89dd58c7b489dd952517350092babbbf063be889f3b33b1a12be35" || p.Initial["dense"] != threestudent.InitialSHA || p.Initial["shared-local"] != threecohort.SHA(sharedInitial()) || !sameJSON(p.Arms, sharedArms[:]) || !sameJSON(p.Parameters, map[string]int{"dense": 18656, "shared-local": 2072}) || len(p.Initial) != 2 || p.Expanded != 18656 || p.Steps != 3200 || p.Device != "mps" || p.Prior != 990581179 || p.NewCap != 64<<20 || p.WholeCap != 2<<30 || p.Promoted {
		return p, errors.New("shared preexecution/source/initialization/budget differs")
	}
	for name, expected := range map[string]string{
		"docs/shared-three-judgment-preregistration-20261002.md":          p.Protocol,
		"docs/shared-three-storage-amendment-20261003.md":                 p.Amendment,
		"preexecution/shared-three-storage-preflight-20261003.json":       p.Storage,
		"publication/own-three-choice-training-input-audit-20261002.json": p.Preparation,
	} {
		pin, err := threestudent.FilePin(name)
		if err != nil || pin.SHA != expected {
			return p, errors.New("shared input/protocol pin differs")
		}
	}
	var preparation struct {
		Manifest string `json:"manifest_sha256"`
	}
	if err := read("publication/own-three-choice-training-input-audit-20261002.json", &preparation); err != nil || preparation.Manifest != p.Prepared {
		return p, errors.New("prepared feature manifest differs")
	}
	if len(p.Sources) != 4 {
		return p, errors.New("closed optimizer source inventory")
	}
	for _, name := range []string{"train_shared_three_judgment_v1.py", "shared_evidence_v1.py", "train_own_three_feedback_v1.py", "train_pilot_v2.py"} {
		pin, err := threestudent.FilePin(filepath.Join("training", name))
		if err != nil || p.Sources[name] != pin.SHA {
			return p, errors.New("shared optimizer source changed")
		}
	}
	return p, nil
}

func auditShared(root, dataset, teacher, output, revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact audit source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean audit source required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh shared audit output required")
	}
	pre, err := verifySharedPre(root)
	if err != nil {
		return err
	}
	states, err := threestudent.Load(dataset, teacher, "publication/own-three-choice-teacher-audit-20261002.json")
	if err != nil {
		return err
	}
	byID := map[string]threefeedback.State{}
	for _, s := range states {
		byID[s.ID] = s
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
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
	if err != nil {
		return err
	}
	if report.Schema != "gooo/shared-three-judgment-training-result/v1" || report.Status != "TRAINED_AND_EXPORTED" || report.Steps != 3200 || report.PreSHA != prePin.SHA || report.Promoted || len(report.Training) != 2 || len(report.Exports) != 2 {
		return errors.New("six-export training report differs")
	}
	observed := map[string]measure{}
	behavior := map[string]development{}
	calls := 0
	for _, arm := range sharedArms {
		if len(report.Training[arm]) != 2 || len(report.Exports[arm]) != 3 {
			return errors.New("missing shared arm stage/export")
		}
		fp, qat := report.Training[arm]["fp32"], report.Training[arm]["qat_ternary"]
		if fp.InitialSHA != pre.Initial[arm] || qat.InitialSHA != fp.SelectedSHA {
			return errors.New("fresh arm/QAT initializer differs")
		}
		var duplicate map[string]stageReport
		if err = read(filepath.Join(root, arm, "training-report.json"), &duplicate); err != nil || !sameJSON(duplicate, report.Training[arm]) {
			return errors.New("arm report differs")
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
			path := filepath.Join(root, arm, "models", variant, "model.json")
			m, err := jointdecision.LoadThree(path)
			if err != nil {
				return err
			}
			pin := report.Exports[arm][variant]
			var meta jointdecision.Metadata
			if err = read(path, &meta); err != nil {
				return err
			}
			if m.Variant() != variant || m.MetadataSHA256() != pin.Metadata || m.WeightsSHA256() != pin.Weights || m.PackedFileBytes() != pin.Packed || meta.Temperature != pin.Temperature || !finite(pin.NLL) || !validTemperature(pin.Temperature) || variant == "fp32" && pin.Temperature != fp.Temperature || variant == "qat_ternary" && pin.Temperature != qat.Temperature {
				return errors.New("shared model export/calibration differs")
			}
			if variant == "fp32" && pin.Weights != fp.SelectedSHA {
				return errors.New("FP32 export is not the selected checkpoint")
			}
			if arm == "shared-local" {
				if err = verifyTopology(path, meta); err != nil {
					return err
				}
			}
			measurement, err := observe(m, variant, rows, byID)
			if err != nil {
				return fmt.Errorf("%s/%s: %w", arm, variant, err)
			}
			key := arm + "/" + variant
			observed[key] = measurement
			calls += measurement.ParityCalls + measurement.WarmCalls + measurement.AllocationCalls
			dev, err := evaluateDevelopment(m, views)
			if err != nil {
				return err
			}
			behavior[key] = dev
			calls += dev.Counts.Views
		}
	}
	value := map[string]any{"schema": "gooo/shared-three-go-audit/v1", "status": "PASS", "auditor_source_revision": revision, "training_source_revision": pre.Source, "preexecution_sha256": prePin.SHA, "dataset_sha256": threecohort.DatasetSHA, "optimizer_updates_reconciled": 3200, "model_exports": 6, "topology_verified_shared_exports": 3, "actual_parity_predictions": 288, "actual_development_predictions": 3072, "actual_go_model_predictions_including_kernel_probes": calls, "models": observed, "development": behavior, "new_optimizer_updates": 0, "new_native_calls": 0, "default_model_promoted": false, "scope": "All actual optimizer journals/calibration selectors, strict Go kernels and full source-bound development initial inputs. Ranking curves replay independently reconstructed finite targets without adaptive feedback or new native calls. This already-observed cohort is not untouched holdout or unseen natural-language generalization. Runtime tensors remain expanded."}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
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

func verifyTopology(path string, meta jointdecision.Metadata) error {
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), meta.WeightsFile))
	if err != nil {
		return err
	}
	var tensors [4][]float32
	for i, t := range meta.Tensors {
		tensors[i] = make([]float32, t.Count)
		for n := range t.Count {
			if t.Encoding == "float32_le" {
				tensors[i][n] = math.Float32frombits(binary.LittleEndian.Uint32(raw[int(t.Offset)+4*n:]))
			} else {
				packed := raw[int(t.Offset)+n/5]
				for k := 0; k < n%5; k++ {
					packed /= 3
				}
				tensors[i][n] = float32(int(packed%3)-1) * float32(t.Scale)
			}
		}
	}
	w1, b1, w2, b2 := tensors[0], tensors[1], tensors[2], tensors[3]
	for row := range 24 {
		for col := range 768 {
			want := float32(0)
			if row/8 == col/256 {
				want = w1[(row%8)*768+col%256]
			}
			if w1[row*768+col] != want {
				return errors.New("shared input matrix block/tie differs")
			}
		}
		if b1[row] != b1[row%8] {
			return errors.New("shared hidden bias tie differs")
		}
	}
	for mask := range 8 {
		for part := range 3 {
			for col := range 8 {
				if w2[mask*24+part*8+col] != w2[((mask>>part)&1)*24+col] {
					return errors.New("shared output bit composition differs")
				}
			}
		}
		if b2[mask] != 0 {
			return errors.New("shared output bias not zero")
		}
	}
	return nil
}
