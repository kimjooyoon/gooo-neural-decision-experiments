// three-composition-curriculum freezes source-bound synthetic native exports.
// Models and optimizer code are never loaded by this preparation phase.
package main

import (
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

const protocol = "docs/own-three-choice-completeness-preregistration-20261002.md"
const protocolSHA = "5d840f7b3379339d684b85b7896a895538bd1a030395a096af3e52c5fc781cd3"
const protocolRevision = "d74a8a455ceed5949fcbad482375405b4704dc9a"
const sdkVersion = "v0.2.13-experimental"

func pin(binary, evidencePath, revision string) (string, string, error) {
	var evidence struct {
		Status string `json:"status"`
		Native string `json:"native_main_revision"`
		SDK    string `json:"sdk_version"`
	}
	evidenceInfo, err := os.Lstat(evidencePath)
	if err != nil || !evidenceInfo.Mode().IsRegular() || evidenceInfo.Size() > 64<<10 {
		return "", "", errors.New("bounded regular native evidence required")
	}
	raw, err := os.ReadFile(evidencePath)
	if err != nil || json.Unmarshal(raw, &evidence) != nil || evidence.Status != "NATIVE_MAIN_VERIFIED" ||
		evidence.SDK != sdkVersion || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(evidence.Native) {
		return "", "", errors.New("verified adopted native main evidence required")
	}
	binaryInfo, err := os.Lstat(binary)
	if err != nil || !binaryInfo.Mode().IsRegular() || binaryInfo.Size() > 64<<20 {
		return "", "", errors.New("bounded regular native executable required")
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return "", "", errors.New("native Go 1.27.1 binary required")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	sdk := ""
	for _, dependency := range info.Deps {
		if dependency.Path == "github.com/kimjooyoon/gooo-decision-runtime" && dependency.Replace == nil {
			sdk = dependency.Version
		}
	}
	if settings["vcs.revision"] != evidence.Native || settings["vcs.modified"] != "false" || sdk != sdkVersion {
		return "", "", errors.New("native source/SDK build pins differ")
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

func collect(binary, evidence, output, revision string) (resultErr error) {
	native, binarySHA, err := pin(binary, evidence, revision)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(protocol)
	if err != nil || hash(raw) != protocolSHA {
		return errors.New("frozen protocol differs")
	}
	committed, err := exec.Command("git", "show", protocolRevision+":"+protocol).Output()
	if err != nil || hash(committed) != protocolSHA {
		return errors.New("protocol was not committed at the frozen revision")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh collection directory required; never restart over evidence")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	preexecution := map[string]any{"schema": "gooo/three-composition-collection-preexecution/v1",
		"runner_revision": revision, "native_main_revision": native, "native_binary_sha256": binarySHA,
		"sdk_version": sdkVersion, "protocol_revision": protocolRevision, "protocol_sha256": protocolSHA,
		"planned_native_export_calls": 3072, "planned_function_views": 3072, "planned_source_input_rows": 9216,
		"raw_evidence_cap_bytes": rawCap, "individual_jsonl_line_cap_bytes": lineCap,
		"model_predictions": 0, "optimizer_updates": 0, "candidate_tests": 0, "selected_emissions": 0}
	if err = save(filepath.Join(output, "preexecution.json"), preexecution); err != nil {
		return err
	}
	var used int64
	journals := map[string]*journal{}
	for _, name := range []string{"fixtures.jsonl", "exports.jsonl", "dataset.jsonl"} {
		j, err := openJournal(filepath.Join(output, name), &used)
		if err != nil {
			for _, opened := range journals {
				opened.file.Close()
			}
			return err
		}
		journals[name] = j
	}
	calls, views := 0, 0
	defer func() {
		pins := map[string]any{}
		for name, j := range journals {
			if err := j.file.Close(); resultErr == nil && err != nil {
				resultErr = err
			}
			pins[name] = j.pin()
		}
		status := "COMPLETE_FIXED_COLLECTION"
		if resultErr != nil {
			status = "FAILED_PARTIAL_COLLECTION_RETAINED"
		}
		attempt := map[string]any{"schema": "gooo/three-composition-collection-attempt/v1", "status": status,
			"actual_native_export_calls": calls, "retained_function_views": views, "raw_bytes": used, "files": pins,
			"model_predictions": 0, "optimizer_updates": 0, "scope": "Native source exports, not model inference or native generative study."}
		if err := save(filepath.Join(output, "collection-attempt.json"), attempt); resultErr == nil && err != nil {
			resultErr = err
		}
	}()
	workspace, err := os.MkdirTemp("", "gooo-three-source-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	splits := map[string]int{}
	for _, family := range threecompositionstudy.Families {
		for config := range 24 {
			for goal := range 8 {
				for _, language := range [2]string{"en", "ko"} {
					doc, source, target, err := fixture(family, config, goal, language)
					if err != nil {
						return err
					}
					id := fmt.Sprintf("%s-c%02d-goal%d-%s", family, config, goal, language)
					if err := journals["fixtures.jsonl"].append(map[string]any{"id": id, "source": string(source),
						"document": doc, "finite_target": target, "source_sha256": hash(source)}); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(workspace, "source.gooo"), source, 0600); err != nil {
						return err
					}
					if err := save(filepath.Join(workspace, "plan.json"), doc); err != nil {
						return err
					}
					capture, wall, runErr := child(workspace, binary)
					calls++
					if !json.Valid(capture) || privateText.Match(capture) {
						// Preserve invalid raw bytes locally; a public bundle must reject this file.
						if used+int64(len(capture)) > rawCap {
							return errors.New("raw cap prevents failed-output storage; completed prefix preserved")
						}
						if err := os.WriteFile(filepath.Join(output, "failed-native-output.private.bin"), capture, 0600); err != nil {
							return err
						}
						used += int64(len(capture))
						return errors.New("invalid/private native output; completed prefix preserved")
					}
					if err := journals["exports.jsonl"].append(map[string]any{"id": id,
						"native_receipt": capture, "wall_ns": wall, "child_succeeded": runErr == nil}); err != nil {
						return err
					}
					if runErr != nil {
						return errors.New("native export failed; incomplete collection cannot train")
					}
					value, text, err := inspect(capture, doc, source)
					if err != nil {
						return err
					}
					row := datasetRow(id, family, config, goal, language, source, capture, value, text, target)
					if err := journals["dataset.jsonl"].append(row); err != nil {
						return err
					}
					views++
					splits[threecompositionstudy.Split(config)]++
					if calls%256 == 0 {
						fmt.Printf("{\"status\":\"COLLECTING\",\"native_export_calls\":%d,\"function_views\":%d}\n", calls, views)
					}
				}
			}
		}
	}
	if views != 3072 || calls != 3072 || splits["train"] != 2048 || splits["calibration"] != 512 || splits["development"] != 512 {
		return errors.New("fixed native collection denominator differs")
	}
	pins := map[string]any{}
	for name, j := range journals {
		if err := j.file.Sync(); err != nil {
			return err
		}
		pins[name] = j.pin()
	}
	return save(filepath.Join(output, "manifest.json"), map[string]any{"schema": "gooo/three-composition-curriculum/v1",
		"status": "SOURCE_BOUND_EXPORTED_PENDING_INDEPENDENT_AUDIT", "runner_revision": revision,
		"native_main_revision": native, "native_binary_sha256": binarySHA, "sdk_version": sdkVersion,
		"protocol_revision": protocolRevision, "protocol_sha256": protocolSHA, "files": pins, "raw_bytes": used,
		"authored_ways": 8, "bilingual_contract_groups": 1536, "function_views": views, "source_input_rows": views * 3,
		"split_function_views": splits, "actual_native_export_calls": calls, "model_predictions": 0,
		"optimizer_updates": 0, "training_authorized_by_this_manifest": false,
		"scope": "Concrete source, cases, plans, complete native inputs and passing-set hashes frozen before model observation."})
}

func main() {
	binary := flag.String("binary", "", "adopted native Gooo executable")
	evidence := flag.String("native-evidence", "", "verified native main receipt")
	output := flag.String("output", "", "fresh collection directory")
	revision := flag.String("runner-revision", "", "clean committed source revision")
	auditDir := flag.String("audit", "", "existing collection; no new native/model calls")
	auditReport := flag.String("audit-report", "", "fresh independent report path")
	flag.Parse()
	if *auditDir != "" {
		if *auditReport == "" || *binary != "" || *evidence != "" || *output != "" || *revision != "" || flag.NArg() != 0 {
			fmt.Fprintln(os.Stderr, "audit requires only audit and audit-report")
			os.Exit(2)
		}
		if err := audit(*auditDir, *auditReport); err != nil {
			fmt.Fprintln(os.Stderr, "three-composition-curriculum audit:", err)
			os.Exit(1)
		}
		return
	}
	if *binary == "" || *evidence == "" || *output == "" || *revision == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "required: binary, native-evidence, output, runner-revision")
		os.Exit(2)
	}
	absBinary, err := filepath.Abs(*binary)
	if err == nil {
		err = collect(absBinary, *evidence, *output, *revision)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "three-composition-curriculum:", err)
		os.Exit(1)
	}
}

func datasetRow(id, family string, config, goal int, language string, source, capture []byte,
	value export, text string, target threecompositionstudy.FiniteTarget) map[string]any {
	return map[string]any{"id": id, "program_contract_group": fmt.Sprintf("%s-c%02d-goal%d", family, config, goal),
		"family": family, "configuration": config, "desired_mask": goal, "language": language,
		"split": threecompositionstudy.Split(config), "source_sha256": hash(source),
		"document_sha256": value.Document, "capture_sha256": hash(capture), "inputs": value.Inputs,
		"complete_three_input": text, "input_sha256": hash([]byte(text)), "full_contract_target": target}
}
