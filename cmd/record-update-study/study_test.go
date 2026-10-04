package main

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisteredSourceShapesAndIndependentExpectations(t *testing.T) {
	for _, s := range shapes() {
		for _, language := range []string{"ko", "en"} {
			source, suite := fixture(s, language, 8)
			if len(source) == 0 {
				t.Fatal("source")
			}
			var decoded map[string]any
			must(json.Unmarshal(suite, &decoded))
			cases := decoded["cases"].([]any)
			var traces []any
			for _, v := range cases {
				c := object(v)
				inputs, expected := object(c["inputs"]), object(c["expected"])
				traces = append(traces, map[string]any{"deliveries": []any{
					map[string]any{"inputs": []any{map[string]any{"value": inputs["Select.input0"]}, map[string]any{"value": inputs["Select.input1"]}}, "actual": expected["Select"]},
					map[string]any{"input": expected["Select"], "actual": expected["Label"]},
				}})
			}
			fields, named := recountTrace(traces, cases, s.name)
			if fields != 12 || named != 8 {
				t.Fatal(s.name, fields, named)
			}
		}
	}
}
func TestAuditRejectsWrongCurrentInputAndConsumerValues(t *testing.T) {
	_, raw := fixture(shapes()[0], "en", 8)
	var suite map[string]any
	must(json.Unmarshal(raw, &suite))
	cases := suite["cases"].([]any)
	for _, bad := range []string{"input", "label"} {
		t.Run(bad, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("incorrect delivery accepted")
				}
			}()
			var traces []any
			for _, v := range cases {
				c := object(v)
				input, expected := object(c["inputs"]), object(c["expected"])
				original := input["Select.input0"]
				label := expected["Label"]
				if bad == "input" {
					original = map[string]any{"title": "wrong", "state": "wrong", "reason": "wrong"}
				} else {
					label = "wrong"
				}
				traces = append(traces, map[string]any{"deliveries": []any{map[string]any{"inputs": []any{map[string]any{"value": original}, map[string]any{"value": input["Select.input1"]}}, "actual": expected["Select"]}, map[string]any{"input": expected["Select"], "actual": label}}})
			}
			recountTrace(traces, cases, "ordered")
		})
	}
}
func TestEvidencePackingRetainsEveryByte(t *testing.T) {
	dir, out := t.TempDir(), filepath.Join(t.TempDir(), "raw.zip")
	for _, name := range []string{"source.gooo.fixture", "empty-stderr.txt", "capture.json"} {
		write(filepath.Join(dir, name), []byte(name+"\n한글"))
	}
	pack(dir, out)
	reader, e := zip.OpenReader(out)
	must(e)
	defer reader.Close()
	if len(reader.File) != 3 {
		t.Fatal("missing archived file")
	}
	if info, e := os.Stat(out); e != nil || info.Size() == 0 {
		t.Fatal("archive", e)
	}
}
