package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func diagnosisOracleVector(template string, mask uint16, inputs []int64) ([]int64, string, error) {
	values := make([]int64, len(inputs))
	digest := sha256.New()
	for i, input := range inputs {
		value, err := compoundstudy.Oracle(template, mask, input)
		if err != nil {
			return nil, "", err
		}
		values[i] = value
		var pair [16]byte
		binary.LittleEndian.PutUint64(pair[:8], uint64(input))
		binary.LittleEndian.PutUint64(pair[8:], uint64(value))
		_, _ = digest.Write(pair[:])
	}
	return values, hex.EncodeToString(digest.Sum(nil)), nil
}

func diagnosisOracleCandidate(row compoundstudy.Case, mask, reference uint16) (pathplan.CandidateDiagnosis, error) {
	var inputs []int64
	for _, test := range row.Document.Cases {
		inputs = append(inputs, test.Input)
	}
	values, caseSHA, err := diagnosisOracleVector(row.Template, mask, inputs)
	if err != nil {
		return pathplan.CandidateDiagnosis{}, err
	}
	refValues, _, err := diagnosisOracleVector(row.Template, reference, inputs)
	if err != nil {
		return pathplan.CandidateDiagnosis{}, err
	}
	probeValues, probeSHA, err := diagnosisOracleVector(row.Template, mask, diagnosisProbes())
	if err != nil {
		return pathplan.CandidateDiagnosis{}, err
	}
	refProbes, _, err := diagnosisOracleVector(row.Template, reference, diagnosisProbes())
	if err != nil {
		return pathplan.CandidateDiagnosis{}, err
	}
	program, err := pathplan.Compile(row.Document.Plan, compoundstudy.Choices(row.Document.Plan, mask))
	if err != nil {
		return pathplan.CandidateDiagnosis{}, err
	}
	value := pathplan.CandidateDiagnosis{Mask: mask, Status: "EVALUATED", GoooSHA256: hash([]byte(program.GoooSource())),
		CaseOutputsSHA256: caseSHA, ProbeOutputsSHA256: probeSHA, CaseIndistinguishable: reflect.DeepEqual(values, refValues)}
	for i, test := range row.Document.Cases {
		if values[i] == test.Expected {
			value.Passed++
		}
	}
	if value.CaseIndistinguishable {
		for i, output := range probeValues {
			if output != refProbes[i] {
				value.Witness = &pathplan.DistinguishingInput{Input: diagnosisProbes()[i], Reference: refProbes[i], Alternative: output}
				break
			}
		}
	}
	return value, nil
}

