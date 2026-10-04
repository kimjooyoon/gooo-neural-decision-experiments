// native-composition-observe measures the actual Gooo CLI with unchanged own
// weights, finite public graph cases and separate saved-composition controls.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

type resources struct {
	WallMS  float64 `json:"wall_ms"`
	UserS   float64 `json:"user_seconds"`
	SystemS float64 `json:"system_seconds"`
	MaxRSS  int64   `json:"maximum_resident_bytes"`
}

type summary struct {
	Mode             string    `json:"mode"`
	Stage            int       `json:"stage"`
	Replayed         bool      `json:"replayed"`
	Resources        resources `json:"resources"`
	GenerationMS     float64   `json:"generation_ms"`
	RuntimeMS        float64   `json:"runtime_ms"`
	ModelCalls       int       `json:"actual_model_calls"`
	StoredModelCalls int       `json:"stored_generation_model_calls"`
	SelectionPassed  int       `json:"selection_passed"`
	SelectionTotal   int       `json:"selection_total"`
	RuntimePassed    int       `json:"runtime_passed"`
	RuntimeTotal     int       `json:"runtime_total"`
	ObservedValues   int       `json:"observed_values"`
	Deliveries       int       `json:"observed_edge_deliveries"`
	NativeRuns       int       `json:"native_runs"`
	GoooSHA          string    `json:"gooo_sha256"`
	GoSHA            string    `json:"generated_go_sha256"`
	DriverSHA        string    `json:"driver_sha256"`
}

type envelope struct {
	GeneratedNow bool `json:"generated_now"`
	Composition  struct {
		Stage     string `json:"stage"`
		Failure   string `json:"failure"`
		Gooo      string `json:"gooo_source"`
		GoSHA     string `json:"generated_sha256"`
		DriverSHA string `json:"driver_sha256"`
		Elapsed   int64  `json:"elapsed_ns"`
		Steps     []struct {
			Generation struct {
				Report struct {
					Compiler string `json:"compiler_source_sha"`
					Paths    *struct {
						Search struct {
							Selection struct {
								Calls int `json:"local_model_predictions"`
							} `json:"selection"`
						} `json:"search"`
						Cases []struct {
							Passed bool `json:"passed"`
						} `json:"native_case_results"`
					} `json:"body_paths"`
				} `json:"report"`
			} `json:"generation"`
		} `json:"steps"`
	} `json:"composition"`
	Runtime struct {
		Stage    string `json:"stage"`
		Failure  string `json:"failure"`
		Replayed bool   `json:"runtime_replayed"`
		Calls    int    `json:"model_calls"`
		Elapsed  int64  `json:"elapsed_ns"`
		Passed   int    `json:"finite_passed"`
		Total    int    `json:"finite_total"`
		Runs     []struct {
			Completed bool `json:"completed"`
			Exit      *int `json:"exit_code"`
		} `json:"runs"`
		Traces []struct {
			Deliveries []struct {
				Producer string `json:"producer_id"`
				Passed   *bool  `json:"passed"`
			} `json:"deliveries"`
		} `json:"traces"`
	} `json:"runtime"`
}

var cpuLine = regexp.MustCompile(`([0-9.]+) real\s+([0-9.]+) user\s+([0-9.]+) sys`)
var rssLine = regexp.MustCompile(`([0-9]+)\s+maximum resident set size`)

func main() {
	compiler := flag.String("gooo", "gooo", "clean compiler executable")
	sha := flag.String("compiler-source", "", "exact compiler source SHA")
	source := flag.String("source", "", "fixed public Gooo graph")
	cases := flag.String("cases", "", "fixed public runtime cases")
	model := flag.String("model", "models/own-three-feedback-v1/set-feedback/models/fp32/model.json", "unchanged own model")
	private := flag.String("private-out", "", "new process resource log directory")
	public := flag.String("out", "", "new public evidence directory")
	flag.Parse()
	if err := observe(*compiler, *sha, *source, *cases, *model, *private, *public); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func observe(compiler, sha, source, cases, model, private, public string) error {
	if runtime.GOOS != "darwin" || len(sha) != 40 || private == "" || public == "" {
		return fmt.Errorf("macOS resources, exact source SHA and two fresh output directories required")
	}
	for _, directory := range []string{private, public} {
		if err := os.Mkdir(directory, 0755); err != nil {
			return err
		}
	}
	for name, path := range map[string]string{"source.gooo.fixture": source, "cases.json": cases} {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(public, name), raw, 0644); err != nil {
			return err
		}
	}
	var rows []summary
	for _, mode := range []string{"model", "deterministic"} {
		input := "source.gooo.fixture"
		var previous envelope
		for stage := range 3 {
			stem := fmt.Sprintf("%s-%d", mode, stage)
			args := []string{"body-compose", "--source", input, "--cases", "cases.json"}
			if mode == "model" {
				args = append(args, "--model", model)
			}
			raw, resource, err := invoke(compiler, public, private, stem, args...)
			if err != nil {
				return err
			}
			row, current, err := readObservation(raw, sha, mode, stage, false, resource)
			if err != nil {
				return err
			}
			if stage > 0 && (current.Composition.Gooo != previous.Composition.Gooo || current.Composition.GoSHA != previous.Composition.GoSHA || current.Composition.DriverSHA != previous.Composition.DriverSHA) {
				return fmt.Errorf("%s source/code/driver fixed point differs", stem)
			}
			if err := os.WriteFile(filepath.Join(public, stem+".json"), raw, 0644); err != nil {
				return err
			}
			var original struct {
				Composition json.RawMessage `json:"composition"`
			}
			if err := json.Unmarshal(raw, &original); err != nil {
				return err
			}
			compositionFile := stem + "-composition.json"
			if err := os.WriteFile(filepath.Join(public, compositionFile), original.Composition, 0644); err != nil {
				return err
			}
			rows = append(rows, row)
			replayRaw, replayResource, err := invoke(compiler, public, private, stem+"-replay", "body-compose", "--source", input, "--cases", "cases.json", "--composition", compositionFile)
			if err != nil {
				return err
			}
			replayed, replay, err := readObservation(replayRaw, sha, mode, stage, true, replayResource)
			if err != nil {
				return err
			}
			if replay.Composition.Gooo != current.Composition.Gooo || replay.Composition.GoSHA != current.Composition.GoSHA || replay.Composition.DriverSHA != current.Composition.DriverSHA {
				return fmt.Errorf("%s saved replay differs", stem)
			}
			if err := os.WriteFile(filepath.Join(public, stem+"-replay.json"), replayRaw, 0644); err != nil {
				return err
			}
			rows = append(rows, replayed)
			input, previous = stem+"-checkpoint.gooo.fixture", current
			if err := os.WriteFile(filepath.Join(public, input), []byte(current.Composition.Gooo), 0644); err != nil {
				return err
			}
		}
	}
	raw, err := json.MarshalIndent(struct {
		Schema string    `json:"schema"`
		Scope  string    `json:"scope"`
		Rows   []summary `json:"rows"`
	}{
		"gooo/native-composition-observation/v1", "One seven-activity graph, two modes, three consecutive generations each and six separate saved replays; seven cases and 49 named expectations per control; process resources include native build/children; fixed order and warm cache; no host utilization or accuracy gain claim", rows}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(public, "metrics.json"), append(raw, '\n'), 0644)
}

