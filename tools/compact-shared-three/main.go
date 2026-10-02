// compact-shared-three converts only the three immutable published shared judges
// and compares actual Go predictions on every frozen initial/feedback state.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const protocol = "preexecution/shared-three-compact-runtime-preregistration-20261003.json"
const protocolSHA = "2dae15f2fdf33991124815685e239f8b77e36d24200685b5caf906236de4fb81"
const outputCap = 32 << 20

var variants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}
var metadataSHA = [3]string{
	"487797f510a0842b1bb166ba11a4d14f892697f3896c8100d194bde1d3318401",
	"0ca5545fc9ecda3387d31ec08c19f78bdc2741aedaca414cb6e41a197b713e40",
	"df3ca87457c3890c1a23d5ebc4cbd7de5a3a7ecb4a0ffc4cf07c46f3f901a7b9",
}
var weightsSHA = [3]string{
	"0c738489e1fbf4759ff970e83cd0685ff466d4abc65de64022f06b5ccb61cb50",
	"40902c3bb00a944e490081353635d5a4fb7834bcf26062df612688a3f5e160d8",
	"b71066c3259dd4e34c97bbf8912a4b2e73f27bc807b13f81402203e340a20dd6",
}

type observation struct {
	Metadata    string  `json:"metadata_sha256"`
	Weights     string  `json:"weights_sha256"`
	Packed      int     `json:"packed_weight_bytes"`
	Resident    int     `json:"resident_tensor_bytes"`
	Scales      int     `json:"matrix_scale_bytes"`
	ParityCalls int     `json:"actual_parity_predictions"`
	ProbeCalls  int     `json:"actual_allocation_probe_predictions"`
	WarmCalls   int     `json:"actual_warmup_predictions"`
	TimingCalls int     `json:"actual_timed_predictions"`
	Allocations float64 `json:"heap_allocations_per_valid_warmed_prediction"`
	MedianNS    int64   `json:"median_prediction_ns"`
	P95NS       int64   `json:"p95_prediction_ns"`
}
type pair struct {
	Variant  string      `json:"variant"`
	Expanded observation `json:"expanded"`
	Compact  observation `json:"compact"`
	Equal    int         `json:"bitwise_equal_complete_workspace_and_prediction_pairs"`
}
type report struct {
	Schema            string  `json:"schema"`
	Status            string  `json:"status"`
	Stage             string  `json:"last_stage"`
	Revision          string  `json:"source_revision"`
	Protocol          string  `json:"protocol_sha256"`
	StatesSHA         string  `json:"frozen_states_sha256"`
	DatasetSHA        string  `json:"frozen_dataset_sha256"`
	States            int     `json:"frozen_states"`
	Models            []pair  `json:"models"`
	LedgerSHA         string  `json:"comparison_ledger_sha256"`
	Workspace         uintptr `json:"caller_workspace_bytes"`
	Go                string  `json:"go_version"`
	OS                string  `json:"os"`
	Arch              string  `json:"arch"`
	CPUs              int     `json:"logical_cpus"`
	WallNS            int64   `json:"audit_wall_ns"`
	CPUSeconds        float64 `json:"audit_process_cpu_seconds"`
	CPUPercent        float64 `json:"audit_process_cpu_percent_one_core"`
	RSSBytes          int64   `json:"audit_process_lifetime_peak_rss_bytes"`
	BytesBeforeReport int64   `json:"new_output_bytes_before_report"`
	NewUpdates        int     `json:"new_optimizer_updates"`
	NativeCalls       int     `json:"new_native_calls"`
	Scope             string  `json:"scope"`
}
type comparison struct {
	ID     string    `json:"state_id"`
	Input  string    `json:"input_sha256"`
	Hashes [3]string `json:"identical_workspace_and_prediction_sha256_by_variant"`
}
type limited struct {
	f *os.File
	n *int64
}

func (w limited) Write(b []byte) (int, error) {
	if int64(len(b))+*w.n > outputCap-(64<<10) {
		return 0, errors.New("compact study storage cap")
	}
	n, e := w.f.Write(b)
	*w.n += int64(n)
	return n, e
}
func write(dir, name string, b []byte, n *int64) error {
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = (limited{f, n}).Write(b)
	syncErr := f.Sync()
	closeErr := f.Close()
	return errors.Join(err, syncErr, closeErr)
}
func asJSON(v any) ([]byte, error) {
	b, e := json.MarshalIndent(v, "", "  ")
	return append(b, '\n'), e
}
func observationFor(m *jointdecision.ThreeModel) observation {
	return observation{Metadata: m.MetadataSHA256(), Weights: m.WeightsSHA256(), Packed: m.PackedFileBytes(), Resident: m.ResidentTensorBytes(), Scales: m.MatrixScaleBytes()}
}
func bitDigest(w jointdecision.ThreeWorkspace, p jointdecision.ThreePrediction) string {
	var raw [(768+24+8+8+8)*4 + 2]byte
	at := 0
	for _, values := range [][]float32{w.Features[:], w.Hidden[:], w.Logits[:], p.Logits[:], p.Probabilities[:]} {
		for _, v := range values {
			binary.LittleEndian.PutUint32(raw[at:], math.Float32bits(v))
			at += 4
		}
	}
	binary.LittleEndian.PutUint16(raw[at:], p.Mask)
	sum := sha256.Sum256(raw[:])
	return hex.EncodeToString(sum[:])
}
func cpu(r syscall.Rusage) float64 {
	return float64(r.Utime.Sec+r.Stime.Sec) + float64(r.Utime.Usec+r.Stime.Usec)/1e6
}

