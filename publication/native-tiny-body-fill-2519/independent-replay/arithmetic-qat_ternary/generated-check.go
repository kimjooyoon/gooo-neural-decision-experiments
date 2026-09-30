package main

import (
	"encoding/json"
	"os"
)

//gooo:generated:start id="tiny-smoke://activity/arithmetic-hole" kind="activity"
func ArithmeticHole(input int64) int64 {
	var result = (input + 2)
	if input < 0 {
		result = result + 1
	} else {
		result = result - 1
	}
	result = result + 1
	return result
}

//gooo:generated:end id="tiny-smoke://activity/arithmetic-hole" kind="activity"

func main() {
	cases := []struct { Input int64; Expected int64 }{
		{Input: -9223372036854775808, Expected: -9223372036854775804},
		{Input: -9, Expected: -5},
		{Input: -1, Expected: 3},
		{Input: 0, Expected: 2},
		{Input: 6, Expected: 8},
		{Input: 9223372036854775807, Expected: -9223372036854775807},
	}
	type result struct { Input int64 `json:"input"`; Expected int64 `json:"expected"`; Actual int64 `json:"actual"`; Passed bool `json:"passed"` }
	results := make([]result, 0, len(cases))
	for _, testCase := range cases {
		actual := ArithmeticHole(testCase.Input)
		results = append(results, result{Input: testCase.Input, Expected: testCase.Expected, Actual: actual, Passed: actual == testCase.Expected})
	}
	_ = json.NewEncoder(os.Stdout).Encode(results)
}
