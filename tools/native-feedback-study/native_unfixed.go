package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func inspectNativeUnfixed(v nativeResult, row compoundstudy.Case, arm familyArm, source string,
	unfixed bool, wall int64) (int, error) {
	return inspectNativeUnfixedFor(v, row, arm, pathplan.CIHint{SourceSHA: source, Status: "PASS"}, unfixed, wall)
}

func inspectNativeUnfixedFor(v nativeResult, row compoundstudy.Case, arm familyArm, ci pathplan.CIHint, unfixed bool, wall int64) (int, error) {
	p := v.Report.Paths
	document, _ := json.Marshal(row.Document)
	cases, _ := json.Marshal(row.Document.Cases)
	if v.Report.Decision != "PASS" || v.Report.Compiler != ci.SourceSHA || !v.Report.Types || !v.Report.Replay ||
		v.Report.Writes != 0 || v.Report.ActivityID != "compound-study://activity/compose-paths" ||
		!p.Bound || p.Unfixed != unfixed || p.OriginalSHA != "sha256:"+hash([]byte(row.Source)) ||
		p.DocumentSHA != "sha256:"+hash(document) || p.SuiteSHA != "sha256:"+hash(cases) || len(p.Cases) != len(row.Document.Cases) {
		return 0, errors.New("native unfixed source/mode/verification binding differs")
	}
	prepared, err := pathplan.Prepare(row.Document.Plan)
	if err != nil {
		return 0, err
	}
	body, err := prepared.Compile(p.Search.Selection.Choices)
	if err != nil {
		return 0, err
	}
	actual, err := compoundFunction(v.Source)
	expected, expectedErr := compoundFunction(body.GoSource())
	if err != nil || expectedErr != nil || actual != expected {
		return 0, errors.New("native emitted function differs from chosen typed body")
	}
	// The SDK renderer is used only after independently binding actual native
	// function AST. This adapts the trace verifier, not the captured native bytes.
	capture := unfixedCapture{CaseID: row.ID, Arm: arm.Name, Unfixed: unfixed, Progress: p.Progress,
		Feedback: p.Feedback, Search: p.Search, Source: body.GoSource(), WallNS: wall}
	skipped, err := verifyUnfixedWithCI(capture, row, arm, ci)
	if err != nil {
		return 0, err
	}
	mask, err := compoundstudy.Mask(row.Document.Plan, p.Search.Selection.Choices)
	if err != nil {
		return 0, err
	}
	passed := 0
	for i, test := range row.Document.Cases {
		value, _ := compoundstudy.Oracle(row.Template, mask, test.Input)
		observed := p.Cases[i]
		if observed.Input != test.Input || observed.Expected != test.Expected || observed.Actual != value || observed.Passed != (value == test.Expected) {
			return 0, errors.New("native recorded finite actual differs from independent state oracle")
		}
		if observed.Passed {
			passed++
		}
	}
	if passed != p.Search.SelectedTrainingPassed || p.Completeness != 100*float64(passed)/float64(len(row.Document.Cases)) {
		return 0, errors.New("native finite completeness denominator differs")
	}
	return skipped, nil
}

func runNativeUnfixedPilot(binary, goBinary, output, revision, native, ciStatus string) error {
	ci := pathplan.CIHint{SourceSHA: native, Status: ciStatus}
	if err := ci.Validate(); err != nil {
		return err
	}
	_, arms, err := familyPreflightFor(binary, goBinary, output, revision,
		familySpec{Revision: native, SDK: "v0.2.7-experimental", CIStatus: ciStatus})
	if err != nil {
		return err
	}
	rows, err := nativeUnfixedRows()
	if err != nil {
		return err
	}
	binarySHA, err := executableHash(binary)
	if err != nil {
		return err
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(output, "captures"), 0755); err != nil {
		return err
	}
	if err = os.Mkdir(filepath.Join(output, "metrics"), 0755); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/native-unfixed-preexecution/v1",
		"runner_revision": revision, "native_revision": native, "native_binary_sha256": binarySHA, "go_binary_sha256": goSHA,
		"go": "1.27.1", "sdk": "v0.2.7-experimental", "planned_native_calls": 48, "views": 6,
		"caller_ci_hint": ci, "caller_ci_hint_is_authority": false,
		"scope": "Three reused compound templates, one numeric configuration, mask-zero intentions, contradictory eight-case contracts, two languages, four own frozen models and paired legacy/opt-in mode. Source CI verification is recorded separately; this unauthenticated caller hint is not authority."}); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "gooo-native-unfixed-pilot-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err = save(filepath.Join(dir, "hint.json"), ci); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	sources := map[string]string{}
	inputsSet := map[int64]bool{}
	for _, row := range rows {
		if err = os.WriteFile(filepath.Join(dir, "source.gooo"), []byte(row.Source), 0600); err != nil {
			return err
		}
		if err = save(filepath.Join(dir, "plan.json"), row.Document); err != nil {
			return err
		}
		for _, test := range append(append([]pathplan.TestCase(nil), row.Document.Cases...), row.Separate...) {
			inputsSet[test.Input] = true
		}
		for _, arm := range arms {
			if !arm.Feedback {
				continue
			}
			for _, unfixed := range []bool{false, true} {
				id := fmt.Sprintf("%s-%s-%t", row.ID, arm.Name, unfixed)
				args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-model", arm.Path,
					"--path-step-attempts", "1", "--path-feedback-rounds", "3", "--path-feedback-ci", "hint.json",
					"--activity", "ComposePaths"}
				if unfixed {
					args = append(args, "--path-feedback-unfixed")
				}
				args = append(args, "source.gooo")
				raw, m, failure := child(ctx, dir, binary, args...)
				if err = os.WriteFile(filepath.Join(output, "captures", id+".json"), raw, 0644); err != nil {
					return err
				}
				if err = save(filepath.Join(output, "metrics", id+".json"), m); err != nil {
					return err
				}
				if failure != nil {
					return failure
				}
				var value nativeResult
				if json.Unmarshal(raw, &value) != nil {
					return errors.New("native unfixed capture decode failed")
				}
				_, err := inspectNativeUnfixedFor(value, row, arm, ci, unfixed, m.Wall)
				if err != nil {
					return fmt.Errorf("%s: %w", id, err)
				}
				sha := hash([]byte(value.Source))
				sources[sha] = value.Source
			}
		}
	}
	var inputs []int64
	for input := range inputsSet {
		inputs = append(inputs, input)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i] < inputs[j] })
	if err = os.Mkdir(filepath.Join(output, "executions"), 0755); err != nil {
		return err
	}
	var keys []string
	for sha := range sources {
		keys = append(keys, sha)
	}
	sort.Strings(keys)
	for _, sha := range keys {
		raw, err := executeFunction(ctx, goBinary, sources[sha], "ComposePaths", inputs)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "executions", sha+".json"), raw, 0644); err != nil {
			return err
		}
		var values []int64
		if json.Unmarshal(raw, &values) != nil || len(values) != len(inputs) {
			return errors.New("actual Go pilot values missing")
		}
	}
	report, err := auditNativeUnfixed(output, revision, native)
	if err != nil {
		return err
	}
	return save(filepath.Join(output, "report.json"), report)
}
