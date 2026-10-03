// Prepare finite inputs; Gooo owns construction and actual execution.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type testCase struct {
	Input    int64 `json:"input"`
	Expected int64 `json:"expected"`
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: prepare example-directory fresh-output-directory")
	}
	base, out := os.Args[1], os.Args[2]
	if err := os.Mkdir(out, 0755); err != nil {
		panic(err)
	}
	source, err := os.ReadFile(filepath.Join(base, "source.gooo"))
	if err != nil {
		panic(err)
	}
	recipe, err := os.ReadFile(filepath.Join(base, "recipe.json"))
	if err != nil {
		panic(err)
	}
	var cases [128]testCase
	for i, input := range [4]int64{math.MinInt64, math.MinInt64 + 1, math.MaxInt64 - 1, math.MaxInt64} {
		cases[i] = testCase{input, input*2 + 1}
	}
	for i := 4; i < len(cases); i++ {
		input := int64(i - 66)
		cases[i] = testCase{input, input*2 + 1}
	}
	b, err := json.Marshal(struct {
		Schema string        `json:"schema"`
		Cases  [128]testCase `json:"cases"`
	}{"gooo/body-runtime-cases/v1", cases})
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(filepath.Join(out, "cases-128.json"), b, 0644); err != nil {
		panic(err)
	}
	for _, language := range [2]string{"ko", "en"} {
		activity := "AssembleKorean"
		if language == "en" {
			activity = "AssembleEnglish"
		}
		named := strings.ReplaceAll(string(source), "Compose", activity)
		if err = os.WriteFile(filepath.Join(out, language+"-source.gooo"), []byte(named), 0644); err != nil {
			panic(err)
		}
		raw := string(recipe)
		if language == "en" {
			raw = strings.ReplaceAll(raw, "계산 순서는 다음과 같다: 2를 곱한다. 이어서 1을 더한다.",
				"The computation order is: multiply by 2. Then add 1.")
		}
		if err = os.WriteFile(filepath.Join(out, language+"-recipe.json"), []byte(raw), 0644); err != nil {
			panic(err)
		}
	}
	fmt.Println("Prepared 128 finite int64 expectations and named Korean/English activities.")
}
