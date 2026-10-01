// collect-own-joint-feedback records real retained failures with an own teacher.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
)

func main() {
	dataset := flag.String("dataset", "", "frozen source curriculum")
	teacher := flag.String("teacher", "", "fixed v1 own joint FP32 metadata")
	output := flag.String("output", "", "fresh ignored output directory")
	revision := flag.String("source-revision", "", "clean exact collector source")
	audit := flag.String("audit-report", "", "fresh zero-prediction audit of existing output")
	flag.Parse()
	var err error
	if *audit != "" {
		err = auditCollection(*dataset, *output, *audit)
	} else {
		err = collect(*dataset, *teacher, *output, *revision)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cleanSource(revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision || len(revision) != 40 {
		return errors.New("exact committed collector required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean collector required")
	}
	raw, err := os.ReadFile(jointfeedback.Protocol)
	if err != nil || jointcohort.SHA(raw) != jointfeedback.ProtocolSHA {
		return errors.New("frozen precollection protocol differs")
	}
	return nil
}

func collect(dataset, teacher, output, revision string) error {
	if err := cleanSource(revision); err != nil {
		return err
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) || output == "" {
		return errors.New("fresh collection directory required")
	}
	views, err := jointcohort.Load(dataset)
	if err != nil {
		return err
	}
	model, err := jointdecision.Load(teacher)
	if err != nil {
		return err
	}
	if model.MetadataSHA256() != jointfeedback.TeacherMetadata || model.WeightsSHA256() != jointfeedback.TeacherWeights || model.Variant() != "fp32" {
		return errors.New("fixed own teacher required")
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	pre := map[string]any{"schema": "gooo/own-joint-feedback-preexecution/v2", "source_revision": revision,
		"protocol_sha256": jointfeedback.ProtocolSHA, "protocol_revision": "d0c8fe0cd251c08ab5aa9fa23fdd2c2c9bf58bbc",
		"dataset_sha256": jointcohort.DatasetSHA, "teacher_metadata_sha256": jointfeedback.TeacherMetadata,
		"teacher_weights_sha256": jointfeedback.TeacherWeights, "planned_actual_teacher_sdk_sessions": 6144,
		"planned_training_function_views": 1536, "planned_initial_state_views_all_splits": 2304,
		"native_calls": 0, "student_optimizer_updates": 0, "ci_hint_supplied": false}
	if err = save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	return execute(output, views, model)
}

func execute(output string, views []jointcohort.View, model *jointdecision.Model) error {
	f, err := os.OpenFile(filepath.Join(output, "teacher-sessions.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriterSize(f, 32768)
	states, counts := map[string]*jointfeedback.State{}, totals{}
	started := time.Now()
	for _, v := range views {
		initial := jointfeedback.Initial(v)
		states[initial.ID] = &initial
		if v.Split != "train" {
			continue
		}
		for rotation := 0; rotation < 4; rotation++ {
			c, raw, err := one(v, model, rotation)
			if err != nil {
				_ = writer.Flush()
				return err
			}
			if _, err = writer.Write(append(raw, '\n')); err != nil {
				return err
			}
			jointfeedback.AddStates(states, v, c, raw, counts.Sessions)
			counts.add(c)
		}
		if counts.Sessions%512 == 0 {
			fmt.Printf("recorded teacher sessions=%d predictions=%d\n", counts.Sessions, counts.Predictions)
		}
	}
	if err = writer.Flush(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if counts.Sessions != 6144 {
		return errors.New("teacher session budget differs")
	}
	return finish(output, states, counts, time.Since(started))
}

func one(v jointcohort.View, model *jointdecision.Model, rotation int) (jointfeedback.Capture, []byte, error) {
	seed := fmt.Sprintf("own-feedback-teacher/v2/rotation-%d", rotation)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	search, _, progress, feedback, err := v.Prepared.SearchJointFeedbackBatches(ctx, model, v.Cases, 4, 1, seed, 3, nil)
	c := jointfeedback.Capture{Schema: "gooo/own-joint-feedback-teacher-capture/v2", ViewID: v.ID, SourceSHA: v.SourceSHA,
		JointSHA: jointcohort.SHA([]byte(v.JointInput)), Rotation: rotation, Seed: seed, WallNS: time.Since(started).Nanoseconds(),
		Search: search, Progress: progress, Feedback: feedback}
	if err != nil {
		return c, nil, err
	}
	if err = jointfeedback.Verify(v, c); err != nil {
		return c, nil, err
	}
	raw, err := json.Marshal(c)
	return c, raw, err
}

func save(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}

func finish(output string, states map[string]*jointfeedback.State, counts totals, duration time.Duration) error {
	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	f, err := os.OpenFile(filepath.Join(output, "states.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	writer := bufio.NewWriterSize(f, 32768)
	for _, id := range ids {
		raw, e := json.Marshal(states[id])
		if e != nil {
			f.Close()
			return e
		}
		if _, err = writer.Write(append(raw, '\n')); err != nil {
			f.Close()
			return err
		}
	}
	if err = writer.Flush(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	files := map[string]string{}
	for _, name := range []string{"preexecution.json", "teacher-sessions.jsonl", "states.jsonl"} {
		sha, err := fileSHA(filepath.Join(output, name))
		if err != nil {
			return err
		}
		files[name] = sha
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/own-joint-feedback-collection/v2",
		"status": "PASS", "actual_teacher": counts, "initial_state_views": 2304, "unique_continuation_states": len(states) - 2304,
		"student_state_rows": len(states), "files_sha256": files, "collection_and_immediate_audit_wall_ns": duration.Nanoseconds(),
		"native_calls": 0, "student_optimizer_updates": 0, "scope": "Actual SDK searches and evaluator case values, checked against ordinary independent Go arithmetic. Only train configurations collect teacher feedback; all splits retain initial source-bound inputs. This is not native compiled-Go execution or student quality evidence."})
}
