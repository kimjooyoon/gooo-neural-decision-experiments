package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func TestNativeUnfixedFrozenPilotReproducesWithoutInference(t *testing.T) {
	t.Chdir("../..")
	for _, pin := range []struct{ root, runner, native, report string }{
		{"runs/unfixed-native-feature-pilot-20261001", "3749fdc6c1f218c36d944a7899bb1125061c6906", "d869be3219a7798b6cc3b13c31efa8a6d899ae22", "4dba2dab42c17af2c069a442b18053fc804bc50a32f5b4276799ddf3f96d3962"},
		{"runs/unfixed-native-main-pilot-20261001", "b093d1b7a855dba34051b0b4ff959ba8be45fdf0", "6f69eb116336b4728db6f94a992130ea56003a48", "6bf4030ee7fe769fd3befe8b5bee66bb6fe00fe272037153f4197d59cc6818f6"},
	} {
		v, err := auditNativeUnfixed(pin.root, pin.runner, pin.native)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		expected, err := read(filepath.Join(pin.root, "report.json"))
		if err != nil || hash(expected) != pin.report || !bytes.Equal(append(raw, '\n'), expected) {
			t.Fatal("frozen native pilot audit differs")
		}
		if _, err := auditNativeUnfixed(pin.root, pin.runner, compoundNative); err == nil {
			t.Fatal("other native source accepted")
		}
	}
}

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
	if _, err := inspectNativeUnfixedFor(v, row, arm, pathplan.CIHint{SourceSHA: compoundNative, Status: "UNKNOWN"}, false, 1); err == nil {
		t.Fatal("stored PASS receipt accepted as UNKNOWN")
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

func TestNativeUnfixedInvalidCIRejectedBeforeBinaryOrOutput(t *testing.T) {
	for _, status := range []string{"", "passed"} {
		err := runNativeUnfixedPilot("missing binary", "missing go", filepath.Join(t.TempDir(), "unwritten"), "runner", compoundNative, status)
		if err == nil || !strings.Contains(err.Error(), "CI hint requires") {
			t.Fatal("invalid CI was not rejected before execution", err)
		}
	}
}
