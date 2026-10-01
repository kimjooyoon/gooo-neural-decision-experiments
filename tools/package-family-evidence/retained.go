package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
)

func retainedNativeNames() ([]string, error) {
	root := "runs/retained-native-feature-pilot-20261001"
	raw, err := read(root + "/report.json")
	if err != nil {
		return nil, err
	}
	var report struct {
		Schema      string            `json:"schema"`
		Decision    string            `json:"decision"`
		Files       map[string]string `json:"evidence_sha256"`
		Calls       int               `json:"valid_constructions"`
		Rejects     int               `json:"source_rejections"`
		Processes   int               `json:"native_processes"`
		Predictions int               `json:"actual_model_predictions"`
	}
	if json.Unmarshal(raw, &report) != nil || report.Schema != "gooo/retained-native-audit/v1" || report.Decision != "PASS" ||
		report.Calls != 180 || report.Rejects != 10 || report.Processes != 70 || report.Predictions != 720 || len(report.Files) != 144 {
		return nil, errors.New("fixed retained inventory differs")
	}
	audit, err := read(root + "/audit.json")
	if err != nil || !bytes.Equal(raw, audit) {
		return nil, errors.New("retained audit differs")
	}
	names := []string{root + "/report.json", root + "/audit.json"}
	pathRE := regexp.MustCompile(`^(preexecution\.json|(captures|metrics)/[a-z0-9_-]+\.jsonl?|executions/[a-f0-9]{64}\.json)$`)
	shaRE := regexp.MustCompile(`^[a-f0-9]{64}$`)
	for path, pin := range report.Files {
		if !pathRE.MatchString(path) || !shaRE.MatchString(pin) {
			return nil, errors.New("unsafe retained path/hash")
		}
		raw, err := read(root + "/" + path)
		if err != nil || hash(raw) != pin {
			return nil, errors.New("retained evidence changed")
		}
		names = append(names, root+"/"+path)
	}
	sort.Strings(names)
	return names, nil
}
