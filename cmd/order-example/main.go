// Run the public order example through retained Gooo generation and immediate
// native execution. Every generation and execution receipt remains on disk.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type options struct {
	compiler, goBin, model, source, recipe, cases, out string
	repeat                                             int
	retainNative                                       bool
	command                                            func(context.Context, string, ...string) *exec.Cmd
}

func (o options) cmd(ctx context.Context, args ...string) *exec.Cmd {
	var c *exec.Cmd
	if o.command != nil {
		c = o.command(ctx, o.compiler, args...)
	} else {
		c = exec.CommandContext(ctx, o.compiler, args...)
	}
	bindCommandGroup(c)
	return c
}

type result struct {
	Schema, Status string
	CorrelationID  string `json:"correlation_id"`
	Sequence       int
	Response       json.RawMessage
	Error          string
	Execution      json.RawMessage
}

type receipt struct {
	ID                        string  `json:"id"`
	ResponseMS                float64 `json:"response_ms"`
	ExecutionMS               float64 `json:"execution_ms"`
	ModelPredictions          int     `json:"model_predictions"`
	PreparationReused         bool    `json:"preparation_reused"`
	NativeRuns                int     `json:"native_runs"`
	Passed                    int     `json:"passed"`
	Total                     int     `json:"total"`
	ResponseIncludesExecution bool    `json:"response_includes_execution,omitempty"`
	NativeArtifactReused      *bool   `json:"native_artifact_reused,omitempty"`
}

func readInput(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (128<<10)+1))
	if err == nil && len(b) > 128<<10 {
		err = fmt.Errorf("%s exceeds 128 KiB", path)
	}
	return b, err
}

func decodeResult(line []byte, id string, sequence int) (json.RawMessage, error) {
	var r result
	if err := json.Unmarshal(line, &r); err != nil {
		return nil, err
	}
	if r.Schema != "gooo/native-body-stream-result/v1" || r.CorrelationID != id || r.Sequence != sequence {
		return nil, errors.New("stream response identity differs")
	}
	if r.Status != "completed" || len(r.Response) == 0 || string(r.Response) == "null" {
		return nil, fmt.Errorf("request %s: %s: %s", id, r.Status, r.Error)
	}
	return r.Response, nil
}

func execute(ctx context.Context, o options, generation string, stderr io.Writer) (receipt, error) {
	var r receipt
	runtimePath := strings.TrimSuffix(generation, "-generation.json") + "-runtime.json"
	file, err := os.OpenFile(runtimePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return r, err
	}
	c := o.cmd(ctx, "body-execute", "--source", filepath.Join(o.out, "source.gooo"),
		"--path-plan", filepath.Join(o.out, "recipe.json"), "--generation", generation,
		"--cases", filepath.Join(o.out, "cases.json"), "--go-bin", o.goBin)
	c.Stdout, c.Stderr, c.WaitDelay = file, stderr, 3*time.Second
	start := time.Now()
	err = c.Run()
	r.ExecutionMS = float64(time.Since(start)) / float64(time.Millisecond)
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		return r, errors.Join(err, closeErr)
	}
	b, err := os.ReadFile(runtimePath)
	if err != nil {
		return r, err
	}
	parsed, err := parseNative(b)
	parsed.ExecutionMS = r.ExecutionMS
	return parsed, err
}

func parseNative(b []byte) (receipt, error) {
	var r receipt
	var n struct {
		Observation struct {
			Stage     string
			Runs      []json.RawMessage
			Cases     []struct{ Passed bool }
			ElapsedNS int64 `json:"elapsed_ns"`
			Artifact  *struct{ Reused bool }
		}
	}
	if err := json.Unmarshal(b, &n); err != nil {
		return r, err
	}
	if n.Observation.Stage != "COMPLETE" || len(n.Observation.Runs) != 2 || len(n.Observation.Cases) == 0 {
		return r, errors.New("native execution is incomplete; inspect retained report")
	}
	r.NativeRuns, r.Total = len(n.Observation.Runs), len(n.Observation.Cases)
	r.ExecutionMS = float64(n.Observation.ElapsedNS) / float64(time.Millisecond)
	if n.Observation.Artifact != nil {
		r.NativeArtifactReused = &n.Observation.Artifact.Reused
	}
	for _, c := range n.Observation.Cases {
		if c.Passed {
			r.Passed++
		}
	}
	return r, nil
}

