// source-assembly-observe compares two authoring forms of the same public finite
// contracts. It invokes the Go compiler CLI; it does no training or inference itself.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type finiteCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}

type record struct {
	Activity         string  `json:"activity"`
	Form             string  `json:"form"`
	Mode             string  `json:"mode"`
	Sequence         int     `json:"sequence"`
	ResponseMS       float64 `json:"response_ms"`
	GenerationMS     float64 `json:"generation_ms"`
	ModelCalls       int     `json:"model_calls"`
	SelectionPassed  int     `json:"selection_passed"`
	SelectionTotal   int     `json:"selection_total"`
	RuntimePassed    int     `json:"runtime_passed"`
	RuntimeTotal     int     `json:"runtime_total"`
	NativeRuns       int     `json:"native_runs"`
	ArtifactReused   bool    `json:"artifact_reused"`
	GeneratedSHA256  string  `json:"generated_sha256"`
	ModelMetadataSHA string  `json:"model_metadata_sha256,omitempty"`
	ModelWeightsSHA  string  `json:"model_weights_sha256,omitempty"`
	ModelInputSHA    string  `json:"model_input_sha256,omitempty"`
	PredictNS        int64   `json:"predict_ns"`
	Evaluated        int     `json:"evaluated_candidates"`
	FirstPassed      int     `json:"first_candidate_passed"`
	FirstTotal       int     `json:"first_candidate_total"`
}

var (
	assemblyBlock = regexp.MustCompile(`(?s) assembling \{[^}]*\}`)
	caseRow       = regexp.MustCompile(`case "(-?[0-9]+)" -> "(-?[0-9]+)"`)
	attemptRow    = regexp.MustCompile(`attempts "([0-9]+)"`)
)

