package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type NativeCase struct {
	Name     string         `json:"-"`
	Inputs   map[string]any `json:"inputs"`
	Expected map[string]any `json:"expected"`
}
type NativeSummary struct {
	ID                    string  `json:"source_view"`
	Profile               string  `json:"profile"`
	Counter               bool    `json:"counter_intent"`
	Budget                int     `json:"candidate_budget"`
	Compiler              string  `json:"compiler_source"`
	OriginalSHA           string  `json:"original_source_sha256"`
	RawSHA                string  `json:"private_raw_sha256"`
	Calls                 int     `json:"model_calls"`
	PredictNS             int64   `json:"predict_ns"`
	Attempts              int     `json:"candidate_attempts"`
	Mask                  uint16  `json:"selected_mask"`
	SelectionPassed       int     `json:"selection_cases_passed"`
	SelectionTotal        int     `json:"selection_cases_total"`
	SelectionFieldsPassed int     `json:"selection_fields_passed"`
	SelectionFieldsTotal  int     `json:"selection_fields_total"`
	RuntimePassed         int     `json:"runtime_named_passed"`
	RuntimeTotal          int     `json:"runtime_named_total"`
	FieldsPassed          int     `json:"runtime_fields_passed"`
	FieldsTotal           int     `json:"runtime_fields_total"`
	WallMS                float64 `json:"wall_ms"`
	CPUSeconds            float64 `json:"process_cpu_seconds"`
	BuildMS               float64 `json:"build_ms"`
	BuildRSS              int64   `json:"build_peak_rss_bytes"`
	RunRSS                int64   `json:"run_peak_rss_bytes_max"`
	RunCPU                float64 `json:"two_runs_cpu_seconds"`
	Mismatches            []any   `json:"mismatches"`
}

type nativeProcess struct {
	Started   bool  `json:"started"`
	Completed bool  `json:"completed"`
	Canceled  bool  `json:"canceled"`
	TimedOut  bool  `json:"timed_out"`
	Exit      int   `json:"exit_code"`
	WallNS    int64 `json:"wall_ns"`
	UserNS    int64 `json:"user_ns"`
	SystemNS  int64 `json:"system_ns"`
	RSS       int64 `json:"peak_rss_bytes"`
}

func (p nativeProcess) verify() {
	if !p.Started || !p.Completed || p.Canceled || p.TimedOut || p.Exit != 0 || p.WallNS <= 0 {
		panic("native process did not finish")
	}
}

func sameJSON(actual json.RawMessage, expected any) bool {
	var value any
	check(json.Unmarshal(actual, &value))
	a, err := json.Marshal(value)
	check(err)
	b, err := json.Marshal(expected)
	check(err)
	return bytes.Equal(a, b)
}

func nativeCases(family int, counter bool) []NativeCase {
	fields := names[family]
	cases := []NativeCase{}
	for i, c := range []struct {
		title, state, reason string
		active               bool
	}{{"새 예제", "queued", "새 사유", true}, {"Runtime", "queued", "new detail", true}, {"kept", "queued", "later", false}, {"already", "ready", "done", true}} {
		input := map[string]string{fields[0]: c.title, fields[1]: c.state, fields[2]: c.reason}
		expected := map[string]string{fields[0]: c.title, fields[1]: c.state, fields[2]: c.reason}
		if c.active && c.state != "ready" {
			if counter {
				expected[fields[0]] = c.state
				expected[fields[1]] = "wait"
				expected[fields[2]] = ":accepted" + c.reason
			} else {
				expected[fields[1]] = "ready"
				expected[fields[2]] = c.reason + ":accepted"
			}
		}
		cases = append(cases, NativeCase{fmt.Sprintf("fresh-%d", i), map[string]any{"Select.input0": input, "Select.input1": c.active}, map[string]any{"Select": expected, "Label": expected[fields[0]] + ":" + expected[fields[1]] + ":" + expected[fields[2]]}})
	}
	return cases
}

