package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type finiteScore struct {
	Passed     int      `json:"passed"`
	Total      int      `json:"total"`
	Unobserved int      `json:"unobserved"`
	Percent    *float64 `json:"percent"`
}
type gap struct {
	Case       int             `json:"case_index"`
	Activity   string          `json:"activity"`
	ActivityID string          `json:"activity_id"`
	Field      string          `json:"field,omitempty"`
	Status     string          `json:"status"`
	Actual     json.RawMessage `json:"actual,omitempty"`
	Expected   json.RawMessage `json:"expected,omitempty"`
}
type completeness struct {
	Schema  string      `json:"schema"`
	Scope   string      `json:"scope"`
	Outputs finiteScore `json:"named_outputs"`
	Fields  finiteScore `json:"record_output_fields"`
	Gaps    []gap       `json:"gaps"`
}
type observation struct {
	Composition struct {
		Stage string `json:"stage"`
		Plan  struct {
			Activities []struct {
				Name   string `json:"name"`
				ID     string `json:"id"`
				Output string `json:"output_type"`
			} `json:"activities"`
			Records []struct {
				Name   string `json:"name"`
				Fields []struct {
					Name string `json:"name"`
				} `json:"fields"`
			} `json:"record_types"`
		} `json:"plan"`
	} `json:"composition"`
	Runtime struct {
		Stage  string `json:"stage"`
		Passed int    `json:"finite_passed"`
		Total  int    `json:"finite_total"`
		Traces []struct {
			Case       int `json:"case_index"`
			Deliveries []struct {
				ActivityID string          `json:"activity_id"`
				Actual     json.RawMessage `json:"actual,omitempty"`
				Expected   json.RawMessage `json:"expected,omitempty"`
				Passed     *bool           `json:"passed,omitempty"`
			} `json:"deliveries"`
		} `json:"traces"`
	} `json:"runtime"`
}

func measure(raw []byte) (completeness, error) {
	result := completeness{Schema: "gooo/finite-record-completeness/v1", Scope: "Observed caller expectations in this captured execution; record-output field values only; unobserved values receive no match credit; finite ratios describe the provided contract"}
	var source observation
	if err := json.Unmarshal(raw, &source); err != nil {
		return result, err
	}
	if source.Composition.Stage != "COMPLETE" || source.Runtime.Stage != "COMPLETE" || len(source.Composition.Plan.Activities) == 0 || len(source.Runtime.Traces) == 0 {
		return result, fmt.Errorf("complete composition/runtime response required")
	}
	nodes := source.Composition.Plan.Activities
	if err := checkDeclarations(source); err != nil {
		return result, err
	}
	for caseIndex, trace := range source.Runtime.Traces {
		if trace.Case != caseIndex || len(trace.Deliveries) != len(nodes) {
			return result, fmt.Errorf("case or activity trace incomplete")
		}
		for i, delivery := range trace.Deliveries {
			node := nodes[i]
			if delivery.ActivityID != node.ID {
				return result, fmt.Errorf("activity identity/order differs")
			}
			location := gap{Case: trace.Case, Activity: node.Name, ActivityID: node.ID, Actual: delivery.Actual, Expected: delivery.Expected}
			if len(delivery.Expected) == 0 {
				if delivery.Passed != nil {
					return result, fmt.Errorf("unscored output has passed flag")
				}
				result.Outputs.Unobserved++
			} else {
				result.Outputs.Total++
				if len(delivery.Actual) == 0 {
					result.Outputs.Unobserved++
				}
				matched, err := equalValues(delivery.Actual, delivery.Expected)
				if err != nil {
					return result, err
				}
				if delivery.Passed == nil || *delivery.Passed != matched {
					return result, fmt.Errorf("named expectation flag differs from actual value")
				}
				if matched {
					result.Outputs.Passed++
				} else {
					location.Status = "mismatch"
					result.Gaps = append(result.Gaps, location)
				}
			}
			for _, record := range source.Composition.Plan.Records {
				if record.Name != node.Output {
					continue
				}
				var actual, expected map[string]json.RawMessage
				if len(delivery.Actual) != 0 {
					if err := json.Unmarshal(delivery.Actual, &actual); err != nil {
						return result, err
					}
				}
				if len(delivery.Expected) != 0 {
					if err := json.Unmarshal(delivery.Expected, &expected); err != nil {
						return result, err
					}
				}
				for _, field := range record.Fields {
					fieldGap := gap{Case: trace.Case, Activity: node.Name, ActivityID: node.ID, Field: field.Name, Actual: actual[field.Name], Expected: expected[field.Name]}
					if fieldGap.Expected == nil {
						result.Fields.Unobserved++
						fieldGap.Status = "unobserved_expectation"
						result.Gaps = append(result.Gaps, fieldGap)
						continue
					}
					result.Fields.Total++
					if fieldGap.Actual == nil {
						result.Fields.Unobserved++
					}
					matched, err := equalValues(fieldGap.Actual, fieldGap.Expected)
					if err != nil {
						return result, err
					}
					if matched {
						result.Fields.Passed++
					} else {
						fieldGap.Status = "mismatch"
						if fieldGap.Actual == nil {
							fieldGap.Status = "unobserved_actual"
						}
						result.Gaps = append(result.Gaps, fieldGap)
					}
				}
			}
		}
	}
	if result.Outputs.Passed != source.Runtime.Passed || result.Outputs.Total != source.Runtime.Total {
		return result, fmt.Errorf("reported finite counts differ from observed values")
	}
	result.Outputs.Percent, result.Fields.Percent = ratio(result.Outputs), ratio(result.Fields)
	return result, nil
}

func checkDeclarations(source observation) error {
	records := make(map[string]bool)
	for _, record := range source.Composition.Plan.Records {
		if record.Name == "" || records[record.Name] || len(record.Fields) == 0 || len(record.Fields) > 16 {
			return fmt.Errorf("record declaration incomplete or repeated")
		}
		records[record.Name] = true
		fields := make(map[string]bool)
		for _, field := range record.Fields {
			if field.Name == "" || fields[field.Name] {
				return fmt.Errorf("field declaration incomplete or repeated")
			}
			fields[field.Name] = true
		}
	}
	ids := make(map[string]bool)
	for _, node := range source.Composition.Plan.Activities {
		if node.ID == "" || node.Name == "" || ids[node.ID] {
			return fmt.Errorf("activity declaration incomplete or repeated")
		}
		ids[node.ID] = true
		if node.Output != "Integer" && node.Output != "Boolean" && node.Output != "Text" && !records[node.Output] {
			return fmt.Errorf("output record declaration absent")
		}
	}
	return nil
}

func ratio(score finiteScore) *float64 {
	if score.Total == 0 {
		return nil
	}
	result := 100 * float64(score.Passed) / float64(score.Total)
	return &result
}

func equalValues(a, b json.RawMessage) (bool, error) {
	if len(a) == 0 || len(b) == 0 {
		return false, nil
	}
	first, err := decodeValue(a)
	if err != nil {
		return false, err
	}
	second, err := decodeValue(b)
	if err != nil {
		return false, err
	}
	canonicalA, err := json.Marshal(first)
	if err != nil {
		return false, err
	}
	canonicalB, err := json.Marshal(second)
	if err != nil {
		return false, err
	}
	return bytes.Equal(canonicalA, canonicalB), nil
}

func decodeValue(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("one complete JSON value required")
	}
	return value, nil
}
