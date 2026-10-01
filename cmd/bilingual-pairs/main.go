package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bilingualstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
)

func run(output string) error {
	if output == "" {
		return fmt.Errorf("fresh output directory required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return fmt.Errorf("output already exists")
	}
	pairs, err := bilingualstudy.Load("data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	counts, programs := map[string]int{}, map[string]bool{}
	for _, pair := range pairs {
		if err := enc.Encode(pair); err != nil {
			return err
		}
		counts[pair.Split]++
		programs[pair.Program] = true
	}
	sources := map[string]string{}
	for _, source := range []string{"internal/bilingualstudy/pairs.go", "cmd/bilingual-pairs/main.go"} {
		raw, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		sources[source] = feedbackstudy.Hash(raw)
	}
	manifest, err := json.MarshalIndent(map[string]any{
		"schema": "gooo/bilingual-gooo-pairs/v1", "pair_count": len(pairs), "split_pairs": counts,
		"pairs_sha256": feedbackstudy.Hash(buf.Bytes()), "dataset_sha256": bilingualstudy.DatasetSHA,
		"source_sha256": sources, "reused_program_groups": len(programs), "new_independent_intentions": 0,
		"languages_in_order": [2]string{"en", "ko"},
		"scope":              "Existing synthetic adjacent templates, same Gooo fallback structure and finite soft target. Pair equality is not correctness. Development partitions reused; no new independent experiments or untouched holdout.",
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(output, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "pairs.jsonl"), buf.Bytes(), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "manifest.json"), append(manifest, '\n'), 0644)
}

func main() {
	out := flag.String("out", "", "fresh pair reference directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
