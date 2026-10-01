package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
)

func auditCollection(dataset, directory, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh collection audit required")
	}
	var report struct {
		Status        string            `json:"status"`
		Actual        totals            `json:"actual_teacher"`
		Initial       int               `json:"initial_state_views"`
		Continuations int               `json:"unique_continuation_states"`
		Rows          int               `json:"student_state_rows"`
		Files         map[string]string `json:"files_sha256"`
	}
	if err := readJSON(filepath.Join(directory, "report.json"), &report); err != nil {
		return err
	}
	if report.Status != "PASS" || report.Initial != 2304 || len(report.Files) != 3 {
		return errors.New("completed collection manifest required")
	}
	for _, name := range []string{"preexecution.json", "teacher-sessions.jsonl", "states.jsonl"} {
		sha, err := fileSHA(filepath.Join(directory, name))
		if err != nil || sha != report.Files[name] {
			return errors.New("collection file bytes differ")
		}
	}
	if err := auditPre(directory); err != nil {
		return err
	}
	views, err := jointcohort.Load(dataset)
	if err != nil {
		return err
	}
	states, actual, err := reconstruct(directory, views)
	if err != nil {
		return err
	}
	if actual != report.Actual || actual.Sessions != 6144 || report.Continuations != len(states)-2304 || report.Rows != len(states) || actual.Predictions != actual.InitialCalls+actual.FeedbackCalls {
		return errors.New("collection actual counts differ")
	}
	if err = auditStates(directory, states); err != nil {
		return err
	}
	sha, err := fileSHA(filepath.Join(directory, "report.json"))
	if err != nil {
		return err
	}
	return save(output, map[string]any{"schema": "gooo/own-joint-feedback-curriculum-audit/v2", "status": "PASS",
		"collection_report_sha256": sha, "files_sha256": report.Files, "dataset_sha256": jointcohort.DatasetSHA,
		"actual_teacher": actual, "student_state_rows": len(states), "initial_state_views": 2304,
		"unique_train_continuation_states": len(states) - 2304, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_updates": 0,
		"scope": "Independent ordered int64 oracle, source hashes, progress/feedback links and exact source-preserving runtime text. Reconstructs student rows and actual receipt origins from all 6144 captures. No operational inference or compiled-Go execution is performed by this audit."})
}

func auditPre(directory string) error {
	var pre struct {
		Protocol string `json:"protocol_sha256"`
		Dataset  string `json:"dataset_sha256"`
		Metadata string `json:"teacher_metadata_sha256"`
		Weights  string `json:"teacher_weights_sha256"`
		Sessions int    `json:"planned_actual_teacher_sdk_sessions"`
		Training int    `json:"planned_training_function_views"`
		Initial  int    `json:"planned_initial_state_views_all_splits"`
		CI       bool   `json:"ci_hint_supplied"`
	}
	if err := readJSON(filepath.Join(directory, "preexecution.json"), &pre); err != nil {
		return err
	}
	if pre.Protocol != jointfeedback.ProtocolSHA || pre.Dataset != jointcohort.DatasetSHA || pre.Metadata != jointfeedback.TeacherMetadata || pre.Weights != jointfeedback.TeacherWeights || pre.Sessions != 6144 || pre.Training != 1536 || pre.Initial != 2304 || pre.CI {
		return errors.New("frozen source/teacher/preexecution bounds differ")
	}
	return nil
}

func reconstruct(directory string, views []jointcohort.View) (map[string]*jointfeedback.State, totals, error) {
	states, byID := map[string]*jointfeedback.State{}, map[string]jointcohort.View{}
	for _, v := range views {
		initial := jointfeedback.Initial(v)
		states[initial.ID] = &initial
		byID[v.ID] = v
	}
	t, seen := totals{}, map[string]bool{}
	err := scan(filepath.Join(directory, "teacher-sessions.jsonl"), func(raw []byte) error {
		var c jointfeedback.Capture
		if err := json.Unmarshal(raw, &c); err != nil {
			return err
		}
		v, ok := byID[c.ViewID]
		key := fmt.Sprintf("%s/%d", c.ViewID, c.Rotation)
		if !ok || seen[key] {
			return errors.New("distinct training teacher rotation required")
		}
		seen[key] = true
		if err := jointfeedback.Verify(v, c); err != nil {
			return err
		}
		jointfeedback.AddStates(states, v, c, raw, t.Sessions)
		t.add(c)
		return nil
	})
	return states, t, err
}

func auditStates(directory string, states map[string]*jointfeedback.State) error {
	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	i := 0
	err := scan(filepath.Join(directory, "states.jsonl"), func(raw []byte) error {
		var s jointfeedback.State
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		if i >= len(ids) || s.ID != ids[i] || !reflect.DeepEqual(s, *states[s.ID]) {
			return errors.New("student state or actual receipt origin differs")
		}
		i++
		return nil
	})
	if err == nil && i != len(ids) {
		err = errors.New("student state denominator differs")
	}
	return err
}

func scan(name string, visit func([]byte) error) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 32768), 1<<20)
	for s.Scan() {
		if err = decision.RejectDuplicateJSONKeys(s.Bytes()); err != nil {
			return err
		}
		if err = visit(s.Bytes()); err != nil {
			return err
		}
	}
	return s.Err()
}

func readJSON(name string, value any) error {
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}
