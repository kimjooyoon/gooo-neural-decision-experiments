package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "publication", "native-record-values-20261004", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCompleteAndPartialFiniteScores(t *testing.T) {
	for _, test := range []struct {
		name   string
		passed int
	}{{"model-0", 36}, {"partial", 35}} {
		r, err := measure(fixture(t, test.name))
		if err != nil {
			t.Fatal(err)
		}
		if r.Outputs.Passed != test.passed || r.Outputs.Total != 36 || r.Fields.Passed != test.passed || r.Fields.Total != 36 || r.Fields.Unobserved != 0 || r.Fields.Percent == nil {
			t.Fatal(r)
		}
		if test.name == "partial" {
			if len(r.Gaps) != 2 || r.Gaps[1].Activity != "Propose" || r.Gaps[1].Field != "state" || string(r.Gaps[1].Actual) != `"ready"` || string(r.Gaps[1].Expected) != `"wait"` {
				t.Fatal(r.Gaps)
			}
		}
	}
}

func TestUnprovidedRecordExpectationsReceiveNoCredit(t *testing.T) {
	var source observation
	if err := json.Unmarshal(fixture(t, "model-0"), &source); err != nil {
		t.Fatal(err)
	}
	for c := range source.Runtime.Traces {
		for _, i := range []int{1, 2, 3} {
			d := &source.Runtime.Traces[c].Deliveries[i]
			d.Expected, d.Passed = nil, nil
		}
	}
	source.Runtime.Passed, source.Runtime.Total = 18, 18
	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	r, err := measure(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Outputs.Unobserved != 18 || r.Fields.Passed != 0 || r.Fields.Total != 0 || r.Fields.Unobserved != 36 || r.Fields.Percent != nil {
		t.Fatal(r)
	}
}

func TestMissingActualFieldIsUnobservedAndUnmatched(t *testing.T) {
	var source observation
	if err := json.Unmarshal(fixture(t, "model-0"), &source); err != nil {
		t.Fatal(err)
	}
	d := &source.Runtime.Traces[0].Deliveries[1]
	d.Actual = json.RawMessage(`{"title":"gooo"}`)
	passed := false
	d.Passed = &passed
	source.Runtime.Passed = 35
	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	r, err := measure(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Fields.Unobserved != 1 || r.Fields.Passed != 35 || r.Fields.Total != 36 || len(r.Gaps) != 2 || r.Gaps[1].Status != "unobserved_actual" {
		t.Fatal(r)
	}
}

func TestRejectInconsistentCapturedCountsAndRecordContract(t *testing.T) {
	for _, mutate := range []func(*observation){
		func(s *observation) { s.Runtime.Passed-- },
		func(s *observation) { s.Composition.Plan.Records = nil },
		func(s *observation) { s.Composition.Plan.Records[0].Fields[1].Name = "title" },
		func(s *observation) { s.Runtime.Traces[0].Deliveries[1].ActivityID = "other" },
	} {
		var source observation
		if err := json.Unmarshal(fixture(t, "model-0"), &source); err != nil {
			t.Fatal(err)
		}
		mutate(&source)
		raw, err := json.Marshal(source)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := measure(raw); err == nil {
			t.Fatal("inconsistent observation accepted")
		}
	}
}

func TestJSONEqualityRetainsExactIntegersAndStringMeaning(t *testing.T) {
	for _, test := range []struct {
		a, b string
		same bool
	}{
		{`{"state":"ready","title":"한글"}`, `{"title":"\ud55c\uae00","state":"ready"}`, true},
		{`9223372036854775807`, `9223372036854775806`, false},
	} {
		matched, err := equalValues(json.RawMessage(test.a), json.RawMessage(test.b))
		if err != nil || matched != test.same {
			t.Fatalf("%v/%v", matched, err)
		}
	}
}

func TestInstalledUnscoredAndExternalRecordControls(t *testing.T) {
	for _, test := range []struct {
		name                        string
		outputs, fields, unobserved int
	}{
		{"installed/partial", 35, 35, 0},
		{"installed/external", 8, 8, 0},
		{"installed/external-replay", 8, 8, 0},
		{"installed/unscored-records", 18, 0, 36},
	} {
		r, err := measure(fixture(t, test.name))
		if err != nil {
			t.Fatal(err)
		}
		if r.Outputs.Passed != test.outputs || r.Fields.Passed != test.fields || r.Fields.Unobserved != test.unobserved {
			t.Fatal(r)
		}
		if test.unobserved != 0 && (r.Fields.Total != 0 || r.Fields.Percent != nil || r.Outputs.Unobserved != 18) {
			t.Fatal(r)
		}
	}
}
