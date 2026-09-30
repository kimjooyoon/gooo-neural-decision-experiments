package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// replaySaved preserves the failed original and performs only Go verification.
func replaySaved(capture, fixtures, output, goBinary string) error {
	raw, err := os.ReadFile(filepath.Join(capture, "report.json"))
	if err != nil {
		return err
	}
	var original struct {
		Revision string `json:"compiler_revision"`
		Cells    []cell `json:"cells"`
	}
	if err := json.Unmarshal(raw, &original); err != nil {
		return err
	}
	if len(original.Cells) != 6 {
		return errors.New("expected six original captures")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("replay output must be fresh")
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	var cells []cell
	for _, item := range original.Cells {
		fixture, _, _ := strings.Cut(item.ID, "-")
		activity := "ArithmeticHole"
		if fixture == "boolean" {
			activity = "BooleanHole"
		} else if fixture != "arithmetic" {
			return errors.New("unknown saved fixture")
		}
		payloadRaw, err := os.ReadFile(filepath.Join(capture, item.ID+".stdout.json"))
		if err != nil {
			return err
		}
		if hash(payloadRaw) != item.RawSHA {
			return errors.New("saved raw receipt hash mismatch")
		}
		var decoded payload
		if err := json.Unmarshal(payloadRaw, &decoded); err != nil {
			return err
		}
		if decoded.Report.Compiler != original.Revision || hash([]byte(decoded.Source)) != item.GeneratedSHA {
			return errors.New("saved compiler/source binding mismatch")
		}
		planRaw, err := os.ReadFile(filepath.Join(fixtures, fixture+".plan.json"))
		if err != nil {
			return err
		}
		if hash(planRaw) != item.PlanSHA {
			return errors.New("saved plan binding mismatch")
		}
		var expected plan
		if err := json.Unmarshal(planRaw, &expected); err != nil {
			return err
		}
		if len(expected.Cases) != item.Total {
			return errors.New("saved case denominator mismatch")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		verification, err := executeGenerated(ctx, goBinary, decoded.Source, activity, expected.Cases)
		cancel()
		item.VerificationSHA = hash(verification)
		item.Status = "FAIL"
		item.Error = "independent replay failed"
		if err == nil {
			item.Status = "PASS"
			item.Error = ""
			item.IndependentPassed = len(expected.Cases)
		}
		if err := os.WriteFile(filepath.Join(output, item.ID+".verification.txt"), verification, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(output, item.ID+".prov.ttl"), trace(item, original.Revision), 0o644); err != nil {
			return err
		}
		cells = append(cells, item)
	}
	status := "PASS"
	for _, item := range cells {
		if item.Status != "PASS" {
			status = "FAIL"
		}
	}
	if err := save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/compiler-dogfood-adjudication/v1", "status": status, "original_report_sha256": hash(raw),
		"native_calls_made_during_replay": 0, "model_calls_made_during_replay": 0, "cells": cells,
		"scope":            "Derived Go-only execution; inherited native/model fields describe original capture, not new calls",
		"original_failure": "trimpath binary could not infer a physical Go executable from runtime.GOROOT; explicit --go fixes the harness"}); err != nil {
		return err
	}
	fmt.Printf("%s: six Go-only replays; zero new native or model calls\n", status)
	if status != "PASS" {
		return errors.New("Go-only replay failed")
	}
	return nil
}
