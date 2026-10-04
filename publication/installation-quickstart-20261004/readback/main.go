package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

const source = "4f6c7566dd4390b04a48589c82eb2d9eea3a6589"
const generatedSHA = "56d585c778ba593f4b220338058feb21527aa323f7585b773b308b4c1507e7ca"

func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}
func read(path string) []byte {
	i, err := os.Lstat(path)
	check(err == nil && i != nil && i.Mode().IsRegular() && i.Size() > 0 && i.Size() <= 1<<20, "invalid saved file: "+path)
	b, err := os.ReadFile(path)
	check(err == nil && int64(len(b)) == i.Size(), "saved file read: "+path)
	return b
}
func decode(path string, into any) {
	check(json.Unmarshal(read(path), into) == nil, "saved JSON: "+path)
}
func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

type caseRow struct {
	Input, Expected, Actual int64
	Passed                  bool
}
type process struct {
	Started, Completed, Canceled bool
	TimedOut                     bool   `json:"timed_out"`
	ExitCode                     int    `json:"exit_code"`
	StdoutSHA                    string `json:"stdout_sha256"`
}
type generation struct {
	Report struct {
		CompilerSHA string `json:"compiler_source_sha"`
		BodyPaths   struct {
			Search struct {
				Selection struct {
					Predictions int `json:"local_model_predictions"`
				}
			} `json:"search"`
		} `json:"body_paths"`
	}
}
type runtimeRecord struct {
	Observation struct {
		Stage              string
		CompilerSHA        string `json:"declared_compiler_source_sha"`
		ProducerSHA        string `json:"producer_source_sha"`
		GoVersion          string `json:"go_version"`
		Replayed           bool   `json:"runtime_replayed"`
		ProjectionReplayed bool   `json:"projection_replayed"`
		Cases              []caseRow
		Runs               []process
	}
}
type timing struct {
	Schema      string
	ProducerSHA string `json:"producer_source_sha"`
	Modified    string `json:"producer_modified"`
	Files       map[string]string
}
type summary struct {
	Schema string
	Rows   []struct {
		Sequence      int
		Status        string
		Passed, Total int
		NativeRuns    int     `json:"native_runs"`
		ResponseMS    float64 `json:"response_ms"`
	}
}

func checkBuild(root, name string, bound bool) {
	var b struct {
		BinarySHA string          `json:"binary_sha256"`
		Info      debug.BuildInfo `json:"build_info"`
	}
	decode(filepath.Join(root, name+"-buildinfo.json"), &b)
	wantSHA := "5287461559e8a57dfdab2751f2c51154ede6c9ef044a3946761c2e604954a867"
	if bound {
		wantSHA = "9f93d03a862d43c4359489201777920aafdf5a6b7848882c63ea9ce25ac4a17d"
	}
	check(b.BinarySHA == wantSHA && b.Info.GoVersion == "go1.27.1" && b.Info.Path == "github.com/kimjooyoon/meta-ontology-go/cmd/gooo", "saved binary identity")
	check(b.Info.Main.Path == "github.com/kimjooyoon/meta-ontology-go", "main module")
	if !bound {
		check(b.Info.Main.Version == "v0.4.0-dev.0.20261004014639-4f6c7566dd43", "module source pin")
	}
	check(len(b.Info.Deps) == 1 && b.Info.Deps[0].Path == "github.com/kimjooyoon/gooo-decision-runtime" && b.Info.Deps[0].Version == "v0.2.21-experimental" && b.Info.Deps[0].Replace == nil, "released SDK identity")
	settings := map[string]string{}
	for _, s := range b.Info.Settings {
		settings[s.Key] = s.Value
	}
	if bound {
		check(settings["vcs.revision"] == source && settings["vcs.modified"] == "false", "clean Git build")
	} else {
		_, revision := settings["vcs.revision"]
		_, modified := settings["vcs.modified"]
		check(!revision && !modified, "original module provenance gap")
	}
}

