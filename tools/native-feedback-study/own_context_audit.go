package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func bareOwnContextSHA(value string) string { return strings.TrimPrefix(value, "sha256:") }

func auditOwnNativeContext(output, revision, native string) (map[string]any, error) {
	rows, err := ownContextRows()
	if err != nil {
		return nil, err
	}
	raw, err := read(filepath.Join(output, "preexecution.json"))
	if err != nil {
		return nil, err
	}
	var pre struct {
		Runner   string            `json:"runner_revision"`
		Native   string            `json:"native_revision"`
		Protocol string            `json:"protocol_sha256"`
		Inputs   map[string]string `json:"input_sha256"`
		Models   map[string]string `json:"model_pins"`
	}
	if err = json.Unmarshal(raw, &pre); err != nil {
		return nil, err
	}
	protocol, err := read(ownNativeContextProtocol)
	if err != nil || pre.Runner != revision || pre.Native != native || pre.Protocol != hash(protocol) ||
		!reflect.DeepEqual(pre.Models, ownNativeContextPins) {
		return nil, errors.New("preexecution source/protocol/model pins differ")
	}
	nativeCalls, predictions, finite, passed, declines, attempts, extras, wrongSparse := 0, 0, 0, 0, 0, 0, 0, 0
	candidateCases := 0
	var observations []map[string]any
	pairedInputs := map[string][]ownContextInput{}
	for _, row := range rows {
		d, _ := json.Marshal(row.Document)
		source, err := read(row.Source)
		if err != nil || pre.Inputs[row.ID+"-document"] != hash(d) || pre.Inputs[row.ID+"-source"] != hash(source) {
			return nil, errors.New("frozen input changed")
		}
		prepared, err := pathplan.Prepare(row.Document.Plan)
		if err != nil {
			return nil, err
		}
		for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary", "offline"} {
			id := row.ID + "-" + arm
			capture, err := read(filepath.Join(output, id+".json"))
			if err != nil {
				return nil, err
			}
			var value nativeResult
			if err = json.Unmarshal(capture, &value); err != nil {
				return nil, err
			}
			if err = inspectOwnNativeContext(value, row, arm, native, prepared); err != nil {
				return nil, fmt.Errorf("%s: %w", id, err)
			}
			p := value.Report.Paths
			predictionNS := int64(0)
			for _, r := range p.Search.Selection.Receipts {
				predictionNS += r.PredictNS
			}
			if err = auditOwnContextAttempts(prepared, row.Document, p.Search); err != nil {
				return nil, err
			}
			eRaw, err := read(filepath.Join(output, id+"-execution.json"))
			if err != nil {
				return nil, err
			}
			var execution ownContextExecution
			if err = json.Unmarshal(eRaw, &execution); err != nil || execution.ID != id || execution.CaptureSHA != hash(capture) ||
				execution.SourceSHA != hash([]byte(value.Source)) || len(execution.Values) != len(p.Cases) {
				return nil, errors.New("compiled Go execution binding differs")
			}
			for i, c := range p.Cases {
				if execution.Values[i] != c.Actual {
					return nil, errors.New("compiled Go/native actuals differ")
				}
			}
			mRaw, err := read(filepath.Join(output, id+"-metrics.json"))
			if err != nil {
				return nil, err
			}
			var m metrics
			if err = json.Unmarshal(mRaw, &m); err != nil || m.Wall <= 0 || m.RSS <= 0 ||
				m.CPU != 100*float64(m.User+m.System)/float64(m.Wall) {
				return nil, errors.New("resource observation differs")
			}
			if p.Context != nil && !row.Overflow && strings.HasSuffix(row.ID, "-sparse") {
				pairedInputs[row.ID[:2]+"-"+arm] = p.Context.Inputs
			}
			if p.Context != nil && strings.HasSuffix(row.ID, "-full") && !reflect.DeepEqual(pairedInputs[row.ID[:2]+"-"+arm], p.Context.Inputs) {
				return nil, errors.New("authored cases changed initial compiler input")
			}
			if strings.HasSuffix(row.ID, "-sparse") && p.Search.Selection.Choices["operands"] == "layout_reverse" {
				wrongSparse++
			}
			if p.Context != nil && p.Context.Status == "DECLINED_TO_DETERMINISTIC" {
				declines++
			}
			nativeCalls++
			predictions += p.Search.Selection.ModelCalls
			finite += len(p.Cases)
			passed += p.Search.SelectedTrainingPassed
			attempts += len(p.Search.Attempts)
			extras += len(p.Search.Attempts) - 1
			for _, a := range p.Search.Attempts {
				candidateCases += len(a.Results)
			}
			observations = append(observations, map[string]any{"id": id, "capture_sha256": hash(capture), "calls": p.Search.Selection.ModelCalls,
				"finite_passed": p.Search.SelectedTrainingPassed, "finite_total": len(p.Cases), "finite_percent": p.Completeness,
				"attempts": len(p.Search.Attempts), "prediction_ns": predictionNS, "context_prepare_ms": p.Timing.Context,
				"native_ms": p.Timing.Total, "search_ms": p.Timing.Search, "model_load_ms": p.Timing.Load, "process_metrics": m,
				"choices": p.Search.Selection.Choices})
		}
	}
	if nativeCalls != 32 || predictions != 57 || finite != 120 || declines != 3 {
		return nil, errors.New("preregistered observation denominators differ")
	}
	files, err := ownContextInventory(output, rows)
	if err != nil {
		return nil, err
	}
	return map[string]any{"schema": "gooo/own-model-native-context-audit/v1", "decision": "PASS", "runner_revision": revision,
		"files_sha256":    files,
		"native_revision": native, "actual_native_calls": nativeCalls, "actual_initial_predictions": predictions,
		"representation_declines": declines, "selected_finite_cases": finite, "selected_finite_passed": passed,
		"actual_generated_go_processes": nativeCalls, "actual_generated_function_invocations": finite,
		"candidate_attempts": attempts, "extra_candidate_attempts": extras, "actual_attempt_case_evaluations": candidateCases,
		"sparse_wrong_intended_choices": wrongSparse, "observations": observations, "new_independent_intentions": 0,
		"scope": "Three reused intentions across variants/languages/contracts and one synthetic bounds probe. Actual source-bound compiler calls and separately compiled emitted Go. Initial input ignores authored case changes. Finite scores are not all-input semantics, language accuracy, causal speedup or host CPU utilization. No training, upstream Laya, arbitrary synthesis or default promotion."}, nil
}

