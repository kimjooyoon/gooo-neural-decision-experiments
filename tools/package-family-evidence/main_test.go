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

func TestOptimizedInventoryIsBounded(t *testing.T) {
	t.Chdir("../..")
	for _, kind := range []string{"optimized-feature", "optimized-main"} {
		names, err := namesKind(kind)
		if err != nil || len(names) != 1105 {
			t.Fatal("optimized frozen inventory differs", kind, err)
		}
	}
	if names, err := namesKind("compound-main"); err != nil || len(names) != 674 {
		t.Fatal("compound frozen inventory differs", err)
	}
	if names, err := namesKind("unfixed-sdk"); err != nil || len(names) != 593 {
		t.Fatal("unfixed SDK allowlist differs", err)
	}
	if names, err := namesKind("unfixed-native-feature"); err != nil || len(names) != 104 {
		t.Fatal("native unfixed pilot allowlist differs", err)
	}
	if names, err := namesKind("unfixed-native-main"); err != nil || len(names) != 104 {
		t.Fatal("native main unfixed pilot allowlist differs", err)
	}
	if _, err := namesKind("untrusted"); err == nil {
		t.Fatal("unknown evidence inventory accepted")
	}
}

func TestFrozenLocalPublicationDeterministicAndRejectsExtra(t *testing.T) {
	// Full frozen archive reproduction is a separate CI command. Race tests
	// exercise the filesystem contract without recompressing 30 MiB three times.
	dir := t.TempDir()
	files := map[string][]byte{"report.json": []byte("synthetic record")}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), files["report.json"], 0600); err != nil {
		t.Fatal(err)
	}
	if count, err := checkLocal(dir, files); err != nil || count != 1 {
		t.Fatal("local inventory differs", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "private.txt"), []byte("unexpected"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := checkLocal(dir, files); err == nil {
		t.Fatal("extra local payload accepted")
	}
}
