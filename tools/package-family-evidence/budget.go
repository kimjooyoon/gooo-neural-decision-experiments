package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
)

func nativeBudgetNames() ([]string, error) {
	root := "runs/native-budget-main-20261001"
	raw, err := read(root + "/report.json")
	if err != nil {
		return nil, err
	}
	var report struct {
		Schema      string            `json:"schema"`
		Decision    string            `json:"decision"`
		Runner      string            `json:"runner_revision"`
		Native      string            `json:"native_revision"`
		Calls       int               `json:"valid_native_constructions"`
		Rejects     int               `json:"source_rejections"`
		Processes   int               `json:"native_processes"`
		Predictions int               `json:"actual_model_predictions"`
		Feedback    int               `json:"feedback_predictions"`
		Candidates  int               `json:"committed_candidates"`
		Executions  int               `json:"actual_generated_go_processes"`
		Invocations int               `json:"actual_generated_function_invocations"`
		Files       map[string]string `json:"files_sha256"`
	}
	if json.Unmarshal(raw, &report) != nil || report.Schema != "gooo/native-budget-study/v1" || report.Decision != "PASS" ||
		report.Runner != "ab8e19b26e2298e713888a64e59602dc5c383529" || report.Native != "37fb287a9c888bd0262194f524dde16426eca3a9" ||
		report.Calls != 1944 || report.Rejects != 162 || report.Processes != 162 || report.Predictions != 4041 || report.Feedback != 585 ||
		report.Candidates != 3005 || report.Executions != 12 || report.Invocations != 192 || len(report.Files) != 337 {
		return nil, errors.New("fixed native budget inventory differs")
	}
	audit, err := read(root + "/audit.json")
	if err != nil || !bytes.Equal(raw, audit) {
		return nil, errors.New("budget audit differs")
	}
	names := []string{root + "/report.json", root + "/audit.json", "studies/compound-path-v1/cohort.jsonl", "studies/compound-path-v1/manifest.json", "docs/native-budget-preregistration.md"}
	pathRE := regexp.MustCompile(`^(preexecution\.json|captures/[a-z0-9_-]+\.jsonl|metrics/[a-z0-9_-]+\.json|executions/[a-f0-9]{64}\.json)$`)
	shaRE := regexp.MustCompile(`^[a-f0-9]{64}$`)
	for path, pin := range report.Files {
		if !pathRE.MatchString(path) || !shaRE.MatchString(pin) {
			return nil, errors.New("unsafe budget evidence path/hash")
		}
		full := root + "/" + path
		data, err := read(full)
		if err != nil || hash(data) != pin {
			return nil, errors.New("budget evidence bytes differ")
		}
		names = append(names, full)
	}
	sort.Strings(names)
	if len(names) != 342 {
		return nil, errors.New("budget archive count differs")
	}
	return names, nil
}
