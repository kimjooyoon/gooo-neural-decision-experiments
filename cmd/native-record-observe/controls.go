package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type finiteCase struct {
	Inputs   map[string]any `json:"inputs"`
	Expected map[string]any `json:"expected"`
}

func additionalControls(compiler, sha, model, private, public string) error {
	if err := os.Mkdir(private, 0755); err != nil {
		return err
	}
	var rows []summary
	type result struct {
		mode     string
		raw      []byte
		resource resources
		err      error
	}
	done := make(chan result, 2)
	for _, mode := range []string{"model", "deterministic"} {
		go func() {
			args := []string{"body-compose", "--source", "source.gooo.fixture", "--cases", "cases.json"}
			if mode == "model" {
				args = append(args, "--model", model)
			}
			raw, resource, err := invoke(compiler, public, private, "parallel-"+mode, args...)
			done <- result{mode, raw, resource, err}
		}()
	}
	for range 2 {
		r := <-done
		if r.err != nil {
			return r.err
		}
		row, _, err := readObservation(r.raw, sha, r.mode, 0, false, r.resource)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(public, "parallel-"+r.mode+".json"), r.raw, 0644); err != nil {
			return err
		}
		rows = append(rows, row)
	}
	extra := additionalCases()
	if err := writeCases(public, "additional-cases.json", extra); err != nil {
		return err
	}
	for _, mode := range []string{"model", "deterministic"} {
		raw, resource, err := invoke(compiler, public, private, "additional-"+mode, "body-compose", "--source", "source.gooo.fixture", "--cases", "additional-cases.json", "--composition", mode+"-0-composition.json")
		if err != nil {
			return err
		}
		row, _, err := readObservationCases(raw, sha, mode, 0, true, resource, len(extra))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(public, "additional-"+mode+".json"), raw, 0644); err != nil {
			return err
		}
		rows = append(rows, row)
	}
	baselineRaw, err := os.ReadFile(filepath.Join(public, "cases.json"))
	if err != nil {
		return err
	}
	var baseline struct {
		Cases []finiteCase `json:"cases"`
	}
	decoder := json.NewDecoder(bytes.NewReader(baselineRaw))
	decoder.UseNumber()
	if err := decoder.Decode(&baseline); err != nil {
		return err
	}
	baseline.Cases[0].Expected["Propose"].(map[string]any)["state"] = "wait"
	if err := writeCases(public, "partial-cases.json", baseline.Cases); err != nil {
		return err
	}
	raw, resource, err := invoke(compiler, public, private, "partial", "body-compose", "--source", "source.gooo.fixture", "--cases", "partial-cases.json", "--composition", "model-0-composition.json")
	if err != nil {
		return err
	}
	if err := checkPartial(raw, sha); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(public, "partial.json"), raw, 0644); err != nil {
		return err
	}
	metrics := struct {
		Schema  string    `json:"schema"`
		Scope   string    `json:"scope"`
		Rows    []summary `json:"rows"`
		Partial struct {
			Passed    int       `json:"passed"`
			Total     int       `json:"total"`
			Resources resources `json:"resources"`
		} `json:"partial"`
	}{Schema: "gooo/native-record-controls/v1", Scope: "Two fresh CLI processes started together, one per mode, original six cases; eight new runtime-only cases through saved compositions in both modes; one intentionally wrong field expectation; completion is finite evidence, not a proof against every deadlock; separate from twelve-row cohort", Rows: rows}
	metrics.Partial.Passed, metrics.Partial.Total, metrics.Partial.Resources = 35, 36, resource
	encoded, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(public, "additional-metrics.json"), append(encoded, '\n'), 0644)
}

func additionalCases() []finiteCase {
	inputs := []int64{-100, 100, 6, 8, 42, -2, 11, -10000}
	texts := []string{"계획", "later", "😀", "quote\"\\", strings.Repeat("x", 1000), "same", "", "tab\tvalue"}
	var result []finiteCase
	for i, input := range inputs {
		score, approve := int64(7)-input, i%2 == 0
		state, reason := "wait", "deferred"
		if score >= 0 {
			state = "ready"
			if approve {
				reason = "accepted"
			}
		}
		candidate := map[string]string{"title": texts[i], "state": state}
		review := map[string]string{"summary": texts[i], "reason": reason}
		result = append(result, finiteCase{
			map[string]any{"Score": input, "Propose.input1": texts[i], "ReviewCandidate.input1": approve},
			map[string]any{"Score": score, "Propose": candidate, "Echo": candidate, "ReviewCandidate": review, "Label": texts[i] + ":" + reason, "Same": true},
		})
	}
	return result
}

func writeCases(directory, name string, cases []finiteCase) error {
	raw, err := json.MarshalIndent(struct {
		Schema string       `json:"schema"`
		Cases  []finiteCase `json:"cases"`
	}{"gooo/body-composition-cases/v1", cases}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, name), append(raw, '\n'), 0644)
}

func checkPartial(raw []byte, sha string) error {
	var current envelope
	if err := json.Unmarshal(raw, &current); err != nil {
		return err
	}
	r := current.Runtime
	if current.GeneratedNow || r.Stage != "COMPLETE" || r.Failure != "" || !r.Replayed || r.Passed != 35 || r.Total != 36 || r.Calls != 0 || len(r.Traces) != 6 || len(r.Runs) != 2 {
		return fmt.Errorf("partial completeness observation differs")
	}
	for _, step := range current.Composition.Steps {
		if step.Generation.Report.Compiler != sha {
			return fmt.Errorf("partial compiler source differs")
		}
	}
	d := r.Traces[0].Deliveries[1]
	if d.Passed == nil || *d.Passed || bytes.Equal(d.Actual, d.Expected) {
		return fmt.Errorf("wrong field expectation was accepted")
	}
	var actual, expected map[string]string
	if json.Unmarshal(d.Actual, &actual) != nil || json.Unmarshal(d.Expected, &expected) != nil || actual["state"] != "ready" || expected["state"] != "wait" || actual["title"] != expected["title"] {
		return fmt.Errorf("partial record difference missing")
	}
	return nil
}
