// Observe source-derived operation signatures without altering a model ABI.
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

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/orderfacts"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const fixtures = "publication/native-order-20261003"

type family struct {
	id                 string
	original, commuted [2]string
}

var families = []family{
	{"add-multiply", [2]string{"value + 1", "value * 2"}, [2]string{"1 + value", "2 * value"}},
	{"subtract-multiply", [2]string{"value - 3", "value * 2"}, [2]string{"value - 3", "2 * value"}},
	{"negate-add", [2]string{"0 - value", "value + 4"}, [2]string{"0 - value", "4 + value"}},
	{"square-add", [2]string{"value * value", "value + 1"}, [2]string{"value * value", "1 + value"}},
}

type exported struct {
	Schema        string
	SourceSHA     string                    `json:"original_source_sha256"`
	SourceBinding struct{ Equivalent bool } `json:"source_binding"`
	Plan          pathplan.Plan             `json:"expanded_plan"`
	Context       struct {
		PlanSHA string `json:"original_plan_sha256"`
	}
	Inputs []struct {
		ID         string `json:"decision_id"`
		InputSHA   string `json:"input_sha256"`
		FeatureSHA string `json:"source_features_sha256"`
		Text       string
	}
	ModelPredictions int  `json:"model_predictions"`
	CandidateTests   int  `json:"candidate_tests"`
	SelectedEmission bool `json:"selected_emission"`
	RepositoryWrites int  `json:"repository_writes"`
}

type record struct {
	ID, Family, Language, Presentation                           string
	SourceOrder                                                  int
	SourceSHA, PlanSHA, LegacyRootInputSHA, LegacyRootFeatureSHA string
	Signatures                                                   [2]string
	ExportMS                                                     float64
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(path string) []byte { b, err := os.ReadFile(path); must(err); return b }
func hash(b []byte) string    { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func save(path string, value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	must(err)
	must(os.WriteFile(path, append(b, '\n'), 0644))
}

func source(f family, order int, presentation string) []byte {
	raw := string(read(filepath.Join(fixtures, f.id+".gooo")))
	old := "value = " + f.original[0] + "; value = " + f.original[1]
	rhs := f.original
	if presentation == "commuted" {
		rhs = f.commuted
	}
	if strings.Count(raw, old) != 1 {
		panic("authored fixture changed")
	}
	raw = strings.Replace(raw, old, "value = "+rhs[order]+"; value = "+rhs[1-order], 1)
	if presentation == "renamed" {
		raw = strings.ReplaceAll(raw, "value", "state")
	}
	return []byte(raw)
}

func observe(compiler, out string, f family, language string, order int, presentation string) record {
	id := fmt.Sprintf("%s-%s-o%d-%s", f.id, language, order, presentation)
	src := source(f, order, presentation)
	filename := filepath.Join(out, id+".gooo")
	recipe := read(filepath.Join(fixtures, f.id+"-"+language+"-o0-a1-recipe.json"))
	planfile := filepath.Join(out, id+"-recipe.json")
	must(os.WriteFile(filename, src, 0644))
	must(os.WriteFile(planfile, recipe, 0644))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, compiler, "body-context", "--include-plan", "--feature-version",
		"semantic_context_intent_v3", "--plan", planfile, "--activity", "Compose", filename)
	cmd.WaitDelay = 5 * time.Second
	var output, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &stderr
	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)
	must(os.WriteFile(filepath.Join(out, id+"-context.json"), output.Bytes(), 0644))
	if err != nil {
		panic(fmt.Sprintf("context: %v: %s", err, stderr.String()))
	}
	var e exported
	must(json.Unmarshal(output.Bytes(), &e))
	must(validateExport(e, src))
	facts, err := orderfacts.Alternatives(e.Plan, "order")
	must(err)
	r := record{ID: id, Family: f.id, Language: language, Presentation: presentation, SourceOrder: order,
		SourceSHA: e.SourceSHA, PlanSHA: e.Context.PlanSHA, ExportMS: float64(elapsed) / 1e6}
	for _, input := range e.Inputs {
		if input.ID == "order" {
			if "sha256:"+hash([]byte(input.Text)) != input.InputSHA {
				panic("legacy input hash differs")
			}
			r.LegacyRootInputSHA, r.LegacyRootFeatureSHA = input.InputSHA, input.FeatureSHA
		}
	}
	if len(r.LegacyRootInputSHA) != 71 || len(r.LegacyRootFeatureSHA) != 71 {
		panic("root input missing")
	}
	for i := range facts {
		r.Signatures[i] = hex.EncodeToString(facts[i][:])
	}
	return r
}

