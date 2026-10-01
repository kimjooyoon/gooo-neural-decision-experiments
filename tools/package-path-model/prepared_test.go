package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreparedPublicationPreservesFrozenModels(t *testing.T) {
	t.Chdir("../..")
	historical := filepath.Join(t.TempDir(), "historical")
	prepared := filepath.Join(t.TempDir(), "prepared")
	if err := assemblePublication(historical, false, false, true); err != nil {
		t.Fatal(err)
	}
	if err := assembleEdition(prepared, false, false, false, true); err != nil {
		t.Fatal(err)
	}
	files, _, err := local(prepared)
	if err != nil || len(files) != 222 {
		t.Fatalf("prepared inventory: %d %v", len(files), err)
	}
	note, err := os.ReadFile("docs/hf-typed-path-v1-prepared-native.md")
	if err != nil {
		t.Fatal(err)
	}
	weights := 0
	for name := range directSources() {
		before, err := os.ReadFile(filepath.Join(historical, name))
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(filepath.Join(prepared, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "README.md" {
			before = append(append(before, '\n'), note...)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("changed frozen artifact %s", name)
		}
		if strings.HasSuffix(name, "/weights.bin") {
			weights++
		}
	}
	if weights != 9 {
		t.Fatalf("changed weight inventory: %d", weights)
	}
}

func TestPreparedEvidenceRejectsChangedAccounting(t *testing.T) {
	t.Chdir("../..")
	if err := auditPreparedRepository(); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for name, source := range preparedSources() {
		if !strings.HasPrefix(name, "native-prepared/") && !strings.HasPrefix(name, "conditional-cohort/") && !strings.HasPrefix(name, "positioned-random/models/") {
			continue
		}
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	reportFile := filepath.Join(root, "native-prepared/report.json")
	original, err := os.ReadFile(reportFile)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		edit func(*pairedReport)
	}{
		{"relabel-partial", func(r *pairedReport) { r.Cells[0].Search.Status = "PASS" }},
		{"alter-pair-source", func(r *pairedReport) { r.Cells[0].SourceSHA = strings.Repeat("0", 64) }},
		{"remove-timing", func(r *pairedReport) {
			for i := range r.Cells {
				if r.Cells[i].Arm != "offline" {
					r.Cells[i].Search.Selection.Receipts[0].PredictNS = 0
					break
				}
			}
		}},
		{"inflate-structure", func(r *pairedReport) { r.Cells[0].Matched++ }},
		{"change-median", func(r *pairedReport) { r.Summaries[0].Stage += 1 }},
		{"lose-order", func(r *pairedReport) { r.Cells[0].Position = 1 }},
		{"invent-predictions", func(r *pairedReport) { r.Predictions++ }},
		{"inflate-functional", func(r *pairedReport) { r.Cells[0].Functional = 100 }},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			var report pairedReport
			if err := json.Unmarshal(original, &report); err != nil {
				t.Fatal(err)
			}
			item.edit(&report)
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(reportFile, raw, 0644); err != nil {
				t.Fatal(err)
			}
			if err := auditPreparedDirectory(filepath.Join(root, "native-prepared"), filepath.Join(root, "conditional-cohort"), ""); err == nil {
				t.Fatal("altered evidence accepted")
			}
		})
	}
	if err := os.WriteFile(reportFile, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := auditPreparedDirectory(filepath.Join(root, "native-prepared"), filepath.Join(root, "conditional-cohort"), ""); err != nil {
		t.Fatal(err)
	}
	goFile := filepath.Join(root, "native-prepared/independent-go-tests.jsonl")
	events, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatal(err)
	}
	// Duplicate actual observations cannot inflate the reported completeness.
	index := bytes.Index(events, []byte("GOOO_OBSERVATION "))
	start := bytes.LastIndex(events[:index], []byte("\n")) + 1
	end := start + bytes.IndexByte(events[start:], '\n') + 1
	if err := os.WriteFile(goFile, append(append([]byte(nil), events...), events[start:end]...), 0644); err != nil {
		t.Fatal(err)
	}
	if err := auditPreparedDirectory(filepath.Join(root, "native-prepared"), filepath.Join(root, "conditional-cohort"), ""); err == nil {
		t.Fatal("duplicate arithmetic observation accepted")
	}
}
