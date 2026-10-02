package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func testArchive(t *testing.T, names ...string) ([]*zip.File, map[string]entry) {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	expected := map[string]entry{}
	for _, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0644)
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		raw := []byte("synthetic captured evidence\n")
		if _, err = f.Write(raw); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		expected[name] = entry{Path: name, Bytes: int64(len(raw)), SHA: hex.EncodeToString(sum[:])}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return r.File, expected
}

func TestIndependentManifestPin(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "publication-manifest.json")
	if err := os.WriteFile(p, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sha, err := pinManifest(dir, "")
	if err != nil || len(sha) != 64 {
		t.Fatal("manifest hash unavailable", err)
	}
	if _, err = pinManifest(dir, sha); err != nil {
		t.Fatal(err)
	}
	if _, err = pinManifest(dir, "bad"); err == nil {
		t.Fatal("malformed independent pin accepted")
	}
	if err = os.WriteFile(p, []byte("{\"tampered\":true}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = pinManifest(dir, sha); err == nil {
		t.Fatal("tampered manifest accepted")
	}
}

func TestVerifiedStreamExtraction(t *testing.T) {
	files, expected := testArchive(t, "teacher/synthetic.jsonl", "source/example.go")
	dest := filepath.Join(t.TempDir(), "evidence")
	if err := extractEvidence(files, expected, dest); err != nil {
		t.Fatal(err)
	}
	for name, want := range expected {
		actual, err := hashFile(filepath.Join(dest, filepath.FromSlash(name)))
		if err != nil || actual.SHA != want.SHA || actual.Bytes != want.Bytes {
			t.Fatal("extracted member differs", err)
		}
	}
	if extractEvidence(files, expected, dest) == nil {
		t.Fatal("existing evidence target overwritten")
	}
}

func TestExtractionRejectsUnsafeEntries(t *testing.T) {
	for _, name := range []string{"../escaped", "/absolute", "dir/../escaped", "dir\\escaped", "C:/escaped", "./escaped"} {
		t.Run(name, func(t *testing.T) {
			files, expected := testArchive(t, name)
			if extractEvidence(files, expected, filepath.Join(t.TempDir(), "evidence")) == nil {
				t.Fatal("unsafe member extracted")
			}
		})
	}
	files, expected := testArchive(t, "same", "same")
	if extractEvidence(files, expected, filepath.Join(t.TempDir(), "duplicate")) == nil {
		t.Fatal("duplicate member extracted")
	}
	files, expected = testArchive(t, "synthetic")
	a := expected["synthetic"]
	a.SHA = "wrong"
	expected["synthetic"] = a
	if extractEvidence(files, expected, filepath.Join(t.TempDir(), "tampered")) == nil {
		t.Fatal("extracted bytes not independently hashed")
	}
	a.Bytes = 256<<20 + 1
	expected["synthetic"] = a
	if extractEvidence(files, expected, filepath.Join(t.TempDir(), "oversize")) == nil {
		t.Fatal("oversized member extracted")
	}
}
