package main

import (
	"encoding/json"
	"fmt"
)

type selectionRecord struct {
	Passed       int               `json:"passed"`
	Total        int               `json:"total"`
	FieldsPassed int               `json:"fields_passed"`
	FieldsTotal  int               `json:"fields_total"`
	Attempts     []json.RawMessage `json:"attempts"`
	Cases        []struct {
		Actual   json.RawMessage `json:"actual"`
		Expected json.RawMessage `json:"expected"`
		Passed   bool            `json:"passed"`
	} `json:"cases"`
}

type selectionScore struct {
	Activity   string      `json:"activity"`
	ActivityID string      `json:"activity_id"`
	Cases      finiteScore `json:"cases"`
	Fields     finiteScore `json:"fields"`
	Attempts   int         `json:"evaluated_candidates"`
	Gaps       []gap       `json:"gaps"`
}

// Selection cases belong to the assembly contract. Runtime cases have their
// own denominators, so neither scope is added to the other's finite scores.
func measureSelections(source observation) ([]selectionScore, error) {
	var scores []selectionScore
	seen := make(map[string]bool)
	for _, step := range source.Composition.Steps {
		r := step.Generation.Report
		if r.Record == nil {
			continue
		}
		if seen[r.ActivityID] || len(r.Record.Cases) == 0 || len(r.Record.Attempts) == 0 {
			return nil, fmt.Errorf("record selection cases or activity incomplete")
		}
		seen[r.ActivityID] = true
		score := selectionScore{ActivityID: r.ActivityID, Attempts: len(r.Record.Attempts)}
		var names []string
		for _, node := range source.Composition.Plan.Activities {
			if node.ID != r.ActivityID {
				continue
			}
			score.Activity = node.Name
			for _, record := range source.Composition.Plan.Records {
				if record.Name == node.Output {
					for _, field := range record.Fields {
						names = append(names, field.Name)
					}
				}
			}
		}
		if score.Activity == "" || len(names) == 0 {
			return nil, fmt.Errorf("record selection declaration absent")
		}
		for i, test := range r.Record.Cases {
			var actual, expected map[string]json.RawMessage
			if json.Unmarshal(test.Actual, &actual) != nil || json.Unmarshal(test.Expected, &expected) != nil ||
				len(expected) != len(names) {
				return nil, fmt.Errorf("record selection values incomplete")
			}
			matched, err := equalValues(test.Actual, test.Expected)
			if err != nil || matched != test.Passed {
				return nil, fmt.Errorf("record selection case flag differs from actual value")
			}
			score.Cases.Total++
			if matched {
				score.Cases.Passed++
			}
			for _, name := range names {
				if expected[name] == nil {
					return nil, fmt.Errorf("record selection expected field absent")
				}
				score.Fields.Total++
				matched, err := equalValues(actual[name], expected[name])
				if err != nil {
					return nil, err
				}
				if matched {
					score.Fields.Passed++
					continue
				}
				fieldGap := gap{Case: i, Activity: score.Activity, ActivityID: score.ActivityID,
					Field: name, Status: "mismatch", Actual: actual[name], Expected: expected[name]}
				if actual[name] == nil {
					score.Fields.Unobserved++
					fieldGap.Status = "unobserved_actual"
				}
				score.Gaps = append(score.Gaps, fieldGap)
			}
		}
		if score.Cases.Passed != r.Record.Passed || score.Cases.Total != r.Record.Total ||
			score.Fields.Passed != r.Record.FieldsPassed || score.Fields.Total != r.Record.FieldsTotal {
			return nil, fmt.Errorf("record selection counts differ from actual values")
		}
		score.Cases.Percent, score.Fields.Percent = ratio(score.Cases), ratio(score.Fields)
		scores = append(scores, score)
	}
	return scores, nil
}
