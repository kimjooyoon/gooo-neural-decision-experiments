package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicPayloadCountAndZipDigestVerification(t *testing.T) {
	if len(publicPaths()) != 35 {
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
	if total, err := verifyArchive(raw, entries); err != nil || total != int64(327*len("synthetic evidence\n")) {
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
	if len(members) != 327 {
		t.Fatal(len(members))
	}
	seen := map[string]bool{}
	for _, m := range members {
		if seen[m.Public] {
			t.Fatal("duplicate", m.Public)
		}
		seen[m.Public] = true
	}
	if !seen["sdk/development-v3-qat_ternary.jsonl"] || !seen["native/branch_schedule-goal3-ko-selected-execution.json"] {
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
