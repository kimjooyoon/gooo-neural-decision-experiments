// collect-own-three-feedback retains actual frozen independent-teacher sessions.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

const freezeRevision = "99a83c96c6c68ae629957773ef526fcc4089e99d"

type preexecution struct {
	Schema           string             `json:"schema"`
	Source           string             `json:"source_revision"`
	Go               string             `json:"go_version"`
	Protocol         string             `json:"protocol_sha256"`
	ProtocolRevision string             `json:"protocol_revision"`
	FreezeRevision   string             `json:"curriculum_freeze_revision"`
	Dataset          string             `json:"dataset_sha256"`
	Manifest         string             `json:"curriculum_manifest_sha256"`
	Audit            string             `json:"curriculum_audit_sha256"`
	Metadata         string             `json:"teacher_metadata_sha256"`
	Weights          string             `json:"teacher_weights_sha256"`
	Sessions         int                `json:"planned_actual_teacher_sdk_sessions"`
	Training         int                `json:"planned_training_function_views"`
	Initial          int                `json:"planned_initial_views_all_splits"`
	Seeds            [2]string          `json:"seed_strings"`
	Budget           int                `json:"candidate_budget"`
	Step             int                `json:"candidates_per_advance"`
	Rounds           int                `json:"maximum_feedback_rounds"`
	CI               bool               `json:"ci_hint_supplied"`
	PriorBytes       int64              `json:"existing_three_choice_evidence_bytes"`
	PriorFiles       map[string]filePin `json:"existing_three_choice_evidence_files"`
	Cap              int64              `json:"entire_study_raw_evidence_cap_bytes"`
	LineCap          int                `json:"individual_jsonl_line_cap_bytes"`
	Scope            string             `json:"scope"`
}

func main() {
	dataset := flag.String("dataset", "", "exact frozen source curriculum")
	teacher := flag.String("teacher", "", "frozen v1 independent own FP32 metadata")
	output := flag.String("output", "", "fresh ignored own-three-* directory under runs")
	revision := flag.String("source-revision", "", "clean exact committed collector")
	auditReport := flag.String("audit-report", "", "fresh offline audit of existing collection")
	flag.Parse()
	if *dataset == "" || *output == "" || flag.NArg() != 0 || (*auditReport != "" && (*teacher != "" || *revision != "")) || (*auditReport == "" && (*teacher == "" || *revision == "")) {
		fmt.Fprintln(os.Stderr, "required: dataset, output, and either teacher/source-revision or audit-report")
		os.Exit(2)
	}
	var err error
	if *auditReport != "" {
		err = audit(*dataset, *output, *auditReport)
	} else {
		err = collect(*dataset, *teacher, *output, *revision)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "collect-own-three-feedback:", err)
		os.Exit(1)
	}
}