func validateDiagnosisSearch(row compoundstudy.Case, capture diagnosisCapture, mask uint16) error {
	s := capture.Search
	wantCalls := 0
	if capture.Arm == "fp32" {
		wantCalls = 2
	} else if capture.Arm != "offline" {
		return errors.New("unknown diagnosis arm")
	}
	if capture.ID != row.ID || capture.SearchNS <= 0 || capture.DiagnosisNS <= 0 || s.Schema != "gooo/typed-path-tdd-search/v1" ||
		s.Selection.ModelCalls != wantCalls || s.Selection.ExternalCalls != 0 || !s.Selection.ExternalCallsKnown ||
		len(s.Selection.Receipts) != 2 || len(s.Attempts) < 1 || len(s.Attempts) > 2 || s.TrainingTotal != len(row.Document.Cases) ||
		s.DeclaredCombinations != 4 || s.TypeRejected != 0 || s.Evaluated != len(s.Attempts) || s.Unattempted != 4-len(s.Attempts) {
		return errors.New("diagnosis search budget or source accounting differs")
	}
	if s.Selection.SeedSHA256 != "" || wantCalls == 0 && (s.Selection.MetadataSHA256 != "" || s.Selection.WeightsSHA256 != "" || s.Selection.ModelVariant != "") ||
		wantCalls != 0 && (s.Selection.MetadataSHA256 != diagnosisModelPin || s.Selection.WeightsSHA256 != diagnosisWeightsPin || s.Selection.ModelVariant != "fp32") {
		return errors.New("diagnosis model identity differs")
	}
	seen, best := map[uint16]bool{}, -1
	for _, attempt := range s.Attempts {
		if attempt.Mask > 3 || seen[attempt.Mask] || attempt.Status != "EVALUATED" || attempt.Total != len(row.Document.Cases) ||
			len(attempt.Results) != len(row.Document.Cases) || !reflect.DeepEqual(attempt.Choices, compoundstudy.Choices(row.Document.Plan, attempt.Mask)) {
			return errors.New("diagnosis finite search attempt differs")
		}
		seen[attempt.Mask] = true
		candidate, err := diagnosisOracleCandidate(row, attempt.Mask, mask)
		if err != nil || candidate.Passed != attempt.Passed || candidate.GoooSHA256 != attempt.GoooSHA {
			return errors.New("diagnosis finite oracle differs")
		}
		best = max(best, candidate.Passed)
		for i, test := range row.Document.Cases {
			actual, _ := compoundstudy.Oracle(row.Template, attempt.Mask, test.Input)
			if attempt.Results[i] != (pathplan.TestResult{Input: test.Input, Expected: test.Expected, Actual: actual, Passed: actual == test.Expected}) {
				return errors.New("diagnosis recorded case result differs")
			}
		}
	}
	selected, err := diagnosisOracleCandidate(row, mask, mask)
	if err != nil || !seen[mask] || best != s.SelectedTrainingPassed || selected.Passed != best {
		return errors.New("diagnosis selected body is not observed best-so-far")
	}
	wantStatus := "PARTIAL"
	if best == len(row.Document.Cases) {
		wantStatus = "TRAINING_COMPLETE"
	}
	if s.Status != wantStatus {
		return errors.New("diagnosis search completion status differs")
	}
	return nil
}

func validateDiagnosisCapture(row compoundstudy.Case, capture diagnosisCapture) error {
	mask, err := compoundstudy.Mask(row.Document.Plan, capture.Search.Selection.Choices)
	if err != nil {
		return err
	}
	if err = validateDiagnosisSearch(row, capture, mask); err != nil {
		return err
	}
	planRaw, _ := json.Marshal(row.Document.Plan)
	caseRaw, _ := json.Marshal(row.Document.Cases)
	probeRaw, _ := json.Marshal(diagnosisProbes())
	d := capture.Diagnosis
	if d.Schema != "gooo/typed-path-diagnosis/v1" || d.Status != "COMPLETE" || d.PlanSHA256 != hash(planRaw) ||
		d.CasesSHA256 != hash(caseRaw) || d.ProbesSHA256 != hash(probeRaw) || d.ReferenceMask != mask || d.Declared != 4 ||
		d.Observed != 4 || d.Unobserved != 0 || d.Typed != 4 || d.TypeRejected != 0 || d.ModelPredictions != 0 ||
		d.FiniteCases != len(row.Document.Cases) || d.ProbeInputs != 31 || len(d.Candidates) != 4 || capture.Search.Selection.PlanSHA256 != d.PlanSHA256 {
		return errors.New("diagnosis observation or input binding differs")
	}
	order := []uint16{mask}
	for next := range uint16(4) {
		if next != mask {
			order = append(order, next)
		}
	}
	same, distinguished, unresolved := 0, 0, 0
	for i, next := range order {
		want, err := diagnosisOracleCandidate(row, next, mask)
		if err != nil || !reflect.DeepEqual(d.Candidates[i], want) {
			return errors.New("diagnosis candidate vector or witness differs from independent state oracle")
		}
		if want.CaseIndistinguishable {
			same++
			if want.Witness == nil {
				unresolved++
			} else {
				distinguished++
			}
		}
	}
	if d.CaseIndistinguishable != same || d.ProbeDistinguished != distinguished || d.ProbeUnresolved != unresolved {
		return errors.New("diagnosis ambiguity counts differ")
	}
	return nil
}