func main() {
	compiler := flag.String("gooo", "gooo", "compiler CLI")
	root := flag.String("compiler-root", "", "compiler checkout with public source fixture")
	model := flag.String("model", "models/own-three-feedback-v1/set-feedback/models/fp32/model.json", "unchanged own model")
	private := flag.String("private-out", "", "fresh private raw observation directory")
	public := flag.String("out", "", "fresh public fixture/metrics directory")
	only := flag.Bool("summarize-only", false, "read retained observations; no model or native calls")
	flag.Parse()
	var err error
	if *only {
		err = summarize(*private, *public)
	} else {
		err = observe(*compiler, *root, *model, *private, *public)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func observe(compiler, root, model, private, public string) error {
	if root == "" || private == "" || public == "" {
		return fmt.Errorf("compiler-root, private-out and out are required")
	}
	for _, dir := range []string{private, public} {
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}
	source, err := os.ReadFile(filepath.Join(root, "examples/body-codegen/source-assembly.gooo.fixture"))
	if err != nil {
		return err
	}
	plain := assemblyBlock.ReplaceAll(source, nil)
	if bytes.Equal(source, plain) {
		return fmt.Errorf("expected the fixed public assembly fixture")
	}
	for name, data := range map[string][]byte{"source.gooo.fixture": source, "external.gooo.fixture": plain} {
		if err := os.WriteFile(filepath.Join(public, name), data, 0644); err != nil {
			return err
		}
	}
	build, err := exec.Command(compiler, "version", "--build", "--json").Output()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(public, "build.json"), build, 0644); err != nil {
		return err
	}
	var rows []record
	for _, activity := range []string{"Qualified", "Clamp"} {
		plan, err := fixtureDocument(compiler, source, filepath.Join(public, "source.gooo.fixture"), activity)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(public, activity+"-plan.json"), plan, 0644); err != nil {
			return err
		}
		casesPath := "publication/condition-path-assembly-20261004/runtime-cases.json"
		if activity == "Clamp" {
			casesPath = filepath.Join(root, "examples/body-codegen/source-assembly-clamp-cases.json")
		}
		cases, err := os.ReadFile(casesPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(public, activity+"-cases.json"), cases, 0644); err != nil {
			return err
		}
		for _, form := range []string{"source", "external"} {
			for _, mode := range []string{"model", "deterministic"} {
				stem := activity + "-" + form + "-" + mode
				args := []string{"body-path-run", "--source", filepath.Join(public, "source.gooo.fixture"),
					"--activity", activity, "--cases", filepath.Join(public, activity+"-cases.json"),
					"--repeat", "4", "--timing", "--out", filepath.Join(private, stem)}
				if form == "external" {
					args[2] = filepath.Join(public, "external.gooo.fixture")
					args = append(args, "--path-plan", filepath.Join(public, activity+"-plan.json"))
				}
				if mode == "model" {
					args = append(args, "--model", model)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				cmd := exec.CommandContext(ctx, "/usr/bin/time", append([]string{"-l", compiler}, args...)...)
				var output, diagnostics bytes.Buffer
				cmd.Stdout, cmd.Stderr = &output, &diagnostics
				runErr := cmd.Run()
				cancel()
				if err := os.WriteFile(filepath.Join(private, stem+".log"), diagnostics.Bytes(), 0644); err != nil {
					return err
				}
				if runErr != nil {
					return fmt.Errorf("%s: %w: %s", stem, runErr, diagnostics.String())
				}
				current, err := readRows(filepath.Join(private, stem), activity, form, mode, public)
				if err != nil {
					return err
				}
				rows = append(rows, current...)
			}
		}
	}
	if err := writeReport(public, rows); err != nil {
		return err
	}
	return writeResources(private, public)
}

func summarize(private, public string) error {
	var rows []record
	for _, activity := range []string{"Qualified", "Clamp"} {
		for _, form := range []string{"source", "external"} {
			for _, mode := range []string{"model", "deterministic"} {
				stem := activity + "-" + form + "-" + mode
				current, err := readRows(filepath.Join(private, stem), activity, form, mode, public)
				if err != nil {
					return err
				}
				rows = append(rows, current...)
			}
		}
	}
	if err := writeReport(public, rows); err != nil {
		return err
	}
	return writeResources(private, public)
}

func writeReport(public string, rows []record) error {
	return writeJSON(filepath.Join(public, "metrics.json"), struct {
		Schema string   `json:"schema"`
		Scope  string   `json:"scope"`
		Rows   []record `json:"rows"`
	}{"gooo/source-assembly-authoring-observation/v1",
		"Two authored finite tasks; four sequential requests per activity/form/mode; unchanged own weights; " +
			"first request builds, later requests reuse; response excludes initial model/input loading and final output; " +
			"raw native observations are private, published generation/metrics use public source only; no host CPU observation", rows})
}

// This reproducer extracts cases only from the fixed fixture, not arbitrary Gooo.
// Its plan is the real compiler's validated context export.
func fixtureDocument(compiler string, source []byte, filename, activity string) ([]byte, error) {
	start := strings.Index(string(source), "activity "+activity+"(")
	if start < 0 {
		return nil, fmt.Errorf("missing fixture activity")
	}
	block := assemblyBlock.Find(source[start:])
	var cases []finiteCase
	for _, match := range caseRow.FindAllStringSubmatch(string(block), -1) {
		input, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, err
		}
		expected, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return nil, err
		}
		cases = append(cases, finiteCase{input, expected})
	}
	attempts := attemptRow.FindStringSubmatch(string(block))
	if len(attempts) != 2 || len(cases) == 0 {
		return nil, fmt.Errorf("incomplete fixed fixture")
	}
	budget, err := strconv.Atoi(attempts[1])
	if err != nil {
		return nil, err
	}
	raw, err := exec.Command(compiler, "body-context", "--include-plan", "--activity", activity, filename).Output()
	if err != nil {
		return nil, err
	}
	var exported struct {
		Plan json.RawMessage `json:"expanded_plan"`
	}
	if err := json.Unmarshal(raw, &exported); err != nil || len(exported.Plan) == 0 {
		return nil, fmt.Errorf("missing validated compiler plan: %v", err)
	}
	return json.Marshal(struct {
		Schema   string          `json:"schema"`
		Plan     json.RawMessage `json:"path_plan"`
		Cases    []finiteCase    `json:"test_cases"`
		Attempts int             `json:"max_attempts"`
	}{"gooo/body-codegen-typed-path-plan/v1", exported.Plan, cases, budget})
}

