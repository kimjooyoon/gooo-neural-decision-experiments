package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const legacyOriginal = "publication/full-input-initial-study-20261003"
const legacyLinux = "publication/full-input-platform-diagnosis-20261003"
const legacyOriginalManifest = "45f78a838c0a7cc7255b26d596db3871a283cf6f1a8504a74f9cb211329ad32a"
const legacyLinuxManifest = "59575533d14a660886a7c6b78d08475b50153ec39919d2ecf91945663e6aac37"

type legacyRow struct {
	View       string                        `json:"view_id"`
	Group      string                        `json:"program_contract_group"`
	Language   string                        `json:"language"`
	Input      string                        `json:"input_sha256"`
	Source     string                        `json:"source_sha256"`
	Prediction jointdecision.ThreePrediction `json:"prediction"`
	Passed     [8]int                        `json:"passed_cases_by_mask"`
	Order      [8]int                        `json:"static_ranked_mask_order"`
	Text       string                        `json:"complete_input"`
}

type legacyRankDifference struct {
	Journal string    `json:"journal"`
	View    string    `json:"view_id"`
	Input   string    `json:"input_sha256"`
	Orders  [2][8]int `json:"ranked_mask_orders"`
}

func legacyBaseline(version, goos, goarch string) (string, error) {
	if version == "go1.27.1" {
		switch goos + "/" + goarch {
		case "darwin/arm64":
			return legacyOriginal, nil
		case "linux/amd64":
			return legacyLinux, nil
		}
	}
	return "", errors.New("no frozen legacy baseline for this Go version/platform")
}

func decodeLegacyJournal(raw []byte) ([]legacyRow, error) {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return nil, errors.New("bounded compressed legacy journal required")
	}
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer z.Close()
	bounded := &io.LimitedReader{R: z, N: (16 << 20) + 1}
	s := bufio.NewScanner(bounded)
	s.Buffer(make([]byte, 32768), 1<<20)
	rows := make([]legacyRow, 0, 512)
	seen := map[string]bool{}
	for s.Scan() {
		if len(rows) >= 512 {
			return nil, errors.New("extra legacy row")
		}
		var r legacyRow
		if err = threecohort.Decode(s.Bytes(), &r); err != nil {
			return nil, err
		}
		if err = legacyRowShape(s.Bytes()); err != nil {
			return nil, err
		}
		if seen[r.View] || r.View == "" || r.Group == "" || (r.Language != "ko" && r.Language != "en") ||
			len(r.Source) != 64 || len(r.Text) > jointdecision.ThreeInputMaxBytes || r.Input != threecohort.SHA([]byte(r.Text)) {
			return nil, errors.New("legacy input identity differs")
		}
		seen[r.View] = true
		order := [8]int{0, 1, 2, 3, 4, 5, 6, 7}
		sort.Slice(order[:], func(i, j int) bool {
			a, b := order[i], order[j]
			return r.Prediction.Probabilities[a] > r.Prediction.Probabilities[b] ||
				r.Prediction.Probabilities[a] == r.Prediction.Probabilities[b] && a < b
		})
		if r.Order != order || int(r.Prediction.Mask) != order[0] {
			return nil, errors.New("legacy ranking differs")
		}
		var mass float64
		for i, p := range r.Prediction.Probabilities {
			l := float64(r.Prediction.Logits[i])
			if p < 0 || p > 1 || math.IsNaN(float64(p)) || math.IsNaN(l) || math.IsInf(l, 0) || r.Passed[i] < 0 || r.Passed[i] > 16 {
				return nil, errors.New("legacy numeric/target extent differs")
			}
			mass += float64(p)
		}
		if math.Abs(mass-1) > 1e-6 {
			return nil, errors.New("legacy probability mass differs")
		}
		rows = append(rows, r)
	}
	if s.Err() != nil || bounded.N == 0 || len(rows) != 512 {
		return nil, errors.New("complete bounded legacy journal required")
	}
	return rows, nil
}

func legacyRowShape(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || len(fields) != 9 {
		return errors.New("complete legacy row fields required")
	}
	for _, v := range fields {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return errors.New("non-null legacy fields required")
		}
	}
	var prediction map[string]json.RawMessage
	if err := json.Unmarshal(fields["prediction"], &prediction); err != nil || len(prediction) != 3 {
		return errors.New("complete legacy prediction fields required")
	}
	for _, v := range prediction {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return errors.New("non-null legacy prediction required")
		}
	}
	for _, raw := range []json.RawMessage{prediction["logits"], prediction["probabilities"], fields["passed_cases_by_mask"], fields["static_ranked_mask_order"]} {
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil || len(values) != 8 {
			return errors.New("exact eight-value legacy arrays required")
		}
		for _, v := range values {
			if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return errors.New("non-null legacy array values required")
			}
		}
	}
	return nil
}