func counterSource(row Row, source []byte) []byte {
	fields := names[row.Family]
	lang := 0
	if row.Language == "en" {
		lang = 1
	}
	goals := [2][3]string{{"제목에 입력 상태 필드의 값을 사용한다.", "상태를 wait 문자열로 설정한다.", "기존 사유 앞에 :accepted를 붙인다."}, {"Use the input state field as the title.", "Set state to the wait string.", "Prefix :accepted to the original reason."}}
	text := string(source)
	for role := range 3 {
		before, _ := json.Marshal(intents[lang][role])
		after, _ := json.Marshal(goals[lang][role])
		text = strings.Replace(text, string(before), string(after), 1)
	}
	for _, c := range []struct{ title, reason string }{{"한글", "검토"}, {"English", "review"}, {"", ""}} {
		before, _ := json.Marshal(map[string]string{fields[0]: c.title, fields[1]: "ready", fields[2]: c.reason + ":accepted"})
		after, _ := json.Marshal(map[string]string{fields[0]: "queued", fields[1]: "wait", fields[2]: ":accepted" + c.reason})
		quotedBefore, _ := json.Marshal(string(before))
		quotedAfter, _ := json.Marshal(string(after))
		text = strings.Replace(text, string(quotedBefore), string(quotedAfter), 1)
	}
	return []byte(text)
}

