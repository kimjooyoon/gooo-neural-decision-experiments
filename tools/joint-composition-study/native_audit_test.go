package main

import (
	"testing"
)

func TestActualGoExecutionAuditRejectsBoundValuesAndMetricsTamper(t *testing.T) {
	v := testView(t, "schedule_branch", 3, "ko")
	doc, _, err := originalDocument(v)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("synthetic retained capture")
	n := nativeResult{Source: "synthetic source"}
	x := executionCapture{Capture: hash(raw), Source: hash([]byte(n.Source)), Executed: true, Cases: 16, Metrics: metrics{Wall: 100, User: 40, System: 10, RSS: 1024, CPU: 50}}
	for _, c := range doc.Cases {
		x.Values = append(x.Values, c.Expected)
	}
	if err = auditExecution(x, raw, n, doc); err != nil {
		t.Fatal(err)
	}
	x.Values[0]++
	if err = auditExecution(x, raw, n, doc); err == nil {
		t.Fatal("incorrect actual value accepted")
	}
	x.Values[0]--
	x.Metrics.CPU++
	if err = auditExecution(x, raw, n, doc); err == nil {
		t.Fatal("inconsistent CPU measurement accepted")
	}
	x.Metrics.CPU--
	x.Source = "changed"
	if err = auditExecution(x, raw, n, doc); err == nil {
		t.Fatal("changed emitted source accepted")
	}
}
