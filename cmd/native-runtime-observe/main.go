// Native-runtime-observe measures actual model selection followed by the
// compiler's independent native execution/reverse-observation producer.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
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
		return 0, fmt.Errorf("capture exceeds limit")
	}
	return c.data.Write(raw)
}

type processCost struct {
	WallNS   int64 `json:"wall_ns"`
	UserNS   int64 `json:"user_ns"`
	SystemNS int64 `json:"system_ns"`
}

func child(dir, binary string, args ...string) ([]byte, processCost, error) {
	return childBound(dir, binary, 8<<20, args...)
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
	if cmd.ProcessState != nil {
		cost.UserNS = cmd.ProcessState.UserTime().Nanoseconds()
		cost.SystemNS = cmd.ProcessState.SystemTime().Nanoseconds()
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
	if len(os.Args) != 5 && (len(os.Args) != 7 || (os.Args[5] != "--shared-models" && os.Args[5] != "--compact-models")) {
		panic("usage: native-runtime-observe output-directory compiler go-binary compiler-source-sha [--shared-models training-directory | --compact-models compact-study-directory]")
	}
	output, compiler, goBinary, revision := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	info, err := buildinfo.ReadFile(compiler)
	must(err)
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if info.GoVersion != "go1.27.1" || settings["vcs.revision"] != revision || settings["vcs.modified"] != "false" {
		panic("clean exact compiler required")
	}
	views, err := threecohort.Load("runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl")
	must(err)
	model, err := filepath.Abs("models/own-three-feedback-v1/set-feedback/models/fp32/model.json")
	must(err)
	modelBytes, err := os.ReadFile(model)
	must(err)
	policies := []runtimePolicy{{"model", model}, {"offline", ""}}
	modelPins := map[string]string{"model": sha(modelBytes)}
	shared := len(os.Args) == 7 && os.Args[5] == "--shared-models"
	var compact *compactStudy
	invoke := child
	if shared {
		policies, modelPins = sharedPolicies(os.Args[6])
		checkSharedStorage()
	}
	if len(os.Args) == 7 && os.Args[5] == "--compact-models" {
		compact = newCompactStudy(os.Args[6], output)
		policies, modelPins = compact.Policies, compact.MetadataPins
		invoke = func(dir, binary string, args ...string) ([]byte, processCost, error) {
			return childBound(dir, binary, 1<<20, args...)
		}
	}
	planned := 16 * len(policies)
	must(os.Mkdir(output, 0700))
	preexecution := map[string]any{"schema": "gooo/native-runtime-study/v1", "compiler_source_sha": revision,
		"dataset_sha256": threecohort.DatasetSHA, "configuration": 20, "goal": 4,
		"selection": "all eight families, Korean/English, all declared policies", "model_metadata_pins": modelPins,
		"shared_model_comparison": shared, "planned_generations": planned, "planned_native_runs": 2 * planned,
		"runtime_inputs_added": []int64{-257, -127, -31, -7, 7, 31, 127, 257}, "new_optimizer_updates": 0,
		"scope": "Frozen observed source cohort; extra runtime inputs are disjoint only from the current selection suite, not a claim of training holdout or accuracy improvement"}
	if compact != nil {
		preexecution["compact_comparison"] = compact.Preexecution()
	}
	if !shared && compact == nil {
		preexecution["model_metadata_sha256"] = sha(modelBytes)
	}
	save(filepath.Join(output, "preexecution.json"), preexecution)
	rows := []map[string]any{}
	predictions, totalPassed := 0, 0
	for _, v := range views {
		if v.Config != 20 || v.Goal != 4 {
			continue
		}
		plan, err := threecompositionstudy.Fixture(v.Family, v.Config, v.Goal, v.Language)
		must(err)
		prepared, err := pathplan.Prepare(plan)
		must(err)
		body, err := json.Marshal(prepared.Fallback().GoooBody())
		must(err)
		source := []byte("package threecomposition\nnamespace threecomposition\nentity Integer id \"threecomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
		if strings.TrimPrefix(sha(source), "sha256:") != v.SourceSHA {
			panic("frozen source differs")
		}
		cases := append([]pathplan.TestCase(nil), v.Cases...)
		for _, input := range []int64{-257, -127, -31, -7, 7, 31, 127, 257} {
			for _, c := range v.Cases {
				if c.Input == input {
					panic("new runtime input overlaps selection")
				}
			}
			expected, err := threecompositionstudy.Oracle(v.Family, v.Config, v.Goal, input)
			must(err)
			cases = append(cases, pathplan.TestCase{Input: input, Expected: expected})
		}
		for _, policy := range policies {
			mode := policy.Name
			if shared {
				checkSharedStorage()
			}
			if compact != nil {
				compact.CheckStorage(compactPairReserve)
			}
			name := v.Family + "-" + v.Language + "-" + mode
			dir := filepath.Join(output, name)
			must(os.Mkdir(dir, 0700))
			must(os.WriteFile(filepath.Join(dir, "input.gooo"), source, 0600))
			save(filepath.Join(dir, "plan.json"), map[string]any{"schema": "gooo/body-codegen-typed-path-plan/v1", "path_plan": plan, "test_cases": v.Cases, "max_attempts": 8})
			save(filepath.Join(dir, "runtime-cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": cases})
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ChoosePath", "input.gooo"}
			if policy.Model != "" {
				args = append(args, "--path-model", policy.Model, "--path-feedback-rounds", "7", "--path-feedback-unfixed")
			}
			raw, codegenCost, err := invoke(dir, compiler, args...)
			must(os.WriteFile(filepath.Join(dir, "generation.json"), raw, 0600))
			must(err)
			var native generation
			must(json.Unmarshal(raw, &native))
			selection := native.Report.Paths.Search.Selection
			if native.Report.Revision != revision || !selection.Known || selection.External != 0 || (policy.Model == "" && selection.Calls != 0) || (policy.Model != "" && selection.Calls == 0) {
				panic("generation/model boundary differs")
			}
			must(os.WriteFile(filepath.Join(dir, "generated.go"), []byte(native.Source), 0600))
			args = []string{"body-execute", "--source", "input.gooo", "--path-plan", "plan.json", "--generation", "generation.json", "--cases", "runtime-cases.json", "--go-bin", goBinary}
			observed, cost, err := invoke(dir, compiler, args...)
			must(os.WriteFile(filepath.Join(dir, "runtime.json"), observed, 0600))
			must(err)
			var result runtimeResult
			must(json.Unmarshal(observed, &result))
			if result.Observation.Stage != "COMPLETE" || result.Observation.Producer != revision || !bytes.Equal(result.Parent, native.Report.Receipt) || result.Observation.ParentSHA != sha(result.Parent) || len(result.Observation.Cases) != 24 {
				panic("native observation binding differs")
			}
			passed := 0
			for i, c := range result.Observation.Cases {
				if c.Input != cases[i].Input || c.Expected != cases[i].Expected || c.Passed != (c.Actual == cases[i].Expected) {
					panic("independent finite oracle mismatch")
				}
				if c.Passed {
					passed++
				}
			}
			if !shared && compact == nil && passed != 24 {
				panic("original finite runtime comparison differs")
			}
			accuracy := runtimeAxis(result, "runtime_finite_accuracy")
			wantStatus := "PROGRESS"
			if passed == 24 {
				wantStatus = "PASS"
			}
			if accuracy.Numerator != passed || accuracy.Denominator != 24 || accuracy.Status != wantStatus {
				panic("finite completeness arithmetic differs")
			}
			for _, id := range []string{"runtime_source_replay", "runtime_build", "execution_boundary", "runtime_deterministic_replay", "reverse_observation_coverage"} {
				if runtimeAxis(result, id).Status != "PASS" {
					panic("runtime axis incomplete: " + id)
				}
			}
			disjoint := runtimeAxis(result, "runtime_selection_disjointness")
			if disjoint.Numerator != 8 || disjoint.Denominator != 24 || result.Receipt.Aggregate != nil || result.Receipt.First.ID != "permission_boundary" {
				panic("scope or unresolved boundary changed")
			}
			row := map[string]any{"name": name, "mode": mode, "view_id": v.ID, "model_calls": selection.Calls, "codegen_cost": codegenCost, "runtime_parent_cost": cost, "generation_sha256": sha(raw), "runtime_sha256": sha(observed), "finite_passed": passed, "selection_disjoint": 8, "first_unresolved": result.Receipt.First.ID}
			save(filepath.Join(dir, "observation.json"), row)
			rows = append(rows, row)
			predictions += selection.Calls
			totalPassed += passed
			fmt.Printf("%s: %d/24 compiled outputs; 8 selection-disjoint inputs; %d model calls\n", name, passed, selection.Calls)
		}
		if compact != nil {
			compact.CompareView(output, v.Family, v.Language, v.ID)
			// Each bilingual pair gets both representation orders; every generation
			// is still immediately followed by its own native execution.
			for i := 0; i < len(policies); i += 2 {
				policies[i], policies[i+1] = policies[i+1], policies[i]
			}
		}
	}
	if len(rows) != planned {
		panic("cohort denominator differs")
	}
	save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/native-runtime-study-result/v1", "status": "PASS", "compiler_source_sha": revision, "actual_generations": planned, "actual_model_predictions": predictions, "runtime_model_predictions": 0, "external_provider_calls": 0, "compiled_program_runs": 2 * planned, "finite_expectations": 24 * planned, "finite_expectations_passed": totalPassed, "ordered_compiled_outputs": 48 * planned, "selection_disjoint_expectations": 8 * planned, "new_optimizer_updates": 0, "rows": rows})
	if compact != nil {
		compact.Finish(output)
	}
}
