package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const nativeDiagnosisFixture = "studies/native-path-diagnosis-v1/"
const nativeDiagnosisPrereg = "docs/native-path-diagnosis-preregistration.md"

type nativeDiagnosisPre struct {
	Schema   string            `json:"schema"`
	Runner   string            `json:"runner_revision"`
	Native   string            `json:"native_revision"`
	Binary   string            `json:"native_binary_sha256"`
	GoBinary string            `json:"go_binary_sha256"`
	Prereg   string            `json:"preregistration_sha256"`
	Files    map[string]string `json:"input_files_sha256"`
}

func nativeDiagnosisDocument(language string) (document, error) {
	raw, err := read(nativeDiagnosisFixture + "plan.json")
	var d document
	if err != nil || json.Unmarshal(raw, &d) != nil || len(d.Plan.Decisions) != 1 {
		return d, errors.New("fixed native diagnosis plan required")
	}
	d.Plan.Decisions[0].Intent = "Subtract two from the input."
	if language == "ko" {
		d.Plan.Decisions[0].Intent = "입력값에서 2를 뺀다."
	}
	return d, nil
}

func nativeDiagnosisFunction(source string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "projection.go", source, parser.SkipObjectResolution)
	if err != nil || len(file.Decls) != 1 {
		return "", errors.New("one diagnostic function required")
	}
	function, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || function.Name.Name != "Probe" || function.Recv != nil || function.Body == nil {
		return "", errors.New("compiler-owned Probe required")
	}
	normalizeCompoundBlock(function.Body)
	var signature, body bytes.Buffer
	if err = format.Node(&signature, fset, function.Type); err != nil {
		return "", err
	}
	if err = format.Node(&body, fset, function.Body); err != nil {
		return "", err
	}
	return signature.String() + body.String(), nil
}

func nativeDiagnosisOracle(mask uint16, x int64) int64 {
	if mask == 0 {
		return x - 2
	}
	return 2 - x
}

