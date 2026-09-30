package main

import ("encoding/json"; "os")

//gooo:generated:start id="tiny-smoke://activity/boolean-hole" kind="activity"
func BooleanHole(input int64) int64 {
	var result int64 = 0
	if input <= 0 {
		result = result + 1
	} else {
		result = result - 1
	}
	result = result + 1
	return result
}

//gooo:generated:end id="tiny-smoke://activity/boolean-hole" kind="activity"

func main() {
	cases := []struct { Input int64; Expected int64 }{
		{Input: -9223372036854775808, Expected: 2},
		{Input: -7, Expected: 2},
		{Input: -1, Expected: 2},
		{Input: 0, Expected: 2},
		{Input: 1, Expected: 0},
		{Input: 9223372036854775807, Expected: 0},
	}
	type result struct { Input int64 `json:"input"`; Expected int64 `json:"expected"`; Actual int64 `json:"actual"`; Passed bool `json:"passed"` }
	results := make([]result, 0, len(cases))
	for _, testCase := range cases {
		actual := BooleanHole(testCase.Input)
		results = append(results, result{Input: testCase.Input, Expected: testCase.Expected, Actual: actual, Passed: actual == testCase.Expected})
	}
	_ = json.NewEncoder(os.Stdout).Encode(results)
}