func inspectOwnNativeContext(v nativeResult, row ownContextRow, arm, native string, prepared *pathplan.PreparedPlan) error {
	r, p := v.Report, v.Report.Paths
	if r.Decision != "PASS" || r.Compiler != native || !r.Types || !r.Replay || r.Writes != 0 || !p.Bound || !p.Binding.Equivalent ||
		p.Search.Selection.ExternalCalls != 0 || !p.Search.Selection.ExternalCallsKnown || len(p.Feedback) != 0 ||
		p.Search.TrainingTotal != len(row.Document.Cases) || len(p.Cases) != len(row.Document.Cases) ||
		bareOwnContextSHA(p.OriginalSHA) != hashMustRead(row.Source) {
		return errors.New("source-bound native invariants differ")
	}
	calls := len(row.Document.Plan.Decisions)
	if arm == "offline" || row.Overflow {
		calls = 0
	}
	if p.Search.Selection.ModelCalls != calls {
		return errors.New("prediction accounting differs")
	}
	if arm == "offline" {
		if p.Context != nil || p.Search.Selection.PlanSHA256 != prepared.PlanSHA256() {
			return errors.New("offline context changed")
		}
	} else {
		c := p.Context
		if c == nil || c.Schema != "gooo/compiler-typed-path-context/v1" || c.Feature != "split_context_intent_ngrams_v2" ||
			c.Metadata != ownNativeContextPins[arm] || c.Activity != r.ActivityID || c.Source != p.Binding.Source ||
			c.Original != prepared.PlanSHA256() {
			return errors.New("compiler fact context identity differs")
		}
		if row.Overflow {
			if c.Status != "DECLINED_TO_DETERMINISTIC" || c.Reason != "COMBINED_CONTEXT_INTENT_EXCEEDS_MODEL_BOUND" ||
				!c.FeedbackSkipped || c.Ranked != "" || len(c.Inputs) != 1 || c.Inputs[0].Bytes <= 512 || c.Inputs[0].SHA != "" {
				return errors.New("decline did not retain deterministic continuation")
			}
		} else {
			if c.Status != "ENCODED" || c.Ranked == "" || c.Ranked != p.Search.Selection.PlanSHA256 || len(c.Inputs) != calls ||
				p.Search.Selection.MetadataSHA256 != c.Metadata {
				return errors.New("ranked context differs")
			}
		}
		for i, input := range c.Inputs {
			choice := row.Document.Plan.Decisions[i]
			intent := choice.Intent
			if index := strings.LastIndex(intent, "intent: "); index >= 0 {
				intent = intent[index+len("intent: "):]
			}
			if input.ID != choice.ID || bareOwnContextSHA(input.Original) != hash([]byte(choice.Intent)) ||
				bareOwnContextSHA(input.Natural) != hash([]byte(intent)) {
				return errors.New("natural intention was lost")
			}
			if !row.Overflow && (input.Bytes < len(intent) || input.Bytes > 512 ||
				bareOwnContextSHA(input.SHA) != bareOwnContextSHA(p.Search.Selection.Receipts[i].IntentSHA256)) {
				return errors.New("prediction/context input binding differs")
			}
		}
	}
	program, err := prepared.Compile(p.Search.Selection.Choices)
	if err != nil {
		return err
	}
	passed := 0
	for i, test := range row.Document.Cases {
		actual, err := program.Evaluate(test.Input)
		c := p.Cases[i]
		if err != nil || c.Input != test.Input || c.Expected != test.Expected || actual.Int != c.Actual || c.Passed != (c.Actual == c.Expected) {
			return errors.New("selected native finite observation differs")
		}
		if row.Activity == "Probe" && c.Actual != nativeDiagnosisOracle(boolMask(p.Search.Selection.Choices["operands"] == "layout_reverse"), c.Input) {
			return errors.New("independent subtraction arithmetic differs")
		}
		if c.Passed {
			passed++
		}
	}
	if passed != p.Search.SelectedTrainingPassed || p.Completeness != 100*float64(passed)/float64(len(p.Cases)) ||
		(row.Overflow && (passed != 6 || p.Search.Status != "PARTIAL")) {
		return errors.New("partial denominator differs")
	}
	return nil
}

