package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyntheticAppendixHasExactInventoryAndRejectsChangedFiles(t *testing.T) {
	t.Chdir("../..")
	dir := filepath.Join(t.TempDir(), "appendix")
	receipt := filepath.Join(t.TempDir(), "receipt.json")
	if err := pack(dir); err != nil {
		t.Fatal(err)
	}
	if err := verify(dir, "", receipt); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("not allowlisted"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verify(dir, "", receipt); err == nil {
		t.Fatal("unlisted appendix file accepted")
	}
	dir = filepath.Join(t.TempDir(), "appendix")
	if err := pack(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verify(dir, "", receipt); err == nil {
		t.Fatal("changed evidence accepted")
	}
	if err := verify(dir, "not-a-revision", receipt); err == nil {
		t.Fatal("mutable or invalid remote revision accepted")
	}
}