func cleanSource(revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision || !hex40.MatchString(revision) || runtime.Version() != "go1.27.1" {
		return errors.New("clean exact Go 1.27.1 committed collector required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("dirty source cannot collect observations")
	}
	for _, p := range []struct{ path, sha, revision string }{
		{threefeedback.Protocol, threefeedback.ProtocolSHA, threefeedback.ProtocolRevision},
		{"publication/own-three-choice-curriculum-manifest-20261002.json", threecohort.ManifestSHA, freezeRevision},
		{"publication/own-three-choice-curriculum-audit-20261002.json", threecohort.AuditSHA, freezeRevision},
	} {
		actual, err := fileSHA(p.path)
		if err != nil || actual != p.sha {
			return errors.New("frozen public protocol/curriculum evidence differs")
		}
		published, err := exec.Command("git", "show", p.revision+":"+p.path).Output()
		if err != nil || threecohort.SHA(published) != p.sha {
			return errors.New("protocol/curriculum was not published before teacher observation")
		}
	}
	return nil
}

func collect(dataset, teacher, output, revision string) (resultErr error) {
	if err := cleanSource(revision); err != nil {
		return err
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) || !strings.HasPrefix(filepath.Base(output), "own-three-") {
		return errors.New("fresh own-three-* output required; preserve every earlier attempt")
	}
	if filepath.Clean(filepath.Dir(output)) != "runs" {
		return errors.New("collection output must join the bounded runs/own-three-* study")
	}
	prior, priorBytes, err := inventory("runs")
	if err != nil {
		return err
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	m, err := decision.LoadPath(teacher)
	if err != nil {
		return err
	}
	if m.MetadataSHA256() != threefeedback.TeacherMetadata || m.WeightsSHA256() != threefeedback.TeacherWeights || m.Variant() != "fp32" || m.FeatureVersion() != decision.SemanticContextIntentFeatureVersion {
		return errors.New("frozen v1 own independent FP32 teacher required")
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	pre := preexecution{Schema: "gooo/own-three-choice-teacher-preexecution/v1", Source: revision, Go: runtime.Version(), Protocol: threefeedback.ProtocolSHA, ProtocolRevision: threefeedback.ProtocolRevision, FreezeRevision: freezeRevision, Dataset: threecohort.DatasetSHA, Manifest: threecohort.ManifestSHA, Audit: threecohort.AuditSHA, Metadata: threefeedback.TeacherMetadata, Weights: threefeedback.TeacherWeights, Sessions: 4096, Training: 2048, Initial: 3072, Seeds: [2]string{"three-source-path-teacher-0", "three-source-path-teacher-1"}, Budget: 8, Step: 1, Rounds: 7, PriorBytes: priorBytes, PriorFiles: prior, Cap: rawCap, LineCap: lineCap, Scope: "Frozen own independent teacher supplies observations only; no student initialization/optimization, native call or CI hint. Derived joint contexts retain all three source inputs and actual selected failures; they are not joint teacher predictions."}
	if err = save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	used := priorBytes
	journal, err := openJournal(filepath.Join(output, "teacher-sessions.jsonl"), &used)
	if err != nil {
		return err
	}
	counts := totals{}
	latency := [4096]int64{}
	started := time.Now()
	var before, after syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &before)
	defer func() {
		if err := journal.file.Sync(); resultErr == nil && err != nil {
			resultErr = err
		}
		if err := journal.file.Close(); resultErr == nil && err != nil {
			resultErr = err
		}
		status := "COMPLETE_PENDING_INDEPENDENT_AUDIT"
		failure := ""
		if resultErr != nil {
			status = "FAILED_PARTIAL_COLLECTION_RETAINED"
			failure = resultErr.Error()
		}
		attempt := map[string]any{"schema": "gooo/own-three-choice-teacher-attempt/v1", "status": status, "error": failure, "actual_teacher": counts, "teacher_journal": journal.pin(), "total_retained_raw_journal_bytes_including_prior_phases": used, "student_optimizer_updates": 0, "native_calls": 0, "scope": "A failed/canceled/declined capture remains in the prefix. No restart, trimming, fixture change or learning is authorized by this attempt record."}
		if err := save(filepath.Join(output, "collection-attempt.json"), attempt); resultErr == nil && err != nil {
			resultErr = err
		}
	}()
	states := map[string]*threefeedback.State{}
	for _, v := range views {
		initial := threefeedback.Initial(v)
		states[initial.ID] = &initial
		if v.Split != "train" {
			continue
		}
		for seedIndex := range 2 {
			c, callErr := one(v, m, seedIndex)
			raw, err := json.Marshal(c)
			if err != nil {
				return err
			}
			// Store the actual call result before any audit or learning gate can fail.
			if err = journal.appendRaw(raw); err != nil {
				return err
			}
			index := counts.Sessions
			counts.add(c)
			latency[index] = c.WallNS
			if callErr != nil {
				return errors.New("teacher session failed; actual captured prefix retained")
			}
			if err = threefeedback.Verify(v, c); err != nil {
				return err
			}
			threefeedback.AddStates(states, v, c, raw, index)
		}
		if counts.Sessions%512 == 0 {
			fmt.Printf("teacher sessions=%d predictions=%d actual candidates=%d\n", counts.Sessions, counts.Predictions, counts.Attempts)
		}
	}
	if counts.Sessions != 4096 || len(states) < 3072 {
		return errors.New("teacher or initial-state denominator differs")
	}
	stateJournal, err := openJournal(filepath.Join(output, "states.jsonl"), &used)
	if err != nil {
		return err
	}
	defer stateJournal.file.Close()
	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		raw, err := json.Marshal(states[id])
		if err != nil {
			return err
		}
		if err = stateJournal.appendRaw(raw); err != nil {
			return err
		}
	}
	if err = stateJournal.file.Sync(); err != nil {
		return err
	}
	if err = journal.file.Sync(); err != nil {
		return err
	}
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &after)
	wall := time.Since(started).Nanoseconds()
	cpu := usageNS(after) - usageNS(before)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	slices.Sort(latency[:])
	median, p95 := latency[2048], latency[(4096*95+99)/100-1]
	preSHA, err := fileSHA(filepath.Join(output, "preexecution.json"))
	if err != nil {
		return err
	}
	report := collectionReport{Schema: "gooo/own-three-choice-teacher-collection/v1", Status: "COMPLETE_PENDING_INDEPENDENT_AUDIT", Actual: counts, Initial: 3072, Continuations: len(states) - 3072, Rows: len(states), Files: map[string]filePin{"teacher-sessions.jsonl": journal.pin(), "states.jsonl": stateJournal.pin()}, PreSHA: preSHA, PriorBytes: priorBytes, NewRawBytes: used - priorBytes, MedianNS: median, P95NS: p95, WallNS: wall, CPUNS: cpu, CPUPercent: float64(cpu) / float64(wall) * 100, PeakRSSBytes: rssBytes(after.Maxrss), GoHeapBytes: mem.HeapAlloc, Scope: "Actual SDK teacher inference, retained candidate evaluations and source-preserving student context projections. Session latency includes SDK ranking/search/evaluator/hash work, not standalone neural-kernel latency. CPU is normalized to one core for collection/state serialization/immediate audit; process lifetime RSS includes data reconstruction and cannot be equated to model tensor RAM. No three-choice student training or actual native generation has occurred."}
	return save(filepath.Join(output, "report.json"), report)
}