func nativeDiagnosisVectorHash(mask uint16, inputs []int64) string {
	digest := sha256.New()
	for _, input := range inputs {
		var pair [16]byte
		binary.LittleEndian.PutUint64(pair[:8], uint64(input))
		binary.LittleEndian.PutUint64(pair[8:], uint64(nativeDiagnosisOracle(mask, input)))
		digest.Write(pair[:])
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func inspectNativeDiagnosis(v nativeResult, language, arm, native string, enabled bool) (uint16, error) {
	doc, err := nativeDiagnosisDocument(language)
	prepared, prepErr := pathplan.Prepare(doc.Plan)
	p := v.Report.Paths
	if err != nil || prepErr != nil || v.Report.Compiler != native || v.Report.Decision != "PASS" || !v.Report.Types ||
		!v.Report.Replay || v.Report.Writes != 0 || !p.Bound || p.Search.Evaluated != 1 || p.Search.TypeRejected != 0 ||
		p.Search.TrainingTotal != 1 || p.Search.SelectedTrainingPassed != 1 || p.Completeness != 100 || len(p.Cases) != 1 ||
		p.Cases[0].Input != 2 || p.Cases[0].Expected != 0 || p.Cases[0].Actual != 0 || !p.Cases[0].Passed ||
		p.Search.Selection.ExternalCalls != 0 || !p.Search.Selection.ExternalCallsKnown {
		return 0, errors.New("native finite/source contract differs")
	}
	selection := p.Search.Selection
	calls := 0
	if arm == "fp32" {
		calls = 1
	}
	if selection.ModelCalls != calls || (calls == 1 && (selection.MetadataSHA256 != diagnosisModelPin || selection.WeightsSHA256 != diagnosisWeightsPin)) ||
		(calls == 0 && (selection.MetadataSHA256 != "" || selection.WeightsSHA256 != "")) {
		return 0, errors.New("native prediction accounting differs")
	}
	mask := uint16(0)
	if selection.Choices["operands"] == "layout_reverse" {
		mask = 1
	} else if selection.Choices["operands"] != "layout_forward" {
		return 0, errors.New("undeclared selected path")
	}
	body, err := prepared.Compile(selection.Choices)
	if err != nil {
		return 0, err
	}
	actual, e1 := nativeDiagnosisFunction(v.Source)
	typed, e2 := nativeDiagnosisFunction(body.GoSource())
	if e1 != nil || e2 != nil || actual != typed {
		return 0, errors.New("native/typed selected projection differs")
	}
	if !enabled {
		if p.Diagnosis != nil || p.DiagnosisBudget != 0 || p.DiagnosisSHA != "" || p.Timing.Diagnosis != 0 {
			return 0, errors.New("default diagnosis leaked")
		}
		return mask, nil
	}
	d := p.Diagnosis
	casesRaw, _ := json.Marshal(doc.Cases)
	probesRaw, _ := json.Marshal([]int64{2, 3})
	if d == nil || d.Schema != "gooo/typed-path-diagnosis/v1" || d.Status != "COMPLETE" || d.ReferenceMask != mask ||
		d.PlanSHA256 != prepared.PlanSHA256() || d.CasesSHA256 != hash(casesRaw) || d.ProbesSHA256 != hash(probesRaw) ||
		d.Declared != 2 || d.Observed != 2 || d.Unobserved != 0 || d.Typed != 2 || d.TypeRejected != 0 ||
		d.CaseIndistinguishable != 2 || d.ProbeDistinguished != 1 || d.ProbeUnresolved != 1 || d.ModelPredictions != 0 ||
		len(d.Candidates) != 2 || p.DiagnosisBudget != 2 || p.Timing.Diagnosis <= 0 ||
		p.DiagnosisSHA != "sha256:"+hash([]byte(`{"inputs":[2,3],"max_candidates":2}`)) {
		return 0, errors.New("native diagnosis bounds differ")
	}
	for index, c := range d.Candidates {
		want := mask
		if index == 1 {
			want = 1 - mask
		}
		if c.Mask != want || c.Status != "EVALUATED" || c.Passed != 1 || !c.CaseIndistinguishable ||
			c.CaseOutputsSHA256 != nativeDiagnosisVectorHash(want, []int64{2}) ||
			c.ProbeOutputsSHA256 != nativeDiagnosisVectorHash(want, []int64{2, 3}) {
			return 0, errors.New("candidate outputs differ from arithmetic oracle")
		}
		if index == 0 && c.Witness != nil {
			return 0, errors.New("reference got a witness")
		}
		if index == 1 && (c.Witness == nil || c.Witness.Input != 3 || c.Witness.Reference != nativeDiagnosisOracle(mask, 3) ||
			c.Witness.Alternative != nativeDiagnosisOracle(want, 3)) {
			return 0, errors.New("separating observation differs")
		}
	}
	return mask, nil
}

func runNativeDiagnosisSmoke(nativeBinary, nativeRoot, goBinary, output, revision, native string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(nativeBinary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 native binary required")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	sdk := ""
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = d.Version
		}
	}
	if settings["vcs.revision"] != native || settings["vcs.modified"] != "false" || sdk != "v0.2.8-experimental" {
		return errors.New("clean pinned native SDK 0.2.8 required")
	}
	model, err := decision.LoadPath(diagnosisModel)
	if err != nil || model.MetadataSHA256() != diagnosisModelPin || model.WeightsSHA256() != diagnosisWeightsPin {
		return errors.New("frozen own model required")
	}
	modelPath, err := filepath.Abs(diagnosisModel)
	if err != nil {
		return err
	}
	binSHA, err := executableHash(nativeBinary)
	if err != nil {
		return err
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	goInfo, err := buildinfo.ReadFile(goBinary)
	if err != nil || goInfo.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 execution toolchain required")
	}
	prereg, err := read(nativeDiagnosisPrereg)
	if err != nil {
		return err
	}
	inputs := map[string][]byte{}
	for name, nativeName := range map[string]string{"source.gooo.fixture": "path-diagnosis.gooo.fixture", "diagnosis.json": "path-diagnosis-inputs.json", "plan.json": "path-diagnosis-plan.json"} {
		raw, e := read(nativeDiagnosisFixture + name)
		if e != nil {
			return e
		}
		original, e := read(filepath.Join(nativeRoot, "examples/body-codegen", nativeName))
		if e != nil || !bytes.Equal(raw, original) {
			return errors.New("native fixture pin differs")
		}
		inputs[name] = raw
	}
	for _, lang := range []string{"ko", "en"} {
		d, e := nativeDiagnosisDocument(lang)
		if e != nil {
			return e
		}
		raw, _ := json.Marshal(d)
		inputs["plan-"+lang+".json"] = raw
	}
	if err = os.MkdirAll(filepath.Join(output, "executions"), 0755); err != nil {
		return err
	}
	pre := nativeDiagnosisPre{"gooo/native-path-diagnosis-preexecution/v1", revision, native, binSHA, goSHA, hash(prereg), map[string]string{}}
	for name, raw := range inputs {
		if err = os.WriteFile(filepath.Join(output, name), raw, 0644); err != nil {
			return err
		}
		pre.Files[name] = hash(raw)
	}
	if err = save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	absOutput, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	executions := map[string]bool{}
	for i, lang := range []string{"ko", "en"} {
		for j, arm := range []string{"offline", "fp32"} {
			order := []bool{false, true}
			if (i*2+j)%2 != 0 {
				order[0], order[1] = order[1], order[0]
			}
			for _, enabled := range order {
				id := lang + "-" + arm + "-off"
				if enabled {
					id = lang + "-" + arm + "-on"
				}
				args := []string{"body-codegen", "--json", "--path-plan", "plan-" + lang + ".json", "--activity", "Probe"}
				if arm == "fp32" {
					args = append(args, "--path-model", modelPath)
				}
				if enabled {
					args = append(args, "--path-diagnosis", "diagnosis.json")
				}
				args = append(args, "source.gooo.fixture")
				raw, m, callErr := child(ctx, absOutput, nativeBinary, args...)
				if err = os.WriteFile(filepath.Join(output, id+".json"), raw, 0644); err != nil {
					return err
				}
				if err = save(filepath.Join(output, id+"-metrics.json"), m); err != nil {
					return err
				}
				if callErr != nil {
					return callErr
				}
				var value nativeResult
				if err = json.Unmarshal(raw, &value); err != nil {
					return err
				}
				if _, err = inspectNativeDiagnosis(value, lang, arm, native, enabled); err != nil {
					return err
				}
				sha := hash([]byte(value.Source))
				if !executions[sha] {
					executed, e := executeFunction(ctx, goBinary, value.Source, "Probe", []int64{2, 3})
					if e != nil {
						return e
					}
					var values []int64
					if json.Unmarshal(executed, &values) != nil {
						return errors.New("actual Go output invalid")
					}
					capture := map[string]any{"generated_go": value.Source, "generated_go_sha256": sha, "inputs": []int64{2, 3}, "actual_go_values": values}
					if err = save(filepath.Join(output, "executions", sha+".json"), capture); err != nil {
						return err
					}
					executions[sha] = true
				}
			}
		}
	}
	value, err := auditNativeDiagnosisSmoke(output, revision, native)
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "report.json"), value); err != nil {
		return err
	}
	return save(filepath.Join(output, "audit.json"), value)
}

