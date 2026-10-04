package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type PairedNativeSummary struct {
	NativeSummary
	Goal          int `json:"intent_goal"`
	Style         int `json:"wording_variant"`
	ActiveTotal   int `json:"active_fields_total"`
	ActivePassed  int `json:"active_fields_passed"`
	GuardTotal    int `json:"guard_fields_total"`
	GuardPassed   int `json:"guard_fields_passed"`
	ChangedTotal  int `json:"required_changed_fields_total"`
	ChangedPassed int `json:"required_changed_fields_passed"`
}

func pairedNativePlan() []PairedSpec {
	var plan []PairedSpec
	for _, view := range []struct {
		f, p int
		m    uint16
	}{{6, 0, 3}, {7, 4, 5}} {
		for _, goal := range []int{0, 1, 6, 7} {
			for _, lang := range []string{"ko", "en"} {
				plan = append(plan, PairedSpec{view.f, view.p, goal, 0, view.m, lang, "test_source"})
			}
		}
	}
	for _, view := range []struct {
		f, p int
		m    uint16
	}{{6, 0, 0}, {7, 4, 7}} {
		for _, pair := range [][2]int{{1, 1}, {6, 2}} {
			for _, lang := range []string{"ko", "en"} {
				plan = append(plan, PairedSpec{view.f, view.p, pair[0], pair[1], view.m, lang, "test_wording_new"})
			}
		}
	}
	return plan
}

func pairedNativeCases(family, goal int) []NativeCase {
	fields := names[family]
	var cases []NativeCase
	for i, c := range []struct {
		title, state, reason string
		active               bool
	}{{"이번 제목", "queued", "이번 사유", true}, {"Current runtime", "queued", "new paired detail", true}, {"keep value", "queued", "later message", false}, {"completed", "ready", "already done", true}} {
		input := map[string]string{fields[0]: c.title, fields[1]: c.state, fields[2]: c.reason}
		expected := map[string]string{fields[0]: c.title, fields[1]: c.state, fields[2]: c.reason}
		if c.active && c.state != "ready" {
			expected = goalValues(family, goal, c.title, c.state, c.reason)
		}
		cases = append(cases, NativeCase{fmt.Sprintf("paired-fresh-%d", i), map[string]any{"Select.input0": input, "Select.input1": c.active}, map[string]any{"Select": expected, "Label": expected[fields[0]] + ":" + expected[fields[1]] + ":" + expected[fields[2]]}})
	}
	return cases
}

func summarizePairedNative(raw []byte, cases []NativeCase, family int) PairedNativeSummary {
	s := PairedNativeSummary{NativeSummary: summarizeNative(raw, cases)}
	var capture struct {
		Runtime struct {
			Traces []struct {
				Deliveries []struct {
					Actual json.RawMessage `json:"actual"`
				} `json:"deliveries"`
			} `json:"traces"`
		} `json:"runtime"`
	}
	check(json.Unmarshal(raw, &capture))
	for i, c := range cases {
		input := c.Inputs["Select.input0"].(map[string]string)
		active := c.Inputs["Select.input1"].(bool) && input[names[family][1]] != "ready"
		var actual map[string]string
		check(json.Unmarshal(capture.Runtime.Traces[i].Deliveries[0].Actual, &actual))
		for field, want := range c.Expected["Select"].(map[string]string) {
			matches := actual[field] == want
			if active {
				s.ActiveTotal++
				if matches {
					s.ActivePassed++
				}
			} else {
				s.GuardTotal++
				if matches {
					s.GuardPassed++
				}
			}
			if active && input[field] != want {
				s.ChangedTotal++
				if matches {
					s.ChangedPassed++
				}
			}
		}
	}
	if s.ActiveTotal != 6 || s.GuardTotal != 6 || s.ActivePassed+s.GuardPassed != s.FieldsPassed {
		panic("paired active/guard recount differs")
	}
	return s
}

