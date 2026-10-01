package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestNativeUnfixedInspectorBindsNativeSourceAndPartialOutcome(t *testing.T) {
	t.Chdir("../..")
	rows, err := nativeUnfixedRows()
	if err != nil {
		t.Fatal(err)
	}
	arms, err := compoundArms()
	if err != nil {
		t.Fatal(err)
	}
	row, arm := rows[0], arms[2]
	if !arm.Feedback {
		t.Fatal("feedback arm required")
	}
	raw, err := read(filepath.Join("runs/compound-path-main-20261001/captures", row.ID+"-"+arm.Name+"-true.json"))
	if err != nil {
		t.Fatal(err)
	}
	decode := func() nativeResult {
		var v nativeResult
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	v := decode()
	if _, err := inspectNativeUnfixed(v, row, arm, compoundNative, false, 1); err != nil {
		t.Fatal(err)
	}
	if v.Report.Paths.Completeness != 87.5 {
		t.Fatal("contradictory partial completeness must be retained")
	}
	for name, mutate := range map[string]func(*nativeResult){
		"mode":         func(v *nativeResult) { v.Report.Paths.Unfixed = true },
		"source":       func(v *nativeResult) { v.Report.Paths.OriginalSHA = hash([]byte("other source")) },
		"document":     func(v *nativeResult) { v.Report.Paths.DocumentSHA = hash([]byte("other document")) },
		"suite":        func(v *nativeResult) { v.Report.Paths.SuiteSHA = hash([]byte("other suite")) },
		"body":         func(v *nativeResult) { v.Source = "package main\nfunc ComposePaths(input int64) int64 {return 0}\n" },
		"actual":       func(v *nativeResult) { v.Report.Paths.Cases[0].Actual++ },
		"completeness": func(v *nativeResult) { v.Report.Paths.Completeness = 100 },
		"write":        func(v *nativeResult) { v.Report.Writes = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			v := decode()
			mutate(&v)
			if _, err := inspectNativeUnfixed(v, row, arm, compoundNative, false, 1); err == nil {
				t.Fatal("mutated native evidence accepted")
			}
		})
	}
}
