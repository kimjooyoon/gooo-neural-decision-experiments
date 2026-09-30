package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodydecision"
)

func marker(id, split string, index int, passed bool) string {
	return fmt.Sprintf("GOOO_CASE\t%s\t%s\t%d\t%t\n", id, split, index, passed)
}

func testRows() []record {
	return []record{{
		ID:       "plan-a",
		Training: []bodydecision.Case{{}, {}},
		Heldout:  []bodydecision.Case{{}},
	}}
}

func TestParseCasesSeparatesTrainingAndHeldout(t *testing.T) {
	rows := testRows()
	output := []byte(marker("plan-a", "training", 1, false) +
		marker("plan-a", "heldout", 0, false) +
		"ordinary Go test output\n" +
		marker("plan-a", "training", 0, true))

	scores, err := parseCases(output, rows)
	if err != nil {
		t.Fatalf("parseCases() error = %v", err)
	}
	if got, want := scores["plan-a"]["training"], (bodydecision.Score{Correct: 1, Total: 2}); got != want {
		t.Errorf("training score = %+v, want %+v", got, want)
	}
	if got, want := scores["plan-a"]["heldout"], (bodydecision.Score{Correct: 0, Total: 1}); got != want {
		t.Errorf("heldout score = %+v, want %+v", got, want)
	}
}

func TestParseCasesRejectsInvalidEvidence(t *testing.T) {
	valid := marker("plan-a", "training", 0, true) +
		marker("plan-a", "training", 1, false) +
		marker("plan-a", "heldout", 0, true)
	tests := []struct {
		name   string
		output string
	}{
		{
			name: "duplicate marker",
			output: marker("plan-a", "training", 0, true) +
				marker("plan-a", "training", 0, false) +
				marker("plan-a", "training", 1, false) +
				marker("plan-a", "heldout", 0, true),
		},
		{name: "missing marker", output: valid[:strings.LastIndex(valid, marker("plan-a", "heldout", 0, true))]},
		{name: "unknown plan", output: valid + marker("plan-b", "training", 0, true)},
		{name: "wrong split", output: valid + marker("plan-a", "validation", 0, true)},
		{name: "out of range index", output: valid + marker("plan-a", "training", 2, true)},
		{name: "malformed marker", output: valid + "GOOO_CASE\tplan-a\ttraining\tbad\ttrue\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseCases([]byte(tt.output), testRows()); err == nil {
				t.Fatal("parseCases() accepted invalid evidence")
			}
		})
	}
}

func TestBoundedBufferEnforcesExactLimit(t *testing.T) {
	buffer := boundedBuffer{limit: 4}
	if n, err := buffer.Write([]byte("1234")); err != nil || n != 4 {
		t.Fatalf("exact-limit write = (%d, %v), want (4, nil)", n, err)
	}
	if n, err := buffer.Write([]byte("5")); err == nil || n != 0 {
		t.Fatalf("over-limit write = (%d, %v), want an error and zero bytes", n, err)
	}
	if got := buffer.String(); got != "1234" {
		t.Fatalf("buffer after rejected write = %q, want %q", got, "1234")
	}
}

func TestCheckLayaRejectsUnsafeURLBeforeHealthRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected authorization header on health request")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"revisions":{"multilingual":"0123456789012345678901234567890123456789"}}`))
	}))
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")
	unsafe := []string{
		server.URL + "/wrong-path",
		"http://user:secret@" + host + "/v1/systemone",
		server.URL + "/v1/systemone?token=secret",
		server.URL + "/v1/systemone#fragment",
		"https://" + host + "/v1/systemone",
		"http://192.0.2.1/v1/systemone",
	}
	for _, endpoint := range unsafe {
		t.Run(endpoint, func(t *testing.T) {
			if err := checkLaya(endpoint, "0123456789012345678901234567890123456789"); err == nil {
				t.Fatalf("checkLaya(%q) unexpectedly succeeded", endpoint)
			}
		})
	}
	if requests != 0 {
		t.Fatalf("unsafe endpoints triggered %d health requests, want zero", requests)
	}
}

func TestContainCommandCancellationKillsProcessGroup(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("process-group cancellation is supported on macOS and Linux")
	}
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "child.pid")
	markerFile := filepath.Join(tempDir, "child-survived")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", `(sleep 0.5; echo escaped > "$2") & echo $! > "$1"; wait`, "sh", pidFile, markerFile)
	if err := containCommand(command); err != nil {
		t.Fatalf("containCommand() error = %v", err)
	}
	if err := command.Start(); err != nil {
		t.Fatalf("start child process group: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(pidFile); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			_ = command.Wait()
			t.Fatal("child process did not start before cancellation")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	select {
	case <-waited:
	case <-time.After(3 * time.Second):
		t.Fatal("process group did not stop after cancellation")
	}
	time.Sleep(650 * time.Millisecond)
	if _, err := os.Stat(markerFile); err == nil {
		t.Fatal("child process survived cancellation and wrote its marker")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat child marker: %v", err)
	}
}
