package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

// Separate arithmetic has its own publication identity; the original bundle stays frozen.
func packSeparate(output string) error {
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh separate publication required")
	}
	const local = "runs/own-three-separate-arm64-20261003"
	const remote = "runs/own-three-separate-linux-20261003"
	const comparison = "runs/own-three-separate-comparison-20261003.json"
	const protocol = "docs/full-input-separate-arithmetic-protocol-20261003.md"
	raw, err := os.ReadFile(comparison)
	if err != nil {
		return err
	}
	var outcome struct {
		Schema  string              `json:"schema"`
		Status  string              `json:"status"`
		Reports [2]threestudent.Pin `json:"collection_reports"`
	}
	if err = json.Unmarshal(raw, &outcome); err != nil || outcome.Schema != "gooo/separate-arithmetic-comparison/v1" || outcome.Status != "PASS" {
		return errors.New("successful complete comparison required")
	}
	for i, root := range []string{local, remote} {
		pin, err := threestudent.FilePin(filepath.Join(root, "report.json"))
		if err != nil || pin != outcome.Reports[i] {
			return errors.New("comparison report identity differs")
		}
	}
	raw, err = os.ReadFile(filepath.Join(local, "report.json"))
	if err != nil {
		return err
	}
	var collection struct {
		Sources map[string]threestudent.Pin `json:"computational_sources"`
	}
	if err = json.Unmarshal(raw, &collection); err != nil || len(collection.Sources) != 69 {
		return errors.New("complete collection source inventory required")
	}
	sources := map[string]string{"protocol.md": protocol}
	for name, want := range collection.Sources {
		if !filepath.IsLocal(name) {
			return errors.New("nonlocal source")
		}
		got, err := threestudent.FilePin(name)
		if err != nil || got != want {
			return errors.New("collection source changed")
		}
		sources[name] = name
	}
	groups := map[string]map[string]string{"sources.zip": sources}
	for archive, root := range map[string]string{"arm64.zip": local, "linux.zip": remote} {
		files, err := tree(root)
		if err != nil || len(files) != 85 {
			return errors.New("complete collection tree required")
		}
		groups[archive] = files
	}
	m := manifest{Schema: "gooo/full-input-separate-publication/v1", Files: map[string]threestudent.Pin{}, Archives: map[string]map[string]threestudent.Pin{}, Scope: "Two complete 73,728-prediction collections over the same 18,432 previously observed inputs, four arithmetic/layout lanes, all converted model artifacts, exact comparison and computational source. Original weights are unchanged; optimizer updates are zero. SDK/native adoption and additional cohorts remain future work."}
	for archive, files := range groups {
		m.Archives[archive] = map[string]threestudent.Pin{}
		for name, path := range files {
			if err = privacy(path); err != nil {
				return err
			}
			pin, err := threestudent.FilePin(path)
			if err != nil {
				return err
			}
			m.Archives[archive][name] = pin
			m.Decoded += pin.Bytes
		}
	}
	if m.Decoded > 64<<20 {
		return errors.New("64 MiB publication extent exceeded")
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	for _, name := range []string{"arm64.zip", "linux.zip", "sources.zip"} {
		if err = archive(filepath.Join(output, name), groups[name], m.Archives[name]); err != nil {
			return err
		}
	}
	files := map[string]string{
		"README.md":         "docs/full-input-separate-arithmetic-results-20261003.md",
		"protocol.md":       protocol,
		"comparison.json":   comparison,
		"arm64-report.json": filepath.Join(local, "report.json"),
		"linux-report.json": filepath.Join(remote, "report.json"),
	}
	models, err := tree(filepath.Join(local, "models"))
	if err != nil {
		return err
	}
	for name, path := range models {
		files["models/"+name] = path
	}
	for name, path := range files {
		if err = privacy(path); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		to := filepath.Join(output, name)
		if err = os.MkdirAll(filepath.Dir(to), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(to, raw, 0644); err != nil {
			return err
		}
	}
	public, err := tree(output)
	if err != nil {
		return err
	}
	for name, path := range public {
		pin, err := threestudent.FilePin(path)
		if err != nil {
			return err
		}
		m.Files[name] = pin
	}
	raw, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(output, "manifest.json"), append(raw, '\n'), 0644); err != nil {
		return err
	}
	return verify(output)
}
