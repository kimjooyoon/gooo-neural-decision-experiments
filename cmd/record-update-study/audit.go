package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

type totals struct {
	Graphs        int     `json:"graphs"`
	Completed     int     `json:"completed"`
	Failures      int     `json:"preflight_failures"`
	Fields        int     `json:"runtime_fields_passed"`
	FieldTotal    int     `json:"runtime_fields_total"`
	Named         int     `json:"named_outputs_passed"`
	NamedTotal    int     `json:"named_outputs_total"`
	Attempts      int     `json:"attempts"`
	WallMedian    int64   `json:"command_wall_median_ns"`
	PredictMedian int64   `json:"prediction_median_ns"`
	CPUMedian     int64   `json:"command_cpu_median_ns"`
	RSSMax        int64   `json:"command_peak_rss_max_bytes"`
	Wall          []int64 `json:"-"`
	Predict       []int64 `json:"-"`
	CPU           []int64 `json:"-"`
}

func mustJSON(raw []byte, target any) { must(json.Unmarshal(raw, target)) }
func itoa(v int) string               { return strconv.Itoa(v) }
func equalJSON(a, b any) bool         { return bytes.Equal(encode(a), encode(b)) }
func readObject(path string) map[string]any {
	b, e := os.ReadFile(path)
	must(e)
	var result map[string]any
	mustJSON(b, &result)
	return result
}
func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func audit(dir string, initial bool) map[string]any {
	summary := readObject(filepath.Join(dir, "summary.json"))
	rows := summary["trials"].([]any)
	require(len(rows) == 72, "72 planned tuples required")
	groups := map[string]*totals{}
	seen := map[string]bool{}
	var rawBytes int64
	for _, value := range rows {
		r := object(value)
		shape, language, profile := r["shape"].(string), r["language"].(string), r["profile"].(string)
		budget := number(r["budget"])
		stem := shape + "-" + language + "-" + profile + "-b" + itoa(budget)
		require(!seen[stem], "duplicate tuple")
		seen[stem] = true
		raw, e := os.ReadFile(filepath.Join(dir, stem+"-raw.json"))
		must(e)
		require(hash(raw) == r["raw_sha256"], "raw digest differs")
		source, e := os.ReadFile(filepath.Join(dir, stem+".gooo.fixture"))
		must(e)
		require(hash(source) == r["source_sha256"], "source digest differs")
		key := profile + "-b" + itoa(budget)
		g := groups[key]
		if g == nil {
			g = &totals{}
			groups[key] = g
		}
		g.Graphs++
		if number(r["exit_code"]) != 0 {
			require(initial && (shape == "record_snapshot" || shape == "scalar_snapshot"), "unexpected failed trial")
			g.Failures++
			continue
		}
		capture := readObject(filepath.Join(dir, stem+"-raw.json"))
		cases := readObject(filepath.Join(dir, stem+"-cases.json"))["cases"].([]any)
		native := object(capture["runtime"])
		require(number(native["model_calls"]) == 0 && native["runtime_replayed"] == true && native["projection_replayed"] == true, "replay observation differs")
		checkProcess(object(native["build"]))
		runs := native["runs"].([]any)
		require(len(runs) == 2, "two executions required")
		for _, p := range runs {
			checkProcess(object(p))
		}
		fields, named := recountTrace(native["traces"].([]any), cases, shape)
		require(fields == number(r["runtime_fields_passed"]) && number(r["runtime_fields_total"]) == 12 && named == number(r["named_outputs_passed"]) && number(native["finite_passed"]) == named && number(native["finite_total"]) == 8, "runtime recount differs")
		step := object(object(capture["composition"])["steps"].([]any)[0])
		a := object(object(object(step["generation"])["report"])["record_assembly"])
		calls := 0
		if profile == "qat_ternary" {
			calls = 1
		}
		require(number(a["model_calls"]) == calls, "generation call count differs")
		selection := 0
		for _, c := range a["cases"].([]any) {
			for _, f := range object(c)["fields"].([]any) {
				p := object(f)
				if p["actual"] == p["expected"] {
					selection++
				}
			}
		}
		require(selection == number(r["selection_fields_passed"]) && selection == fields && number(a["fields_total"]) == 12, "selection and native fields differ")
		if budget == 8 {
			require(fields == 12 && named == 8, "full budget did not complete")
		}
		g.Completed++
		g.Fields += fields
		g.FieldTotal += 12
		g.Named += named
		g.NamedTotal += 8
		g.Attempts += len(a["attempts"].([]any))
		g.Wall = append(g.Wall, int64(r["command_wall_ns"].(float64)))
		g.Predict = append(g.Predict, int64(a["predict_ns"].(float64)))
		g.CPU = append(g.CPU, int64(r["command_cpu_ns"].(float64)))
		g.RSSMax = max(g.RSSMax, int64(r["command_peak_rss_bytes"].(float64)))
	}
	for _, g := range groups {
		g.WallMedian = median(g.Wall)
		g.PredictMedian = median(g.Predict)
		g.CPUMedian = median(g.CPU)
	}
	must(filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			info, e := d.Info()
			if e != nil {
				return e
			}
			rawBytes += info.Size()
		}
		return nil
	}))
	build := readObject(filepath.Join(dir, "compiler-build.json"))
	require(build["source_status"] == "CLEAN_VCS" && build["compiler_source_sha"] == "f070e7743688815db7684134a42d01256ea3f9a0", "source identity differs")
	return map[string]any{"schema": "gooo/sequential-record-independent-audit/v1", "groups": groups, "raw_bytes": rawBytes, "compiler_source_sha": build["compiler_source_sha"], "source_initial_failures_retained": initial, "scope": "all goals request the second expression mask7; same authored cases used for selection/native; whole-command costs include compilation/startup"}
}
func checkProcess(p map[string]any) {
	require(p["started"] == true && p["completed"] == true && p["timed_out"] == false && p["canceled"] == false && number(p["exit_code"]) == 0, "process did not complete")
}
func recountTrace(traces, cases []any, shape string) (int, int) {
	require(len(traces) == 4 && len(cases) == 4, "four explicit cases required")
	fields, named := 0, 0
	for i, value := range traces {
		c, t := object(cases[i]), object(value)
		input := object(c["inputs"])
		original := object(input["Select.input0"])
		wanted := map[string]any{"title": original["title"], "state": "ready"}
		title, state, reason := original["title"].(string), original["state"].(string), original["reason"].(string)
		switch shape {
		case "ordered", "guarded", "scalar_snapshot":
			wanted["reason"] = title + ":ready"
		case "record_snapshot":
			wanted["reason"] = title + ":" + state
		case "self_append":
			wanted["reason"] = reason + ":accepted"
		case "second_copy":
			wanted["reason"] = state + ":" + title + reason
		default:
			panic("unexpected shape")
		}
		if shape == "guarded" && (input["Select.input1"] == false || state == "ready") {
			wanted = original
		}
		expected := object(c["expected"])
		require(equalJSON(expected["Select"], wanted), "authored expectation differs from independent source rules")
		label := wanted["title"].(string) + ":" + wanted["state"].(string) + ":" + wanted["reason"].(string)
		require(expected["Label"] == label, "label expectation differs")
		deliveries := t["deliveries"].([]any)
		require(len(deliveries) == 2, "two activities required")
		producer, consumer := object(deliveries[0]), object(deliveries[1])
		ports := producer["inputs"].([]any)
		require(len(ports) == 2 && equalJSON(object(ports[0])["value"], original) && object(ports[1])["value"] == input["Select.input1"], "current inputs differ")
		actual := object(producer["actual"])
		require(equalJSON(consumer["input"], actual), "producer delivery differs")
		require(consumer["actual"] == actual["title"].(string)+":"+actual["state"].(string)+":"+actual["reason"].(string), "native consumer result differs")
		for field, w := range wanted {
			if actual[field] == w {
				fields++
			}
		}
		if equalJSON(actual, wanted) {
			named++
		}
		if consumer["actual"] == label {
			named++
		}
	}
	return fields, named
}
func median(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	v := append([]int64(nil), values...)
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	return v[len(v)/2]
}
func pack(dir, out string) {
	file, e := os.Create(out)
	must(e)
	z := zip.NewWriter(file)
	must(filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		name, e := filepath.Rel(dir, path)
		if e != nil {
			return e
		}
		w, e := z.Create(filepath.ToSlash(name))
		if e != nil {
			return e
		}
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		_, e = io.Copy(w, f)
		closeErr := f.Close()
		if e != nil {
			return e
		}
		return closeErr
	}))
	must(z.Close())
	must(file.Close())
	reader, e := zip.OpenReader(out)
	must(e)
	defer reader.Close()
	for _, entry := range reader.File {
		f, e := entry.Open()
		must(e)
		raw, e := io.ReadAll(f)
		must(e)
		must(f.Close())
		original, e := os.ReadFile(filepath.Join(dir, filepath.FromSlash(entry.Name)))
		must(e)
		require(bytes.Equal(raw, original), "packed evidence differs")
	}
}