type diagnosisSummary struct {
	Arm             string  `json:"arm"`
	Contract        string  `json:"contract"`
	Views           int     `json:"views"`
	AllFinitePassed int     `json:"all_finite_cases_passed_views"`
	Ambiguous       int     `json:"case_ambiguous_views"`
	AmbiguousPass   int     `json:"case_ambiguous_all_pass_views"`
	Distinguishable int     `json:"probe_distinguishable_views"`
	Unresolved      int     `json:"probe_unresolved_alternative_views"`
	Predictions     int     `json:"initial_predictions"`
	SearchMedian    float64 `json:"search_median_ns"`
	DiagnosisMedian float64 `json:"diagnosis_median_ns"`
	searchTimes     []float64
	diagnosisTimes  []float64
}

func auditDiagnosisStudy(output, revision string) (map[string]any, error) {
	preRaw, err := read(filepath.Join(output, "preexecution.json"))
	if err != nil {
		return nil, err
	}
	var pre struct {
		Schema  string  `json:"schema"`
		Runner  string  `json:"runner_revision"`
		Cohort  string  `json:"cohort_sha256"`
		Prereg  string  `json:"preregistration_sha256"`
		Model   string  `json:"model_metadata_sha256"`
		Weights string  `json:"model_weights_sha256"`
		Probes  []int64 `json:"probe_inputs"`
		Search  int     `json:"search_budget"`
		Budget  int     `json:"diagnosis_budget"`
	}
	prereg, err := read(diagnosisPrereg)
	if err != nil {
		return nil, err
	}
	cohort, err := read(compoundRoot + "/cohort.jsonl")
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(preRaw, &pre) != nil || pre.Schema != "gooo/path-diagnosis-preexecution/v1" || pre.Runner != revision ||
		pre.Cohort != hash(cohort) || pre.Prereg != hash(prereg) || pre.Model != diagnosisModelPin || pre.Weights != diagnosisWeightsPin || pre.Search != 2 || pre.Budget != 4 || !reflect.DeepEqual(pre.Probes, diagnosisProbes()) {
		return nil, errors.New("diagnosis preexecution pins differ")
	}
	rows, err := compoundRows()
	if err != nil {
		return nil, err
	}
	byID := map[string]compoundstudy.Case{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	raw, err := read(filepath.Join(output, "observations.jsonl"))
	if err != nil {
		return nil, err
	}
	files := map[string]string{"preexecution.json": hash(preRaw), "observations.jsonl": hash(raw)}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 128<<10)
	seen, summaries := map[string]bool{}, map[string]*diagnosisSummary{}
	observed, predictions, candidates, witnesses := 0, 0, 0, 0
	for scanner.Scan() {
		var capture diagnosisCapture
		if err = json.Unmarshal(scanner.Bytes(), &capture); err != nil {
			return nil, err
		}
		row, ok := byID[capture.ID]
		key := capture.Arm + "/" + capture.ID
		if !ok || seen[key] {
			return nil, errors.New("unknown or duplicate diagnosis capture")
		}
		seen[key] = true
		if err = validateDiagnosisCapture(row, capture); err != nil {
			return nil, err
		}
		observed++
		predictions += capture.Search.Selection.ModelCalls
		candidates += capture.Diagnosis.Observed
		witnesses += capture.Diagnosis.ProbeDistinguished
		key = capture.Arm + "/" + row.Contract
		if summaries[key] == nil {
			summaries[key] = &diagnosisSummary{Arm: capture.Arm, Contract: row.Contract}
		}
		s := summaries[key]
		s.Views++
		pass := capture.Search.SelectedTrainingPassed == len(row.Document.Cases)
		if pass {
			s.AllFinitePassed++
		}
		if capture.Diagnosis.CaseIndistinguishable > 1 {
			s.Ambiguous++
			if pass {
				s.AmbiguousPass++
			}
		}
		if capture.Diagnosis.ProbeDistinguished > 0 {
			s.Distinguishable++
		}
		if capture.Diagnosis.ProbeUnresolved > 1 {
			s.Unresolved++
		}
		s.Predictions += capture.Search.Selection.ModelCalls
		s.searchTimes = append(s.searchTimes, float64(capture.SearchNS))
		s.diagnosisTimes = append(s.diagnosisTimes, float64(capture.DiagnosisNS))
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	if observed != 144 || predictions != 144 || candidates != 576 || len(summaries) != 6 {
		return nil, errors.New("diagnosis planned denominators not observed")
	}
	var summary []diagnosisSummary
	for _, arm := range []string{"offline", "fp32"} {
		for _, contract := range []string{"complete", "sparse", "contradictory"} {
			s := summaries[arm+"/"+contract]
			if s == nil || s.Views != 24 {
				return nil, errors.New("diagnosis summary denominator differs")
			}
			s.SearchMedian, s.DiagnosisMedian = median(s.searchTimes), median(s.diagnosisTimes)
			summary = append(summary, *s)
		}
	}
	invocations, err := auditDiagnosisExecutions(output, rows, files)
	if err != nil {
		return nil, err
	}
	return map[string]any{"schema": "gooo/path-diagnosis-study/v1", "decision": "PASS", "runner_revision": revision,
		"sdk_searches": observed, "diagnoses": observed, "candidate_observations": candidates, "actual_initial_model_predictions": predictions,
		"diagnosis_model_predictions": 0, "distinguishing_candidate_witnesses": witnesses, "summaries": summary,
		"actual_generated_go_execution_jobs": 12, "actual_generated_function_invocations": invocations, "files_sha256": files,
		"scope": "Existing 72 bilingual/contract views over twelve intention groups. Each arm runs a two-candidate SDK search followed by a separate deterministic four-candidate diagnosis on 31 generic probes. Each distinct emitted Go program actually runs all probes. Independent integer-state oracle reconciles all candidates, search cases, selected best-so-far and witnesses. Audit performs typed-source reconstruction/oracle checks but zero new predictions/native/generated-Go executions; captured model-ranking origin relies on pinned clean runner/model/source CI, not an independently repeated prediction. Witnesses are divergent candidate values, not expected answers or permission to rewrite intent. Probe agreement is unresolved bounded evidence. No new training, GPU, Laya, native deployment, arbitrary language/all-input proof, host utilization or causal timing claim."}, nil
}

func auditDiagnosisExecutions(output string, rows []compoundstudy.Case, files map[string]string) (int, error) {
	wanted := map[string]string{}
	for _, template := range compoundstudy.Templates {
		plan, err := compoundstudy.Fixture(template, "en", 0)
		if err != nil {
			return 0, err
		}
		for mask := range uint16(4) {
			program, err := pathplan.Compile(plan, compoundstudy.Choices(plan, mask))
			if err != nil {
				return 0, err
			}
			wanted[hash([]byte(program.GoSource()))] = program.GoSource()
		}
	}
	names, err := os.ReadDir(filepath.Join(output, "executions"))
	if err != nil || len(names) != len(wanted) || len(wanted) != 12 {
		return 0, errors.New("diagnosis execution inventory differs")
	}
	invocations := 0
	for _, name := range names {
		relative := "executions/" + name.Name()
		raw, err := read(filepath.Join(output, relative))
		if err != nil {
			return 0, err
		}
		var value diagnosisExecution
		if json.Unmarshal(raw, &value) != nil || value.Schema != "gooo/path-diagnosis-go-execution/v1" || value.SHA+".json" != name.Name() ||
			hash([]byte(value.Source)) != value.SHA || value.Source != wanted[value.SHA] || !reflect.DeepEqual(value.Inputs, diagnosisProbes()) || len(value.Values) != 31 {
			return 0, errors.New("diagnosis actual Go execution binding differs")
		}
		plan, err := compoundstudy.Fixture(value.Template, "en", 0)
		if err != nil {
			return 0, err
		}
		program, err := pathplan.Compile(plan, compoundstudy.Choices(plan, value.Mask))
		if err != nil || program.GoSource() != value.Source {
			return 0, errors.New("diagnosis execution template/mask differs")
		}
		values, _, err := diagnosisOracleVector(value.Template, value.Mask, value.Inputs)
		if err != nil || !reflect.DeepEqual(values, value.Values) {
			return 0, errors.New("actual generated Go differs from diagnosis oracle")
		}
		invocations += len(value.Values)
		files[relative] = hash(raw)
	}
	return invocations, nil
}