func readObservation(raw []byte, sha, mode string, stage int, replayed bool, resource resources) (summary, envelope, error) {
	var observation envelope
	row := summary{Mode: mode, Stage: stage, Replayed: replayed, Resources: resource}
	if err := json.Unmarshal(raw, &observation); err != nil {
		return row, observation, err
	}
	c, r := observation.Composition, observation.Runtime
	if observation.GeneratedNow == replayed || c.Stage != "COMPLETE" || c.Failure != "" || r.Stage != "COMPLETE" || r.Failure != "" || !r.Replayed || r.Calls != 0 || len(c.Steps) != 7 || len(r.Traces) != 7 || len(r.Runs) != 2 || r.Total != 49 || r.Passed != 49 {
		return row, observation, fmt.Errorf("missing successful graph/runtime fields")
	}
	for _, run := range r.Runs {
		if !run.Completed || run.Exit == nil || *run.Exit != 0 {
			return row, observation, fmt.Errorf("native run incomplete")
		}
	}
	for _, step := range c.Steps {
		if step.Generation.Report.Compiler != sha {
			return row, observation, fmt.Errorf("generation compiler source differs")
		}
		if paths := step.Generation.Report.Paths; paths != nil {
			row.StoredModelCalls += paths.Search.Selection.Calls
			row.SelectionTotal += len(paths.Cases)
			for _, test := range paths.Cases {
				if test.Passed {
					row.SelectionPassed++
				}
			}
		}
	}
	expectedCalls := 0
	if mode == "model" {
		expectedCalls = 2
	}
	if row.StoredModelCalls != expectedCalls || row.SelectionTotal != 9 || row.SelectionPassed != 9 {
		return row, observation, fmt.Errorf("selection/model call observations differ")
	}
	if !replayed {
		row.ModelCalls, row.GenerationMS = row.StoredModelCalls, float64(c.Elapsed)/1e6
	}
	row.RuntimeMS, row.RuntimePassed, row.RuntimeTotal, row.NativeRuns = float64(r.Elapsed)/1e6, r.Passed, r.Total, len(r.Runs)
	row.GoSHA, row.DriverSHA, row.GoooSHA = c.GoSHA, c.DriverSHA, fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(c.Gooo)))
	for _, trace := range r.Traces {
		row.ObservedValues += len(trace.Deliveries)
		for _, delivery := range trace.Deliveries {
			if delivery.Producer != "" {
				row.Deliveries++
			}
			if delivery.Passed == nil || !*delivery.Passed {
				return row, observation, fmt.Errorf("missing named native expectation")
			}
		}
	}
	if row.ObservedValues != 49 || row.Deliveries != 35 {
		return row, observation, fmt.Errorf("native trace count differs")
	}
	return row, observation, nil
}

func invoke(compiler, public, private, stem string, args ...string) ([]byte, resources, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/time", append([]string{"-l", compiler}, args...)...)
	cmd.Dir = public
	var output, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &diagnostics
	start := time.Now()
	err := cmd.Run()
	r := resources{WallMS: float64(time.Since(start).Nanoseconds()) / 1e6}
	if writeErr := os.WriteFile(filepath.Join(private, stem+".log"), diagnostics.Bytes(), 0600); writeErr != nil {
		return nil, r, writeErr
	}
	if err != nil {
		return nil, r, fmt.Errorf("%s: %w", stem, err)
	}
	clock, rss := cpuLine.FindStringSubmatch(diagnostics.String()), rssLine.FindStringSubmatch(diagnostics.String())
	if len(clock) != 4 || len(rss) != 2 {
		return nil, r, fmt.Errorf("resource observations missing")
	}
	r.UserS, err = strconv.ParseFloat(clock[2], 64)
	if err != nil {
		return nil, r, err
	}
	r.SystemS, err = strconv.ParseFloat(clock[3], 64)
	if err != nil {
		return nil, r, err
	}
	r.MaxRSS, err = strconv.ParseInt(rss[1], 10, 64)
	return output.Bytes(), r, err
}
