package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicPayloadCountAndZipDigestVerification(t *testing.T) {
	if len(publicPaths()) != 38 {
		t.Fatal(len(publicPaths()))
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "source")
	if err := os.WriteFile(input, []byte("synthetic evidence\n"), 0600); err != nil {
		t.Fatal(err)
	}
	members := rawMembers("", "")
	for i := range members {
		members[i].Local = input
	}
	archiveName := filepath.Join(dir, "evidence.zip")
	entries, err := archive(archiveName, members)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(archiveName)
	if err != nil {
		t.Fatal(err)
	}
	if total, err := verifyArchive(raw, entries); err != nil || total != int64(478*len("synthetic evidence\n")) {
		t.Fatal(total, err)
	}
	entries[0].SHA = "wrong"
	if _, err := verifyArchive(raw, entries); err == nil {
		t.Fatal("changed member digest accepted")
	}
	var malformed bytes.Buffer
	w := zip.NewWriter(&malformed)
	header := &zip.FileHeader{Name: "../outside", Method: zip.Deflate}
	member, err := w.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = member.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = verifyArchive(malformed.Bytes(), entries); err == nil {
		t.Fatal("outside archive accepted")
	}
}

func TestFixedRawArchiveAllowlist(t *testing.T) {
	members := rawMembers("sdk", "native")
	if len(members) != 478 {
		t.Fatal(len(members))
	}
	seen := map[string]bool{}
	for _, m := range members {
		if seen[m.Public] {
			t.Fatal("duplicate", m.Public)
		}
		seen[m.Public] = true
	}
	if !seen["sdk/development-joint-qat_ternary.jsonl"] || !seen["native/schedule_branch-goal3-ko-selected-execution.json"] {
		t.Fatal("missing known actual captures")
	}
}
func TestPublicationPrivacyAndRegularFileBounds(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "input")
	if err := os.WriteFile(name, []byte("public synthetic Korean/English Gooo"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := scan(name); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("/Users/local/private input"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := scan(name); err == nil {
		t.Fatal("local path accepted")
	}
	if _, err := hashFile(dir); err == nil {
		t.Fatal("directory accepted")
	}
}

func TestEvidenceExtractionUsesVerifiedFixedInventory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "synthetic")
	if err := os.WriteFile(input, []byte("public evidence\n"), 0600); err != nil {
		t.Fatal(err)
	}
	members := rawMembers("", "")
	for i := range members {
		members[i].Local = input
	}
	entries, err := archive(filepath.Join(dir, "raw-evidence.zip"), members)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"archive_members": entries})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "publication-manifest.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "evidence")
	if err = extractEvidence(dir, output); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		value, e := os.ReadFile(filepath.Join(output, entry.Path))
		if e != nil || string(value) != "public evidence\n" {
			t.Fatal(entry.Path, e)
		}
	}
	if err = extractEvidence(dir, output); err == nil {
		t.Fatal("existing extraction accepted")
	}
	entries[0].SHA = "changed"
	raw, _ = json.Marshal(map[string]any{"archive_members": entries})
	if err = os.WriteFile(filepath.Join(dir, "publication-manifest.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "rejected")
	if err = extractEvidence(dir, bad); err == nil {
		t.Fatal("tampered inventory accepted")
	}
	if _, err = os.Stat(bad); !os.IsNotExist(err) {
		t.Fatal("invalid extraction wrote output")
	}
}

func TestIntermediateEditionExcludesUnfinishedNativeClaims(t *testing.T) {
	paths := stagePaths()
	if len(paths) != 28 || !paths["joint/models/fp32/model.json"] || paths["native-report.json"] || paths["raw-evidence.zip"] || paths["provenance/joint-fp32.ttl"] {
		t.Fatal("intermediate scope differs", len(paths))
	}
	output := filepath.Join(t.TempDir(), "rejected")
	if err := verifyStage("unused", "main", output); err == nil {
		t.Fatal("mutable stage revision accepted")
	}
	if err := fetchBundle(output, "main", "invalid"); err == nil {
		t.Fatal("unpinned download accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("invalid pins created output")
	}
}
