package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJournalCapPreservesCompletedPrefix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.jsonl")
	used := rawCap - metadataReserve - 8
	j, err := openJournal(path, &used)
	if err != nil {
		t.Fatal(err)
	}
	defer j.file.Close()
	if err = j.appendRaw([]byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	before := j.pin()
	if err = j.appendRaw([]byte(`{"a":2}`)); err == nil {
		t.Fatal("global raw cap ignored")
	}
	if j.pin() != before || used != rawCap-metadataReserve {
		t.Fatal("completed prefix changed on cap decline")
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "{\"a\":1}\n" {
		t.Fatal("recorded prefix lost")
	}
}
func TestScanRejectsPrivateAndOversizeEvidence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.jsonl")
	if err := os.WriteFile(path, []byte("{\"path\":\"/Users/private\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := scan(path, func([]byte) error { return nil }); err == nil {
		t.Fatal("private evidence accepted")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := scan(link, func([]byte) error { return nil }); err == nil {
		t.Fatal("symlink evidence accepted")
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(rawCap + 1); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := scan(path, func([]byte) error { return nil }); err == nil {
		t.Fatal("oversize evidence accepted")
	}
}
func TestMetadataNeverOverwritesEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preexecution.json")
	if err := save(path, map[string]int{"original": 1}); err != nil {
		t.Fatal(err)
	}
	if err := save(path, map[string]int{"original": 2}); err == nil {
		t.Fatal("immutable phase evidence overwritten")
	}
	var v struct {
		Original int `json:"original"`
	}
	if err := readJSON(path, &v); err != nil || v.Original != 1 {
		t.Fatal("original evidence changed")
	}
}
