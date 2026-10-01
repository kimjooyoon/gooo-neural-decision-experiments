package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

type compilerNativeCell struct {
	ID          string  `json:"id"`
	Row         string  `json:"row_id"`
	Arm         string  `json:"arm"`
	Capture     string  `json:"capture_sha256"`
	Metrics     metrics `json:"process_metrics"`
	GoExecution bool    `json:"actual_compiled_go_execution"`
	GoValues    []int64 `json:"actual_go_values,omitempty"`
}

func compilerNativeSubset(rows []compilerTrainingRow) ([]compilerTrainingRow, error) {
	counts := map[string]int{}
	var result []compilerTrainingRow
	for _, row := range rows {
		if row.Split != "test" || counts[row.Family] >= 8 {
			continue
		}
		counts[row.Family]++
		result = append(result, row)
	}
	if len(result) != 40 || len(counts) != 5 {
		return nil, errors.New("five families/eight development views required")
	}
	return result, nil
}

func compilerNativeDocument(row compilerTrainingRow) (document, []byte, error) {
	plan, err := pathstudy.Fixture(row.Family, row.Configuration, row.Control)
	if err != nil {
		return document{}, nil, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return document{}, nil, err
	}
	body, err := json.Marshal(prepared.Fallback().GoooBody())
	if err != nil {
		return document{}, nil, err
	}
	cases, err := compilerFullCases(row)
	if err != nil {
		return document{}, nil, err
	}
	source := []byte("package owntraining\nnamespace owntraining\nentity Integer id \"owntraining://entity/integer\"\n" +
		"activity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	return document{"gooo/body-codegen-typed-path-plan/v1", plan, cases, 2}, source, nil
}

func inspectCompilerNative(raw []byte, row compilerTrainingRow, arm, native string, model *decision.Model) (nativeResult, error) {
	var value nativeResult
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, err
	}
	p, r := value.Report.Paths, value.Report
	cases, err := compilerFullCases(row)
	if err != nil {
		return value, err
	}
	if r.Decision != "PASS" || !r.Types || !r.Replay || r.Compiler != native || r.Writes != 0 || !p.Bound ||
		!p.Binding.Equivalent || p.OriginalSHA != "sha256:"+row.SourceSHA || len(p.Cases) != len(cases) ||
		p.Search.SelectedTrainingPassed != len(cases) || p.Search.TrainingTotal != len(cases) || p.Search.Evaluated > 2 ||
		p.Search.Selection.ExternalCalls != 0 || p.Completeness != 100 {
		return value, errors.New("native source/finite contract differs")
	}
	if arm == "offline" {
		if p.Context != nil || p.Search.Selection.ModelCalls != 0 {
			return value, errors.New("offline path invoked model/context")
		}
	} else {
		if p.Context == nil || p.Context.Schema != "gooo/compiler-typed-path-context/v2" || p.Context.Status != "ENCODED" ||
			len(p.Context.Inputs) != 1 || bareOwnContextSHA(p.Context.Inputs[0].SHA) != row.SHA ||
			p.Context.Metadata != model.MetadataSHA256() || p.Search.Selection.ModelCalls != 1 ||
			len(p.Search.Selection.Receipts) != 1 || p.Search.Selection.Receipts[0].IntentSHA256 != row.SHA {
			return value, errors.New("native/export/model canonical input differs")
		}
	}
	for i, c := range cases {
		if p.Cases[i].Input != c.Input || p.Cases[i].Expected != c.Expected || p.Cases[i].Actual != c.Expected || !p.Cases[i].Passed {
			return value, errors.New("native actuals differ from independent arithmetic oracle")
		}
	}
	doc, _, err := compilerNativeDocument(row)
	if err != nil {
		return value, err
	}
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		return value, err
	}
	if err = auditOwnContextAttempts(prepared, doc, p.Search); err != nil {
		return value, err
	}
	return value, nil
}