func readRows(dir, activity, form, mode, public string) ([]record, error) {
	var rows []record
	for i := 1; i <= 4; i++ {
		stem := fmt.Sprintf("run-%d", i)
		raw, err := os.ReadFile(filepath.Join(dir, stem+"-response.json"))
		if err != nil {
			return nil, err
		}
		var r struct {
			Status   string `json:"status"`
			Response struct {
				Report struct {
					GeneratedSHA string `json:"generated_digest"`
					Paths        struct {
						Declared int     `json:"declared_test_cases"`
						Percent  float64 `json:"finite_functional_completeness_percent"`
						Search   struct {
							Evaluated int `json:"evaluated_candidates"`
							Attempts  []struct {
								Passed int `json:"training_cases_passed"`
								Total  int `json:"training_cases_total"`
							} `json:"attempts"`
							Selection struct {
								ModelCalls int    `json:"local_model_predictions"`
								Metadata   string `json:"model_metadata_sha256"`
								Weights    string `json:"model_weights_sha256"`
								Prediction struct {
									InputSHA  string `json:"input_sha256"`
									PredictNS int64  `json:"predict_ns"`
								} `json:"three_choice_prediction"`
							} `json:"selection"`
						} `json:"search"`
					} `json:"body_paths"`
				} `json:"report"`
			} `json:"response"`
			Execution struct {
				Observation struct {
					Replayed bool                     `json:"runtime_replayed"`
					Declared int                      `json:"declared_cases"`
					Cases    []struct{ Passed bool }  `json:"cases"`
					Runs     []struct{ Started bool } `json:"runs"`
					Artifact struct{ Reused bool }    `json:"artifact"`
				} `json:"observation"`
			} `json:"execution"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		if r.Status != "completed" || !r.Execution.Observation.Replayed || r.Response.Report.Paths.Percent != 100 {
			return nil, fmt.Errorf("incomplete %s %s %s request %d", activity, form, mode, i)
		}
		timingRaw, err := os.ReadFile(filepath.Join(dir, stem+"-timing.json"))
		if err != nil {
			return nil, err
		}
		var timing struct {
			ResponseNS int64 `json:"response_ns"`
			Wall       struct {
				Phases []struct {
					Name    string `json:"name"`
					StartNS int64  `json:"start_ns"`
					EndNS   int64  `json:"end_ns"`
				} `json:"phases"`
			} `json:"wall"`
		}
		if err := json.Unmarshal(timingRaw, &timing); err != nil {
			return nil, err
		}
		row := record{Activity: activity, Form: form, Mode: mode, Sequence: i, ResponseMS: float64(timing.ResponseNS) / 1e6,
			ModelCalls: r.Response.Report.Paths.Search.Selection.ModelCalls, SelectionPassed: r.Response.Report.Paths.Declared,
			SelectionTotal: r.Response.Report.Paths.Declared, RuntimeTotal: r.Execution.Observation.Declared,
			ArtifactReused: r.Execution.Observation.Artifact.Reused, GeneratedSHA256: r.Response.Report.GeneratedSHA,
			ModelMetadataSHA: r.Response.Report.Paths.Search.Selection.Metadata, ModelWeightsSHA: r.Response.Report.Paths.Search.Selection.Weights,
			ModelInputSHA: r.Response.Report.Paths.Search.Selection.Prediction.InputSHA,
			PredictNS:     r.Response.Report.Paths.Search.Selection.Prediction.PredictNS, Evaluated: r.Response.Report.Paths.Search.Evaluated}
		if len(r.Response.Report.Paths.Search.Attempts) > 0 {
			first := r.Response.Report.Paths.Search.Attempts[0]
			row.FirstPassed, row.FirstTotal = first.Passed, first.Total
		}
		for _, c := range r.Execution.Observation.Cases {
			if c.Passed {
				row.RuntimePassed++
			}
		}
		for _, run := range r.Execution.Observation.Runs {
			if run.Started {
				row.NativeRuns++
			}
		}
		for _, phase := range timing.Wall.Phases {
			if phase.Name == "generation" {
				row.GenerationMS += float64(phase.EndNS-phase.StartNS) / 1e6
			}
		}
		if row.RuntimePassed != row.RuntimeTotal || row.NativeRuns != 2 || row.GenerationMS <= 0 ||
			(mode == "model" && (row.ModelCalls < 1 || row.ModelWeightsSHA == "")) ||
			(mode == "deterministic" && row.ModelCalls != 0) {
			return nil, fmt.Errorf("finite native expectations not satisfied")
		}
		rows = append(rows, row)
		generation, err := os.ReadFile(filepath.Join(dir, stem+"-generation.json"))
		if err != nil {
			return nil, err
		}
		if bytes.Contains(generation, []byte("/Users/")) || bytes.Contains(generation, []byte("/private/")) {
			return nil, fmt.Errorf("generation contains a local path")
		}
		name := fmt.Sprintf("%s-%s-%s-%d-generation.json", activity, form, mode, i)
		if err := os.WriteFile(filepath.Join(public, name), generation, 0644); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
