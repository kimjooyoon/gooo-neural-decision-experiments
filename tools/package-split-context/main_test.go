package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFrozenAllowlistAndExtraFileRejection(t *testing.T) {
	t.Chdir("../..")
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := pack(bundle); err != nil {
		t.Fatal(err)
	}
	if err := verify(bundle, "", filepath.Join(t.TempDir(), "receipt.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "extra.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verify(bundle, "", filepath.Join(t.TempDir(), "rejected.json")); err == nil {
		t.Fatal("unlisted payload accepted")
	}
	if err := pack(bundle); err == nil {
		t.Fatal("existing bundle overwritten")
	}
}
func TestPrivateTextAndInvalidRevision(t *testing.T) {
	for _, s := range []string{"/Users/example/a", "Bearer credential", "hf_abcdefghijklmnopqrstuvwxyz123"} {
		if !privacy.MatchString(s) {
			t.Fatal("private text accepted")
		}
	}
	if err := verify("missing", "main", "unused"); err == nil {
		t.Fatal("mutable revision accepted")
	}
}

func TestEvidenceScopeAndSymlinkRejection(t *testing.T) {
	t.Chdir("../..")
	value, err := sources()
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Files) != 29 || len(value.Models) != 6 {
		t.Fatal("experimental file/model denominator differs")
	}
	for _, e := range value.Files {
		if e.Path == "native/report.json" || e.Path == "training/attempts.json" {
			t.Fatal("prior successful native evidence included as new v2 evidence")
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := pack(bundle); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("README.md", filepath.Join(bundle, "linked.md")); err != nil {
		t.Fatal(err)
	}
	if err := verify(bundle, "", filepath.Join(t.TempDir(), "rejected.json")); err == nil {
		t.Fatal("publication symlink accepted")
	}
}
