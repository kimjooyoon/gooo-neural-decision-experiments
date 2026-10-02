package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

func TestNativeChildCancellationRetainsOutputWithoutWaitingForDescendant(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("collector process-group cancellation is supported on Darwin/Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	raw, stderr, metrics, err := childWithContext(ctx, t.TempDir(), "/bin/sh", "-c", "printf 'before-timeout'; sleep 30 & wait")
	if !errors.Is(err, context.DeadlineExceeded) || string(raw) != "before-timeout" || stderr != "" || !metrics.Started || time.Since(start) > 2*time.Second {
		t.Fatalf("bounded cancellation must retain earlier bytes and stop the process group: %q %q %+v %v", raw, stderr, metrics, err)
	}
}

func TestNativeChildOutputOverflowPreservesRecordedPrefix(t *testing.T) {
	buffer := childBuffer{Cap: 4}
	if n, err := buffer.Write([]byte("old")); n != 3 || err != nil {
		t.Fatal("initial bounded output rejected")
	}
	if n, err := buffer.Write([]byte("overflow")); n != 0 || err == nil || buffer.String() != "old" {
		t.Fatal("overflow must retain prior bytes and explicitly fail")
	}
}

func TestNativeOriginalCallerInputIsCompleteDespiteABIDecline(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		plan, err := threecompositionstudy.Fixture("chained_operands", 0, 7, language)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := pathplan.Prepare(plan)
		if err != nil {
			t.Fatal(err)
		}
		var expected strings.Builder
		expected.WriteString("gooo;joint3|")
		for _, choice := range plan.Decisions {
			expected.WriteString(strconv.Itoa(len(choice.Intent)))
			expected.WriteByte(':')
			expected.WriteString(choice.Intent)
		}
		text, err := prepared.ThreeInput()
		if err == nil || text != expected.String() {
			t.Fatal("original noncanonical caller input must decline the ABI and retain every byte in declared order")
		}
	}
}

func TestNativeFixedArrayProcessSummaries(t *testing.T) {
	var values [128]childMetrics
	for i := range values {
		n := int64(i + 1)
		values[i] = childMetrics{Started: true, Wall: n * 20, User: n * 10, System: n * 5, RSS: n * 100, CPU: 75}
	}
	s, err := summarizeChildren(values)
	if err != nil || s.Children != 128 || s.Median != 1290 || s.P95 != 2440 || s.RSS != 6450 || s.MaxRSS != 12800 || s.CPUPercent != 75 {
		t.Fatalf("fixed-array median/p95/resource definitions: %+v %v", s, err)
	}
	for _, bad := range []childMetrics{{Started: false, Wall: 20, RSS: 100}, {Started: true, Wall: 0, RSS: 100}, {Started: true, Wall: 20, User: -1, RSS: 100}, {Started: true, Wall: 20, RSS: 0}, {Started: true, Wall: 20, RSS: 100, CPU: math.NaN()}, {Started: true, Wall: 20, RSS: 100, CPU: 99}} {
		changed := values
		changed[0] = bad
		if _, err := summarizeChildren(changed); err == nil {
			t.Fatal("invalid actual child measurements accepted")
		}
	}
	store := storage{Cap: continuationCap, Used: continuationCap - receiptReserve - 2*lineCap + 1}
	if err := beforeNativePair(&store); err == nil {
		t.Fatal("pair allowed without complete native/execution/terminal reserve")
	}
}

