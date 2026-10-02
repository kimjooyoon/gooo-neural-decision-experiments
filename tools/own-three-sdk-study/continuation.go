package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const continuationDocument = "docs/own-three-choice-sdk-storage-continuation-preregistration-20261002.md"
const continuationSHA = "b127f62dc5c45694afc9703be713bd9db834bee0699f1c31614e62729a721e10"
const originalAttemptSHA = "05445e1aac33617875144126ae072ee21f4dac18002e46d77b79e62a0991b73b"
const originalProducer = "b8c515fcd5a5e623aae5fd3b93c3b8f69f9f4608"
const continuationCap int64 = 2 << 30
const continuationSessions = 2485

type missingIdentity struct {
	Policy string    `json:"policy"`
	View   string    `json:"view_id"`
	Source string    `json:"original_source_sha256"`
	Input  string    `json:"complete_input_sha256"`
	Parts  [3]string `json:"ordered_part_sha256"`
	Model  pin       `json:"model_pin"`
}
type tailPreexecution struct {
	Schema        string                      `json:"schema"`
	Source        string                      `json:"source_revision"`
	Go            string                      `json:"go"`
	Protocol      string                      `json:"original_protocol_sha256"`
	Amendment     string                      `json:"storage_continuation_sha256"`
	Dataset       string                      `json:"source_dataset_sha256"`
	OriginalPhase string                      `json:"original_phase"`
	OriginalFiles map[string]threestudent.Pin `json:"original_phase_files"`
	OriginalAudit threestudent.Pin            `json:"original_independent_audit"`
	Models        map[string]pin              `json:"unchanged_model_pins"`
	Selected      string                      `json:"original_calibration_selected_candidate"`
	PriorFiles    map[string]threestudent.Pin `json:"prior_raw_files"`
	PriorBytes    int64                       `json:"prior_retained_raw_bytes"`
	RawCap        int64                       `json:"amended_whole_study_raw_cap_bytes"`
	Available     uint64                      `json:"available_disk_bytes_before_phase"`
	Missing       []missingIdentity           `json:"only_missing_development_identities"`
	Budget        int                         `json:"candidate_budget"`
	Step          int                         `json:"candidates_per_advance"`
	Rounds        int                         `json:"maximum_feedback_rounds"`
	Seed          string                      `json:"seed"`
	CI            bool                        `json:"ci_hint_supplied"`
}
type tailAttempt struct {
	Schema      string                      `json:"schema"`
	Status      string                      `json:"status"`
	Error       string                      `json:"error"`
	Sessions    int                         `json:"actual_tail_sdk_sessions"`
	Predictions int                         `json:"actual_tail_model_predictions"`
	Prior       int64                       `json:"prior_raw_bytes"`
	New         int64                       `json:"retained_new_bytes_before_attempt"`
	Files       map[string]threestudent.Pin `json:"retained_files_before_attempt"`
	WallNS      int64                       `json:"tail_collector_wall_ns"`
	CPUNS       int64                       `json:"tail_collector_cpu_ns"`
	CPUPercent  float64                     `json:"cpu_percent_of_one_core"`
	RSS         int64                       `json:"process_lifetime_peak_rss_bytes"`
	Setup       map[string]int64            `json:"model_load_and_pin_wall_ns"`
	Updates     int                         `json:"new_optimizer_updates"`
	Native      int                         `json:"native_calls"`
	Promoted    bool                        `json:"default_model_promoted"`
}

func closedFiles(dir string) (map[string]threestudent.Pin, int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	all := map[string]threestudent.Pin{}
	var used int64
	for _, d := range entries {
		if d.IsDir() {
			return nil, 0, errors.New("regular closed phase members required")
		}
		p, err := threestudent.FilePin(filepath.Join(dir, d.Name()))
		if err != nil {
			return nil, 0, err
		}
		all[d.Name()] = p
		used += p.Bytes
	}
	return all, used, nil
}
func amendment() error {
	raw, err := os.ReadFile(continuationDocument)
	if err != nil {
		return err
	}
	if threecohort.SHA(raw) != continuationSHA {
		return errors.New("separate immutable storage continuation differs")
	}
	return nil
}
func originalPins(prefix string) (map[string]threestudent.Pin, error) {
	all, _, err := closedFiles(prefix)
	if err != nil {
		return nil, err
	}
	if len(all) != 21 || all["collection-attempt.json"].SHA != originalAttemptSHA {
		return nil, errors.New("exact published cap-stopped original prefix required")
	}
	var pre preexecution
	if err = strict(filepath.Join(prefix, "preexecution.json"), &pre); err != nil {
		return nil, err
	}
	if pre.Source != originalProducer {
		return nil, errors.New("original producer changed")
	}
	return all, nil
}