func sameLegacyRow(a, b legacyRow) bool {
	if a != b {
		return false
	}
	for i := range a.Prediction.Logits {
		if math.Float32bits(a.Prediction.Logits[i]) != math.Float32bits(b.Prediction.Logits[i]) ||
			math.Float32bits(a.Prediction.Probabilities[i]) != math.Float32bits(b.Prediction.Probabilities[i]) {
			return false
		}
	}
	return true
}

func reconcileLegacyRows(name string, original, expected, actual []legacyRow) ([]legacyRankDifference, error) {
	if len(original) != 512 || len(expected) != 512 || len(actual) != 512 {
		return nil, errors.New("legacy row denominator differs")
	}
	changed := []legacyRankDifference{}
	for i, row := range actual {
		if !sameLegacyRow(row, expected[i]) {
			return nil, fmt.Errorf("same-platform legacy row differs: %s row %d", name, i)
		}
		a, b := original[i], row
		a.Prediction, a.Order = b.Prediction, b.Order
		if a != b {
			return nil, fmt.Errorf("original legacy source/text/target differs: %s row %d", name, i)
		}
		if original[i].Order != row.Order {
			changed = append(changed, legacyRankDifference{name, row.View, row.Input, [2][8]int{original[i].Order, row.Order}})
		}
	}
	return changed, nil
}

func reconcileLegacyReport(original, expected, actual map[string]any) ([]auditDifference, error) {
	if err := compareValue("", expected, actual); err != nil {
		return nil, err
	}
	want, got := []auditDifference{}, []auditDifference{}
	collectDifferences("", original, expected, &want)
	collectDifferences("", original, actual, &got)
	if !reflect.DeepEqual(want, got) {
		return nil, errors.New("original legacy difference paths/values changed")
	}
	return got, nil
}

func legacyManifest(root, digest string) (manifest, error) {
	var m manifest
	raw, err := legacyRead(filepath.Join(root, "manifest.json"), 1<<20)
	if err != nil || threecohort.SHA(raw) != digest {
		return m, errors.New("frozen legacy manifest differs")
	}
	if err = threecohort.Decode(raw, &m); err != nil {
		return m, err
	}
	if err = verify(root); err != nil {
		return m, err
	}
	return m, nil
}

func legacyRead(path string, cap int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > cap {
		return nil, errors.New("bounded regular legacy file required")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, cap+1))
	if err != nil || len(raw) > int(cap) {
		return nil, errors.New("legacy file exceeded bound")
	}
	return raw, nil
}

func legacyMember(z *zip.ReadCloser, name string, pin threestudent.Pin) ([]byte, error) {
	if pin.Bytes <= 0 || pin.Bytes > 4<<20 {
		return nil, errors.New("bounded frozen legacy member required")
	}
	var raw []byte
	for _, f := range z.File {
		if f.Name != name {
			continue
		}
		if raw != nil || f.UncompressedSize64 != uint64(pin.Bytes) {
			return nil, errors.New("frozen legacy member inventory differs")
		}
		in, err := f.Open()
		if err != nil {
			return nil, err
		}
		raw, err = io.ReadAll(io.LimitReader(in, pin.Bytes+1))
		if err = errors.Join(err, in.Close()); err != nil {
			return nil, err
		}
	}
	if int64(len(raw)) != pin.Bytes || threecohort.SHA(raw) != pin.SHA {
		return nil, errors.New("frozen legacy member bytes differ")
	}
	return raw, nil
}

