package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const pairedNativeMain = "b431ef7547a968e2532fa2dcf0891ee44ec2ee86"

var pairedNativeArms = []string{"selected", "reference", "offline"}

type pairedNativeReport struct {
	Schema      string               `json:"schema"`
	Status      string               `json:"status"`
	Selected    string               `json:"selected_candidate"`
	Cells       []compilerNativeCell `json:"cells"`
	Calls       int                  `json:"actual_native_calls"`
	Predictions int                  `json:"actual_model_predictions"`
	Executions  int                  `json:"actual_compiled_go_executions"`
	Invocations int                  `json:"actual_go_function_invocations"`
	Scope       string               `json:"scope"`
}

func pairedNativeModels(models, selectionPath string) (map[string]*decision.Model, map[string]string, pairedSelection, error) {
	selection, err := readPairedSelection(models, selectionPath)
	if err != nil {
		return nil, nil, selection, err
	}
	paths := map[string]string{"selected": filepath.Join(models, selection.Model), "reference": "runs/compiler-context-training-mps-20261001/compiler_context/models/fp32/model.json"}
	loaded := map[string]*decision.Model{}
	for arm, path := range paths {
		loaded[arm], err = decision.LoadPath(path)
		if err != nil {
			return nil, nil, selection, err
		}
		paths[arm], err = filepath.Abs(path)
		if err != nil {
			return nil, nil, selection, err
		}
	}
	if loaded["selected"].MetadataSHA256() != selection.Metadata || loaded["selected"].WeightsSHA256() != selection.Weights ||
		loaded["reference"].MetadataSHA256() != "af702111c4f25880180dbe2e58affe0118f1d353302bc6820a78bba6c0ace846" ||
		loaded["reference"].WeightsSHA256() != "4c977ed55ad4e6ef7b93f5d1b143cc85f3860f4f306ae1262b8562b624bad154" {
		return nil, nil, selection, errors.New("selected/reference model pins differ")
	}
	return loaded, paths, selection, nil
}

func pairedNativeBinary(binary, native string) error {
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" || native != pairedNativeMain {
		return errors.New("pinned Go 1.27.1 main required")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	sdk := ""
	for _, dep := range info.Deps {
		if dep.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = dep.Version
		}
	}
	if settings["vcs.revision"] != native || settings["vcs.modified"] != "false" || sdk != "v0.2.9-experimental" {
		return errors.New("clean pinned main/SDK required")
	}
	return nil
}

func pairedExecuteGo(goBinary string, value nativeResult, doc document) ([]int64, error) {
	var inputs []int64
	for _, c := range doc.Cases {
		inputs = append(inputs, c.Input)
	}
	raw, err := executeFunction(context.Background(), goBinary, value.Source, "ChoosePath", inputs)
	if err != nil {
		return nil, err
	}
	var values []int64
	if err = json.Unmarshal(raw, &values); err != nil || len(values) != len(doc.Cases) {
		return nil, errors.New("compiled Go values malformed")
	}
	for i, c := range doc.Cases {
		if values[i] != c.Expected {
			return nil, errors.New("compiled Go independent actual differs")
		}
	}
	return values, nil
}

