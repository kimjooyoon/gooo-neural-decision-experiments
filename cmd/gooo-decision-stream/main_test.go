package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestRunRejectsOutOfRangeWorkersBeforeLoadingModel(t *testing.T) {
	var output, diagnostics bytes.Buffer
	code := run(context.Background(), []string{"--model", "missing-model.json", "--workers", "9"}, io.NopCloser(strings.NewReader("")), testWriteCloser{Writer: &output}, &diagnostics)
	if code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "workers must be between 1 and 8") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, output.String(), diagnostics.String())
	}
}

func TestRunRequiresModelPath(t *testing.T) {
	var output, diagnostics bytes.Buffer
	code := run(context.Background(), nil, io.NopCloser(strings.NewReader("")), testWriteCloser{Writer: &output}, &diagnostics)
	if code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "usage: gooo-decision-stream") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, output.String(), diagnostics.String())
	}
}

type testWriteCloser struct{ io.Writer }

func (testWriteCloser) Close() error { return nil }

func TestDefaultWorkersStaysWithinConfiguredBound(t *testing.T) {
	workers := defaultWorkers()
	if workers < 1 || workers > 8 {
		t.Fatalf("default worker count = %d, outside [1,8]", workers)
	}
}
