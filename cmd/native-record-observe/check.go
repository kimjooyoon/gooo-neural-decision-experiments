package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

const modelWeights = "f5af1e35288cbad938d8416dd51873d83dff369a0e0c6e7e16f6e1f75b7f429b"
const modelMetadata = "e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b"

func readObservation(raw []byte, sha, mode string, stage int, replayed bool, resource resources) (summary, envelope, error) {
	return readObservationCases(raw, sha, mode, stage, replayed, resource, 6)
}

func readObservationCases(raw []byte, sha, mode string, stage int, replayed bool, resource resources, cases int) (summary, envelope, error) {
	var observation envelope
	row := summary{Mode: mode, Stage: stage, Replayed: replayed, Resources: resource}
	if err := json.Unmarshal(raw, &observation); err != nil {
		return row, observation, err
	}
	c, r := observation.Composition, observation.Runtime
	if observation.GeneratedNow == replayed || c.Stage != "COMPLETE" || c.Failure != "" || r.Source != sha || r.Stage != "COMPLETE" || r.Failure != "" || !r.Replayed || r.Calls != 0 || len(c.Steps) != 6 || len(r.Traces) != cases || len(r.Runs) != 2 || r.Total != cases*6 || r.Passed != cases*6 {
		return row, observation, fmt.Errorf("incomplete six-activity, six-case record graph")
	}
	for _, run := range r.Runs {
		if !run.Completed || run.Exit == nil || *run.Exit != 0 {
			return row, observation, fmt.Errorf("native run incomplete")
		}
	}
	if err := checkPlan(c.Plan.Activities, c.Plan.Records); err != nil {
		return row, observation, err
	}
	assemblies := 0
	for _, step := range c.Steps {
		if step.Generation.Report.Compiler != sha {
			return row, observation, fmt.Errorf("compiler source differs")
		}
		if paths := step.Generation.Report.Paths; paths != nil {
			assemblies++
			s := paths.Search.Selection
			if s.External != 0 {
				return row, observation, fmt.Errorf("external provider call observed")
			}
			row.StoredModelCalls += s.Calls
			row.Evaluated += paths.Search.Evaluated
			row.Unattempted += paths.Search.Unattempted
			row.SelectionTotal += len(paths.Cases)
			for _, test := range paths.Cases {
				if test.Passed {
					row.SelectionPassed++
				}
			}
			if mode == "model" {
				if s.Weights != modelWeights || s.Metadata != modelMetadata || s.Prediction == nil || s.Prediction.Calls != 1 || s.Prediction.Mask != 6 || s.Prediction.NS <= 0 {
					return row, observation, fmt.Errorf("unchanged own model observation differs")
				}
				row.PredictNS += s.Prediction.NS
			} else if mode != "deterministic" || s.Prediction != nil {
				return row, observation, fmt.Errorf("unexpected deterministic prediction")
			}
		}
	}
	expectedCalls, expectedCandidates := 0, 8
	if mode == "model" {
		expectedCalls, expectedCandidates = 1, 2
	}
	if assemblies != 1 || row.StoredModelCalls != expectedCalls || row.SelectionTotal != 6 || row.SelectionPassed != 6 || row.Evaluated != expectedCandidates || row.Unattempted != 8-expectedCandidates {
		return row, observation, fmt.Errorf("finite search observation differs")
	}
	if !replayed {
		row.ModelCalls, row.GenerationMS = row.StoredModelCalls, float64(c.Elapsed)/1e6
	}
	row.RuntimeMS, row.RuntimePassed, row.RuntimeTotal, row.NativeRuns = float64(r.Elapsed)/1e6, r.Passed, r.Total, len(r.Runs)
	row.GoSHA, row.DriverSHA, row.GoooSHA = c.GoSHA, c.DriverSHA, fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(c.Gooo)))
	for _, trace := range r.Traces {
		if err := checkTrace(trace.Deliveries, c.Plan.Activities, c.Plan.Records, &row); err != nil {
			return row, observation, err
		}
	}
	if row.InputSlots != cases*9 || row.Deliveries != cases*6 || row.Fields != cases*16 {
		return row, observation, fmt.Errorf("field/port/edge counts differ")
	}
	return row, observation, nil
}