func runPairedNative(binary, goBinary, curriculum, models, selectionPath, output, revision, native string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	if err := pairedNativeBinary(binary, native); err != nil {
		return err
	}
	if err := auditPairedCompiler(curriculum, models, filepath.Dir(selectionPath)); err != nil {
		return err
	}
	rows, err := compilerTrainingRows(curriculum)
	if err != nil {
		return err
	}
	rows, err = compilerNativeSubset(rows)
	if err != nil {
		return err
	}
	loaded, paths, selection, err := pairedNativeModels(models, selectionPath)
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
	selectionRaw, err := read(selectionPath)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	pins := map[string]map[string]string{}
	for arm, model := range loaded {
		pins[arm] = map[string]string{"metadata": model.MetadataSHA256(), "weights": model.WeightsSHA256()}
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/paired-compiler-native-preexecution/v1", "runner_revision": revision, "native_revision": native, "native_binary_sha256": binarySHA, "go_binary_sha256": goSHA, "model_pins": pins, "selected_candidate": selection.ID, "selection_sha256": hash(selectionRaw), "planned_native_calls": 120, "planned_model_predictions": 80, "planned_compiled_go_executions": 15, "planned_go_function_invocations": 180, "new_independent_intentions": 0}); err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "gooo-paired-native-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	report := pairedNativeReport{Schema: "gooo/paired-compiler-native/v1", Status: "PASS", Selected: selection.ID, Scope: "40 reused development views, selected/reference/disconnected arms; child CPU is normalized to one core, not host utilization; finite actuals and exact generated Go checked"}
	first := map[string]bool{}
	for _, row := range rows {
		doc, source, err := compilerNativeDocument(row)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(workspace, "source.gooo"), source, 0600); err != nil {
			return err
		}
		if err = save(filepath.Join(workspace, "plan.json"), doc); err != nil {
			return err
		}
		docRaw, err := read(filepath.Join(workspace, "plan.json"))
		if err != nil {
			return err
		}
		for _, arm := range pairedNativeArms {
			id := row.ID + "-" + arm
			args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--activity", "ChoosePath", "source.gooo"}
			if arm != "offline" {
				args = append(args, "--path-model", paths[arm])
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
			if value.Report.Paths.DocumentSHA != "sha256:"+hash(docRaw) {
				return errors.New("native document hash differs")
			}
			cell := compilerNativeCell{ID: id, Row: row.ID, Arm: arm, Capture: hash(raw), Metrics: m}
			if !first[row.Family] {
				cell.GoValues, err = pairedExecuteGo(goBinary, value, doc)
				if err != nil {
					return err
				}
				cell.GoExecution = true
				report.Executions++
				report.Invocations += len(cell.GoValues)
			}
			report.Cells = append(report.Cells, cell)
			report.Calls++
			report.Predictions += value.Report.Paths.Search.Selection.ModelCalls
		}
		first[row.Family] = true
	}
	if report.Calls != 120 || report.Predictions != 80 || report.Executions != 15 || report.Invocations != 180 {
		return errors.New("paired native execution denominator differs")
	}
	return save(filepath.Join(output, "report.json"), report)
}

func auditPairedNative(curriculum, models, selectionPath, output, native string) error {
	if native != pairedNativeMain {
		return errors.New("pinned native main required")
	}
	rows, err := compilerTrainingRows(curriculum)
	if err != nil {
		return err
	}
	rows, err = compilerNativeSubset(rows)
	if err != nil {
		return err
	}
	loaded, _, selection, err := pairedNativeModels(models, selectionPath)
	if err != nil {
		return err
	}
	raw, err := read(filepath.Join(output, "report.json"))
	if err != nil {
		return err
	}
	var report pairedNativeReport
	if err = json.Unmarshal(raw, &report); err != nil || report.Schema != "gooo/paired-compiler-native/v1" || report.Status != "PASS" ||
		report.Selected != selection.ID || len(report.Cells) != 120 || report.Calls != 120 || report.Predictions != 80 || report.Executions != 15 || report.Invocations != 180 {
		return errors.New("paired native report denominator differs")
	}
	index, executions, invocations, predictions := 0, 0, 0, 0
	first := map[string]bool{}
	for _, row := range rows {
		doc, _, err := compilerNativeDocument(row)
		if err != nil {
			return err
		}
		docRaw, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		docRaw = append(docRaw, '\n')
		for _, arm := range pairedNativeArms {
			cell := report.Cells[index]
			index++
			id := row.ID + "-" + arm
			if cell.ID != id || cell.Row != row.ID || cell.Arm != arm || cell.GoExecution == first[row.Family] {
				return errors.New("paired native row/order/execution subset differs")
			}
			raw, err := read(filepath.Join(output, id+".json"))
			if err != nil || hash(raw) != cell.Capture {
				return errors.New("paired native raw capture hash differs")
			}
			value, err := inspectCompilerNative(raw, row, arm, native, loaded[arm])
			if err != nil {
				return err
			}
			if value.Report.Paths.DocumentSHA != "sha256:"+hash(docRaw) {
				return errors.New("captured document hash differs")
			}
			predictions += value.Report.Paths.Search.Selection.ModelCalls
			if cell.GoExecution {
				if len(cell.GoValues) != len(doc.Cases) {
					return errors.New("captured Go invocation count differs")
				}
				for i, c := range doc.Cases {
					if cell.GoValues[i] != c.Expected {
						return errors.New("captured Go actual differs")
					}
				}
				executions++
				invocations += len(cell.GoValues)
			} else if len(cell.GoValues) != 0 {
				return errors.New("unexecuted cell has Go values")
			}
		}
		first[row.Family] = true
	}
	if predictions != 80 || executions != 15 || invocations != 180 {
		return errors.New("reconstructed native denominator differs")
	}
	return nil
}
