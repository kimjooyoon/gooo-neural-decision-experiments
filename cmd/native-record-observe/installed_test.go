package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

const installedSource = "b629a664ea4a667945ab5f894557f42e97cafa35"

func TestInstalledRecordControlsMatchCandidateProgram(t *testing.T) {
	calls, outputs, fields := 0, 0, 0
	for _, mode := range []string{"model", "deterministic"} {
		for stage := range 3 {
			for _, saved := range []bool{false, true} {
				stem := fmt.Sprintf("%s-%d", mode, stage)
				if saved {
					stem += "-replay"
				}
				row, actual, err := readObservation(observationFile(t, "installed/"+stem), installedSource, mode, stage, saved, resources{})
				if err != nil {
					t.Fatal(err)
				}
				_, candidate, err := readObservation(observationFile(t, stem), candidateSource, mode, stage, saved, resources{})
				if err != nil {
					t.Fatal(err)
				}
				if !sameProgram(candidate, actual) {
					t.Fatalf("%s program differs after installation", stem)
				}
				calls += row.ModelCalls
				outputs += row.RuntimePassed
				fields += row.Fields
			}
		}
		if _, _, err := readObservation(observationFile(t, "installed/parallel-"+mode), installedSource, mode, 0, false, resources{}); err != nil {
			t.Fatal(err)
		}
		if _, _, err := readObservationCases(observationFile(t, "installed/additional-"+mode), installedSource, mode, 0, true, resources{}, 8); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 || outputs != 432 || fields != 1152 {
		t.Fatalf("installed counts %d/%d/%d", calls, outputs, fields)
	}
	if err := checkPartial(observationFile(t, "installed/partial"), installedSource); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledExternalRecordKeepsOriginalInputAndPassesReplacement(t *testing.T) {
	for _, name := range []string{"external", "external-replay"} {
		var capture envelope
		if err := json.Unmarshal(observationFile(t, "installed/"+name), &capture); err != nil {
			t.Fatal(err)
		}
		c, r := capture.Composition, capture.Runtime
		if c.Stage != "COMPLETE" || r.Stage != "COMPLETE" || r.Passed != 8 || r.Total != 8 || len(r.Traces) != 4 || len(r.Runs) != 2 || r.Calls != 0 || len(c.Plan.Records) != 1 {
			t.Fatal("external record scope differs")
		}
		for _, step := range c.Steps {
			if step.Generation.Report.Compiler != installedSource || step.Generation.Report.Paths != nil {
				t.Fatal("external generation source or inference differs")
			}
		}
		for _, run := range r.Runs {
			if !run.Completed || run.Exit == nil || *run.Exit != 0 {
				t.Fatal("external native run incomplete")
			}
		}
		var row summary
		for _, trace := range r.Traces {
			if err := checkTrace(trace.Deliveries, c.Plan.Activities, c.Plan.Records, &row); err != nil {
				t.Fatal(err)
			}
			d := trace.Deliveries[0]
			var inputState, outputState string
			if json.Unmarshal(d.InputFields[1].Value, &inputState) != nil || json.Unmarshal(d.ActualFields[1].Value, &outputState) != nil || outputState != inputState+"!" || d.InputFields[0].ID != "recordinput://name" || d.InputFields[1].ID != "recordinput://state" {
				t.Fatal("copy/local replacement differs")
			}
		}
		if row.Fields != 24 || row.InputSlots != 8 || row.Deliveries != 4 {
			t.Fatal(row)
		}
	}
}

func TestInstalledCompilerReplaysPreviousScalarJoinCompositions(t *testing.T) {
	for _, mode := range []string{"model", "deterministic"} {
		var capture envelope
		if err := json.Unmarshal(observationFile(t, "installed/legacy-"+mode+"-replay"), &capture); err != nil {
			t.Fatal(err)
		}
		r := capture.Runtime
		if capture.GeneratedNow || r.Source != installedSource || r.Stage != "COMPLETE" || !r.Replayed || r.Calls != 0 || r.Passed != 49 || r.Total != 49 || len(r.Runs) != 2 {
			t.Fatal("previous scalar replay differs")
		}
	}
}
