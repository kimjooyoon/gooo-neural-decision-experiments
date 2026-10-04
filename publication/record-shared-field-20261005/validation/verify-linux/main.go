package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const repo = "kimjooyoon/gooo-neural-decision-experiments"
const run = "37226752241"
const source = "b7ce25b4cccf128a9bbc12968cdab88ff266159a"

var root string

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func require(ok bool, msg string) {
	if !ok {
		panic(msg)
	}
}
func hash(b []byte) string           { s := sha256.Sum256(b); return fmt.Sprintf("%x", s) }
func read(p string) []byte           { b, e := os.ReadFile(p); must(e); return b }
func object(b []byte) map[string]any { var v map[string]any; must(json.Unmarshal(b, &v)); return v }
func equal(a, b any) bool            { return reflect.DeepEqual(a, b) }
func obj(v any) map[string]any       { return v.(map[string]any) }
func num(v any) int                  { return int(v.(float64)) }
func proc(v any) {
	p := obj(v)
	require(p["started"] == true && p["completed"] == true && p["timed_out"] == false && p["canceled"] == false && num(p["exit_code"]) == 0 && p["wall_ns"].(float64) > 0, "native process did not complete")
}
func main() {
	if len(os.Args) != 4 {
		panic("extracted pilot evidence root, exact Linux CI ZIP and output JSON required")
	}
	root = os.Args[1]
	id := int64(11311379960)
	digest := "sha256:3dc4281c2024fca7e88d2c1089699c80bf9941fe973728603ea2723d59979eeb"
	raw := read(os.Args[2])
	require(digest == "sha256:"+hash(raw), "archive digest differs")
	archive, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	must(e)
	members := map[string]*zip.File{}
	var uncompressed uint64
	for _, f := range archive.File {
		members[f.Name] = f
		uncompressed += f.UncompressedSize64
	}
	member := func(name string) []byte {
		f := members[name]
		require(f != nil, "member absent: "+name)
		in, e := f.Open()
		must(e)
		b, e := io.ReadAll(in)
		must(e)
		must(in.Close())
		return b
	}
	macAudit := object(read(filepath.Join(root, "audit/report.json")))
	linuxAudit := object(member("shared-audit/report.json"))
	require(num(linuxAudit["parity_records"]) == 120 && linuxAudit["max_logit_error"].(float64) == 0, "Linux numerical sample differs")
	for name, v := range obj(macAudit["models"]) {
		a, b := obj(v), obj(obj(linuxAudit["models"])[name])
		for _, key := range []string{"counts", "metadata_sha256", "weights_sha256", "weight_file_bytes", "resident_tensor_bytes", "scale_bytes"} {
			require(equal(a[key], b[key]), "model comparison differs: "+name+"/"+key)
		}
	}
	count := 0
	seen := map[string]bool{}
	scan := bufio.NewScanner(bytes.NewReader(member("shared-native-fresh/summaries.jsonl")))
	scan.Buffer(make([]byte, 4096), 1<<20)
	for scan.Scan() {
		s := object(scan.Bytes())
		stem := fmt.Sprintf("%s-%s-b%d", s["source_view"], s["profile"], num(s["candidate_budget"]))
		require(!seen[stem], "duplicate graph")
		seen[stem] = true
		prefix := "shared-native-fresh/" + stem
		require(bytes.Equal(member(prefix+".gooo.fixture"), read(filepath.Join(root, "native", stem+".gooo.fixture"))), "source differs")
		ciCases := object(member(prefix + "-cases.json"))
		macCases := object(read(filepath.Join(root, "native", stem+"-cases.json")))
		require(equal(ciCases, macCases), "cases differ")
		captured := member(prefix + "-raw.json")
		require(hash(captured) == s["private_raw_sha256"], "native capture digest differs")
		capture := object(captured)
		mac := object(read(filepath.Join(root, "native", stem+"-raw.json")))
		assembly := func(c map[string]any) map[string]any {
			step := obj(obj(c["composition"])["steps"].([]any)[0])
			return obj(obj(obj(step["generation"])["report"])["record_assembly"])
		}
		chosen, priorChosen := assembly(capture), assembly(mac)
		for _, key := range []string{"selected_mask", "ranking", "prediction", "model_calls", "cases", "fields_passed", "fields_total", "passed", "total", "model_context"} {
			require(equal(chosen[key], priorChosen[key]), "source selection meaning differs: "+key)
		}
		r := obj(capture["runtime"])
		mr := obj(mac["runtime"])
		for _, key := range []string{"traces", "finite_passed", "finite_total", "model_calls", "producer_source_sha", "original_source_sha256"} {
			require(equal(r[key], mr[key]), "runtime meaning differs: "+key)
		}
		require(num(r["model_calls"]) == 0 && r["producer_source_sha"] == "7cc80858972120acfbadb35ea1e5ffd824b5d81f", "execution inference/source differs")
		proc(r["build"])
		runs := r["runs"].([]any)
		require(len(runs) == 2, "two runs required")
		for _, p := range runs {
			proc(p)
		}
		traces := r["traces"].([]any)
		cases := ciCases["cases"].([]any)
		require(len(traces) == 4 && len(cases) == 4, "four current inputs required")
		active, activeTotal, guard, guardTotal, changed, changedTotal, named := 0, 0, 0, 0, 0, 0, 0
		for i, t := range traces {
			deliveries := obj(t)["deliveries"].([]any)
			require(len(deliveries) == 2, "two deliveries required")
			record, label := obj(deliveries[0]), obj(deliveries[1])
			c := obj(cases[i])
			inputs, expected := obj(c["inputs"]), obj(c["expected"])
			original := obj(inputs["Select.input0"])
			actual, want := obj(record["actual"]), obj(expected["Select"])
			require(equal(record["expected"], want) && equal(label["expected"], expected["Label"]), "trace expected value differs")
			delivered := record["inputs"].([]any)
			require(len(delivered) == 2 && equal(obj(delivered[0])["value"], original) && equal(obj(delivered[1])["value"], inputs["Select.input1"]), "record inputs differ")
			require(equal(label["input"], actual), "producer value did not reach next activity")
			isActive := inputs["Select.input1"] == true
			state := ""
			for _, field := range obj(delivered[0])["fields"].([]any) {
				definition := obj(field)
				if strings.HasSuffix(definition["id"].(string), "/field/1") {
					state = definition["name"].(string)
				}
			}
			require(state != "", "source-defined state field absent")
			isActive = isActive && original[state] != "ready"
			for field, value := range want {
				matches := equal(actual[field], value)
				if isActive {
					activeTotal++
					if matches {
						active++
					}
				} else {
					guardTotal++
					if matches {
						guard++
					}
				}
				if isActive && !equal(original[field], value) {
					changedTotal++
					if matches {
						changed++
					}
				}
			}
			if equal(actual, want) {
				named++
			}
			if equal(label["actual"], expected["Label"]) {
				named++
			}
		}
		require(active == num(s["active_fields_passed"]) && activeTotal == num(s["active_fields_total"]) && guard == num(s["guard_fields_passed"]) && guardTotal == num(s["guard_fields_total"]) && changed == num(s["required_changed_fields_passed"]) && changedTotal == num(s["required_changed_fields_total"]) && named == num(s["runtime_named_passed"]), "independent field/expected recount differs")
		count++
	}
	must(scan.Err())
	require(count == 360, "360 graphs required")
	report := map[string]any{"schema": "gooo/shared-field-linux-validation/v1", "run": run, "source": source, "artifact_id": id, "artifact_sha256": strings.TrimPrefix(digest, "sha256:"), "compressed_bytes": len(raw), "ci_uncompressed_bytes": uncompressed, "zip_members": len(members), "graphs_independently_recounted": count, "runs": 2 * count, "source_case_trace_semantics_equal_to_mac": count, "parity_records": 120, "maximum_logit_error": 0, "model_quality_counts_equal": true, "archive_extracted_to_disk": false, "scope": "ZIP members checked in memory against independently recounted Mac sources/cases and native values. Recomputed active/guard/changed/named outputs and checked producer delivery plus build/two-run completion. Original local raw pilot remains under64MiB; this Linux replay is a separately recorded subsequent CI phase, retained locally only as an 8MB compressed artifact."}
	out, e := json.MarshalIndent(report, "", "  ")
	must(e)
	must(os.WriteFile(os.Args[3], append(out, '\n'), 0600))
	fmt.Printf("Verified %d Linux graphs from %d zipped bytes, without extraction.\n", count, len(raw))
}
