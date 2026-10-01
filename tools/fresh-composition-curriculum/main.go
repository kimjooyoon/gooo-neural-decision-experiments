// fresh-composition-curriculum captures released native source-bound inputs.
// It does not load weights, predict, optimize or select candidates.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const protocol = "docs/fresh-composition-semantic-v3-preregistration-20261002.md"
const protocolRevision = "d9fc4d1c8140c278794c6aabde3a7585af4692c0"

var featureArms = [2]string{decision.SplitContextIntentFeatureVersion, decision.SemanticContextIntentFeatureVersion}
var privatePattern = regexp.MustCompile(`/Users/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func save(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}

type document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Max    int                 `json:"max_attempts"`
}

type input struct {
	ID             string `json:"decision_id"`
	Text           string `json:"text"`
	SHA            string `json:"input_sha256"`
	Natural        string `json:"natural_intent_sha256"`
	Original       string `json:"original_intent_sha256"`
	SourceFeatures string `json:"source_features_sha256"`
	Bytes          int    `json:"bytes"`
	Replaced       bool   `json:"caller_prefix_replaced"`
}

type export struct {
	Schema   string `json:"schema"`
	Source   string `json:"original_source_sha256"`
	Document string `json:"document_sha256"`
	TestsSHA string `json:"test_suite_sha256"`
	Binding  struct {
		Equivalent bool   `json:"equivalent"`
		Semantic   string `json:"source_semantic_digest"`
	} `json:"source_binding"`
	Context struct {
		Schema   string  `json:"schema"`
		Status   string  `json:"status"`
		Feature  string  `json:"feature_version"`
		Original string  `json:"original_plan_sha256"`
		Semantic string  `json:"source_semantic_sha256"`
		Metadata string  `json:"model_metadata_sha256"`
		Inputs   []input `json:"inputs"`
	} `json:"context"`
	Inputs      []input `json:"inputs"`
	Predictions int     `json:"model_predictions"`
	Tests       int     `json:"candidate_tests"`
	Emission    bool    `json:"selected_emission"`
	Writes      int     `json:"repository_writes"`
}

func fixture(family string, config, desired int, language string) (document, []byte, compositionstudy.FiniteTarget, error) {
	plan, err := compositionstudy.Fixture(family, config, desired, language)
	if err != nil {
		return document{}, nil, compositionstudy.FiniteTarget{}, err
	}
	target, err := compositionstudy.Target(plan, family, config, desired)
	if err != nil {
		return document{}, nil, target, err
	}
	cases, err := compositionstudy.Cases(family, config, desired)
	if err != nil {
		return document{}, nil, target, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return document{}, nil, target, err
	}
	body, err := json.Marshal(prepared.Fallback().GoooBody())
	if err != nil {
		return document{}, nil, target, err
	}
	source := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	return document{"gooo/body-codegen-typed-path-plan/v1", plan, cases, 4}, source, target, nil
}

func inspect(raw []byte, doc document, source []byte, feature string) (export, error) {
	var value export
	if len(raw) > 1<<20 || privatePattern.Match(raw) || json.Unmarshal(raw, &value) != nil {
		return value, errors.New("bounded public export JSON required")
	}
	var counters struct {
		Predictions *int  `json:"model_predictions"`
		Tests       *int  `json:"candidate_tests"`
		Emission    *bool `json:"selected_emission"`
		Writes      *int  `json:"repository_writes"`
	}
	if json.Unmarshal(raw, &counters) != nil || counters.Predictions == nil || counters.Tests == nil || counters.Emission == nil || counters.Writes == nil {
		return value, errors.New("explicit zero-counter export envelope required")
	}
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		return value, err
	}
	docRaw, err := json.Marshal(doc)
	if err != nil {
		return value, err
	}
	schema, contextSchema := "gooo/compiler-path-input-export/v1", "gooo/compiler-typed-path-context/v2"
	if feature == decision.SemanticContextIntentFeatureVersion {
		schema, contextSchema = "gooo/compiler-path-input-export/v2", "gooo/compiler-typed-path-context/v3"
	}
	if value.Schema != schema || value.Source != "sha256:"+hash(source) || value.Document != "sha256:"+hash(docRaw) || value.TestsSHA == "" || !value.Binding.Equivalent || value.Binding.Semantic == "" || value.Context.Semantic != value.Binding.Semantic || value.Context.Schema != contextSchema || value.Context.Status != "ENCODED" || value.Context.Feature != feature || value.Context.Original != prepared.PlanSHA256() || value.Context.Metadata != "" || len(value.Inputs) != 2 || len(value.Context.Inputs) != 2 || value.Predictions != 0 || value.Tests != 0 || value.Emission || value.Writes != 0 {
		return value, errors.New("native source-bound zero-prediction export contract differs")
	}
	for i, in := range value.Inputs {
		choice := doc.Plan.Decisions[i]
		natural := choice.Intent[strings.LastIndex(choice.Intent, "intent: ")+8:]
		other := value.Context.Inputs[i]
		if in.ID != choice.ID || in.SHA != "sha256:"+hash([]byte(in.Text)) || in.SHA != other.SHA || in.Bytes != len(in.Text) || in.Bytes > 512 || in.Natural != "sha256:"+hash([]byte(natural)) || in.Original != "sha256:"+hash([]byte(choice.Intent)) || !in.Replaced || !strings.HasSuffix(in.Text, ";intent: "+natural) || in.ID != other.ID || in.Natural != other.Natural || in.SourceFeatures != other.SourceFeatures || in.Bytes != other.Bytes {
			return value, errors.New("complete native input or digest differs")
		}
		if feature == decision.SemanticContextIntentFeatureVersion {
			fields, err := prepared.SourceFeatures(choice.ID)
			if err != nil {
				return value, err
			}
			text, err := decision.EncodeSemanticContextInput(fields, natural)
			if err != nil || in.Text != text || in.SourceFeatures != "sha256:"+hash(fields[:]) {
				return value, errors.New("native/source-array encoder differs")
			}
		} else if in.SourceFeatures != "" || !strings.Contains(in.Text, ";basis=source_fallback;") {
			return value, errors.New("legacy compiler feature arm differs")
		}
	}
	return value, nil
}

func pin(binary, evidencePath, revision string) (string, string, error) {
	var evidence struct {
		Status string `json:"status"`
		Native string `json:"native_main_revision"`
		SDK    string `json:"sdk_version"`
	}
	raw, err := os.ReadFile(evidencePath)
	if err != nil || json.Unmarshal(raw, &evidence) != nil || evidence.Status != "NATIVE_MAIN_VERIFIED" || evidence.SDK != "v0.2.11-experimental" || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(evidence.Native) {
		return "", "", errors.New("released native main evidence required")
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return "", "", errors.New("Go 1.27.1 native binary required")
	}
	settings := map[string]string{}
	for _, value := range info.Settings {
		settings[value.Key] = value.Value
	}
	sdk := ""
	for _, dep := range info.Deps {
		if dep.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = dep.Version
		}
	}
	if settings["vcs.revision"] != evidence.Native || settings["vcs.modified"] != "false" || sdk != evidence.SDK {
		return "", "", errors.New("native main build and SDK differ from evidence")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return "", "", errors.New("exact committed runner revision required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return "", "", errors.New("clean runner required before collection")
	}
	raw, err = os.ReadFile(binary)
	if err != nil {
		return "", "", err
	}
	return evidence.Native, hash(raw), nil
}

type row struct {
	ID          string                        `json:"id"`
	Group       string                        `json:"program_contract_group"`
	Pair        string                        `json:"bilingual_decision_pair"`
	Family      string                        `json:"family"`
	Config      int                           `json:"configuration"`
	Desired     int                           `json:"desired_mask"`
	Language    string                        `json:"language"`
	Split       string                        `json:"split"`
	Template    string                        `json:"template_pair_id"`
	Feature     string                        `json:"feature_version"`
	Coordinate  int                           `json:"coordinate"`
	Input       input                         `json:"input"`
	Options     [2]string                     `json:"eligible_labels"`
	Targets     [2]float32                    `json:"finite_soft_targets"`
	Finite      compositionstudy.FiniteTarget `json:"full_contract_target"`
	SourceSHA   string                        `json:"source_sha256"`
	DocumentSHA string                        `json:"document_sha256"`
	CaptureSHA  string                        `json:"capture_sha256"`
}

// Output is bounded even on malformed child behavior; the context kills a
// child that fails to finish. Collection makes one native call at a time.
type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 1<<20 {
		return 0, errors.New("child output exceeds capture bound")
	}
	return b.Buffer.Write(raw)
}

func child(workspace, binary, feature string) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "body-context", "--plan", "plan.json", "--activity", "ChoosePath", "--feature-version", feature, "source.gooo")
	command.Dir = workspace
	var stdout, stderr limitedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	start := time.Now()
	err := command.Run()
	return stdout.Bytes(), time.Since(start).Nanoseconds(), err
}

func collect(binary, evidencePath, output, revision string) (resultErr error) {
	native, binarySHA, err := pin(binary, evidencePath, revision)
	if err != nil {
		return err
	}
	protocolRaw, err := exec.Command("git", "show", protocolRevision+":"+protocol).Output()
	if err != nil {
		return err
	}
	if _, err = os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh collection directory required")
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	preexecution := map[string]any{"schema": "gooo/fresh-composition-collection-preexecution/v1", "runner_revision": revision, "native_main_revision": native, "native_binary_sha256": binarySHA, "sdk": "v0.2.11-experimental", "protocol_revision": protocolRevision, "protocol_sha256": hash(protocolRaw), "planned_native_export_calls": 4608, "planned_decision_rows": 9216, "feature_arms": featureArms, "new_optimizer_updates": 0, "model_predictions": 0, "candidate_tests": 0}
	if err = save(filepath.Join(output, "preexecution.json"), preexecution); err != nil {
		return err
	}
	rowsFile, err := os.Create(filepath.Join(output, "dataset.jsonl"))
	if err != nil {
		return err
	}
	defer rowsFile.Close()
	captureFile, err := os.Create(filepath.Join(output, "exports.jsonl"))
	if err != nil {
		return err
	}
	defer captureFile.Close()
	rows, captures := json.NewEncoder(rowsFile), json.NewEncoder(captureFile)
	workspace, err := os.MkdirTemp("", "gooo-fresh-composition-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	calls, count := 0, 0
	defer func() {
		status := "COMPLETE_FIXED_COLLECTION"
		if resultErr != nil {
			status = "FAILED_PARTIAL_COLLECTION_RETAINED"
		}
		err := save(filepath.Join(output, "collection-attempt.json"), map[string]any{"schema": "gooo/fresh-composition-collection-attempt/v1", "status": status, "actual_native_export_calls": calls, "retained_decision_rows": count, "planned_native_export_calls": 4608, "planned_decision_rows": 9216, "new_optimizer_updates": 0, "scope": "attempt counts retained even on failure; incomplete collections cannot train"})
		if resultErr == nil && err != nil {
			resultErr = err
		}
	}()
	splits := map[string]int{}
	for _, family := range compositionstudy.Families {
		for config := 0; config < 48; config++ {
			for desired := 0; desired < 4; desired++ {
				for _, language := range [2]string{"en", "ko"} {
					doc, source, target, err := fixture(family, config, desired, language)
					if err != nil {
						return err
					}
					if err = os.WriteFile(filepath.Join(workspace, "source.gooo"), source, 0600); err != nil {
						return err
					}
					if err = save(filepath.Join(workspace, "plan.json"), doc); err != nil {
						return err
					}
					group := fmt.Sprintf("%s-c%02d-goal%d", family, config, desired)
					for _, feature := range featureArms {
						id := group + "-" + language + "-" + feature
						raw, wall, runErr := child(workspace, binary, feature)
						calls++
						if !json.Valid(raw) || privatePattern.Match(raw) {
							return errors.New("native export did not provide public bounded JSON")
						}
						if err = captures.Encode(map[string]any{"id": id, "native_receipt": json.RawMessage(raw), "wall_ns": wall, "child_succeeded": runErr == nil}); err != nil {
							return err
						}
						if runErr != nil {
							return errors.New("native export failed; partial collection retained without training")
						}
						value, err := inspect(raw, doc, source, feature)
						if err != nil {
							return err
						}
						for i, in := range value.Inputs {
							choice := doc.Plan.Decisions[i]
							pair := fmt.Sprintf("%s-%s-choice%d", group, feature, i)
							r := row{id + "-" + choice.ID, group, pair, family, config, desired, language, compositionstudy.Split(config), compositionstudy.TemplateID(choice.Kind, config, language), feature, i, in, [2]string{choice.Options[0].Label, choice.Options[1].Label}, target.Marginals[i], target, hash(source), value.Document, hash(raw)}
							if err = rows.Encode(r); err != nil {
								return err
							}
							count++
							splits[feature+"/"+r.Split]++
						}
						if calls%256 == 0 {
							fmt.Printf("{\"status\":\"COLLECTING\",\"native_export_calls\":%d,\"decision_rows\":%d}\n", calls, count)
						}
					}
				}
			}
		}
	}
	if calls != 4608 || count != 9216 {
		return errors.New("complete fixed collection denominator differs")
	}
	if err = rowsFile.Close(); err != nil {
		return err
	}
	if err = captureFile.Close(); err != nil {
		return err
	}
	dataset, err := os.ReadFile(filepath.Join(output, "dataset.jsonl"))
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(output, "exports.jsonl"))
	if err != nil {
		return err
	}
	return save(filepath.Join(output, "manifest.json"), map[string]any{"schema": "gooo/fresh-composition-curriculum/v1", "status": "SOURCE_BOUND_EXPORTED", "runner_revision": revision, "native_main_revision": native, "native_binary_sha256": binarySHA, "sdk": "v0.2.11-experimental", "protocol_revision": protocolRevision, "protocol_sha256": hash(protocolRaw), "dataset_sha256": hash(dataset), "captures_sha256": hash(raw), "dataset_bytes": len(dataset), "capture_bytes": len(raw), "program_contract_groups": 1152, "bilingual_function_views": 2304, "native_export_calls": calls, "decision_rows": count, "decision_views_per_arm": 4608, "bilingual_decision_pairs_per_arm": 2304, "split_decision_rows": splits, "model_predictions": 0, "candidate_tests": 0, "selected_emissions": 0, "new_optimizer_updates": 0, "scope": "six compositions and four goals with parameter/language variants; independent full finite targets retain ties; no learned-quality or independent-intention claim"})
}

func main() {
	binary := flag.String("binary", "", "released native Gooo binary")
	evidence := flag.String("native-evidence", "", "public verified native main evidence")
	output := flag.String("output", "", "fresh collection directory")
	revision := flag.String("runner-revision", "", "clean committed runner source")
	flag.Parse()
	if flag.NArg() != 0 || *binary == "" || *evidence == "" || *output == "" || *revision == "" {
		fmt.Fprintln(os.Stderr, "binary, native-evidence, output and runner-revision required")
		os.Exit(2)
	}
	if err := collect(*binary, *evidence, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, "fresh-composition-curriculum:", err)
		os.Exit(1)
	}
	fmt.Println(`{"status":"SOURCE_BOUND_EXPORTED","native_export_calls":4608,"model_predictions":0}`)
}
