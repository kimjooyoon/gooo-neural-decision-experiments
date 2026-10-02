package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type nativePolicy struct {
	Name      string `json:"name"`
	Candidate string `json:"candidate"`
}
type executionCapture struct {
	Capture   string  `json:"capture_sha256"`
	Source    string  `json:"emitted_go_sha256"`
	Executed  bool    `json:"actual_compiled_go_execution"`
	Values    []int64 `json:"ordered_actual_values"`
	Cases     int     `json:"case_denominator"`
	Codegen   metrics `json:"codegen_process_metrics"`
	Execution metrics `json:"go_run_process_metrics"`
}

func nativePolicies(root, sdk string) (map[string]*model, []nativePolicy, error) {
	models, _, err := loadModels(root)
	if err != nil {
		return nil, nil, err
	}
	var s struct {
		Selected    string            `json:"selected_candidate"`
		Calibration map[string]counts `json:"calibration"`
		Pins        map[string]pin    `json:"model_pins"`
		Development bool              `json:"development_seen"`
	}
	if err = read(filepath.Join(sdk, "selection.json"), &s); err != nil {
		return nil, nil, err
	}
	var a struct {
		Status   string `json:"status"`
		Sessions int    `json:"sdk_sessions_audited"`
		Selected string `json:"selected_candidate"`
	}
	if err = read(filepath.Join(sdk, "independent-audit-strengthened.json"), &a); err != nil {
		return nil, nil, err
	}
	if s.Development || len(s.Calibration) != 12 || len(s.Pins) != 11 || a.Status != "PASS" || a.Sessions != 9216 || a.Selected != s.Selected || choose(s.Calibration, models) != s.Selected {
		return nil, nil, errors.New("frozen calibration selector and all-policy audit required")
	}
	for id, m := range models {
		if m != nil && s.Pins[id] != m.Pin {
			return nil, nil, errors.New("calibration model pin differs")
		}
	}
	return models, []nativePolicy{{"selected", s.Selected}, {"uniform-initial", "uniform-initial/fp32"}, {"set-initial", "set-initial/fp32"}, {"set-feedback", "set-feedback/fp32"}, {"offline", "offline"}}, nil
}
func modelPath(root, id string) (string, error) {
	if id == "offline" {
		return "", nil
	}
	if strings.HasPrefix(id, "reference-v1-") {
		return filepath.Abs(filepath.Join("models/joint-composition-v1", strings.TrimPrefix(id, "reference-v1-"), "models/fp32/model.json"))
	}
	p := strings.Split(id, "/")
	if len(p) != 2 {
		return "", errors.New("known model candidate required")
	}
	return filepath.Abs(filepath.Join(root, p[0], "models", p[1], "model.json"))
}
func originalDocument(v jointcohort.View) (document, []byte, error) {
	p, err := jointcompositionstudy.Fixture(v.Family, v.Config, v.Goal, v.Language)
	if err != nil {
		return document{}, nil, err
	}
	prepared, err := pathplan.Prepare(p)
	if err != nil {
		return document{}, nil, err
	}
	body, err := json.Marshal(prepared.Fallback().GoooBody())
	if err != nil {
		return document{}, nil, err
	}
	source := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	if jointcohort.SHA(source) != v.SourceSHA {
		return document{}, nil, errors.New("original native source differs")
	}
	return document{"gooo/body-codegen-typed-path-plan/v1", p, v.Cases, 4}, source, nil
}
func inspectNative(raw []byte, v jointcohort.View, doc document, source []byte, m *model, wall int64) (nativeResult, jointfeedback.Capture, error) {
	var n nativeResult
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return n, jointfeedback.Capture{}, err
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		return n, jointfeedback.Capture{}, err
	}
	r, p := n.Report, n.Report.Paths
	docRaw, err := json.Marshal(doc)
	if err != nil || r.Decision != "PASS" || r.Compiler != nativeDeployed || !r.Types || !r.Replay || r.Writes != 0 || !p.Bound || !p.Binding.Equivalent || p.Original != "sha256:"+jointcohort.SHA(source) || p.Document != "sha256:"+jointcohort.SHA(docRaw) || p.Completeness != 100 || len(p.Cases) != 16 {
		return n, jointfeedback.Capture{}, errors.New("native source/document/final contract binding differs")
	}
	if m == nil {
		if p.Context != nil || p.Search.Selection.ModelCalls != 0 {
			return n, jointfeedback.Capture{}, errors.New("disconnected native performed model work")
		}
	} else {
		schema, feature := "gooo/compiler-typed-path-context/v3", decision.SemanticContextIntentFeatureVersion
		if m.Joint != nil {
			schema, feature = "gooo/compiler-joint-path-context/v1", m.Joint.FeatureVersion()
		}
		if p.Context == nil || p.Context.Schema != schema || p.Context.Status != "ENCODED" || p.Context.Feature != feature || p.Context.Metadata != m.Pin.Metadata || len(p.Context.Inputs) != 2 || len(p.Search.Selection.Receipts) != 2 {
			return n, jointfeedback.Capture{}, errors.New("native original model context differs")
		}
		parts, err := jointParts(v)
		if err != nil {
			return n, jointfeedback.Capture{}, err
		}
		for i, input := range p.Context.Inputs {
			if input.ID != doc.Plan.Decisions[i].ID || input.SHA != "sha256:"+jointcohort.SHA([]byte(parts[i])) || p.Search.Selection.Receipts[i].IntentSHA256 != jointcohort.SHA([]byte(parts[i])) {
				return n, jointfeedback.Capture{}, errors.New("native full original intent differs")
			}
		}
	}
	for i, c := range p.Cases {
		if c.Input != v.Cases[i].Input || c.Expected != v.Cases[i].Expected || c.Actual != c.Expected || !c.Passed {
			return n, jointfeedback.Capture{}, errors.New("native final ordered value differs")
		}
	}
	c := jointfeedback.Capture{Schema: "gooo/own-joint-feedback-student-capture/v2", ViewID: v.ID, SourceSHA: v.SourceSHA, JointSHA: jointcohort.SHA([]byte(v.JointInput)), Rotation: -1, WallNS: wall, Search: p.Search, Progress: p.Progress, Feedback: p.Feedback}
	return n, c, verifyObservation(v, c, m)
}
func validMetrics(m metrics) bool {
	return m.Wall > 0 && m.User >= 0 && m.System >= 0 && m.RSS > 0 && m.CPU == 100*float64(m.User+m.System)/float64(m.Wall)
}
func auditExecution(x executionCapture, raw []byte, n nativeResult, v jointcohort.View) error {
	if x.Capture != jointcohort.SHA(raw) || x.Source != jointcohort.SHA([]byte(n.Source)) || !x.Executed || x.Cases != 16 || len(x.Values) != 16 || !validMetrics(x.Codegen) || !validMetrics(x.Execution) {
		return errors.New("actual Go execution or process binding differs")
	}
	for i, a := range x.Values {
		if a != v.Cases[i].Expected {
			return errors.New("actual compiled Go independent oracle differs")
		}
	}
	return nil
}
func nativeStudy(dataset, root, sdk, out, revision, binary, goBinary, audit string) error {
	if audit == "" {
		if err := source(revision); err != nil {
			return err
		}
		if err := nativePins(binary, goBinary); err != nil {
			return err
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			return errors.New("fresh native directory required")
		}
		if err := os.MkdirAll(out, 0755); err != nil {
			return err
		}
	} else if _, err := os.Stat(audit); !os.IsNotExist(err) {
		return errors.New("fresh native audit required")
	}
	views, err := jointcohort.Load(dataset)
	if err != nil {
		return err
	}
	models, policies, err := nativePolicies(root, sdk)
	if err != nil {
		return err
	}
	priors, err := sdkObservations(sdk, policies, views)
	if err != nil {
		return err
	}
	selection, err := os.ReadFile(filepath.Join(sdk, "selection.json"))
	if err != nil {
		return err
	}
	pins := map[string]pin{}
	for _, p := range policies {
		if m := models[p.Candidate]; m != nil {
			pins[p.Candidate] = m.Pin
		}
	}
	if audit == "" {
		bin, err := os.ReadFile(binary)
		if err != nil {
			return err
		}
		gobin, err := os.ReadFile(goBinary)
		if err != nil {
			return err
		}
		if err = save(filepath.Join(out, "preexecution.json"), map[string]any{"schema": "gooo/own-joint-feedback-native-preexecution/v2", "source_revision": revision, "native_revision": nativeDeployed, "sdk": "v0.2.12-experimental", "protocol_sha256": jointfeedback.ProtocolSHA, "dataset_sha256": jointcohort.DatasetSHA, "selection_sha256": jointcohort.SHA(selection), "model_pins": pins, "policies": policies, "planned_native_generations": 240, "planned_actual_go_executions": 240, "planned_ordered_invocations": 3840, "native_binary_sha256": jointcohort.SHA(bin), "go_binary_sha256": jointcohort.SHA(gobin), "ci_hint_supplied": false}); err != nil {
			return err
		}
	} else {
		if err = verifyNativePreexecution(out, selection, pins, policies); err != nil {
			return err
		}
	}
	workspace := ""
	if audit == "" {
		workspace, err = os.MkdirTemp("", "gooo-feedback-native-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(workspace)
	}
	totals := map[string]counts{}
	pairs := map[string]map[string]map[string]uint16{}
	measurements := map[string][]metrics{}
	calls, predictions := 0, 0
	for _, v := range views {
		if v.Config != 40 {
			continue
		}
		doc, src, err := originalDocument(v)
		if err != nil {
			return err
		}
		for _, p := range policies {
			name := fmt.Sprintf("%s-goal%d-%s-%s", v.Family, v.Goal, v.Language, p.Name)
			var raw []byte
			var x executionCapture
			if audit == "" {
				if err = save(filepath.Join(workspace, "plan.json"), doc); err != nil {
					return err
				}
				if err = os.WriteFile(filepath.Join(workspace, "source.gooo"), src, 0600); err != nil {
					return err
				}
				args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ChoosePath", "source.gooo"}
				if p.Candidate != "offline" {
					path, err := modelPath(root, p.Candidate)
					if err != nil {
						return err
					}
					args = append(args, "--path-model", path, "--path-feedback-rounds", "3", "--path-feedback-unfixed")
				}
				var runErr error
				raw, x.Codegen, runErr = child(workspace, binary, args...)
				if err = os.WriteFile(filepath.Join(out, name+".json"), raw, 0644); err != nil {
					return err
				}
				if runErr != nil {
					return runErr
				}
			} else {
				raw, err = os.ReadFile(filepath.Join(out, name+".json"))
				if err != nil {
					return err
				}
				if err = read(filepath.Join(out, name+"-execution.json"), &x); err != nil {
					return err
				}
			}
			n, c, err := inspectNative(raw, v, doc, src, models[p.Candidate], x.Codegen.Wall)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			prior, ok := priors[p.Candidate][v.ID]
			if !ok {
				return errors.New("native frozen SDK prior missing")
			}
			if !reflect.DeepEqual(c.Search.Attempts, prior.Search.Attempts) || !reflect.DeepEqual(c.Search.InitialProposals, prior.Search.InitialProposals) || c.Search.Selection.ModelCalls != prior.Search.Selection.ModelCalls {
				return errors.New("actual native and frozen SDK paths differ")
			}
			if audit == "" {
				x.Values, x.Execution, err = executeGo(goBinary, n.Source, v.Cases)
				if err != nil {
					return err
				}
				x.Capture, x.Source, x.Executed, x.Cases = jointcohort.SHA(raw), jointcohort.SHA([]byte(n.Source)), true, 16
				if err = save(filepath.Join(out, name+"-execution.json"), x); err != nil {
					return err
				}
			}
			if err = auditExecution(x, raw, n, v); err != nil {
				return err
			}
			calls++
			predictions += c.Search.Selection.ModelCalls
			count := totals[p.Name]
			if err = count.add(v, c); err != nil {
				return err
			}
			totals[p.Name] = count
			measurements[p.Name] = append(measurements[p.Name], x.Codegen)
			if pairs[p.Name] == nil {
				pairs[p.Name] = map[string]map[string]uint16{}
			}
			if pairs[p.Name][v.Group] == nil {
				pairs[p.Name][v.Group] = map[string]uint16{}
			}
			pairs[p.Name][v.Group][v.Language] = c.Search.Attempts[0].Mask
			if audit == "" && calls%40 == 0 {
				fmt.Printf("native actual generations/executions=%d predictions=%d\n", calls, predictions)
			}
		}
	}
	if calls != 240 {
		return errors.New("native fixed function view count differs")
	}
	for policy, groups := range pairs {
		c := totals[policy]
		for _, langs := range groups {
			if len(langs) != 2 {
				return errors.New("native bilingual pair missing")
			}
			c.Pairs++
			if langs["en"] != langs["ko"] {
				c.Disagreement++
			}
		}
		totals[policy] = c
	}
	if audit != "" {
		var captured struct {
			Status      string            `json:"status"`
			Calls       int               `json:"actual_native_generations"`
			Predictions int               `json:"actual_model_predictions"`
			Totals      map[string]counts `json:"totals"`
		}
		if err = read(filepath.Join(out, "report.json"), &captured); err != nil {
			return err
		}
		if captured.Status != "PASS" || captured.Calls != calls || captured.Predictions != predictions || len(captured.Totals) != 5 {
			return errors.New("native totals differ")
		}
		for id, c := range totals {
			if !equalCounts(c, captured.Totals[id]) {
				return errors.New("native reconstructed cell differs")
			}
		}
		return save(audit, map[string]any{"schema": "gooo/own-joint-feedback-native-audit/v2", "status": "PASS", "native_generations_audited": calls, "actual_go_executions_audited": calls, "ordered_invocations_audited": calls * 16, "recorded_model_predictions": predictions, "new_native_calls": 0, "new_model_predictions": 0, "totals": totals})
	}
	return save(filepath.Join(out, "report.json"), map[string]any{"schema": "gooo/own-joint-feedback-native-report/v2", "status": "PASS", "source_revision": revision, "native_revision": nativeDeployed, "actual_native_generations": calls, "actual_compiled_go_executions": calls, "actual_ordered_function_invocations": calls * 16, "actual_model_predictions": predictions, "totals": totals, "codegen_process_metrics": measurements, "ci_hint_supplied": false, "new_optimizer_updates": 0, "scope": "Five policies on 48 config-40 function views. All emitted Go independently compiled/executed against 16 ordered oracle cases. Codegen and go-run process measurements are distinct. Model probabilities rank declared legal structure; existing types and ordinary arithmetic determine finite acceptance. No implicit default promotion or causal host-utilization claim."})
}