func main() {
	check(len(os.Args) == 2, "usage: readback <publication scope>")
	root := os.Args[1]
	checkBuild(root, "module", false)
	checkBuild(root, "checkout", true)
	var rows []map[string]any
	for _, lane := range []string{"module-default-tool", "module-explicit-tool", "checkout-explicit-tool"} {
		dir := filepath.Join(root, lane, "_results")
		bound, completed := lane == "checkout-explicit-tool", lane != "module-default-tool"
		wantSource := "UNBOUND_LOCAL_SOURCE"
		if bound {
			wantSource = source
		}
		var s summary
		decode(filepath.Join(dir, "summary.json"), &s)
		check(s.Schema == "gooo/body-path-file-run/v1" && len(s.Rows) == 2, "summary scope")
		var inputs struct{ Cases []caseRow }
		decode(filepath.Join(dir, "cases.json"), &inputs)
		check(len(inputs.Cases) == 8, "eight expectations required")
		for n, c := range inputs.Cases {
			check(c.Input == int64(n-3) && c.Expected == 2*c.Input+1, "frozen input/expectation")
		}
		var times []float64
		for n, row := range s.Rows {
			prefix := fmt.Sprintf("run-%d", n+1)
			check(row.Sequence == n+1 && row.Total == 8 && row.ResponseMS > 0, "summary sequence/scope")
			var g generation
			decode(filepath.Join(dir, prefix+"-generation.json"), &g)
			check(g.Report.CompilerSHA == wantSource && g.Report.BodyPaths.Search.Selection.Predictions == 1, "source/prediction receipt")
			check(digest(read(filepath.Join(dir, prefix+"-generated.go"))) == generatedSHA, "frozen generated Go")
			var r runtimeRecord
			decode(filepath.Join(dir, prefix+"-runtime.json"), &r)
			check(r.Observation.CompilerSHA == wantSource && r.Observation.ProducerSHA == wantSource, "runtime source")
			if completed {
				check(row.Status == "completed" && row.Passed == 8 && row.NativeRuns == 2, "completed summary")
				check(r.Observation.Stage == "COMPLETE" && r.Observation.Replayed && r.Observation.ProjectionReplayed && len(r.Observation.Cases) == 8 && len(r.Observation.Runs) == 2, "complete native observation")
				check(r.Observation.GoVersion == "go version go1.27.1 darwin/arm64", "actual Go1.27.1")
				for i, c := range r.Observation.Cases {
					check(c.Input == inputs.Cases[i].Input && c.Expected == inputs.Cases[i].Expected && c.Actual == c.Expected && c.Passed, "int64 expected/actual")
				}
				for _, p := range r.Observation.Runs {
					check(p.Started && p.Completed && !p.Canceled && !p.TimedOut && p.ExitCode == 0, "joined native run")
				}
				check(r.Observation.Runs[0].StdoutSHA == r.Observation.Runs[1].StdoutSHA && strings.HasPrefix(r.Observation.Runs[0].StdoutSHA, "sha256:"), "native replay bytes")
			} else {
				check(row.Status == "execution_failed" && row.NativeRuns == 0 && row.Passed == 0, "original tool failure")
				check(r.Observation.Stage == "TOOLCHAIN" && len(r.Observation.Cases) == 0 && len(r.Observation.Runs) == 0 && r.Observation.GoVersion == "go version go1.26.5 darwin/arm64", "unobserved expectations")
			}
			var t timing
			decode(filepath.Join(dir, prefix+"-timing.json"), &t)
			check(t.Schema == "gooo/body-path-file-timing/v1" && len(t.Files) == 5, "timing file scope")
			if bound {
				check(t.ProducerSHA == source && t.Modified == "false", "clean timing source")
			} else {
				check(t.ProducerSHA == "unobserved" && t.Modified == "unobserved", "module timing provenance gap")
			}
			for _, suffix := range []string{"generated.go", "generation.json", "request.json", "response.json", "runtime.json"} {
				name := prefix + "-" + suffix
				check(t.Files[name] == "sha256:"+digest(read(filepath.Join(dir, name))), "sidecar hash: "+name)
			}
			times = append(times, row.ResponseMS)
		}
		row := map[string]any{"lane": lane, "constructions": 2, "actual_model_predictions": 2, "compiler_source": wantSource, "response_ms": times, "native_runs": 0, "finite_passed": nil, "finite_total": nil, "unobserved_declared_expectations": 16}
		if completed {
			row["native_runs"] = 4
			row["finite_passed"] = 16
			row["finite_total"] = 16
			row["unobserved_declared_expectations"] = 0
		}
		rows = append(rows, row)
	}
	check(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "historical_lanes": rows, "reader_model_predictions": 0, "reader_native_runs": 0, "new_intent_tasks": 0, "training_updates": 0}) == nil, "write readback")
}
