package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const bilingualModels = "runs/bilingual-judgment-mps-20261001/"

func bilingualDocument(language string, full bool) (document, error) {
	d, err := nativeDiagnosisDocument(language)
	if full {
		d.Cases = []pathplan.TestCase{{Input: 2, Expected: 0}, {Input: 3, Expected: 1}}
		d.Max = 2
	}
	return d, err
}

func inspectBilingual(v nativeResult, language, arm string, full bool, native string) (uint16, error) {
	r, p := v.Report, v.Report.Paths
	wantCalls, wantCases := 1, 1
	if arm == "offline" {
		wantCalls = 0
	}
	if full {
		wantCases = 2
	}
	mask := uint16(0)
	if p.Search.Selection.Choices["operands"] == "layout_reverse" {
		mask = 1
	} else if p.Search.Selection.Choices["operands"] != "layout_forward" {
		return 0, errors.New("unknown compiler operand choice")
	}
	if r.Decision != "PASS" || r.Compiler != native || !r.Types || !r.Replay || r.Writes != 0 ||
		!p.Bound || p.Search.Selection.ModelCalls != wantCalls || p.Search.Selection.ExternalCalls != 0 ||
		p.Search.TrainingTotal != wantCases || p.Search.SelectedTrainingPassed != wantCases ||
		p.Search.Evaluated < 1 || p.Search.Evaluated > wantCases || len(p.Feedback) != 0 ||
		(full && mask != 0) || p.Diagnosis != nil || len(p.Cases) != wantCases {
		return 0, errors.New("bounded native bilingual contract differs")
	}
	d, err := bilingualDocument(language, full)
	if err != nil {
		return 0, err
	}
	prepared, err := pathplan.Prepare(d.Plan)
	if err != nil {
		return 0, err
	}
	if p.Search.Selection.PlanSHA256 != prepared.PlanSHA256() {
		return 0, errors.New("native typed plan digest differs")
	}
	body, err := prepared.Compile(p.Search.Selection.Choices)
	if err != nil {
		return 0, err
	}
	actual, err := nativeDiagnosisFunction(v.Source)
	if err != nil {
		return 0, err
	}
	expected, err := nativeDiagnosisFunction(body.GoSource())
	if err != nil || actual != expected {
		return 0, errors.New("selected native function differs from typed path")
	}
	for i, c := range p.Cases {
		if c.Input != d.Cases[i].Input || c.Expected != d.Cases[i].Expected ||
			c.Actual != nativeDiagnosisOracle(mask, c.Input) || !c.Passed {
			return 0, errors.New("native finite observations differ from arithmetic oracle")
		}
	}
	return mask, nil
}

