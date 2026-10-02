package main

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

func audit(dataset, directory, output string) error {
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh offline audit output required")
	}
	var report collectionReport
	if err := readJSON(filepath.Join(directory, "report.json"), &report); err != nil {
		return err
	}
	if report.Schema != "gooo/own-three-choice-teacher-collection/v1" || report.Status != "COMPLETE_PENDING_INDEPENDENT_AUDIT" || report.Initial != 3072 || len(report.Files) != 2 || report.WallNS <= 0 || report.CPUNS < 0 || math.Abs(report.CPUPercent-float64(report.CPUNS)/float64(report.WallNS)*100) > 1e-9 {
		return errors.New("complete bounded teacher report required")
	}
	var pre preexecution
	if err := readJSON(filepath.Join(directory, "preexecution.json"), &pre); err != nil {
		return err
	}
	preSHA, err := fileSHA(filepath.Join(directory, "preexecution.json"))
	if err != nil || preSHA != report.PreSHA {
		return errors.New("preexecution byte pin differs")
	}
	if pre.Schema != "gooo/own-three-choice-teacher-preexecution/v1" || !hex40.MatchString(pre.Source) || pre.Go != "go1.27.1" || pre.Protocol != threefeedback.ProtocolSHA || pre.ProtocolRevision != threefeedback.ProtocolRevision || pre.FreezeRevision != freezeRevision || pre.Dataset != threecohort.DatasetSHA || pre.Manifest != threecohort.ManifestSHA || pre.Audit != threecohort.AuditSHA || pre.Metadata != threefeedback.TeacherMetadata || pre.Weights != threefeedback.TeacherWeights || pre.Sessions != 4096 || pre.Training != 2048 || pre.Initial != 3072 || pre.Seeds != [2]string{"three-source-path-teacher-0", "three-source-path-teacher-1"} || pre.Budget != 8 || pre.Step != 1 || pre.Rounds != 7 || pre.CI || pre.Cap != rawCap || pre.LineCap != lineCap || pre.PriorBytes != report.PriorBytes {
		return errors.New("frozen independent teacher/source/study protocol differs")
	}
	var priorBytes int64
	for name, pin := range pre.PriorFiles {
		if filepath.IsAbs(name) || filepath.ToSlash(filepath.Clean(name)) != name || strings.HasPrefix(name, "../") || !strings.HasPrefix(name, "own-three-") {
			return errors.New("invalid prior phase file name")
		}
		path := filepath.Join(filepath.Dir(directory), filepath.FromSlash(name))
		s, err := os.Lstat(path)
		if err != nil || !s.Mode().IsRegular() || s.Size() != pin.Bytes || pin.Lines != 0 {
			return errors.New("prior phase file bound differs")
		}
		sha, err := fileSHA(path)
		if err != nil || sha != pin.SHA {
			return errors.New("prior frozen/failed phase file bytes differ")
		}
		priorBytes += pin.Bytes
	}
	if priorBytes != pre.PriorBytes || priorBytes <= 0 {
		return errors.New("whole-study prior evidence denominator differs")
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	states := map[string]*threefeedback.State{}
	training := make([]threecohort.View, 0, 2048)
	for _, v := range views {
		initial := threefeedback.Initial(v)
		states[initial.ID] = &initial
		if v.Split == "train" {
			training = append(training, v)
		}
	}
	actual := totals{}
	latency := [4096]int64{}
	sessionsPin, err := scan(filepath.Join(directory, "teacher-sessions.jsonl"), func(raw []byte) error {
		if actual.Sessions >= 4096 {
			return errors.New("unexpected extra teacher session")
		}
		var c threefeedback.Capture
		if err := threecohort.Decode(raw, &c); err != nil {
			return err
		}
		v := training[actual.Sessions/2]
		if c.ViewID != v.ID || c.SeedIndex != actual.Sessions%2 {
			return errors.New("distinct fixed ordered training sessions required")
		}
		if err := threefeedback.Verify(v, c); err != nil {
			return err
		}
		threefeedback.AddStates(states, v, c, raw, actual.Sessions)
		latency[actual.Sessions] = c.WallNS
		actual.add(c)
		return nil
	})
	if err != nil {
		return err
	}
	if sessionsPin != report.Files["teacher-sessions.jsonl"] || actual != report.Actual || actual.Sessions != 4096 || actual.Complete != 4096 || actual.InitialCalls != 12288 || actual.Predictions != actual.InitialCalls+actual.FeedbackCalls || report.Rows != len(states) || report.Continuations != len(states)-3072 {
		return errors.New("actual teacher or reconstructed student-state denominator differs")
	}
	slices.Sort(latency[:])
	if report.MedianNS != latency[2048] || report.P95NS != latency[(4096*95+99)/100-1] {
		return errors.New("recorded session latency summary differs")
	}
	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	row := 0
	statesPin, err := scan(filepath.Join(directory, "states.jsonl"), func(raw []byte) error {
		var state threefeedback.State
		if err := threecohort.Decode(raw, &state); err != nil {
			return err
		}
		if row >= len(ids) || state.ID != ids[row] || !reflect.DeepEqual(state, *states[state.ID]) {
			return errors.New("derived full input, split, target or actual origin differs")
		}
		row++
		return nil
	})
	if err != nil {
		return err
	}
	if row != len(states) || statesPin != report.Files["states.jsonl"] || report.NewRawBytes != sessionsPin.Bytes+statesPin.Bytes {
		return errors.New("student state or actual raw byte denominator differs")
	}
	var attempt struct {
		Schema   string  `json:"schema"`
		Status   string  `json:"status"`
		Error    string  `json:"error"`
		Actual   totals  `json:"actual_teacher"`
		Journal  filePin `json:"teacher_journal"`
		RawBytes int64   `json:"total_retained_raw_journal_bytes_including_prior_phases"`
		Updates  int     `json:"student_optimizer_updates"`
		Native   int     `json:"native_calls"`
		Scope    string  `json:"scope"`
	}
	if err = readJSON(filepath.Join(directory, "collection-attempt.json"), &attempt); err != nil {
		return err
	}
	if attempt.Schema != "gooo/own-three-choice-teacher-attempt/v1" || attempt.Status != report.Status || attempt.Error != "" || attempt.Actual != actual || attempt.Journal != sessionsPin || attempt.RawBytes != pre.PriorBytes+report.NewRawBytes || attempt.Updates != 0 || attempt.Native != 0 {
		return errors.New("terminal attempt counters differ")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	allowed := map[string]bool{"preexecution.json": true, "teacher-sessions.jsonl": true, "states.jsonl": true, "report.json": true, "collection-attempt.json": true}
	var phaseBytes int64
	if len(entries) != len(allowed) {
		return errors.New("closed raw phase inventory differs")
	}
	for _, entry := range entries {
		s, err := entry.Info()
		if err != nil || !allowed[entry.Name()] || !s.Mode().IsRegular() {
			return errors.New("unexpected raw phase file")
		}
		phaseBytes += s.Size()
	}
	if pre.PriorBytes+phaseBytes > rawCap {
		return errors.New("entire-study retained evidence cap exceeded")
	}
	reportSHA, err := fileSHA(filepath.Join(directory, "report.json"))
	if err != nil {
		return err
	}
	return save(output, map[string]any{"schema": "gooo/own-three-choice-teacher-independent-audit/v1", "status": "PASS", "collection_source_revision": pre.Source, "collection_report_sha256": reportSHA, "preexecution_sha256": preSHA, "raw_files": report.Files, "actual_teacher": actual, "initial_state_views": 3072, "unique_train_continuation_states": len(states) - 3072, "student_state_rows": len(states), "retained_bytes_including_prior_phases": pre.PriorBytes + phaseBytes, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_updates": 0, "scope": "Offline reconstruction of actual ordered int64 results, eight-mask frontier from captured distributions, seeded initial proposals, immutable source/test/model pins, progress/feedback chains, fixed coordinates, zero-call declines and exact derived full three-choice contexts/origins. It does not re-run model predictions, prove unobserved model outputs, execute native compiled Go, train a student or establish arbitrary-language completeness."})
}
