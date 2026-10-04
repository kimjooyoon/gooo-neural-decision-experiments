// record-update-study observes frozen model ordering on sequential Gooo values.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

type record struct {
	Title  string `json:"title"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}
type sample struct {
	input   record
	enabled bool
}
type shape struct {
	name, body, reason string
	index              [3]int
	guarded            bool
	wanted             func(record) string
}
type trial struct {
	Shape           string `json:"shape"`
	Language        string `json:"language"`
	Profile         string `json:"profile"`
	Budget          int    `json:"budget"`
	SourceSHA       string `json:"source_sha256"`
	RawSHA          string `json:"raw_sha256"`
	SelectedMask    int    `json:"selected_mask"`
	SelectionPassed int    `json:"selection_fields_passed"`
	SelectionTotal  int    `json:"selection_fields_total"`
	RuntimePassed   int    `json:"runtime_fields_passed"`
	RuntimeTotal    int    `json:"runtime_fields_total"`
	NamedPassed     int    `json:"named_outputs_passed"`
	NamedTotal      int    `json:"named_outputs_total"`
	Attempts        int    `json:"attempts"`
	ModelCalls      int    `json:"model_calls"`
	PredictNS       int64  `json:"predict_ns"`
	WallNS          int64  `json:"command_wall_ns"`
	CPUNS           int64  `json:"command_cpu_ns"`
	RSSBytes        int64  `json:"command_peak_rss_bytes"`
	Exit            int    `json:"exit_code"`
	Error           string `json:"error,omitempty"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func hash(raw []byte) string        { return fmt.Sprintf("%x", sha256.Sum256(raw)) }
func encode(v any) []byte           { b, e := json.Marshal(v); must(e); return b }
func write(path string, raw []byte) { must(os.WriteFile(path, raw, 0644)) }
func object(v any) map[string]any   { return v.(map[string]any) }
func number(v any) int              { return int(v.(float64)) }
func main() {
	cli := flag.String("gooo", "", "clean candidate CLI")
	goBin := flag.String("go-bin", "", "Go1.27.1")
	model := flag.String("model", "", "frozen QAT model.json")
	out := flag.String("out", "", "new output directory")
	flag.Parse()
	if *cli == "" || *goBin == "" || *model == "" || *out == "" {
		panic("gooo/go-bin/model/out required")
	}
	if _, e := os.Stat(*out); !os.IsNotExist(e) {
		panic("new output required")
	}
	must(os.MkdirAll(*out, 0755))
	version, e := exec.Command(*cli, "version", "--build", "--json").Output()
	must(e)
	write(filepath.Join(*out, "compiler-build.json"), version)
	weights, e := os.ReadFile(filepath.Join(filepath.Dir(*model), "weights.bin"))
	must(e)
	meta, e := os.ReadFile(*model)
	must(e)
	write(filepath.Join(*out, "model-identity.json"), encode(map[string]any{"metadata_sha256": hash(meta), "weights_sha256": hash(weights), "weight_bytes": len(weights), "model_repo": "asketeddy/gooo-record-shared-field-tiny-v1", "revision": "cd994c1b1e952924c6ce7e821fc3db25bcc81c25"}))
	var rows []trial
	for _, s := range shapes() {
		for _, language := range []string{"ko", "en"} {
			for _, profile := range []string{"deterministic", "qat_ternary"} {
				for _, budget := range []int{1, 2, 8} {
					rows = append(rows, run(*cli, *goBin, *model, *out, s, language, profile, budget))
					fmt.Printf("%s %s %s b%d: %d/%d fields, %d attempts\n", s.name, language, profile, budget, rows[len(rows)-1].RuntimePassed, rows[len(rows)-1].RuntimeTotal, rows[len(rows)-1].Attempts)
				}
			}
		}
	}
	write(filepath.Join(*out, "summary.json"), encode(map[string]any{"schema": "gooo/sequential-record-study/v1", "trials": rows, "approaches": 1, "graphs": len(rows), "scope": "six authored shapes; frozen expression-only model; CPU/wall uses one-core basis; no host utilization delta"}))
	var total int64
	must(filepath.WalkDir(*out, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			info, e := d.Info()
			if e != nil {
				return e
			}
			total += info.Size()
		}
		return nil
	}))
	write(filepath.Join(*out, "storage.json"), encode(map[string]any{"raw_bytes_before_storage_receipt": total, "registered_cap_bytes": 16 << 20, "within_cap": total < 16<<20}))
	if total >= 16<<20 {
		panic("registered storage cap exceeded; raw preserved")
	}
}
func run(cli, goBin, model, out string, s shape, language, profile string, budget int) trial {
	stem := fmt.Sprintf("%s-%s-%s-b%d", s.name, language, profile, budget)
	source, suite := fixture(s, language, budget)
	path := filepath.Join(out, stem+".gooo.fixture")
	cases := filepath.Join(out, stem+"-cases.json")
	write(path, source)
	write(cases, suite)
	args := []string{"body-compose", "--source", path, "--cases", cases, "--go-bin", goBin}
	if profile != "deterministic" {
		args = append(args, "--model", model)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cli, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	started := time.Now()
	raw, err := cmd.Output()
	elapsed := time.Since(started).Nanoseconds()
	write(filepath.Join(out, stem+"-raw.json"), raw)
	write(filepath.Join(out, stem+"-stderr.txt"), stderr.Bytes())
	r := trial{Shape: s.name, Language: language, Profile: profile, Budget: budget, SourceSHA: hash(source), RawSHA: hash(raw), WallNS: elapsed, Exit: -1}
	if p := cmd.ProcessState; p != nil {
		r.Exit = p.ExitCode()
		r.CPUNS = (p.UserTime() + p.SystemTime()).Nanoseconds()
		if u, ok := p.SysUsage().(*syscall.Rusage); ok {
			r.RSSBytes = u.Maxrss
			if runtime.GOOS != "darwin" {
				r.RSSBytes *= 1024
			}
		}
	}
	if err != nil {
		r.Error = err.Error()
		return r
	}
	var capture map[string]any
	must(json.Unmarshal(raw, &capture))
	observe(capture, &r)
	return r
}
func observe(capture map[string]any, r *trial) {
	steps := object(capture["composition"])["steps"].([]any)
	a := object(object(object(steps[0])["generation"])["report"])["record_assembly"].(map[string]any)
	r.Attempts = len(a["attempts"].([]any))
	r.SelectedMask = number(a["selected_mask"])
	r.ModelCalls = number(a["model_calls"])
	r.PredictNS = int64(a["predict_ns"].(float64))
	for _, c := range a["cases"].([]any) {
		for _, f := range object(c)["fields"].([]any) {
			field := object(f)
			r.SelectionTotal++
			if field["actual"] == field["expected"] {
				r.SelectionPassed++
			}
		}
	}
	if r.SelectionPassed != number(a["fields_passed"]) || r.SelectionTotal != number(a["fields_total"]) {
		panic("selection recount differs")
	}
	native := object(capture["runtime"])
	if number(native["model_calls"]) != 0 || native["runtime_replayed"] != true {
		panic("native replay/call observation")
	}
	for _, trace := range native["traces"].([]any) {
		for i, v := range object(trace)["deliveries"].([]any) {
			d := object(v)
			r.NamedTotal++
			if bytes.Equal(encode(d["actual"]), encode(d["expected"])) {
				r.NamedPassed++
			}
			if i == 0 {
				for key, wanted := range object(d["expected"]) {
					r.RuntimeTotal++
					if object(d["actual"])[key] == wanted {
						r.RuntimePassed++
					}
				}
			}
		}
	}
	if r.NamedPassed != number(native["finite_passed"]) || r.NamedTotal != number(native["finite_total"]) {
		panic("native recount differs")
	}
}
