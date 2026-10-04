package main

import (
	"fmt"
	"os"
	"testing"
)

const installedJoinCompiler = "1bef47fc1e7a4aca8ffb1cbd5a5aa7ded2353ce5"

func TestInstalledInputJoinControlsMatchCandidateAndActualCounts(t *testing.T) {
	var passed, total, inputs, deliveries, calls, runs int
	for _, mode := range []string{"model", "deterministic"} {
		for stage := range 3 {
			for _, replay := range []bool{false, true} {
				name := fmt.Sprintf("%s-%d", mode, stage)
				if replay {
					name += "-replay"
				}
				name += ".json"
				root := "../../publication/native-input-joins-20261004/"
				raw, err := os.ReadFile(root + "installed/" + name)
				if err != nil {
					t.Fatal(err)
				}
				row, _, err := readObservationProfile(raw, installedJoinCompiler, mode, stage, replay, resources{}, true)
				if err != nil {
					t.Fatal(name, err)
				}
				candidate, err := os.ReadFile(root + name)
				if err != nil {
					t.Fatal(err)
				}
				previous, _, err := readObservationProfile(candidate, joinCompiler, mode, stage, replay, resources{}, true)
				if err != nil {
					t.Fatal(name, err)
				}
				if row.GoooSHA != previous.GoooSHA || row.GoSHA != previous.GoSHA ||
					row.DriverSHA != previous.DriverSHA || row.ModelCalls != previous.ModelCalls ||
					row.StoredModelCalls != previous.StoredModelCalls {
					t.Fatal("installed source/code/driver or actual/stored predictions differ", name)
				}
				passed += row.RuntimePassed
				total += row.RuntimeTotal
				inputs += row.InputSlots
				deliveries += row.Deliveries
				calls += row.ModelCalls
				runs += row.NativeRuns
			}
		}
	}
	if passed != 588 || total != 588 || inputs != 1008 || deliveries != 504 || calls != 3 || runs != 24 {
		t.Fatalf("installed aggregate differs: %d/%d inputs=%d deliveries=%d calls=%d runs=%d",
			passed, total, inputs, deliveries, calls, runs)
	}
}
