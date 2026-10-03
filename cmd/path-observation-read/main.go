// Read the fixed small pilot without invoking a model, compiler or generated code.
package main

import (
	"bytes"
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
	ProbeEvaluations    int       `json:"probe_evaluations"`
	CachedComparisons   int       `json:"cached_comparisons"`
	ReusedProbeValues   int       `json:"reused_probe_values"`
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

func firstRankingSHA(raw []byte) string {
	for _, key := range []string{"report", "body_paths", "observation", "rounds"} {
		var fields map[string]json.RawMessage
		check(json.Unmarshal(raw, &fields) == nil, "ranking container")
		raw = fields[key]
	}
	var rounds []map[string]json.RawMessage
	check(json.Unmarshal(raw, &rounds) == nil && len(rounds) > 0, "initial ranking")
	var compact bytes.Buffer
	check(json.Compact(&compact, rounds[0]["ranking"]) == nil, "ranking JSON")
	return sha(compact.Bytes())
}

func observations(p object, mode, initialSHA string) []finiteCase {
	cases := []finiteCase{{10, -3}}
	check(p["test_suite_sha256"] == "sha256:"+sha(canonical(cases)), "initial suite digest")
	if mode == "none" {
		check(p["observation"] == nil, "unexpected probes")
		return cases
	}
	o := obj(p["observation"])
	rounds := list(o["rounds"])
	check(o["source_sha256"] == p["original_source_sha256"], "oracle source binding")
	oracle := mode == "oracle" || mode == "oracle_reuse"
	check(len(rounds) == 1 || oracle && len(rounds) == 2, "round bound")
	check((obj(o["options"])["reuse_probe_outputs"] == true) == (mode == "oracle_reuse"), "reuse mode")
	for index, item := range rounds {
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
		wantEvaluations := 8*len(cases) + 3*len(survivors)
		if mode == "oracle_reuse" {
			reused := obj(r["reuse"])
			values, comparisons := 0, 0
			if index > 0 {
				wantEvaluations, values, comparisons = 0, 3*len(survivors), 2
			}
			check(num(reused["revision"]) == index && reused["initial_ranking_sha256"] == initialSHA &&
				num(reused["reused_probe_values"]) == values && num(reused["cached_comparisons"]) == comparisons &&
				num(reused["total_cached_comparisons"]) == comparisons && num(reused["total_evaluation_attempts"]) == 14, "reuse accounting")
		} else {
			check(r["reuse"] == nil, "unexpected reuse record")
		}
		check(num(q["evaluation_attempts"]) == wantEvaluations, "probe evaluation count")
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
	if oracle {
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
	repeats := num(manifest["repetitions"])
	if repeats == 0 {
		repeats = 2
	}
	check(repeats >= 2 && repeats <= 20, "repetition bound")
	modes := []string{"none", "rank_only", "oracle"}
	if manifest["include_reuse"] == true {
		modes = append(modes, "oracle_reuse")
	}
	wantRequests := 4 * len(modes) * repeats
	check(len(rows) == wantRequests && num(manifest["generations"]) == wantRequests &&
		num(manifest["compiled_runs"]) == 2*wantRequests, "fixed pilot request count")
	sourceSHA := "sha256:" + sha(read(filepath.Join(*dir, "source.gooo")))
	groups := map[string]*aggregate{}
	seen := map[string]bool{}
	predictions, passed, attempts := 0, 0, 0
	for _, row := range rows {
		id := row["id"].(string)
		check(!seen[id] && !strings.ContainsAny(id, "/\\"), "duplicate or invalid ID")
		seen[id] = true
		generationRaw := read(filepath.Join(*dir, id+"-generation.json"))
		g := decode(generationRaw)
		r := obj(g["report"])
		p := obj(r["body_paths"])
		s := obj(p["search"])
		check(r["compiler_source_sha"] == manifest["compiler_sha"] && p["original_source_sha256"] == sourceSHA, "compiler/source binding")
		check(r["generated_digest"] == "sha256:"+sha([]byte(g["source"].(string))), "generated bytes")
		mode := row["mode"].(string)
		useModel := row["model"].(bool)
		initialSHA := ""
		if mode == "oracle_reuse" {
			initialSHA = firstRankingSHA(generationRaw)
		}
		cases := observations(p, mode, initialSHA)
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
		if mode != "none" {
			for _, rv := range list(obj(p["observation"])["rounds"]) {
				round := obj(rv)
				a.ProbeEvaluations += num(obj(round["ranking"])["evaluation_attempts"])
				if round["reuse"] != nil {
					a.CachedComparisons += num(obj(round["reuse"])["cached_comparisons"])
					a.ReusedProbeValues += num(obj(round["reuse"])["reused_probe_values"])
				}
			}
		}
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
	for repeat := range repeats {
		for _, lang := range []string{"ko", "en"} {
			for _, mode := range modes {
				for _, model := range []bool{false, true} {
					check(seen[fmt.Sprintf("%s-%s-model%t-r%d", lang, mode, model, repeat)], "missing declared arm")
				}
			}
		}
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
	result := object{"schema": "gooo/path-observation-pilot-reading/v1", "status": "PASS", "requests": wantRequests, "compiled_runs": 2 * wantRequests, "runtime_passed": passed, "runtime_cases": 6 * wantRequests, "model_predictions": predictions, "search_attempts": attempts, "groups": table, "reader_model_calls": 0, "reader_program_executions": 0, "scope": "Independent arithmetic and accounting checks of one fixed authored pilot; timing samples are observations, not a causal benchmark."}
	output, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		panic(e)
	}
	fmt.Println(string(output))
}
