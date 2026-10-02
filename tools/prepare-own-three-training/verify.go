package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func readJSON(name string, v any) error {
	p, err := threestudent.FilePin(name)
	if err != nil || p.Bytes > 1<<20 {
		return errors.New("bounded regular JSON required")
	}
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func verifyPrepared(dataset, teacher, audit, directory, output, revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact verifier source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean verifier source required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh independent verification output required")
	}
	var m struct {
		Schema   string                      `json:"schema"`
		Status   string                      `json:"status"`
		Source   string                      `json:"source_revision"`
		Files    map[string]threestudent.Pin `json:"files"`
		Dim      int                         `json:"feature_dim"`
		Hidden   int                         `json:"hidden_dim"`
		Labels   int                         `json:"label_count"`
		Features string                      `json:"feature_version"`
		Rows     int                         `json:"student_state_rows"`
		Counts   map[string]int              `json:"rows_by_split_phase"`
		Groups   map[string]int              `json:"function_groups_by_split"`
		Audit    string                      `json:"teacher_audit_sha256"`
		Initial  int                         `json:"initial_seed"`
		Shuffle  int                         `json:"shuffle_seed"`
		Steps    int                         `json:"planned_optimizer_steps"`
		Prior    int64                       `json:"prior_retained_raw_bytes"`
		Cap      int64                       `json:"raw_cap_bytes"`
	}
	if err = readJSON(filepath.Join(directory, "manifest.json"), &m); err != nil {
		return err
	}
	if m.Schema != "gooo/own-three-training-inputs/v1" || m.Status != "PREPARED_PENDING_INDEPENDENT_REPLAY" || len(m.Source) != 40 || m.Dim != 768 || m.Hidden != 24 || m.Labels != 8 || m.Features != jointdecision.ThreeFeatureVersion || m.Rows != 10739 || m.Audit != threestudent.TeacherAuditSHA || m.Initial != threestudent.InitialSeed || m.Shuffle != threestudent.ShuffleSeed || m.Steps != 4800 || m.Cap != threestudent.RawCap || len(m.Files) != 5 {
		return errors.New("prepared frozen ABI/optimizer budget differs")
	}
	for _, name := range []string{"preexecution.json", "initial-fp32.bin", "features-f32le.bin", "rows.jsonl", "parity-inputs.json"} {
		p, err := threestudent.FilePin(filepath.Join(directory, name))
		if err != nil || p != m.Files[name] {
			return errors.New("prepared file pin differs")
		}
	}
	if m.Files["initial-fp32.bin"].SHA != threecohort.SHA(threestudent.InitialWeights()) || m.Files["initial-fp32.bin"].Bytes != 74624 || m.Files["features-f32le.bin"].Bytes != 10739*768*4 {
		return errors.New("fresh independent initializer or matrix shape differs")
	}
	var pre struct {
		Source     string                      `json:"source_revision"`
		Go         string                      `json:"go"`
		Protocol   string                      `json:"protocol_sha256"`
		Dataset    string                      `json:"source_dataset_sha256"`
		States     string                      `json:"student_states_sha256"`
		Audit      string                      `json:"teacher_audit_sha256"`
		Report     string                      `json:"teacher_report_sha256"`
		InitialSHA string                      `json:"initial_state_sha256"`
		Initial    int                         `json:"initial_seed"`
		Shuffle    int                         `json:"shuffle_seed"`
		Steps      int                         `json:"planned_optimizer_steps"`
		Prior      int64                       `json:"prior_retained_raw_bytes"`
		Files      map[string]threestudent.Pin `json:"prior_raw_files"`
		Cap        int64                       `json:"whole_study_raw_cap_bytes"`
	}
	if err = readJSON(filepath.Join(directory, "preexecution.json"), &pre); err != nil {
		return err
	}
	if pre.Source != m.Source || pre.Go != "go1.27.1" || pre.Protocol != threefeedback.ProtocolSHA || pre.Dataset != threecohort.DatasetSHA || pre.States != threestudent.StatesSHA || pre.Audit != threestudent.TeacherAuditSHA || pre.Report != threestudent.TeacherReportSHA || pre.InitialSHA != m.Files["initial-fp32.bin"].SHA || pre.Initial != m.Initial || pre.Shuffle != m.Shuffle || pre.Steps != m.Steps || pre.Prior != m.Prior || pre.Cap != m.Cap {
		return errors.New("preparation source/preexecution pin differs")
	}
	var priorBytes int64
	for name, pin := range pre.Files {
		if filepath.IsAbs(name) || filepath.ToSlash(filepath.Clean(name)) != name || strings.HasPrefix(name, "../") || !strings.HasPrefix(name, "own-three-") {
			return errors.New("invalid prior raw filename")
		}
		p, err := threestudent.FilePin(filepath.Join(filepath.Dir(directory), filepath.FromSlash(name)))
		if err != nil || p != pin {
			return errors.New("prior raw bytes changed")
		}
		priorBytes += p.Bytes
	}
	if priorBytes != m.Prior {
		return errors.New("prior evidence denominator differs")
	}
	states, err := threestudent.Load(dataset, teacher, audit)
	if err != nil {
		return err
	}
	rows, err := threestudent.Rows(states)
	if err != nil {
		return err
	}
	x, err := os.Open(filepath.Join(directory, "features-f32le.bin"))
	if err != nil {
		return err
	}
	defer x.Close()
	j, err := os.Open(filepath.Join(directory, "rows.jsonl"))
	if err != nil {
		return err
	}
	defer j.Close()
	scanner := bufio.NewScanner(j)
	scanner.Buffer(make([]byte, 32768), 1<<20)
	counts := map[string]int{}
	groups := map[string]map[string]bool{"train": {}, "calibration": {}, "development": {}}
	for i, s := range states {
		if !scanner.Scan() {
			return errors.New("missing metadata row")
		}
		var actual threestudent.Row
		if err = threecohort.Decode(scanner.Bytes(), &actual); err != nil || !reflect.DeepEqual(actual, rows[i]) {
			return errors.New("row identity or group/language/state weighting differs")
		}
		var f [768]float32
		if err = jointdecision.FeaturesIntoThree(s.Text, &f); err != nil {
			return err
		}
		var expected, raw [768 * 4]byte
		for k, v := range f {
			binary.LittleEndian.PutUint32(expected[k*4:], math.Float32bits(v))
		}
		if _, err = io.ReadFull(x, raw[:]); err != nil || !bytes.Equal(raw[:], expected[:]) {
			return errors.New("source-bound feature bytes differ")
		}
		counts[s.Split+"/"+s.Phase]++
		groups[s.Split][s.Group] = true
	}
	if scanner.Scan() || scanner.Err() != nil {
		return errors.New("extra/invalid metadata rows")
	}
	var one [1]byte
	if n, err := x.Read(one[:]); n != 0 || err != io.EOF {
		return errors.New("extra feature bytes")
	}
	groupCounts := map[string]int{}
	for split, g := range groups {
		groupCounts[split] = len(g)
	}
	if !reflect.DeepEqual(counts, m.Counts) || !reflect.DeepEqual(groupCounts, m.Groups) {
		return errors.New("split/group counts differ")
	}
	var parity struct {
		Rows []struct {
			Index int    `json:"feature_row_index"`
			ID    string `json:"state_id"`
			Phase string `json:"phase"`
			Text  string `json:"text"`
			SHA   string `json:"input_sha256"`
		}
	}
	if err = readJSON(filepath.Join(directory, "parity-inputs.json"), &parity); err != nil || len(parity.Rows) != 48 {
		return errors.New("48 frozen parity inputs required")
	}
	for i, p := range parity.Rows {
		phase, split, count := "initial", "development", 32
		offset := i
		if i >= 32 {
			phase, split, count, offset = "feedback", "train", 16, i-32
		}
		indices := []int{}
		for n, s := range states {
			if s.Phase == phase && s.Split == split {
				indices = append(indices, n)
			}
		}
		n := indices[offset*(len(indices)-1)/(count-1)]
		s := states[n]
		if p.Index != n || p.ID != s.ID || p.Phase != s.Phase || p.Text != s.Text || p.SHA != s.InputSHA {
			return errors.New("source-bound parity selection differs")
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 7 {
		return errors.New("closed prepared phase inventory differs")
	}
	var phaseBytes int64
	for _, entry := range entries {
		p, err := threestudent.FilePin(filepath.Join(directory, entry.Name()))
		if err != nil {
			return err
		}
		phaseBytes += p.Bytes
	}
	if priorBytes+phaseBytes > m.Cap {
		return errors.New("whole-study evidence cap exceeded")
	}
	var attempt struct {
		Status  string `json:"status"`
		Error   string `json:"error"`
		Updates int    `json:"student_optimizer_updates"`
		Calls   int    `json:"model_predictions"`
		Native  int    `json:"native_calls"`
	}
	if err = readJSON(filepath.Join(directory, "preparation-attempt.json"), &attempt); err != nil || attempt.Status != m.Status || attempt.Error != "" || attempt.Updates != 0 || attempt.Calls != 0 || attempt.Native != 0 {
		return errors.New("complete zero-learning preparation attempt required")
	}
	mp, err := threestudent.FilePin(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return err
	}
	used := priorBytes + phaseBytes
	return saveJSON(output, map[string]any{"schema": "gooo/own-three-training-input-independent-replay/v1", "status": "PASS", "auditor_source_revision": revision, "preparation_source_revision": m.Source, "manifest_sha256": mp.SHA, "initial_state_sha256": m.Files["initial-fp32.bin"].SHA, "teacher_audit_sha256": threestudent.TeacherAuditSHA, "source_dataset_sha256": threecohort.DatasetSHA, "student_states_sha256": threestudent.StatesSHA, "files": m.Files, "student_state_rows": 10739, "feature_float32_values_recomputed": 10739 * 768, "function_groups_by_split": groupCounts, "whole_study_retained_bytes_before_replay_receipt": used, "model_predictions": 0, "optimizer_updates": 0, "native_calls": 0, "scope": "Byte-exact independent recomputation of every complete Go source feature, balanced group/language/state weight, finite target, fresh common initializer, original raw evidence pin and fixed parity input. This is not model training or student behavior."}, &used)
}