func run(ctx context.Context, o options, stdout, stderr io.Writer) error {
	if o.repeat < 1 || o.repeat > 16 || o.out == "" {
		return errors.New("--out is required; --repeat must be 1..16")
	}
	inputs := make([][]byte, 3)
	for i, path := range []string{o.source, o.recipe, o.cases} {
		b, err := readInput(path)
		if err != nil {
			return err
		}
		if i > 0 && !json.Valid(b) {
			return fmt.Errorf("invalid JSON: %s", path)
		}
		inputs[i] = b
	}
	if err := os.Mkdir(o.out, 0755); err != nil {
		return fmt.Errorf("create fresh output directory: %w", err)
	}
	for i, name := range []string{"source.gooo", "recipe.json", "cases.json"} {
		if err := os.WriteFile(filepath.Join(o.out, name), inputs[i], 0644); err != nil {
			return err
		}
	}
	args := []string{"body-path-stream", "--workers", "1"}
	if o.retainNative {
		args = append(args, "--execute", "--go-bin", o.goBin)
	}
	if o.model != "" {
		args = append(args, "--model", o.model)
	}
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	c := o.cmd(streamCtx, args...)
	c.Stderr, c.WaitDelay = stderr, 3*time.Second
	in, err := c.StdinPipe()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	if err := c.Start(); err != nil {
		return err
	}
	defer func() {
		cancel()
		if c.ProcessState == nil {
			_ = c.Wait()
		}
	}()
	encoder, scanner := json.NewEncoder(in), bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	rows := make([]receipt, 0, o.repeat)
	for i := range o.repeat {
		id := fmt.Sprintf("order-%d", i+1)
		start := time.Now()
		request := map[string]any{"schema": "gooo/native-body-stream-request/v1", "correlation_id": id,
			"source": string(inputs[0]), "activity": "Compose", "document": json.RawMessage(inputs[1])}
		if o.retainNative {
			request["execution_cases"] = json.RawMessage(inputs[2])
		}
		err := encoder.Encode(request)
		if err != nil {
			return err
		}
		if !scanner.Scan() {
			return errors.Join(errors.New("stream ended before response"), scanner.Err())
		}
		elapsed := time.Since(start)
		generation := filepath.Join(o.out, id+"-generation.json")
		// Preserve the entire response, including rejections, before interpreting it.
		if err := os.WriteFile(filepath.Join(o.out, id+"-response.json"), scanner.Bytes(), 0644); err != nil {
			return err
		}
		var embedded result
		if o.retainNative {
			if err := json.Unmarshal(scanner.Bytes(), &embedded); err != nil {
				return err
			}
			if len(embedded.Execution) > 0 {
				if err := os.WriteFile(filepath.Join(o.out, id+"-runtime.json"), embedded.Execution, 0644); err != nil {
					return err
				}
			}
		}
		body, err := decodeResult(scanner.Bytes(), id, i+1)
		if err != nil {
			return err
		}
		if err := os.WriteFile(generation, body, 0644); err != nil {
			return err
		}
		var r receipt
		if o.retainNative {
			r, err = parseNative(embedded.Execution)
			r.ResponseIncludesExecution = true
		} else {
			r, err = execute(ctx, o, generation, stderr)
		}
		if err != nil {
			return err
		}
		r.ID, r.ResponseMS = id, float64(elapsed)/float64(time.Millisecond)
		var g struct {
			Report struct {
				Paths struct {
					Search struct {
						Selection struct {
							Calls int `json:"local_model_predictions"`
						}
					}
					Preparation struct{ Reused bool } `json:"whole_candidate_preparation"`
				} `json:"body_paths"`
			}
		}
		if err := json.Unmarshal(body, &g); err != nil {
			return err
		}
		r.ModelPredictions, r.PreparationReused = g.Report.Paths.Search.Selection.Calls, g.Report.Paths.Preparation.Reused
		rows = append(rows, r)
		if err := json.NewEncoder(stdout).Encode(r); err != nil {
			return err
		}
	}
	if err := in.Close(); err != nil {
		return err
	}
	if scanner.Scan() || scanner.Err() != nil {
		return errors.Join(errors.New("unexpected trailing stream result"), scanner.Err())
	}
	if err := c.Wait(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(o.out, "summary.json"), append(b, '\n'), 0644); err != nil {
		return err
	}
	for _, r := range rows {
		if r.Passed != r.Total {
			return errors.New("finite expectations remain unmet; inspect retained reports")
		}
	}
	return nil
}

func main() {
	var o options
	flag.StringVar(&o.compiler, "compiler", "gooo", "installed Gooo with body-path-stream")
	flag.StringVar(&o.goBin, "go-bin", "go", "Go compiler for native execution")
	flag.StringVar(&o.model, "model", "", "explicit local model.json; omission is deterministic")
	flag.StringVar(&o.source, "source", "examples/whole-candidate-order/source.gooo", "Gooo source")
	flag.StringVar(&o.recipe, "recipe", "examples/whole-candidate-order/recipe.json", "source recipe")
	flag.StringVar(&o.cases, "cases", "examples/whole-candidate-order/cases.json", "independent finite execution cases")
	flag.StringVar(&o.out, "out", "", "fresh output directory")
	flag.IntVar(&o.repeat, "repeat", 2, "sequential requests with stdin kept open (1..16)")
	flag.BoolVar(&o.retainNative, "retain-native", false, "use stream --execute and one owned executable; requires the new compiler entrypoint")
	flag.Parse()
	interruptCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(interruptCtx, 90*time.Second)
	defer cancel()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional argument")
		os.Exit(2)
	}
	if err := run(ctx, o, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "order-example:", err)
		os.Exit(1)
	}
}