func runBilingualSmoke(binary, goBinary, output, revision, native string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" || native != "1e01c96c54f2f8dd43334b8f580af93ffaea24df" {
		return errors.New("exact adopted main compiler required")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["vcs.revision"] != native || settings["vcs.modified"] != "false" {
		return errors.New("clean native source required")
	}
	models, pins := map[string]string{}, map[string]string{}
	for _, arm := range []string{"control", "paired"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			name := arm + "-" + variant
			filename := filepath.Join(bilingualModels, arm, "models", variant, "model.json")
			model, err := decision.LoadPath(filename)
			if err != nil {
				return err
			}
			models[name], err = filepath.Abs(filename)
			if err != nil {
				return err
			}
			pins[name] = model.MetadataSHA256()
		}
	}
	if err := os.Mkdir(output, 0755); err != nil {
		return err
	}
	fixture, err := read(nativeDiagnosisFixture + "source.gooo.fixture")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "fixture.gooo"), fixture, 0644); err != nil {
		return err
	}
	binSHA, err := executableHash(binary)
	if err != nil {
		return err
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	prereg, err := read("docs/bilingual-judgment-preregistration.md")
	if err != nil {
		return err
	}
	files := map[string]string{"fixture.gooo": hash(fixture)}
	for _, language := range []string{"en", "ko"} {
		for _, full := range []bool{false, true} {
			d, err := bilingualDocument(language, full)
			if err != nil {
				return err
			}
			contract := "sparse"
			if full {
				contract = "full"
			}
			name := "plan-" + language + "-" + contract + ".json"
			if err := save(filepath.Join(output, name), d); err != nil {
				return err
			}
			raw, err := read(filepath.Join(output, name))
			if err != nil {
				return err
			}
			files[name] = hash(raw)
		}
	}
	pre := map[string]any{"schema": "gooo/bilingual-native-preexecution/v1", "runner": revision, "native": native,
		"native_binary_sha256": binSHA, "go_binary_sha256": goSHA, "model_metadata_sha256": pins,
		"preregistration_sha256": hash(prereg), "input_sha256": files, "maximum_native_calls": 28,
		"scope": "One reused subtraction intention, two languages, six trained arms and disconnected control; separate sparse and explicit full contracts. Full calls rerank once, rather than resuming the sparse call. No feedback training, upstream Laya or new independent tasks."}
	if err := save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	root, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	executions, counts, captures := map[string]diagnosisExecution{}, map[string]int{}, map[string]string{}
	for _, language := range []string{"en", "ko"} {
		for _, arm := range []string{"offline", "control-fp32", "control-ptq_ternary", "control-qat_ternary", "paired-fp32", "paired-ptq_ternary", "paired-qat_ternary"} {
			for _, full := range []bool{false, true} {
				contract := "sparse"
				if full {
					contract = "full"
				}
				id := language + "-" + arm + "-" + contract
				args := []string{"body-codegen", "--json", "--path-plan", "plan-" + language + "-" + contract + ".json", "--activity", "Probe", "fixture.gooo"}
				if arm != "offline" {
					args = append(args, "--path-model", models[arm])
				}
				raw, metrics, failure := child(ctx, root, binary, args...)
				if err := os.WriteFile(filepath.Join(output, id+".json"), raw, 0644); err != nil {
					return err
				}
				if err := save(filepath.Join(output, id+"-metrics.json"), metrics); err != nil {
					return err
				}
				captures[id+".json"] = hash(raw)
				metricRaw, err := read(filepath.Join(output, id+"-metrics.json"))
				if err != nil {
					return err
				}
				captures[id+"-metrics.json"] = hash(metricRaw)
				if failure != nil {
					return failure
				}
				var value nativeResult
				if err := json.Unmarshal(raw, &value); err != nil {
					return err
				}
				mask, err := inspectBilingual(value, language, arm, full, native)
				if err != nil {
					return err
				}
				p := value.Report.Paths
				if arm != "offline" && p.Search.Selection.MetadataSHA256 != pins[arm] {
					return errors.New("native model metadata pin differs")
				}
				counts["native_calls"]++
				counts["model_predictions"] += p.Search.Selection.ModelCalls
				if !full && mask == 1 {
					counts["sparse_selected_wrong_intention"]++
				}
				if full {
					counts["full_contract_complete"]++
					counts["full_extra_candidate_attempts"] += p.Search.Evaluated - 1
				}
				sha := hash([]byte(value.Source))
				executions[sha] = diagnosisExecution{Schema: "gooo/path-diagnosis-go-execution/v1", Mask: mask,
					Source: value.Source, SHA: sha, Inputs: []int64{2, 3}}
			}
		}
	}
	executionHashes := map[string]string{}
	for sha, execution := range executions {
		raw, err := executeFunction(ctx, goBinary, execution.Source, "Probe", execution.Inputs)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &execution.Values); err != nil || len(execution.Values) != 2 {
			return errors.New("actual emitted-Go values missing")
		}
		for i, input := range execution.Inputs {
			if execution.Values[i] != nativeDiagnosisOracle(execution.Mask, input) {
				return errors.New("actual emitted-Go values differ from independent oracle")
			}
		}
		name := "execution-" + sha + ".json"
		if err := save(filepath.Join(output, name), execution); err != nil {
			return err
		}
		raw, _ = read(filepath.Join(output, name))
		executionHashes[name] = hash(raw)
	}
	return save(filepath.Join(output, "report.json"), map[string]any{
		"schema": "gooo/bilingual-native-smoke/v1", "decision": "PASS", "counts": counts,
		"actual_generated_go_processes": len(executions), "actual_go_function_invocations": 2 * len(executions),
		"execution_sha256": executionHashes, "new_independent_intentions": 0, "scope": pre["scope"],
		"capture_sha256": captures,
		"limitation":     "Fixed arm order and 28 child calls are integration evidence, not a causal latency or host utilization experiment. Full contracts are new native calls, not zero-call continuation."})
}

