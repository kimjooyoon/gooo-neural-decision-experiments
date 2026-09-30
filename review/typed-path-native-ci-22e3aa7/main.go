package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func need(ok bool) {
	if !ok {
		panic("native CI evidence mismatch")
	}
}
func read(path string) []byte { raw, err := os.ReadFile(path); need(err == nil); return raw }
func hash(raw []byte) string  { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func main() {
	root := os.Args[1]
	reportRaw := read(filepath.Join(root, "report.json"))
	var report struct {
		Status      string `json:"status"`
		Source      string `json:"runner_source_revision"`
		Native      int    `json:"native_compiler_calls"`
		Predictions int    `json:"additional_local_model_predictions"`
		External    int    `json:"external_provider_calls"`
		Passed      int    `json:"independent_cases_passed"`
		Total       int    `json:"independent_cases_total"`
		Unique      int    `json:"unique_generated_functions"`
		ReplaySHA   string `json:"replay_stdout_sha256"`
		Cells       []struct {
			ID        string `json:"id"`
			Input     string `json:"gooo_input_sha256"`
			Generated string `json:"generated_go_sha256"`
			Receipt   string `json:"native_receipt_sha256"`
			Cases     int    `json:"independent_finite_cases"`
		} `json:"cells"`
	}
	need(json.Unmarshal(reportRaw, &report) == nil)
	need(report.Status == "PASS" && report.Source == "22e3aa741f531e65f680549a9d6b50dbf4b3c33e" && report.Native == 20 && report.Predictions == 0 && report.External == 0 && report.Passed == 240 && report.Total == 240 && report.Unique == 5 && len(report.Cells) == 20)
	expected := map[string]bool{}
	for _, arm := range []string{"offline", "fp32"} {
		for _, family := range []string{"local_reference", "assignment_target", "operand_order", "branch_layout", "root_order"} {
			for _, language := range []string{"en", "ko"} {
				expected[arm+"-"+family+"-"+language] = true
			}
		}
	}
	unique := map[string]bool{}
	cases := 0
	for _, cell := range report.Cells {
		need(expected[cell.ID])
		delete(expected, cell.ID)
		dir := filepath.Join(root, cell.ID)
		need(hash(read(filepath.Join(dir, "input.gooo"))) == cell.Input)
		generated := read(filepath.Join(dir, "generated.go.txt"))
		need(hash(generated) == cell.Generated)
		receipt := read(filepath.Join(dir, "native-stdout.json"))
		need(hash(receipt) == cell.Receipt)
		var native struct {
			Source string `json:"source"`
			Report struct {
				Decision  string `json:"decision"`
				Compiler  string `json:"compiler_source_sha"`
				Typecheck bool   `json:"typecheck_passed"`
				Replay    bool   `json:"deterministic_replay"`
			} `json:"report"`
		}
		need(json.Unmarshal(receipt, &native) == nil)
		need(native.Report.Decision == "PASS" && native.Report.Compiler == "68361e64def5457f0d0e6de972570a9885cceb96" && native.Report.Typecheck && native.Report.Replay && bytes.Equal([]byte(native.Source), generated))
		need(cell.Cases == 12)
		cases += cell.Cases
		unique[cell.Generated] = true
	}
	need(len(expected) == 0 && cases == 240 && len(unique) == 5)
	replay := read(filepath.Join(root, "go-replay-stdout.jsonl"))
	need(hash(replay) == report.ReplaySHA)
	packages, tests := 0, 0
	scanner := bufio.NewScanner(bytes.NewReader(replay))
	for scanner.Scan() {
		var event struct{ Action, Test string }
		need(json.Unmarshal(scanner.Bytes(), &event) == nil)
		need(event.Action != "fail")
		if event.Action == "pass" {
			if event.Test == "TestIndependentBehavior" {
				tests++
			} else if event.Test == "" {
				packages++
			}
		}
	}
	need(scanner.Err() == nil && packages == 20 && tests == 20)
	json.NewEncoder(os.Stdout).Encode(map[string]any{"schema": "gooo/typed-path-native-ci-capture-verification/v1", "status": "PASS", "ci_run": 36778715279, "ci_source_revision": report.Source, "report_sha256": hash(reportRaw), "native_captures_verified": 20, "exact_generated_go_payloads_verified": 20, "go_test_packages_passed": packages, "go_test_functions_passed": tests, "independent_cases_accounted": cases, "unique_generated_functions": len(unique), "additional_model_predictions": 0, "external_provider_calls": 0, "verification_native_calls": 0, "verification_model_calls": 0})
	fmt.Fprintln(os.Stderr, "native CI capture hashes and Go test events verified")
}
