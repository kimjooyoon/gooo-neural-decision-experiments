package main

import (
	"encoding/json"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
)

func TestOriginalNativeDocumentAndIndependentExecutionAudit(t *testing.T) {
	v := testView(t)
	doc, src, err := originalDocument(v)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Max != 4 || len(doc.Plan.Decisions) != 2 || jointcohort.SHA(src) != v.SourceSHA {
		t.Fatal("original source/doc mismatch")
	}
	changed := v
	changed.SourceSHA = "changed"
	if _, _, err = originalDocument(changed); err == nil {
		t.Fatal("different original source accepted")
	}
	raw := []byte("synthetic unit capture")
	n := nativeResult{Source: "synthetic unit emitted Go"}
	x := executionCapture{Capture: jointcohort.SHA(raw), Source: jointcohort.SHA([]byte(n.Source)), Executed: true, Cases: 16, Codegen: metrics{Wall: 100, User: 40, System: 10, RSS: 1024, CPU: 50}, Execution: metrics{Wall: 200, User: 40, System: 10, RSS: 2048, CPU: 25}}
	for _, c := range v.Cases {
		x.Values = append(x.Values, c.Expected)
	}
	if err = auditExecution(x, raw, n, v); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*executionCapture){func(x *executionCapture) { x.Values[0]++ }, func(x *executionCapture) { x.Cases-- }, func(x *executionCapture) { x.Execution.CPU++ }, func(x *executionCapture) { x.Codegen.RSS = 0 }, func(x *executionCapture) { x.Executed = false }, func(x *executionCapture) { x.Source = "changed" }} {
		b, _ := json.Marshal(x)
		var y executionCapture
		if err = json.Unmarshal(b, &y); err != nil {
			t.Fatal(err)
		}
		mutate(&y)
		if auditExecution(y, raw, n, v) == nil {
			t.Fatal("tampered execution accepted")
		}
	}
}
