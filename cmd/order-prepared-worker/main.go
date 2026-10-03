// Exercise the existing retained Gooo worker with open-stdin round trips and
// bounded batches, executing every received projection immediately.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

const baselineSHA = "47315bf031f4734497f1a9f3710e93b6c0e8f43ed385f78ec7d7cc51adfa2be1"
const weightsSHA = "cf00ccc83d17d28ed73fcb869366151a48ffccd3aa8ca8e635aabf19810b9e78"

type prior struct {
	ID, Request, Arm, SourceSHA, GenerationSHA, RuntimeSHA string
	Budget                                                 int
}
type native struct {
	Observation struct {
		Stage string
		Runs  []json.RawMessage
		Cases []struct {
			Input, Expected, Actual int64
			Passed                  bool
		}
	}
}
type generation struct {
	Source string
	Report struct {
		Compiler string `json:"compiler_source_sha"`
		Paths    struct {
			SourceSHA   string `json:"original_source_sha256"`
			Bound       bool   `json:"source_base_matched"`
			Search      json.RawMessage
			Ranking     map[string]any `json:"whole_candidate_judgment"`
			Preparation *struct {
				Reused         bool
				CandidateCount int `json:"candidate_count"`
			} `json:"whole_candidate_preparation"`
		} `json:"body_paths"`
	}
}
type response struct {
	Schema, Status string
	CorrelationID  string `json:"correlation_id"`
	Sequence       uint64
	Response       json.RawMessage
	Error          string
}
type observation struct {
	ID, Request, Arm, Phase                          string
	Workers, Trial, Calls, NativeRuns, Passed, Total int
	Sequence                                         uint64
	ResponseNS                                       int64
	Reused                                           bool
	GenerationSHA, RuntimeSHA                        string
	Execution                                        cost
}
type pending struct {
	id, request, phase string
	sequence           uint64
	trial              int
	started            time.Time
}
type inputs struct {
	worker, compiler, revision, goBin, baseline, model, out string
	priors                                                  map[string]prior
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
func read(path string) []byte    { b, e := os.ReadFile(path); must(e); return b }
func hash(b []byte) string       { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func decode(b []byte, value any) { must(json.Unmarshal(b, value)) }
func save(path string, v any) string {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	b = append(b, '\n')
	must(os.WriteFile(path, b, 0644))
	return hash(b)
}
func canonical(b []byte) string {
	var v any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	must(d.Decode(&v))
	raw, e := json.Marshal(v)
	must(e)
	return string(raw)
}
func binaryIdentity(goBin, binary, revision string) {
	b, e := exec.Command(goBin, "version", "-m", binary).Output()
	must(e)
	require(bytes.Contains(b, []byte("vcs.revision="+revision)) && bytes.Contains(b, []byte("vcs.modified=false")) &&
		bytes.Contains(b, []byte("github.com/kimjooyoon/gooo-decision-runtime\tv0.2.20-experimental\t")) && !bytes.Contains(b, []byte("\n\t=>")), "clean compiler/worker identity differs")
}

func (in inputs) executeResponse(r response, p pending, arm string, workers int) observation {
	roundTrip := time.Since(p.started).Nanoseconds()
	require(r.Schema == "gooo/native-body-stream-result/v1" && r.Status == "completed" && r.Sequence == p.sequence, "worker response rejected or mismatched: "+r.Error)
	old := in.priors[p.request+"-"+arm+"-b8"]
	var g, before generation
	decode(r.Response, &g)
	raw := read(filepath.Join(in.baseline, old.ID+"-generation.json"))
	require(hash(raw) == old.GenerationSHA, "frozen generation changed")
	decode(raw, &before)
	require(g.Source == before.Source && g.Report.Compiler == in.revision && g.Report.Paths.Bound && g.Report.Paths.SourceSHA == old.SourceSHA && canonical(g.Report.Paths.Search) == canonical(before.Report.Paths.Search), "generation semantics changed")
	a, b := g.Report.Paths.Ranking, before.Report.Paths.Ranking
	if a != nil {
		a["predict_ns"] = float64(0)
	}
	if b != nil {
		b["predict_ns"] = float64(0)
	}
	ax, _ := json.Marshal(a)
	bx, _ := json.Marshal(b)
	require(bytes.Equal(ax, bx), "complete ranking changed")
	o := observation{ID: p.id, Request: p.request, Arm: arm, Phase: p.phase, Workers: workers, Trial: p.trial, Sequence: r.Sequence, ResponseNS: roundTrip}
	var accounting struct {
		Selection struct {
			Calls int `json:"local_model_predictions"`
		}
	}
	decode(g.Report.Paths.Search, &accounting)
	o.Calls = accounting.Selection.Calls
	if arm == "model" {
		require(g.Report.Paths.Preparation != nil && g.Report.Paths.Preparation.CandidateCount == 8, "preparation missing")
		o.Reused = g.Report.Paths.Preparation.Reused
		if p.phase == "sequential" {
			require(o.Reused == (p.trial == 1), "sequential reuse differs")
		}
		require(o.Calls == 1, "model call count differs")
	} else {
		require(g.Report.Paths.Preparation == nil && g.Report.Paths.Ranking == nil && o.Calls == 0, "deterministic arm inferred")
	}
	// Save the unmodified response bytes, not the normalized comparison object.
	must(os.WriteFile(filepath.Join(in.out, p.id+"-generation.json"), r.Response, 0644))
	o.GenerationSHA = hash(r.Response)
	o.Execution = execute(in.compiler, in.out, p.id+"-runtime.json", "body-execute", "--source", p.request+".gooo",
		"--path-plan", p.request+"-b8-recipe.json", "--generation", p.id+"-generation.json", "--cases", p.request+"-cases.json", "--go-bin", in.goBin)
	var actual, reference native
	raw = read(filepath.Join(in.out, p.id+"-runtime.json"))
	decode(raw, &actual)
	o.RuntimeSHA = hash(raw)
	original := read(filepath.Join(in.baseline, old.ID+"-runtime.json"))
	require(hash(original) == old.RuntimeSHA, "frozen native record changed")
	decode(original, &reference)
	actualCases, _ := json.Marshal(actual.Observation.Cases)
	referenceCases, _ := json.Marshal(reference.Observation.Cases)
	require(actual.Observation.Stage == "COMPLETE" && len(actual.Observation.Runs) == 2 && len(actual.Observation.Cases) == 8 && bytes.Equal(actualCases, referenceCases), "native outcomes changed")
	o.NativeRuns = len(actual.Observation.Runs)
	for _, c := range actual.Observation.Cases {
		o.Total++
		if c.Passed {
			o.Passed++
		}
	}
	return o
}

func (in inputs) run(requests []string, arm string, workers int) []observation {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	args := []string{"--workers", fmt.Sprint(workers)}
	if arm == "model" {
		args = append(args, "--model", in.model)
	}
	c := exec.CommandContext(ctx, in.worker, args...)
	c.WaitDelay = 5 * time.Second
	stdin, err := c.StdinPipe()
	must(err)
	stdout, err := c.StdoutPipe()
	must(err)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	must(c.Start())
	defer func() {
		cancel()
		_ = stdin.Close()
		if c.ProcessState == nil {
			_ = c.Wait()
		}
	}()
	encoder, decoder := json.NewEncoder(stdin), json.NewDecoder(bufio.NewReader(stdout))
	prefix := fmt.Sprintf("%s-w%d", arm, workers)
	var sequence uint64
	send := func(request, phase string, trial int) pending {
		sequence++
		p := pending{id: fmt.Sprintf("%s-%02d", prefix, sequence), request: request, phase: phase, trial: trial, sequence: sequence, started: time.Now()}
		must(encoder.Encode(map[string]any{"schema": "gooo/native-body-stream-request/v1", "correlation_id": p.id,
			"source": string(read(filepath.Join(in.out, request+".gooo"))), "activity": "Compose", "document": json.RawMessage(read(filepath.Join(in.out, request+"-b8-recipe.json")))}))
		return p
	}
	var records []observation
	receive := func(waiting map[string]pending) {
		var r response
		must(decoder.Decode(&r))
		p, ok := waiting[r.CorrelationID]
		require(ok, "duplicate or unknown correlation")
		delete(waiting, r.CorrelationID)
		records = append(records, in.executeResponse(r, p, arm, workers))
	}
	for _, request := range requests {
		for trial := range 2 {
			p := send(request, "sequential", trial)
			receive(map[string]pending{p.id: p})
		}
	}
	// Four in-flight records, only two source plans. Responses may be reordered.
	// Reading a response immediately triggers native execution before the next read.
	for _, start := range []int{0, 6} {
		waiting := map[string]pending{}
		for i := range 4 {
			p := send(requests[start+i%2], "batch", i)
			waiting[p.id] = p
		}
		for len(waiting) > 0 {
			receive(waiting)
		}
	}
	must(stdin.Close())
	must(c.Wait())
	must(os.WriteFile(filepath.Join(in.out, prefix+"-setup.json"), stderr.Bytes(), 0644))
	var info struct {
		Loaded  bool
		Weights string `json:"weights_sha256"`
		Bytes   int    `json:"resident_tensor_bytes"`
	}
	decode(stderr.Bytes(), &info)
	require(info.Loaded == (arm == "model") && (arm != "model" || info.Weights == weightsSHA && info.Bytes == 16384), "worker setup differs")
	require(len(records) == 24, "worker request count differs")
	save(filepath.Join(in.out, prefix+"-records.json"), records)
	fmt.Printf("completed %s: generations=%d native_runs=%d\n", prefix, len(records), len(records)*2)
	return records
}

func main() {
	var in inputs
	flag.StringVar(&in.worker, "worker", "", "clean Gooo worker")
	flag.StringVar(&in.compiler, "compiler", "", "same-revision Gooo compiler")
	flag.StringVar(&in.revision, "compiler-sha", "", "exact compiler revision")
	flag.StringVar(&in.goBin, "go-bin", "go", "Go 1.27.1 binary")
	flag.StringVar(&in.baseline, "baseline", "", "frozen native observations")
	flag.StringVar(&in.model, "model", "", "fixed public model.json")
	flag.StringVar(&in.out, "out", "", "fresh output directory")
	flag.Parse()
	require(len(in.revision) == 40 && in.worker != "" && in.compiler != "" && in.baseline != "" && in.model != "" && in.out != "" && flag.NArg() == 0, "pinned inputs required")
	b, ok := debug.ReadBuildInfo()
	require(ok && b.GoVersion == "go1.27.1", "pinned collector build required")
	var collector string
	clean := false
	for _, s := range b.Settings {
		if s.Key == "vcs.revision" {
			collector = s.Value
		}
		if s.Key == "vcs.modified" {
			clean = s.Value == "false"
		}
	}
	require(clean && len(collector) == 40, "clean collector required")
	binaryIdentity(in.goBin, in.worker, in.revision)
	binaryIdentity(in.goBin, in.compiler, in.revision)
	require(hash(read(filepath.Join(filepath.Dir(in.model), "weights.bin"))) == weightsSHA, "weights differ")
	raw := read(filepath.Join(in.baseline, "records.json"))
	require(hash(raw) == baselineSHA, "baseline differs")
	var priors []prior
	decode(raw, &priors)
	in.priors = map[string]prior{}
	var requests []string
	for _, p := range priors {
		in.priors[p.ID] = p
		if p.Arm == "model" && p.Budget == 8 && strings.HasPrefix(p.Request, "new-template-") && strings.HasSuffix(p.Request, "-s0-w1-t2") {
			requests = append(requests, p.Request)
		}
	}
	sort.Strings(requests)
	require(len(requests) == 8, "cohort differs")
	_, err := os.Stat(in.out)
	require(os.IsNotExist(err), "fresh output required")
	must(os.MkdirAll(in.out, 0755))
	for _, request := range requests {
		for _, suffix := range []string{".gooo", "-b8-recipe.json", "-cases.json"} {
			must(os.WriteFile(filepath.Join(in.out, request+suffix), read(filepath.Join(in.baseline, request+suffix)), 0644))
		}
	}
	var records []observation
	for _, workers := range []int{1, 4} {
		arms := []string{"deterministic", "model"}
		if workers == 4 {
			arms[0], arms[1] = arms[1], arms[0]
		}
		for _, arm := range arms {
			records = append(records, in.run(requests, arm, workers)...)
		}
	}
	save(filepath.Join(in.out, "records.json"), records)
	calls, runs := 0, 0
	for _, r := range records {
		calls += r.Calls
		runs += r.NativeRuns
	}
	require(len(records) == 96 && calls == 48 && runs == 192, "completed cohort counts differ")
	save(filepath.Join(in.out, "manifest.json"), map[string]any{"schema": "gooo/prepared-worker-native/v1", "compiler_sha": in.revision, "collector_sha": collector,
		"compiler_binary_sha256": hash(read(in.compiler)), "worker_binary_sha256": hash(read(in.worker)), "baseline_records_sha256": baselineSHA,
		"model_weights_sha256": weightsSHA, "requests": 8, "generations": len(records), "model_predictions": calls, "native_runs": runs, "training_updates": 0,
		"scope": "Known development examples through existing NDJSON worker; 1/4 workers, model/deterministic, sequential open-stdin responses and bounded four-request batches. Every received result immediately executes twice. Batch response time includes waiting while previous responses execute; no batch latency speedup claim."})
}
