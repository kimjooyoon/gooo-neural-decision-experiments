package main

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

// Filled only after independently verified main promotion; empty blocks execution.
const nativeDeployed = "363a3d8aa365c35dd634c241248b444de0050973"

type metrics struct {
	Wall   int64   `json:"wall_ns"`
	User   int64   `json:"user_cpu_ns"`
	System int64   `json:"system_cpu_ns"`
	RSS    int64   `json:"child_max_rss_bytes"`
	CPU    float64 `json:"process_cpu_percent_of_one_core_over_wall"`
}
type bounded struct{ bytes.Buffer }

func (b *bounded) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 1<<20 {
		return 0, errors.New("child byte bound exceeded")
	}
	return b.Buffer.Write(raw)
}
func child(dir, binary string, args ...string) ([]byte, metrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	cmd.Env = append(os.Environ(), "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	configure(cmd)
	var stdout, stderr bounded
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	err := cmd.Run()
	m := metrics{Wall: time.Since(start).Nanoseconds()}
	if cmd.ProcessState != nil {
		m.User, m.System = cmd.ProcessState.UserTime().Nanoseconds(), cmd.ProcessState.SystemTime().Nanoseconds()
		m.RSS = maxRSS(cmd.ProcessState)
		m.CPU = 100 * float64(m.User+m.System) / float64(m.Wall)
	}
	if err != nil || stderr.Len() != 0 {
		return stdout.Bytes(), m, errors.New("bounded child failed; stdout retained")
	}
	return stdout.Bytes(), m, nil
}

type document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Max    int                 `json:"max_attempts"`
}
type nativeResult struct {
	Source string `json:"source"`
	Report struct {
		Decision string `json:"decision"`
		Compiler string `json:"compiler_source_sha"`
		Types    bool   `json:"typecheck_passed"`
		Replay   bool   `json:"deterministic_replay"`
		Writes   int    `json:"repository_writes"`
		Paths    struct {
			Original string `json:"original_source_sha256"`
			Document string `json:"document_sha256"`
			Bound    bool   `json:"source_base_matched"`
			Binding  struct {
				Equivalent bool `json:"equivalent"`
			} `json:"source_binding"`
			Search       pathplan.SearchResult      `json:"search"`
			Progress     []pathplan.SessionProgress `json:"session_progress"`
			Feedback     []pathplan.FeedbackReceipt `json:"feedback_judgments"`
			Completeness float64                    `json:"finite_functional_completeness_percent"`
			Cases        []pathplan.TestResult      `json:"native_case_results"`
			Context      *struct {
				Schema   string `json:"schema"`
				Status   string `json:"status"`
				Feature  string `json:"feature_version"`
				Metadata string `json:"model_metadata_sha256"`
				Inputs   []struct {
					ID  string `json:"decision_id"`
					SHA string `json:"input_sha256"`
				} `json:"inputs"`
			} `json:"model_context"`
		} `json:"body_paths"`
	} `json:"report"`
}

func originalDocument(v view) (document, []byte, error) {
	r := v.Rows[0]
	plan, err := jointcompositionstudy.Fixture(r.Family, r.Config, r.Desired, r.Language)
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
	source := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	cases, err := jointcompositionstudy.Cases(r.Family, r.Config, r.Desired)
	return document{"gooo/body-codegen-typed-path-plan/v1", plan, cases, 4}, source, err
}

func nativePins(binary, goBinary string) error {
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("exact native Go 1.27.1 required")
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
	if settings["vcs.revision"] != nativeDeployed || settings["vcs.modified"] != "false" || sdk != "v0.2.12-experimental" {
		return errors.New("clean native main and SDK.12 required")
	}
	info, err = buildinfo.ReadFile(goBinary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("exact emitted-Go execution compiler required")
	}
	return nil
}

