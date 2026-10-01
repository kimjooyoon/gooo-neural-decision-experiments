package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bilingualstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/feedbackstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathstudy"
)

const compilerCurriculumProtocol = "docs/compiler-context-training-preregistration.md"

// This partial decoder retains the entire raw native receipt separately.
type compilerExport struct {
	Schema   string `json:"schema"`
	Source   string `json:"original_source_sha256"`
	Document string `json:"document_sha256"`
	Binding  struct {
		Equivalent bool   `json:"equivalent"`
		Semantic   string `json:"source_semantic_digest"`
	} `json:"source_binding"`
	Context *ownContextReceipt `json:"context"`
	Inputs  []struct {
		ID      string `json:"decision_id"`
		Text    string `json:"text"`
		SHA     string `json:"input_sha256"`
		Natural string `json:"natural_intent_sha256"`
		Bytes   int    `json:"bytes"`
	} `json:"inputs"`
	Predictions int  `json:"model_predictions"`
	Tests       int  `json:"candidate_tests"`
	Emission    bool `json:"selected_emission"`
	Writes      int  `json:"repository_writes"`
}

type compilerTrainingRow struct {
	ID            string              `json:"id"`
	Pair          string              `json:"pair_id"`
	Program       string              `json:"program_id"`
	Template      string              `json:"template_pair_id"`
	Family        string              `json:"family"`
	Configuration int                 `json:"configuration_index"`
	Language      string              `json:"language"`
	Split         string              `json:"split"`
	Text          string              `json:"text"`
	SHA           string              `json:"input_sha256"`
	Control       string              `json:"caller_context_text"`
	ControlSHA    string              `json:"caller_context_sha256"`
	Options       [2]string           `json:"eligible_labels"`
	Targets       [2]float32          `json:"finite_soft_targets"`
	Intention     string              `json:"intention_label"`
	Accepted      []string            `json:"best_finite_labels"`
	Cases         []pathplan.TestCase `json:"finite_cases"`
	SourceSHA     string              `json:"source_sha256"`
	PlanSHA       string              `json:"original_plan_sha256"`
	CaptureSHA    string              `json:"export_capture_sha256"`
}

type compilerCapture struct {
	ID      string  `json:"id"`
	Receipt []byte  `json:"native_receipt"`
	Metrics metrics `json:"process_metrics"`
}

func compilerFixture(row feedbackstudy.Row) (document, []byte, error) {
	plan, err := pathstudy.Fixture(row.Family, row.Configuration, row.OriginalText)
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
	source := []byte("package owntraining\nnamespace owntraining\nentity Integer id \"owntraining://entity/integer\"\n" +
		"activity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	return document{Schema: "gooo/body-codegen-typed-path-plan/v1", Plan: plan, Cases: row.Cases, Max: 2}, source, nil
}

func inspectCompilerExport(raw []byte, doc document, source []byte) (compilerExport, error) {
	var value compilerExport
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, err
	}
	docRaw, _ := json.Marshal(doc)
	if value.Schema != "gooo/compiler-path-input-export/v1" || value.Source != "sha256:"+hash(source) ||
		value.Document != "sha256:"+hash(docRaw) || !value.Binding.Equivalent || value.Binding.Semantic == "" ||
		value.Context == nil || value.Context.Schema != "gooo/compiler-typed-path-context/v2" || value.Context.Status != "ENCODED" ||
		value.Context.Feature != decision.SplitContextIntentFeatureVersion || value.Context.Metadata != "" ||
		len(value.Inputs) != 1 || len(value.Context.Inputs) != 1 || value.Predictions != 0 || value.Tests != 0 || value.Emission || value.Writes != 0 {
		return value, errors.New("compiler export authority or scope mismatch")
	}
	in := value.Inputs[0]
	intent := doc.Plan.Decisions[0].Intent
	intent = intent[strings.LastIndex(intent, "intent: ")+len("intent: "):]
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil || in.ID != "structure" || in.SHA != "sha256:"+hash([]byte(in.Text)) || in.SHA != value.Context.Inputs[0].SHA ||
		in.Bytes != len(in.Text) || in.Bytes > 512 || in.Natural != "sha256:"+hash([]byte(intent)) ||
		!strings.HasSuffix(in.Text, ";intent: "+intent) || !strings.Contains(in.Text, ";basis=source_fallback;") ||
		value.Context.Original != prepared.PlanSHA256() {
		return value, errors.New("compiler export input/plan digest differs")
	}
	return value, nil
}

