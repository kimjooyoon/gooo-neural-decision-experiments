package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

func runNativePair(store *storage, out, workspace, binary, goBinary, models string, v threecohort.View, p nativePolicy, doc nativeDocument, src []byte, m *model, prior nativePrior) (x nativeExecution, raw []byte, n nativeResult, c threefeedback.Capture, auditView threecohort.View, failure error) {
	x = nativeExecution{Schema: "gooo/own-three-native-execution/v1", View: v.ID, Policy: p, Cases: 16, Values: []int64{}}
	auditView = v
	name := nativeName(v, p)
	defer func() {
		if failure != nil {
			x.Error = failure.Error()
		}
		if err := store.save(filepath.Join(out, name+"-execution.json"), x); err != nil && failure == nil {
			failure = err
		}
	}()
	docRaw, err := json.Marshal(doc)
	if err != nil {
		failure = err
		return
	}
	if err = os.WriteFile(filepath.Join(workspace, "plan.json"), docRaw, 0600); err != nil {
		failure = err
		return
	}
	if err = os.WriteFile(filepath.Join(workspace, "source.gooo"), src, 0600); err != nil {
		failure = err
		return
	}
	args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ChoosePath", "source.gooo"}
	if m != nil {
		parts := strings.Split(p.Candidate, "/")
		if len(parts) != 2 {
			failure = errors.New("known own-model candidate required")
			return
		}
		path, e := filepath.Abs(filepath.Join(models, parts[0], "models", parts[1], "model.json"))
		if e != nil {
			failure = e
			return
		}
		args = append(args, "--path-model", path, "--path-feedback-rounds", "7", "--path-feedback-unfixed")
	}
	raw, x.CodegenStderr, x.Codegen, err = child(workspace, binary, args...)
	x.NativeCalled = x.Codegen.Started
	x.Capture = threecohort.SHA(raw)
	// Preserve the exact original stdout immediately, before decoding or
	// invoking the independent Go execution compiler.
	if e := store.saveBytes(filepath.Join(out, name+"-native.json"), raw); e != nil {
		failure = e
		return
	}
	if !utf8.ValidString(x.CodegenStderr) {
		failure = errors.New("non-UTF8 native stderr cannot be silently converted")
		return
	}
	if err != nil {
		failure = err
		return
	}
	n, c, auditView, err = inspectNative(raw, v, doc, src, m, x.Codegen.Wall)
	if err != nil {
		failure = err
		return
	}
	if err = compareSDK(c, prior); err != nil {
		failure = err
		return
	}
	x.Source = threecohort.SHA([]byte(n.Source))
	if err = store.beforeCall(); err != nil {
		failure = err
		return
	}
	x.Values, x.GoStdout, x.GoStderr, x.Execution, err = executeGo(goBinary, n.Source, v.Cases)
	x.GoCalled = x.Execution.Started
	if err != nil {
		failure = err
		return
	}
	x.Executed = true
	return
}
