// package-path-model assembles or verifies one fixed synthetic-only public bundle.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

const repository = "asketeddy/gooo-typed-path-tiny-v1"
const schema = "gooo/public-typed-path-allowlist/v1"
const reviewedSchema = "gooo/public-typed-path-allowlist/v2"
const directSchema = "gooo/public-typed-path-allowlist/v3"
const preparedSchema = "gooo/public-typed-path-allowlist/v4"
const incrementalSchema = "gooo/public-typed-path-allowlist/v5"

var privateText = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)

type artifact struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema           string     `json:"schema"`
	Files            []artifact `json:"files"`
	TextScanned      bool       `json:"credentials_and_host_paths_scanned"`
	BinaryProvenance string     `json:"binary_provenance"`
}

func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func sources() map[string]string {
	files := map[string]string{
		"README.md": "docs/hf-typed-path-v1-model-card.md", "LICENSE": "LICENSE",
		"dataset/dataset.jsonl":            "data/typed-path-v1/dataset.jsonl",
		"dataset/uniform-manifest.json":    "data/typed-path-v1/manifest.json",
		"dataset/positioned-manifest.json": "data/typed-path-positioned-v2/manifest.json",
		"failure-first-training.json":      "runs/typed-path-v1-mps-20261001/failure.json",
		"probe/report.json":                "runs/typed-path-reserved-probe-tdd-20261001/report.json",
		"probe/preexecution.json":          "runs/typed-path-reserved-probe-tdd-20261001/preexecution.json",
		"probe/probe.jsonl":                "runs/typed-path-reserved-probe-tdd-20261001/probe.jsonl",
		"probe/process-metrics.json":       "runs/typed-path-reserved-probe-tdd-20261001/process-metrics.json",
	}
	studies := [][3]string{
		{"uniform-transfer", "runs/typed-path-v1-mps-20261001-v2", "runs/typed-path-v1-go-audit-20261001-v2"},
		{"positioned-transfer", "runs/typed-path-positioned-transfer-20261001", "runs/typed-path-positioned-transfer-audit-20261001"},
		{"positioned-random", "runs/typed-path-positioned-random-20261001", "runs/typed-path-positioned-random-audit-20261001"},
	}
	for _, study := range studies {
		for _, name := range []string{"report.json", "preexecution.json", "go-parity.json"} {
			files[study[0]+"/"+name] = study[1] + "/" + name
		}
		files[study[0]+"/go-audit.json"] = study[2] + "/report.json"
		files[study[0]+"/parity-audit.json"] = study[2] + "/parity-audit.json"
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, name := range []string{"model.json", "weights.bin"} {
				relative := "models/" + variant + "/" + name
				files[study[0]+"/"+relative] = study[1] + "/" + relative
			}
		}
	}
	return files
}

func reviewedSources() map[string]string {
	files := sources()
	for _, name := range []string{"report.json", "preexecution.json", "runner-binding.json"} {
		files["native-review/"+name] = "runs/typed-path-native-replay-bound-20261001/" + name
	}
	files["native-review/earlier-source-binding-correction.json"] = "runs/typed-path-native-replay-20261001/source-binding-correction.json"
	return files
}
func directSources() map[string]string {
	files := reviewedSources()
	root := "runs/native-typed-path-compound-20261001/"
	for _, name := range []string{"report.json", "preexecution.json", "independent-go-tests.jsonl"} {
		files["native-direct/"+name] = root + name
	}
	for _, language := range []string{"en", "ko"} {
		for _, arm := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
			for _, budget := range []int{4, 8} {
				for _, contract := range []string{"full", "partial"} {
					id := fmt.Sprintf("%s-%s-%d-%s", language, arm, budget, contract)
					for _, name := range []string{"input.gooo", "plan.json", "native-stdout.json", "generated.go.txt", "independent_test.go.txt"} {
						files["native-direct/"+id+"/"+name] = root + id + "/" + name
					}
				}
			}
		}
	}
	return files
}
func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 4<<20 {
		return nil, errors.New("invalid bounded regular public input")
	}
	return os.ReadFile(path)
}
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func assemble(output string) error {
	return assembleVersion(output, false)
}