func nativeStudy(prepared, models, compiler, compilerSource, oldModel, output string) {
	if prepared == "" || models == "" || compiler == "" || oldModel == "" || len(compilerSource) != 40 || output == "" {
		panic("complete native inputs required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh native directory required")
	}
	check(os.MkdirAll(output, 0755))
	version := invoke(compiler, "version", "--build", "--json")
	var identity struct {
		Source string `json:"compiler_source_sha"`
		Status string `json:"source_status"`
	}
	check(json.Unmarshal(version, &identity))
	if identity.Source != compilerSource || identity.Status != "CLEAN_VCS" {
		panic("native compiler identity differs")
	}
	rows := readRows(prepared)
	file, err := os.OpenFile(filepath.Join(output, "summaries.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer file.Close()
	encoder := json.NewEncoder(file)
	observed := 0
	for _, id := range []string{"f6-p0-m3-ko", "f6-p0-m3-en", "f7-p4-m5-ko", "f7-p4-m5-en"} {
		var row Row
		for _, candidate := range rows {
			if candidate.ID == id {
				row = candidate
			}
		}
		if row.ID == "" {
			panic("native source view absent")
		}
		base, err := os.ReadFile(filepath.Join(prepared, row.SourceFile))
		check(err)
		for _, counter := range []bool{false, true} {
			if counter && row.Family != 6 {
				continue
			}
			for _, profile := range []string{"deterministic", "frozen_ordinal", "fp32", "ptq_ternary", "qat_ternary"} {
				for _, budget := range []int{1, 2, 4, 8} {
					if counter && budget != 1 && budget != 8 {
						continue
					}
					stem := fmt.Sprintf("%s-%t-%s-b%d", id, counter, profile, budget)
					source := base
					if counter {
						source = counterSource(row, source)
					}
					source = []byte(strings.Replace(string(source), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1))
					sourceFile := filepath.Join(output, stem+".gooo.fixture")
					check(os.WriteFile(sourceFile, source, 0644))
					cases := nativeCases(row.Family, counter)
					caseFile := filepath.Join(output, stem+"-cases.json")
					save(caseFile, map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": cases})
					args := []string{"body-compose", "--source", sourceFile, "--cases", caseFile}
					if profile == "frozen_ordinal" {
						args = append(args, "--model", oldModel)
					} else if profile != "deterministic" {
						args = append(args, "--model", filepath.Join(models, profile, "model.json"))
					}
					ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
					command := exec.CommandContext(ctx, compiler, args...)
					var stdout, stderr bytes.Buffer
					command.Stdout = &stdout
					command.Stderr = &stderr
					started := time.Now()
					err = command.Run()
					elapsed := time.Since(started)
					cancel()
					if err != nil {
						panic(fmt.Sprintf("native generation: %v: %.1000s", err, stderr.Bytes()))
					}
					raw := stdout.Bytes()
					check(os.WriteFile(filepath.Join(output, stem+"-raw.json"), raw, 0600))
					summary := summarizeNative(raw, cases)
					summary.ID, summary.Profile, summary.Counter, summary.Budget = row.ID, profile, counter, budget
					summary.Compiler = compilerSource
					summary.OriginalSHA = hash(source)
					summary.RawSHA = hash(raw)
					summary.WallMS = float64(elapsed.Nanoseconds()) / 1e6
					summary.CPUSeconds = command.ProcessState.UserTime().Seconds() + command.ProcessState.SystemTime().Seconds()
					if (profile == "deterministic" && summary.Calls != 0) || (profile != "deterministic" && summary.Calls != 1) {
						panic("actual construction prediction count differs")
					}
					if budget == 8 && (summary.RuntimePassed != 8 || summary.FieldsPassed != 12 || summary.SelectionFieldsPassed != 15) {
						panic("full finite budget did not complete current inputs")
					}
					check(encoder.Encode(summary))
					observed++
					if observed%20 == 0 {
						fmt.Printf("Compiled and executed %d current-input graph constructions\n", observed)
					}
				}
			}
		}
	}
	if observed != 100 {
		panic("complete native construction inventory required")
	}
}

func summarizeNative(raw []byte, cases []NativeCase) NativeSummary {
	var capture struct {
		Composition struct {
			Steps []struct {
				Generation struct {
					Report struct {
						Activity   string `json:"activity"`
						ActivityID string `json:"activity_id"`
						Assembly   struct {
							Calls        int    `json:"model_calls"`
							PredictNS    int64  `json:"predict_ns"`
							Mask         uint16 `json:"selected_mask"`
							Attempts     []any  `json:"attempts"`
							Passed       int    `json:"passed"`
							Total        int    `json:"total"`
							FieldsPassed int    `json:"fields_passed"`
							FieldsTotal  int    `json:"fields_total"`
						} `json:"record_assembly"`
					} `json:"report"`
				} `json:"generation"`
			} `json:"steps"`
		} `json:"composition"`
		Runtime struct {
			Passed   int             `json:"finite_passed"`
			Total    int             `json:"finite_total"`
			Calls    int             `json:"model_calls"`
			Replayed bool            `json:"runtime_replayed"`
			Runs     []nativeProcess `json:"runs"`
			Build    nativeProcess   `json:"build"`
			Traces   []struct {
				CaseIndex  int `json:"case_index"`
				Deliveries []struct {
					ActivityID string          `json:"activity_id"`
					Actual     json.RawMessage `json:"actual"`
					Input      json.RawMessage `json:"input"`
					Producer   string          `json:"producer_id"`
					Inputs     []struct {
						Port  string          `json:"port"`
						Value json.RawMessage `json:"value"`
					} `json:"inputs"`
				} `json:"deliveries"`
			} `json:"traces"`
		} `json:"runtime"`
	}
	check(json.Unmarshal(raw, &capture))
	if len(capture.Composition.Steps) != 2 || len(capture.Runtime.Traces) != len(cases) || capture.Runtime.Calls != 0 || !capture.Runtime.Replayed || len(capture.Runtime.Runs) != 2 {
		panic("actual graph construction/replay shape differs")
	}
	a := capture.Composition.Steps[0].Generation.Report.Assembly
	summary := NativeSummary{Calls: a.Calls, PredictNS: a.PredictNS, Attempts: len(a.Attempts), Mask: a.Mask, SelectionPassed: a.Passed, SelectionTotal: a.Total, SelectionFieldsPassed: a.FieldsPassed, SelectionFieldsTotal: a.FieldsTotal, BuildMS: float64(capture.Runtime.Build.WallNS) / 1e6, Mismatches: []any{}}
	capture.Runtime.Build.verify()
	summary.BuildRSS = capture.Runtime.Build.RSS
	for _, process := range capture.Runtime.Runs {
		process.verify()
		summary.RunRSS = max(summary.RunRSS, process.RSS)
		summary.RunCPU += float64(process.UserNS+process.SystemNS) / 1e9
	}
	for i, trace := range capture.Runtime.Traces {
		if trace.CaseIndex != i || len(trace.Deliveries) != 2 {
			panic("current native trace identity differs")
		}
		for index, delivery := range trace.Deliveries {
			node := capture.Composition.Steps[index].Generation.Report.Activity
			if delivery.ActivityID != capture.Composition.Steps[index].Generation.Report.ActivityID {
				panic("native activity identity differs")
			}
			if index == 0 {
				if len(delivery.Inputs) != 2 {
					panic("current root inputs absent")
				}
				seen := map[string]bool{}
				for _, input := range delivery.Inputs {
					want, ok := cases[i].Inputs[node+"."+input.Port]
					if !ok || seen[input.Port] || !sameJSON(input.Value, want) {
						panic("native root input differs from current case")
					}
					seen[input.Port] = true
				}
			} else {
				if delivery.Producer != trace.Deliveries[0].ActivityID || !sameJSON(delivery.Input, json.RawMessage(trace.Deliveries[0].Actual)) {
					panic("native consumer did not receive producer value")
				}
			}
			var actual any
			check(json.Unmarshal(delivery.Actual, &actual))
			expected, ok := cases[i].Expected[node]
			if !ok {
				panic("unexpected native node")
			}
			actualRaw, _ := json.Marshal(actual)
			expectedRaw, _ := json.Marshal(expected)
			summary.RuntimeTotal++
			if bytes.Equal(actualRaw, expectedRaw) {
				summary.RuntimePassed++
			} else {
				summary.Mismatches = append(summary.Mismatches, map[string]any{"case": cases[i].Name, "node": node, "expected": expected, "actual": actual})
			}
			if node == "Select" {
				var record map[string]string
				check(json.Unmarshal(delivery.Actual, &record))
				for field, want := range expected.(map[string]string) {
					summary.FieldsTotal++
					if record[field] == want {
						summary.FieldsPassed++
					}
				}
			}
		}
	}
	if summary.RuntimePassed != capture.Runtime.Passed || summary.RuntimeTotal != capture.Runtime.Total {
		panic("independent named output count differs")
	}
	return summary
}

func verifyNativeDirectory(prepared, directory, output string) {
	rows := readRows(prepared)
	byID := map[string]Row{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	raw, err := os.ReadFile(filepath.Join(directory, "summaries.jsonl"))
	check(err)
	lines := bytes.Split(bytes.TrimSpace(raw), []byte{'\n'})
	if len(lines) != 100 {
		panic("complete native observations required")
	}
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, line := range lines {
		var prior NativeSummary
		check(json.Unmarshal(line, &prior))
		row, ok := byID[prior.ID]
		if !ok || row.Split != "test" {
			panic("native source view outside heldout families")
		}
		stem := fmt.Sprintf("%s-%t-%s-b%d", prior.ID, prior.Counter, prior.Profile, prior.Budget)
		capture, err := os.ReadFile(filepath.Join(directory, stem+"-raw.json"))
		check(err)
		if hash(capture) != prior.RawSHA {
			panic("native raw identity differs")
		}
		source, err := os.ReadFile(filepath.Join(directory, stem+".gooo.fixture"))
		check(err)
		if hash(source) != prior.OriginalSHA {
			panic("native source identity differs")
		}
		base, err := os.ReadFile(filepath.Join(prepared, row.SourceFile))
		check(err)
		if prior.Counter {
			base = counterSource(row, base)
		}
		base = []byte(strings.Replace(string(base), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, prior.Budget), 1))
		if !bytes.Equal(source, base) {
			panic("native source differs from declared counter/budget transformation")
		}
		observed := summarizeNative(capture, nativeCases(row.Family, prior.Counter))
		if observed.Calls != prior.Calls || observed.Mask != prior.Mask || observed.Attempts != prior.Attempts || observed.RuntimePassed != prior.RuntimePassed || observed.FieldsPassed != prior.FieldsPassed || observed.PredictNS != prior.PredictNS {
			panic("native summary does not replay")
		}
		prior.BuildRSS, prior.RunRSS, prior.RunCPU = observed.BuildRSS, observed.RunRSS, observed.RunCPU
		check(encoder.Encode(prior))
	}
}