func run(modelDir, dataset, teacher, output, revision string) (resultErr error) {
	head, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact audit source revision required")
	}
	dirty, e := exec.Command("git", "status", "--porcelain").Output()
	if e != nil || len(dirty) != 0 {
		return errors.New("clean committed source required")
	}
	pin, e := threestudent.FilePin(protocol)
	if e != nil || pin.SHA != protocolSHA {
		return errors.New("preregistered compact protocol changed")
	}
	if output == "" {
		return errors.New("fresh output directory required")
	}
	if _, e = os.Lstat(output); !os.IsNotExist(e) {
		return errors.New("fresh output directory required")
	}
	parent := filepath.Dir(output)
	if e = os.MkdirAll(parent, 0755); e != nil {
		return e
	}
	var disk syscall.Statfs_t
	if e = syscall.Statfs(parent, &disk); e != nil {
		return e
	}
	if disk.Bavail*uint64(disk.Bsize) < 4<<30 {
		return errors.New("four GiB free storage required")
	}
	if e = os.Mkdir(output, 0755); e != nil {
		return e
	}
	var written int64
	start := time.Now()
	var before syscall.Rusage
	if e = syscall.Getrusage(syscall.RUSAGE_SELF, &before); e != nil {
		return e
	}
	r := report{Schema: "gooo/shared-three-compact-runtime-audit/v1", Status: "FAIL", Stage: "frozen-input-validation", Revision: revision, Protocol: protocolSHA, StatesSHA: threestudent.StatesSHA, DatasetSHA: threecohort.DatasetSHA, Workspace: unsafe.Sizeof(jointdecision.ThreeWorkspace{}), Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(), Scope: "Bitwise parity on the previously observed frozen curriculum, not an accuracy improvement or new holdout. Timings are paired interleaved Go kernels including feature projection, excluding file load/source codegen. RSS/CPU describe the entire auditor including dataset reconstruction; resident tensor sizes describe the inference model. Same artifact/seed remains reproducible; cross-format seeded paths are not equated because byte identities differ. GPU utilization was not sampled and no GPU work was requested."}
	defer func() {
		r.WallNS = time.Since(start).Nanoseconds()
		var after syscall.Rusage
		resourceErr := syscall.Getrusage(syscall.RUSAGE_SELF, &after)
		if resourceErr == nil {
			r.CPUSeconds = cpu(after) - cpu(before)
			r.CPUPercent = r.CPUSeconds / (float64(r.WallNS) / 1e9) * 100
			r.RSSBytes = after.Maxrss
			if runtime.GOOS != "darwin" {
				r.RSSBytes *= 1024
			}
		}
		r.BytesBeforeReport = written
		if resultErr == nil && resourceErr == nil {
			r.Status = "PASS"
		}
		b, err := asJSON(r)
		if err == nil {
			err = write(output, "report.json", b, &written)
		}
		resultErr = errors.Join(resultErr, resourceErr, err)
	}()
	pre, e := asJSON(map[string]any{"schema": "gooo/shared-three-compact-preexecution/v1", "source_revision": revision, "protocol_sha256": protocolSHA, "model_metadata_sha256": metadataSHA, "model_weights_sha256": weightsSHA, "planned_states": 10739, "planned_parity_predictions": 64434, "raw_cap_bytes": outputCap, "available_bytes": disk.Bavail * uint64(disk.Bsize), "new_optimizer_updates": 0})
	if e != nil {
		return e
	}
	if e = write(output, "preexecution.json", pre, &written); e != nil {
		return e
	}
	states, e := threestudent.Load(dataset, teacher, "publication/own-three-choice-teacher-audit-20261002.json")
	if e != nil {
		return e
	}
	r.States = len(states)
	var models [3][2]*jointdecision.ThreeModel
	r.Stage = "strict-shared-conversion"
	for i, variant := range variants {
		expanded, e := jointdecision.LoadThree(filepath.Join(modelDir, variant, "model.json"))
		if e != nil {
			return e
		}
		if expanded.MetadataSHA256() != metadataSHA[i] || expanded.WeightsSHA256() != weightsSHA[i] {
			return errors.New("immutable published shared-local model pins differ")
		}
		meta, weights, e := jointdecision.CompactThree(expanded)
		if e != nil {
			return e
		}
		b, e := asJSON(meta)
		if e != nil {
			return e
		}
		if e = write(output, filepath.Join("models", variant, "weights.bin"), weights, &written); e != nil {
			return e
		}
		if e = write(output, filepath.Join("models", variant, "model.json"), b, &written); e != nil {
			return e
		}
		compact, e := jointdecision.LoadSharedThree(filepath.Join(output, "models", variant, "model.json"))
		if e != nil {
			return e
		}
		models[i] = [2]*jointdecision.ThreeModel{expanded, compact}
		r.Models = append(r.Models, pair{Variant: variant, Expanded: observationFor(expanded), Compact: observationFor(compact)})
	}
	r.Stage = "all-frozen-state-parity"
	f, e := os.OpenFile(filepath.Join(output, "comparisons.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	buf := bufio.NewWriter(limited{f, &written})
	defer func() { resultErr = errors.Join(resultErr, buf.Flush()) }()
	enc := json.NewEncoder(buf)
	for _, s := range states {
		row := comparison{ID: s.ID, Input: s.InputSHA}
		for i, modelPair := range models {
			var work [2]jointdecision.ThreeWorkspace
			var prediction [2]jointdecision.ThreePrediction
			for j, m := range modelPair {
				if e = m.PredictInto(s.Text, &work[j], &prediction[j]); e != nil {
					return e
				}
			}
			r.Models[i].Expanded.ParityCalls++
			r.Models[i].Compact.ParityCalls++
			a, b := bitDigest(work[0], prediction[0]), bitDigest(work[1], prediction[1])
			if a != b {
				return fmt.Errorf("full-bit parity failed for %s %s", s.ID, variants[i])
			}
			r.Models[i].Equal++
			row.Hashes[i] = a
		}
		if e = enc.Encode(row); e != nil {
			return e
		}
	}
	if e = buf.Flush(); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	ledger, e := threestudent.FilePin(filepath.Join(output, "comparisons.jsonl"))
	if e != nil {
		return e
	}
	r.LedgerSHA = ledger.SHA
	r.Stage = "warmed-allocation-and-interleaved-time"
	for i, pair := range models {
		stats := [2]*observation{&r.Models[i].Expanded, &r.Models[i].Compact}
		var w jointdecision.ThreeWorkspace
		var p jointdecision.ThreePrediction
		for j, m := range pair {
			stats[j].Allocations = testing.AllocsPerRun(100, func() {
				stats[j].ProbeCalls++
				if err := m.PredictInto(states[0].Text, &w, &p); err != nil {
					panic(err)
				}
			})
			if stats[j].Allocations != 0 {
				return errors.New("warmed valid kernel allocated")
			}
			for range 100 {
				stats[j].WarmCalls++
				if e = m.PredictInto(states[0].Text, &w, &p); e != nil {
					return e
				}
			}
		}
		var times [2][]int64
		for j := range times {
			times[j] = make([]int64, 2000)
		}
		for k := range 2000 {
			for order := range 2 {
				j := (order + k) % 2
				t := time.Now()
				e = pair[j].PredictInto(states[(k*17)%len(states)].Text, &w, &p)
				elapsed := time.Since(t).Nanoseconds()
				stats[j].TimingCalls++
				if e != nil {
					return e
				}
				times[j][k] = elapsed
			}
		}
		for j := range times {
			sort.Slice(times[j], func(a, b int) bool { return times[j][a] < times[j][b] })
			stats[j].MedianNS = times[j][1000]
			stats[j].P95NS = times[j][1899]
		}
	}
	r.Stage = "reconcile-output-budget"
	var actual int64
	e = filepath.WalkDir(output, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		s, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if s.Mode().IsRegular() {
			actual += s.Size()
		} else if !s.IsDir() {
			return errors.New("nonregular study output")
		}
		return nil
	})
	if e != nil {
		return e
	}
	if actual != written || actual > outputCap-(64<<10) {
		return errors.New("storage ledger differs")
	}
	r.Stage = "complete"
	return nil
}

func main() {
	models := flag.String("models", "", "three published shared-local expanded models")
	dataset := flag.String("dataset", "", "frozen native source dataset")
	teacher := flag.String("teacher-curriculum", "", "frozen observed teacher states")
	output := flag.String("output", "", "fresh compact study directory")
	revision := flag.String("source-revision", "", "exact clean audit source")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments")
		os.Exit(2)
	}
	if e := run(*models, *dataset, *teacher, *output, *revision); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("PASS: all three compact models preserve every frozen-state prediction; report records exact calls and storage.")
}