func validateExport(e exported, source []byte) error {
	encoded, err := json.Marshal(e.Plan)
	if err != nil {
		return err
	}
	// Compiler receipts prefix byte digests; SDK plan identities are bare hex.
	if e.Schema != "gooo/compiler-path-input-export/v2" || !e.SourceBinding.Equivalent || e.SourceSHA != "sha256:"+hash(source) ||
		e.Context.PlanSHA != hash(encoded) || e.ModelPredictions != 0 || e.CandidateTests != 0 || e.SelectedEmission || e.RepositoryWrites != 0 {
		return fmt.Errorf("context identity or observation-only contract differs")
	}
	return nil
}

func summary(rows []record) map[string]int {
	byID := make(map[string]record, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	counts := map[string]int{"exports": len(rows), "model_predictions": 0, "candidate_tests": 0, "training_updates": 0, "native_runs": 0}
	for _, r := range rows {
		if r.SourceOrder == 0 {
			x := byID[fmt.Sprintf("%s-%s-o1-%s", r.Family, r.Language, r.Presentation)]
			counts["source_reversal_pairs"]++
			if r.Signatures[0] == x.Signatures[1] && r.Signatures[1] == x.Signatures[0] && r.Signatures[0] != r.Signatures[1] {
				counts["source_reversal_distinguished"]++
			}
			if r.LegacyRootInputSHA == x.LegacyRootInputSHA {
				counts["legacy_root_input_aliases"]++
			}
		}
		if r.Presentation != "base" {
			x := byID[fmt.Sprintf("%s-%s-o%d-base", r.Family, r.Language, r.SourceOrder)]
			counts[r.Presentation+"_pairs"]++
			if r.Signatures == x.Signatures {
				counts[r.Presentation+"_invariant"]++
			}
		}
		if r.Language == "en" {
			x := byID[fmt.Sprintf("%s-ko-o%d-%s", r.Family, r.SourceOrder, r.Presentation)]
			counts["language_pairs"]++
			if r.Signatures == x.Signatures {
				counts["language_invariant"]++
			}
		}
	}
	return counts
}

func main() {
	compiler := flag.String("compiler", "", "clean compiler binary")
	revision := flag.String("compiler-sha", "", "exact compiler revision")
	collector := flag.String("collector-sha", "", "clean collector revision")
	goBin := flag.String("go", "go", "Go1.27.1 binary")
	out := flag.String("out", "", "new output directory")
	flag.Parse()
	if *compiler == "" || len(*revision) != 40 || len(*collector) != 40 || *out == "" || flag.NArg() != 0 {
		panic("exact identities required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	must(err)
	if strings.TrimSpace(string(head)) != *collector || len(dirty) != 0 || runtime.Version() != "go1.27.1" {
		panic("clean pinned collector required")
	}
	build, err := exec.Command(*goBin, "version", "-m", *compiler).Output()
	must(err)
	if !bytes.Contains(build, []byte("vcs.revision="+*revision)) || !bytes.Contains(build, []byte("vcs.modified=false")) {
		panic("compiler identity differs")
	}
	if _, err = os.Stat(*out); !os.IsNotExist(err) {
		panic("output already exists")
	}
	must(os.MkdirAll(*out, 0755))
	var rows []record
	for _, f := range families {
		for _, language := range []string{"en", "ko"} {
			for order := range 2 {
				for _, presentation := range []string{"base", "renamed", "commuted"} {
					rows = append(rows, observe(*compiler, *out, f, language, order, presentation))
				}
			}
		}
	}
	save(filepath.Join(*out, "records.json"), rows)
	save(filepath.Join(*out, "summary.json"), summary(rows))
	save(filepath.Join(*out, "manifest.json"), map[string]any{"schema": "gooo/source-order-preflight/v1", "compiler_sha": *revision,
		"compiler_binary_sha256": hash(read(*compiler)), "collector_sha": *collector, "feature_version": orderfacts.Version,
		"bytes_per_alternative": orderfacts.Bytes, "os": runtime.GOOS, "arch": runtime.GOARCH,
		"scope": "Compiler-bound source observation; fixed authored families; existing model ABI and weights unchanged; no accuracy or training result."})
}
