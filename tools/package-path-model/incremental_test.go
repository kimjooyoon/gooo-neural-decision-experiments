package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncrementalBundlePreservesAllPreviousEvidence(t *testing.T) {
	t.Chdir("../..")
	root := filepath.Join(t.TempDir(), "incremental")
	if err := assembleIncrementalEdition(root, false, false, false, false, true); err != nil {
		t.Fatal(err)
	}
	files, _, err := local(root)
	if err != nil || len(files) != 225 {
		t.Fatalf("incremental fixed inventory: %d %v", len(files), err)
	}
	note, err := os.ReadFile("docs/hf-typed-path-v1-incremental.md")
	if err != nil {
		t.Fatal(err)
	}
	weights := 0
	for name := range preparedSources() {
		before, err := os.ReadFile(filepath.Join("publication/hf-typed-path-v1-prepared-native", name))
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "README.md" {
			before = append(append(before, '\n'), note...)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("changed previous artifact %s", name)
		}
		if strings.HasSuffix(name, "/weights.bin") {
			weights++
		}
	}
	if weights != 9 {
		t.Fatal("weight inventory changed")
	}
}

func TestIncrementalAuditRejectsAlteredCasesAndEvidence(t *testing.T) {
	t.Chdir("../..")
	root := t.TempDir()
	cohort := filepath.Join(root, "cohort")
	if err := os.Mkdir(cohort, 0755); err != nil {
		t.Fatal(err)
	}
	for name := range incrementalPins {
		raw, err := os.ReadFile(filepath.Join(incrementalRun, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, language := range []string{"en", "ko"} {
		name := language + "-budget-64.json"
		raw, err := os.ReadFile(filepath.Join(conditionalCohort, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cohort, name), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := auditIncrementalDirectory(root, cohort, ""); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(cohort, "en-budget-64.json")
	original, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(original, []byte(`"expected": 24`), []byte(`"expected": 998`), 1)
	if bytes.Equal(changed, original) {
		t.Fatal("fixture edit had no effect")
	}
	if err := os.WriteFile(name, changed, 0644); err != nil {
		t.Fatal(err)
	}
	// Report bytes remain exactly pinned: the arena audit must detect changed cases.
	if err := auditIncrementalDirectory(root, cohort, ""); err == nil {
		t.Fatal("changed finite cases were accepted")
	}
	if err := os.WriteFile(name, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "summary.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := auditIncrementalDirectory(root, cohort, ""); err == nil {
		t.Fatal("changed fixed evidence was accepted")
	}
}