// cellRows streams a contiguous slice of one frozen cell. A tail never begins
// at row zero of a partially completed cell and cannot inject a duplicate row.
func cellRows(name, policy string, cell []threecohort.View, offset int, a *accumulator, m *model, verifyRows bool) (int, error) {
	if offset < 0 || offset > len(cell) {
		return 0, errors.New("cell offset outside frozen identity list")
	}
	f, err := os.Open(name)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), lineCap)
	n := 0
	for scan.Scan() {
		if offset+n >= len(cell) {
			return n, errors.New("extra or duplicated view outside exact remaining cell")
		}
		var o observation
		if err = threecohort.Decode(scan.Bytes(), &o); err != nil {
			return n, err
		}
		v := cell[offset+n]
		if o.Policy != policy || o.Capture.ViewID != v.ID || o.Capture.SourceSHA != v.SourceSHA || o.Capture.InputSHA != threecohort.SHA([]byte(v.Text)) {
			return n, errors.New("actual cell identity differs from original ordered source/input/policy")
		}
		if verifyRows {
			if err = verify(v, o.Capture, m); err != nil {
				return n, fmt.Errorf("%s %s: %w", policy, v.ID, err)
			}
		}
		if a != nil {
			if err = a.add(v, o.Capture); err != nil {
				return n, err
			}
		}
		n++
	}
	return n, scan.Err()
}
func splitViews(views []threecohort.View, split string) []threecohort.View {
	cell := make([]threecohort.View, 0, 512)
	for _, v := range views {
		if v.Split == split {
			cell = append(cell, v)
		}
	}
	return cell
}
func deriveMissing(cell []threecohort.View, ids []string, seen map[string]int, models map[string]*model) ([]missingIdentity, error) {
	if len(cell) != 512 || len(ids) != 11 || len(seen) != 11 {
		return nil, errors.New("exact original cell denominator required")
	}
	want := map[string]int{"set-initial/ptq_ternary": 75, "set-initial/qat_ternary": 0, "uniform-initial/fp32": 0, "uniform-initial/ptq_ternary": 0, "uniform-initial/qat_ternary": 0}
	missing := make([]missingIdentity, 0, continuationSessions)
	for _, id := range ids {
		start, ok := want[id]
		if !ok {
			start = 512
		}
		if seen[id] != start {
			return nil, errors.New("observed original prefix differs from fixed continuation boundary")
		}
		for _, v := range cell[start:] {
			item := missingIdentity{Policy: id, View: v.ID, Source: v.SourceSHA, Input: threecohort.SHA([]byte(v.Text)), Model: models[id].Pin}
			for i, part := range v.Parts {
				item.Parts[i] = threecohort.SHA([]byte(part))
			}
			missing = append(missing, item)
		}
	}
	if len(missing) != continuationSessions {
		return nil, errors.New("exact 2485 missing sessions required")
	}
	return missing, nil
}
func originalMissing(prefix string, views []threecohort.View, ids []string, models map[string]*model) ([]missingIdentity, error) {
	cell := splitViews(views, "development")
	seen := map[string]int{}
	for _, id := range ids {
		n, err := cellRows(filepath.Join(prefix, filename("development", id)), id, cell, 0, nil, models[id], false)
		if err != nil {
			return nil, err
		}
		seen[id] = n
	}
	return deriveMissing(cell, ids, seen, models)
}

