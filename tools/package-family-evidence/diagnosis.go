package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
)

func pathDiagnosisNames() ([]string, error) {
	root := "runs/path-diagnosis-sdk-20261001"
	raw, err := read(root + "/report.json")
	if err != nil {
		return nil, err
	}
	var report struct {
		Schema               string            `json:"schema"`
		Decision             string            `json:"decision"`
		Runner               string            `json:"runner_revision"`
		Searches             int               `json:"sdk_searches"`
		Diagnoses            int               `json:"diagnoses"`
		Candidates           int               `json:"candidate_observations"`
		Predictions          int               `json:"actual_initial_model_predictions"`
		DiagnosisPredictions int               `json:"diagnosis_model_predictions"`
		Executions           int               `json:"actual_generated_go_execution_jobs"`
		Invocations          int               `json:"actual_generated_function_invocations"`
		Files                map[string]string `json:"files_sha256"`
	}
	if hash(raw) != "00859f2e75650f6983d03baf5a43445f39dbff793daee025235540ca9c4aedbe" || json.Unmarshal(raw, &report) != nil ||
		report.Schema != "gooo/path-diagnosis-study/v1" || report.Decision != "PASS" || report.Runner != "a2b7c0c6960888d93f408f0f45af698c579d68cf" ||
		report.Searches != 144 || report.Diagnoses != 144 || report.Candidates != 576 || report.Predictions != 144 || report.DiagnosisPredictions != 0 ||
		report.Executions != 12 || report.Invocations != 372 || len(report.Files) != 14 {
		return nil, errors.New("fixed path diagnosis inventory differs")
	}
	audit, err := read(root + "/audit.json")
	if err != nil || !bytes.Equal(raw, audit) {
		return nil, errors.New("path diagnosis audit differs")
	}
	names := []string{root + "/report.json", root + "/audit.json", "studies/compound-path-v1/cohort.jsonl", "studies/compound-path-v1/manifest.json", "docs/path-diagnosis-preregistration.md"}
	pathRE := regexp.MustCompile(`^(preexecution\.json|observations\.jsonl|executions/[a-f0-9]{64}\.json)$`)
	shaRE := regexp.MustCompile(`^[a-f0-9]{64}$`)
	for path, pin := range report.Files {
		if !pathRE.MatchString(path) || !shaRE.MatchString(pin) {
			return nil, errors.New("unsafe diagnosis path/hash")
		}
		full := root + "/" + path
		data, err := read(full)
		if err != nil || hash(data) != pin {
			return nil, errors.New("path diagnosis bytes differ")
		}
		names = append(names, full)
	}
	sort.Strings(names)
	if len(names) != 19 {
		return nil, errors.New("path diagnosis archive count differs")
	}
	return names, nil
}