func inspectNative(raw []byte, v view, doc document, source []byte, model *loadedModel) (nativeResult, observation, error) {
	var n nativeResult
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return n, observation{}, err
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		return n, observation{}, err
	}
	r, p := n.Report, n.Report.Paths
	docRaw, _ := json.Marshal(doc)
	if r.Decision != "PASS" || r.Compiler != nativeDeployed || !r.Types || !r.Replay || r.Writes != 0 || !p.Bound || !p.Binding.Equivalent ||
		p.Original != "sha256:"+hash(source) || p.Document != "sha256:"+hash(docRaw) || p.Completeness != 100 || len(p.Cases) != 16 {
		return n, observation{}, errors.New("native original source/document/contract differs")
	}
	if model == nil {
		if p.Context != nil || p.Search.Selection.ModelCalls != 0 {
			return n, observation{}, errors.New("offline invoked model/context")
		}
	} else {
		contextSchema := "gooo/compiler-typed-path-context/v2"
		if model.FeatureVersion() == decision.SemanticContextIntentFeatureVersion {
			contextSchema = "gooo/compiler-typed-path-context/v3"
		}
		if model.joint != nil {
			contextSchema = "gooo/compiler-joint-path-context/v1"
		}
		if p.Context == nil || p.Context.Schema != contextSchema || p.Context.Status != "ENCODED" || p.Context.Feature != model.RuntimeFeatureVersion() ||
			p.Context.Metadata != model.MetadataSHA256() || len(p.Context.Inputs) != 2 || len(p.Search.Selection.Receipts) != 2 {
			return n, observation{}, errors.New("native compiler context differs")
		}
		for i, input := range p.Context.Inputs {
			if input.ID != v.Rows[i].Input.ID || input.SHA != v.Rows[i].Input.SHA ||
				p.Search.Selection.Receipts[i].IntentSHA256 != strings.TrimPrefix(input.SHA, "sha256:") {
				return n, observation{}, errors.New("source-bound native model input differs")
			}
		}
		if p.Search.Selection.MetadataSHA256 != model.MetadataSHA256() || p.Search.Selection.WeightsSHA256 != model.WeightsSHA256() {
			return n, observation{}, errors.New("native model weights differ")
		}
	}
	for i, c := range p.Cases {
		if c.Input != doc.Cases[i].Input || c.Expected != doc.Cases[i].Expected || c.Actual != doc.Cases[i].Expected || !c.Passed {
			return n, observation{}, errors.New("native final actual differs")
		}
	}
	o := observation{ID: v.ID, Inputs: [2]string{v.Rows[0].Input.SHA, v.Rows[1].Input.SHA}, Search: p.Search, Progress: p.Progress, Feedback: p.Feedback}
	kind := "offline"
	if model != nil {
		kind = model.kind
	}
	if kind == "joint" {
		r := p.Search.Selection.Joint
		if r == nil || r.Schema != jointdecision.Schema || r.Input != v.Rows[0].JointText || r.InputSHA != v.Rows[0].JointSHA || r.Calls != 1 {
			return n, o, errors.New("actual native joint input differs")
		}
	}
	return n, o, validateObservation(v, o, kind)
}

func executeGo(goBinary, source string, cases []pathplan.TestCase) ([]int64, error) {
	dir, err := os.MkdirTemp("", "gooo-fresh-execute-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err = os.Mkdir(filepath.Join(dir, "projection"), 0700); err != nil {
		return nil, err
	}
	var caller strings.Builder
	caller.WriteString("package main\nimport(\"encoding/json\";\"os\";p \"gooo.fresh.execution/projection\")\nfunc main(){json.NewEncoder(os.Stdout).Encode([]int64{")
	for _, c := range cases {
		fmt.Fprintf(&caller, "p.ChoosePath(%d),", c.Input)
	}
	caller.WriteString("})}\n")
	for name, content := range map[string]string{"go.mod": "module gooo.fresh.execution\n\ngo 1.27.1\n", "projection/generated.go": source, "main.go": caller.String()} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			return nil, err
		}
	}
	raw, _, err := child(dir, goBinary, "run", ".")
	if err != nil {
		return nil, err
	}
	var values []int64
	if err = json.Unmarshal(raw, &values); err != nil || len(values) != len(cases) {
		return nil, errors.New("compiled Go actual count differs")
	}
	for i, v := range values {
		if v != cases[i].Expected {
			return nil, errors.New("compiled Go independent oracle differs")
		}
	}
	return values, nil
}

