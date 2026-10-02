package main

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

type firstOutput struct {
	Mask     uint16
	Actual   [16]int64
	Expected [16]int64
	Complete bool
}
type bilingualCounts struct {
	Pairs                   int `json:"bilingual_function_pairs"`
	MaskDiff                int `json:"initial_mask_disagreement_pairs"`
	OutputDiff              int `json:"initial_ordered_output_disagreement_pairs"`
	DifferentCases          int `json:"initial_different_ordered_case_values"`
	Cases                   int `json:"ordered_case_pairs"`
	SameOutputDifferentMask int `json:"different_masks_same_ordered_outputs_pairs"`
	BothComplete            int `json:"both_initial_functions_complete_pairs"`
	OneComplete             int `json:"exactly_one_initial_function_complete_pairs"`
	NeitherComplete         int `json:"neither_initial_function_complete_pairs"`
}

func first(v threecohort.View, c threefeedback.Capture) (firstOutput, error) {
	if c.ViewID != v.ID || len(c.Search.Attempts) == 0 || len(c.Search.Attempts[0].Results) != 16 || len(v.Cases) != 16 {
		return firstOutput{}, errors.New("exact actual initial ordered outputs required")
	}
	o := firstOutput{Mask: c.Search.Attempts[0].Mask, Complete: c.Search.Attempts[0].Passed == 16}
	for i, r := range c.Search.Attempts[0].Results {
		o.Actual[i], o.Expected[i] = r.Actual, v.Cases[i].Expected
	}
	return o, nil
}
func (c *bilingualCounts) add(en, ko firstOutput) error {
	if en.Expected != ko.Expected {
		return errors.New("bilingual pair has different authored ordered contracts")
	}
	c.Pairs++
	c.Cases += 16
	if en.Mask != ko.Mask {
		c.MaskDiff++
	}
	different := 0
	for i := range en.Actual {
		if en.Actual[i] != ko.Actual[i] {
			different++
		}
	}
	c.DifferentCases += different
	if different > 0 {
		c.OutputDiff++
	} else if en.Mask != ko.Mask {
		c.SameOutputDifferentMask++
	}
	if en.Complete && ko.Complete {
		c.BothComplete++
	} else if en.Complete || ko.Complete {
		c.OneComplete++
	} else {
		c.NeitherComplete++
	}
	return nil
}
func bilingualAudit(dataset, root, prefix, tail, destination string) error {
	temp, err := os.MkdirTemp("", "gooo-three-bilingual-audit-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	if err = auditContinuation(dataset, root, prefix, tail, filepath.Join(temp, "combined.json")); err != nil {
		return err
	}
	var audit struct {
		Status string `json:"status"`
	}
	if err = read(filepath.Join(temp, "combined.json"), &audit); err != nil {
		return err
	}
	if audit.Status != "PASS_WITH_SEPARATE_STORAGE_AMENDMENT" {
		return errors.New("complete combined observations required for diagnostic")
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	_, ids, err := loadModels(root)
	if err != nil {
		return err
	}
	all := map[string]map[string]bilingualCounts{}
	for _, split := range []string{"calibration", "development"} {
		all[split] = map[string]bilingualCounts{}
		cell := splitViews(views, split)
		for _, id := range ids {
			pairs := map[string]map[string]firstOutput{}
			n := 0
			for _, dir := range []string{prefix, tail} {
				f, e := os.Open(filepath.Join(dir, filename(split, id)))
				if os.IsNotExist(e) {
					continue
				}
				if e != nil {
					return e
				}
				scan := bufio.NewScanner(f)
				scan.Buffer(make([]byte, 32768), lineCap)
				for scan.Scan() {
					if n >= len(cell) {
						f.Close()
						return errors.New("extra bilingual diagnostic row")
					}
					var o observation
					if e = threecohort.Decode(scan.Bytes(), &o); e != nil {
						f.Close()
						return e
					}
					v := cell[n]
					if o.Policy != id {
						f.Close()
						return errors.New("bilingual diagnostic policy differs")
					}
					x, e := first(v, o.Capture)
					if e != nil {
						f.Close()
						return e
					}
					if pairs[v.Group] == nil {
						pairs[v.Group] = map[string]firstOutput{}
					}
					if _, duplicate := pairs[v.Group][v.Language]; duplicate {
						f.Close()
						return errors.New("duplicate bilingual diagnostic language")
					}
					pairs[v.Group][v.Language] = x
					n++
				}
				e = scan.Err()
				f.Close()
				if e != nil {
					return e
				}
			}
			if n != 512 || len(pairs) != 256 {
				return errors.New("complete frozen bilingual diagnostic denominator required")
			}
			c := bilingualCounts{}
			for _, pair := range pairs {
				en, enOK := pair["en"]
				ko, koOK := pair["ko"]
				if len(pair) != 2 || !enOK || !koOK {
					return errors.New("exact Korean/English diagnostic pair required")
				}
				if err = c.add(en, ko); err != nil {
					return err
				}
			}
			all[split][id] = c
		}
	}
	return (&storage{}).save(destination, map[string]any{"schema": "gooo/own-three-sdk-bilingual-functional-diagnostic/v1", "status": "PASS", "dataset_sha256": threecohort.DatasetSHA, "cells": all, "actual_session_observations": 11264, "bilingual_pairs": 5632, "ordered_case_pairs": 90112, "new_model_predictions": 0, "new_optimizer_updates": 0, "selector_changed": false, "post_hoc_diagnostic": true, "scope": "Initial 16 ordered outputs on identical bilingual authored contracts. Different masks can have identical finite outputs; equal outputs are finite-case equivalence, not universal program equivalence. This diagnostic uses verified original/tail observations and does not select a model or establish paraphrase/general-language accuracy."})
}
