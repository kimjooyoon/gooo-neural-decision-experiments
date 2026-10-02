// audit-full-input-forms prepares and verifies the complete registered wording
// matrix. Preparation never invokes a model or changes model weights.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

const protocolPath = "docs/full-input-all-forms-protocol-20261003.md"
const phasePrefix = "own-three-full-input-forms-"
const protocolSHA256 = "ce5e4b5ebcdf8fa110439f66e64aaca1b69f2907456e3fbf25141059f30b567b"
const datasetPath = "runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl"

type inputRecord struct {
	Index       int       `json:"index"`
	ViewIndex   int       `json:"view_index"`
	ViewID      string    `json:"view_id"`
	Group       string    `json:"program_group"`
	Family      string    `json:"family"`
	Split       string    `json:"split"`
	Language    string    `json:"language"`
	Form        string    `json:"form"`
	SourceSHA   string    `json:"source_sha256"`
	OriginalSHA string    `json:"original_input_sha256"`
	Text        string    `json:"complete_input"`
	InputSHA    string    `json:"input_sha256"`
	Features    [2]string `json:"v3_v4_feature_sha256"`
	PartBytes   [3]int    `json:"part_bytes"`
	Declined    bool      `json:"representation_declined"`
	Passed      [8]int    `json:"finite_passed_by_mask"`
	Cases       int       `json:"finite_cases"`
	Valid       uint8     `json:"valid_masks_bitset"`
}

func prepareInput(v threecohort.View, vi, fi int) (inputRecord, error) {
	if vi < 0 || fi < 0 || fi >= len(forms) {
		return inputRecord{}, errors.New("bounded input coordinates required")
	}
	input, err := applyForm(v, forms[fi])
	if err != nil {
		return inputRecord{}, err
	}
	valid, err := passingSet(v.Target)
	if err != nil {
		return inputRecord{}, err
	}
	row := inputRecord{Index: vi*len(forms) + fi, ViewIndex: vi, ViewID: v.ID, Group: v.Group, Family: v.Family, Split: v.Split, Language: v.Language, Form: forms[fi], SourceSHA: v.SourceSHA, OriginalSHA: threecohort.SHA([]byte(v.Text)), Text: input.Text, InputSHA: threecohort.SHA([]byte(input.Text)), PartBytes: input.PartBytes, Declined: input.Declined, Passed: v.Target.Passed, Cases: v.Target.Cases, Valid: valid}
	original, err := features(v.Text, false)
	if err != nil {
		return row, err
	}
	for i, bag := range [2]bool{false, true} {
		candidate, e := features(input.Text, bag)
		if (e != nil) != input.Declined {
			return row, errors.New("representability differs between contracts")
		}
		if e == nil {
			if !preservedSource(&original, &candidate) {
				return row, errors.New("source coordinates changed")
			}
			raw := featureBytes(&candidate)
			row.Features[i] = threecohort.SHA(raw[:])
		}
	}
	return row, nil
}