func assembleVersion(output string, reviewed bool) error {
	return assembleVersions(output, reviewed, false)
}
func assembleVersions(output string, reviewed, direct bool) error {
	return assemblePublication(output, reviewed, direct, false)
}
func assemblePublication(output string, reviewed, direct, mainPromotion bool) error {
	return assembleEdition(output, reviewed, direct, mainPromotion, false)
}
func assembleEdition(output string, reviewed, direct, mainPromotion, prepared bool) error {
	return assembleIncrementalEdition(output, reviewed, direct, mainPromotion, prepared, false)
}
func assembleIncrementalEdition(output string, reviewed, direct, mainPromotion, prepared, incremental bool) error {
	prepared = prepared || incremental
	mainPromotion = mainPromotion || prepared
	direct = direct || mainPromotion
	reviewed = reviewed || direct
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("bundle output must be fresh")
	}
	contents := map[string][]byte{}
	value := manifest{Schema: schema, TextScanned: true, BinaryProvenance: "All nine weight files use the disclosed public synthetic bilingual Gooo/PROV-O curriculum; no private repository text or Laya weights were used. Binary files are validated as model tensors, not scanned as text."}
	files := sources()
	if reviewed {
		value.Schema = reviewedSchema
		files = reviewedSources()
	}
	if direct {
		value.Schema, files = directSchema, directSources()
	}
	if prepared {
		value.Schema, files = preparedSchema, preparedSources()
		if err := auditPreparedRepository(); err != nil {
			return err
		}
	}
	if incremental {
		value.Schema, files = incrementalSchema, incrementalSources()
		if err := auditIncrementalDirectory(incrementalRun, conditionalCohort, incrementalRun+"/captures"); err != nil {
			return err
		}
	}
	for name, source := range files {
		raw, err := read(source)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if !strings.HasSuffix(name, ".bin") && privateText.Match(raw) {
			return fmt.Errorf("private text in %s", name)
		}
		if reviewed && name == "README.md" {
			note, err := read("docs/hf-typed-path-v1-native-review.md")
			if err != nil {
				return err
			}
			if privateText.Match(note) {
				return errors.New("private review text")
			}
			raw = append(append(raw, '\n'), note...)
		}
		if direct && name == "README.md" {
			note, err := read("docs/hf-typed-path-v1-native-direct.md")
			if err != nil || privateText.Match(note) {
				return errors.New("invalid or private direct-inference text")
			}
			raw = append(append(raw, '\n'), note...)
		}
		if mainPromotion && name == "README.md" {
			note, err := read("docs/hf-typed-path-v1-main-promotion.md")
			if err != nil || privateText.Match(note) {
				return errors.New("invalid or private main-promotion text")
			}
			raw = append(append(raw, '\n'), note...)
		}
		if prepared && name == "README.md" {
			note, err := read("docs/hf-typed-path-v1-prepared-native.md")
			if err != nil || privateText.Match(note) {
				return errors.New("invalid or private prepared-native text")
			}
			raw = append(append(raw, '\n'), note...)
		}
		if incremental && name == "README.md" {
			note, err := read("docs/hf-typed-path-v1-incremental.md")
			if err != nil || privateText.Match(note) {
				return errors.New("invalid or private incremental text")
			}
			raw = append(append(raw, '\n'), note...)
		}
		if strings.HasSuffix(name, "/model.json") {
			if _, err := decision.LoadPath(source); err != nil {
				return err
			}
		}
		if strings.HasSuffix(name, "/go-audit.json") {
			var audit struct {
				Decision string `json:"decision"`
			}
			if json.Unmarshal(raw, &audit) != nil || audit.Decision != "PASS" {
				return errors.New("required numerical audit is incomplete")
			}
		}
		if name == "native-review/report.json" {
			var report struct {
				Status      string `json:"status"`
				Calls       int    `json:"native_compiler_calls"`
				Passed      int    `json:"independent_cases_passed"`
				Total       int    `json:"independent_cases_total"`
				Predictions int    `json:"additional_local_model_predictions"`
				Source      string `json:"runner_source_revision"`
			}
			if json.Unmarshal(raw, &report) != nil || report.Status != "PASS" || report.Calls != 20 || report.Passed != 240 || report.Total != 240 || report.Predictions != 0 || report.Source != "643ca6ead45cef35d85175864aa3b16556346bca" {
				return errors.New("required native replay is incomplete or incorrectly bound")
			}
		}
		if name == "native-direct/report.json" {
			if err := validateDirectEvidence(raw); err != nil {
				return err
			}
		}
		contents[name] = raw
		value.Files = append(value.Files, artifact{name, digest(raw), len(raw)})
	}
	sort.Slice(value.Files, func(i, j int) bool { return value.Files[i].Path < value.Files[j].Path })
	for name, raw := range contents {
		path := filepath.Join(output, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, raw, 0644); err != nil {
			return err
		}
	}
	return save(filepath.Join(output, "publication-manifest.json"), value)
}
func local(root string) ([]artifact, string, error) {
	raw, err := read(filepath.Join(root, "publication-manifest.json"))
	if err != nil {
		return nil, "", err
	}
	var value manifest
	if err := strictjson.Decode(raw, &value); err != nil {
		return nil, "", err
	}
	expected := sources()
	if value.Schema == reviewedSchema {
		expected = reviewedSources()
	}
	if value.Schema == directSchema {
		expected = directSources()
	}
	if value.Schema == preparedSchema {
		expected = preparedSources()
	}
	if value.Schema == incrementalSchema {
		expected = incrementalSources()
	}
	validSchema := value.Schema == schema || value.Schema == reviewedSchema || value.Schema == directSchema || value.Schema == preparedSchema || value.Schema == incrementalSchema
	if !validSchema || !value.TextScanned || value.BinaryProvenance == "" || len(value.Files) != len(expected) {
		return nil, "", errors.New("invalid fixed publication manifest")
	}
	seen := map[string]bool{}
	for _, file := range value.Files {
		if expected[file.Path] == "" || seen[file.Path] || file.Bytes <= 0 || file.Bytes > 4<<20 || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(file.SHA) {
			return nil, "", errors.New("unexpected fixed artifact")
		}
		seen[file.Path] = true
		content, err := read(filepath.Join(root, file.Path))
		if err != nil {
			return nil, "", err
		}
		if len(content) != file.Bytes || digest(content) != file.SHA {
			return nil, "", errors.New("local artifact digest mismatch")
		}
		if !strings.HasSuffix(file.Path, ".bin") && privateText.Match(content) {
			return nil, "", errors.New("private bundle text")
		}
		if strings.HasSuffix(file.Path, "/model.json") {
			if _, err := decision.LoadPath(filepath.Join(root, file.Path)); err != nil {
				return nil, "", err
			}
		}
	}
	seen["publication-manifest.json"] = true
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !seen[filepath.ToSlash(relative)] {
			return errors.New("extra public artifact")
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if value.Schema == preparedSchema || value.Schema == incrementalSchema {
		if err := auditPreparedDirectory(filepath.Join(root, "native-prepared"), filepath.Join(root, "conditional-cohort"), ""); err != nil {
			return nil, "", err
		}
	}
	if value.Schema == incrementalSchema {
		if err := auditIncrementalDirectory(filepath.Join(root, "native-incremental"), filepath.Join(root, "conditional-cohort"), ""); err != nil {
			return nil, "", err
		}
	}
	return append(value.Files, artifact{"publication-manifest.json", digest(raw), len(raw)}), digest(raw), nil
}
func fetch(ctx context.Context, client *http.Client, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anonymous GET status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 4<<20 {
		return nil, errors.New("public response too large")
	}
	return raw, nil
}
func verify(root, revision, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("verification output must be fresh")
	}
	files, manifestSHA, err := local(root)
	if err != nil {
		return err
	}
	receipt := map[string]any{"schema": "gooo/typed-path-public-verification/v1", "status": "PASS", "manifest_sha256": manifestSHA, "local_files_verified": len(files), "credentials_sent": false, "local_model_predictions": 0, "network_requests": 0}
	if revision != "" {
		if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
			return errors.New("immutable revision required")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 8 || req.URL.Scheme != "https" || req.URL.User != nil {
				return errors.New("unsafe redirect")
			}
			return nil
		}}
		raw, err := fetch(ctx, client, "https://huggingface.co/api/models/"+repository+"/revision/"+revision)
		if err != nil {
			return err
		}
		var info struct {
			SHA      string `json:"sha"`
			Private  bool   `json:"private"`
			Siblings []struct {
				Path string `json:"rfilename"`
			} `json:"siblings"`
		}
		if json.Unmarshal(raw, &info) != nil || info.SHA != revision || info.Private {
			return errors.New("public revision mismatch")
		}
		expected := map[string]bool{".gitattributes": true}
		for _, file := range files {
			expected[file.Path] = true
		}
		seen := map[string]bool{}
		for _, file := range info.Siblings {
			if !expected[file.Path] || seen[file.Path] {
				return errors.New("unexpected public repository artifact")
			}
			seen[file.Path] = true
		}
		for _, file := range files {
			if !seen[file.Path] {
				return errors.New("missing public artifact")
			}
		}
		jobs := make(chan artifact, len(files))
		results := make(chan error, len(files))
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
		for range 4 {
			go func() {
				for file := range jobs {
					raw, err := fetch(ctx, client, "https://huggingface.co/"+repository+"/resolve/"+revision+"/"+file.Path)
					if err == nil && (len(raw) != file.Bytes || digest(raw) != file.SHA) {
						err = errors.New("public digest mismatch")
					}
					results <- err
				}
			}()
		}
		var firstError error
		for range files {
			if err := <-results; err != nil && firstError == nil {
				firstError = err
				cancel()
			}
		}
		if firstError != nil {
			return firstError
		}
		receipt["repository"], receipt["commit_oid"], receipt["public_files_verified"], receipt["network_requests"] = repository, revision, len(files), len(files)+1
	}
	return save(output, receipt)
}
func main() {
	mode := flag.String("mode", "assemble", "assemble or verify")
	bundle := flag.String("bundle", "publication/hf-typed-path-v1", "fixed bundle directory")
	output := flag.String("output", "", "fresh bundle or verification output")
	revision := flag.String("revision", "", "optional immutable public revision")
	reviewed := flag.Bool("native-review", false, "include separately captured native structural replay")
	direct := flag.Bool("native-direct", false, "include fresh native structural inference and exact Go observations")
	mainPromotion := flag.Bool("main-promotion", false, "append verified native source main availability")
	prepared := flag.Bool("prepared-native", false, "include paired prepared native evidence and fixed bilingual cohort")
	incremental := flag.Bool("incremental-native", false, "include continued finite search and repeated-restart costs")
	flag.Parse()
	var err error
	if *output == "" || flag.NArg() != 0 {
		err = errors.New("fresh output required")
	} else if *mode == "assemble" {
		err = assembleIncrementalEdition(*output, *reviewed, *direct, *mainPromotion, *prepared, *incremental)
	} else if *mode == "verify" {
		err = verify(*bundle, *revision, *output)
	} else {
		err = errors.New("invalid mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "package-path-model:", err)
		os.Exit(1)
	}
	fmt.Println(`{"status":"PASS","local_model_predictions":0}`)
}
