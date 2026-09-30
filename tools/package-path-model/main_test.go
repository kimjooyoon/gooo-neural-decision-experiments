package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFixedPublicPathBundleAndExtraFileRejection(t *testing.T) {
	t.Chdir("../..")
	root := filepath.Join(t.TempDir(), "bundle")
	if err := assemble(root); err != nil {
		t.Fatal(err)
	}
	files, _, err := local(root)
	if err != nil || len(files) != len(sources())+1 {
		t.Fatalf("bundle: %d %v", len(files), err)
	}
	if err := assemble(root); err == nil {
		t.Fatal("existing public bundle was overwritten")
	}
	extra := filepath.Join(root, "extra.txt")
	if err := os.WriteFile(extra, []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := local(root); err == nil {
		t.Fatal("extra bundle file was accepted")
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("altered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := local(root); err == nil {
		t.Fatal("modified public artifact was accepted")
	}
}
