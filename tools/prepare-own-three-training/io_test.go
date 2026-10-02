package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func TestCapPreservesCompletedPrefix(t *testing.T) {
	used := threestudent.RawCap - (1 << 20) - 3
	var b bytes.Buffer
	if err := writeBounded(&b, []byte("abc"), &used); err != nil {
		t.Fatal(err)
	}
	if err := writeBounded(&b, []byte("d"), &used); err == nil || b.String() != "abc" {
		t.Fatal("cap did not preserve exact prefix")
	}
}
func TestEvidenceCannotBeOverwrittenOrPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meta.json")
	var used int64
	if err := saveJSON(path, map[string]string{"status": "public"}, &used); err != nil {
		t.Fatal(err)
	}
	if err := saveJSON(path, map[string]string{"status": "replacement"}, &used); err == nil {
		t.Fatal("overwrite accepted")
	}
	other := filepath.Join(t.TempDir(), "private.json")
	if err := saveJSON(other, map[string]string{"path": "/Users/private/project"}, &used); err == nil {
		t.Fatal("private text accepted")
	}
	if _, err := os.Stat(other); !os.IsNotExist(err) {
		t.Fatal("private bytes written")
	}
}