func runCompilerCurriculum(binary, output, revision, nativeRevision string) error {
	if err := diagnosisPreflight(output, revision); err != nil {
		return err
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 binary required")
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
	if settings["vcs.revision"] != nativeRevision || settings["vcs.modified"] != "false" || sdk != "v0.2.9-experimental" {
		return errors.New("clean pinned native source and SDK .9 required")
	}
	pairs, err := bilingualstudy.Load("data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		return err
	}
	if _, err = os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh curriculum directory required")
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	binarySHA, err := executableHash(binary)
	if err != nil {
		return err
	}
	protocol, err := read(compilerCurriculumProtocol)
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{
		"schema": "gooo/compiler-curriculum-preexecution/v1", "runner_revision": revision, "native_revision": nativeRevision,
		"native_binary_sha256": binarySHA, "dataset_sha256": bilingualstudy.DatasetSHA, "protocol_sha256": hash(protocol),
		"planned_native_export_calls": 2080, "model_predictions": 0, "new_independent_intentions": 0}); err != nil {
		return err
	}
	rowsFile, err := os.OpenFile(filepath.Join(output, "dataset.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer rowsFile.Close()
	captures, err := os.OpenFile(filepath.Join(output, "exports.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer captures.Close()
	workspace, err := os.MkdirTemp("", "gooo-public-curriculum-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	rowsEncoder, captureEncoder := json.NewEncoder(rowsFile), json.NewEncoder(captures)
	count, programs, splits := 0, map[string]bool{}, map[string]int{}
	for _, pair := range pairs {
		for _, row := range pair.Rows {
			doc, source, err := compilerFixture(row)
			if err != nil {
				return err
			}
			if err = os.WriteFile(filepath.Join(workspace, "source.gooo"), source, 0644); err != nil {
				return err
			}
			if err = save(filepath.Join(workspace, "plan.json"), doc); err != nil {
				return err
			}
			raw, m, runErr := child(context.Background(), workspace, binary, "body-context", "--plan", "plan.json", "--activity", "ChoosePath", "source.gooo")
			count++
			if err = captureEncoder.Encode(compilerCapture{row.ID, raw, m}); err != nil {
				return err
			}
			if runErr != nil {
				return runErr
			}
			value, err := inspectCompilerExport(raw, doc, source)
			if err != nil {
				return err
			}
			in := value.Inputs[0]
			training := compilerTrainingRow{row.ID, pair.ID, pair.Program, pair.Template, pair.Family, row.Configuration, row.Language, pair.Split,
				in.Text, hash([]byte(in.Text)), row.OriginalText, hash([]byte(row.OriginalText)), pair.Options, pair.Targets, row.IntentionLabel,
				row.Accepted, row.Cases, hash(source), value.Context.Original, hash(raw)}
			if err = rowsEncoder.Encode(training); err != nil {
				return err
			}
			programs[pair.Program], splits[pair.Split] = true, splits[pair.Split]+1
		}
	}
	if count != 2080 || len(programs) != 640 || splits["train"] != 1600 || splits["calibration"] != 160 || splits["test"] != 320 {
		return errors.New("curriculum denominators changed")
	}
	if err = rowsFile.Close(); err != nil {
		return err
	}
	if err = captures.Close(); err != nil {
		return err
	}
	data, err := readCompilerCurriculumFile(filepath.Join(output, "dataset.jsonl"))
	if err != nil {
		return err
	}
	raw, err := readCompilerCurriculumFile(filepath.Join(output, "exports.jsonl"))
	if err != nil {
		return err
	}
	return save(filepath.Join(output, "manifest.json"), map[string]any{
		"schema": "gooo/compiler-curriculum/v1", "status": "SOURCE_BOUND_EXPORTED", "native_revision": nativeRevision, "runner_revision": revision,
		"dataset_sha256": hash(data), "captures_sha256": hash(raw), "rows": count, "bilingual_pairs": len(pairs), "program_groups": len(programs), "views_by_split": splits,
		"actual_native_export_calls": count, "model_predictions": 0, "candidate_tests": 0, "selected_emissions": 0, "new_independent_intentions": 0,
		"scope": "reused synthetic public intentions; original source/fallback validation projections generated; canonical initial inputs exclude finite expectations and observed feedback"})
}

func auditCompilerCurriculum(output string) error {
	pairs, err := bilingualstudy.Load("data/feedback-path-v1/dataset.jsonl")
	if err != nil {
		return err
	}
	manifestRaw, err := read(filepath.Join(output, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Data     string `json:"dataset_sha256"`
		Captures string `json:"captures_sha256"`
		Rows     int    `json:"rows"`
		Calls    int    `json:"actual_native_export_calls"`
	}
	if err = json.Unmarshal(manifestRaw, &manifest); err != nil {
		return err
	}
	data, err := readCompilerCurriculumFile(filepath.Join(output, "dataset.jsonl"))
	if err != nil {
		return err
	}
	captures, err := readCompilerCurriculumFile(filepath.Join(output, "exports.jsonl"))
	if err != nil {
		return err
	}
	if manifest.Data != hash(data) || manifest.Captures != hash(captures) || manifest.Rows != 2080 || manifest.Calls != 2080 {
		return errors.New("manifest capture/data pins differ")
	}
	rows, receipts := strings.Split(strings.TrimSpace(string(data)), "\n"), strings.Split(strings.TrimSpace(string(captures)), "\n")
	if len(rows) != 2080 || len(receipts) != 2080 {
		return errors.New("2080 bound rows required")
	}
	index := 0
	for _, pair := range pairs {
		for _, original := range pair.Rows {
			var row compilerTrainingRow
			var captured compilerCapture
			if err = json.Unmarshal([]byte(rows[index]), &row); err != nil {
				return err
			}
			if err = json.Unmarshal([]byte(receipts[index]), &captured); err != nil {
				return err
			}
			doc, source, err := compilerFixture(original)
			if err != nil {
				return err
			}
			value, err := inspectCompilerExport(captured.Receipt, doc, source)
			if err != nil {
				return err
			}
			expected := compilerTrainingRow{original.ID, pair.ID, pair.Program, pair.Template, pair.Family, original.Configuration, original.Language, pair.Split,
				value.Inputs[0].Text, hash([]byte(value.Inputs[0].Text)), original.OriginalText, hash([]byte(original.OriginalText)), pair.Options, pair.Targets,
				original.IntentionLabel, original.Accepted, original.Cases, hash(source), value.Context.Original, hash(captured.Receipt)}
			a, _ := json.Marshal(row)
			b, _ := json.Marshal(expected)
			if string(a) != string(b) || captured.ID != row.ID {
				return fmt.Errorf("row %d source/target binding differs", index)
			}
			index++
		}
	}
	return nil
}

// Dedicated bounded cohort reader; the generic old-study 8MiB limit is unchanged.
func readCompilerCurriculumFile(name string) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 32<<20 {
		return nil, errors.New("bounded regular compiler cohort required")
	}
	return os.ReadFile(name)
}
