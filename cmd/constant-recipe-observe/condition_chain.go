package main

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

func conditionFixtures(sourceRoot, revision, out string) {
	for target, origin := range map[string]string{
		"source.gooo": "examples/body-codegen/condition-chain.gooo.fixture",
		"recipe.json": "examples/body-codegen/condition-chain.json",
	} {
		c := exec.Command("git", "show", revision+":"+origin)
		c.Dir = sourceRoot
		raw, err := c.Output()
		must(err)
		must(os.WriteFile(filepath.Join(out, target), raw, 0644))
	}
	var cases []any
	for _, input := range []int64{math.MinInt64, -1, 0, 5, 10, 11, math.MaxInt64} {
		expected := max(int64(0), min(input, int64(10)))
		cases = append(cases, map[string]int64{"input": input, "expected": expected})
	}
	save(filepath.Join(out, "cases.json"), map[string]any{"schema": "gooo/body-runtime-cases/v1", "cases": cases})
}

func collectorFiles() map[string]string {
	result := map[string]string{}
	for _, name := range []string{"main.go", "condition_chain.go"} {
		result[name] = hash(read(filepath.Join("cmd/constant-recipe-observe", name)))
	}
	return result
}
