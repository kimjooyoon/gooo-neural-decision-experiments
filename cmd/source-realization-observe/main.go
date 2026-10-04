// source-realization-observe dogfoods reusable source checkpoints through the
// real Go CLI. Only fixed public fixtures/generation records are published.
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
	"strconv"
	"time"
)

type resource struct {
	WallMS  float64 `json:"wall_ms"`
	UserS   float64 `json:"user_seconds"`
	SystemS float64 `json:"system_seconds"`
	MaxRSS  int64   `json:"maximum_resident_bytes"`
}

type row struct {
	Activity        string   `json:"activity"`
	Mode            string   `json:"mode"`
	Stage           int      `json:"stage"`
	Generation      resource `json:"generation"`
	Realization     resource `json:"realization"`
	Execution       resource `json:"execution"`
	SelectedGoooSHA string   `json:"selected_gooo_sha256"`
	GeneratedGoSHA  string   `json:"generated_go_sha256"`
	DocumentSHA     string   `json:"document_sha256"`
	ModelInputSHA   string   `json:"model_input_sha256,omitempty"`
	PredictNS       int64    `json:"predict_ns"`
	ModelCalls      int      `json:"model_calls"`
	RealizeCalls    int      `json:"realization_model_calls"`
	SelectionPassed int      `json:"selection_passed"`
	SelectionTotal  int      `json:"selection_total"`
	RuntimePassed   int      `json:"runtime_passed"`
	RuntimeTotal    int      `json:"runtime_total"`
}

type generation struct {
	GoooSource string `json:"gooo_source"`
	Source     string `json:"source"`
	Report     struct {
		BodyPaths struct {
			SelectedSHA string `json:"selected_source_sha256"`
			DocumentSHA string `json:"document_sha256"`
			Search      struct {
				Selection struct {
					Calls      int `json:"local_model_predictions"`
					Prediction *struct {
						SHA string `json:"input_sha256"`
						NS  int64  `json:"predict_ns"`
					} `json:"three_choice_prediction"`
				} `json:"selection"`
			} `json:"search"`
			NativeCases []struct {
				Passed bool `json:"passed"`
			} `json:"native_case_results"`
		} `json:"body_paths"`
	} `json:"report"`
}

var clockLine = regexp.MustCompile(`([0-9.]+) real\s+([0-9.]+) user\s+([0-9.]+) sys`)
var rssLine = regexp.MustCompile(`([0-9]+)\s+maximum resident set size`)