func checkPlan(nodes []activity, records []recordType) error {
	if len(nodes) != 6 || len(records) != 2 {
		return fmt.Errorf("record plan absent")
	}
	names := []string{"Score", "Propose", "Echo", "ReviewCandidate", "Label", "Same"}
	ids := []string{"score", "propose", "echo", "review-candidate", "label", "same"}
	for i, node := range nodes {
		if node.Name != names[i] || node.ID != "records://activity/"+ids[i] {
			return fmt.Errorf("source activity order/identity differs")
		}
	}
	for i, record := range records {
		name, id, fields := "Candidate", "candidate", []string{"title", "state"}
		if i == 1 {
			name, id, fields = "Review", "review", []string{"summary", "reason"}
		}
		if record.Name != name || record.ID != "records://"+id || record.GoName != fmt.Sprintf("GoooRecord%x", sha256.Sum256([]byte(record.ID))) || len(record.Fields) != 2 {
			return fmt.Errorf("record identity/layout differs")
		}
		for f, field := range record.Fields {
			if field.Name != fields[f] || field.ID != record.ID+"/"+field.Name || field.GoName != fmt.Sprintf("GoooField%x", sha256.Sum256([]byte(field.ID))) {
				return fmt.Errorf("declared field order/identity differs")
			}
		}
	}
	return nil
}

func checkTrace(trace []delivery, nodes []activity, records []recordType, row *summary) error {
	if len(trace) != len(nodes) {
		return fmt.Errorf("native trace missing activities")
	}
	for i, d := range trace {
		n := nodes[i]
		if d.Activity != n.ID || d.Passed == nil || !*d.Passed || len(d.Expected) == 0 || !bytes.Equal(d.Actual, d.Expected) {
			return fmt.Errorf("named actual output differs")
		}
		if err := checkFields(n.OutputType, d.Actual, d.ActualFields, records, row); err != nil {
			return err
		}
		if len(n.Inputs) == 0 {
			if len(d.Inputs) != 0 || len(d.Input) == 0 {
				return fmt.Errorf("single-input trace differs")
			}
			if err := checkProducer(i, n.From, d.Producer, d.Input, nodes, trace, row); err != nil {
				return err
			}
			if err := checkFields(n.InputType, d.Input, d.InputFields, records, row); err != nil {
				return err
			}
			row.InputSlots++
		} else {
			if len(d.Input) != 0 || d.Producer != "" || len(d.InputFields) != 0 || len(d.Inputs) != len(n.Inputs) {
				return fmt.Errorf("ordered input arity differs")
			}
			for p, input := range d.Inputs {
				slot := n.Inputs[p]
				if input.Port != fmt.Sprintf("input%d", p) || input.Port != slot.Port || input.Entity != slot.Entity {
					return fmt.Errorf("ordered port identity differs")
				}
				if err := checkProducer(i, slot.From, input.Producer, input.Value, nodes, trace, row); err != nil {
					return err
				}
				if err := checkFields(slot.Type, input.Value, input.Fields, records, row); err != nil {
					return err
				}
				row.InputSlots++
			}
		}
	}
	return nil
}

func checkProducer(index, from int, producer string, value json.RawMessage, nodes []activity, trace []delivery, row *summary) error {
	if len(value) == 0 || bytes.Equal(value, []byte("null")) {
		return fmt.Errorf("actual input value absent")
	}
	if from < 0 {
		if producer != "" {
			return fmt.Errorf("external input has producer")
		}
		return nil
	}
	if from >= index || producer != nodes[from].ID || !bytes.Equal(value, trace[from].Actual) {
		return fmt.Errorf("input differs from actual producer output")
	}
	row.Deliveries++
	return nil
}

func checkFields(name string, value json.RawMessage, fields []fieldValue, records []recordType, row *summary) error {
	var record *recordType
	for i := range records {
		if records[i].Name == name {
			record = &records[i]
			break
		}
	}
	if record == nil {
		if len(fields) != 0 {
			return fmt.Errorf("scalar has record fields")
		}
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(value, &object); err != nil || len(object) != len(record.Fields) || len(fields) != len(record.Fields) {
		return fmt.Errorf("record object/field observations incomplete")
	}
	for i, declared := range record.Fields {
		field := fields[i]
		var text string
		if field.ID != declared.ID || field.Name != declared.Name || !bytes.Equal(object[declared.Name], field.Value) || json.Unmarshal(field.Value, &text) != nil || bytes.Equal(field.Value, []byte("null")) {
			return fmt.Errorf("record field identity/order/value differs")
		}
		row.Fields++
	}
	return nil
}
