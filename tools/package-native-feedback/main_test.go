package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeAppendixPrivacyAndUnknownRevision(t *testing.T) {
	if !privateText.MatchString("/Users/fixture/source") || !privateText.MatchString("hf_abcdefghijklmnopqrstuvwxyz") || privateText.MatchString(repository) {
		t.Fatal("public appendix privacy filter differs")
	}
	if err := verify(t.TempDir(), "main", "ignored"); err == nil {
		t.Fatal("mutable public revision accepted")
	}
}
func TestNativeAppendixRejectsExtraFiles(t *testing.T) {
	t.Chdir("../..")
	root := t.TempDir()
	bundle := filepath.Join(root, "bundle")
	if err := pack(bundle); err != nil {
		t.Fatal(err)
	}
	if err := verify(bundle, "", filepath.Join(root, "verification.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "unexpected.txt"), []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verify(bundle, "", filepath.Join(root, "rejected.json")); err == nil {
		t.Fatal("unknown evidence file accepted")
	}
}