// Replay the already retained optional validation after auditor changes,
// without repeating its compiler processes, executions or model predictions.
func TestRetainedActualNativeValidationWithZeroOperationalCalls(t *testing.T) {
	dir := os.Getenv("GOOO_THREE_NATIVE_REPLAY_OUTPUT")
	if dir == "" {
		t.Skip("retained optional integration evidence supplied explicitly")
	}
	enterRepo(t)
	models, _, err := loadModels("models/own-three-feedback-v1")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("internal/threefeedback/testdata/native-goal7-rows.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), lineCap)
	pairs := 0
	for scan.Scan() {
		v, err := threecohort.ReconstructRow(scan.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		doc, src, err := nativeOriginal(v)
		if err != nil {
			t.Fatal(err)
		}
		for _, policy := range []nativePolicy{nativePolicies[0], nativePolicies[4]} {
			name := nativeName(v, policy)
			raw, err := os.ReadFile(filepath.Join(dir, name+"-native.json"))
			if err != nil {
				t.Fatal(err)
			}
			var x nativeExecution
			if err = strict(filepath.Join(dir, name+"-execution.json"), &x); err != nil {
				t.Fatal(err)
			}
			n, _, _, err := inspectNative(raw, v, doc, src, models[policy.Candidate], x.Codegen.Wall)
			if err != nil {
				t.Fatal(name, err)
			}
			if err = auditNativeExecution(x, raw, n, v, policy); err != nil {
				t.Fatal(name, err)
			}
			if policy.Name == "selected" {
				n.Report.Paths.Context.Semantic = "wrong-source-semantic"
				changed, err := json.Marshal(n)
				if err != nil {
					t.Fatal(err)
				}
				if _, _, _, err = inspectNative(changed, v, doc, src, models[policy.Candidate], x.Codegen.Wall); err == nil {
					t.Fatal("unbound projected source semantics accepted")
				}
			}
			pairs++
		}
	}
	if err = scan.Err(); err != nil || pairs != 32 {
		t.Fatal("all 32 retained real validation pairs required", pairs, err)
	}
	t.Log("32 actual captured native/compiled pairs replayed; new native, Go and model calls=0")
}

// Optional local integration validation. These 32 pairs are not the frozen
// 640 operational pairs; the published native collector never counts them.
func TestActualNativeAndCompiledGoOnAllEightFamilies(t *testing.T) {
	binary, goBinary := os.Getenv("GOOO_THREE_NATIVE_TEST_BINARY"), os.Getenv("GOOO_THREE_NATIVE_TEST_GO_BINARY")
	if binary == "" || goBinary == "" {
		t.Skip("pinned native/Go binaries supplied only for explicit integration validation")
	}
	enterRepo(t)
	if _, _, _, err := compilerPins(binary, goBinary); err != nil {
		t.Fatal(err)
	}
	models, _, err := loadModels("models/own-three-feedback-v1")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("internal/threefeedback/testdata/native-goal7-rows.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), lineCap)
	var chosenRaw []byte
	var chosenExecution nativeExecution
	var chosenView threecohort.View
	var chosenDoc nativeDocument
	var chosenSource []byte
	pairs, predictions := 0, 0
	store := &storage{Cap: continuationCap}
	out, workspace := t.TempDir(), t.TempDir()
	if retained := os.Getenv("GOOO_THREE_NATIVE_TEST_OUTPUT"); retained != "" {
		if err = os.Mkdir(retained, 0700); err != nil {
			t.Fatal("fresh retained integration output required", err)
		}
		out = retained
	}
	for scan.Scan() {
		v, err := threecohort.ReconstructRow(scan.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		doc, src, err := nativeOriginal(v)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []nativePolicy{nativePolicies[0], nativePolicies[4]} {
			m := models[p.Candidate]
			prior, err := one(v, m)
			if err != nil {
				t.Fatal(err)
			}
			x, raw, n, _, _, err := runNativePair(store, out, workspace, binary, goBinary, "models/own-three-feedback-v1", v, p, doc, src, m, priorOf(prior))
			if err != nil {
				t.Fatalf("%s: %v", nativeName(v, p), err)
			}
			if err = auditNativeExecution(x, raw, n, v, p); err != nil {
				t.Fatal(err)
			}
			if len(raw) > lineCap {
				t.Fatal("actual complete native capture bound exceeded")
			}
			if p.Candidate == "offline" && n.Report.Paths.Search.Selection.ModelCalls != 0 {
				t.Fatal("disconnected native made model call")
			}
			if chosenRaw == nil && m != nil {
				chosenRaw, chosenExecution, chosenView, chosenDoc, chosenSource = append([]byte{}, raw...), x, v, doc, append([]byte{}, src...)
			}
			pairs++
			predictions += n.Report.Paths.Search.Selection.ModelCalls
		}
	}
	if err = scan.Err(); err != nil {
		t.Fatal(err)
	}
	if pairs != 32 || chosenRaw == nil {
		t.Fatal("actual all-family 16-view online/offline validation required")
	}
	t.Logf("integration validation only: native pairs=%d, compiled Go invocations=%d, native model predictions=%d", pairs, pairs*16, predictions)
	mutations := []func(*nativeResult){
		func(n *nativeResult) { n.Report.Paths.Context.Inputs[0].SHA = "incomplete" },
		func(n *nativeResult) { n.Report.Paths.Context.Declared.Text += " truncated intent" },
		func(n *nativeResult) { n.Report.Paths.Context.RankedPlan = "wrong-source" },
		func(n *nativeResult) { n.Report.Paths.Context.Semantic = "wrong-source-semantic" },
		func(n *nativeResult) { n.Report.Paths.Document = "wrong-document" },
		func(n *nativeResult) { n.Report.Paths.Cases[0].Actual++ },
		func(n *nativeResult) { n.Report.Writes = 1 },
		func(n *nativeResult) { n.Report.Compiler = "unpromoted-feature" },
	}
	for i, mutate := range mutations {
		var n nativeResult
		if err = json.Unmarshal(chosenRaw, &n); err != nil {
			t.Fatal(err)
		}
		mutate(&n)
		raw, err := json.Marshal(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, err = inspectNative(raw, chosenView, chosenDoc, chosenSource, models[nativePolicies[0].Candidate], chosenExecution.Codegen.Wall); err == nil {
			t.Fatalf("native mutation %d accepted", i)
		}
	}
	var n nativeResult
	if err = json.Unmarshal(chosenRaw, &n); err != nil {
		t.Fatal(err)
	}
	changed := chosenExecution
	changed.Values = append([]int64{}, chosenExecution.Values...)
	changed.Values[0]++
	if err = auditNativeExecution(changed, chosenRaw, n, chosenView, nativePolicies[0]); err == nil {
		t.Fatal("actual Go stdout/value disagreement accepted")
	}
	if _, err = os.Stat(filepath.Join(out, nativeName(chosenView, nativePolicies[0])+"-native.json")); err != nil {
		t.Fatal("original native stdout not retained")
	}
}
