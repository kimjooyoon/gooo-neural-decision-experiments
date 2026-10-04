package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func writeText(out io.Writer, m measurement, raw []byte) error {
	var e envelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return err
	}
	var text strings.Builder
	fmt.Fprintf(&text, "Profile: %s; current model calls: %d; stored construction calls: %d\n", m.Profile, m.Calls, m.StoredCalls)
	if m.StoredCalls > 0 {
		fmt.Fprintf(&text, "Stored construction prediction: %.3f us; resident tensors: %d bytes\n", float64(m.PredictNS)/1e3, m.TensorBytes)
	}
	for i, f := range m.Frames {
		action := "build"
		if f.Reused {
			action = "reuse"
		}
		fmt.Fprintf(&text, "Suite %d: outputs %d/%d (%.2f%%); record fields %d/%d (%.2f%%); %s; current runtime %.2f ms\n",
			i+1, f.Passed, f.Total, 100*float64(f.Passed)/float64(f.Total), f.FieldsPassed, f.FieldsTotal,
			100*float64(f.FieldsPassed)/float64(f.FieldsTotal), action, f.RuntimeMS)
		for _, trace := range e.History[i].Traces {
			for _, d := range trace.Deliveries {
				if d.Passed != nil && !*d.Passed {
					fmt.Fprintf(&text, "  Case %d %s: actual=%s expected=%s\n", trace.Index, d.ID, d.Actual, d.Expected)
				}
			}
		}
	}
	text.WriteString("Counts cover the supplied current suites. Each suite runs twice; the counts describe one observed result.\n")
	_, err := io.WriteString(out, text.String())
	return err
}
