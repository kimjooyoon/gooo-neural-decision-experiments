// Full-input-native-observe records V3/V4 model choices followed immediately by
// compiled execution. The earlier native-runtime-observe source remains frozen.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func sha(raw []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }
func save(path string, value any) {
	raw, err := json.MarshalIndent(value, "", "  ")
	must(err)
	must(os.WriteFile(path, append(raw, '\n'), 0600))
}

// Do not embed bytes.Buffer: its promoted ReaderFrom would bypass Write's bound.
type capture struct {
	data    bytes.Buffer
	maximum int
}

func (c *capture) Write(raw []byte) (int, error) {
	if c.data.Len()+len(raw) > c.maximum {
		n, _ := c.data.Write(raw[:c.maximum-c.data.Len()])
		return n, fmt.Errorf("capture exceeds limit")
	}
	return c.data.Write(raw)
}

type processCost struct {
	WallNS       int64  `json:"wall_ns"`
	UserNS       int64  `json:"user_ns"`
	SystemNS     int64  `json:"system_ns"`
	PeakRSSBytes int64  `json:"peak_rss_bytes,omitempty"`
	ExitCode     *int   `json:"exit_code"`
	TimedOut     bool   `json:"timed_out"`
	Diagnostics  []byte `json:"stderr_bytes,omitempty"`
}

func childBound(dir, binary string, limit int, args ...string) ([]byte, processCost, error) {
	ctx, stop := context.WithTimeout(context.Background(), 75*time.Second)
	defer stop()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "PATH", "HOME", "TMPDIR", "TEMP", "TMP", "SystemRoot", "USERPROFILE", "LOCALAPPDATA", "GOCACHE":
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "GOTOOLCHAIN=local", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off")
	out, diagnostics := &capture{maximum: limit}, &capture{maximum: 64 << 10}
	cmd.Stdout, cmd.Stderr = out, diagnostics
	start := time.Now()
	err := cmd.Run()
	cost := processCost{WallNS: time.Since(start).Nanoseconds()}
	cost.TimedOut = ctx.Err() != nil
	cost.Diagnostics = bytes.Clone(diagnostics.data.Bytes())
	if cmd.ProcessState != nil {
		exit := cmd.ProcessState.ExitCode()
		cost.ExitCode = &exit
		cost.UserNS = cmd.ProcessState.UserTime().Nanoseconds()
		cost.SystemNS = cmd.ProcessState.SystemTime().Nanoseconds()
		if usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			cost.PeakRSSBytes = usage.Maxrss
			if runtime.GOOS != "darwin" {
				cost.PeakRSSBytes *= 1024
			}
		}
	}
	if err == nil && diagnostics.data.Len() != 0 {
		err = fmt.Errorf("unexpected diagnostics")
	}
	return bytes.Clone(out.data.Bytes()), cost, err
}

type generation struct {
	Source string `json:"source"`
	Report struct {
		Revision string          `json:"compiler_source_sha"`
		Receipt  json.RawMessage `json:"completeness_receipt"`
		Paths    struct {
			Search struct {
				Selection struct {
					Calls    int  `json:"local_model_predictions"`
					External int  `json:"external_provider_calls"`
					Known    bool `json:"external_provider_calls_known"`
				} `json:"selection"`
			} `json:"search"`
		} `json:"body_paths"`
	} `json:"report"`
}
type dimension struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Numerator   int    `json:"numerator"`
	Denominator int    `json:"denominator"`
}
type runtimeResult struct {
	Parent      []byte `json:"parent_receipt_bytes"`
	Observation struct {
		Stage     string `json:"stage"`
		Producer  string `json:"producer_source_sha"`
		ParentSHA string `json:"parent_receipt_sha256"`
		Cases     []struct {
			Input    int64 `json:"input"`
			Expected int64 `json:"expected"`
			Actual   int64 `json:"actual"`
			Passed   bool  `json:"passed"`
		} `json:"cases"`
	} `json:"observation"`
	Receipt struct {
		Profile    string      `json:"profile_id"`
		Aggregate  any         `json:"aggregate_completeness_score"`
		Dimensions []dimension `json:"dimensions"`
		First      struct {
			ID string `json:"id"`
		} `json:"first_unresolved"`
	} `json:"completeness_receipt"`
}

func runtimeAxis(result runtimeResult, id string) dimension {
	for _, d := range result.Receipt.Dimensions {
		if d.ID == id {
			return d
		}
	}
	panic("missing runtime axis " + id)
}

func main() {
	if len(os.Args) != 6 {
		panic("usage: full-input-native-observe output compiler go-binary compiler-source-sha arithmetic-bundle")
	}
	runFullInput(os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5])
}
