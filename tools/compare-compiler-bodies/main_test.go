package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestFrozenComparisonMetrics(t *testing.T) {
	raw, err := os.ReadFile("../../studies/body-plan-v1/cohort.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	gold := map[string]map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var r row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		gold[r.ID] = r.Oracle
	}
	for _, version := range []struct {
		Name        string
		FP, QAT, PT int
	}{{"parent", 950, 916, 679}, {"improved", 1086, 768, 638}} {
		var report capture
		if _, err := readJSON("../../runs/compiler-prov-v3-bodyplan-20261001/"+version.Name+"/report.json", &report); err != nil {
			t.Fatal(err)
		}
		metrics, err := aggregate(report, gold)
		if err != nil {
			t.Fatal(err)
		}
		if metrics["fp32"].Test.Correct != version.FP || metrics["qat_ternary"].Test.Correct != version.QAT || metrics["ptq_ternary"].Test.Correct != version.PT {
			t.Fatal("frozen metrics differ")
		}
		report.Cells = report.Cells[:1023]
		if _, err := aggregate(report, gold); err == nil {
			t.Fatal("incomplete capture accepted")
		}
	}
}
