package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func testLegacyRows() []legacyRow {
	rows := make([]legacyRow, 512)
	for i := range rows {
		r := &rows[i]
		r.View = strings.Repeat("v", i+1)
		r.Text, r.Input = "retained input", threecohort.SHA([]byte("retained input"))
		r.Source, r.Language, r.Group = strings.Repeat("a", 64), "ko", "group"
		r.Order = [8]int{0, 1, 2, 3, 4, 5, 6, 7}
		for j := range r.Prediction.Probabilities {
			r.Prediction.Probabilities[j] = .125
		}
		r.Passed[0] = 16
	}
	return rows
}

func TestLegacyRowsRequireEverySamePlatformValue(t *testing.T) {
	want := testLegacyRows()
	if changed, err := reconcileLegacyRows("fixture", want, want, want); err != nil || len(changed) != 0 {
		t.Fatal(changed, err)
	}
	for _, name := range []string{"text", "source", "target", "logit", "signed-zero", "probability", "rank", "mask", "view"} {
		t.Run(name, func(t *testing.T) {
			got := append([]legacyRow(nil), want...)
			switch name {
			case "text":
				got[7].Text += " changed"
			case "source":
				got[7].Source = strings.Repeat("b", 64)
			case "target":
				got[7].Passed[0]--
			case "logit":
				got[7].Prediction.Logits[3] = 1e-8
			case "signed-zero":
				got[7].Prediction.Logits[3] = math.Float32frombits(1 << 31)
			case "probability":
				got[7].Prediction.Probabilities[3] += 1e-7
			case "rank":
				got[7].Order[1], got[7].Order[2] = got[7].Order[2], got[7].Order[1]
			case "mask":
				got[7].Prediction.Mask = 1
			case "view":
				got[7].View = got[6].View
			}
			if _, err := reconcileLegacyRows("fixture", want, want, got); err == nil {
				t.Fatal("changed row accepted")
			}
		})
	}
	for _, got := range [][]legacyRow{want[:511], append(append([]legacyRow(nil), want...), want[0])} {
		if _, err := reconcileLegacyRows("fixture", want, want, got); err == nil {
			t.Fatal("row denominator drift accepted")
		}
	}
}