func auditNativeDiagnosisSmoke(output, revision, native string) (map[string]any, error) {
	raw, err := read(filepath.Join(output, "preexecution.json"))
	var pre nativeDiagnosisPre
	if err != nil || json.Unmarshal(raw, &pre) != nil || pre.Schema != "gooo/native-path-diagnosis-preexecution/v1" || pre.Runner != revision || pre.Native != native {
		return nil, errors.New("native diagnosis preexecution differs")
	}
	prereg, err := read(nativeDiagnosisPrereg)
	if err != nil || hash(prereg) != pre.Prereg {
		return nil, errors.New("preregistration changed")
	}
	files := map[string]string{"preexecution.json": hash(raw)}
	expectedInputs := map[string][]byte{}
	for _, name := range []string{"source.gooo.fixture", "diagnosis.json", "plan.json"} {
		data, e := read(nativeDiagnosisFixture + name)
		if e != nil {
			return nil, e
		}
		expectedInputs[name] = data
	}
	for _, lang := range []string{"ko", "en"} {
		doc, e := nativeDiagnosisDocument(lang)
		if e != nil {
			return nil, e
		}
		data, _ := json.Marshal(doc)
		expectedInputs["plan-"+lang+".json"] = data
	}
	if len(pre.Files) != 5 {
		return nil, errors.New("fixed five inputs required")
	}
	for name, expected := range expectedInputs {
		pin := pre.Files[name]
		data, e := read(filepath.Join(output, name))
		if e != nil || hash(data) != pin || !bytes.Equal(data, expected) {
			return nil, errors.New("input bytes changed")
		}
		files[name] = pin
	}
	sources := map[string]bool{}
	predictions := 0
	diagnoses := 0
	var diagnosticMS []float64
	for _, lang := range []string{"ko", "en"} {
		for _, arm := range []string{"offline", "fp32"} {
			var pair [2]nativeResult
			for index, mode := range []string{"off", "on"} {
				id := lang + "-" + arm + "-" + mode
				data, e := read(filepath.Join(output, id+".json"))
				if e != nil || json.Unmarshal(data, &pair[index]) != nil {
					return nil, errors.New("native capture invalid")
				}
				files[id+".json"] = hash(data)
				mask, e := inspectNativeDiagnosis(pair[index], lang, arm, native, index == 1)
				if e != nil {
					return nil, e
				}
				predictions += pair[index].Report.Paths.Search.Selection.ModelCalls
				if index == 1 {
					diagnoses++
					diagnosticMS = append(diagnosticMS, pair[index].Report.Paths.Timing.Diagnosis)
				}
				cost, e := read(filepath.Join(output, id+"-metrics.json"))
				var m metrics
				if e != nil || json.Unmarshal(cost, &m) != nil || m.Wall <= 0 || m.User < 0 || m.System < 0 || m.RSS <= 0 {
					return nil, errors.New("process metrics invalid")
				}
				files[id+"-metrics.json"] = hash(cost)
				sha := hash([]byte(pair[index].Source))
				name := "executions/" + sha + ".json"
				ex, e := read(filepath.Join(output, name))
				var execution struct {
					Source string  `json:"generated_go"`
					SHA    string  `json:"generated_go_sha256"`
					Inputs []int64 `json:"inputs"`
					Values []int64 `json:"actual_go_values"`
				}
				if e != nil || json.Unmarshal(ex, &execution) != nil || execution.Source != pair[index].Source || execution.SHA != sha ||
					!reflect.DeepEqual(execution.Inputs, []int64{2, 3}) || !reflect.DeepEqual(execution.Values, []int64{0, nativeDiagnosisOracle(mask, 3)}) {
					return nil, errors.New("actual Go values differ from independent oracle")
				}
				files[name] = hash(ex)
				sources[sha] = true
			}
			if pair[0].Source != pair[1].Source || !reflect.DeepEqual(pair[0].Report.Paths.Search.Selection.Choices, pair[1].Report.Paths.Search.Selection.Choices) {
				return nil, errors.New("diagnosis changed construction")
			}
		}
	}
	return map[string]any{"schema": "gooo/native-path-diagnosis-smoke/v1", "decision": "PASS", "runner_revision": revision, "native_revision": native,
		"native_binary_sha256": pre.Binary, "actual_native_calls": 8, "actual_initial_model_predictions": predictions, "diagnoses": diagnoses,
		"candidate_observations": diagnoses * 2, "repeated_separating_witnesses": diagnoses, "same_source_pairs": 4, "finite_passes": 8, "finite_cases": 8,
		"actual_generated_go_processes": len(sources), "actual_generated_function_invocations": len(sources) * 2, "diagnostic_ms": diagnosticMS,
		"files_sha256": files, "new_audit_predictions": 0, "new_audit_native_calls": 0, "new_audit_go_processes": 0,
		"scope": "One synthetic intention, two language views and four repeated pairs. Finite test passing is not intent correctness. Independent arithmetic output/witness and selected typed/native function reconciliation; model ranking origin relies on pinned clean runner/model and source CI, not offline inference replay. Only actually selected emitted bodies are Go-executed. Whole-child costs are raw integration observations, not causal timing or host utilization estimates. No training/GPU/upstream Laya/model promotion."}, nil
}
