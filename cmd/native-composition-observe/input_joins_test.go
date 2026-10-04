package main

import (
	"encoding/json"
	"os"
	"testing"
)

const joinCompiler = "5af78bcbdf5c881ff7e2bf7034be5018b524ae04"

func TestPublishedInputJoinsRetainPerPortValuesAndActualCalls(t *testing.T) {
	for _, replay := range []bool{false, true} {
		name := "model-0.json"
		if replay {
			name = "model-0-replay.json"
		}
		raw, err := os.ReadFile("../../publication/native-input-joins-20261004/" + name)
		if err != nil {
			t.Fatal(err)
		}
		row, _, err := readObservationProfile(raw, joinCompiler, "model", 0, replay, resources{}, true)
		if err != nil {
			t.Fatal(err)
		}
		calls := 1
		if replay {
			calls = 0
		}
		if row.ModelCalls != calls || row.StoredModelCalls != 1 || row.SelectionTotal != 6 || row.Deliveries != 42 || row.InputSlots != 84 || row.ObservedValues != 49 {
			t.Fatalf("input observation counts differ: %+v", row)
		}
	}
}

func TestMissingOrMisorderedInputObservationsAreRejected(t *testing.T) {
	raw, err := os.ReadFile("../../publication/native-input-joins-20261004/model-0.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"port", "entity", "value", "producer", "arity", "output", "plan", "input-order"} {
		var observation envelope
		if err := json.Unmarshal(raw, &observation); err != nil {
			t.Fatal(err)
		}
		delivery := &observation.Runtime.Traces[0].Deliveries[4] // Add, after Left and Right.
		switch change {
		case "port":
			delivery.Inputs[0].Port = "input01"
		case "entity":
			delivery.Inputs[0].Entity = ""
		case "value":
			delivery.Inputs[0].Value = json.RawMessage("null")
		case "producer":
			delivery.Inputs[0].Producer = delivery.Inputs[1].Producer
		case "arity":
			delivery.Inputs = delivery.Inputs[:1]
		case "output":
			delivery.Actual = json.RawMessage("-19")
		case "plan":
			observation.Composition.Plan.Activities = nil
		case "input-order":
			delivery.Inputs[0], delivery.Inputs[1] = delivery.Inputs[1], delivery.Inputs[0]
		}
		changed, _ := json.Marshal(observation)
		if _, _, err := readObservationProfile(changed, joinCompiler, "model", 0, false, resources{}, true); err == nil {
			t.Fatal("invalid ordered observation accepted", change)
		}
	}
}
