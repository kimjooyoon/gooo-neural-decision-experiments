package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const ownNativeContextRoot = "studies/own-model-native-context-v1/"
const ownNativeContextProtocol = "docs/own-model-native-context-preregistration.md"
const ownNativeModels = "runs/split-context-judgment-mps-20261001/v2/models/"

var ownNativeContextPins = map[string]string{
	"fp32":        "7998ca6e5cbe28455e0467a19bc77c03f99683d8623f35ebe95f79e905624fbd",
	"ptq_ternary": "2f996c11983485c4e2fb1e9c2b08444042d8725a3bf0947025b318014d28ea50",
	"qat_ternary": "6a0162e814c589ac833b3f3ac2b7eeff4dd48ad75a015ee822523c534995196d",
}

type ownContextInput struct {
	ID       string `json:"decision_id"`
	Original string `json:"original_intent_sha256"`
	Natural  string `json:"natural_intent_sha256"`
	SHA      string `json:"input_sha256,omitempty"`
	Bytes    int    `json:"bytes"`
}
type ownContextReceipt struct {
	Schema          string            `json:"schema"`
	Status          string            `json:"status"`
	Activity        string            `json:"activity_id"`
	Source          string            `json:"source_semantic_sha256"`
	Original        string            `json:"original_plan_sha256"`
	Ranked          string            `json:"ranked_plan_sha256,omitempty"`
	Metadata        string            `json:"model_metadata_sha256"`
	Feature         string            `json:"feature_version"`
	Inputs          []ownContextInput `json:"inputs"`
	Reason          string            `json:"reason,omitempty"`
	FeedbackSkipped bool              `json:"feedback_skipped,omitempty"`
}
type ownContextRow struct {
	ID, Source, Activity string
	Document             document
	Overflow             bool
}
type ownContextExecution struct {
	ID         string  `json:"id"`
	CaptureSHA string  `json:"capture_sha256"`
	SourceSHA  string  `json:"generated_go_sha256"`
	Values     []int64 `json:"actual_go_values"`
}

func ownContextRows() ([]ownContextRow, error) {
	var rows []ownContextRow
	for _, language := range []string{"en", "ko"} {
		for _, contract := range []string{"sparse", "full"} {
			id := language + "-" + contract
			raw, err := read("studies/own-model-sdk-context-v1/" + id + ".json")
			if err != nil {
				return nil, err
			}
			var d document
			if err = json.Unmarshal(raw, &d); err != nil {
				return nil, err
			}
			rows = append(rows, ownContextRow{id, ownNativeContextRoot + "subtraction.gooo.fixture", "Probe", d, false})
		}
	}
	for _, language := range []string{"en", "ko"} {
		raw, err := read(ownNativeContextRoot + "conditional-" + language + ".json")
		if err != nil {
			return nil, err
		}
		var d document
		if err = json.Unmarshal(raw, &d); err != nil {
			return nil, err
		}
		rows = append(rows, ownContextRow{"conditional-" + language, ownNativeContextRoot + "conditional.gooo.fixture", "ConditionalAssign", d, false})
	}
	raw, err := read(ownNativeContextRoot + "compound.json")
	if err != nil {
		return nil, err
	}
	var d document
	if err = json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	rows = append(rows, ownContextRow{"compound", ownNativeContextRoot + "compound.gooo.fixture", "Combined", d, false})
	raw, err = read(ownNativeContextRoot + "conditional-ko.json")
	if err != nil {
		return nil, err
	}
	d = document{}
	if err = json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	d.Plan.Decisions[0].Intent = strings.Repeat("가", 150)
	d.Cases[6].Expected = 999
	rows = append(rows, ownContextRow{"overflow-partial", ownNativeContextRoot + "conditional.gooo.fixture", "ConditionalAssign", d, true})
	return rows, nil
}

