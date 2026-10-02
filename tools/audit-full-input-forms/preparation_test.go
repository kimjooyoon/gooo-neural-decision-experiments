package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisteredStorageReservesWholePhaseAndArchive(t *testing.T) {
	s := storageState{Available: 4 << 30, Whole: 768 << 20, Study: (768 << 20) - rawCap - publicAllowance}
	if !storageFits(s, true) {
		t.Fatal("exact budget rejected")
	}
	s.Study++
	if storageFits(s, true) {
		t.Fatal("future archive not reserved")
	}
	s.Study--
	s.Phase = 1024
	s.Study += 1024
	if !storageFits(s, false) || storageFits(s, true) {
		t.Fatal("existing phase accounting differs")
	}
	s.Available--
	if storageFits(s, false) {
		t.Fatal("free disk floor ignored")
	}
}

func TestJournalsRetainPrefixAndRejectOversizedRows(t *testing.T) {
	root := t.TempDir()
	used := int64(0)
	if err := saveFresh(root, "preexecution.json", map[string]int{"calls": 0}); err != nil {
		t.Fatal(err)
	}
	w, err := newJournal(root, "inputs.jsonl", &used)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Append(map[string]int{"index": 0}); err != nil {
		t.Fatal(err)
	}
	before := used
	if err := w.Append(strings.Repeat("x", maxRow)); err == nil || used != before {
		t.Fatal("oversized row changed prefix")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Append(1); err == nil {
		t.Fatal("write after close")
	}
	rows := 0
	if err := walkJournal(filepath.Join(root, "inputs.jsonl"), func(i int, b []byte) error { rows++; return nil }); err != nil || rows != 1 {
		t.Fatal(rows, err)
	}
	if _, err := newJournal(root, "inputs.jsonl", &used); err == nil {
		t.Fatal("original journal overwritten")
	}
	if err := saveFresh(root, "preexecution.json", 1); err == nil {
		t.Fatal("original metadata overwritten")
	}
	if err := saveFresh(root, "../escape.json", 1); err == nil {
		t.Fatal("escaped output")
	}
}

func TestBoundedReaderAndClosedJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.json")
	if err := os.WriteFile(path, []byte(`{"index":0}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := boundedFile(path, 2); err == nil {
		t.Fatal("file bound ignored")
	}
	if err := os.Symlink(path, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := boundedFile(filepath.Join(root, "link"), maxRow); err == nil {
		t.Fatal("symbolic input accepted")
	}
	var r struct {
		Index int `json:"index"`
	}
	for _, b := range []string{`{"index":0,"extra":1}`, `{"index":0} {}`, `{"index":0} garbage`, `{"index":1,"index":0}`} {
		if err := decodeExact([]byte(b), &r); err == nil {
			t.Fatal("invalid JSON accepted", b)
		}
	}
}

func TestCollisionRowsKeepEverySourceAndTargetIntersection(t *testing.T) {
	c := newCollisionIndex()
	a := inputRecord{Index: 0, ViewID: "en", Form: "original", Features: [2]string{"v3-a", "v4"}, Valid: 1, SourceSHA: "source-a", InputSHA: "input-a"}
	b := inputRecord{Index: 1, ViewID: "ko", Form: "original", Features: [2]string{"v3-b", "v4"}, Valid: 2, SourceSHA: "source-b", InputSHA: "input-b"}
	c.add(a)
	c.add(b)
	rows, stats := c.records()
	if stats != (collisionStats{Groups: 1, Conflicts: 1, Members: 2}) || len(rows) != 2 {
		t.Fatal(stats)
	}
	if rows[0].Member.SourceSHA != "source-a" || rows[1].Member.InputSHA != "input-b" || !rows[0].Conflict || rows[0].Intersection != 0 {
		t.Fatal(rows)
	}
	b.Form = "bare"
	c.add(b)
	b.Declined = true
	b.Form = "original"
	c.add(b)
	again, next := c.records()
	if next != stats || !sameJSON(rows, again) {
		t.Fatal("forms or declines mixed")
	}
	for _, row := range rows {
		encoded, err := json.Marshal(row)
		if err != nil || len(encoded) > maxRow {
			t.Fatal("unbounded member row")
		}
	}
}

func TestPreparationKeepsCompleteInputAndRejectsBadCoordinates(t *testing.T) {
	v := formView(t, "development", "ko")
	v.Target.Cases = 16
	v.Target.Passed[0] = 16
	r, err := prepareInput(v, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != v.Text || r.Valid != 1 || r.Declined || r.Features[0] == "" || r.Features[1] == "" {
		t.Fatal("incomplete original input")
	}
	for _, pair := range [][2]int{{-1, 0}, {0, -1}, {0, 11}} {
		if _, err := prepareInput(v, pair[0], pair[1]); err == nil {
			t.Fatal("unbounded coordinate")
		}
	}
}
