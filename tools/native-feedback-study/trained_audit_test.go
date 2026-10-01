package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestTrainedAuditFrozenEditionsAndChangedCapture(t *testing.T) {
	t.Chdir("../..")
	for _, name := range []string{"feedback-trained-native-20261001", "feedback-trained-main-smokes-20261001"} {
		original := filepath.Join("runs", name)
		value, err := auditTrainedDogfood(original)
		if err != nil || value["new_model_predictions"] != 0 || value["candidate_attempts_reinterpreted"] != 384 {
			t.Fatalf("%s: %v", name, err)
		}
		dir := t.TempDir()
		if err := filepath.WalkDir(original, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(original, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dir, rel)
			if entry.IsDir() {
				return os.MkdirAll(target, 0700)
			}
			raw, err := read(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, raw, 0600)
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := auditTrainedDogfood(dir); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "en-fp32.json")
		raw, err := read(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(raw, ' '), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := auditTrainedDogfood(dir); err == nil {
			t.Fatal("changed native capture accepted")
		}
	}
}
