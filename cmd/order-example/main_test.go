package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeResult(t *testing.T) {
	good := `{"schema":"gooo/native-body-stream-result/v1","status":"completed","correlation_id":"order-1","sequence":1,"response":{"source":"return input"}}`
	if _, err := decodeResult([]byte(good), "order-1", 1); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{`, strings.Replace(good, "order-1", "other", 1),
		strings.Replace(good, `"sequence":1`, `"sequence":2`, 1),
		strings.Replace(good, "completed", "rejected", 1), strings.Replace(good, `{"source":"return input"}`, `null`, 1)} {
		if _, err := decodeResult([]byte(bad), "order-1", 1); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestInputBound(t *testing.T) {
	p := filepath.Join(t.TempDir(), "source.gooo")
	for _, size := range []int{128 << 10, (128 << 10) + 1} {
		if err := os.WriteFile(p, make([]byte, size), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := readInput(p)
		if (err == nil) != (size == 128<<10) {
			t.Fatalf("size=%d: %v", size, err)
		}
	}
}

func TestImmediateExecutionAndFiniteFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "partial"}[fail], func(t *testing.T) {
			dir := t.TempDir()
			for name, b := range map[string][]byte{"source": []byte("public source"), "recipe": []byte(`{}`), "cases": []byte(`[]`)} {
				if err := os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
					t.Fatal(err)
				}
			}
			o := options{compiler: filepath.Join(dir, "output"), out: filepath.Join(dir, "output"),
				source: filepath.Join(dir, "source"), recipe: filepath.Join(dir, "recipe"), cases: filepath.Join(dir, "cases"), repeat: 2}
			if fail {
				o.goBin = "partial"
			}
			o.command = func(ctx context.Context, binary string, args ...string) *exec.Cmd {
				c := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=TestExampleProcess", "--", binary}, args...)...)
				c.Env = append(os.Environ(), "GOOO_EXAMPLE_PROCESS=1")
				return c
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var output bytes.Buffer
			err := run(ctx, o, &output, io.Discard)
			if (err != nil) != fail {
				t.Fatalf("partial=%v: %v", fail, err)
			}
			b, err := os.ReadFile(filepath.Join(o.out, "summary.json"))
			if err != nil {
				t.Fatal(err)
			}
			var rows []receipt
			if err := json.Unmarshal(b, &rows); err != nil || len(rows) != 2 || rows[1].NativeRuns != 2 {
				t.Fatalf("receipts: %s: %v", b, err)
			}
			if err := run(ctx, o, io.Discard, io.Discard); err == nil {
				t.Fatal("reused an existing output directory")
			}
		})
	}
}

func TestBlockedResponseCancellation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "input.json")
	if err := os.WriteFile(p, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	o := options{compiler: filepath.Join(dir, "output"), out: filepath.Join(dir, "output"),
		source: p, recipe: p, cases: p, model: "block", repeat: 1}
	o.command = func(ctx context.Context, binary string, args ...string) *exec.Cmd {
		c := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=TestExampleProcess", "--", binary}, args...)...)
		c.Env = append(os.Environ(), "GOOO_EXAMPLE_PROCESS=1")
		return c
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := run(ctx, o, io.Discard, io.Discard); err == nil || time.Since(start) > 2*time.Second {
		t.Fatalf("blocked child: %v, %s", err, time.Since(start))
	}
}

// The stream child requires each prior body to have executed before the next
// request. It waits for EOF after its last result, exercising open-input use.
func TestExampleProcess(t *testing.T) {
	if os.Getenv("GOOO_EXAMPLE_PROCESS") != "1" {
		return
	}
	args := os.Args[3:]
	for _, arg := range args {
		if arg == "block" {
			for {
				time.Sleep(time.Hour)
			}
		}
	}
	if args[1] == "body-path-stream" {
		d := json.NewDecoder(os.Stdin)
		for sequence := 1; ; sequence++ {
			var request struct {
				CorrelationID string `json:"correlation_id"`
			}
			if err := d.Decode(&request); err == io.EOF {
				os.Exit(0)
			} else if err != nil {
				os.Exit(3)
			}
			if sequence > 1 {
				if _, err := os.Stat(filepath.Join(args[0], "executed")); err != nil {
					os.Exit(4)
				}
			}
			b := map[string]any{"schema": "gooo/native-body-stream-result/v1", "status": "completed",
				"correlation_id": request.CorrelationID, "sequence": sequence, "response": map[string]any{"source": "return input"}}
			if err := json.NewEncoder(os.Stdout).Encode(b); err != nil {
				os.Exit(5)
			}
		}
	}
	passed := true
	for _, arg := range args {
		if arg == "partial" {
			passed = false
		}
	}
	if err := os.WriteFile(filepath.Join(args[0], "executed"), []byte("executed before next request"), 0644); err != nil {
		os.Exit(6)
	}
	b := map[string]any{"observation": map[string]any{"stage": "COMPLETE", "runs": []any{map[string]any{}, map[string]any{}},
		"cases": []any{map[string]any{"passed": passed}, map[string]any{"passed": true}}}}
	if err := json.NewEncoder(os.Stdout).Encode(b); err != nil {
		os.Exit(7)
	}
	os.Exit(0)
}
