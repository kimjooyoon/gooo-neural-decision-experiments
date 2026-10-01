package main

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

func ownNativeContextNames() ([]string, error) {
	return ownNativeContextNamesKind(false)
}

func ownNativeContextNamesKind(main bool) ([]string, error) {
	root := "runs/own-model-native-context-feature-20261001/"
	reportSHA := "07574eaf52b3eed57252775e838cdfb139b5b90fcb5eddfb9d49a9fecd20e97a"
	native, runner := "052c250208e38f33a65242e38b042aab80cf85b7", "a60b5b802e0f2251b262c7a31cc7ebe6751a4464"
	if main {
		root, reportSHA = "runs/own-model-native-context-main-20261001/", "629f9d3209ee749e86e344af0623dce8269dd2f3eccb6f745151d6533586ab6b"
		native, runner = "6e0ad85e91c6de0e2c74c55ed7cd4f1e9fa3e6bb", "56940d9c36ff3a926b6da74c8cc91967f01744fb"
	}
	raw, err := read(root + "report.json")
	var report struct {
		Decision    string            `json:"decision"`
		Native      string            `json:"native_revision"`
		Runner      string            `json:"runner_revision"`
		Calls       int               `json:"actual_native_calls"`
		Predictions int               `json:"actual_initial_predictions"`
		Cases       int               `json:"selected_finite_cases"`
		Passed      int               `json:"selected_finite_passed"`
		GoProcesses int               `json:"actual_generated_go_processes"`
		GoValues    int               `json:"actual_generated_function_invocations"`
		Declines    int               `json:"representation_declines"`
		Wrong       int               `json:"sparse_wrong_intended_choices"`
		Files       map[string]string `json:"files_sha256"`
	}
	if err != nil || hash(raw) != reportSHA || json.Unmarshal(raw, &report) != nil || report.Decision != "PASS" ||
		report.Native != native || report.Runner != runner ||
		report.Calls != 32 || report.Predictions != 57 || report.Cases != 120 || report.Passed != 116 ||
		report.GoProcesses != 32 || report.GoValues != 120 || report.Declines != 3 || report.Wrong != 3 || len(report.Files) != 108 {
		return nil, errors.New("frozen own-model native context report differs")
	}
	audit, err := read(root + "audit.json")
	if err != nil || hash(audit) != reportSHA {
		return nil, errors.New("independent native context audit differs")
	}
	names := []string{root + "report.json", root + "audit.json"}
	for name, pin := range report.Files {
		if strings.Contains(name, "..") || strings.HasPrefix(name, "/") ||
			!(strings.HasPrefix(name, root) || strings.HasPrefix(name, "studies/own-model-native-context-v1/") ||
				strings.HasPrefix(name, "studies/own-model-sdk-context-v1/") || name == "docs/own-model-native-context-preregistration.md") {
			return nil, errors.New("native context archive path outside allowlist")
		}
		raw, err := read(name)
		if err != nil || hash(raw) != pin {
			return nil, errors.New("native context source/capture digest differs")
		}
		names = append(names, name)
	}
	expected := 110
	if main {
		pins := map[string]string{
			"publication/own-model-native-context-main-adoption-20261001.json":           "b9dbbca20d48aae1709bff4862ca92e149ada7cd502aea525bb5be513aa7e20e",
			"publication/own-model-native-context-feature-main-comparison-20261001.json": "37100cef4e23f362a57b4614ccddf2f96106377410a0710c6085ac0af6c6b35d",
			"docs/own-model-native-context-main-replay.md":                               "57982cd747899d900a4981d1537ff79f53ca0deb3713d55628afab98b68a4849",
		}
		for name, pin := range pins {
			raw, err := read(name)
			if err != nil || hash(raw) != pin {
				return nil, errors.New("main adoption/replay binding differs")
			}
			names = append(names, name)
		}
		expected = 113
	}
	sort.Strings(names)
	if len(names) != expected {
		return nil, errors.New("native context inventory differs")
	}
	return names, nil
}