func runCompilerNativeDogfood(binary, goBinary, curriculum, models, output, revision, native string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 native required")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["vcs.revision"] != native || settings["vcs.modified"] != "false" {
		return errors.New("clean exact native revision required")
	}
	rows, err := compilerTrainingRows(curriculum)
	if err != nil {
		return err
	}
	rows, err = compilerNativeSubset(rows)
	if err != nil {
		return err
	}
	binarySHA, err := executableHash(binary)
	if err != nil {
		return err
	}
	loaded := map[string]*decision.Model{}
	pins := map[string]string{}
	for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		loaded[arm], err = decision.LoadPath(filepath.Join(models, "compiler_context", "models", arm, "model.json"))
		if err != nil {
			return err
		}
		pins[arm] = loaded[arm].MetadataSHA256()
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	protocol, err := read(compilerCurriculumProtocol)
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/compiler-native-dogfood-preexecution/v1",
		"runner_revision": revision, "native_revision": native, "binary_sha256": binarySHA, "model_metadata_sha256": pins, "protocol_sha256": hash(protocol),
		"planned_native_calls": 160, "planned_model_predictions": 120, "planned_compiled_go_executions": 20, "new_independent_intentions": 0}); err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "gooo-own-model-dogfood-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	absoluteModels, err := filepath.Abs(models)
	if err != nil {
		return err
	}
	var cells []compilerNativeCell
	firstFamilies := map[string]bool{}
	for _, row := range rows {
		doc, source, err := compilerNativeDocument(row)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(workspace, "source.gooo"), source, 0644); err != nil {
			return err
		}
		if err = save(filepath.Join(workspace, "plan.json"), doc); err != nil {
			return err
		}
		for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary", "offline"} {
			id := row.ID + "-" + arm
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--activity", "ChoosePath", "source.gooo"}
			if arm != "offline" {
				args = append(args, "--path-model", filepath.Join(absoluteModels, "compiler_context", "models", arm, "model.json"))
			}
			raw, m, runErr := child(context.Background(), workspace, binary, args...)
			if err = os.WriteFile(filepath.Join(output, id+".json"), raw, 0644); err != nil {
				return err
			}
			if runErr != nil {
				return runErr
			}
			value, err := inspectCompilerNative(raw, row, arm, native, loaded[arm])
			if err != nil {
				return err
			}
			cell := compilerNativeCell{ID: id, Row: row.ID, Arm: arm, Capture: hash(raw), Metrics: m}
			if !firstFamilies[row.Family] {
				var inputs []int64
				for _, c := range doc.Cases {
					inputs = append(inputs, c.Input)
				}
				rawValues, err := executeFunction(context.Background(), goBinary, value.Source, "ChoosePath", inputs)
				if err != nil {
					return err
				}
				var values []int64
				if err = json.Unmarshal(rawValues, &values); err != nil || len(values) != len(doc.Cases) {
					return errors.New("compiled Go values malformed")
				}
				for i, actual := range values {
					if actual != doc.Cases[i].Expected {
						return errors.New("compiled Go actuals differ")
					}
				}
				cell.GoExecution, cell.GoValues = true, values
			}
			cells = append(cells, cell)
		}
		firstFamilies[row.Family] = true
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/compiler-native-dogfood/v1", "status": "PASS",
		"cells": cells, "actual_native_calls": len(cells), "actual_model_predictions": 120, "actual_compiled_go_executions": 20,
		"scope": "40 deterministic reused development views, four policy arms; canonical input hashes/native actuals/20 compiled Go executions verified; process CPU is one-core-normalized child CPU, not host utilization"})
}

func auditCompilerNativeDogfood(curriculum, models, output, native string) error {
	rows, err := compilerTrainingRows(curriculum)
	if err != nil {
		return err
	}
	rows, err = compilerNativeSubset(rows)
	if err != nil {
		return err
	}
	raw, err := read(filepath.Join(output, "report.json"))
	if err != nil {
		return err
	}
	var report struct {
		Cells       []compilerNativeCell `json:"cells"`
		Calls       int                  `json:"actual_native_calls"`
		Predictions int                  `json:"actual_model_predictions"`
		Executions  int                  `json:"actual_compiled_go_executions"`
	}
	if err = json.Unmarshal(raw, &report); err != nil || report.Calls != 160 || report.Predictions != 120 || report.Executions != 20 || len(report.Cells) != 160 {
		return errors.New("native dogfood report denominators differ")
	}
	index, executions := 0, 0
	for _, row := range rows {
		for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary", "offline"} {
			cell := report.Cells[index]
			index++
			id := row.ID + "-" + arm
			if cell.ID != id || cell.Row != row.ID || cell.Arm != arm {
				return errors.New("native dogfood row order differs")
			}
			raw, err := read(filepath.Join(output, id+".json"))
			if err != nil || cell.Capture != hash(raw) {
				return errors.New("native capture hash differs")
			}
			var model *decision.Model
			if arm != "offline" {
				model, err = decision.LoadPath(filepath.Join(models, "compiler_context", "models", arm, "model.json"))
				if err != nil {
					return err
				}
			}
			if _, err = inspectCompilerNative(raw, row, arm, native, model); err != nil {
				return err
			}
			if cell.GoExecution {
				executions++
				cases, err := compilerFullCases(row)
				if err != nil || len(cases) != len(cell.GoValues) {
					return errors.New("Go execution count differs")
				}
				for i, c := range cases {
					if c.Expected != cell.GoValues[i] {
						return errors.New("Go execution actual changed")
					}
				}
			}
		}
	}
	if executions != 20 {
		return errors.New("20 compiled Go executions required")
	}
	return nil
}