func continueStudy(dataset, modelRoot, prefix, output, revision string) (failure error) {
	if err := source(revision); err != nil {
		return err
	}
	if err := amendment(); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(output)
	if err != nil || filepath.Dir(absolute) != filepath.Join(cwd, "runs") || !strings.HasPrefix(filepath.Base(absolute), "own-three-") {
		return errors.New("fresh own-three continuation directly under runs required")
	}
	if _, err = os.Lstat(absolute); !os.IsNotExist(err) {
		return errors.New("never overwrite or restart retained continuation")
	}
	original, err := originalPins(prefix)
	if err != nil {
		return err
	}
	prior, used, err := inventory()
	if err != nil {
		return err
	}
	store := &storage{Used: used, Prior: used, Cap: continuationCap, FreePath: cwd}
	if err = store.beforeCall(); err != nil {
		return err
	}
	available, err := freeBytes(cwd)
	if err != nil {
		return err
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	start, cpuStart := time.Now(), cpuNS()
	models, ids, err := loadModels(modelRoot)
	if err != nil {
		return err
	}
	pins, setup := map[string]pin{}, map[string]int64{}
	for id, m := range models {
		if m != nil {
			pins[id], setup[id] = m.Pin, m.SetupNS
		}
	}
	if err = os.Mkdir(absolute, 0755); err != nil {
		return err
	}
	sessions, predictions := 0, 0
	defer func() {
		wall, cpu := time.Since(start).Nanoseconds(), cpuNS()-cpuStart
		status, message := "SDK_TAIL_CAPTURED_PENDING_EXTERNAL_REPLAY", ""
		if failure != nil {
			status, message = "FAILED_TAIL_PREFIX_RETAINED", failure.Error()
		}
		files, _, e := closedFiles(absolute)
		if e != nil {
			if failure == nil {
				failure = e
			}
			return
		}
		a := tailAttempt{Schema: "gooo/own-three-sdk-tail-attempt/v1", Status: status, Error: message, Sessions: sessions, Predictions: predictions, Prior: store.Prior, New: store.Used - store.Prior, Files: files, WallNS: wall, CPUNS: cpu, CPUPercent: 100 * float64(cpu) / float64(wall), RSS: peakRSS(), Setup: setup}
		if e = store.save(filepath.Join(absolute, "collection-attempt.json"), a); e != nil && failure == nil {
			failure = e
		}
	}()
	// The complete independent original audit is retained before deriving the
	// missing list, before any operational inference or candidate evaluation.
	auditName := filepath.Join(absolute, "original-prefix-audit.json")
	if err = auditStudy(dataset, modelRoot, prefix, auditName); err != nil {
		return err
	}
	auditPin, err := threestudent.FilePin(auditName)
	if err != nil {
		return err
	}
	store.Used += auditPin.Bytes
	missing, err := originalMissing(prefix, views, ids, models)
	if err != nil {
		return err
	}
	pre := tailPreexecution{Schema: "gooo/own-three-sdk-tail-preexecution/v1", Source: revision, Go: runtime.Version(), Protocol: threefeedback.ProtocolSHA, Amendment: continuationSHA, Dataset: threecohort.DatasetSHA, OriginalPhase: filepath.Base(prefix), OriginalFiles: original, OriginalAudit: auditPin, Models: pins, Selected: "set-feedback/fp32", PriorFiles: prior, PriorBytes: used, RawCap: continuationCap, Available: available, Missing: missing, Budget: 8, Step: 1, Rounds: 7}
	if err = store.save(filepath.Join(absolute, "preexecution.json"), pre); err != nil {
		return err
	}
	byID := map[string]threecohort.View{}
	for _, v := range views {
		byID[v.ID] = v
	}
	var f *os.File
	policy := ""
	closeFile := func() error {
		if f == nil {
			return nil
		}
		err := f.Sync()
		closeErr := f.Close()
		f = nil
		if err != nil {
			return err
		}
		return closeErr
	}
	defer func() {
		if e := closeFile(); e != nil && failure == nil {
			failure = e
		}
	}()
	for _, item := range pre.Missing {
		if item.Policy != policy {
			if err = closeFile(); err != nil {
				return err
			}
			policy = item.Policy
			f, err = os.OpenFile(filepath.Join(absolute, filename("development", policy)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
		}
		if err = store.beforeCall(); err != nil {
			return err
		}
		v := byID[item.View]
		capture, runtimeErr := one(v, models[policy])
		raw, err := json.Marshal(observation{policy, capture})
		if err != nil {
			return err
		}
		if len(raw)+1 > lineCap {
			return errors.New("full tail capture exceeds unchanged one-MiB line cap")
		}
		if err = store.write(f, append(raw, '\n'), receiptReserve); err != nil {
			return err
		}
		sessions++
		predictions += capture.Search.Selection.ModelCalls
		if runtimeErr == nil {
			runtimeErr = verify(v, capture, models[policy])
		}
		if runtimeErr != nil {
			return runtimeErr
		}
		if sessions%128 == 0 {
			fmt.Printf("tail sessions=%d/%d calls=%d raw=%d\n", sessions, continuationSessions, predictions, store.Used)
		}
	}
	if err = closeFile(); err != nil {
		return err
	}
	if sessions != continuationSessions {
		return errors.New("full fixed continuation denominator differs")
	}
	after, err := originalPins(prefix)
	if err != nil || !reflect.DeepEqual(after, original) {
		return errors.New("original bytes changed during separate tail")
	}
	return nil
}
