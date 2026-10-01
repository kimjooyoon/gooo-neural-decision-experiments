package main

import (
	"context"
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
