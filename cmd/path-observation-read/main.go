// Read the fixed small pilot without invoking a model, compiler or generated code.
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

type object = map[string]any
type finiteCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}
type aggregate struct {
	Mode                string    `json:"mode"`
	Model               bool      `json:"model"`
	Requests            int       `json:"requests"`
	RuntimePassed       int       `json:"runtime_passed"`
	RuntimeCases        int       `json:"runtime_cases"`
	Predictions         int       `json:"model_predictions"`
	Attempts            int       `json:"search_attempts"`
	CodegenMS           []float64 `json:"codegen_ms_samples"`
	CPU                 []float64 `json:"cpu_one_core_percent_samples"`
	RSS                 []float64 `json:"rss_mib_samples"`
	ObservationMS       []float64 `json:"observation_ms_samples"`
	MedianCodegenMS     float64   `json:"median_codegen_ms"`
	MedianCPU           float64   `json:"median_cpu_one_core_percent"`
	MedianRSS           float64   `json:"median_rss_mib"`
	MedianObservationMS float64   `json:"median_observation_ms"`
}

func check(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func read(name string) []byte {
	b, e := os.ReadFile(name)
	if e != nil {
		panic(e)
	}
	return b
}
func decode(b []byte) object {
	var v object
	if e := json.Unmarshal(b, &v); e != nil {
		panic(e)
	}
	privacy(v)
	return v
}
func obj(v any) object { return v.(map[string]any) }
func list(v any) []any {
	if v == nil {
		return nil
	}
	return v.([]any)
}
func num(v any) int {
	if v == nil {
		return 0
	}
	return int(v.(float64))
}
func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func canonical(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func median(v []float64) float64 {
	v = append([]float64(nil), v...)
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

var private = regexp.MustCompile(`/Users/|/home/|/var/folders/|hf_[A-Za-z0-9]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|-----BEGIN .*PRIVATE KEY-----`)

func privacy(v any) {
	switch x := v.(type) {
	case string:
		check(!private.MatchString(x), "private pattern in decoded publication")
	case []any:
		for _, a := range x {
			privacy(a)
		}
	case map[string]any:
		for k, a := range x {
			privacy(a)
			if k == "parent_receipt_bytes" {
				b, e := base64.StdEncoding.DecodeString(a.(string))
				check(e == nil, "parent encoding")
				decode(b)
			}
		}
	}
}

// Independent ordinary arithmetic for the authored three-subtraction task.
func evaluate(mask int, input int64) int64 {
	v := input
	for i, c := range []int64{1, 3, 9} {
		if mask&(1<<i) == 0 {
			v -= c
		} else {
			v = c - v
		}
	}
	return v
}

func observations(p object, mode string) []finiteCase {
	cases := []finiteCase{{10, -3}}
	check(p["test_suite_sha256"] == "sha256:"+sha(canonical(cases)), "initial suite digest")
	if mode == "none" {
		check(p["observation"] == nil, "unexpected probes")
		return cases
	}
	o := obj(p["observation"])
	rounds := list(o["rounds"])
	check(o["source_sha256"] == p["original_source_sha256"], "oracle source binding")
	check(len(rounds) == 1 || mode == "oracle" && len(rounds) == 2, "round bound")
	for _, item := range rounds {
		r := obj(item)
		q := obj(r["ranking"])
		check(q["cases_sha256"] == sha(canonical(cases)) && num(q["model_predictions"]) == 0, "probe suite/model count")
		check(num(q["observed_combinations"]) == 8 && num(q["unobserved_combinations"]) == 0, "candidate enumeration")
		var survivors []int
		for mask := 0; mask < 8; mask++ {
			matches := true
			for _, c := range cases {
				matches = matches && evaluate(mask, c.Input) == c.Expected
			}
			if matches {
				survivors = append(survivors, mask)
			}
		}
		check(string(canonical(survivors)) == string(canonical(q["surviving_masks"])), "survivors differ")
		for i, pv := range list(q["probes"]) {
			probe := obj(pv)
			input := int64(num(probe["input"]))
			check(input == []int64{0, 3, -1}[i], "probe input")
			outputs := list(probe["candidate_outputs"])
			check(len(outputs) == len(survivors), "output count")
			for j, m := range survivors {
				check(int64(num(outputs[j])) == evaluate(m, input), "candidate output differs")
			}
		}
		if r["oracle_observation"] != nil {
			c := obj(r["oracle_observation"])
			input := int64(num(c["input"]))
			expected := int64(num(c["expected"]))
			check(input == 0 && expected == 7-input && num(q["recommended_probe_index"]) == 0, "oracle expectation")
			cases = append(cases, finiteCase{input, expected})
		}
	}
	check(o["effective_cases_sha256"] == "sha256:"+sha(canonical(cases)) && num(o["effective_cases"]) == len(cases), "effective suite")
	if mode == "oracle" {
		check(o["status"] == "ONE_SURVIVING_CANDIDATE" && len(cases) == 2 && num(o["oracle_evaluations"]) == 2, "oracle status")
	} else {
		check(o["status"] == "ORACLE_UNAVAILABLE" && len(cases) == 1, "unresolved oracle")
	}
	return cases
}

func main() {
	dir := flag.String("dir", "publication/path-observation-loop-20261003", "pilot directory")
	flag.Parse()
	manifest := decode(read(filepath.Join(*dir, "manifest.json")))
	var rows []object
	if e := json.Unmarshal(read(filepath.Join(*dir, "processes.json")), &rows); e != nil {
		panic(e)
	}
	check(len(rows) == 24, "fixed pilot request count")
	sourceSHA := "sha256:" + sha(read(filepath.Join(*dir, "source.gooo")))
	groups := map[string]*aggregate{}
	seen := map[string]bool{}
	predictions, passed, attempts := 0, 0, 0
	for _, row := range rows {
		id := row["id"].(string)
		check(!seen[id] && !strings.ContainsAny(id, "/\\"), "duplicate or invalid ID")
		seen[id] = true
		g := decode(read(filepath.Join(*dir, id+"-generation.json")))
		r := obj(g["report"])
		p := obj(r["body_paths"])
		s := obj(p["search"])
		check(r["compiler_source_sha"] == manifest["compiler_sha"] && p["original_source_sha256"] == sourceSHA, "compiler/source binding")
		check(r["generated_digest"] == "sha256:"+sha([]byte(g["source"].(string))), "generated bytes")
		mode := row["mode"].(string)
		useModel := row["model"].(bool)
		cases := observations(p, mode)
		check(num(p["declared_test_cases"]) == 1 && num(s["training_cases"]) == len(cases), "selection denominator")
		for _, av := range list(s["attempts"]) {
			a := obj(av)
			mask := num(a["choice_mask"])
			actuals := list(a["case_results"])
			check(len(actuals) == len(cases), "attempt case count")
			for i, cv := range actuals {
				c := obj(cv)
				want := evaluate(mask, cases[i].Input)
				check(int64(num(c["actual"])) == want && int64(num(c["expected"])) == cases[i].Expected && c["passed"] == (want == cases[i].Expected), "attempt arithmetic")
			}
		}
		selection := obj(s["selection"])
		calls := num(selection["local_model_predictions"])
		actualCalls := 0
		if useModel {
			actualCalls = num(obj(selection["three_choice_prediction"])["actual_predictions"])
		}
		for _, fv := range list(p["feedback_judgments"]) {
			actualCalls += num(obj(fv)["new_local_model_predictions"])
		}
		check(calls == actualCalls && (useModel || calls == 0) && num(selection["external_provider_calls"]) == 0, "prediction accounting")
		n := decode(read(filepath.Join(*dir, id+"-runtime.json")))
		o := obj(n["observation"])
		check(o["projection_replayed"] == true && o["runtime_replayed"] == true && len(list(o["runs"])) == 2, "native execution/replay")
		parent, _ := base64.StdEncoding.DecodeString(n["parent_receipt_bytes"].(string))
		check(reflect.DeepEqual(decode(parent), obj(r["completeness_receipt"])), "immutable parent")
		count := 0
		disjoint := 0
		for _, cv := range list(o["cases"]) {
			c := obj(cv)
			input := int64(num(c["input"]))
			expected := 7 - input
			actual := int64(num(c["actual"]))
			check(int64(num(c["expected"])) == expected && c["passed"] == (actual == expected), "runtime scoring")
			if actual == expected {
				count++
			}
			found := false
			for _, known := range cases {
				found = found || known.Input == input
			}
			if !found {
				disjoint++
			}
		}
		check(len(list(o["cases"])) == 6 && num(o["selection_disjoint_inputs"]) == disjoint, "runtime denominator/disjointness")
		key := fmt.Sprintf("%s-model%t", mode, useModel)
		a := groups[key]
		if a == nil {
			a = &aggregate{Mode: mode, Model: useModel}
			groups[key] = a
		}
		a.Requests++
		a.RuntimePassed += count
		a.RuntimeCases += 6
		a.Predictions += calls
		a.Attempts += len(list(s["attempts"]))
		cost := obj(row["generation"])
		a.CodegenMS = append(a.CodegenMS, cost["wall_ms"].(float64))
		a.CPU = append(a.CPU, cost["cpu_one_core_percent"].(float64))
		a.RSS = append(a.RSS, cost["peak_rss_bytes"].(float64)/1048576)
		observationMS := 0.0
		if v := obj(p["timing"])["observation_ms"]; v != nil {
			observationMS = v.(float64)
		}
		a.ObservationMS = append(a.ObservationMS, observationMS)
		predictions += calls
		passed += count
		attempts += len(list(s["attempts"]))
	}
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var table []*aggregate
	for _, key := range keys {
		a := groups[key]
		a.MedianCodegenMS = median(a.CodegenMS)
		a.MedianCPU = median(a.CPU)
		a.MedianRSS = median(a.RSS)
		a.MedianObservationMS = median(a.ObservationMS)
		table = append(table, a)
	}
	result := object{"schema": "gooo/path-observation-pilot-reading/v1", "status": "PASS", "requests": 24, "compiled_runs": 48, "runtime_passed": passed, "runtime_cases": 144, "model_predictions": predictions, "search_attempts": attempts, "groups": table, "reader_model_calls": 0, "reader_program_executions": 0, "scope": "Independent arithmetic and accounting checks of one fixed authored pilot; timing samples are observations, not a causal benchmark."}
	output, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		panic(e)
	}
	fmt.Println(string(output))
}
