package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
	review := filepath.Join(t.TempDir(), "review")
	if err := assembleVersion(review, true); err != nil {
		t.Fatal(err)
	}
	files, _, err = local(review)
	if err != nil || len(files) != len(reviewedSources())+1 {
		t.Fatalf("native review: %d %v", len(files), err)
	}
}

func TestMainAnnouncementPreservesDirectPayload(t *testing.T) {
	t.Chdir("../..")
	direct := filepath.Join(t.TempDir(), "direct")
	main := filepath.Join(t.TempDir(), "main")
	if err := assembleVersions(direct, false, true); err != nil {
		t.Fatal(err)
	}
	// Main availability implies the direct and reviewed inventories.
	if err := assemblePublication(main, false, false, true); err != nil {
		t.Fatal(err)
	}
	files, _, err := local(main)
	if err != nil || len(files) != len(directSources())+1 {
		t.Fatalf("main inventory: %d %v", len(files), err)
	}
	note, err := os.ReadFile("docs/hf-typed-path-v1-main-promotion.md")
	if err != nil {
		t.Fatal(err)
	}
	weights := 0
	for _, file := range files {
		if file.Path == "publication-manifest.json" {
			continue
		}
		before, err := os.ReadFile(filepath.Join(direct, file.Path))
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(filepath.Join(main, file.Path))
		if err != nil {
			t.Fatal(err)
		}
		if file.Path == "README.md" {
			if !bytes.Equal(after, append(append(before, '\n'), note...)) {
				t.Fatal("main announcement did not append exactly the public note")
			}
			continue
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("main announcement changed frozen payload %s", file.Path)
		}
		if strings.HasSuffix(file.Path, "/weights.bin") {
			weights++
		}
	}
	if weights != 9 {
		t.Fatalf("expected nine unchanged weight bundles, got %d", weights)
	}
}
