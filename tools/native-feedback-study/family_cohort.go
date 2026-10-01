package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/familystudy"
)

func writeFamilyCohort(dir string) error {
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		return errors.New("fresh cohort directory required")
	}
	rows, err := familystudy.Cohort()
	if err != nil {
		return err
	}
	var raw []byte
	sources, groups := map[string]bool{}, map[string]bool{}
	ambiguous := 0
	for _, r := range rows {
		line, err := json.Marshal(r)
		if err != nil {
			return err
		}
		raw = append(raw, line...)
		raw = append(raw, '\n')
		sources[hash([]byte(r.Source))] = true
		groups[r.Family+"/"+r.IntentionLabel+"/"+r.Source] = true
		if len(r.FiniteBestLabels) > 1 {
			ambiguous++
		}
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "cohort.jsonl"), raw, 0644); err != nil {
		return err
	}
	return save(filepath.Join(dir, "manifest.json"), map[string]any{"schema": "gooo/native-feedback-family-cohort/v1", "rows": len(rows), "cohort_sha256": hash(raw),
		"source_bodies": len(sources), "intention_groups": len(groups), "ambiguous_finite_views": ambiguous, "families": 5, "configurations": []int{64, 79}, "languages": []string{"en", "ko"},
		"contracts": []string{"complete", "sparse", "contradictory"}, "maximum_candidates_per_view": 2, "planned_matrix_native_calls": 1080, "planned_pilot_native_calls": 10,
		"scope": "Five existing structural families, ten parameterized fallback bodies, twenty intended path behaviors and forty bilingual instructions, each with three finite contracts. Configurations were excluded from model training but prior reserved-probe work has used these families/templates; this is development evidence, not an untouched language benchmark or 120 new independent experiments."})
}
