// Verify downloaded CI captures without rerunning inference or generated code.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func read(root, name string) ([]byte, error) {
	info, err := os.Lstat(filepath.Join(root, name))
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 4<<20 {
		return nil, errors.New("bounded regular capture required")
	}
	return os.ReadFile(filepath.Join(root, name))
}
func run(root, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh receipt required")
	}
	raw, err := read(root, "report.json")
	if err != nil {
		return err
	}
	reportHash := hash(raw)
	var report struct {
		Runner      string `json:"runner_revision"`
		Native      string `json:"native_revision"`
		Calls       int    `json:"native_calls"`
		Predictions int    `json:"local_model_predictions"`
		External    int    `json:"external_calls"`
		Known       bool   `json:"external_calls_known"`
		Cells       []struct {
			ID           string `json:"id"`
			Budget       int    `json:"max_attempts"`
			SourceSHA    string `json:"gooo_input_sha256"`
			DocumentSHA  string `json:"input_document_sha256"`
			ReceiptSHA   string `json:"native_receipt_sha256"`
			GeneratedSHA string `json:"generated_go_sha256"`
			Passed       int    `json:"independent_gold_cases_passed"`
			Total        int    `json:"independent_gold_cases_total"`
		} `json:"cells"`
	}
	if json.Unmarshal(raw, &report) != nil || report.Runner != "6de93a8ffb6f106d187a6e6cbc592a1fa1dc82b5" || report.Native != "d21ce275ec0832e38ad4962ad03f4546c1b46e9f" || report.Calls != 32 || report.Predictions != 72 || report.External != 0 || !report.Known || len(report.Cells) != 32 {
		return errors.New("CI source/call binding failed")
	}
	seen := map[string]bool{}
	goldByID := map[string]int{}
	modelCalls := 0
	files := 0
	for _, row := range report.Cells {
		if strings.ContainsAny(row.ID, "/\\") || strings.Contains(row.ID, "..") || seen[row.ID] || row.Total != 10 {
			return errors.New("invalid CI cell")
		}
		seen[row.ID] = true
		goldByID[row.ID] = row.Passed
		for name, pin := range map[string]string{"input.gooo": row.SourceSHA, "plan.json": row.DocumentSHA, "native-stdout.json": row.ReceiptSHA, "generated.go.txt": row.GeneratedSHA} {
			data, err := read(root, row.ID+"/"+name)
			if err != nil || hash(data) != pin {
				return errors.New("CI capture SHA mismatch")
			}
			files++
		}
		data, err := read(root, row.ID+"/native-stdout.json")
		if err != nil {
			return err
		}
		var payload struct {
			Source string `json:"source"`
			Report struct {
				Decision  string `json:"decision"`
				Compiler  string `json:"compiler_source_sha"`
				Typecheck bool   `json:"typecheck_passed"`
				Replay    bool   `json:"deterministic_replay"`
				Writes    int    `json:"repository_writes"`
				Paths     struct {
					Matched bool `json:"source_base_matched"`
					Search  struct {
						Selection struct {
							Calls    int  `json:"local_model_predictions"`
							External int  `json:"external_provider_calls"`
							Known    bool `json:"external_provider_calls_known"`
						} `json:"selection"`
					} `json:"search"`
				} `json:"body_paths"`
			} `json:"report"`
		}
		if json.Unmarshal(data, &payload) != nil || payload.Report.Decision != "PASS" || payload.Report.Compiler != report.Native || !payload.Report.Typecheck || !payload.Report.Replay || payload.Report.Writes != 0 || !payload.Report.Paths.Matched || hash([]byte(payload.Source)) != row.GeneratedSHA || payload.Report.Paths.Search.Selection.External != 0 || !payload.Report.Paths.Search.Selection.Known {
			return errors.New("CI native payload failed")
		}
		modelCalls += payload.Report.Paths.Search.Selection.Calls
	}
	data, err := read(root, "independent-go-tests.jsonl")
	if err != nil {
		return err
	}
	testsSHA := hash(data)
	observed := map[string]map[int64]bool{}
	passedByID := map[string]int{}
	cases, passed, fullPassed, fullTotal, tests, packages := 0, 0, 0, 0, 0, 0
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 8192), 1<<20)
	for scanner.Scan() {
		var event struct {
			Action string
			Test   string
			Output string
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			return errors.New("invalid Go event")
		}
		if event.Action == "pass" {
			if event.Test == "" {
				packages++
			} else if event.Test == "TestIndependentObservation" {
				tests++
			}
		}
		index := strings.Index(event.Output, "GOOO_OBSERVATION ")
		if index < 0 {
			continue
		}
		var item struct {
			Cell     string `json:"cell"`
			Input    int64  `json:"input"`
			Expected int64  `json:"expected"`
			Actual   int64  `json:"actual"`
			Passed   bool   `json:"passed"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(event.Output[index+len("GOOO_OBSERVATION "):])), &item) != nil || !seen[item.Cell] {
			return errors.New("invalid CI observation")
		}
		expected := 5*item.Input + 2
		if item.Input < 0 {
			expected -= 3
		} else {
			expected += 3
		}
		if item.Expected != expected || item.Passed != (item.Actual == expected) {
			return errors.New("CI arithmetic observation differs")
		}
		if observed[item.Cell] == nil {
			observed[item.Cell] = map[int64]bool{}
		}
		if observed[item.Cell][item.Input] {
			return errors.New("duplicate CI input")
		}
		observed[item.Cell][item.Input] = true
		cases++
		if item.Passed {
			passed++
			passedByID[item.Cell]++
		}
		if strings.Contains(item.Cell, "-8-") {
			fullTotal++
			if item.Passed {
				fullPassed++
			}
		}
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	for id, want := range goldByID {
		if len(observed[id]) != 10 || passedByID[id] != want {
			return errors.New("CI per-cell gold differs")
		}
	}
	if files != 128 || cases != 320 || passed != 226 || fullTotal != 160 || fullPassed != 160 || tests != 32 || packages != 32 || modelCalls != 72 {
		return errors.New("CI raw capture accounting failed")
	}
	pre, err := read(root, "preexecution.json")
	if err != nil {
		return err
	}
	result := map[string]any{"schema": "gooo/native-typed-path-ci-capture-verification/v1", "status": "PASS", "ci_run_id": 36784825951, "ci_url": "https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36784825951", "runner_revision": report.Runner, "native_revision": report.Native, "report_sha256": reportHash, "preexecution_sha256": hash(pre), "go_events_sha256": testsSHA, "native_payload_files_verified": files, "native_calls_recorded": 32, "model_predictions_recorded": 72, "independent_cases": cases, "independent_gold_passed": passed, "budget8_gold_passed": fullPassed, "budget8_gold_total": fullTotal, "independent_parity_tests_passed": tests, "additional_model_predictions": 0, "additional_native_calls": 0, "external_calls": 0, "scope": "Downloaded exact Linux CI captures, unchanged closed fixtures/model snapshots; parity pass is distinct from finite arithmetic intent observations; no new inference or training"}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(encoded, '\n'), 0644)
}
func main() {
	root := flag.String("captures", "", "downloaded private CI capture directory")
	output := flag.String("output", "", "fresh public verification receipt")
	flag.Parse()
	if err := run(*root, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("VERIFIED 128 payload hashes, 32 native calls/72 predictions, 226/320 arithmetic observations and 160/160 budget8; 0 new inference")
}
