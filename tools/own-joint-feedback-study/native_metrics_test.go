package main

import (
	"math"
	"testing"
)

func TestProcessSummaryDefinitions(t *testing.T) {
	var values []metrics
	for i := int64(1); i <= 48; i++ {
		values = append(values, metrics{Wall: i * 20, User: i * 10, System: i * 5, RSS: i * 100, CPU: 75})
	}
	s, err := summarizeProcesses(values)
	if err != nil || s.Children != 48 || s.Median != 490 || s.P95 != 920 || s.RSS != 2450 || s.MaxRSS != 4800 || s.CPU != 75 {
		t.Fatal("median, nearest-rank p95 or summed CPU definition differs", s, err)
	}
	if _, err = summarizeProcesses(values[:47]); err == nil {
		t.Fatal("missing process accepted")
	}
	for _, bad := range []metrics{{Wall: 0, RSS: 100}, {Wall: 20, User: -1, RSS: 100}, {Wall: 20, RSS: 0}, {Wall: 20, RSS: 100, CPU: math.NaN()}, {Wall: 20, RSS: 100, CPU: 99}} {
		v := append([]metrics{}, values...)
		v[0] = bad
		if _, err = summarizeProcesses(v); err == nil {
			t.Fatal("invalid child metrics accepted")
		}
	}
}
