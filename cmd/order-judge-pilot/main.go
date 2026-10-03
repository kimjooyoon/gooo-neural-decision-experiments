// Train and observe a small source-conditioned candidate scorer in Go.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/orderfacts"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/orderjudge"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type exported struct {
	SourceSHA string                    `json:"original_source_sha256"`
	Binding   struct{ Equivalent bool } `json:"source_binding"`
	Plan      pathplan.Plan             `json:"expanded_plan"`
	Context   struct {
		PlanSHA string `json:"original_plan_sha256"`
	}
	Calls int `json:"model_predictions"`
	Tests int `json:"candidate_tests"`
}
type row struct {
	Task                       task `json:"task"`
	SourceSHA, PlanSHA, Intent string
	Descriptors                [8]string
	Sample                     orderjudge.Sample
	Inputs                     [8]int64
	Expected                   [8]int64
	Observed                   [8][8]int64
	Deterministic              [2]outcome // Actual SDK searches with budgets 1 and 8.
}
type outcome struct {
	Budget, Attempts, SelectedMask, Passed, Total int
}
type observed struct {
	ID, Group, Language  string
	Prediction           orderjudge.Prediction
	PredictNS            int64
	Model, Deterministic [2]outcome
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func hash(b []byte) string    { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func read(path string) []byte { b, err := os.ReadFile(path); must(err); return b }
func save(path string, value any) []byte {
	raw, err := json.MarshalIndent(value, "", "  ")
	must(err)
	raw = append(raw, '\n')
	must(os.WriteFile(path, raw, 0644))
	return raw
}

func collect(compiler, out string, t task) row {
	base := filepath.Join(out, t.ID)
	source := []byte(t.source())
	must(os.WriteFile(base+".gooo", source, 0644))
	save(base+"-recipe.json", t.recipe())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, compiler, "body-context", "--include-plan", "--feature-version", "semantic_context_intent_v3", "--activity", "Compose", "--plan", base+"-recipe.json", base+".gooo")
	cmd.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	must(os.WriteFile(base+"-context.json", stdout.Bytes(), 0644))
	if err != nil {
		panic(fmt.Sprintf("context export: %v: %s", err, stderr.String()))
	}
	var e exported
	must(json.Unmarshal(stdout.Bytes(), &e))
	planBytes, err := json.Marshal(e.Plan)
	must(err)
	if !e.Binding.Equivalent || e.SourceSHA != "sha256:"+hash(source) || e.Context.PlanSHA != hash(planBytes) || e.Calls != 0 || e.Tests != 0 {
		panic("source-bound export differs")
	}
	r := row{Task: t, SourceSHA: e.SourceSHA, PlanSHA: e.Context.PlanSHA, Inputs: [8]int64{-3, -2, -1, 0, 1, 2, 3, 4}}
	var intents []string
	for _, c := range e.Plan.Decisions {
		intents = append(intents, c.Intent)
	}
	r.Intent = strings.Join(intents, "\n")
	must(orderjudge.IntentFeatures(r.Intent, &r.Sample.Intent))
	signatures, err := orderfacts.Candidates(e.Plan)
	must(err)
	prepared, err := pathplan.Prepare(e.Plan)
	must(err)
	for i, x := range r.Inputs {
		r.Expected[i] = t.expected(x)
	}
	for mask := range 8 {
		r.Descriptors[mask] = hex.EncodeToString(signatures[mask][:])
		must(orderjudge.SourceFeatures(signatures[mask], &r.Sample.Candidates[mask]))
		choices := prepared.Defaults()
		for i, c := range e.Plan.Decisions {
			choices[c.ID] = c.Options[(mask>>i)&1].Label
		}
		program, err := prepared.Compile(choices)
		must(err)
		for i, x := range r.Inputs {
			value, err := program.Evaluate(x)
			must(err)
			r.Observed[mask][i] = value.Int
		}
		if selectionPassed(r, mask) == 3 {
			r.Sample.Acceptable |= 1 << mask
		}
	}
	if r.Sample.Acceptable == 0 {
		panic("authored reference has no finite candidate")
	}
	var cases []pathplan.TestCase
	for _, x := range []int64{-2, 0, 3} {
		cases = append(cases, pathplan.TestCase{Input: x, Expected: t.expected(x)})
	}
	for i, budget := range []int{1, 8} {
		result, _, err := prepared.Search(ctx, nil, cases, budget, "")
		must(err)
		mask := 0
		for bit, c := range e.Plan.Decisions {
			if result.Selection.Choices[c.ID] == c.Options[1].Label {
				mask |= 1 << bit
			}
		}
		r.Deterministic[i] = outcome{budget, result.Evaluated, mask, evaluationPassed(r, mask), 8}
	}
	return r
}

func selectionPassed(r row, mask int) int {
	n := 0
	for _, i := range []int{1, 3, 6} {
		if r.Observed[mask][i] == r.Expected[i] {
			n++
		}
	}
	return n
}
func evaluationPassed(r row, mask int) int {
	n := 0
	for i, x := range r.Observed[mask] {
		if x == r.Expected[i] {
			n++
		}
	}
	return n
}
func search(r row, ranking [8]uint8, budget int) outcome {
	best, bestPassed, attempts := 0, -1, 0
	for _, mask := range ranking[:budget] {
		attempts++
		passed := selectionPassed(r, int(mask))
		if passed > bestPassed {
			best, bestPassed = int(mask), passed
		}
		if passed == 3 {
			break
		}
	}
	return outcome{budget, attempts, best, evaluationPassed(r, best), 8}
}

func summarize(rows []observed) map[string]any {
	groups := make(map[string]any)
	for _, group := range []string{"train", "new-template", "presentation", "new-constants"} {
		n := 0
		var first, complete, firstCases, completeCases, attempts [2]int
		for _, r := range rows {
			if r.Group != group {
				continue
			}
			n++
			for arm, values := range [2][2]outcome{r.Deterministic, r.Model} {
				if values[0].Passed == 8 {
					first[arm]++
				}
				if values[1].Passed == 8 {
					complete[arm]++
				}
				firstCases[arm] += values[0].Passed
				completeCases[arm] += values[1].Passed
				attempts[arm] += values[1].Attempts
			}
		}
		groups[group] = map[string]any{"requests": n, "arm_order": []string{"deterministic_sdk", "new_model"}, "first_complete": first, "bounded_complete": complete, "first_cases": firstCases, "bounded_cases": completeCases, "bounded_attempts": attempts, "cases_per_request": 8}
	}
	return groups
}

func main() {
	compiler := flag.String("compiler", "", "clean source-export compiler")
	revision := flag.String("compiler-sha", "", "compiler revision")
	collector := flag.String("collector-sha", "", "clean pilot source revision")
	goBin := flag.String("go", "go", "Go1.27.1")
	out := flag.String("out", "", "fresh output directory")
	flag.Parse()
	if *compiler == "" || len(*revision) != 40 || len(*collector) != 40 || *out == "" || flag.NArg() != 0 {
		panic("pinned identities required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	must(err)
	if strings.TrimSpace(string(head)) != *collector || len(dirty) != 0 || runtime.Version() != "go1.27.1" {
		panic("clean Go1.27.1 collector required")
	}
	build, err := exec.Command(*goBin, "version", "-m", *compiler).Output()
	must(err)
	if !bytes.Contains(build, []byte("vcs.revision="+*revision)) || !bytes.Contains(build, []byte("vcs.modified=false")) {
		panic("compiler identity differs")
	}
	if _, err = os.Stat(*out); !os.IsNotExist(err) {
		panic("output exists")
	}
	must(os.MkdirAll(filepath.Join(*out, "sources"), 0755))
	started := time.Now()
	var data []row
	var training []orderjudge.Sample
	for _, t := range tasks() {
		r := collect(*compiler, filepath.Join(*out, "sources"), t)
		data = append(data, r)
		if t.Group == "train" {
			training = append(training, r.Sample)
		}
	}
	dataset := save(filepath.Join(*out, "dataset.json"), data)
	options := orderjudge.FitOptions{Epochs: 400, LearningRate: 2, L2: 0.0001}
	save(filepath.Join(*out, "pretraining.json"), map[string]any{"collector_sha": *collector, "compiler_sha": *revision, "compiler_binary_sha256": hash(read(*compiler)), "dataset_sha256": hash(dataset), "training_requests": len(training), "total_requests": len(data), "fit_options": options, "protocol_sha256": hash(read("docs/order-judge-protocol-20261003.md")), "feature_version": orderjudge.FeatureVersion})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	fitStart := time.Now()
	model, history, err := orderjudge.Fit(ctx, training, options)
	must(err)
	fitMS := float64(time.Since(fitStart)) / 1e6
	save(filepath.Join(*out, "loss.json"), history)
	metadata, weights, err := model.Marshal()
	must(err)
	must(os.WriteFile(filepath.Join(*out, "model.json"), metadata, 0644))
	must(os.WriteFile(filepath.Join(*out, "weights.bin"), weights, 0644))
	loaded, err := orderjudge.Load(metadata, weights)
	must(err)
	var observations []observed
	for _, r := range data {
		var work orderjudge.Workspace
		var p, check orderjudge.Prediction
		start := time.Now()
		must(loaded.Predict(&r.Sample.Intent, &r.Sample.Candidates, &work, &p))
		ns := time.Since(start).Nanoseconds()
		must(model.Predict(&r.Sample.Intent, &r.Sample.Candidates, &work, &check))
		if p != check {
			panic("published model differs from fit")
		}
		observations = append(observations, observed{r.Task.ID, r.Task.Group, r.Task.Language, p, ns, [2]outcome{search(r, p.Ranking, 1), search(r, p.Ranking, 8)}, r.Deterministic})
	}
	save(filepath.Join(*out, "observations.json"), observations)
	save(filepath.Join(*out, "summary.json"), summarize(observations))
	save(filepath.Join(*out, "manifest.json"), map[string]any{"schema": "gooo/order-judge-pilot/v1", "compiler_sha": *revision, "collector_sha": *collector, "dataset_sha256": hash(dataset), "weights_sha256": hash(weights), "parameters": orderjudge.ParameterCount, "weights_bytes": len(weights), "workspace_bytes": unsafe.Sizeof(orderjudge.Workspace{}), "fit_ms": fitMS, "total_ms": float64(time.Since(started)) / 1e6, "actual_postfit_predictions": 2 * len(data), "optimizer_updates": options.Epochs, "native_runs": 0, "os": runtime.GOOS, "arch": runtime.GOARCH, "scope": "Source-bound standalone learning and typed interpreter evaluation. No in-compiler invocation or native execution of the learned selections yet. No held-out tuning or general-language accuracy claim."})
}