func TestLegacyRowRequiresAllFieldsAndArrayPositions(t *testing.T) {
	raw, err := json.Marshal(testLegacyRows()[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := legacyRowShape(raw); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	fields["passed_cases_by_mask"] = json.RawMessage(`[16]`)
	short, _ := json.Marshal(fields)
	if legacyRowShape(short) == nil {
		t.Fatal("short array accepted")
	}
	delete(fields, "passed_cases_by_mask")
	missing, _ := json.Marshal(fields)
	if legacyRowShape(missing) == nil {
		t.Fatal("missing field accepted")
	}
	fields["passed_cases_by_mask"] = json.RawMessage(`[16,0,0,0,0,0,0,null]`)
	null, _ := json.Marshal(fields)
	if legacyRowShape(null) == nil {
		t.Fatal("null array entry accepted")
	}
}

func TestFrozenLegacyArchivesKeepAllOriginalDifferences(t *testing.T) {
	root := filepath.Join("..", "..")
	a, err := legacyManifest(filepath.Join(root, legacyOriginal), legacyOriginalManifest)
	if err != nil {
		t.Fatal(err)
	}
	b, err := legacyManifest(filepath.Join(root, legacyLinux), legacyLinuxManifest)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(filepath.Join(root, legacyOriginal, "initial-audit.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	lz, err := zip.OpenReader(filepath.Join(root, legacyLinux, "evidence.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer lz.Close()
	rows, ranks := 0, 0
	for name, pin := range a.Archives["initial-audit.zip"] {
		x, err := legacyMember(z, name, pin)
		if err != nil {
			t.Fatal(err)
		}
		y, err := legacyMember(lz, name, b.Archives["evidence.zip"][name])
		if err != nil {
			t.Fatal(err)
		}
		if name == "report.json" {
			var original, linux map[string]any
			if err = threecohort.Decode(x, &original); err != nil {
				t.Fatal(err)
			}
			if err = threecohort.Decode(y, &linux); err != nil {
				t.Fatal(err)
			}
			diff, err := reconcileLegacyReport(original, linux, linux)
			if err != nil || len(diff) != 101 {
				t.Fatal(len(diff), err)
			}
			continue
		}
		if !strings.HasSuffix(name, ".jsonl.gz") {
			if !bytes.Equal(x, y) {
				t.Fatal("frozen model changed", name)
			}
			bad := pin
			bad.SHA = strings.Repeat("0", 64)
			if _, err = legacyMember(z, name, bad); err == nil {
				t.Fatal("changed member digest accepted")
			}
			continue
		}
		original, err := decodeLegacyJournal(x)
		if err != nil {
			t.Fatal(err)
		}
		linux, err := decodeLegacyJournal(y)
		if err != nil {
			t.Fatal(err)
		}
		diff, err := reconcileLegacyRows(name, original, linux, linux)
		if err != nil {
			t.Fatal(err)
		}
		if unchanged, err := reconcileLegacyRows(name, original, original, original); err != nil || len(unchanged) != 0 {
			t.Fatal(unchanged, err)
		}
		rows += len(linux)
		ranks += len(diff)
	}
	if rows != 18432 || ranks != 272 {
		t.Fatal(rows, ranks)
	}
}

func TestLegacyReportRequiresDifferencePathsAndValues(t *testing.T) {
	original := map[string]any{"curve": []any{float64(3), float64(4)}}
	want := map[string]any{"curve": []any{float64(4), float64(4)}}
	changes, err := reconcileLegacyReport(original, want, want)
	if err != nil || len(changes) != 1 || changes[0].Path != "/curve[0]" {
		t.Fatal(changes, err)
	}
	for _, got := range []map[string]any{
		{"curve": []any{float64(5), float64(4)}},
		{"curve": []any{float64(3), float64(5)}},
		{"curve": []any{float64(4)}},
		{"curve": []any{float64(4), float64(4)}, "extra": true},
	} {
		if _, err := reconcileLegacyReport(original, want, got); err == nil {
			t.Fatal("unexpected summary differences accepted")
		}
	}
}

func TestLegacyBaselineUsesOnlyKnownGoPlatforms(t *testing.T) {
	for _, platform := range [][3]string{{"go1.27.1", "darwin", "arm64"}, {"go1.27.1", "linux", "amd64"}} {
		if _, err := legacyBaseline(platform[0], platform[1], platform[2]); err != nil {
			t.Fatal(err)
		}
	}
	for _, platform := range [][3]string{{"go1.27.2", "linux", "amd64"}, {"go1.27.1", "linux", "arm64"}, {"go1.27.1", "darwin", "amd64"}} {
		if _, err := legacyBaseline(platform[0], platform[1], platform[2]); err == nil {
			t.Fatal("unobserved platform accepted")
		}
	}
}

func TestLegacyJournalBoundedStrictDecoder(t *testing.T) {
	encode := func(rows []legacyRow) []byte {
		var b bytes.Buffer
		z := gzip.NewWriter(&b)
		e := json.NewEncoder(z)
		for _, r := range rows {
			if err := e.Encode(r); err != nil {
				t.Fatal(err)
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	rows := testLegacyRows()
	if got, err := decodeLegacyJournal(encode(rows)); err != nil || len(got) != 512 {
		t.Fatal(len(got), err)
	}
	for _, bad := range [][]byte{encode(rows[:511]), encode(append(rows, rows[0])), []byte("bad gzip")} {
		if _, err := decodeLegacyJournal(bad); err == nil {
			t.Fatal("bad journal accepted")
		}
	}
	rows = testLegacyRows()
	rows[0].Input = strings.Repeat("0", 64)
	if _, err := decodeLegacyJournal(encode(rows)); err == nil {
		t.Fatal("input digest drift accepted")
	}
	rows = testLegacyRows()
	rows[1].View = rows[0].View
	if _, err := decodeLegacyJournal(encode(rows)); err == nil {
		t.Fatal("duplicate view accepted")
	}
}