// This reader observes its own platform. Input JSON has no baseline-selection authority.
func verifyLegacyReplay(actual, revision, output string) error {
	baseline, err := legacyBaseline(runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	if output == "" {
		return errors.New("fresh legacy comparison output required")
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh legacy comparison output required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact legacy replay source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean legacy replay source required")
	}
	original, err := legacyManifest(legacyOriginal, legacyOriginalManifest)
	if err != nil {
		return err
	}
	linux, err := legacyManifest(legacyLinux, legacyLinuxManifest)
	if err != nil {
		return err
	}
	z, err := zip.OpenReader(filepath.Join(legacyOriginal, "initial-audit.zip"))
	if err != nil {
		return err
	}
	defer z.Close()
	lz, err := zip.OpenReader(filepath.Join(legacyLinux, "evidence.zip"))
	if err != nil {
		return err
	}
	defer lz.Close()
	expectedZip, expectedPins := z, original.Archives["initial-audit.zip"]
	if baseline == legacyLinux {
		expectedZip, expectedPins = lz, linux.Archives["evidence.zip"]
	}
	files, err := tree(actual)
	if err != nil {
		return err
	}
	if len(files) != len(original.Archives["initial-audit.zip"]) {
		return errors.New("legacy replay file denominator differs")
	}
	rows, models := 0, 0
	changed := []legacyRankDifference{}
	pins := map[string][3]threestudent.Pin{}
	var originalReport, expectedReport, actualReport map[string]any
	for name, originalPin := range original.Archives["initial-audit.zip"] {
		path, ok := files[name]
		if !ok {
			return fmt.Errorf("legacy replay file missing: %s", name)
		}
		a, err := legacyMember(z, name, originalPin)
		if err != nil {
			return err
		}
		b, err := legacyMember(expectedZip, name, expectedPins[name])
		if err != nil {
			return err
		}
		c, err := legacyRead(path, 4<<20)
		if err != nil {
			return err
		}
		actualPin := threestudent.Pin{SHA: threecohort.SHA(c), Bytes: int64(len(c))}
		pins[name] = [3]threestudent.Pin{originalPin, expectedPins[name], actualPin}
		switch {
		case name == "report.json":
			if err = threecohort.Decode(a, &originalReport); err != nil {
				return err
			}
			if err = threecohort.Decode(b, &expectedReport); err != nil {
				return err
			}
			if err = threecohort.Decode(c, &actualReport); err != nil {
				return err
			}
			if actualReport["auditor_source_revision"] != revision {
				return errors.New("legacy auditor source differs")
			}
		case strings.HasSuffix(name, ".jsonl.gz"):
			x, err := decodeLegacyJournal(a)
			if err != nil {
				return err
			}
			y, err := decodeLegacyJournal(b)
			if err != nil {
				return err
			}
			v, err := decodeLegacyJournal(c)
			if err != nil {
				return err
			}
			diff, err := reconcileLegacyRows(name, x, y, v)
			if err != nil {
				return err
			}
			changed = append(changed, diff...)
			rows += len(v)
		case strings.HasPrefix(name, "compact/"):
			if actualPin != originalPin || actualPin != expectedPins[name] {
				return fmt.Errorf("legacy model bytes differ: %s", name)
			}
			models++
		default:
			return errors.New("unexpected legacy member")
		}
	}
	if rows != 18432 || models != 24 {
		return errors.New("complete legacy replay required")
	}
	diff, err := reconcileLegacyReport(originalReport, expectedReport, actualReport)
	if err != nil {
		return err
	}
	sort.Slice(changed, func(i, j int) bool {
		return changed[i].Journal < changed[j].Journal || changed[i].Journal == changed[j].Journal && changed[i].View < changed[j].View
	})
	sourceFiles, err := exec.Command("git", "ls-files", "internal/decision", "internal/jointdecision", "internal/threecohort", "internal/threestudent", "internal/pathplan", "tools/audit-own-three-models", "tools/package-full-input-study", "go.mod", "go.sum").Output()
	if err != nil {
		return err
	}
	sources := map[string]threestudent.Pin{}
	for _, name := range strings.Fields(string(sourceFiles)) {
		p, err := threestudent.FilePin(name)
		if err != nil {
			return err
		}
		sources[name] = p
	}
	originalStatus := "PASS"
	if len(diff) != 0 {
		originalStatus = "MISMATCH"
	}
	value := map[string]any{
		"schema": "gooo/legacy-platform-replay/v1", "status": "PASS", "source_revision": revision,
		"go_version": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH,
		"frozen_baseline": baseline, "frozen_manifest_sha256": [2]string{legacyOriginalManifest, legacyLinuxManifest},
		"complete_rows": rows, "compact_model_files": models, "files": pins, "computational_sources": sources,
		"original_comparison":      map[string]any{"status": originalStatus, "summary_differences": diff, "changed_rank_rows": changed},
		"reader_model_predictions": 0, "new_optimizer_updates": 0,
		"scope": "Exact same-platform decoded legacy observations and compact bytes; original summary tolerances unchanged. Original cross-platform mismatch remains recorded independently of explicit-arithmetic acceptance. Known replay inputs add no new intentions.",
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil || len(raw) > 4<<20 {
		return errors.New("bounded legacy replay result required")
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	fmt.Printf("{\"status\":\"PASS\",\"comparison\":\"same-platform legacy replay\",\"rows\":%d,\"original_comparison\":%q,\"original_summary_differences\":%d,\"changed_original_rank_rows\":%d,\"reader_model_predictions\":0}\n", rows, originalStatus, len(diff), len(changed))
	return nil
}
