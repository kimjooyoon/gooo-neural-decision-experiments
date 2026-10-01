// package-fresh-composition publishes only bounded own-model synthetic evidence.
package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const repository = "asketeddy/gooo-semantic-composition-tiny-v1"
const datasetSHA = "44393fb6ae27d1dc75c9706ea50077fab8e2573928e4f59422e3e3a65aa79620"

var sensitive = regexp.MustCompile(`/Users/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

type artifact struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}
type member struct{ Public, Local string }

func hashFile(name string) (artifact, error) {
	info, err := os.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 32<<20 {
		return artifact{}, errors.New("bounded regular artifact required")
	}
	f, err := os.Open(name)
	if err != nil {
		return artifact{}, err
	}
	defer f.Close()
	h := sha256.New()
	var buffer [32768]byte
	n, err := io.CopyBuffer(h, f, buffer[:])
	if err != nil {
		return artifact{}, err
	}
	return artifact{SHA: hex.EncodeToString(h.Sum(nil)), Bytes: n}, nil
}
func scan(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 1<<20)
	for s.Scan() {
		if sensitive.Match(s.Bytes()) {
			return errors.New("sensitive text in publication input")
		}
	}
	return s.Err()
}
func save(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}
func copyFile(from, to string) error {
	if err := scan(from); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
		return err
	}
	raw, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, raw, 0644)
}
func archive(out string, members []member) ([]artifact, error) {
	f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	z := zip.NewWriter(f)
	var entries []artifact
	seen := map[string]bool{}
	for _, m := range members {
		if m.Public == "" || strings.Contains(m.Public, "..") || strings.HasPrefix(m.Public, "/") || seen[m.Public] {
			return nil, errors.New("unsafe or duplicate archive path")
		}
		seen[m.Public] = true
		entry, e := hashFile(m.Local)
		if e != nil {
			return nil, e
		}
		if e = scan(m.Local); e != nil {
			return nil, e
		}
		header := &zip.FileHeader{Name: m.Public, Method: zip.Deflate}
		header.SetMode(0644)
		w, e := z.CreateHeader(header)
		if e != nil {
			return nil, e
		}
		r, e := os.Open(m.Local)
		if e != nil {
			return nil, e
		}
		var buffer [32768]byte
		_, e = io.CopyBuffer(w, r, buffer[:])
		closeErr := r.Close()
		if e != nil {
			return nil, e
		}
		if closeErr != nil {
			return nil, closeErr
		}
		entry.Path = m.Public
		entries = append(entries, entry)
	}
	if err = z.Close(); err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	return entries, nil
}
func rawMembers(study, native string) []member {
	var result []member
	for _, split := range []string{"calibration", "development"} {
		for _, id := range []string{"v2-fp32", "v2-ptq_ternary", "v2-qat_ternary", "v3-fp32", "v3-ptq_ternary", "v3-qat_ternary", "offline"} {
			name := split + "-" + id + ".jsonl"
			result = append(result, member{"sdk/" + name, filepath.Join(study, name)})
		}
	}
	for _, name := range []string{"preexecution.json", "report.json", "selection.json", "independent-audit.json"} {
		result = append(result, member{"sdk/" + name, filepath.Join(study, name)})
	}
	for _, family := range []string{"reference_operand", "assignment_branch", "predicate_branch", "assignment_schedule", "reference_schedule", "branch_schedule"} {
		for goal := 0; goal < 4; goal++ {
			for _, language := range []string{"en", "ko"} {
				for _, policy := range []string{"selected", "reference", "offline"} {
					name := fmt.Sprintf("%s-goal%d-%s-%s", family, goal, language, policy)
					for _, suffix := range []string{".json", "-execution.json"} {
						result = append(result, member{"native/" + name + suffix, filepath.Join(native, name+suffix)})
					}
				}
			}
		}
	}
	for _, name := range []string{"preexecution.json", "report.json"} {
		result = append(result, member{"native/" + name, filepath.Join(native, name)})
	}
	for _, name := range []string{"train_fresh_composition_v1.py", "semantic_features_v3.py", "split_features_v2.py", "train_bilingual_judgment_v1.py", "train_pilot_v2.py", "train_typed_path_v1.py"} {
		result = append(result, member{"source/training/" + name, "training/" + name})
	}
	for _, name := range []string{"fixture.go", "natural.go", "oracle.go", "fixture_test.go"} {
		result = append(result, member{"source/internal/compositionstudy/" + name, "internal/compositionstudy/" + name})
	}
	for _, name := range []string{"main.go", "evaluation.go", "models.go", "native.go", "audit.go", "main_test.go", "process_unix.go", "process_other.go", "process_unix_test.go"} {
		result = append(result, member{"source/tools/fresh-composition-study/" + name, "tools/fresh-composition-study/" + name})
	}
	return result
}
func packageBundle(models, study, native, output, revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact source revision required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean committed source required")
	}
	var audit struct {
		Status     string `json:"status"`
		Dataset    string `json:"dataset_sha256"`
		Calls      int    `json:"native_captures"`
		Executions int    `json:"independently_executed_go_captures"`
		SDK        int    `json:"frozen_sdk_function_observations"`
	}
	raw, err := os.ReadFile(filepath.Join(study, "independent-audit.json"))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &audit); err != nil || audit.Status != "PASS" || audit.Dataset != datasetSHA || audit.Calls != 144 || audit.Executions != 144 || audit.SDK != 5376 {
		return errors.New("independent complete arithmetic/capture audit required")
	}
	if _, err = os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh fixed publication output required")
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	files := map[string]string{"LICENSE": "LICENSE", "protocol.md": "docs/fresh-composition-semantic-v3-preregistration-20261002.md",
		"results.md": "docs/fresh-composition-model-results-20261002.md", "training-report.json": filepath.Join(models, "report.json"), "training-preexecution.json": filepath.Join(models, "preexecution.json"),
		"study-report.json": filepath.Join(study, "report.json"), "study-preexecution.json": filepath.Join(study, "preexecution.json"), "calibration-selection.json": filepath.Join(study, "selection.json"),
		"independent-audit.json": filepath.Join(study, "independent-audit.json"), "native-report.json": filepath.Join(native, "report.json"), "native-preexecution.json": filepath.Join(native, "preexecution.json"),
		"curriculum-manifest.json": "publication/fresh-composition-manifest-20261002.json", "curriculum-audit.json": "publication/fresh-composition-audit-20261002.json"}
	for _, arm := range []string{"v2", "v3"} {
		name := arm + "/go-parity.json"
		files[name] = filepath.Join(models, name)
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			name := arm + "/models/" + variant + "/model.json"
			if _, e := decision.LoadPath(filepath.Join(models, name)); e != nil {
				return e
			}
			files[name] = filepath.Join(models, name)
			name = arm + "/models/" + variant + "/weights.bin"
			files[name] = filepath.Join(models, name)
		}
	}
	for name, source := range files {
		target := filepath.Join(output, name)
		if strings.HasSuffix(name, "weights.bin") {
			if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			raw, e := os.ReadFile(source)
			if e != nil {
				return e
			}
			err = os.WriteFile(target, raw, 0644)
		} else {
			err = copyFile(source, target)
		}
		if err != nil {
			return err
		}
	}
	provenance, err := modelProvenance(models, output)
	if err != nil {
		return err
	}
	for _, name := range provenance {
		files[name] = ""
	}
	card := `---
license: mit
language: [en, ko]
tags: [gooo, metaprogramming, ternary, from-scratch, go]
---
# Own Gooo semantic composition tiny v1

Experimental 12,728-parameter models initialized from our own random state.
No Laya/pretrained/earlier own-model weights. They rank legal compiler-owned
Gooo paths from typed source plus complete Korean/English intent; they are not
unrestricted text code generators. Optional disconnected generation is deterministic.

Six v2/v3 FP32/PTQ/QAT exports retain both improvements and regressions.
Calibration selected v3 QAT; development v3 FP32 needs fewer extra attempts.
All 384 development finite contracts finish within four candidates in each policy.
Warm selected inference: about 9.7 microseconds, zero per-call heap allocations.
Ternary weights: 2,759 disk bytes, decoded int8 tensors 12,896 bytes plus scales;
1,248-byte workspace. Whole-process RAM and packed arithmetic are separate.

Actual Gooo main dogfood: 144 calls, 370 predictions, 144 independently compiled
Go executions and 2,304 function invocations. Every full 16-case contract passes.
Read results.md for language disagreement, retained capture/counter errors,
complete-function curves and timing/CPU/RSS measurement scope.

Use released Gooo main with SDK v0.2.11 and an explicit path model:

    gooo body-codegen --json --path-plan plan.json --path-model v3/models/qat_ternary/model.json --path-step-attempts 1 --path-feedback-rounds 3 --path-feedback-unfixed --activity ChoosePath source.gooo

The original source and typed plan must bind; the compiler creates the source
context. Omit the path model for deterministic continuation. Initial inputs
exclude test outcomes; later actual failures may rank remaining paths only.

protocol.md was frozen before program/oracle implementation and training.
raw-evidence.zip preserves SDK/native raw captures, actual-Go values and source.
Only deliberately public synthetic evidence is included; publication-manifest.json
lists exact payload/archive hashes. Native bilingual counter correction is in
independent-audit.json; old raw reports are preserved.
provenance/*.ttl links each model's weights and metadata to its training/export
activity, split roles, own random initialization and compiler source using PROV-O.

Source/trainer/runtime: https://github.com/kimjooyoon/gooo-neural-decision-experiments
SDK: https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.11-experimental
Compiler main: https://github.com/kimjooyoon/meta-ontology-go/commit/f4813dc6251037767c8cff7295ccfdab2b044ff2
Public source-bound curriculum, including rejected first capture:
https://huggingface.co/asketeddy/gooo-compiler-context-tiny-v1/tree/3537f6d3f77ec163fe478d4aef440baee17b9121/research/fresh-composition-curriculum-20261002

These are six compositions with goal, parameter and language variants, one
matched training seed and four enumerable candidates; not broad Gooo language
understanding or universal semantic correctness. No default model promotion.
`
	if err = os.WriteFile(filepath.Join(output, "README.md"), []byte(card), 0644); err != nil {
		return err
	}
	files["README.md"] = ""
	members, err := archive(filepath.Join(output, "raw-evidence.zip"), rawMembers(study, native))
	if err != nil {
		return err
	}
	files["raw-evidence.zip"] = ""
	var payloads []artifact
	for name := range files {
		entry, e := hashFile(filepath.Join(output, name))
		if e != nil {
			return e
		}
		entry.Path = name
		payloads = append(payloads, entry)
	}
	sort.Slice(payloads, func(i, j int) bool { return payloads[i].Path < payloads[j].Path })
	return save(filepath.Join(output, "publication-manifest.json"), map[string]any{"schema": "gooo/own-semantic-composition-publication/v1", "repository": repository,
		"source_revision": revision, "files": payloads, "archive_members": members, "credentials_and_host_paths_scanned": true,
		"dataset_sha256": datasetSHA, "optimizer_updates": 960, "model_exports": 6, "actual_native_calls": 144, "actual_native_predictions": 370,
		"independently_compiled_go_executions": 144, "ordered_go_function_invocations": 2304, "default_model_promoted": false,
		"scope": "Six own random-init bounded structural models, all variants and negative comparisons; fixed allowlist contains public synthetic source/targets/captures only. Offline Python optimizer, Go runtime/oracle/orchestration/audit/publication."})
}
func main() {
	mode := flag.String("mode", "package", "package or verify")
	bundle := flag.String("bundle", "", "local frozen public bundle for verification")
	publicRevision := flag.String("verify-revision", "", "immutable public Hugging Face revision")
	models := flag.String("models", "models/fresh-composition-v1", "six public frozen models")
	study := flag.String("study", "", "SDK raw observation directory")
	native := flag.String("native", "", "native raw capture directory")
	output := flag.String("output", "", "fresh fixed public bundle")
	revision := flag.String("source-revision", "", "exact committed producer")
	flag.Parse()
	if flag.NArg() != 0 || *output == "" {
		fmt.Fprintln(os.Stderr, "complete fixed publication arguments required")
		os.Exit(1)
	}
	var err error
	switch *mode {
	case "package":
		if *study == "" || *native == "" || *revision == "" {
			err = errors.New("complete fixed source/evidence paths required")
		} else {
			err = packageBundle(*models, *study, *native, *output, *revision)
		}
	case "verify":
		if *bundle == "" {
			err = errors.New("local frozen bundle required")
		} else {
			err = verify(*bundle, *publicRevision, *output)
		}
	default:
		err = errors.New("unknown publication mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