func boolMask(reverse bool) uint16 {
	if reverse {
		return 1
	}
	return 0
}
func hashMustRead(name string) string {
	raw, err := read(name)
	if err != nil {
		return ""
	}
	return hash(raw)
}

func auditOwnContextAttempts(prepared *pathplan.PreparedPlan, d document, search pathplan.SearchResult) error {
	seen := map[uint16]bool{}
	evaluated, rejected := 0, 0
	for _, a := range search.Attempts {
		if seen[a.Mask] {
			return errors.New("candidate mask repeated")
		}
		seen[a.Mask] = true
		mask := uint16(0)
		for i, choice := range d.Plan.Decisions {
			label := a.Choices[choice.ID]
			if label == choice.Options[1].Label {
				mask |= 1 << i
			} else if label != choice.Options[0].Label {
				return errors.New("undeclared candidate choice")
			}
		}
		if mask != a.Mask {
			return errors.New("candidate mask/choices differ")
		}
		program, err := prepared.Compile(a.Choices)
		if err != nil {
			if a.Status != "TYPE_REJECTED" {
				return errors.New("candidate type rejection differs")
			}
			rejected++
			continue
		}
		if len(a.Results) != len(d.Cases) || a.Total != len(d.Cases) {
			return errors.New("candidate case count differs")
		}
		passed := 0
		for i, test := range d.Cases {
			value, err := program.Evaluate(test.Input)
			result := a.Results[i]
			if err != nil || result.Input != test.Input || result.Expected != test.Expected || result.Actual != value.Int ||
				result.Passed != (value.Int == test.Expected) {
				return errors.New("candidate actual/pass flag differs")
			}
			if result.Passed {
				passed++
			}
		}
		if passed != a.Passed || a.GoooSHA != hash([]byte(program.GoooSource())) {
			return errors.New("candidate body/count differs")
		}
		evaluated++
	}
	if evaluated != search.Evaluated || rejected != search.TypeRejected {
		return errors.New("candidate aggregates differ")
	}
	return nil
}
