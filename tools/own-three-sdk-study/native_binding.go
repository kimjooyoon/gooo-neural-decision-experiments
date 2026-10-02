package main

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

func nativeOriginal(v threecohort.View) (nativeDocument, []byte, error) {
	plan, err := threecompositionstudy.Fixture(v.Family, v.Config, v.Goal, v.Language)
	if err != nil {
		return nativeDocument{}, nil, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return nativeDocument{}, nil, err
	}
	body, err := json.Marshal(prepared.Fallback().GoooBody())
	if err != nil {
		return nativeDocument{}, nil, err
	}
	source := []byte("package threecomposition\nnamespace threecomposition\nentity Integer id \"threecomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	if threecohort.SHA(source) != v.SourceSHA {
		return nativeDocument{}, nil, errors.New("native original authored source differs")
	}
	return nativeDocument{"gooo/body-codegen-typed-path-plan/v1", plan, v.Cases, 8}, source, nil
}
func inspectNative(raw []byte, v threecohort.View, doc nativeDocument, source []byte, m *model, wall int64) (nativeResult, threefeedback.Capture, threecohort.View, error) {
	var n nativeResult
	var c threefeedback.Capture
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return n, c, v, err
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		return n, c, v, err
	}
	r, p := n.Report, n.Report.Paths
	docRaw, _ := json.Marshal(doc)
	tests, _ := json.Marshal(v.Cases)
	if r.Decision != "PASS" || r.Compiler != nativeRevision || !r.Types || !r.Replay || r.Writes != 0 || !p.Bound || !p.Binding.Equivalent || len(p.Binding.Semantic) != 71 || !strings.HasPrefix(p.Binding.Semantic, "sha256:") || strings.Trim(p.Binding.Semantic[7:], "0123456789abcdef") != "" || p.Original != "sha256:"+threecohort.SHA(source) || p.Document != "sha256:"+threecohort.SHA(docRaw) || p.Tests != "sha256:"+threecohort.SHA(tests) || p.Completeness != 100 || len(p.Cases) != 16 || len(n.Source) == 0 {
		return n, c, v, errors.New("native source/document/test/type/final contract binding differs")
	}
	for _, x := range []float64{p.Timing.Plan, p.Timing.Binding, p.Timing.Load, p.Timing.Context, p.Timing.Search, p.Timing.Emission, p.Timing.Total} {
		if x < 0 || math.IsNaN(x) || math.IsInf(x, 0) {
			return n, c, v, errors.New("finite nonnegative native phase timing required")
		}
	}
	if p.Timing.Total <= 0 {
		return n, c, v, errors.New("actual native phase timing missing")
	}
	original, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		return n, c, v, err
	}
	auditView := v
	if m == nil {
		if p.Context != nil || p.Search.Selection.ModelCalls != 0 || p.Unfixed || p.Timing.Load != 0 || p.Timing.Context != 0 {
			return n, c, v, errors.New("disconnected native performed model/context work")
		}
		auditView.Prepared, auditView.Plan = original, doc.Plan
	} else {
		if m.Three == nil || p.Context == nil || p.Context.Schema != "gooo/compiler-three-choice-path-context/v1" || p.Context.Status != "ENCODED" || p.Context.Feature != jointdecision.ThreeFeatureVersion || p.Context.Metadata != m.Pin.Metadata || p.Context.Semantic != p.Binding.Semantic || p.Context.OriginalPlan != original.PlanSHA256() || p.Context.RankedPlan != v.Prepared.PlanSHA256() || len(p.Context.Inputs) != 3 || !p.Unfixed {
			return n, c, v, errors.New("native actual three-part model context differs")
		}
		// The original caller text is intentionally not the projected model ABI.
		// ThreeInput returns every length-framed original part even when its
		// canonical-ABI validation declines. Verify those complete bytes, while
		// validating the separate projected inputs and model receipts below.
		declared, _ := original.ThreeInput()
		if p.Context.Declared == nil || p.Context.Declared.Text != declared || p.Context.Declared.SHA != "sha256:"+threecohort.SHA([]byte(declared)) || p.Context.Declared.Bytes != len(declared) || p.Context.Declared.Decisions != 3 {
			return n, c, v, errors.New("full original declared inputs not preserved")
		}
		for i, input := range p.Context.Inputs {
			if input.ID != doc.Plan.Decisions[i].ID || input.SHA != "sha256:"+threecohort.SHA([]byte(v.Parts[i])) || input.Bytes != len(v.Parts[i]) || !input.Replaced {
				return n, c, v, errors.New("native complete ordered projected input differs")
			}
		}
	}
	for i, result := range p.Cases {
		if result.Input != v.Cases[i].Input || result.Expected != v.Cases[i].Expected || result.Actual != result.Expected || !result.Passed {
			return n, c, v, errors.New("actual native final ordered value differs")
		}
	}
	c = threefeedback.Capture{Schema: "gooo/own-three-choice-sdk-capture/v1", ViewID: v.ID, SourceSHA: v.SourceSHA, InputSHA: threecohort.SHA([]byte(v.Text)), SeedIndex: -1, WallNS: wall, Search: p.Search, Progress: p.Progress, Feedback: p.Feedback, Derived: []threefeedback.Projection{}, TeacherInputs: []threefeedback.TeacherInput{}}
	if err = verify(auditView, c, m); err != nil {
		return n, c, auditView, err
	}
	return n, c, auditView, nil
}
func compareSDK(c threefeedback.Capture, prior nativePrior) error {
	if !reflect.DeepEqual(c.Search.Attempts, prior.Attempts) || !reflect.DeepEqual(c.Search.InitialProposals, prior.Proposals) || c.Search.Selection.ModelCalls != prior.Calls {
		return errors.New("actual native typed paths differ from frozen SDK observations")
	}
	return nil
}
func auditNativeExecution(x nativeExecution, raw []byte, n nativeResult, v threecohort.View, policy nativePolicy) error {
	if x.Schema != "gooo/own-three-native-execution/v1" || x.View != v.ID || x.Policy != policy || x.Capture != threecohort.SHA(raw) || x.Source != threecohort.SHA([]byte(n.Source)) || !x.NativeCalled || !x.GoCalled || !x.Executed || x.Cases != 16 || len(x.Values) != 16 || x.Error != "" || x.CodegenStderr != "" || x.GoStderr != "" || !validChild(x.Codegen) || !validChild(x.Execution) {
		return errors.New("actual compiled Go/process/source binding differs")
	}
	var actual []int64
	if err := json.Unmarshal([]byte(x.GoStdout), &actual); err != nil || !reflect.DeepEqual(actual, x.Values) {
		return errors.New("actual compiled stdout differs from recorded values")
	}
	for i, a := range actual {
		if a != v.Cases[i].Expected {
			return errors.New("compiled Go independent ordered oracle differs")
		}
	}
	return nil
}
