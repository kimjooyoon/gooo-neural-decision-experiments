// Observe frozen positioned/bag judges in actual Gooo order assembly.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

const modelRoot = "publication/full-input-separate-arithmetic-20261003/models/compact/"

var modelPaths = map[string]string{"positioned": modelRoot + "positioned-original/fp32/model.json", "bag": modelRoot + "bag-original/fp32/model.json"}

type cost struct {
	WallMS, CPUms, OneCoreCPUPercent float64
	PeakRSSBytes                     int64
}
type record struct {
	ID, Family, Language, Arm                                               string
	Order, Budget, ModelPredictions, Evaluations, Passed, Total             int
	SearchStatus, EmittedSHA, FeatureSHA, ModelMetadataSHA, ModelWeightsSHA string
	InitialMask, SelectedMask                                               uint16
	Probabilities                                                           [8]float32
	Generation, Execution                                                   cost
}
type generation struct {
	Source string
	Report struct {
		Compiler string `json:"compiler_source_sha"`
		Paths    struct {
			Search struct {
				Status      string
				Evaluations int `json:"evaluated_candidates"`
				Selection   struct {
					Calls    int    `json:"local_model_predictions"`
					Metadata string `json:"model_metadata_sha256"`
					Weights  string `json:"model_weights_sha256"`
					Choices  map[string]string
					Three    struct {
						Input         string
						Mask          uint16     `json:"initial_mask"`
						Probabilities [8]float32 `json:"full_mask_probabilities"`
					} `json:"three_choice_prediction"`
				} `json:"selection"`
			} `json:"search"`
		} `json:"body_paths"`
	} `json:"report"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(path string) []byte { b, e := os.ReadFile(path); must(e); return b }
func hash(b []byte) string    { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func save(path string, value any) {
	b, e := json.MarshalIndent(value, "", "  ")
	must(e)
	must(os.WriteFile(path, append(b, '\n'), 0644))
}
func execute(bin, output string, args ...string) cost {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, bin, args...)
	c.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	start := time.Now()
	err := c.Run()
	wall := time.Since(start)
	must(os.WriteFile(output, stdout.Bytes(), 0644))
	if err != nil {
		must(os.WriteFile(output+".stderr", stderr.Bytes(), 0644))
		panic(fmt.Sprintf("child: %v: %s", err, stderr.String()))
	}
	r := cost{WallMS: float64(wall) / 1e6, CPUms: float64(c.ProcessState.UserTime()+c.ProcessState.SystemTime()) / 1e6}
	r.OneCoreCPUPercent = 100 * r.CPUms / r.WallMS
	if u, ok := c.ProcessState.SysUsage().(*syscall.Rusage); ok {
		r.PeakRSSBytes = u.Maxrss
		if runtime.GOOS == "linux" {
			r.PeakRSSBytes *= 1024
		}
	}
	return r
}

func features(input, arm string) string {
	var values [jointdecision.ThreeFeatureDim]float32
	if arm == "positioned" {
		must(jointdecision.FeaturesIntoThree(input, &values))
	} else {
		must(jointdecision.FeaturesIntoThreeBag(input, &values))
	}
	var raw [jointdecision.ThreeFeatureDim * 4]byte
	for i, v := range values {
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(v))
	}
	return hash(raw[:])
}

func observe(compiler, revision, goBin, out string, t orderTask, language, order, budget int, arm string) record {
	lang := []string{"en", "ko"}[language]
	base := fmt.Sprintf("%s-%s-o%d", t.ID, lang, order)
	id := fmt.Sprintf("%s-a%d-%s", base, budget, arm)
	source, plan, cases := filepath.Join(out, t.ID+".gooo"), filepath.Join(out, fmt.Sprintf("%s-a%d-recipe.json", base, budget)), filepath.Join(out, base+"-cases.json")
	gen, execution := filepath.Join(out, id+"-generation.json"), filepath.Join(out, id+"-runtime.json")
	args := []string{"body-codegen", "--json", "--activity", "Compose", "--path-plan", plan}
	if arm != "deterministic" {
		args = append(args, "--path-model", modelPaths[arm])
	}
	r := record{ID: id, Family: t.ID, Language: lang, Arm: arm, Order: order, Budget: budget}
	r.Generation = execute(compiler, gen, append(args, source)...)
	var g generation
	must(json.Unmarshal(read(gen), &g))
	if g.Report.Compiler != revision || g.Source == "" {
		panic("source identity or emission missing")
	}
	s := g.Report.Paths.Search
	r.ModelPredictions = s.Selection.Calls
	r.Evaluations = s.Evaluations
	r.SearchStatus = s.Status
	r.EmittedSHA = hash([]byte(g.Source))
	r.InitialMask = s.Selection.Three.Mask
	r.Probabilities = s.Selection.Three.Probabilities
	r.ModelMetadataSHA, r.ModelWeightsSHA = s.Selection.Metadata, s.Selection.Weights
	if s.Selection.Choices["order"] == "schedule_reverse" {
		r.SelectedMask |= 1
	}
	if s.Selection.Choices["first-operands"] == "layout_reverse" {
		r.SelectedMask |= 2
	}
	if s.Selection.Choices["second-operands"] == "layout_reverse" {
		r.SelectedMask |= 4
	}
	if arm != "deterministic" {
		if r.ModelPredictions != 1 {
			panic("one actual prediction required")
		}
		r.FeatureSHA = features(s.Selection.Three.Input, arm)
	} else if r.ModelPredictions != 0 {
		panic("deterministic arm predicted")
	}
	r.Execution = execute(compiler, execution, "body-execute", "--source", source, "--path-plan", plan, "--generation", gen, "--cases", cases, "--go-bin", goBin)
	var result struct {
		Observation struct {
			Stage string
			Runs  []json.RawMessage
			Cases []struct{ Passed bool }
		}
	}
	must(json.Unmarshal(read(execution), &result))
	if result.Observation.Stage != "COMPLETE" || len(result.Observation.Runs) != 2 || len(result.Observation.Cases) != 8 {
		panic("native execution incomplete")
	}
	for _, c := range result.Observation.Cases {
		r.Total++
		if c.Passed {
			r.Passed++
		}
	}
	// Functional failures remain ordinary rows, including one-attempt failures.
	return r
}

func main() {
	compiler := flag.String("compiler", "", "clean main compiler")
	revision := flag.String("compiler-sha", "", "exact compiler revision")
	collector := flag.String("source-revision", "", "clean collector revision")
	goBin := flag.String("go-bin", "go", "Go 1.27.1")
	out := flag.String("out", "", "fresh output")
	flag.Parse()
	if len(*revision) != 40 || len(*collector) != 40 || *compiler == "" || *out == "" || flag.NArg() != 0 {
		panic("exact identities and fresh output required")
	}
	head, e := exec.Command("git", "rev-parse", "HEAD").Output()
	must(e)
	dirty, e := exec.Command("git", "status", "--porcelain").Output()
	must(e)
	if strings.TrimSpace(string(head)) != *collector || len(dirty) != 0 || runtime.Version() != "go1.27.1" {
		panic("clean pinned Go1.27.1 collector required")
	}
	build, e := exec.Command(*goBin, "version", "-m", *compiler).Output()
	must(e)
	if !bytes.Contains(build, []byte("vcs.revision="+*revision)) || !bytes.Contains(build, []byte("vcs.modified=false")) || !bytes.Contains(build, []byte("v0.2.18-experimental")) {
		panic("compiler identity differs")
	}
	if _, e = os.Stat(*out); !os.IsNotExist(e) {
		panic("output exists")
	}
	must(os.MkdirAll(*out, 0755))
	var rows []record
	sequence := 0
	for _, t := range orderTasks() {
		must(os.WriteFile(filepath.Join(*out, t.ID+".gooo"), []byte(t.source()), 0644))
		for language := range 2 {
			for order := range 2 {
				base := fmt.Sprintf("%s-%s-o%d", t.ID, []string{"en", "ko"}[language], order)
				save(filepath.Join(*out, base+"-cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": t.cases(order, []int64{-3, -2, -1, 0, 1, 2, 3, 4})})
				budgets := []int{1, 8}
				if sequence%2 == 1 {
					budgets = []int{8, 1}
				}
				for _, budget := range budgets {
					save(filepath.Join(*out, fmt.Sprintf("%s-a%d-recipe.json", base, budget)), t.recipe(language, order, budget))
					arms := []string{"deterministic", "positioned", "bag"}
					for i := range 3 {
						arm := arms[(i+sequence)%3]
						rows = append(rows, observe(*compiler, *revision, *goBin, *out, t, language, order, budget, arm))
					}
				}
				sequence++
			}
		}
	}
	save(filepath.Join(*out, "records.json"), rows)
	passed, calls := 0, 0
	for _, r := range rows {
		passed += r.Passed
		calls += r.ModelPredictions
	}
	save(filepath.Join(*out, "manifest.json"), map[string]any{"schema": "gooo/native-order-observation/v1", "compiler_sha": *revision, "compiler_binary_sha256": hash(read(*compiler)), "collector_sha": *collector, "os": runtime.GOOS, "arch": runtime.GOARCH, "generations": len(rows), "compiled_runs": len(rows) * 2, "model_predictions": calls, "passed": passed, "total": len(rows) * 8, "training_updates": 0, "protocol_sha256": hash(read("cmd/native-order-observe/README.md")), "collector_files_sha256": map[string]string{"main.go": hash(read("cmd/native-order-observe/main.go")), "tasks.go": hash(read("cmd/native-order-observe/tasks.go"))}})
	fmt.Printf("generations=%d runs=%d predictions=%d finite_expectations=%d/%d\n", len(rows), len(rows)*2, calls, passed, len(rows)*8)
}
