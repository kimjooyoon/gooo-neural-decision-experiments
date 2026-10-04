package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func TestHistoricalSourcePinSurvivesWorkingFileChange(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", repo}, args...)...)
		c.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid")
		b, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture git: %v: %s", err, b)
		}
		return strings.TrimSpace(string(b))
	}
	git("init", "--quiet")
	file := filepath.Join(repo, "model.go")
	if err := os.WriteFile(file, []byte("package fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	pin, err := threestudent.FilePin(file)
	if err != nil {
		t.Fatal(err)
	}
	git("add", "model.go")
	git("-c", "core.hooksPath=/dev/null", "commit", "--quiet", "-m", "Original source")
	revision := git("rev-parse", "HEAD")
	if err := os.WriteFile(file, []byte("package fixture\n// bounded loader added\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if pinned(file, pin) == nil {
		t.Fatal("strict working-source check accepted a change")
	}
	if err := pinnedGitSource(repo, revision, "model.go", pin); err != nil {
		t.Fatal(err)
	}
	bad := pin
	bad.SHA = strings.Repeat("0", 64)
	if pinnedGitSource(repo, revision, "model.go", bad) == nil {
		t.Fatal("wrong historical digest accepted")
	}
	bad = pin
	bad.Bytes++
	if pinnedGitSource(repo, revision, "model.go", bad) == nil {
		t.Fatal("wrong historical extent accepted")
	}
	for _, name := range []string{"../model.go", "missing.go"} {
		if pinnedGitSource(repo, revision, name, pin) == nil {
			t.Fatal("invalid/missing historical source accepted")
		}
	}
	if pinnedGitSource(repo, strings.Repeat("x", 40), "model.go", pin) == nil {
		t.Fatal("non-hex revision accepted")
	}
}

func TestSourceChangesRetainAddedRemovedAndChangedPins(t *testing.T) {
	x := threestudent.Pin{SHA: strings.Repeat("1", 64), Bytes: 12}
	y := threestudent.Pin{SHA: strings.Repeat("2", 64), Bytes: 13}
	a := map[string]threestudent.Pin{"same": x, "changed": x, "removed": x}
	b := map[string]threestudent.Pin{"same": x, "changed": y, "added": y}
	changes := sourceChanges(a, b)
	if len(changes) != 3 || changes[0]["file"] != "added" || changes[0]["before"] != nil || changes[0]["after"] != y || changes[1]["before"] != x || changes[1]["after"] != y || changes[2]["after"] != nil {
		t.Fatal(changes)
	}
	if len(sourceChanges(a, a)) != 0 {
		t.Fatal("equal source reported as changed")
	}
}
