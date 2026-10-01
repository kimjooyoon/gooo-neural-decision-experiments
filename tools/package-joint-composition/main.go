// package-joint-composition publishes only bounded own-model synthetic evidence.
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
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

const repository = "asketeddy/gooo-joint-path-tiny-v1"
const datasetSHA = "2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383"

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
			// This exact already-public collector source contains the detector
			// definition itself. A different file digest or any other matching
			// line still fails; raw evidence has no exemption.
			if string(s.Bytes()) == "var privatePattern = regexp.MustCompile(`"+sensitive.String()+"`)" {
				entry, e := hashFile(name)
				if e == nil && entry.SHA == "7c7f90ca403a72d9b3dfe908ec4bd81870716577efb8b7102069b18745232957" {
					continue
				}
			}
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

const curriculumRoot = "runs/joint-composition-curriculum-20261002"
const sdkRoot = "../gooo-decision-runtime"

func rawMembers(study, native string) []member {
	var result []member
	for _, split := range []string{"calibration", "development"} {
		for _, id := range []string{"independent-fp32", "independent-ptq_ternary", "independent-qat_ternary", "joint-fp32", "joint-ptq_ternary", "joint-qat_ternary", "offline"} {
			name := split + "-" + id + ".jsonl"
			result = append(result, member{"sdk/" + name, filepath.Join(study, name)})
		}
	}
	for _, name := range []string{"preexecution.json", "report.json", "selection.json", "independent-audit.json"} {
		result = append(result, member{"sdk/" + name, filepath.Join(study, name)})
	}
	for _, family := range []string{"assignment_reference", "operand_assignment", "branch_reference", "predicate_assignment", "schedule_operand", "schedule_branch"} {
		for goal := 0; goal < 4; goal++ {
			for _, language := range []string{"en", "ko"} {
				for _, policy := range []string{"selected", "independent", "joint", "offline"} {
					name := fmt.Sprintf("%s-goal%d-%s-%s", family, goal, language, policy)
					for _, suffix := range []string{".json", "-execution.json"} {
						result = append(result, member{"native/" + name + suffix, filepath.Join(native, name+suffix)})
					}
				}
			}
		}
	}
	for _, name := range []string{"preexecution.json", "report.json", "independent-audit.json"} {
		result = append(result, member{"native/" + name, filepath.Join(native, name)})
	}
	for _, name := range []string{"train_joint_composition_v1.py", "semantic_features_v3.py", "train_pilot_v2.py", "train_typed_path_v1.py"} {
		result = append(result, member{"source/training/" + name, "training/" + name})
	}
	for _, name := range []string{"fixture.go", "natural.go", "oracle.go", "fixture_test.go"} {
		result = append(result, member{"source/internal/jointcompositionstudy/" + name, "internal/jointcompositionstudy/" + name})
	}
	for _, name := range []string{"main.go", "evaluation.go", "models.go", "native.go", "audit.go", "native_audit.go", "native_audit_test.go", "main_test.go", "process_unix.go", "process_other.go", "process_unix_test.go"} {
		result = append(result, member{"source/tools/joint-composition-study/" + name, "tools/joint-composition-study/" + name})
	}
	for _, name := range []string{"manifest.json", "collection-attempt.json", "exports.jsonl", "preexecution.json", "dataset.jsonl", "audit.json"} {
		result = append(result, member{"curriculum/" + name, filepath.Join(curriculumRoot, name)})
	}
	for _, name := range []string{"main.go", "audit.go", "main_test.go"} {
		result = append(result, member{"source/tools/joint-composition-curriculum/" + name, "tools/joint-composition-curriculum/" + name})
	}
	for _, name := range runtimeOrigins() {
		result = append(result, member{"source/runtime/" + name, name})
	}
	result = append(result, member{"source/tools/extract-typed-sdk/main.go", "tools/extract-typed-sdk/main.go"}, member{"source/go.mod", "go.mod"}, member{"sdk-source-provenance.json", filepath.Join(sdkRoot, "source-provenance.json")})
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
		Status      string `json:"status"`
		Dataset     string `json:"dataset_sha256"`
		Calls       int    `json:"native_captures"`
		Executions  int    `json:"independently_executed_go_captures"`
		SDK         int    `json:"frozen_sdk_function_observations"`
		Predictions int    `json:"recorded_native_model_predictions"`
	}
	raw, err := os.ReadFile(filepath.Join(native, "independent-audit.json"))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &audit); err != nil || audit.Status != "PASS" || audit.Dataset != datasetSHA || audit.Calls != 192 || audit.Executions != 192 || audit.Predictions <= 0 {
		return errors.New("independent complete arithmetic/capture audit required")
	}
	var sdkAudit struct {
		Status  string `json:"status"`
		Dataset string `json:"dataset_sha256"`
		SDK     int    `json:"frozen_sdk_function_observations"`
		Calls   int    `json:"actual_model_predictions_all_stages"`
	}
	raw, err = os.ReadFile(filepath.Join(study, "independent-audit.json"))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &sdkAudit); err != nil || sdkAudit.Status != "PASS" || sdkAudit.Dataset != datasetSHA || sdkAudit.SDK != 5376 || sdkAudit.Calls != 25471 {
		return errors.New("independent complete SDK audit required")
	}
	provenancePin, err := hashFile(filepath.Join(sdkRoot, "source-provenance.json"))
	if err != nil || provenancePin.SHA != "230c72560143464a2df2cbf3ddf124123af9aff36e1d71faf5d0e489c18e4cc2" {
		return errors.New("exact released SDK.12 provenance required")
	}
	if _, err = os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh fixed publication output required")
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	files := map[string]string{"LICENSE": "LICENSE", "protocol.md": "docs/joint-path-composition-preregistration-20261002.md", "prefixture-amendment.md": "docs/joint-path-composition-prefixture-amendment-20261002.md",
		"results.md": "docs/joint-composition-model-results-20261002.md", "training-report.json": filepath.Join(models, "report.json"), "training-preexecution.json": filepath.Join(models, "preexecution.json"),
		"study-report.json": filepath.Join(study, "report.json"), "study-preexecution.json": filepath.Join(study, "preexecution.json"), "calibration-selection.json": filepath.Join(study, "selection.json"),
		"independent-audit.json": filepath.Join(study, "independent-audit.json"), "native-report.json": filepath.Join(native, "report.json"), "native-preexecution.json": filepath.Join(native, "preexecution.json"),
		"curriculum-manifest.json": "publication/joint-composition-curriculum-manifest-20261002.json", "curriculum-audit.json": "publication/joint-composition-curriculum-audit-20261002.json", "native-independent-audit.json": filepath.Join(native, "independent-audit.json"), "sdk-source-provenance.json": filepath.Join(sdkRoot, "source-provenance.json")}
	for _, arm := range []string{"independent", "joint"} {
		name := arm + "/go-parity.json"
		files[name] = filepath.Join(models, name)
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			name := arm + "/models/" + variant + "/model.json"
			if arm == "joint" {
				if _, e := jointdecision.Load(filepath.Join(models, name)); e != nil {
					return e
				}
			} else {
				if _, e := decision.LoadPath(filepath.Join(models, name)); e != nil {
					return e
				}
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
	provenance, err := modelProvenance(models, native, output)
	if err != nil {
		return err
	}
	for _, name := range provenance {
		files[name] = ""
	}
	card := modelCard(audit.Predictions)
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
	return save(filepath.Join(output, "publication-manifest.json"), map[string]any{"schema": "gooo/own-joint-composition-publication/v1", "repository": repository,
		"source_revision": revision, "files": payloads, "archive_members": members, "credentials_and_host_paths_scanned": true,
		"static_privacy_detector_source_sha256": "7c7f90ca403a72d9b3dfe908ec4bd81870716577efb8b7102069b18745232957",
		"dataset_sha256":                        datasetSHA, "optimizer_updates": 480, "model_exports": 6, "actual_native_calls": 192, "actual_native_predictions": audit.Predictions,
		"independently_compiled_go_executions": 192, "ordered_go_function_invocations": 3072, "default_model_promoted": false,
		"scope": "Six own random-init bounded structural models, all variants and negative comparisons; fixed allowlist contains public synthetic source/targets/captures only. Offline Python optimizer, Go runtime/oracle/orchestration/audit/publication."})
}
func main() {
	mode := flag.String("mode", "package", "stage, package, verify or fetch")
	bundle := flag.String("bundle", "", "local frozen public bundle for verification")
	publicRevision := flag.String("verify-revision", "", "immutable public Hugging Face revision")
	manifestSHA := flag.String("manifest-sha256", "", "immutable public manifest digest for fetch")
	evidence := flag.String("evidence", "", "fresh bounded raw evidence extraction directory for fetch")
	models := flag.String("models", "models/joint-composition-v1", "six public frozen models")
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
	case "stage":
		if *study == "" || *revision == "" || *native != "" {
			err = errors.New("complete SDK stage arguments required")
		} else {
			err = stageSDK(*models, *study, *output, *revision)
		}
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
	case "stage-verify":
		if *bundle == "" {
			err = errors.New("local intermediate bundle required")
		} else {
			err = verifyStage(*bundle, *publicRevision, *output)
		}
	case "fetch":
		if *bundle == "" || *manifestSHA == "" {
			err = errors.New("fresh bundle and pinned manifest digest required")
		} else {
			err = fetchBundle(*bundle, *publicRevision, *manifestSHA)
			if err == nil {
				err = verify(*bundle, *publicRevision, *output)
			}
			if err == nil && *evidence != "" {
				err = extractEvidence(*bundle, *evidence)
			}
		}
	default:
		err = errors.New("unknown publication mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
