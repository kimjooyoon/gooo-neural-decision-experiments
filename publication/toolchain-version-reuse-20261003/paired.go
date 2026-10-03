// Paired local observations of two compiler binaries. No training or new intent task.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"syscall"
	"time"
)

type object = map[string]any

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func raw(path string) []byte    { b, e := os.ReadFile(path); must(e); return b }
func decode(path string) object { var o object; must(json.Unmarshal(raw(path), &o)); return o }
func obj(o object, k string) object {
	r, ok := o[k].(map[string]any)
	if !ok {
		panic("missing object " + k)
	}
	return r
}
func num(o object, k string) float64 {
	r, ok := o[k].(float64)
	if !ok {
		panic("missing number " + k)
	}
	return r
}
func check(ok bool, reason string) {
	if !ok {
		panic(reason)
	}
}
func write(path string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(path, append(b, '\n'), 0600))
}
func hash(path string) string { s := sha256.Sum256(raw(path)); return hex.EncodeToString(s[:]) }

func run(binary string, args []string, dir, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/time", append([]string{"-l", binary}, args...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 3 * time.Second
	stdout, e := os.OpenFile(filepath.Join(dir, name+".stdout"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(e)
	defer stdout.Close()
	stderr, e := os.OpenFile(filepath.Join(dir, name+".time"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(e)
	defer stderr.Close()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	start := time.Now()
	e = cmd.Run()
	write(filepath.Join(dir, name+"-process.json"), object{"elapsed_ns": time.Since(start).Nanoseconds(), "completed": e == nil, "timeout": ctx.Err() != nil})
	must(e)
}

func median(xs []float64) float64 {
	ys := append([]float64(nil), xs...)
	sort.Float64s(ys)
	n := len(ys)
	if n%2 == 1 {
		return ys[n/2]
	}
	return (ys[n/2-1] + ys[n/2]) / 2
}

func main() {
	check(len(os.Args) == 6, "usage paired baseline candidate inputs model output")
	baseline, candidate, inputs, model, out := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]
	must(os.Mkdir(out, 0700))
	goTool := os.Getenv("GOOO_LOCAL_GO")
	check(goTool != "", "set GOOO_LOCAL_GO to the native Go1.27.1 binary")
	paths := []string{baseline, candidate, goTool, model, filepath.Join(filepath.Dir(model), "weights.bin"), filepath.Join(inputs, "cases-128.json")}
	for _, lang := range []string{"ko", "en"} {
		paths = append(paths, filepath.Join(inputs, lang+"-source.gooo"), filepath.Join(inputs, lang+"-recipe.json"))
	}
	before := map[string]string{}
	for _, path := range paths {
		before[path] = hash(path)
	}
	write(filepath.Join(out, "input-hashes-before.json"), before)
	write(filepath.Join(out, "protocol.json"), object{"source_scope": "two earlier clamp-wrapped ordering fixtures; repetitions are not new intent tasks", "window_order": []string{"baseline", "candidate", "candidate", "baseline"}, "conditions": []string{"ko-model", "ko-deterministic", "en-model", "en-deterministic"}, "repetitions_per_window": 6, "finite_cases_per_request": 128, "response_scope": "CLI response_ms excludes initial setup and saving; current children only for CPU/RSS", "whole_host_cpu": "unobserved", "compiler_sources": object{"baseline": "eb8477d51f58131e0dc3b7126c547bde2c5d0e5f", "candidate": "a7fe6d4f9bc3a846dbad6fbff8e90f8bc7c309ae"}})
	records := []object{}
	for _, lang := range []string{"ko", "en"} {
		for _, mode := range []string{"model", "deterministic"} {
			condition := lang + "-" + mode
			activity := "AssembleKorean"
			if lang == "en" {
				activity = "AssembleEnglish"
			}
			old := filepath.Join(filepath.Dir(inputs), "raw-computes-installed", condition)
			expectedGo := raw(filepath.Join(old, "run-1-generated.go"))
			for window, label := range []string{"baseline", "candidate", "candidate", "baseline"} {
				name := fmt.Sprintf("%s-%d-%s", condition, window, label)
				dest := filepath.Join(out, name)
				binary := baseline
				if label == "candidate" {
					binary = candidate
				}
				args := []string{"body-path-run", "--source", filepath.Join(inputs, lang+"-source.gooo"), "--activity", activity, "--path-plan", filepath.Join(inputs, lang+"-recipe.json"), "--cases", filepath.Join(inputs, "cases-128.json"), "--go-bin", goTool, "--out", dest, "--repeat", "6"}
				if mode == "model" {
					args = append(args, "--model", model)
				}
				run(binary, args, out, name)
				rows := decode(filepath.Join(dest, "summary.json"))["rows"].([]any)
				check(len(rows) == 6, "row count")
				var originalSourceCheck, originalSourceOutput any
				for i, row := range rows {
					s := row.(map[string]any)
					generation := decode(filepath.Join(dest, fmt.Sprintf("run-%d-generation.json", i+1)))
					runtime := decode(filepath.Join(dest, fmt.Sprintf("run-%d-runtime.json", i+1)))
					o := obj(runtime, "observation")
					check(s["status"] == "completed" && num(s, "passed") == 128 && num(s, "total") == 128 && num(s, "native_runs") == 2, "finite/native failure")
					check(bytes.Equal(raw(filepath.Join(dest, fmt.Sprintf("run-%d-generated.go", i+1))), expectedGo), "generated bytes changed")
					check(o["projection_replayed"] == true && o["runtime_replayed"] == true && o["stage"] == "COMPLETE", "missing replay")
					check(o["go_tool_sha256"] == "sha256:"+before[goTool], "Go bytes")
					b := obj(obj(generation, "report"), "body_paths")
					selection := obj(obj(b, "search"), "selection")
					predictions := num(selection, "local_model_predictions")
					expectedPred := float64(0)
					if mode == "model" {
						expectedPred = 1
					}
					check(predictions == expectedPred && num(selection, "external_provider_calls") == 0, "model accounting")
					tool := obj(o, "toolchain")
					build := obj(o, "build")
					artifact := obj(o, "artifact")
					check(artifact["reused"] == (i > 0) && build["started"] == (i == 0), "artifact reuse")
					if label == "candidate" {
						ref := obj(o, "toolchain_reference")
						check(ref["reused"] == (i > 0) && ref["bytes_verified"] == true && ref["build_info_verified"] == true, "tool ref")
						if i == 0 {
							originalSourceCheck = ref["source_check"]
							originalSourceOutput = ref["source_output"]
						}
						check(reflect.DeepEqual(originalSourceCheck, ref["source_check"]) && reflect.DeepEqual(originalSourceOutput, ref["source_output"]), "tool history changed")
						check(tool["started"] == (i == 0) && obj(runtime, "completeness_receipt")["profile_id"] == "gooo/typed-path-runtime-v3", "tool current/profile")
					} else {
						check(tool["started"] == true && obj(runtime, "completeness_receipt")["profile_id"] == "gooo/typed-path-runtime-v2", "baseline current/profile")
					}
					processes := []any{tool, build}
					processes = append(processes, o["runs"].([]any)...)
					cpu, wall, rss, started := float64(0), float64(0), float64(0), 0
					for _, p := range processes {
						p := p.(map[string]any)
						if p["started"] == true {
							check(p["completed"] == true && num(p, "exit_code") == 0, "child failure")
							started++
							cpu += num(p, "user_ns") + num(p, "system_ns")
							wall += num(p, "wall_ns")
							rss = max(rss, num(p, "peak_rss_bytes"))
						}
					}
					expectedChildren := 3
					if i == 0 {
						expectedChildren = 4
					}
					if label == "candidate" && i > 0 {
						expectedChildren = 2
					}
					check(started == expectedChildren, "current children")
					for _, axis := range obj(runtime, "completeness_receipt")["dimensions"].([]any) {
						a := axis.(map[string]any)
						if a["id"] == "runtime_child_resources" {
							check(num(a, "numerator") == float64(started) && num(a, "denominator") == float64(started), "resource denominator")
						}
					}
					records = append(records, object{"condition": condition, "window": window, "variant": label, "sequence": i + 1, "warm": i > 0, "response_ms": num(s, "response_ms"), "generation_ms": num(obj(b, "timing"), "total_ms"), "runtime_ms": num(o, "elapsed_ns") / 1e6, "current_child_cpu_ms": cpu / 1e6, "current_child_wall_ms": wall / 1e6, "max_current_child_rss_bytes": rss, "current_children": started, "local_model_predictions": predictions, "passed": 128, "total": 128, "native_runs": 2})
				}
				write(filepath.Join(out, "records.json"), records)
				fmt.Printf("completed %s: 6 requests, 768/768 expectations\n", name)
			}
		}
	}
	stats := []object{}
	for _, condition := range []string{"all", "ko-model", "ko-deterministic", "en-model", "en-deterministic"} {
		for _, label := range []string{"baseline", "candidate"} {
			for _, warm := range []bool{false, true} {
				values := map[string][]float64{}
				for _, r := range records {
					if r["variant"] == label && r["warm"] == warm && (condition == "all" || r["condition"] == condition) {
						for _, key := range []string{"response_ms", "generation_ms", "runtime_ms", "current_child_cpu_ms", "current_child_wall_ms", "max_current_child_rss_bytes"} {
							values[key] = append(values[key], num(r, key))
						}
					}
				}
				stat := object{"condition": condition, "variant": label, "warm": warm, "n": len(values["response_ms"])}
				for k, v := range values {
					sort.Float64s(v)
					stat[k] = object{"min": v[0], "median": median(v), "max": v[len(v)-1]}
				}
				stats = append(stats, stat)
			}
		}
	}
	after := map[string]string{}
	for _, path := range paths {
		after[path] = hash(path)
		check(after[path] == before[path], "input mutated "+filepath.Base(path))
	}
	write(filepath.Join(out, "input-hashes-after.json"), after)
	write(filepath.Join(out, "summary.json"), object{"status": "PASS_WITHIN_REPEATED_FIXTURE_SCOPE", "requests": 96, "known_intent_fixtures": 2, "distinct_intent_tasks_added": 0, "local_model_predictions": 48, "native_runs": 192, "finite_passed": 12288, "finite_total": 12288, "toolchain_children": object{"baseline": 48, "candidate": 8}, "native_build_children": object{"baseline": 8, "candidate": 8}, "profiles": object{"baseline": "gooo/typed-path-runtime-v2", "candidate": "gooo/typed-path-runtime-v3"}, "numeric_completeness_comparison": "different profile contracts; INCOMPARABLE_SCOPE", "measurements": stats})
}
