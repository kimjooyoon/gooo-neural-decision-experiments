package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

type preparationHeader struct {
	Schema   string       `json:"schema"`
	Source   string       `json:"source_revision"`
	Go       string       `json:"go_version"`
	Protocol string       `json:"protocol_sha256"`
	Dataset  string       `json:"dataset_sha256"`
	Forms    [11]string   `json:"forms"`
	Planned  int          `json:"planned_input_rows"`
	Calls    int          `json:"actual_model_calls"`
	Updates  int          `json:"optimizer_updates"`
	Storage  storageState `json:"storage"`
	Scope    string       `json:"scope"`
}
type preparationReport struct {
	Schema       string                `json:"schema"`
	Status       string                `json:"status"`
	Source       string                `json:"source_revision"`
	Protocol     string                `json:"protocol_sha256"`
	Rows         int                   `json:"input_rows"`
	Accepted     int                   `json:"accepted_inputs"`
	Declined     int                   `json:"declined_inputs"`
	Splits       map[string]int        `json:"inputs_by_split"`
	Collisions   collisionStats        `json:"feature_collisions"`
	Files        map[string]fileDigest `json:"files"`
	WallNS       int64                 `json:"wall_ns"`
	CPUSeconds   float64               `json:"process_cpu_seconds"`
	PeakRSSBytes int64                 `json:"process_peak_rss_bytes"`
	Calls        int                   `json:"actual_model_calls"`
	Updates      int                   `json:"optimizer_updates"`
}
type processUsage struct {
	CPUSeconds   float64
	PeakRSSBytes int64
}

func usage() (processUsage, error) {
	var r syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &r); err != nil {
		return processUsage{}, err
	}
	u := processUsage{CPUSeconds: float64(r.Utime.Sec+r.Stime.Sec) + float64(r.Utime.Usec+r.Stime.Usec)/1e6, PeakRSSBytes: r.Maxrss}
	if runtime.GOOS != "darwin" {
		u.PeakRSSBytes *= 1024
	}
	return u, nil
}
func decodeExact(b []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("one JSON object required")
	}
	canonical, err := json.Marshal(out)
	if err != nil {
		return err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		return err
	}
	if !bytes.Equal(canonical, compact.Bytes()) {
		return errors.New("canonical producer JSON required; duplicate or reordered fields rejected")
	}
	return nil
}
func readMetadata(root, name string, out any) error {
	b, err := boundedFile(filepath.Join(root, name), 1<<20)
	if err != nil {
		return err
	}
	return decodeExact(b, out)
}
func walkJournal(path string, visit func(int, []byte) error) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > rawCap {
		return errors.New("bounded regular journal required")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), maxRow+1)
	i := 0
	for s.Scan() {
		if len(s.Bytes())+1 > maxRow {
			return errors.New("row exceeds bound")
		}
		if err := visit(i, s.Bytes()); err != nil {
			return err
		}
		i++
	}
	return s.Err()
}

// This preparation verifier regenerates every input and collision group with
// the same frozen source. A separate prediction reader is required for collection.
func verifyPreparation(output, revision string) error {
	if err := cleanSource(revision); err != nil {
		return err
	}
	if _, err := storagePreflight(output, false); err != nil {
		return err
	}
	deadline := time.Now().Add(15 * time.Minute)
	protocol, err := boundedFile(protocolPath, 64<<10)
	if err != nil {
		return err
	}
	if threecohort.SHA(protocol) != protocolSHA256 {
		return errors.New("published protocol differs")
	}
	var pre preparationHeader
	var report preparationReport
	if err := readMetadata(output, "preexecution.json", &pre); err != nil {
		return err
	}
	if err := readMetadata(output, "report.json", &report); err != nil {
		return err
	}
	if pre.Schema != "gooo/full-input-forms-preparation/v1" || pre.Source != revision || pre.Go != "go1.27.1" || pre.Protocol != protocolSHA256 || pre.Dataset != threecohort.DatasetSHA || pre.Forms != forms || pre.Planned != 33792 || pre.Calls != 0 || pre.Updates != 0 || !storageFits(pre.Storage, true) {
		return errors.New("preparation identity differs")
	}
	if report.Schema != "gooo/full-input-forms-preparation-result/v1" || report.Status != "PREPARED" || report.Source != revision || report.Protocol != protocolSHA256 || report.Calls != 0 || report.Updates != 0 || report.WallNS <= 0 || report.CPUSeconds < 0 || math.IsNaN(report.CPUSeconds) || math.IsInf(report.CPUSeconds, 0) || report.PeakRSSBytes <= 0 {
		return errors.New("preparation result identity differs")
	}
	if len(report.Files) != 3 {
		return errors.New("preparation file inventory differs")
	}
	for _, name := range []string{"inputs.jsonl", "collisions.jsonl", "order-control.json"} {
		pin, err := filePin(filepath.Join(output, name))
		if err != nil {
			return err
		}
		if report.Files[name] != pin {
			return fmt.Errorf("file digest differs: %s", name)
		}
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		return err
	}
	if len(entries) != 5 {
		return errors.New("preparation requires exactly five files")
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "inputs.jsonl", "collisions.jsonl", "order-control.json", "preexecution.json", "report.json":
		default:
			return errors.New("unexpected preparation file")
		}
	}
	control, err := fullinputstudy.MakeOrderControl()
	if err != nil {
		return err
	}
	var savedControl fullinputstudy.OrderControl
	if err := readMetadata(output, "order-control.json", &savedControl); err != nil {
		return err
	}
	if !sameJSON(control, savedControl) {
		return errors.New("order control differs")
	}
	views, err := threecohort.Load(datasetPath)
	if err != nil {
		return err
	}
	count, declined := 0, 0
	splits := map[string]int{}
	index := newCollisionIndex()
	err = walkJournal(filepath.Join(output, "inputs.jsonl"), func(i int, b []byte) error {
		if time.Now().After(deadline) {
			return errors.New("preparation replay deadline")
		}
		if i >= len(views)*len(forms) {
			return errors.New("extra input row")
		}
		var actual inputRecord
		if err := decodeExact(b, &actual); err != nil {
			return err
		}
		expected, err := prepareInput(views[i/len(forms)], i/len(forms), i%len(forms))
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("prepared input differs at %d", i)
		}
		count++
		splits[actual.Split]++
		if actual.Declined {
			declined++
		} else {
			index.add(actual)
		}
		return nil
	})
	if err != nil {
		return err
	}
	records, stats := index.records()
	collisionRows := 0
	err = walkJournal(filepath.Join(output, "collisions.jsonl"), func(i int, b []byte) error {
		if time.Now().After(deadline) {
			return errors.New("collision replay deadline")
		}
		if i >= len(records) {
			return errors.New("extra collision member")
		}
		var actual collisionRecord
		if err := decodeExact(b, &actual); err != nil {
			return err
		}
		if actual != records[i] {
			return fmt.Errorf("collision differs at %d", i)
		}
		collisionRows++
		return nil
	})
	if err != nil {
		return err
	}
	if count != 33792 || report.Rows != count || report.Accepted != count-declined || report.Declined != declined || !sameJSON(splits, report.Splits) || report.Collisions != stats || collisionRows != len(records) {
		return errors.New("recomputed preparation summary differs")
	}
	fmt.Printf("VERIFIED: %d inputs, %d collision groups, zero model calls\n", count, stats.Groups)
	return nil
}
