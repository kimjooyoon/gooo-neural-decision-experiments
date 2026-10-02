package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func enterRepo(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatal(err)
		}
	})
}
func TestAllElevenPoliciesOnSixteenActualNativeViews(t *testing.T) {
	enterRepo(t)
	models, ids, err := loadModels("models/own-three-feedback-v1")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("internal/threefeedback/testdata/native-goal7-rows.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), lineCap)
	sessions, predictions, attempts := 0, 0, 0
	var chosen observation
	var chosenView threecohort.View
	for scan.Scan() {
		v, err := threecohort.ReconstructRow(scan.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range ids {
			c, err := one(v, models[id])
			if err != nil {
				t.Fatal(err)
			}
			if err = verify(v, c, models[id]); err != nil {
				t.Fatalf("%s %s: %v", v.ID, id, err)
			}
			raw, err := json.Marshal(observation{id, c})
			if err != nil {
				t.Fatal(err)
			}
			if len(raw)+1 > lineCap {
				t.Fatal("actual representative capture exceeds line cap")
			}
			var roundTrip observation
			if err = threecohort.Decode(raw, &roundTrip); err != nil {
				t.Fatal(err)
			}
			if err = verify(v, roundTrip.Capture, models[id]); err != nil {
				t.Fatal(err)
			}
			if id == "offline" && c.Search.Selection.ModelCalls != 0 {
				t.Fatal("disconnected control made a prediction")
			}
			if chosen.Policy == "" && models[id] != nil && models[id].Three != nil && len(c.Feedback) > 0 {
				chosen, chosenView = roundTrip, v
			}
			sessions++
			predictions += c.Search.Selection.ModelCalls
			attempts += len(c.Search.Attempts)
		}
	}
	if err = scan.Err(); err != nil {
		t.Fatal(err)
	}
	if sessions != 176 || chosen.Policy == "" {
		t.Fatal("all eleven policies and actual feedback must be validated")
	}
	t.Logf("representative validation only: sessions=%d predictions=%d candidates=%d; separate from planned 11264 sessions", sessions, predictions, attempts)
	raw, _ := json.Marshal(chosen)
	mutations := []func(*observation){
		func(o *observation) { o.Capture.Seed = "post-hoc-seed" },
		func(o *observation) { o.Capture.Search.Attempts[0].Results[0].Actual++ },
		func(o *observation) { o.Capture.Progress[0].Selection.Three.Input += " future-test=PASS" },
		func(o *observation) { o.Capture.Progress[0].Selection.Three.PartSHA[1] = "missing" },
		func(o *observation) { o.Capture.Feedback[0].ModelCalls++ },
		func(o *observation) { o.Capture.Feedback[0].FirstFailure.Expected++ },
		func(o *observation) { o.Capture.Progress[1].FrontierNodes = 0 },
		func(o *observation) { o.Capture.Progress[0].Selection.MetadataSHA256 = "wrong-model" },
	}
	for i, mutate := range mutations {
		var changed observation
		if err := threecohort.Decode(raw, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		if err := verify(chosenView, changed.Capture, models[chosen.Policy]); err == nil {
			t.Fatalf("mutation %d accepted", i)
		}
	}
}

func TestRawCapStopsBeforeAnotherCallAndPreservesEarlierBytes(t *testing.T) {
	f, err := os.OpenFile(filepath.Join(t.TempDir(), "prefix.jsonl"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := storage{Used: threestudent.RawCap - receiptReserve - lineCap}
	if err = s.beforeCall(); err != nil {
		t.Fatal(err)
	}
	if err = s.write(f, []byte("complete prior record\n"), receiptReserve); err != nil {
		t.Fatal(err)
	}
	if err = s.beforeCall(); err == nil {
		t.Fatal("another potentially one-MiB capture allowed")
	}
	s.Used = threestudent.RawCap - receiptReserve
	if err = s.write(f, []byte("must not be written"), receiptReserve); err == nil {
		t.Fatal("over-cap write accepted")
	}
	stat, err := f.Stat()
	if err != nil || stat.Size() != int64(len("complete prior record\n")) {
		t.Fatal("completed prefix was changed")
	}
}