func nativePaired(prepared, models, compiler, revision, frozen, output string) {
	if prepared == "" || models == "" || compiler == "" || len(revision) != 40 || frozen == "" || output == "" {
		panic("explicit paired native inputs required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh paired native output required")
	}
	check(os.MkdirAll(output, 0755))
	version := invoke(compiler, "version", "--build", "--json")
	var identity struct {
		Source string `json:"compiler_source_sha"`
		State  string `json:"source_status"`
		Go     string `json:"go_version"`
	}
	check(json.Unmarshal(version, &identity))
	if identity.Source != revision || identity.State != "CLEAN_VCS" || identity.Go != "go1.27.1" {
		panic("paired native compiler identity differs")
	}
	check(os.WriteFile(filepath.Join(output, "compiler-build.json"), version, 0644))
	byID := map[string]PairedRow{}
	for _, r := range readPaired(prepared) {
		byID[r.ID] = r
	}
	file, err := os.OpenFile(filepath.Join(output, "summaries.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer file.Close()
	enc := json.NewEncoder(file)
	observed := 0
	for _, spec := range pairedNativePlan() {
		r, ok := byID[spec.ID()]
		if !ok || r.Split != spec.Split {
			panic("registered paired native view missing")
		}
		base, err := os.ReadFile(filepath.Join(prepared, r.SourceFile))
		check(err)
		for _, profile := range []string{"deterministic", "frozen_field_v1", "fp32", "ptq_ternary", "qat_ternary"} {
			for _, budget := range []int{1, 2, 8} {
				stem := fmt.Sprintf("%s-%s-b%d", r.ID, profile, budget)
				source := []byte(strings.Replace(string(base), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1))
				sourceFile := filepath.Join(output, stem+".gooo.fixture")
				check(os.WriteFile(sourceFile, source, 0644))
				cases := pairedNativeCases(r.Family, r.Goal)
				caseFile := filepath.Join(output, stem+"-cases.json")
				save(caseFile, map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases})
				args := []string{"body-compose", "--source", sourceFile, "--cases", caseFile}
				if profile == "frozen_field_v1" {
					args = append(args, "--model", frozen)
				} else if profile != "deterministic" {
					args = append(args, "--model", filepath.Join(models, profile, "model.json"))
				}
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				command := exec.CommandContext(ctx, compiler, args...)
				var stdout, stderr bytes.Buffer
				command.Stdout = &stdout
				command.Stderr = &stderr
				started := time.Now()
				err = command.Run()
				elapsed := time.Since(started)
				cancel()
				if err != nil {
					check(os.WriteFile(filepath.Join(output, stem+"-failed.json"), stdout.Bytes(), 0600))
					panic(fmt.Sprintf("paired construction: %v: %.1000s", err, stderr.Bytes()))
				}
				raw := stdout.Bytes()
				check(os.WriteFile(filepath.Join(output, stem+"-raw.json"), raw, 0600))
				s := summarizePairedNative(raw, cases, r.Family)
				s.ID, s.Profile, s.Budget, s.Compiler, s.Goal, s.Style = r.ID, profile, budget, revision, r.Goal, r.Style
				s.OriginalSHA, s.RawSHA = hash(source), hash(raw)
				s.WallMS = float64(elapsed.Nanoseconds()) / 1e6
				s.CPUSeconds = command.ProcessState.UserTime().Seconds() + command.ProcessState.SystemTime().Seconds()
				if (profile == "deterministic" && s.Calls != 0) || (profile != "deterministic" && s.Calls != 1) {
					panic("paired prediction count differs")
				}
				if budget == 8 && (s.RuntimePassed != 8 || s.FieldsPassed != 12 || s.SelectionFieldsPassed != 15) {
					panic("full paired finite budget incomplete")
				}
				check(enc.Encode(s))
				observed++
				if observed%30 == 0 {
					fmt.Printf("Compiled and executed %d paired graph constructions\n", observed)
				}
			}
		}
	}
	if observed != 360 {
		panic("paired native inventory incomplete")
	}
	save(filepath.Join(output, "manifest.json"), map[string]any{"schema": "gooo/record-paired-native/v2", "constructions": observed, "builds": observed, "runs": observed * 2, "model_predictions_during_native_execution": 0, "source_views": len(pairedNativePlan()), "compiler_source": revision, "scope": "Sequential matched controls, two active and two unchanged guard inputs per graph. Raw traces bind inputs/producer delivery and output values; active, guard and required-changed fields counted separately. No concurrency speedup claim."})
}