func main() {
	compiler := flag.String("gooo", "gooo", "clean exact-source compiler CLI")
	fixtures := flag.String("fixtures", "publication/source-assembly-20261004", "fixed public source/cases")
	model := flag.String("model", "models/own-three-feedback-v1/set-feedback/models/fp32/model.json", "unchanged own model")
	private := flag.String("private-out", "", "fresh raw/resource directory")
	public := flag.String("out", "", "fresh public generation/metrics directory")
	flag.Parse()
	if err := observe(*compiler, *fixtures, *model, *private, *public); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func observe(compiler, fixtures, model, private, public string) error {
	if private == "" || public == "" {
		return fmt.Errorf("private-out and out are required")
	}
	for _, dir := range []string{private, public} {
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}
	var rows []row
	for _, activity := range []string{"Qualified", "Clamp"} {
		for _, mode := range []string{"model", "deterministic"} {
			source := filepath.Join(fixtures, "source.gooo.fixture")
			var previous generation
			for stage := 0; stage < 3; stage++ {
				stem := fmt.Sprintf("%s-%s-%d", activity, mode, stage)
				args := []string{"body-codegen", "--json", "--activity", activity, source}
				if mode == "model" {
					args = append(args, "--path-model", model)
				}
				raw, genResource, err := invoke(compiler, private, stem+"-generation", args...)
				if err != nil {
					return err
				}
				var current generation
				if err := json.Unmarshal(raw, &current); err != nil {
					return err
				}
				if current.GoooSource == "" || (stage > 0 &&
					(current.GoooSource != previous.GoooSource || current.Source != previous.Source ||
						current.Report.BodyPaths.DocumentSHA != previous.Report.BodyPaths.DocumentSHA)) {
					return fmt.Errorf("%s: checkpoint is not a stable reusable generation", stem)
				}
				genPath := filepath.Join(public, stem+"-generation.json")
				if err := os.WriteFile(genPath, raw, 0644); err != nil {
					return err
				}
				realDir := filepath.Join(private, stem+"-realized")
				realRaw, realResource, err := invoke(compiler, private, stem+"-realization", "body-realize",
					"--source", source, "--generation", genPath, "--out", realDir)
				if err != nil {
					return err
				}
				var realized struct {
					Source string `json:"gooo_source"`
					Calls  int    `json:"model_calls"`
				}
				if err := json.Unmarshal(realRaw, &realized); err != nil {
					return err
				}
				if realized.Calls != 0 || realized.Source != current.GoooSource {
					return fmt.Errorf("%s: realization changed source or made predictions", stem)
				}
				runtimeRaw, execResource, err := invoke(compiler, private, stem+"-execution", "body-execute",
					"--source", source, "--generation", genPath, "--cases", filepath.Join(fixtures, activity+"-cases.json"))
				if err != nil {
					return err
				}
				var runtime struct {
					Observation struct {
						Replayed bool `json:"runtime_replayed"`
						Cases    []struct {
							Passed bool `json:"passed"`
						} `json:"cases"`
					} `json:"observation"`
				}
				if err := json.Unmarshal(runtimeRaw, &runtime); err != nil {
					return err
				}
				if !runtime.Observation.Replayed || len(runtime.Observation.Cases) == 0 {
					return fmt.Errorf("%s: independent runtime observation missing", stem)
				}
				p := current.Report.BodyPaths
				expectedCalls := 0
				if mode == "model" {
					expectedCalls = 1
				}
				if p.Search.Selection.Calls != expectedCalls || len(p.NativeCases) == 0 ||
					p.SelectedSHA != "sha256:"+hash([]byte(current.GoooSource)) {
					return fmt.Errorf("%s: generation metrics are absent or inconsistent", stem)
				}
				observed := row{Activity: activity, Mode: mode, Stage: stage, Generation: genResource,
					Realization: realResource, Execution: execResource, SelectedGoooSHA: hash([]byte(current.GoooSource)),
					GeneratedGoSHA: hash([]byte(current.Source)), DocumentSHA: p.DocumentSHA,
					ModelCalls: p.Search.Selection.Calls, RealizeCalls: realized.Calls,
					SelectionTotal: len(p.NativeCases), RuntimeTotal: len(runtime.Observation.Cases)}
				if prediction := p.Search.Selection.Prediction; prediction != nil {
					observed.ModelInputSHA, observed.PredictNS = prediction.SHA, prediction.NS
				}
				for _, c := range p.NativeCases {
					if c.Passed {
						observed.SelectionPassed++
					}
				}
				for _, c := range runtime.Observation.Cases {
					if c.Passed {
						observed.RuntimePassed++
					}
				}
				rows = append(rows, observed)
				source, previous = filepath.Join(realDir, "realized.gooo"), current
			}
		}
	}
	raw, err := json.MarshalIndent(struct {
		Schema string `json:"schema"`
		Scope  string `json:"scope"`
		Rows   []row  `json:"rows"`
	}{"gooo/source-realization-observation/v1",
		"Two authored tasks, two modes, three consecutive generations; each phase is a separate process; " +
			"fresh native compilation and replay per execution; resource data includes children; no host utilization measurement", rows}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(public, "metrics.json"), append(raw, '\n'), 0644)
}

func hash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func invoke(compiler, private, stem string, args ...string) ([]byte, resource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/time", append([]string{"-l", compiler}, args...)...)
	var output, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &diagnostics
	started := time.Now()
	err := cmd.Run()
	r := resource{WallMS: float64(time.Since(started).Nanoseconds()) / 1e6}
	if writeErr := os.WriteFile(filepath.Join(private, stem+".log"), diagnostics.Bytes(), 0600); writeErr != nil {
		return nil, r, writeErr
	}
	if err != nil {
		return nil, r, fmt.Errorf("%s: %w: %s", stem, err, diagnostics.String())
	}
	if err := os.WriteFile(filepath.Join(private, stem+".json"), output.Bytes(), 0600); err != nil {
		return nil, r, err
	}
	clock, rss := clockLine.FindStringSubmatch(diagnostics.String()), rssLine.FindStringSubmatch(diagnostics.String())
	if len(clock) != 4 || len(rss) != 2 {
		return nil, r, fmt.Errorf("%s: missing resource observation", stem)
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