func auditBilingualSmoke(output, revision, native string) (map[string]any, error) {
	raw, err := read(filepath.Join(output, "report.json"))
	if err != nil {
		return nil, err
	}
	var report struct {
		Schema      string            `json:"schema"`
		Counts      map[string]int    `json:"counts"`
		Captures    map[string]string `json:"capture_sha256"`
		Executions  map[string]string `json:"execution_sha256"`
		Processes   int               `json:"actual_generated_go_processes"`
		Invocations int               `json:"actual_go_function_invocations"`
	}
	if err := json.Unmarshal(raw, &report); err != nil || report.Schema != "gooo/bilingual-native-smoke/v1" ||
		len(report.Captures) != 56 || report.Processes != len(report.Executions) || report.Invocations != 2*report.Processes {
		return nil, errors.New("bounded native report layout differs")
	}
	reportSHA := hash(raw)
	raw, err = read(filepath.Join(output, "preexecution.json"))
	if err != nil {
		return nil, err
	}
	preSHA := hash(raw)
	var pre struct {
		Runner string            `json:"runner"`
		Native string            `json:"native"`
		Models map[string]string `json:"model_metadata_sha256"`
		Inputs map[string]string `json:"input_sha256"`
	}
	if json.Unmarshal(raw, &pre) != nil || pre.Runner != revision || pre.Native != native ||
		len(pre.Models) != 6 || len(pre.Inputs) != 5 {
		return nil, errors.New("native preexecution identity differs")
	}
	for name, pin := range pre.Inputs {
		raw, err := read(filepath.Join(output, name))
		if err != nil || hash(raw) != pin {
			return nil, errors.New("native input digest differs")
		}
	}
	executions := map[string]diagnosisExecution{}
	for name, pin := range report.Executions {
		raw, err := read(filepath.Join(output, name))
		var execution diagnosisExecution
		if err != nil || hash(raw) != pin || json.Unmarshal(raw, &execution) != nil ||
			hash([]byte(execution.Source)) != execution.SHA || execution.Mask > 1 ||
			len(execution.Values) != 2 || len(execution.Inputs) != 2 || execution.Inputs[0] != 2 || execution.Inputs[1] != 3 {
			return nil, errors.New("actual generated-Go evidence differs")
		}
		for i, input := range execution.Inputs {
			if execution.Values[i] != nativeDiagnosisOracle(execution.Mask, input) {
				return nil, errors.New("actual generated-Go value differs")
			}
		}
		executions[execution.SHA] = execution
	}
	counts := map[string]int{}
	for _, language := range []string{"en", "ko"} {
		for _, arm := range []string{"offline", "control-fp32", "control-ptq_ternary", "control-qat_ternary", "paired-fp32", "paired-ptq_ternary", "paired-qat_ternary"} {
			for _, full := range []bool{false, true} {
				contract := "sparse"
				if full {
					contract = "full"
				}
				id := language + "-" + arm + "-" + contract
				raw, err := read(filepath.Join(output, id+".json"))
				var v nativeResult
				if err != nil || hash(raw) != report.Captures[id+".json"] || json.Unmarshal(raw, &v) != nil {
					return nil, errors.New("native raw capture digest differs")
				}
				mask, err := inspectBilingual(v, language, arm, full, native)
				if err != nil {
					return nil, err
				}
				if arm != "offline" && v.Report.Paths.Search.Selection.MetadataSHA256 != pre.Models[arm] {
					return nil, errors.New("recorded native model pin differs")
				}
				execution, ok := executions[hash([]byte(v.Source))]
				if !ok || execution.Mask != mask {
					return nil, errors.New("selected native body lacks actual Go evidence")
				}
				raw, err = read(filepath.Join(output, id+"-metrics.json"))
				var m metrics
				if err != nil || hash(raw) != report.Captures[id+"-metrics.json"] || json.Unmarshal(raw, &m) != nil ||
					m.Wall <= 0 || m.RSS <= 0 || m.User < 0 || m.System < 0 {
					return nil, errors.New("recorded native process metrics differ")
				}
				counts["native_calls"]++
				counts["model_predictions"] += v.Report.Paths.Search.Selection.ModelCalls
				if !full && mask == 1 {
					counts["sparse_selected_wrong_intention"]++
				}
				if full {
					counts["full_contract_complete"]++
					counts["full_extra_candidate_attempts"] += v.Report.Paths.Search.Evaluated - 1
				}
			}
		}
	}
	if !reflect.DeepEqual(counts, report.Counts) {
		return nil, errors.New("native count reconciliation differs")
	}
	return map[string]any{"schema": "gooo/bilingual-native-audit/v1", "decision": "PASS", "report_sha256": reportSHA,
		"preexecution_sha256": preSHA, "recorded_native_calls": 28, "recorded_model_predictions": 24,
		"recorded_go_processes": report.Processes, "recorded_go_function_invocations": report.Invocations,
		"new_native_calls": 0, "new_model_calls": 0, "new_go_processes": 0,
		"scope": "Offline reconciliation of captures, source/model/test identities, selected typed/native function and actual emitted-Go oracle values; ranking origin relies on pinned runner and native source evidence. No inference or subprocess replay."}, nil
}
