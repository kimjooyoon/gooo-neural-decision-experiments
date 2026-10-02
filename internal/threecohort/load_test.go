package threecohort

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestActualNativeRowReconstruction(t *testing.T) {
	raw, err := os.ReadFile("testdata/first-native-row.json")
	if err != nil {
		t.Fatal(err)
	}
	view, err := ReconstructRow(raw)
	if err != nil {
		t.Fatal(err)
	}
	if view.ID != "chained_operands-c00-goal0-en" || view.Target.Cases != 16 || len(view.Cases) != 16 || len(view.Plan.Decisions) != 3 || view.Split != "train" {
		t.Fatal("complete native source contract lost")
	}
	var original row
	if err = Decode(raw, &original); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*row){
		func(r *row) { r.Inputs = r.Inputs[:2] },
		func(r *row) { r.Inputs[0], r.Inputs[1] = r.Inputs[1], r.Inputs[0] },
		func(r *row) { r.Inputs[0].Text += " future result=PASS" },
		func(r *row) { r.Target.Joint[0] = 0.5 },
		func(r *row) { r.DocumentSHA = "sha256:invalid" },
		func(r *row) { r.SourceSHA = "invalid" },
		func(r *row) { r.Split = "development" },
		func(r *row) { r.CaptureSHA = "" },
	} {
		var r row
		if err = Decode(raw, &r); err != nil {
			t.Fatal(err)
		}
		mutate(&r)
		changed, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = ReconstructRow(changed); err == nil {
			t.Fatal("modified source-bound row accepted")
		}
	}
	if _, err = ReconstructRow(append([]byte(`{"id":"duplicate",`), raw[1:]...)); err == nil {
		t.Fatal("duplicate JSON accepted")
	}
	if _, err = ReconstructRow(append([]byte(`{"future_result":true,`), raw[1:]...)); err == nil {
		t.Fatal("unknown JSON accepted")
	}
}

func TestFrozenDatasetRejectsPrefixAndSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dataset.jsonl")
	raw, err := os.ReadFile("testdata/first-native-row.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(path); err == nil {
		t.Fatal("incomplete collection accepted")
	}
	link := filepath.Join(dir, "link")
	if err = os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(link); err == nil {
		t.Fatal("symlink collection accepted")
	}
}
