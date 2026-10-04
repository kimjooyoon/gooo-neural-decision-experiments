package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func require(ok bool) {
	if !ok {
		panic("public shared model observation differs")
	}
}
func read(p string) []byte { b, e := os.ReadFile(p); must(e); return b }
func sha(b []byte) string  { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func save(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(p, append(b, '\n'), 0644))
}
func main() {
	if len(os.Args) != 9 {
		panic("original publication, original pinned public readback, extracted native directory, compiler, worker and fresh output, exact compiler source and HF revision required")
	}
	local, public, native, compiler, worker, out := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6]
	require(func() bool { _, e := os.Stat(out); return os.IsNotExist(e) }())
	must(os.MkdirAll(out, 0755))
	files := 0
	must(filepath.WalkDir(local, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			r, e := filepath.Rel(local, p)
			must(e)
			require(bytes.Equal(read(p), read(filepath.Join(public, r))))
			files++
		}
		return nil
	}))
	require(files >= 13) // Compare every file in the supplied publication, including later validation records.
	var identity struct {
		Source string `json:"compiler_source_sha"`
		State  string `json:"source_status"`
		Go     string `json:"go_version"`
	}
	raw, e := exec.Command(compiler, "version", "--build", "--json").Output()
	must(e)
	must(json.Unmarshal(raw, &identity))
	require(identity.Source == os.Args[7] && identity.State == "CLEAN_VCS" && identity.Go == "go1.27.1")
	observations := map[string]any{}
	for _, profile := range []string{"deterministic", "fp32", "ptq_ternary", "qat_ternary"} {
		for _, budget := range []int{1, 8} {
			if budget == 1 && profile != "deterministic" && profile != "fp32" {
				continue
			}
			stem := fmt.Sprintf("public-%s-b%d", profile, budget)
			source := strings.Replace(string(read(filepath.Join(public, "demo.gooo.fixture"))), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1)
			sourceFile := filepath.Join(out, stem+".gooo.fixture")
			must(os.WriteFile(sourceFile, []byte(source), 0644))
			args := []string{"body-compose", "--source", sourceFile, "--cases", filepath.Join(public, "demo-cases.json")}
			if profile != "deterministic" {
				args = append(args, "--model", filepath.Join(public, "models", profile, "model.json"))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			command := exec.CommandContext(ctx, compiler, args...)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			started := time.Now()
			e := command.Run()
			wall := time.Since(started).Seconds()
			cancel()
			must(e)
			must(os.WriteFile(filepath.Join(out, stem+".json"), stdout.Bytes(), 0600))
			var capture struct {
				Composition struct {
					Steps []struct {
						Generation struct {
							Report struct {
								Assembly struct {
									Calls  int `json:"model_calls"`
									Fields int `json:"fields_passed"`
									Total  int `json:"fields_total"`
								} `json:"record_assembly"`
							} `json:"report"`
						} `json:"generation"`
					} `json:"steps"`
				} `json:"composition"`
				Runtime struct {
					Calls  int    `json:"model_calls"`
					Passed int    `json:"finite_passed"`
					Total  int    `json:"finite_total"`
					Source string `json:"producer_source_sha"`
				} `json:"runtime"`
			}
			must(json.Unmarshal(stdout.Bytes(), &capture))
			require(len(capture.Composition.Steps) == 2 && capture.Runtime.Calls == 0 && capture.Runtime.Total == 8 && capture.Runtime.Source == identity.Source)
			a := capture.Composition.Steps[0].Generation.Report.Assembly
			require(a.Total == 15 && ((profile == "deterministic" && a.Calls == 0) || (profile != "deterministic" && a.Calls == 1)))
			if budget == 8 {
				require(a.Fields == 15 && capture.Runtime.Passed == 8)
			}
			must(os.WriteFile(filepath.Join(out, stem+".json"), stdout.Bytes(), 0600))
			observations[stem] = map[string]any{"selection_fields_passed": a.Fields, "selection_fields_total": a.Total, "named_outputs_passed": capture.Runtime.Passed, "named_outputs_total": capture.Runtime.Total, "model_calls": a.Calls, "native_model_calls": capture.Runtime.Calls, "wall_seconds": wall, "cpu_seconds": command.ProcessState.UserTime().Seconds() + command.ProcessState.SystemTime().Seconds(), "raw_sha256": sha(stdout.Bytes())}
		}
	}
	// Submit one request and require its response while stdin remains open. Only
	// then send the remaining mixed-goal sources. This detects an EOF barrier.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, worker, "--workers", "2", "--model", filepath.Join(public, "models", "fp32", "model.json"))
	stdin, e := command.StdinPipe()
	must(e)
	stdout, e := command.StdoutPipe()
	must(e)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	started := time.Now()
	must(command.Start())
	first := make(chan struct{})
	writes := make(chan error, 1)
	// The complete public demo is one goal; use current paired sources for the rest.
	entries, e := os.ReadDir(native)
	must(e)
	var sources []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "-fp32-b8.gooo.fixture") {
			sources = append(sources, string(read(filepath.Join(native, entry.Name()))))
		}
	}
	require(len(sources) == 24)
	go func() {
		defer stdin.Close()
		for i := range 32 {
			row, e := json.Marshal(map[string]any{"schema": "gooo/native-body-stream-request/v1", "correlation_id": fmt.Sprintf("paired-public-%d", i), "source": sources[i%len(sources)], "activity": "Select", "options": map[string]any{}})
			if e == nil {
				_, e = stdin.Write(append(row, '\n'))
			}
			if e != nil {
				writes <- e
				return
			}
			if i == 0 {
				select {
				case <-first:
				case <-ctx.Done():
					writes <- ctx.Err()
					return
				}
			}
		}
		writes <- nil
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var responses bytes.Buffer
	seen := map[string]bool{}
	firstSeconds := 0.
	fields := 0
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var r struct {
			Status   string `json:"status"`
			ID       string `json:"correlation_id"`
			Response struct {
				Report struct {
					Assembly struct {
						Calls  int `json:"model_calls"`
						Fields int `json:"fields_passed"`
						Total  int `json:"fields_total"`
					} `json:"record_assembly"`
				} `json:"report"`
			} `json:"response"`
		}
		must(json.Unmarshal(line, &r))
		a := r.Response.Report.Assembly
		require(r.Status == "completed" && !seen[r.ID] && a.Calls == 1 && a.Fields == 15 && a.Total == 15)
		if len(seen) == 0 {
			require(r.ID == "paired-public-0")
			firstSeconds = time.Since(started).Seconds()
			close(first)
		}
		seen[r.ID] = true
		fields += a.Fields
		responses.Write(line)
		responses.WriteByte('\n')
	}
	must(scanner.Err())
	must(<-writes)
	must(command.Wait())
	wholeSeconds := time.Since(started).Seconds()
	require(len(seen) == 32 && fields == 480)
	usage, ok := command.ProcessState.SysUsage().(*syscall.Rusage)
	require(ok && usage.Maxrss > 0)
	workerRSS := usage.Maxrss
	if runtime.GOOS == "linux" {
		workerRSS *= 1024
	}
	must(os.WriteFile(filepath.Join(out, "worker-responses.jsonl"), responses.Bytes(), 0600))
	must(os.WriteFile(filepath.Join(out, "worker-setup.json"), stderr.Bytes(), 0600))
	save(filepath.Join(out, "report.json"), map[string]any{"schema": "gooo/shared-field-public-dogfood/v1", "hf_revision": os.Args[8], "verified_public_files": files, "compiler_source": identity.Source, "compiler_binary_sha256": sha(read(compiler)), "worker_binary_sha256": sha(read(worker)), "native_constructions": observations, "worker": map[string]any{"workers": 2, "requests_completed": 32, "selection_fields_passed": fields, "selection_fields_total": 480, "first_response_before_stdin_close": true, "first_response_wall_seconds": firstSeconds, "whole_wall_seconds": wholeSeconds, "process_cpu_seconds": command.ProcessState.UserTime().Seconds() + command.ProcessState.SystemTime().Seconds(), "peak_rss_bytes": workerRSS, "platform": runtime.GOOS, "deadline_seconds": 60, "status": "COMPLETED"}, "scope": "Downloaded exact public weights drive generated Gooo programs. Mixed-goal worker construction finishes before input EOF and all requests complete. Native graph execution uses body-compose; worker record construction alone is measured here. Finite bounded observation, no universal deadlock claim."})
	fmt.Printf("Verified %d public files; executed six controls; 32 two-worker requests completed with 480/480 fields and first response before stdin closed.\n", files)
}
