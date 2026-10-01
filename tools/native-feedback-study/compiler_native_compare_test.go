package main

import "testing"

func TestCompilerNativeNormalizationRetainsSemanticsAndIgnoresTiming(t *testing.T) {
	value := nativeResult{Source: "package generated"}
	cell := compilerNativeCell{ID: "cell", GoExecution: true, GoValues: []int64{7}}
	before, err := normalizedCompilerNative(value, cell)
	if err != nil {
		t.Fatal(err)
	}
	value.Report.Compiler = "new source revision"
	value.Report.Paths.Timing.Total = 999
	after, err := normalizedCompilerNative(value, cell)
	if err != nil || string(before) != string(after) {
		t.Fatal("revision/timing changed semantic comparison")
	}
	cell.GoValues[0]++
	after, err = normalizedCompilerNative(value, cell)
	if err != nil || string(before) == string(after) {
		t.Fatal("changed executed value hidden")
	}
	cell.GoValues[0]--
	value.Report.Paths.Context = &ownContextReceipt{Schema: "context", Inputs: []ownContextInput{{SHA: "different-input"}}}
	after, err = normalizedCompilerNative(value, cell)
	if err != nil || string(before) == string(after) {
		t.Fatal("changed model input hidden")
	}
}
