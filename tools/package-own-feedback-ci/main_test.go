package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestClosedCIInventory(t *testing.T) {
	all, reports := inventory()
	if len(all) != 519 || len(reports) != 15 {
		t.Fatal("closed CI cohort changed", len(all), len(reports))
	}
	for p := range reports {
		if !all[p] {
			t.Fatal("report absent from fixed raw cohort")
		}
	}
}
func TestStreamPrivacyAndActualBytes(t *testing.T) {
	raw := "{\"synthetic\":true}\n"
	c := &counter{W: io.Discard}
	a, e := streamText(strings.NewReader(raw), c, int64(len(raw)))
	if e != nil || a.Bytes != int64(len(raw)) || len(a.SHA) != 64 {
		t.Fatal("stream binding differs", e)
	}
	if _, e = streamText(strings.NewReader("/Users/private-example/data\n"), &counter{W: io.Discard}, 1000); e == nil {
		t.Fatal("private host path accepted")
	}
}
func TestCompletedRunIdentity(t *testing.T) {
	jobs := []map[string]string{}
	for name := range requiredJobs() {
		jobs = append(jobs, map[string]string{"name": name, "status": "completed", "conclusion": "success"})
	}
	r := map[string]any{"databaseId": runID, "headSha": ciHead, "status": "completed", "conclusion": "success", "jobs": jobs}
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	if e = validateRun(b); e != nil {
		t.Fatal(e)
	}
	r["headSha"] = "wrong"
	b, e = json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	if validateRun(b) == nil {
		t.Fatal("wrong source accepted")
	}
	r["headSha"] = ciHead
	r["jobs"] = jobs[:9]
	b, e = json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	if validateRun(b) == nil {
		t.Fatal("missing job accepted")
	}
}