func runNative(curriculum, models, study, output, revision, binary, goBinary string) error {
	if err := nativePins(binary, goBinary); err != nil {
		return err
	}
	if err := preflight(output, revision); err != nil {
		return err
	}
	views, err := loadViews(curriculum)
	if err != nil {
		return err
	}
	loaded, pins, err := loadModels(models)
	if err != nil {
		return err
	}
	var selection struct {
		Selected    string           `json:"selected"`
		Model       modelPin         `json:"model"`
		Calibration map[string]score `json:"calibration"`
	}
	if err = decodeFile(filepath.Join(study, "selection.json"), &selection); err != nil {
		return err
	}
	chosen, err := choose(selection.Calibration, pins)
	if err != nil || chosen != selection.Selected || selection.Model != pins[chosen] {
		return errors.New("frozen calibration selection differs")
	}
	ids := [4]string{chosen, "independent-fp32", "joint-fp32", "offline"}
	policies := [4]string{"selected", "independent", "joint", "offline"}
	priors := map[string]map[string]observation{}
	paths := map[string]string{}
	for _, id := range ids {
		priors[id], err = priorObservations(study, "development", id)
		if err != nil {
			return err
		}
		if id != "offline" {
			parts := strings.SplitN(id, "-", 2)
			paths[id], err = filepath.Abs(filepath.Join(models, parts[0], "models", parts[1], "model.json"))
			if err != nil {
				return err
			}
		}
	}
	binSHA, err := fileHash(binary)
	if err != nil {
		return err
	}
	goSHA, err := fileHash(goBinary)
	if err != nil {
		return err
	}
	selectionSHA, err := fileHash(filepath.Join(study, "selection.json"))
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/joint-composition-native-preexecution/v1", "runner_revision": revision,
		"native_revision": nativeDeployed, "sdk": "v0.2.12-experimental", "native_binary_sha256": binSHA, "go_binary_sha256": goSHA,
		"model_pins": pins, "selected_candidate": chosen, "selection_sha256": selectionSHA, "planned_native_calls": 192,
		"planned_independently_compiled_go_executions": 192, "planned_ordered_go_function_invocations": 3072,
		"dataset_sha256": datasetSHA, "new_optimizer_updates": 0, "host_cpu_utilization_delta_measured": false}); err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "gooo-fresh-native-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	if err = save(filepath.Join(workspace, "ci.json"), pathplan.CIHint{SourceSHA: nativeMain, Status: "PASS"}); err != nil {
		return err
	}
	totals := map[string]counts{}
	measurements := map[string][]metrics{}
	initialPairs := map[string]map[string]map[string]uint16{}
	calls, predictions, executions := 0, 0, 0
	for _, family := range jointcompositionstudy.Families {
		for goal := 0; goal < 4; goal++ {
			for _, language := range []string{"en", "ko"} {
				for i, id := range ids {
					feature := decision.SemanticContextIntentFeatureVersion
					if id != "offline" {
						feature = loaded[id].FeatureVersion()
					}
					var v view
					for _, candidate := range views {
						r := candidate.Rows[0]
						if r.Family == family && r.Config == 40 && r.Desired == goal && r.Language == language && r.Feature == feature {
							v = candidate
							break
						}
					}
					if v.ID == "" {
						return errors.New("fixed native subset missing")
					}
					doc, source, e := originalDocument(v)
					if e != nil {
						return e
					}
					if err = save(filepath.Join(workspace, "plan.json"), doc); err != nil {
						return err
					}
					if err = os.WriteFile(filepath.Join(workspace, "source.gooo"), source, 0600); err != nil {
						return err
					}
					args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ChoosePath", "source.gooo"}
					if id != "offline" {
						args = append(args, "--path-model", paths[id], "--path-feedback-rounds", "3", "--path-feedback-unfixed", "--path-feedback-ci", "ci.json")
					}
					raw, m, runErr := child(workspace, binary, args...)
					calls++
					name := fmt.Sprintf("%s-goal%d-%s-%s", family, goal, language, policies[i])
					if err = os.WriteFile(filepath.Join(output, name+".json"), raw, 0644); err != nil {
						return err
					}
					if runErr != nil {
						return runErr
					}
					n, o, e := inspectNative(raw, v, doc, source, loaded[id])
					if e != nil {
						return fmt.Errorf("%s: %w", name, e)
					}
					prior := priors[id][v.ID]
					if !reflect.DeepEqual(o.Search.Attempts, prior.Search.Attempts) || o.Search.Selection.ModelCalls != prior.Search.Selection.ModelCalls ||
						!reflect.DeepEqual(o.Search.InitialProposals, prior.Search.InitialProposals) {
						return errors.New("native/SDK candidate observations differ")
					}
					predictions += o.Search.Selection.ModelCalls
					values, e := executeGo(goBinary, n.Source, doc.Cases)
					if e != nil {
						return e
					}
					executions++
					if err = save(filepath.Join(output, name+"-execution.json"), map[string]any{"capture_sha256": hash(raw), "emitted_go_sha256": hash([]byte(n.Source)),
						"actual_compiled_go_execution": true, "ordered_actual_values": values, "case_denominator": 16, "process_metrics": m}); err != nil {
						return err
					}
					count := totals[policies[i]]
					add(&count, o)
					if err = addTarget(&count, o, v.Rows[0].Finite.Joint); err != nil {
						return err
					}
					totals[policies[i]] = count
					measurements[policies[i]] = append(measurements[policies[i]], m)
					if initialPairs[policies[i]] == nil {
						initialPairs[policies[i]] = map[string]map[string]uint16{}
					}
					pairKey := fmt.Sprintf("%s/%d", family, goal)
					if initialPairs[policies[i]][pairKey] == nil {
						initialPairs[policies[i]][pairKey] = map[string]uint16{}
					}
					initialPairs[policies[i]][pairKey][language] = o.Search.Attempts[0].Mask
				}
			}
		}
	}
	if calls != 192 || executions != 192 {
		return errors.New("fixed native execution count differs")
	}
	for policy, pairs := range initialPairs {
		c := totals[policy]
		for _, languages := range pairs {
			if len(languages) != 2 {
				return errors.New("native bilingual pair missing")
			}
			c.Pairs++
			if languages["en"] != languages["ko"] {
				c.Different++
			}
		}
		totals[policy] = c
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/joint-composition-native/v1", "status": "PASS", "source_revision": revision,
		"native_revision": nativeDeployed, "selected_candidate": chosen, "actual_native_calls": calls, "actual_model_predictions": predictions,
		"independently_compiled_go_executions": executions, "ordered_go_function_invocations": executions * 16, "totals": totals, "native_process_measurements": measurements,
		"new_optimizer_updates": 0, "subagents_used": 0, "host_cpu_utilization_delta_measured": false,
		"scope": "48 fresh config-40 bilingual function views across selected independent FP32, independent FP32 reference, joint FP32 and disconnected policies. Selected/reference are duplicate model replays. All 192 native outputs independently compiled/executed over all 16 ordered oracle cases. Source/context/model/candidates agree with frozen SDK study. Process CPU is normalized to one core; fixed arm order and child RSS do not measure causal host utilization or memory deltas."})
}