func runOwnNativeContext(binary, goBinary, output, revision, nativeRevision string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 compiler required")
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
	if settings["vcs.revision"] != nativeRevision || settings["vcs.modified"] != "false" || sdk != "v0.2.9-experimental" {
		return errors.New("clean pinned native source and SDK .9 required")
	}
	rows, err := ownContextRows()
	if err != nil || len(rows) != 8 {
		return errors.New("eight fixed documents required")
	}
	binarySHA, err := executableHash(binary)
	if err != nil {
		return err
	}
	inputs := map[string]string{}
	for _, row := range rows {
		raw, _ := json.Marshal(row.Document)
		inputs[row.ID+"-document"] = hash(raw)
		raw, err = read(row.Source)
		if err != nil {
			return err
		}
		inputs[row.ID+"-source"] = hash(raw)
	}
	protocol, err := read(ownNativeContextProtocol)
	if err != nil {
		return err
	}
	for arm, pin := range ownNativeContextPins {
		model, e := decision.LoadPath(ownNativeModels + arm + "/model.json")
		if e != nil || model.MetadataSHA256() != pin {
			return errors.New("own model pin mismatch")
		}
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	pre := map[string]any{"schema": "gooo/own-model-native-context-preexecution/v1", "runner_revision": revision,
		"native_revision": nativeRevision, "native_binary_sha256": binarySHA, "sdk": "v0.2.9-experimental",
		"protocol_sha256": hash(protocol), "input_sha256": inputs, "model_pins": ownNativeContextPins,
		"planned_native_calls": 32, "planned_model_predictions": 57, "planned_go_execution_processes": 32,
		"planned_selected_finite_cases": 120, "new_independent_intentions": 0}
	if err = save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	for i, row := range rows {
		arms := []string{"fp32", "ptq_ternary", "qat_ternary", "offline"}
		if i%2 != 0 {
			arms = []string{"offline", "qat_ternary", "ptq_ternary", "fp32"}
		}
		for _, arm := range arms {
			if err = captureOwnNativeContext(binary, goBinary, output, row, arm); err != nil {
				return err
			}
		}
	}
	report, err := auditOwnNativeContext(output, revision, nativeRevision)
	if err != nil {
		return err
	}
	return save(filepath.Join(output, "report.json"), report)
}

func captureOwnNativeContext(binary, goBinary, output string, row ownContextRow, arm string) error {
	id := row.ID + "-" + arm
	dir, err := os.MkdirTemp("", "gooo-own-native-context-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	raw, _ := json.Marshal(row.Document)
	if err = os.WriteFile(filepath.Join(dir, "plan.json"), raw, 0600); err != nil {
		return err
	}
	source, err := read(row.Source)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "source.gooo"), source, 0600); err != nil {
		return err
	}
	args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--activity", row.Activity, "source.gooo"}
	if arm != "offline" {
		model, e := filepath.Abs(ownNativeModels + arm + "/model.json")
		if e != nil {
			return e
		}
		args = append(args, "--path-model", model)
		if row.Overflow {
			args = append(args, "--path-step-attempts", "8", "--path-feedback-rounds", "2")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	capture, m, err := child(ctx, dir, binary, args...)
	if e := os.WriteFile(filepath.Join(output, id+".json"), capture, 0644); e != nil {
		return e
	}
	if e := save(filepath.Join(output, id+"-metrics.json"), m); e != nil {
		return e
	}
	if err != nil {
		return err
	}
	var result nativeResult
	if err = json.Unmarshal(capture, &result); err != nil {
		return err
	}
	var inputs []int64
	for _, test := range row.Document.Cases {
		inputs = append(inputs, test.Input)
	}
	values, err := executeFunction(ctx, goBinary, result.Source, row.Activity, inputs)
	if err != nil {
		return err
	}
	execution := ownContextExecution{ID: id, CaptureSHA: hash(capture), SourceSHA: hash([]byte(result.Source))}
	if err = json.Unmarshal(values, &execution.Values); err != nil {
		return err
	}
	return save(filepath.Join(output, id+"-execution.json"), execution)
}
