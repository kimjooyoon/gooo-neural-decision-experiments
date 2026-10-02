package main

import (
	"bytes"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

const fullProtocolSHA = "95cecea073effa783183803812ce5ae61f62f904beb1ca40ab01b741413431d6"
const fullManifestSHA = "ce4ad854c0d7fcb3e7fa049658f40ea4b0393a9bff39a2ef21f64d83f221bc3a"

var fullArms = [4]string{"positioned-original", "positioned-varied", "bag-original", "bag-varied"}

type fullBinding struct {
	compactBinding
	Feature    string `json:"feature_version"`
	Arithmetic string `json:"arithmetic_version"`
}

type fullStudy struct {
	Source                string                 `json:"collector_source_sha"`
	Compiler              string                 `json:"compiler_source_sha"`
	Bindings              map[string]fullBinding `json:"model_bindings"`
	Storage               fullStorage            `json:"storage"`
	ConsumerSHA           string                 `json:"consumer_sha256"`
	Stage                 string                 `json:"stage"`
	Current               string                 `json:"current_view_policy"`
	GenerationInvocations int                    `json:"generation_invocations"`
	CompletedGenerations  int                    `json:"completed_generations"`
	CompletedPredictions  int                    `json:"predictions_in_completed_generations"`
	ExecutionInvocations  int                    `json:"execution_invocations"`
	CompletedRuns         int                    `json:"completed_compiled_runs"`
	Expectations          int                    `json:"finite_expectations"`
	Passed                int                    `json:"finite_expectations_passed"`
	Rows                  []map[string]any       `json:"rows"`
	Pairs                 []map[string]any       `json:"pairs"`
	output                string
}

func newFullStudy(output, compiler, revision, bundle string) (*fullStudy, []runtimePolicy) {
	if runtime.Version() != "go1.27.1" {
		panic("Go 1.27.1 collector required")
	}
	info, err := buildinfo.ReadFile(compiler)
	must(err)
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if len(revision) != 40 || info.GoVersion != "go1.27.1" || settings["vcs.revision"] != revision || settings["vcs.modified"] != "false" {
		panic("clean exact compiler required")
	}
	sdk := false
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = d.Version == "v0.2.15-experimental" && d.Replace == nil
		}
	}
	if !sdk {
		panic("released SDK v0.2.15 required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	must(err)
	if len(dirty) != 0 {
		panic("clean collector required")
	}
	protocol := boundedFullFile("docs/full-input-sdk-native-protocol-20261003.md", 64<<10)
	manifest := boundedFullFile(filepath.Join(bundle, "manifest.json"), 256<<10)
	if sha(protocol) != "sha256:"+fullProtocolSHA || sha(manifest) != "sha256:"+fullManifestSHA {
		panic("registered protocol/reference changed")
	}
	var inventory struct {
		Files map[string]struct {
			SHA   string `json:"sha256"`
			Bytes int64  `json:"bytes"`
		} `json:"files"`
	}
	must(json.Unmarshal(manifest, &inventory))
	s := &fullStudy{Source: strings.TrimSpace(string(head)), Compiler: revision, Bindings: map[string]fullBinding{}, Stage: "preflight", output: output}
	s.ConsumerSHA = sha(boundedFullFile("tools/consume-full-input-native/main.go.txt", 64<<10))
	var policies []runtimePolicy
	for _, arm := range fullArms {
		for _, variant := range compactVariants {
			for _, layout := range []string{"expanded", "compact"} {
				relative := filepath.Join("models", layout, arm, variant, "model.json")
				path, err := filepath.Abs(filepath.Join(bundle, relative))
				must(err)
				for _, file := range []string{relative, filepath.Join(filepath.Dir(relative), "weights.bin")} {
					pin, ok := inventory.Files[filepath.ToSlash(file)]
					if !ok || pin.Bytes <= 0 || pin.Bytes > 128<<10 {
						panic("bounded pinned model required")
					}
					b := boundedFullFile(filepath.Join(bundle, file), 128<<10)
					if int64(len(b)) != pin.Bytes || sha(b) != "sha256:"+pin.SHA {
						panic("model file differs from public manifest")
					}
				}
				feature := jointdecision.ThreeFeatureVersion
				loader := jointdecision.LoadThree
				if strings.HasPrefix(arm, "bag-") {
					feature, loader = jointdecision.ThreeBagFeatureVersion, jointdecision.LoadThreeBag
				}
				if layout == "compact" {
					loader = jointdecision.LoadSharedThree
				}
				model, err := loader(path)
				must(err)
				if model.FeatureVersion() != feature || model.ArithmeticVersion() != jointdecision.SeparateArithmeticVersion || model.Variant() != variant {
					panic("actual model contract differs")
				}
				name := arm + "-" + variant + "-" + layout
				policies = append(policies, runtimePolicy{name, path})
				s.Bindings[name] = fullBinding{compactBinding{model.MetadataSHA256(), model.WeightsSHA256(), model.Schema()}, feature, model.ArithmeticVersion()}
			}
		}
	}
	policies = append(policies, runtimePolicy{"offline", ""})
	s.Storage = inspectFullStorage(output, true)
	return s, policies
}

func validateFullGeneration(raw []byte, binding fullBinding) []byte {
	var detail struct {
		Report struct {
			Paths struct {
				ModelContext struct {
					Status     string `json:"status"`
					Feature    string `json:"feature_version"`
					Arithmetic string `json:"arithmetic_version"`
					Metadata   string `json:"model_metadata_sha256"`
				} `json:"model_context"`
			} `json:"body_paths"`
		} `json:"report"`
	}
	must(json.Unmarshal(raw, &detail))
	c := detail.Report.Paths.ModelContext
	if c.Status != "ENCODED" || c.Feature != binding.Feature || c.Arithmetic != binding.Arithmetic || strings.TrimPrefix(c.Metadata, "sha256:") != binding.Metadata {
		panic("compiler context model identity differs")
	}
	var g pairedGeneration
	must(json.Unmarshal(raw, &g))
	check := func(r *pathplan.ThreeReceipt) {
		if r == nil || r.Feature != binding.Feature || r.Arithmetic != binding.Arithmetic {
			panic("native prediction identity differs")
		}
	}
	check(g.Report.Paths.Search.Selection.Three)
	for _, f := range g.Report.Paths.Feedback {
		if f.Three != nil {
			check(f.Three)
		}
	}
	return normalizeGeneration(raw, binding.compactBinding)
}

func (s *fullStudy) checkpoint(stage string) {
	s.Stage = stage
	save(filepath.Join(s.output, "progress.json"), s)
}

func runFullInput(output, compiler, goBinary, revision, bundle string) {
	if !filepath.IsAbs(goBinary) {
		panic("absolute Go 1.27.1 executable required")
	}
	version, _, err := childBound("", goBinary, 4096, "version")
	must(err)
	if !strings.HasPrefix(string(version), "go version go1.27.1 ") {
		panic("native child toolchain differs from Go 1.27.1")
	}
	s, policies := newFullStudy(output, compiler, revision, bundle)
	must(os.Mkdir(output, 0700))
	started := time.Now()
	defer func() {
		if problem := recover(); problem != nil {
			// Original generation/runtime bytes and the last successful prefix stay intact.
			save(filepath.Join(output, "failure.json"), map[string]any{"status": "FAIL", "phase": s.Stage, "error": fmt.Sprint(problem), "observed_prefix": s, "failing_invocation_totals": "unknown beyond retained child output", "automatic_retries": 0, "optimizer_updates": 0})
			panic(problem)
		}
	}()
	save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/full-input-native-preexecution/v1", "compiler_source_sha": revision, "collector_source_sha": s.Source, "consumer_sha256": s.ConsumerSHA, "protocol_sha256": fullProtocolSHA, "reference_manifest_sha256": fullManifestSHA, "dataset_sha256": threecohort.DatasetSHA, "sdk": "v0.2.15-experimental", "models": s.Bindings, "storage": s.Storage, "planned_generations": 400, "planned_compiled_runs": 800, "planned_finite_expectations": 9600, "optimizer_updates": 0, "seed": "", "ci_hint": false, "ordering": "arm/variant expanded then compact, pair order reversed after each view, offline last; each generation immediately builds and executes twice"})
	views, err := threecohort.Load("runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl")
	must(err)
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
					panic("extra runtime input overlaps selection")
				}
			}
			expected, err := threecompositionstudy.Oracle(v.Family, v.Config, v.Goal, input)
			must(err)
			cases = append(cases, pathplan.TestCase{Input: input, Expected: expected})
		}
		for _, policy := range policies {
			s.Storage = inspectFullStorage(output, false)
			s.Current = v.Family + "-" + v.Language + "-" + policy.Name
			dir := filepath.Join(output, s.Current)
			must(os.Mkdir(dir, 0700))
			must(os.WriteFile(filepath.Join(dir, "input.gooo"), source, 0600))
			save(filepath.Join(dir, "plan.json"), map[string]any{"schema": "gooo/body-codegen-typed-path-plan/v1", "path_plan": plan, "test_cases": v.Cases, "max_attempts": 8})
			save(filepath.Join(dir, "runtime-cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": cases})
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ChoosePath", "input.gooo"}
			if policy.Model != "" {
				args = append(args, "--path-model", policy.Model, "--path-feedback-rounds", "7", "--path-feedback-unfixed")
			}
			s.GenerationInvocations++
			s.checkpoint("generation")
			raw, cost, err := childBound(dir, compiler, 1<<20, args...)
			must(os.WriteFile(filepath.Join(dir, "generation.json"), raw, 0600))
			save(filepath.Join(dir, "generation-process.json"), cost)
			must(err)
			var g generation
			must(json.Unmarshal(raw, &g))
			selection := g.Report.Paths.Search.Selection
			if g.Report.Revision != revision || !selection.Known || selection.External != 0 {
				panic("generation boundary differs")
			}
			if policy.Model == "" {
				if selection.Calls != 0 {
					panic("offline model call")
				}
			} else {
				validateFullGeneration(raw, s.Bindings[policy.Name])
			}
			s.CompletedGenerations++
			s.CompletedPredictions += selection.Calls
			must(os.WriteFile(filepath.Join(dir, "generated.go"), []byte(g.Source), 0600))
			s.ExecutionInvocations++
			s.checkpoint("native_execution")
			observed, nativeCost, err := childBound(dir, compiler, 1<<20, "body-execute", "--source", "input.gooo", "--path-plan", "plan.json", "--generation", "generation.json", "--cases", "runtime-cases.json", "--go-bin", goBinary)
			must(os.WriteFile(filepath.Join(dir, "runtime.json"), observed, 0600))
			save(filepath.Join(dir, "runtime-process.json"), nativeCost)
			must(err)
			passed := validateFullRuntime(observed, g.Report.Receipt, cases, revision)
			s.CompletedRuns += 2
			s.Expectations += len(cases)
			s.Passed += passed
			row := map[string]any{"name": s.Current, "mode": policy.Name, "view_id": v.ID, "model_calls": selection.Calls, "codegen_cost": cost, "runtime_parent_cost": nativeCost, "finite_passed": passed, "finite_cases": len(cases), "generation_sha256": sha(raw), "runtime_sha256": sha(observed), "selection_disjoint": 8, "first_unresolved": "permission_boundary"}
			save(filepath.Join(dir, "observation.json"), row)
			s.Rows = append(s.Rows, row)
			s.checkpoint("completed_pair")
			fmt.Printf("%s: %d/24 outputs, %d model calls\n", s.Current, passed, selection.Calls)
		}
		s.compareView(v.Family, v.Language, v.ID)
		for i := 0; i < len(policies)-1; i += 2 {
			policies[i], policies[i+1] = policies[i+1], policies[i]
		}
	}
	if s.CompletedGenerations != 400 || s.CompletedRuns != 800 || s.Expectations != 9600 || len(s.Pairs) != 192 {
		panic("full cohort denominator differs")
	}
	s.Storage = inspectFullStorage(output, false)
	s.checkpoint("complete")
	save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/full-input-native-result/v1", "status": "PASS", "observations": s, "wall_ns": time.Since(started).Nanoseconds(), "actual_model_predictions": s.CompletedPredictions, "ordered_compiled_outputs": 2 * s.Expectations, "optimizer_updates": 0, "default_model_promoted": false, "scope": "known authored cohort, unseeded finite behavior and representation comparison; runtime and selection-case denominators remain separate"})
}

func (s *fullStudy) compareView(family, language, id string) {
	for _, arm := range fullArms {
		for _, variant := range compactVariants {
			var generations, cases [2][]byte
			for i, layout := range []string{"expanded", "compact"} {
				name := arm + "-" + variant + "-" + layout
				dir := filepath.Join(s.output, family+"-"+language+"-"+name)
				generations[i] = validateFullGeneration(boundedFullFile(filepath.Join(dir, "generation.json"), 1<<20), s.Bindings[name])
				var r runtimeResult
				must(json.Unmarshal(boundedFullFile(filepath.Join(dir, "runtime.json"), 1<<20), &r))
				var err error
				cases[i], err = json.Marshal(r.Observation.Cases)
				must(err)
			}
			if !bytes.Equal(generations[0], generations[1]) || !bytes.Equal(cases[0], cases[1]) {
				panic("expanded/compact native semantics differ")
			}
			s.Pairs = append(s.Pairs, map[string]any{"view_id": id, "arm": arm, "variant": variant, "semantic_generation_sha256": sha(generations[0]), "ordered_native_cases_sha256": sha(cases[0])})
		}
	}
	s.checkpoint("view_compared")
}
