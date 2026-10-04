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
	InputSlots       int       `json:"observed_input_slots,omitempty"`
	NativeRuns       int       `json:"native_runs"`
	GoooSHA          string    `json:"gooo_sha256"`
	GoSHA            string    `json:"generated_go_sha256"`
	DriverSHA        string    `json:"driver_sha256"`
}

type envelope struct {
	GeneratedNow bool `json:"generated_now"`
	Composition  struct {
		Plan struct {
			Activities []struct {
				ID     string `json:"id"`
				From   int    `json:"input_from"`
				Inputs []struct {
					Port   string `json:"port"`
					Entity string `json:"entity_id"`
					From   int    `json:"from"`
				} `json:"inputs"`
			} `json:"activities"`
		} `json:"plan"`
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
				Activity string          `json:"activity_id"`
				Producer string          `json:"producer_id"`
				Input    json.RawMessage `json:"input"`
				Actual   json.RawMessage `json:"actual"`
				Expected json.RawMessage `json:"expected"`
				Inputs   []struct {
					Port     string          `json:"port"`
					Entity   string          `json:"entity_id"`
					Producer string          `json:"producer_id"`
					Value    json.RawMessage `json:"value"`
				} `json:"inputs"`
				Passed *bool `json:"passed"`
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
	inputJoins := flag.Bool("input-joins", false, "seven-activity repeated/mixed/partly bound input profile")
	flag.Parse()
	if err := observe(*compiler, *sha, *source, *cases, *model, *private, *public, *inputJoins); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func observe(compiler, sha, source, cases, model, private, public string, inputJoins bool) error {
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
			row, current, err := readObservationProfile(raw, sha, mode, stage, false, resource, inputJoins)
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
			replayed, replay, err := readObservationProfile(replayRaw, sha, mode, stage, true, replayResource, inputJoins)
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
	scope := "One seven-activity graph, two modes, three consecutive generations each and six separate saved replays; seven cases and 49 named expectations per control; process resources include native build/children; fixed order and warm cache; no host utilization or accuracy gain claim"
	if inputJoins {
		scope += "; input-joins profile has one assembly, six selection examples, six bound edges and 12 input slots per case; ordered per-port values are checked against actual producer outputs"
	}
	raw, err := json.MarshalIndent(struct {
		Schema string    `json:"schema"`
		Scope  string    `json:"scope"`
		Rows   []summary `json:"rows"`
	}{
		"gooo/native-composition-observation/v1", scope, rows}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(public, "metrics.json"), append(raw, '\n'), 0644)
}

func readObservation(raw []byte, sha, mode string, stage int, replayed bool, resource resources) (summary, envelope, error) {
	return readObservationProfile(raw, sha, mode, stage, replayed, resource, false)
}

func readObservationProfile(raw []byte, sha, mode string, stage int, replayed bool, resource resources, inputJoins bool) (summary, envelope, error) {
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
		if inputJoins {
			expectedCalls = 1
		}
	}
	expectedSelection := 9
	if inputJoins {
		expectedSelection = 6
	}
	if row.StoredModelCalls != expectedCalls || row.SelectionTotal != expectedSelection || row.SelectionPassed != expectedSelection {
		return row, observation, fmt.Errorf("selection/model call observations differ")
	}
	if !replayed {
		row.ModelCalls, row.GenerationMS = row.StoredModelCalls, float64(c.Elapsed)/1e6
	}
	row.RuntimeMS, row.RuntimePassed, row.RuntimeTotal, row.NativeRuns = float64(r.Elapsed)/1e6, r.Passed, r.Total, len(r.Runs)
	row.GoSHA, row.DriverSHA, row.GoooSHA = c.GoSHA, c.DriverSHA, fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(c.Gooo)))
	if inputJoins && len(c.Plan.Activities) != 7 {
		return row, observation, fmt.Errorf("input plan missing activities")
	}
	for _, trace := range r.Traces {
		row.ObservedValues += len(trace.Deliveries)
		if inputJoins && len(trace.Deliveries) != 7 {
			return row, observation, fmt.Errorf("input trace missing activities")
		}
		for i, delivery := range trace.Deliveries {
			if delivery.Producer != "" {
				row.Deliveries++
			}
			if inputJoins {
				node := c.Plan.Activities[i]
				if delivery.Activity != node.ID || len(delivery.Inputs) != len(node.Inputs) {
					return row, observation, fmt.Errorf("input trace arity or activity differs")
				}
				if len(delivery.Expected) == 0 || !bytes.Equal(delivery.Actual, delivery.Expected) {
					return row, observation, fmt.Errorf("named output differs from its finite expectation")
				}
				if len(node.Inputs) == 0 {
					if len(delivery.Input) == 0 {
						return row, observation, fmt.Errorf("single input value missing")
					}
					row.InputSlots++
					if node.From >= 0 && (node.From >= i || delivery.Producer != c.Plan.Activities[node.From].ID || !bytes.Equal(delivery.Input, trace.Deliveries[node.From].Actual)) {
						return row, observation, fmt.Errorf("single input differs from its producer")
					}
					if node.From < 0 && delivery.Producer != "" {
						return row, observation, fmt.Errorf("external single input has an undeclared producer")
					}
				} else {
					if len(delivery.Input) != 0 || delivery.Producer != "" {
						return row, observation, fmt.Errorf("multiple input trace has an ambiguous scalar value")
					}
					for p, input := range delivery.Inputs {
						slot := node.Inputs[p]
						if input.Port != fmt.Sprintf("input%d", p) || input.Port != slot.Port || input.Entity == "" || input.Entity != slot.Entity || len(input.Value) == 0 || bytes.Equal(input.Value, []byte("null")) {
							return row, observation, fmt.Errorf("ordered input value/identity missing")
						}
						row.InputSlots++
						if slot.From >= 0 {
							if slot.From >= i || input.Producer != c.Plan.Activities[slot.From].ID || !bytes.Equal(input.Value, trace.Deliveries[slot.From].Actual) {
								return row, observation, fmt.Errorf("ordered input differs from its producer")
							}
							row.Deliveries++
						} else if input.Producer != "" {
							return row, observation, fmt.Errorf("external input has an undeclared producer")
						}
					}
				}
			}
			if delivery.Passed == nil || !*delivery.Passed {
				return row, observation, fmt.Errorf("missing named native expectation")
			}
		}
	}
	expectedDeliveries := 35
	if inputJoins {
		expectedDeliveries = 42
	}
	if row.ObservedValues != 49 || row.Deliveries != expectedDeliveries || inputJoins && row.InputSlots != 84 {
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
