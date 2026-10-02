// prepare-own-three-training exports bounded fixed-array Go features, equal
// function/language weights and a new common random initializer before MPS.
package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func main() {
	dataset := flag.String("dataset", "", "frozen actual native source dataset")
	teacher := flag.String("teacher-curriculum", "", "complete own-teacher observation directory")
	audit := flag.String("teacher-audit", "publication/own-three-choice-teacher-audit-20261002.json", "frozen independent zero-inference audit")
	output := flag.String("output", "", "fresh ignored own-three training-input directory")
	revision := flag.String("source-revision", "", "exact clean published source")
	verify := flag.String("verify-report", "", "verify prepared output without inference or learning")
	flag.Parse()
	var err error
	if *verify != "" {
		err = verifyPrepared(*dataset, *teacher, *audit, *output, *verify, *revision)
	} else {
		err = prepare(*dataset, *teacher, *audit, *output, *revision)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepare(dataset, teacher, audit, output, revision string) (failure error) {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision || runtime.Version() != "go1.27.1" {
		return errors.New("exact clean Go 1.27.1 preparation source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("dirty training preparation source")
	}
	protocol, err := os.ReadFile(threefeedback.Protocol)
	if err != nil || threecohort.SHA(protocol) != threefeedback.ProtocolSHA {
		return errors.New("immutable three-choice protocol required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(output)
	if err != nil || filepath.Dir(absolute) != filepath.Join(cwd, "runs") || !strings.HasPrefix(filepath.Base(absolute), "own-three-") {
		return errors.New("fresh own-three phase directly under runs required")
	}
	if _, err = os.Lstat(absolute); !os.IsNotExist(err) {
		return errors.New("training inputs must not overwrite an earlier phase")
	}
	priorFiles, used, err := inventory(filepath.Join(cwd, "runs"))
	if err != nil {
		return err
	}
	states, err := threestudent.Load(dataset, teacher, audit)
	if err != nil {
		return err
	}
	rows, err := threestudent.Rows(states)
	if err != nil {
		return err
	}
	initial := threestudent.InitialWeights()
	if err = os.Mkdir(absolute, 0755); err != nil {
		return err
	}
	startBytes := used
	defer func() {
		status, message := "PREPARED_PENDING_INDEPENDENT_REPLAY", ""
		if failure != nil {
			status, message = "FAILED_PREFIX_RETAINED", failure.Error()
		}
		if err := saveJSON(filepath.Join(absolute, "preparation-attempt.json"), map[string]any{"schema": "gooo/own-three-training-preparation-attempt/v1", "status": status, "error": message, "student_optimizer_updates": 0, "model_predictions": 0, "native_calls": 0, "prior_retained_raw_bytes": startBytes, "retained_new_bytes_before_attempt": used - startBytes}, &used); err != nil && failure == nil {
			failure = err
		}
	}()
	pre := map[string]any{"schema": "gooo/own-three-training-input-preexecution/v1", "source_revision": revision, "go": runtime.Version(), "protocol_sha256": threefeedback.ProtocolSHA, "source_dataset_sha256": threecohort.DatasetSHA, "student_states_sha256": threestudent.StatesSHA, "teacher_audit_sha256": threestudent.TeacherAuditSHA, "teacher_report_sha256": threestudent.TeacherReportSHA, "prior_raw_files": priorFiles, "prior_retained_raw_bytes": startBytes, "whole_study_raw_cap_bytes": threestudent.RawCap, "initial_seed": threestudent.InitialSeed, "shuffle_seed": threestudent.ShuffleSeed, "initial_state_sha256": threecohort.SHA(initial), "initialization": "Fresh Go math/rand.NewSource(seed) float32 uniform [-1/sqrt(fan_in),1/sqrt(fan_in)], w1,b1,w2,b2; no pretrained/Laya/teacher/prior student weights.", "planned_optimizer_steps": 4800, "scope": "Go data preparation only. No optimizer, model prediction or native execution."}
	if err = saveJSON(filepath.Join(absolute, "preexecution.json"), pre, &used); err != nil {
		return err
	}
	if err = saveRaw(filepath.Join(absolute, "initial-fp32.bin"), initial, &used); err != nil {
		return err
	}
	x, err := os.OpenFile(filepath.Join(absolute, "features-f32le.bin"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer x.Close()
	j, err := os.OpenFile(filepath.Join(absolute, "rows.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer j.Close()
	counts := map[string]int{}
	groups := map[string]map[string]bool{"train": {}, "calibration": {}, "development": {}}
	for i, s := range states {
		var f [jointdecision.ThreeFeatureDim]float32
		if err = jointdecision.FeaturesIntoThree(s.Text, &f); err != nil {
			return err
		}
		var raw [jointdecision.ThreeFeatureDim * 4]byte
		for k, v := range f {
			binary.LittleEndian.PutUint32(raw[k*4:], math.Float32bits(v))
		}
		if err = writeBounded(x, raw[:], &used); err != nil {
			return err
		}
		rowRaw, err := json.Marshal(rows[i])
		if err != nil {
			return err
		}
		if err = writeBounded(j, append(rowRaw, '\n'), &used); err != nil {
			return err
		}
		counts[s.Split+"/"+s.Phase]++
		groups[s.Split][s.Group] = true
	}
	if err = x.Sync(); err != nil {
		return err
	}
	if err = j.Sync(); err != nil {
		return err
	}
	if err = x.Close(); err != nil {
		return err
	}
	if err = j.Close(); err != nil {
		return err
	}
	if len(groups["train"]) != 1024 || len(groups["calibration"]) != 256 || len(groups["development"]) != 256 || counts["train/initial"] != 2048 || counts["train/feedback"] != 7667 || counts["calibration/initial"] != 512 || counts["development/initial"] != 512 {
		return errors.New("disjoint source-group denominator differs")
	}
	parity := make([]any, 0, 48)
	for _, selection := range []struct {
		split, phase string
		count        int
	}{{"development", "initial", 32}, {"train", "feedback", 16}} {
		indices := []int{}
		for i, s := range states {
			if s.Split == selection.split && s.Phase == selection.phase {
				indices = append(indices, i)
			}
		}
		for i := range selection.count {
			n := indices[i*(len(indices)-1)/(selection.count-1)]
			s := states[n]
			parity = append(parity, map[string]any{"feature_row_index": n, "state_id": s.ID, "phase": s.Phase, "text": s.Text, "input_sha256": s.InputSHA})
		}
	}
	if err = saveJSON(filepath.Join(absolute, "parity-inputs.json"), map[string]any{"rows": parity}, &used); err != nil {
		return err
	}
	pins := map[string]threestudent.Pin{}
	for _, name := range []string{"preexecution.json", "initial-fp32.bin", "features-f32le.bin", "rows.jsonl", "parity-inputs.json"} {
		p, err := threestudent.FilePin(filepath.Join(absolute, name))
		if err != nil {
			return err
		}
		pins[name] = p
	}
	if err = saveJSON(filepath.Join(absolute, "manifest.json"), map[string]any{"schema": "gooo/own-three-training-inputs/v1", "status": "PREPARED_PENDING_INDEPENDENT_REPLAY", "source_revision": revision, "files": pins, "feature_dim": 768, "hidden_dim": 24, "label_count": 8, "feature_version": jointdecision.ThreeFeatureVersion, "feature_matrix_layout": "row-major little-endian float32; each row preserves three complete scaled source-v3 parts", "student_state_rows": len(rows), "rows_by_split_phase": counts, "function_groups_by_split": map[string]int{"train": 1024, "calibration": 256, "development": 256}, "teacher_audit_sha256": threestudent.TeacherAuditSHA, "initial_seed": threestudent.InitialSeed, "shuffle_seed": threestudent.ShuffleSeed, "planned_optimizer_steps": 4800, "prior_retained_raw_bytes": startBytes, "raw_cap_bytes": threestudent.RawCap, "scope": "Source-bound features and balanced finite passing-mask targets. Development rows are retained for parity/evaluation only. No model or optimizer calls."}, &used); err != nil {
		return err
	}
	fmt.Printf("PREPARED: %d states, 1024 train / 256 calibration / 256 development groups; new initializer %s; zero model/optimizer/native calls\n", len(rows), pins["initial-fp32.bin"].SHA)
	return nil
}
