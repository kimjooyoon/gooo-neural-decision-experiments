package main

import (
	"bufio"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type epoch struct {
	Epoch int                `json:"epoch"`
	Steps int                `json:"actual_stage_updates"`
	Train float64            `json:"train_objective"`
	NLL   map[string]float64 `json:"calibration_passing_set_nll_by_temperature"`
}
type stageReport struct {
	Steps       int     `json:"optimizer_steps"`
	Epochs      int     `json:"epochs"`
	Groups      int     `json:"training_function_groups"`
	Rows        int     `json:"training_weighted_state_rows"`
	InitialSHA  string  `json:"initial_state_sha256"`
	SelectedSHA string  `json:"selected_checkpoint_sha256"`
	Selected    int     `json:"selected_epoch"`
	Temperature float64 `json:"selected_temperature"`
	NLL         float64 `json:"selected_calibration_passing_set_nll"`
	Selection   string  `json:"selection"`
	Wall        float64 `json:"loop_wall_seconds"`
	CPU         float64 `json:"process_cpu_seconds_during_loop"`
	CPUPercent  float64 `json:"process_cpu_percent_one_core_normalization"`
	RSS         int64   `json:"process_lifetime_peak_rss_bytes"`
	MPS         int64   `json:"mps_allocated_peak_sampled_bytes"`
	Driver      int64   `json:"mps_driver_peak_sampled_bytes"`
	History     []epoch `json:"history"`
}

func verifyPre(name string) error {
	var p struct {
		Schema           string            `json:"schema"`
		Source           string            `json:"source_revision"`
		Sources          map[string]string `json:"sources_sha256"`
		Protocol         string            `json:"protocol_sha256"`
		States           string            `json:"student_states_sha256"`
		Audit            string            `json:"teacher_audit_sha256"`
		InitialSHA       string            `json:"initial_state_sha256"`
		Initial          int               `json:"initial_seed"`
		Shuffle          int               `json:"shuffle_seed"`
		Steps            int               `json:"planned_optimizer_steps"`
		Arms             []string          `json:"arms"`
		Device           string            `json:"device"`
		Optimizer        string            `json:"optimizer"`
		LR               float64           `json:"learning_rate"`
		Decay            float64           `json:"weight_decay"`
		Temperatures     []float64         `json:"temperatures"`
		Prepared         string            `json:"prepared_manifest_sha256"`
		PreparationAudit string            `json:"preparation_audit_sha256"`
		Prior            int64             `json:"prior_retained_raw_bytes"`
		Cap              int64             `json:"whole_study_raw_cap_bytes"`
		Promoted         bool              `json:"default_model_promoted"`
	}
	if err := read(name, &p); err != nil {
		return err
	}
	if p.Schema != "gooo/own-three-choice-training-preexecution/v1" || len(p.Source) != 40 || p.Protocol != threefeedback.ProtocolSHA || p.States != threestudent.StatesSHA || p.Audit != threestudent.TeacherAuditSHA || p.InitialSHA != threestudent.InitialSHA || p.Initial != threestudent.InitialSeed || p.Shuffle != threestudent.ShuffleSeed || p.Steps != 4800 || !sameJSON(p.Arms, arms[:]) || p.Device != "mps" || p.Optimizer != "AdamW" || p.LR != .001 || p.Decay != .01 || !sameJSON(p.Temperatures, []float64{.5, 1, 2, 4}) || len(p.Prepared) != 64 || len(p.PreparationAudit) != 64 || p.Prior <= 0 || p.Prior >= threestudent.RawCap || p.Cap != threestudent.RawCap || p.Promoted {
		return errors.New("fixed independent fresh training preexecution differs")
	}
	for _, n := range []string{"train_own_three_feedback_v1.py", "train_pilot_v2.py"} {
		pin, err := threestudent.FilePin(filepath.Join("training", n))
		if err != nil || pin.SHA != p.Sources[n] {
			return errors.New("published optimizer source differs")
		}
	}
	return nil
}

func verifyStage(directory, arm, stage string, r stageReport) error {
	rows := 2048
	if arm == "set-feedback" || arm == "dense" || arm == "shared-local" {
		rows = 9715
	}
	return verifyStageRows(directory, arm, stage, r, rows)
}

func verifyStageRows(directory, arm, stage string, r stageReport, rows int) error {
	if r.Steps != 800 || r.Epochs != 100 || r.Groups != 1024 || r.Rows != rows || len(r.History) != 100 || len(r.InitialSHA) != 64 || len(r.SelectedSHA) != 64 || !validTemperature(r.Temperature) || !finite(r.NLL) || r.Wall <= 0 || !finite(r.Wall) || r.CPU < 0 || !finite(r.CPU) || math.Abs(r.CPUPercent-r.CPU/r.Wall*100) > 1e-9 || r.RSS <= 0 || r.MPS <= 0 || r.Driver < r.MPS {
		return errors.New("stage counts/cost scopes differ")
	}
	f, err := os.Open(filepath.Join(directory, "optimizer-updates.jsonl"))
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 32768), 1<<20)
	var objectives [100]float64
	actual := 0
	for scanner.Scan() {
		var s struct {
			Arm       string  `json:"arm"`
			Stage     string  `json:"stage"`
			Epoch     int     `json:"epoch"`
			Batch     int     `json:"batch"`
			Groups    int     `json:"function_groups"`
			Steps     int     `json:"actual_stage_updates"`
			Objective float64 `json:"objective"`
		}
		if err = threecohort.Decode(scanner.Bytes(), &s); err != nil {
			return err
		}
		if actual >= 800 || s.Arm != arm || s.Stage != stage || s.Epoch != actual/8+1 || s.Batch != actual%8+1 || s.Groups != 128 || s.Steps != actual+1 || !finite(s.Objective) {
			return errors.New("actual stage update prefix differs")
		}
		objectives[actual/8] += s.Objective / 8
		actual++
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	if actual != 800 {
		return errors.New("fixed actual update budget incomplete")
	}
	bestLoss, bestEpoch, bestTemperature := math.Inf(1), 0, 0.
	for i, e := range r.History {
		if e.Epoch != i+1 || e.Steps != (i+1)*8 || len(e.NLL) != 4 || !finite(e.Train) || math.Abs(e.Train-objectives[i]) > 1e-12 {
			return errors.New("recorded epoch counts/objective differ")
		}
		var duplicate epoch
		if err = read(filepath.Join(directory, "epoch-"+pad3(i+1)+".json"), &duplicate); err != nil || !sameJSON(duplicate, e) {
			return errors.New("immutable epoch record differs")
		}
		for _, temperature := range []float64{.5, 1, 2, 4} {
			key := strconv.FormatFloat(temperature, 'f', 1, 64)
			loss, ok := e.NLL[key]
			if !ok || !finite(loss) {
				return errors.New("complete finite temperature grid required")
			}
			if loss < bestLoss || loss == bestLoss && (i+1 < bestEpoch || i+1 == bestEpoch && temperature < bestTemperature) {
				bestLoss, bestEpoch, bestTemperature = loss, i+1, temperature
			}
		}
	}
	if r.NLL != bestLoss || r.Selected != bestEpoch || r.Temperature != bestTemperature {
		return errors.New("joint calibration epoch/temperature selector differs")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 101 {
		return errors.New("closed stage receipt inventory differs")
	}
	for _, entry := range entries {
		if _, err = threestudent.FilePin(filepath.Join(directory, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func pad3(v int) string { return strconv.FormatInt(int64(1000+v), 10)[1:] }
