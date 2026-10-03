// Call the new judge inside Gooo codegen and immediately execute each result.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/orderjudge"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const expectedWeights = "cf00ccc83d17d28ed73fcb869366151a48ffccd3aa8ca8e635aabf19810b9e78"
const expectedDataset = "b724470bc624c5030d8456b1a79da501a05f3da3545af5b8c221864ac47e9806"
const expectedRuntime = "bd953c547cfb18f5e6db0a3d96b15d66eefb765cf94a738819223b1e48534ece"

type task struct {
	ID, Group, Family, Language string
	WantedOrder, SourceOrder    int
	NewConstants                bool
}
type original struct {
	Task                       task `json:"task"`
	SourceSHA, PlanSHA, Intent string
	Descriptors                [8]string
	Inputs, Expected           [8]int64
}
type outcome struct{ Budget, Attempts, SelectedMask, Passed, Total int }
type prior struct {
	ID, Group, Arm string
	Budget         int
	Deduplicate    bool
	Outcome        outcome
	Ranking        *orderjudge.SearchReceipt
}
type record struct {
	ID, Request, Group, Language, Arm                         string
	Budget                                                    int
	Outcome                                                   outcome
	Calls, NativeRuns, Aliases                                int
	PredictionNS                                              int64
	CodegenMS                                                 float64
	SourceSHA, PlanSHA, EmittedSHA, GenerationSHA, RuntimeSHA string
	Generation, Execution                                     cost
}
type generation struct {
	Source string
	Report struct {
		Compiler string `json:"compiler_source_sha"`
		Paths    struct {
			SourceSHA   string `json:"original_source_sha256"`
			SourceBound bool   `json:"source_base_matched"`
			Search      pathplan.SearchResult
			Ranking     *orderjudge.SearchReceipt `json:"whole_candidate_judgment"`
			Timing      struct {
				TotalMS float64 `json:"total_ms"`
			}
		} `json:"body_paths"`
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func read(path string) []byte     { b, err := os.ReadFile(path); must(err); return b }
func hash(b []byte) string        { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func decode(path string, out any) { must(json.Unmarshal(read(path), out)) }
func save(path string, value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	must(err)
	must(os.WriteFile(path, append(b, '\n'), 0644))
}

func expected(t task, x int64) int64 {
	a, b := int64(1), int64(2)
	switch t.Family {
	case "add-multiply":
		if t.NewConstants {
			a, b = 2, 3
		}
		if t.WantedOrder == 0 {
			return (x + a) * b
		}
		return x*b + a
	case "subtract-multiply":
		a = 3
		if t.NewConstants {
			a, b = 1, 4
		}
		if t.WantedOrder == 0 {
			return (x - a) * b
		}
		return x*b - a
	case "negate-add":
		b = 4
		if t.NewConstants {
			b = 2
		}
		if t.WantedOrder == 0 {
			return -x + b
		}
		return -(x + b)
	case "square-add":
		b = 1
		if t.NewConstants {
			b = 3
		}
		if t.WantedOrder == 0 {
			return x*x + b
		}
		return (x + b) * (x + b)
	}
	panic("unknown arithmetic family")
}

func observe(compiler, revision, goBin, model, out string, row original, p prior, arm string, budget int) record {
	id := fmt.Sprintf("%s-%s-b%d", row.Task.ID, arm, budget)
	source := row.Task.ID + ".gooo"
	recipe := fmt.Sprintf("%s-b%d-recipe.json", row.Task.ID, budget)
	gen, executed := id+"-generation.json", id+"-runtime.json"
	args := []string{"body-codegen", "--json", "--activity", "Compose", "--path-plan", recipe}
	if arm == "model" {
		args = append(args, "--path-model", model)
	}
	r := record{ID: id, Request: row.Task.ID, Group: row.Task.Group, Language: row.Task.Language, Arm: arm, Budget: budget, SourceSHA: row.SourceSHA, PlanSHA: row.PlanSHA}
	r.Generation = execute(compiler, out, gen, append(args, source)...)
	var g generation
	decode(filepath.Join(out, gen), &g)
	s := g.Report.Paths.Search
	require(g.Report.Compiler == revision && g.Source != "" && g.Report.Paths.SourceBound && g.Report.Paths.SourceSHA == row.SourceSHA && s.Selection.PlanSHA256 == row.PlanSHA, id+": source or compiler binding differs")
	r.Calls, r.CodegenMS = s.Selection.ModelCalls, g.Report.Paths.Timing.TotalMS
	if arm == "model" {
		ranking := g.Report.Paths.Ranking
		require(r.Calls == 1 && ranking != nil && ranking.Deduplicate && ranking.Descriptors == row.Descriptors && ranking.Prediction == p.Ranking.Prediction, id+": direct model ranking differs")
		require(s.Selection.WeightsSHA256 == expectedWeights && ranking.IntentSHA256 == hash([]byte(row.Intent)), id+": model or complete intent differs")
		r.PredictionNS, r.Aliases = ranking.PredictNS, len(ranking.Aliases)
	} else {
		require(r.Calls == 0 && g.Report.Paths.Ranking == nil, id+": disconnected arm predicted")
	}
	require(s.Selection.ExternalCallsKnown && s.Selection.ExternalCalls == 0, id+": unexpected external calls")
	mask := selectedMask(s.Selection.Choices)
	r.Outcome = outcome{Budget: budget, Attempts: s.Evaluated, SelectedMask: mask}
	// Nothing is generated for the next row until this exact result has built/run.
	r.Execution = execute(compiler, out, executed, "body-execute", "--source", source, "--path-plan", recipe,
		"--generation", gen, "--cases", row.Task.ID+"-cases.json", "--go-bin", goBin)
	var native struct {
		Observation struct {
			Stage string
			Runs  []json.RawMessage
			Cases []struct {
				Input, Expected, Actual int64
				Passed                  bool
			}
		}
	}
	decode(filepath.Join(out, executed), &native)
	require(native.Observation.Stage == "COMPLETE" && len(native.Observation.Runs) == 2 && len(native.Observation.Cases) == 8, id+": native run incomplete")
	for i, c := range native.Observation.Cases {
		require(c.Input == row.Inputs[i] && c.Expected == expected(row.Task, c.Input) && c.Passed == (c.Actual == c.Expected), id+": native case/reference mismatch")
		r.Outcome.Total++
		if c.Passed {
			r.Outcome.Passed++
		}
	}
	require(r.Outcome == p.Outcome, id+": native result differs from frozen typed search")
	r.NativeRuns = len(native.Observation.Runs)
	r.EmittedSHA, r.GenerationSHA, r.RuntimeSHA = hash([]byte(g.Source)), hash(read(filepath.Join(out, gen))), hash(read(filepath.Join(out, executed)))
	return r
}

func selectedMask(choices map[string]string) int {
	mask := 0
	for bit, id := range []string{"order", "first-operands", "second-operands"} {
		label, ok := choices[id]
		require(ok, "missing selected choice")
		if label == "schedule_reverse" || label == "layout_reverse" {
			mask |= 1 << bit
		}
	}
	return mask
}

func main() {
	compiler := flag.String("compiler", "", "clean compiler binary")
	revision := flag.String("compiler-sha", "", "exact compiler revision")
	collector := flag.String("source-revision", "", "clean collector revision")
	goBin := flag.String("go-bin", "go", "Go1.27.1")
	initial := flag.String("initial", "", "decoded original fit evidence")
	runtimeDir := flag.String("runtime", "", "decoded original actual search replay")
	model := flag.String("model", "", "public original model.json")
	out := flag.String("out", "", "fresh output")
	flag.Parse()
	require(len(*revision) == 40 && len(*collector) == 40 && *compiler != "" && *initial != "" && *runtimeDir != "" && *model != "" && *out != "" && flag.NArg() == 0, "pinned inputs required")
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	must(err)
	require(strings.TrimSpace(string(head)) == *collector && len(dirty) == 0 && runtime.Version() == "go1.27.1", "clean pinned collector required")
	build, err := exec.Command(*goBin, "version", "-m", *compiler).Output()
	must(err)
	require(bytes.Contains(build, []byte("vcs.revision="+*revision)) && bytes.Contains(build, []byte("vcs.modified=false")) && bytes.Contains(build, []byte("v0.2.19-experimental")), "compiler/SDK identity differs")
	require(hash(read(filepath.Join(*initial, "dataset.json"))) == expectedDataset, "frozen dataset digest differs")
	require(hash(read(filepath.Join(*runtimeDir, "records.json"))) == expectedRuntime, "frozen runtime evidence digest differs")
	require(hash(read(filepath.Join(filepath.Dir(*model), "weights.bin"))) == expectedWeights, "fixed weights differ")
	_, err = orderjudge.Load(read(*model), read(filepath.Join(filepath.Dir(*model), "weights.bin")))
	must(err)
	*model, err = filepath.Abs(*model)
	must(err)
	_, err = os.Stat(*out)
	require(os.IsNotExist(err), "output must be fresh")
	must(os.MkdirAll(*out, 0755))
	var originals []original
	decode(filepath.Join(*initial, "dataset.json"), &originals)
	var priors []prior
	decode(filepath.Join(*runtimeDir, "records.json"), &priors)
	require(len(originals) == 160 && len(priors) == 960, "original cohort count differs")
	expectedRows := map[string]prior{}
	for _, p := range priors {
		if p.Budget == 1 && !p.Deduplicate || p.Budget == 8 && p.Deduplicate == (p.Arm == "model") {
			expectedRows[fmt.Sprintf("%s-%s-b%d", p.ID, p.Arm, p.Budget)] = p
		}
	}
	var records []record
	journal, err := os.Create(filepath.Join(*out, "progress.jsonl"))
	must(err)
	defer journal.Close()
	enc := json.NewEncoder(journal)
	requests := 0
	for _, row := range originals {
		if row.Task.Group != "new-template" && row.Task.Group != "new-constants" {
			continue
		}
		require(filepath.Base(row.Task.ID) == row.Task.ID && row.Task.ID != "", "invalid request ID")
		base := filepath.Join(*initial, "sources", row.Task.ID)
		source := read(base + ".gooo")
		require("sha256:"+hash(source) == row.SourceSHA, "source differs")
		must(os.WriteFile(filepath.Join(*out, row.Task.ID+".gooo"), source, 0644))
		var recipe map[string]any
		decode(base+"-recipe.json", &recipe)
		var cases []pathplan.TestCase
		for i, x := range row.Inputs {
			require(row.Expected[i] == expected(row.Task, x), "authored reference differs")
			cases = append(cases, pathplan.TestCase{Input: x, Expected: row.Expected[i]})
		}
		save(filepath.Join(*out, row.Task.ID+"-cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": cases})
		budgets, arms := []int{1, 8}, []string{"deterministic", "model"}
		if requests%2 == 1 {
			budgets = []int{8, 1}
			arms = []string{"model", "deterministic"}
		}
		for _, budget := range budgets {
			recipe["max_attempts"] = budget
			save(filepath.Join(*out, fmt.Sprintf("%s-b%d-recipe.json", row.Task.ID, budget)), recipe)
			for _, arm := range arms {
				id := fmt.Sprintf("%s-%s-b%d", row.Task.ID, arm, budget)
				prior, ok := expectedRows[id]
				require(ok, "frozen search missing")
				r := observe(*compiler, *revision, *goBin, *model, *out, row, prior, arm, budget)
				records = append(records, r)
				must(enc.Encode(r))
				must(journal.Sync())
				if len(records)%16 == 0 {
					fmt.Printf("completed generations=%d compiled_runs=%d\n", len(records), len(records)*2)
				}
			}
		}
		requests++
	}
	require(requests == 64 && len(records) == 256, "native cohort incomplete")
	save(filepath.Join(*out, "records.json"), records)
	summary := map[string]map[string]int{}
	actualCalls, actualRuns, passed, total := 0, 0, 0, 0
	for _, r := range records {
		actualCalls += r.Calls
		actualRuns += r.NativeRuns
		passed += r.Outcome.Passed
		total += r.Outcome.Total
		key := fmt.Sprintf("%s/%s/budget%d", r.Group, r.Arm, r.Budget)
		if summary[key] == nil {
			summary[key] = map[string]int{}
		}
		s := summary[key]
		s["requests"]++
		s["model_calls"] += r.Calls
		s["attempts"] += r.Outcome.Attempts
		s["aliases"] += r.Aliases
		s["passed"] += r.Outcome.Passed
		s["total"] += r.Outcome.Total
		s["native_runs"] += r.NativeRuns
		if r.Outcome.Passed == r.Outcome.Total {
			s["complete"]++
		}
	}
	save(filepath.Join(*out, "summary.json"), summary)
	save(filepath.Join(*out, "manifest.json"), map[string]any{"schema": "gooo/order-judge-native/v1", "compiler_sha": *revision, "compiler_binary_sha256": hash(read(*compiler)), "collector_sha": *collector,
		"sdk": "v0.2.19-experimental", "model_weights_sha256": expectedWeights, "dataset_sha256": expectedDataset, "protocol_sha256": hash(read("docs/order-judge-native-protocol-20261003.md")),
		"runtime_evidence_sha256": expectedRuntime, "requests": requests, "generations": len(records), "native_runs": actualRuns,
		"model_predictions": actualCalls, "passed": passed, "total": total, "training_updates": 0, "os": runtime.GOOS, "arch": runtime.GOARCH,
		"scope": "Direct in-compiler whole-candidate inference, immediate build and two executions per generated result. Previously observed development cohort; all budget-one functional failures retained."})
	fmt.Println("native cohort complete")
}
