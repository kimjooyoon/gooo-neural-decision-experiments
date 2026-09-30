package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	m := manifest{Schema: "gooo/public-compiler-model-allowlist/v1", PrivateTextScanned: true, BinaryProvenance: "public synthetic data"}
	for name := range fixedPaths() {
		raw := []byte("public fixture")
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), raw, 0644); err != nil {
			t.Fatal(err)
		}
		m.Files = append(m.Files, artifact{name, digest(raw), len(raw)})
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "publication-manifest.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestFixedLocalBundle(t *testing.T) {
	files, _, err := readLocal(fixture(t))
	if err != nil || len(files) != 23 {
		t.Fatalf("files=%d err=%v", len(files), err)
	}
}

func TestLocalBundleRejectsChangedAndExtraFiles(t *testing.T) {
	for _, name := range []string{"README.md", "extra.txt"} {
		t.Run(name, func(t *testing.T) {
			root := fixture(t)
			if err := os.WriteFile(filepath.Join(root, name), []byte("changed"), 0644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := readLocal(root); err == nil {
				t.Fatal("invalid bundle accepted")
			}
		})
	}
}
