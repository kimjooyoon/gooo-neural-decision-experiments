package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func recountPairedNative(prepared, directory, output string) {
	if prepared == "" || directory == "" || output == "" {
		panic("paired bank, saved native directory and fresh recount output required")
	}
	rows := readPaired(prepared)
	byID := map[string]PairedRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	planned := map[string]bool{}
	for _, s := range pairedNativePlan() {
		for _, p := range []string{"deterministic", "frozen_field_v1", "fp32", "ptq_ternary", "qat_ternary"} {
			for _, b := range []int{1, 2, 8} {
				planned[fmt.Sprintf("%s-%s-b%d", s.ID(), p, b)] = true
			}
		}
	}
	file, err := os.Open(filepath.Join(directory, "summaries.jsonl"))
	check(err)
	defer file.Close()
	out, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer out.Close()
	encoder := json.NewEncoder(out)
	seen := map[string]bool{}
	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 4096), 1<<20)
	for scan.Scan() {
		var prior PairedNativeSummary
		check(json.Unmarshal(scan.Bytes(), &prior))
		stem := fmt.Sprintf("%s-%s-b%d", prior.ID, prior.Profile, prior.Budget)
		r, ok := byID[prior.ID]
		if !ok || !planned[stem] || seen[stem] || prior.Goal != r.Goal || prior.Style != r.Style || prior.Counter || prior.Compiler != "aeff3641254ac5f795fa1fb8c94bed702f10cbe4" {
			panic("paired native inventory differs")
		}
		seen[stem] = true
		if !slices.Contains([]int{1, 2, 8}, prior.Budget) {
			panic("paired budget differs")
		}
		source, err := os.ReadFile(filepath.Join(directory, stem+".gooo.fixture"))
		check(err)
		base := goalFixture(r.Family, orders[r.Permutation], r.Orientation, r.Language, r.Goal, r.Style)
		want := []byte(strings.Replace(string(base), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, prior.Budget), 1))
		if !bytes.Equal(source, want) || hash(source) != prior.OriginalSHA {
			panic("paired native source pin differs")
		}
		cases := pairedNativeCases(r.Family, r.Goal)
		caseRaw, err := os.ReadFile(filepath.Join(directory, stem+"-cases.json"))
		check(err)
		if !sameRecountJSON(caseRaw, map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases}) {
			panic("fresh paired native cases differ")
		}
		raw, err := os.ReadFile(filepath.Join(directory, stem+"-raw.json"))
		check(err)
		if hash(raw) != prior.RawSHA {
			panic("paired native capture pin differs")
		}
		actual := summarizePairedNative(raw, cases, r.Family)
		actual.ID, actual.Profile, actual.Budget, actual.Compiler, actual.Goal, actual.Style = prior.ID, prior.Profile, prior.Budget, prior.Compiler, prior.Goal, prior.Style
		actual.OriginalSHA, actual.RawSHA, actual.WallMS, actual.CPUSeconds = prior.OriginalSHA, prior.RawSHA, prior.WallMS, prior.CPUSeconds
		actualRaw, err := json.Marshal(actual)
		check(err)
		if !sameRecountJSON(actualRaw, prior) {
			panic("independent paired values/field/process recount differs")
		}
		check(encoder.Encode(actual))
	}
	check(scan.Err())
	if len(seen) != len(planned) || len(seen) != 360 {
		panic("complete paired native recount required")
	}
}

// Struct fields and decoded object keys can have different JSON ordering.
// Normalize both values recursively before comparing the actual content.
func sameRecountJSON(raw json.RawMessage, expected any) bool {
	want, err := json.Marshal(expected)
	check(err)
	var normalized any
	check(json.Unmarshal(want, &normalized))
	return sameJSON(raw, normalized)
}
