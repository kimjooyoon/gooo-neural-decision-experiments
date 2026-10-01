package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
)

func nativePathDiagnosisNames() ([]string, error) {
	root := "runs/native-path-diagnosis-feature-20261001"
	raw, err := read(root + "/report.json")
	var report struct {
		Schema      string            `json:"schema"`
		Decision    string            `json:"decision"`
		Runner      string            `json:"runner_revision"`
		Native      string            `json:"native_revision"`
		Calls       int               `json:"actual_native_calls"`
		Predictions int               `json:"actual_initial_model_predictions"`
		Diagnoses   int               `json:"diagnoses"`
		Candidates  int               `json:"candidate_observations"`
		Pairs       int               `json:"same_source_pairs"`
		Jobs        int               `json:"actual_generated_go_processes"`
		Values      int               `json:"actual_generated_function_invocations"`
		Files       map[string]string `json:"files_sha256"`
	}
	if err != nil || hash(raw) != "3c144810d626ae752230001654fc1b75960377d023988f2f6e24c0067b154edb" || json.Unmarshal(raw, &report) != nil ||
		report.Schema != "gooo/native-path-diagnosis-smoke/v1" || report.Decision != "PASS" || report.Runner != "18ae830c18b47b377ae8fa97dcc39df75f6b593b" ||
		report.Native != "5ae486d05b3521ecc48b18bfa28d45cdd9041203" || report.Calls != 8 || report.Predictions != 4 || report.Diagnoses != 4 ||
		report.Candidates != 8 || report.Pairs != 4 || report.Jobs != 2 || report.Values != 4 || len(report.Files) != 24 {
		return nil, errors.New("fixed native diagnosis report differs")
	}
	audit, err := read(root + "/audit.json")
	if err != nil || !bytes.Equal(raw, audit) {
		return nil, errors.New("native diagnosis audit differs")
	}
	names := []string{root + "/report.json", root + "/audit.json", "docs/native-path-diagnosis-preregistration.md",
		"studies/native-path-diagnosis-v1/source.gooo.fixture", "studies/native-path-diagnosis-v1/plan.json", "studies/native-path-diagnosis-v1/diagnosis.json"}
	pathRE := regexp.MustCompile(`^(preexecution\.json|source\.gooo\.fixture|diagnosis\.json|plan(-ko|-en)?\.json|(ko|en)-(offline|fp32)-(off|on)(-metrics)?\.json|executions/[a-f0-9]{64}\.json)$`)
	shaRE := regexp.MustCompile(`^[a-f0-9]{64}$`)
	for name, pin := range report.Files {
		if !pathRE.MatchString(name) || !shaRE.MatchString(pin) {
			return nil, errors.New("unsafe native diagnosis path/hash")
		}
		full := root + "/" + name
		data, e := read(full)
		if e != nil || hash(data) != pin {
			return nil, errors.New("native diagnosis bytes differ")
		}
		names = append(names, full)
	}
	sort.Strings(names)
	if len(names) != 30 {
		return nil, errors.New("native diagnosis archive inventory differs")
	}
	return names, nil
}
