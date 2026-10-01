package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveRejectsChangedDigestAndExtraEntries(t *testing.T) {
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	w := tar.NewWriter(z)
	raw := []byte("synthetic evidence")
	if err := w.WriteHeader(&tar.Header{Name: "safe.json", Size: int64(len(raw)), Mode: 0644, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	e := []entry{{"safe.json", hash(raw), len(raw)}}
	if err := checkArchive(buf.Bytes(), e); err != nil {
		t.Fatal(err)
	}
	e[0].SHA = hash([]byte("changed"))
	if err := checkArchive(buf.Bytes(), e); err == nil {
		t.Fatal("changed digest accepted")
	}
	if err := checkArchive(buf.Bytes(), nil); err == nil {
		t.Fatal("extra entry accepted")
	}
}

func TestFrozenLocalPublicationDeterministicAndRejectsExtra(t *testing.T) {
	t.Chdir("../..")
	dir := filepath.Join(t.TempDir(), "bundle")
	if err := execute(dir, "", "", true); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "verified.json")
	if err := execute(dir, "", output, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "private.txt"), []byte("unexpected"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := execute(dir, "", output, false); err == nil {
		t.Fatal("extra local payload accepted")
	}
}
