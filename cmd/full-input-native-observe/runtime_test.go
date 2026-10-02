package main

import (
	"fmt"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func runtimeFixture() (map[string]any, []pathplan.TestCase) {
	parent := []byte(`{"source":"synthetic"}`)
	cases := make([]pathplan.TestCase, 24)
	observed := make([]map[string]any, 24)
	for i := range cases {
		cases[i] = pathplan.TestCase{Input: int64(i), Expected: int64(i + 1)}
		observed[i] = map[string]any{"input": i, "expected": i + 1, "actual": i + 1, "passed": true}
	}
	dimensions := []dimension{{"runtime_finite_accuracy", "PASS", 24, 24}, {"runtime_selection_disjointness", "PROGRESS", 8, 24}}
	for _, id := range []string{"runtime_source_replay", "runtime_build", "execution_boundary", "runtime_deterministic_replay", "reverse_observation_coverage"} {
		dimensions = append(dimensions, dimension{ID: id, Status: "PASS"})
	}
	process := func() map[string]any { return map[string]any{"started": true, "completed": true, "exit_code": 0} }
	return map[string]any{
		"parent_receipt_bytes": parent,
		"observation":          map[string]any{"stage": "COMPLETE", "producer_source_sha": "revision", "parent_receipt_sha256": sha(parent), "cases": observed, "build": process(), "runs": []map[string]any{process(), process()}},
		"completeness_receipt": map[string]any{"dimensions": dimensions, "aggregate_completeness_score": nil, "first_unresolved": map[string]any{"id": "permission_boundary"}},
	}, cases
}

func TestFullRuntimePreservesActualScope(t *testing.T) {
	parent := []byte(`{"source":"synthetic"}`)
	g, cases := runtimeFixture()
	if validateFullRuntime(fullBytes(t, g), parent, cases, "revision") != 24 {
		t.Fatal("actual passing count differs")
	}
	mutations := []func(map[string]any){
		func(g map[string]any) { g["observation"].(map[string]any)["producer_source_sha"] = "wrong" },
		func(g map[string]any) { g["parent_receipt_bytes"] = []byte("wrong") },
		func(g map[string]any) {
			delete(g["observation"].(map[string]any)["build"].(map[string]any), "exit_code")
		},
		func(g map[string]any) {
			delete(g["observation"].(map[string]any)["runs"].([]map[string]any)[0], "exit_code")
		},
		func(g map[string]any) { g["observation"].(map[string]any)["runs"] = []map[string]any{} },
		func(g map[string]any) {
			g["observation"].(map[string]any)["cases"].([]map[string]any)[0]["actual"] = 99
		},
		func(g map[string]any) { g["observation"].(map[string]any)["cases"].([]map[string]any)[0]["input"] = 99 },
		func(g map[string]any) {
			g["completeness_receipt"].(map[string]any)["aggregate_completeness_score"] = 100
		},
		func(g map[string]any) {
			g["completeness_receipt"].(map[string]any)["first_unresolved"] = map[string]any{"id": "none"}
		},
		func(g map[string]any) {
			g["completeness_receipt"].(map[string]any)["dimensions"].([]dimension)[1].Numerator = 24
		},
	}
	for i, mutate := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			g, cases := runtimeFixture()
			mutate(g)
			expectFullPanic(t, func() { validateFullRuntime(fullBytes(t, g), parent, cases, "revision") })
		})
	}
	g, cases = runtimeFixture()
	actual := g["observation"].(map[string]any)["cases"].([]map[string]any)[0]
	actual["actual"], actual["passed"] = 99, false
	dimensions := g["completeness_receipt"].(map[string]any)["dimensions"].([]dimension)
	dimensions[0].Numerator, dimensions[0].Status = 23, "PROGRESS"
	if validateFullRuntime(fullBytes(t, g), parent, cases, "revision") != 23 {
		t.Fatal("partial finite behavior lost")
	}
}

func TestCaptureRetainsExactOverflowPrefix(t *testing.T) {
	c := &capture{maximum: 3}
	n, err := c.Write([]byte("abcdef"))
	if n != 3 || err == nil || c.data.String() != "abc" {
		t.Fatal("overflow prefix not retained")
	}
}