func cleanSource(revision string) error {
	if runtime.Version() != "go1.27.1" || len(revision) != 40 {
		return errors.New("exact Go 1.27.1 source required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return err
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != revision || len(dirty) != 0 {
		return errors.New("clean exact source required")
	}
	return nil
}

func prepare(output, revision string) (failure error) {
	if err := cleanSource(revision); err != nil {
		return err
	}
	storage, err := storagePreflight(output, true)
	if err != nil {
		return err
	}
	protocol, err := boundedFile(protocolPath, 64<<10)
	if err != nil {
		return err
	}
	protocolSHA := threecohort.SHA(protocol)
	if protocolSHA != protocolSHA256 {
		return errors.New("published protocol bytes differ")
	}
	if err = os.Mkdir(output, 0700); err != nil {
		return err
	}
	used := int64(0)
	completed, declined := 0, 0
	start := time.Now()
	deadline := start.Add(15 * time.Minute)
	defer func() {
		if failure != nil {
			_ = saveFresh(output, "failure.json", map[string]any{"status": "FAILED_PREFIX_RETAINED", "stage": "input_preparation", "completed_inputs": completed, "declined_inputs": declined, "actual_model_calls": 0, "optimizer_updates": 0, "automatic_retries": 0, "error": failure.Error()})
		}
	}()
	before, err := usage()
	if err != nil {
		return err
	}
	pre := preparationHeader{Schema: "gooo/full-input-forms-preparation/v1", Source: revision, Go: runtime.Version(), Protocol: protocolSHA, Dataset: threecohort.DatasetSHA, Forms: forms, Planned: 33792, Storage: storage, Scope: "Complete initial views from all three previously used/observed splits; diagnostic forms remain labeled interventions."}
	if err = saveFresh(output, "preexecution.json", pre); err != nil {
		return err
	}
	control, err := fullinputstudy.MakeOrderControl()
	if err != nil {
		return err
	}
	if err = saveFresh(output, "order-control.json", control); err != nil {
		return err
	}
	views, err := threecohort.Load(datasetPath)
	if err != nil {
		return err
	}
	writer, err := newJournal(output, "inputs.jsonl", &used)
	if err != nil {
		return err
	}
	defer writer.Close()
	splits := map[string]int{}
	collisions := newCollisionIndex()
	for vi, v := range views {
		if time.Now().After(deadline) {
			return errors.New("preparation deadline; original prefix retained")
		}
		for fi := range forms {
			row, e := prepareInput(v, vi, fi)
			if e != nil {
				return e
			}
			if e = writer.Append(row); e != nil {
				return e
			}
			completed++
			splits[v.Split]++
			if row.Declined {
				declined++
			} else {
				collisions.add(row)
			}
		}
	}
	if err = writer.Close(); err != nil {
		return err
	}
	if completed != 33792 || splits["train"] != 22528 || splits["calibration"] != 5632 || splits["development"] != 5632 {
		return errors.New("input split denominator differs")
	}
	records, collisionCounts := collisions.records()
	collisionWriter, err := newJournal(output, "collisions.jsonl", &used)
	if err != nil {
		return err
	}
	defer collisionWriter.Close()
	for _, record := range records {
		if err = collisionWriter.Append(record); err != nil {
			return err
		}
	}
	if err = collisionWriter.Close(); err != nil {
		return err
	}
	inputs, err := filePin(filepath.Join(output, "inputs.jsonl"))
	if err != nil {
		return err
	}
	collisionPin, err := filePin(filepath.Join(output, "collisions.jsonl"))
	if err != nil {
		return err
	}
	controlPin, err := filePin(filepath.Join(output, "order-control.json"))
	if err != nil {
		return err
	}
	after, err := usage()
	if err != nil {
		return err
	}
	report := preparationReport{Schema: "gooo/full-input-forms-preparation-result/v1", Status: "PREPARED", Source: revision, Protocol: protocolSHA, Rows: completed, Accepted: completed - declined, Declined: declined, Splits: splits, Collisions: collisionCounts, Files: map[string]fileDigest{"inputs.jsonl": inputs, "collisions.jsonl": collisionPin, "order-control.json": controlPin}, WallNS: time.Since(start).Nanoseconds(), CPUSeconds: after.CPUSeconds - before.CPUSeconds, PeakRSSBytes: after.PeakRSSBytes}
	if err = saveFresh(output, "report.json", report); err != nil {
		return err
	}
	if _, err = storagePreflight(output, false); err != nil {
		return err
	}
	fmt.Printf("PREPARED: %d complete input rows; %d declined; zero model calls\n", completed, declined)
	return nil
}

func main() {
	mode := flag.String("mode", "prepare", "prepare or verify-prepare")
	output := flag.String("output", "", "fresh preparation directory or existing directory to verify")
	revision := flag.String("source-revision", "", "clean full source revision")
	flag.Parse()
	var err error
	switch *mode {
	case "prepare":
		err = prepare(*output, *revision)
	case "verify-prepare":
		err = verifyPreparation(*output, *revision)
	default:
		err = errors.New("unknown mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func sameJSON(a, b any) bool {
	left, e := json.Marshal(a)
	if e != nil {
		return false
	}
	right, e := json.Marshal(b)
	return e == nil && string(left) == string(right)
}
