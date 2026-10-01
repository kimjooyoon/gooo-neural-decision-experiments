package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestOwnNativeContextMatrixAndAttemptArithmetic(t *testing.T) {
	t.Chdir("../..")
	rows, err := ownContextRows()
	if err != nil || len(rows) != 8 {
		t.Fatal("fixed documents missing", err)
	}
	finite, predictions := 0, 0
	for _, row := range rows {
		finite += 4 * len(row.Document.Cases)
		if !row.Overflow {
			predictions += 3 * len(row.Document.Plan.Decisions)
		}
		prepared, err := pathplan.Prepare(row.Document.Plan)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		search, _, err := prepared.Search(ctx, nil, row.Document.Cases, row.Document.Max, "")
		cancel()
		if err != nil || auditOwnContextAttempts(prepared, row.Document, search) != nil {
			t.Fatal("valid typed attempts rejected", err)
		}
		for i := range search.Attempts {
			if len(search.Attempts[i].Results) == 0 {
				continue
			}
			search.Attempts[i].Results[0].Actual++
			if auditOwnContextAttempts(prepared, row.Document, search) == nil {
				t.Fatal("forged actual accepted")
			}
			break
		}
	}
	if finite != 120 || predictions != 57 {
		t.Fatal("preregistered denominator changed")
	}
}

func TestOwnNativeContextComparisonRetainsActualsAndIgnoresTiming(t *testing.T) {
	t.Chdir("../..")
	root := "runs/own-model-native-context-feature-20261001"
	rows, err := ownContextRows()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, row := range rows {
		for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary", "offline"} {
			for _, suffix := range []string{".json", "-execution.json"} {
				name := row.ID + "-" + arm + suffix
				raw, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	name := filepath.Join(dir, "en-sparse-fp32.json")
	raw, _ := os.ReadFile(name)
	var value nativeResult
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	value.Report.Paths.Search.Selection.Receipts[0].PredictNS++
	raw, _ = json.Marshal(value)
	if err = os.WriteFile(name, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = compareOwnNativeContext(root, dir); err != nil {
		t.Fatal("timing altered semantic comparison", err)
	}
	value.Report.Paths.Cases[0].Actual++
	raw, _ = json.Marshal(value)
	if err = os.WriteFile(name, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = compareOwnNativeContext(root, dir); err == nil {
		t.Fatal("changed actual accepted")
	}
}