func one(v threecohort.View, m *decision.Model, index int) (threefeedback.Capture, error) {
	seed := fmt.Sprintf("three-source-path-teacher-%d", index)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	s, _, p, f, err := v.Prepared.SearchFeedbackBatchesUnfixed(ctx, m, v.Cases, 8, 1, seed, 7, nil)
	c := threefeedback.Capture{Schema: "gooo/own-three-choice-teacher-capture/v1", ViewID: v.ID, SourceSHA: v.SourceSHA, InputSHA: threecohort.SHA([]byte(v.Text)), SeedIndex: index, Seed: seed, WallNS: time.Since(start).Nanoseconds(), Search: s, Progress: p, Feedback: f, Derived: []threefeedback.Projection{}, TeacherInputs: []threefeedback.TeacherInput{}}
	if err != nil {
		c.RuntimeError = err.Error()
	}
	for i, r := range f {
		if 2*i+1 >= len(p) {
			c.RuntimeError = "missing feedback cause"
			return c, errors.New(c.RuntimeError)
		}
		inputs, e := threefeedback.AttemptedInputs(v, r, p[2*i+1])
		if e != nil {
			c.RuntimeError = e.Error()
			return c, e
		}
		c.TeacherInputs = append(c.TeacherInputs, inputs...)
		projection, e := threefeedback.Derive(v, r, p[2*i+1])
		if e != nil {
			c.RuntimeError = e.Error()
			return c, e
		}
		if projection != nil {
			c.Derived = append(c.Derived, *projection)
		}
	}
	return c, err
}

func usageNS(u syscall.Rusage) int64 {
	return u.Utime.Sec*1e9 + int64(u.Utime.Usec)*1e3 + u.Stime.Sec*1e9 + int64(u.Stime.Usec)*1e3
}
func rssBytes(v int64) int64 {
	if runtime.GOOS == "darwin" {
		return v
	}
	if runtime.GOOS == "linux" {
		return v * 1024
	}
	return 0
}

type totals struct {
	Sessions         int    `json:"actual_sdk_sessions"`
	Predictions      int    `json:"actual_teacher_predictions"`
	InitialCalls     int    `json:"initial_teacher_predictions"`
	FeedbackCalls    int    `json:"feedback_teacher_predictions"`
	Attempts         int    `json:"actual_candidate_attempts"`
	CaseInvocations  int    `json:"actual_ordered_evaluator_invocations"`
	TeacherDeclines  int    `json:"teacher_zero_call_context_declines"`
	StudentDeclines  int    `json:"derived_complete_student_context_declines"`
	Unnecessary      int    `json:"sole_remaining_zero_call_feedback"`
	FixedCoordinates int    `json:"committed_fixed_coordinates_skipped"`
	Complete         int    `json:"complete_sessions"`
	Curve            [8]int `json:"complete_sessions_by_budgets_1_to_8"`
}

func (t *totals) add(c threefeedback.Capture) {
	t.Sessions++
	t.Predictions += c.Search.Selection.ModelCalls
	if len(c.Progress) > 0 {
		t.InitialCalls += c.Progress[0].Selection.ModelCalls
	}
	t.Attempts += len(c.Search.Attempts)
	for _, a := range c.Search.Attempts {
		t.CaseInvocations += len(a.Results)
	}
	if c.Search.Status == "TRAINING_COMPLETE" {
		t.Complete++
		for i := len(c.Search.Attempts) - 1; i < 8; i++ {
			t.Curve[i]++
		}
	}
	for _, f := range c.Feedback {
		t.FeedbackCalls += f.ModelCalls
		t.FixedCoordinates += len(f.FixedCoordinates)
		if f.ContextDeclined {
			t.TeacherDeclines++
		}
		if f.RankingUnnecessary {
			t.Unnecessary++
		}
	}
	for _, p := range c.Derived {
		if !p.Representable {
			t.StudentDeclines++
		}
	}
}

type collectionReport struct {
	Schema        string             `json:"schema"`
	Status        string             `json:"status"`
	Actual        totals             `json:"actual_teacher"`
	Initial       int                `json:"initial_state_views"`
	Continuations int                `json:"unique_train_continuation_states"`
	Rows          int                `json:"student_state_rows"`
	Files         map[string]filePin `json:"raw_files"`
	PreSHA        string             `json:"preexecution_sha256"`
	PriorBytes    int64              `json:"prior_phase_evidence_bytes"`
	NewRawBytes   int64              `json:"new_raw_journal_bytes"`
	MedianNS      int64              `json:"sdk_teacher_session_median_ns"`
	P95NS         int64              `json:"sdk_teacher_session_p95_ns"`
	WallNS        int64              `json:"collection_state_serialization_and_immediate_audit_wall_ns"`
	CPUNS         int64              `json:"collection_state_serialization_and_immediate_audit_cpu_ns"`
	CPUPercent    float64            `json:"collection_cpu_percent_one_core"`
	PeakRSSBytes  int64              `json:"process_lifetime_peak_rss_bytes"`
	GoHeapBytes   uint64             `json:"go_heap_allocated_bytes_after_collection"`
	Scope         string             `json:"scope"`
}
