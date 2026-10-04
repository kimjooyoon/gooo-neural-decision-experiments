package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func pinnedGitSource(repo, revision, name string, want threestudent.Pin) error {
	if len(revision) != 40 || !filepath.IsLocal(name) || want.Bytes <= 0 || want.Bytes > 1<<20 || len(want.SHA) != 64 {
		return errors.New("bounded source revision/file required")
	}
	for _, c := range revision {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return errors.New("lowercase Git revision required")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	object := revision + ":" + filepath.ToSlash(name)
	size, err := exec.CommandContext(ctx, "git", "-C", repo, "cat-file", "-s", object).Output()
	if err != nil {
		return fmt.Errorf("source object unavailable: %s: %w", name, err)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(size)), 10, 64)
	if err != nil || n != want.Bytes {
		return fmt.Errorf("source object extent differs: %s", name)
	}
	raw, err := exec.CommandContext(ctx, "git", "-C", repo, "cat-file", "blob", object).Output()
	if err != nil || int64(len(raw)) != want.Bytes || threecohort.SHA(raw) != want.SHA {
		return fmt.Errorf("source object digest differs: %s", name)
	}
	return nil
}

func sourceChanges(a, b map[string]threestudent.Pin) []map[string]any {
	names := map[string]bool{}
	for name := range a {
		names[name] = true
	}
	for name := range b {
		names[name] = true
	}
	var sorted []string
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	changes := make([]map[string]any, 0)
	for _, name := range sorted {
		x, old := a[name]
		y, current := b[name]
		if old && current && x == y {
			continue
		}
		row := map[string]any{"file": name, "before": nil, "after": nil}
		if old {
			row["before"] = x
		}
		if current {
			row["after"] = y
		}
		changes = append(changes, row)
	}
	return changes
}
