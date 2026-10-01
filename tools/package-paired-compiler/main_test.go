package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnModelPublicBundleIsReproducibleAndBounded(t *testing.T) {
	t.Chdir("../..")
	first, second := filepath.Join(t.TempDir(), "one"), filepath.Join(t.TempDir(), "two")
	if err := pack(first); err != nil {
		t.Fatal(err)
	}
	if err := pack(second); err != nil {
		t.Fatal(err)
	}
	a, _ := read(filepath.Join(first, "publication-manifest.json"))
	b, _ := read(filepath.Join(second, "publication-manifest.json"))
	if hash(a) != hash(b) {
		t.Fatal("nondeterministic model publication manifest")
	}
	var value manifest
	if err := json.Unmarshal(a, &value); err != nil || len(value.Files) != 30 || value.Prefix != "research/paired-compiler-20261002" {
		t.Fatal("allowlist count differs")
	}
	entries := map[string]artifact{}
	for _, item := range value.Archive {
		if _, found := entries[item.Path]; found {
			t.Fatal("duplicate archive entry")
		}
		entries[item.Path] = item
	}
	f, err := os.Open(filepath.Join(first, "evidence.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tarReader := tar.NewReader(gz)
	count := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		item, found := entries[header.Name]
		if !found || header.Size != int64(item.Bytes) || header.Typeflag != tar.TypeReg {
			t.Fatal("archive inventory differs")
		}
		raw, err := io.ReadAll(tarReader)
		if err != nil || hash(raw) != item.SHA {
			t.Fatal("archive bytes differ")
		}
		count++
	}
	if count != len(value.Archive) {
		t.Fatal("incomplete archive")
	}
	if err := pack(first); err == nil {
		t.Fatal("existing publication was replaced")
	}
}

func TestPublicModelPrivacyPatterns(t *testing.T) {
	for _, text := range []string{"/Users/example/private", "Bearer exampleToken", "hf_abcdefghijklmnopqrstuvwxyz1234"} {
		if !privateText.MatchString(text) {
			t.Fatal("private marker missed")
		}
	}
	if privateText.MatchString("owntraining://activity/choosepath sha256:123") {
		t.Fatal("synthetic source rejected")
	}
}
