package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFixedRawInventoryAndPayloads(t *testing.T) {
	members, err := rawMembers()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, m := range members {
		if seen[m.Public] || m.Public == "" {
			t.Fatal("unsafe raw member inventory")
		}
		seen[m.Public] = true
	}
	if len(requiredFiles()) != 48 {
		t.Fatal("fixed 48-payload publication changed")
	}
	if !seen["teacher/teacher-sessions.jsonl"] || !seen["teacher/states.jsonl"] || !seen["native-first-attempt/assignment_reference-goal0-en-offline.json"] || !seen["source/training/train_own_joint_feedback_v2.py"] {
		t.Fatal("required raw provenance missing")
	}
}
func TestPrivacyAndRegularFileBounds(t *testing.T) {
	dir := t.TempDir()
	safe := filepath.Join(dir, "safe.json")
	if err := os.WriteFile(safe, []byte("{\"source\":\"public synthetic\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := scan(safe); err != nil {
		t.Fatal(err)
	}
	unsafe := filepath.Join(dir, "private.json")
	if err := os.WriteFile(unsafe, []byte("/Users/private-example/data"), 0600); err != nil {
		t.Fatal(err)
	}
	if scan(unsafe) == nil {
		t.Fatal("private host path accepted")
	}
	link := filepath.Join(dir, "symlink")
	if err := os.Symlink(safe, link); err != nil {
		t.Fatal(err)
	}
	if _, err := hashFile(link); err == nil {
		t.Fatal("publication symlink accepted")
	}
	if _, err := archive(filepath.Join(dir, "duplicate.zip"), []member{{"same", safe}, {"same", safe}}); err == nil {
		t.Fatal("duplicate archive entry accepted")
	}
}
