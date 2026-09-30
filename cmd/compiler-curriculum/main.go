// compiler-curriculum adds compiler views while preserving the original split.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unicode/utf8"
)

type row struct {
	ID            string `json:"id"`
	Template      string `json:"template_id"`
	Configuration string `json:"configuration_id"`
	Language      string `json:"language"`
	Split         string `json:"split"`
	Text          string `json:"text"`
	Label         string `json:"label"`
}

type repair struct{ Text, Label string }

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func run() error {
	base := flag.String("base", "data/synthetic-ops-v1/dataset.jsonl", "frozen original dataset")
	out := flag.String("output", "data/compiler-prov-v2", "new output directory")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	raw, err := os.ReadFile(*base)
	if err != nil {
		return err
	}
	if digest(raw) != "af0a637320a8256cd6ebcdb3486a6885f2766441c90f0cb154c11599f3c8ca3d" {
		return fmt.Errorf("base dataset differs from frozen source")
	}
	if _, err := os.Stat(*out); !os.IsNotExist(err) {
		return fmt.Errorf("output must not exist")
	}
	var rows []row
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		var baseRow row
		if err := json.Unmarshal(scanner.Bytes(), &baseRow); err != nil {
			return err
		}
		for index, view := range []string{"intent", "gooo", "prov"} {
			item := baseRow
			item.ID += fmt.Sprintf("-view%d", index)
			item.Configuration += "/" + view
			switch view {
			case "gooo":
				input := "Integer"
				if item.Label == "and" || item.Label == "or" {
					input = "Boolean"
				}
				result := "Boolean"
				if item.Label == "add" || item.Label == "subtract" || item.Label == "multiply" {
					result = "Integer"
				}
				item.Text = fmt.Sprintf("Gooo activity Decide(%s) -> %s. Intent: %s", input, result, item.Text)
			case "prov":
				item.Text = "PROV-O: compiler activity uses an intent entity and generates an IR entity. Required behavior: " + item.Text
			}
			rows = append(rows, item)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(rows) != 6144 {
		return fmt.Errorf("base row count differs")
	}
	// These are observed failures, so they are training regressions, never holdout.
	repairs := []repair{
		{"Add the input and two before the conditional adjustment.", "add"},
		{"Use the less-than-or-equal comparison to select zero and negative inputs.", "less_equal"},
	}
	for index, item := range repairs {
		rows = append(rows, row{ID: fmt.Sprintf("compiler-repair-%d", index), Template: fmt.Sprintf("compiler-repair-%d", index),
			Configuration: "observed-native-failure", Language: "en", Split: "train", Text: item.Text, Label: item.Label})
	}
	seen := map[string]bool{}
	groups := map[string]string{}
	counts := map[string]int{}
	var encoded bytes.Buffer
	for _, item := range rows {
		if seen[item.Text] || !utf8.ValidString(item.Text) || len(item.Text) > 512 {
			return fmt.Errorf("duplicate or oversized prompt")
		}
		seen[item.Text] = true
		if prior, ok := groups[item.Template]; ok && prior != item.Split {
			return fmt.Errorf("template crosses split")
		}
		groups[item.Template] = item.Split
		counts[item.Split]++
		line, err := json.Marshal(item)
		if err != nil {
			return err
		}
		encoded.Write(line)
		encoded.WriteByte('\n')
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*out, "dataset.jsonl"), encoded.Bytes(), 0o644); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(*out, "repair-regressions.json"), repairs); err != nil {
		return err
	}
	manifest := map[string]any{
		"schema": "gooo/compiler-prov-curriculum/v2", "dataset_sha256": digest(encoded.Bytes()),
		"base_dataset_sha256": digest(raw), "rows": counts, "total": len(rows), "go_version": runtime.Version(),
		"views": []string{"intent", "gooo", "prov"}, "repair_training_rows": len(repairs),
		"split_policy":    "Original template groups retained across all three views; repair prompts are train only",
		"provenance_spec": "https://www.w3.org/TR/prov-o/", "source_material": "Public synthetic instructions and two disclosed compiler failures",
		"limitations": []string{"PROV view supplies context, not OWL reasoning labels", "No independent operation-family or domain holdout"},
	}
	if err := writeJSON(filepath.Join(*out, "manifest.json"), manifest); err != nil {
		return err
	}
	fmt.Printf("generated %d rows: train=%d calibration=%d test=%d\n", len(rows), counts["train"], counts["calibration"], counts["test"])
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
