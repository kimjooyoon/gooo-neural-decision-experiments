package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decisionstream"
)

func TestLoadFrozenRowsMatchesManifestAndTestDenominators(t *testing.T) {
	root := findRepoRoot(t)
	manifest, manifestSHA, rows, labels, languages, err := loadFrozenRows(
		filepath.Join(root, "data", "synthetic-ops-v1", "manifest.json"),
		filepath.Join(root, "data", "synthetic-ops-v1", "dataset.jsonl"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifestSHA) != 64 || len(rows) != 256 || manifest.DatasetSHA == "" {
		t.Fatalf("unexpected frozen data pin or size: sha=%q rows=%d", manifestSHA, len(rows))
	}
	for _, label := range operationLabels {
		if labels[label] != 32 {
			t.Fatalf("test rows for %s = %d, want 32", label, labels[label])
		}
	}
	if languages["en"] != 128 || languages["ko"] != 128 {
		t.Fatalf("language counts = %#v", languages)
	}
}

func TestMakeRequestUsesPromptOperandsAndExcludesGold(t *testing.T) {
	row := datasetRow{
		ID: "example-test", TemplateID: "template-test", ConfigurationID: "cfg-01",
		Language: "ko", Split: "test", Text: "불리언 입력은 flag_a, flag_b이다. 두 플래그가 참이면 참으로 표시하라.", Label: "and",
	}
	request, err := makeRequest(row, 2)
	if err != nil {
		t.Fatal(err)
	}
	if request.Request.Left.Name != "flag_a" || request.Request.Right.Name != "flag_b" || request.Request.Left.Type != decision.TypeBool || request.Request.Right.Type != decision.TypeBool {
		t.Fatalf("request operands were not derived from prompt: %+v", request.Request)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(`"label"`)) || bytes.Contains(encoded, []byte(`"split"`)) || bytes.Contains(encoded, []byte(`"template_id"`)) {
		t.Fatalf("evaluation target or dataset metadata leaked into selection request: %s", encoded)
	}
	if request.CorrelationID != "example-test-r3" {
		t.Fatalf("correlation ID = %q", request.CorrelationID)
	}
}

func TestResultParserScoresOnlyUniqueCorrelatedRows(t *testing.T) {
	rows := []datasetRow{
		{ID: "example-add", Text: "Operands: lhs, rhs. Compute the sum.", Label: "add"},
		{ID: "example-and", Text: "Boolean inputs: left_flag, right_flag. Combine flags.", Label: "and"},
	}
	parser := newResultParser(rows, 1, "fp32", "weights-pin")
	for _, sequence := range []int{2, 1, 1} {
		row := rows[sequence-1]
		parsed, ok := parseOperands(row.Text)
		if !ok {
			t.Fatalf("test fixture prompt did not parse: %s", row.Text)
		}
		ir, err := decision.BuildTypedBinary(row.Label, decision.Identifier{Name: parsed.left, Type: parsed.kind}, decision.Identifier{Name: parsed.right, Type: parsed.kind})
		if err != nil {
			t.Fatalf("BuildTypedBinary(%q): %v", row.Label, err)
		}
		response := decision.DecisionResponse{
			Schema: decision.DecisionResponseSchema, Status: "decision", ModelVariant: "fp32", WeightsSHA256: "weights-pin",
			BestLabel:     row.Label,
			TypedBinaryIR: &ir,
		}
		result := decisionstream.Result{
			Schema: decisionstream.ResultSchema, Sequence: uint64(sequence),
			CorrelationID: row.ID + "-r1", Status: "completed", Response: &response,
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = parser.Write(append(encoded, '\n'))
	}
	parser.finish()
	if parser.summary.observed != 3 || parser.summary.uniqueSequences != 2 || parser.summary.duplicates != 1 || parser.summary.missing != 0 || parser.summary.correlated != 2 {
		t.Fatalf("sequence accounting = %+v", parser.summary)
	}
	if parser.summary.bestLabelObserved != 2 || parser.summary.bestLabelCorrect != 2 || parser.summary.acceptedCorrect != 2 || parser.summary.errorRecords != 1 {
		t.Fatalf("score accounting = %+v", parser.summary)
	}
}

func TestResultParserBoundsMalformedOutputAndCountsMissing(t *testing.T) {
	parser := newResultParser([]datasetRow{{ID: "example", Label: "add", Text: "Operands: lhs, rhs."}}, 1, "fp32", "pin")
	fragment := bytes.Repeat([]byte{'x'}, maxOutputLineSize+1024)
	if written, err := parser.Write(fragment); err != nil || written != len(fragment) {
		t.Fatalf("Write() = (%d, %v), want all bytes consumed", written, err)
	}
	if len(parser.line) != 0 || !parser.discardLine {
		t.Fatalf("oversized output line was retained: bytes=%d discard=%t", len(parser.line), parser.discardLine)
	}
	parser.finish()
	if parser.summary.unterminated != 1 || parser.summary.errorRecords != 1 || parser.summary.missing != 1 {
		t.Fatalf("bounded failure accounting = %+v", parser.summary)
	}
}

func TestPeakRSSDarwinAndLinuxUnits(t *testing.T) {
	if value, ok := peakRSSBytes(4096, "darwin"); !ok || value != 4096 {
		t.Fatalf("Darwin peak RSS = (%d,%t), want byte-preserving", value, ok)
	}
	if value, ok := peakRSSBytes(4, "linux"); !ok || value != 4096 {
		t.Fatalf("Linux peak RSS = (%d,%t), want KiB converted to bytes", value, ok)
	}
	if _, ok := peakRSSBytes(4, "freebsd"); ok {
		t.Fatal("unsupported platform unexpectedly reported peak RSS")
	}
}

func TestWriteReportRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeNewReport(path, benchmarkReport{Schema: benchmarkSchema}); err == nil {
		t.Fatal("writeNewReport overwrote an existing report")
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "kept" {
		t.Fatalf("existing report changed: %q, %v", contents, err)
	}
}

func TestRunRejectsMissingOutputParentBeforeStartingChild(t *testing.T) {
	root := findRepoRoot(t)
	temporary := t.TempDir()
	streamPath := filepath.Join(temporary, "fake-stream")
	markerPath := filepath.Join(temporary, "child-started")
	if err := os.WriteFile(streamPath, []byte("#!/bin/sh\n/usr/bin/touch \"$BENCHMARK_STREAM_TEST_MARKER\"\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(streamPath, 0o700); err != nil {
		t.Fatal(err)
	}
	streamSHA, err := hashRegularFile(streamPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("BENCHMARK_STREAM_TEST_MARKER", markerPath)
	outputPath := filepath.Join(temporary, "missing-parent", "report.json")
	var diagnostics bytes.Buffer
	code := run([]string{
		"--stream-bin", streamPath,
		"--binary-sha256", streamSHA,
		"--output", outputPath,
		"--repo-root", root,
	}, &diagnostics)
	if code != 2 || !strings.Contains(diagnostics.String(), "existing non-symlink directory") {
		t.Fatalf("run() = %d, diagnostics=%q", code, diagnostics.String())
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("child process ran before rejecting output parent: stat err=%v", err)
	}
}

func TestValidateNewOutputPathRejectsSymlinkParent(t *testing.T) {
	temporary := t.TempDir()
	realDirectory := filepath.Join(temporary, "real")
	if err := os.Mkdir(realDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	symlinkDirectory := filepath.Join(temporary, "linked")
	if err := os.Symlink(realDirectory, symlinkDirectory); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := validateNewOutputPath(filepath.Join(symlinkDirectory, "report.json")); err == nil {
		t.Fatal("symlink output parent was accepted")
	}
}

func TestDecodeSHA256RequiresLowercaseFullDigest(t *testing.T) {
	if _, err := decodeSHA256(strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeSHA256(strings.Repeat("A", 64)); err == nil {
		t.Fatal("uppercase digest was accepted")
	}
	if _, err := decodeSHA256("abc"); err == nil {
		t.Fatal("short digest was accepted")
	}
}

func TestReplaceEnvSetsOneValue(t *testing.T) {
	got := replaceEnv([]string{"A=1", "GOMAXPROCS=old", "B=2", "GOMAXPROCS=older"}, "GOMAXPROCS", "4")
	count := 0
	for _, entry := range got {
		if strings.HasPrefix(entry, "GOMAXPROCS=") {
			count++
			if entry != "GOMAXPROCS=4" {
				t.Fatalf("unexpected GOMAXPROCS entry: %q", entry)
			}
		}
	}
	if count != 1 {
		t.Fatalf("GOMAXPROCS entries = %d", count)
	}
}

func TestMainRejectsMissingPinsBeforeStartingProcess(t *testing.T) {
	var diagnostics bytes.Buffer
	if code := run(nil, &diagnostics); code == 0 || !strings.Contains(diagnostics.String(), "usage:") {
		t.Fatalf("run() = %d, diagnostics=%q", code, diagnostics.String())
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for directory := workingDirectory; ; directory = filepath.Dir(directory) {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("could not find go.mod")
		}
	}
}

var _ io.Writer = (*resultParser)(nil)
