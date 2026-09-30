package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestGroupedStructuralCurriculumIsDeterministic(t *testing.T) {
	first, counts, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := generate()
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("generator is not deterministic")
	}
	if counts["train"] != 4800 || counts["calibration"] != 480 || counts["test"] != 960 {
		t.Fatalf("wrong split counts: %+v", counts)
	}
	groups, templates, configs := make(map[string]string), make(map[string]string), make(map[string]string)
	rows := bytes.Split(bytes.TrimSpace(first), []byte{'\n'})
	if len(rows) != 6240 {
		t.Fatal("wrong total row count")
	}
	for _, raw := range rows {
		var record row
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatal(err)
		}
		for _, pair := range []struct {
			values map[string]string
			key    string
		}{{groups, record.InstructionID}, {templates, record.TemplateID}, {configs, record.ConfigurationID}} {
			if old := pair.values[pair.key]; old != "" && old != record.Split {
				t.Fatal("split group leakage")
			}
			pair.values[pair.key] = record.Split
		}
	}
	if len(groups) != 2080 {
		t.Fatalf("original instruction count: %d", len(groups))
	}
}
