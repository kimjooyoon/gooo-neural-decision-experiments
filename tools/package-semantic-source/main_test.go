package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceKeepsSourceOnlyBoundary(t *testing.T) {
	valid := `{"schema":"gooo/semantic-source-v3-implementation/v1","status":"NATIVE_MAIN_VERIFIED","native_main_revision":"` + strings.Repeat("a", 40) + `","feature_version":"semantic_context_intent_v3","new_optimizer_updates":0,"new_model_exports":0}`
	if err := validateEvidence([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{strings.Replace(valid, `"new_optimizer_updates":0`, `"new_optimizer_updates":1`, 1), strings.Replace(valid, `,"new_model_exports":0`, "", 1), strings.Replace(valid, "NATIVE_MAIN_VERIFIED", "PENDING", 1), strings.Replace(valid, strings.Repeat("a", 40), "short", 1), valid + "hf_" + strings.Repeat("a", 25)} {
		if err := validateEvidence([]byte(invalid)); err == nil {
			t.Fatal("invalid implementation evidence accepted")
		}
	}
}

func TestSourceArchiveRejectsUnsafePathsAndPrivateBytes(t *testing.T) {
	build := func(extra, content string) []byte {
		var raw bytes.Buffer
		writer := zip.NewWriter(&raw)
		for _, name := range []string{"go.mod", "source-provenance.json", "semantic_features.go", "pathplan/source_features.go", extra} {
			if name == "" {
				continue
			}
			file, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = file.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return raw.Bytes()
	}
	if err := validateSourceArchive(build("", "package decision\n")); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range [][]byte{nil, build("../escape", "ok"), build("/absolute", "ok"), build("go.mod", "duplicate"), build("", "hf_"+strings.Repeat("a", 25))} {
		if err := validateSourceArchive(invalid); err == nil {
			t.Fatal("unsafe archive accepted")
		}
	}
}

func TestPackagingRequiresFreshDirectoryAndDoesNotUpload(t *testing.T) {
	output := filepath.Join(t.TempDir(), "public")
	payloads := map[string][]byte{"README.md": []byte("Public source only.\n")}
	if err := writePayloads(output, payloads); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(output, "publication-manifest.json"))
	if err != nil || !bytes.Contains(manifest, []byte(digest(payloads["README.md"]))) {
		t.Fatal("payload bytes not bound", err)
	}
	if err = writePayloads(output, payloads); err == nil {
		t.Fatal("existing evidence overwritten")
	}
	if err = writePayloads(filepath.Join(t.TempDir(), "private"), map[string][]byte{"implementation.json": []byte("github_pat_sensitive")}); err == nil {
		t.Fatal("private bytes packaged")
	}
}
