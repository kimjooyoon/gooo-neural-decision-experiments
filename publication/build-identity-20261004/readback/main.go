package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const source = "ebd5f653eaab0036b2586a5cbdd8187bce780600"
const frozenGo = "56d585c778ba593f4b220338058feb21527aa323f7585b773b308b4c1507e7ca"

func require(ok bool, why string) {
	if !ok {
		panic(why)
	}
}
func raw(path string) []byte {
	i, e := os.Lstat(path)
	require(e == nil && i != nil && i.Mode().IsRegular() && i.Size() > 0 && i.Size() <= 1<<20, "bounded file")
	b, e := os.ReadFile(path)
	require(e == nil && int64(len(b)) == i.Size(), "file extent")
	return b
}
func decode(path string, into any) { require(json.Unmarshal(raw(path), into) == nil, "saved JSON") }
func sha(b []byte) string          { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

type generation struct {
	Report struct {
		CompilerSHA string `json:"compiler_source_sha"`
		Paths       struct {
			Search struct {
				Selection struct {
					Predictions int `json:"local_model_predictions"`
				}
			}
		} `json:"body_paths"`
	}
}
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
type runtimeRecord struct {
	Observation struct {
		Stage       string
		Cases       []caseRow
		Runs        []process
		CompilerSHA string `json:"declared_compiler_source_sha"`
		ProducerSHA string `json:"producer_source_sha"`
		GoVersion   string `json:"go_version"`
		ParentSHA   string `json:"parent_receipt_sha256"`
		Replayed    bool   `json:"runtime_replayed"`
		Projection  bool   `json:"projection_replayed"`
	}
}
type summary struct {
	Rows []struct {
		Sequence      int
		Status        string
		Passed, Total int
		NativeRuns    int     `json:"native_runs"`
		ResponseMS    float64 `json:"response_ms"`
	}
}
type timing struct {
	Schema      string
	ProducerSHA string `json:"producer_source_sha"`
	Modified    string `json:"producer_modified"`
	Files       map[string]string
}

func main() {
	require(len(os.Args) == 2, "usage: readback <publication scope>")
	root := os.Args[1]
	var identity struct {
		Schema       string
		CompilerSHA  string `json:"compiler_source_sha"`
		SourceStatus string `json:"source_status"`
		GoVersion    string `json:"go_version"`
		Modified     string `json:"vcs_modified"`
		SDK          struct {
			Version  string
			Replaced bool
		} `json:"decision_runtime"`
	}
	decode(filepath.Join(root, "candidate-build-identity.json"), &identity)
	require(identity.Schema == "gooo/build-identity/v1" && identity.CompilerSHA == source && identity.SourceStatus == "CLEAN_VCS" && identity.Modified == "false" && identity.GoVersion == "go1.27.1" && identity.SDK.Version == "v0.2.21-experimental" && !identity.SDK.Replaced, "actual embedded identity")
	require(string(raw(filepath.Join(root, "candidate-version-contracts.json"))) == string(raw(filepath.Join(root, "installed-version-contracts.json"))), "legacy version bytes")
	var rows []map[string]any
	for _, lane := range []string{"model", "deterministic", "wrong-go"} {
		dir := filepath.Join(root, "_observations", lane)
		wrong := lane == "wrong-go"
		count := 2
		predictions := 1
		if wrong {
			count = 1
		}
		if lane == "deterministic" {
			predictions = 0
		}
		var s summary
		decode(filepath.Join(dir, "summary.json"), &s)
		require(len(s.Rows) == count, "request count")
		var inputs struct{ Cases []caseRow }
		decode(filepath.Join(dir, "cases.json"), &inputs)
		require(len(inputs.Cases) == 8, "eight authored expectations")
		for i, c := range inputs.Cases {
			require(c.Input == int64(i-3) && c.Expected == 2*c.Input+1, "frozen int64 inputs")
		}
		var times []float64
		for i, row := range s.Rows {
			prefix := fmt.Sprintf("run-%d", i+1)
			require(row.Sequence == i+1 && row.Total == 8 && row.ResponseMS > 0, "summary scope")
			var g generation
			decode(filepath.Join(dir, prefix+"-generation.json"), &g)
			require(g.Report.CompilerSHA == source && g.Report.Paths.Search.Selection.Predictions == predictions, "model count/source")
			require(sha(raw(filepath.Join(dir, prefix+"-generated.go"))) == frozenGo, "unchanged generated Go")
			var r runtimeRecord
			decode(filepath.Join(dir, prefix+"-runtime.json"), &r)
			require(r.Observation.CompilerSHA == source && r.Observation.ProducerSHA == source, "runtime source")
			if wrong {
				require(row.Status == "execution_failed" && row.NativeRuns == 0 && len(r.Observation.Cases) == 0 && len(r.Observation.Runs) == 0 && r.Observation.Stage == "TOOLCHAIN" && r.Observation.GoVersion == "go version go1.26.5 darwin/arm64", "actual wrong tool, no native outputs")
			} else {
				require(r.Observation.GoVersion == "go version go1.27.1 darwin/arm64", "actual native Go1.27.1")
				require(row.Status == "completed" && row.Passed == 8 && row.NativeRuns == 2 && r.Observation.Stage == "COMPLETE" && len(r.Observation.Cases) == 8 && len(r.Observation.Runs) == 2 && r.Observation.Replayed && r.Observation.Projection, "current native completion")
				for j, c := range r.Observation.Cases {
					require(c.Input == inputs.Cases[j].Input && c.Expected == inputs.Cases[j].Expected && c.Actual == c.Expected && c.Passed, "int64 expectation/actual")
				}
				for _, p := range r.Observation.Runs {
					require(p.Started && p.Completed && !p.Canceled && !p.TimedOut && p.ExitCode == 0, "native process outcome")
				}
				require(r.Observation.Runs[0].StdoutSHA == r.Observation.Runs[1].StdoutSHA && strings.HasPrefix(r.Observation.Runs[0].StdoutSHA, "sha256:"), "runtime replay bytes")
			}
			var t timing
			decode(filepath.Join(dir, prefix+"-timing.json"), &t)
			require(t.Schema == "gooo/body-path-file-timing/v1" && t.ProducerSHA == source && t.Modified == "false" && len(t.Files) == 5, "timing source/scope")
			for _, suffix := range []string{"generated.go", "generation.json", "request.json", "response.json", "runtime.json"} {
				name := prefix + "-" + suffix
				require(t.Files[name] == "sha256:"+sha(raw(filepath.Join(dir, name))), "timing artifact hash")
			}
			times = append(times, row.ResponseMS)
		}
		row := map[string]any{"lane": lane, "constructions": count, "actual_model_predictions": count * predictions, "response_ms": times, "native_runs": 4, "finite_passed": 16, "finite_total": 16, "unobserved_declared_expectations": 0}
		if wrong {
			row["native_runs"] = 0
			row["finite_passed"] = nil
			row["finite_total"] = nil
			row["unobserved_declared_expectations"] = 8
		}
		rows = append(rows, row)
	}
	hint := string(raw(filepath.Join(root, "wrong-go.stderr")))
	require(strings.Contains(hint, "go1.26.5") && strings.Contains(hint, "--go-bin"), "actual next action")
	var failed, resumed runtimeRecord
	decode(filepath.Join(root, "_observations", "wrong-go", "run-1-runtime.json"), &failed)
	decode(filepath.Join(root, "recovered-execution.json"), &resumed)
	o := resumed.Observation
	require(o.Stage == "COMPLETE" && o.CompilerSHA == source && o.ProducerSHA == source && o.GoVersion == "go version go1.27.1 darwin/arm64" && o.Replayed && o.Projection && len(o.Cases) == 8 && len(o.Runs) == 2 && o.ParentSHA == failed.Observation.ParentSHA && strings.HasPrefix(o.ParentSHA, "sha256:"), "resumed same generation parent")
	for i, c := range o.Cases {
		require(c.Input == int64(i-3) && c.Expected == 2*c.Input+1 && c.Actual == c.Expected && c.Passed, "resumed int64 expectations")
	}
	for _, p := range o.Runs {
		require(p.Started && p.Completed && !p.Canceled && !p.TimedOut && p.ExitCode == 0, "resumed native outcomes")
	}
	require(o.Runs[0].StdoutSHA == o.Runs[1].StdoutSHA && strings.HasPrefix(o.Runs[0].StdoutSHA, "sha256:"), "resumed replay")
	require(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "historical_lanes": rows, "resumed_execution": map[string]any{"finite_passed": 8, "finite_total": 8, "native_runs": 2, "new_model_predictions": 0, "same_generation_parent": true}, "reader_model_predictions": 0, "reader_native_runs": 0, "new_intent_tasks": 0, "training_updates": 0}) == nil, "readback output")
}
