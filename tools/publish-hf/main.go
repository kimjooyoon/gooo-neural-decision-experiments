// publish-hf uploads only an exact, digest-pinned public-export bundle.
// The default mode is a read-only preflight. A public write requires --publish.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	hubOrigin              = "https://huggingface.co"
	expectedModule         = "github.com/kimjooyoon/gooo-neural-decision-experiments"
	expectedNamespace      = "asketeddy"
	expectedRepoName       = "gooo-ir-operator-tiny-v1"
	allowlistSchema        = "gooo/public-export-allowlist/v1"
	verificationSchema     = "gooo/public-export-verification/v1"
	maxVerifierOutputBytes = 2 << 20
	maxRawBundleBytes      = 3 << 20
	maxCommitBodyBytes     = 4 << 20
	maxAPIResponseBytes    = 2 << 20
	maxFileBytes           = 64 << 20
)

var expectedBundleRoles = map[string]int{
	"dataset_manifest":      1,
	"synthetic_dataset":     1,
	"model_contract":        1,
	"generator_source":      1,
	"training_source":       1,
	"training_record":       1,
	"training_report":       1,
	"model_card":            1,
	"documentation":         1,
	"external_verification": 1,
	"python_parity":         1,
	"model_metadata":        3,
	"model_weights":         3,
}

var expectedBundlePaths = map[string]string{
	"dataset_manifest":      "data/synthetic-ops-v1/manifest.json",
	"synthetic_dataset":     "data/synthetic-ops-v1/dataset.jsonl",
	"model_contract":        "model-contract.json",
	"generator_source":      "cmd/dataset/main.go",
	"training_source":       "runs/pilot-mps-20260930-v1/training-source.py",
	"training_record":       "runs/pilot-mps-20260930-v1/preexecution.json",
	"training_report":       "runs/pilot-mps-20260930-v1/public-training-summary.json",
	"model_card":            "model-card.json",
	"documentation":         "README.md",
	"external_verification": "runs/pilot-mps-20260930-v1/go-audit.json",
	"python_parity":         "runs/pilot-mps-20260930-v1/go-parity.json",
	"model_metadata_fp32":   "runs/pilot-mps-20260930-v1/models/fp32/model.json",
	"model_weights_fp32":    "runs/pilot-mps-20260930-v1/models/fp32/weights.bin",
	"model_metadata_ptq":    "runs/pilot-mps-20260930-v1/models/ptq_ternary/model.json",
	"model_weights_ptq":     "runs/pilot-mps-20260930-v1/models/ptq_ternary/weights.bin",
	"model_metadata_qat":    "runs/pilot-mps-20260930-v1/models/qat_ternary/model.json",
	"model_weights_qat":     "runs/pilot-mps-20260930-v1/models/qat_ternary/weights.bin",
}

var expectedModelVariants = map[string]string{
	"runs/pilot-mps-20260930-v1/models/fp32/model.json":        "fp32",
	"runs/pilot-mps-20260930-v1/models/ptq_ternary/model.json": "ptq_ternary",
	"runs/pilot-mps-20260930-v1/models/qat_ternary/model.json": "qat_ternary",
}

type allowlist struct {
	Schema string            `json:"schema"`
	Files  []allowedArtifact `json:"files"`
}

type allowedArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Role   string `json:"role"`
}

type verification struct {
	Schema      string          `json:"schema"`
	Decision    string          `json:"decision"`
	FileCount   int             `json:"file_count"`
	TotalBytes  int64           `json:"total_bytes"`
	Roles       map[string]int  `json:"roles"`
	Files       []verifiedFile  `json:"files"`
	Models      []verifiedModel `json:"models"`
	Provenance  json.RawMessage `json:"provenance"`
	Limitations []string        `json:"limitations"`
}

type verifiedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	Role   string `json:"role"`
}

type verifiedModel struct {
	Path               string `json:"path"`
	Variant            string `json:"variant"`
	WeightsSHA256      string `json:"weights_sha256"`
	PackedFileBytes    int    `json:"packed_file_bytes"`
	DecodedWeightBytes int    `json:"decoded_weight_bytes"`
}

type bundleFile struct {
	Path    string
	SHA256  string
	Role    string
	Content []byte
}

type preflightResult struct {
	Schema            string          `json:"schema"`
	Mode              string          `json:"mode"`
	Decision          string          `json:"decision"`
	Repository        string          `json:"repository"`
	AuthenticatedUser string          `json:"authenticated_user"`
	FileCount         int             `json:"file_count"`
	TotalBytes        int64           `json:"total_bytes"`
	CommitBodyBytes   int64           `json:"commit_body_bytes"`
	RepositoryExists  bool            `json:"repository_exists"`
	Roles             map[string]int  `json:"roles"`
	Models            []verifiedModel `json:"models"`
	ParentCommit      string          `json:"parent_commit,omitempty"`
	Files             []verifiedFile  `json:"files"`
	Limitations       []string        `json:"limitations"`
}

type hfClient struct {
	api    *http.Client
	public *http.Client
	token  string
}

type whoAmI struct {
	Name string `json:"name"`
}

type refsResponse struct {
	Branches []repoRef `json:"branches"`
}

type repoRef struct {
	Name         string `json:"name"`
	Ref          string `json:"ref"`
	TargetCommit string `json:"targetCommit"`
}

type createRepoRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

type createRepoResponse struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	ID   string `json:"id"`
}

type commitHeader struct {
	Summary      string `json:"summary"`
	Description  string `json:"description,omitempty"`
	ParentCommit string `json:"parentCommit,omitempty"`
}

type commitFilePrefix struct {
	Key   string `json:"key"`
	Value struct {
		Path     string `json:"path"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	} `json:"value"`
}

type commitResponse struct {
	Success   bool   `json:"success"`
	CommitOID string `json:"commitOid"`
	CommitURL string `json:"commitUrl"`
}

type publishResult struct {
	Schema        string         `json:"schema"`
	Mode          string         `json:"mode"`
	Decision      string         `json:"decision"`
	Repository    string         `json:"repository"`
	CommitOID     string         `json:"commit_oid"`
	CommitURL     string         `json:"commit_url"`
	Created       bool           `json:"created"`
	UploadedFiles []verifiedFile `json:"uploaded_files"`
	PublicFiles   []verifiedFile `json:"public_files_verified"`
	Limitations   []string       `json:"limitations"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "publish-hf:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("publish-hf", flag.ContinueOnError)
	flags.SetOutput(stderr)
	bundleArg := flags.String("bundle", "", "exact export bundle directory")
	allowlistArg := flags.String("allowlist", "", "external digest-pinned allowlist JSON")
	publish := flags.Bool("publish", false, "create/update the fixed public Hugging Face model repository")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *bundleArg == "" || *allowlistArg == "" {
		return errors.New("usage: publish-hf --bundle DIR --allowlist FILE [--publish]")
	}

	root, err := findRepoRoot()
	if err != nil {
		return err
	}
	bundlePath, err := filepath.Abs(*bundleArg)
	if err != nil {
		return errors.New("could not resolve bundle path")
	}
	allowlistPath, err := filepath.Abs(*allowlistArg)
	if err != nil {
		return errors.New("could not resolve allowlist path")
	}
	if pathWithin(bundlePath, allowlistPath) {
		return errors.New("allowlist must be stored outside the bundle")
	}

	verified, err := runVerifier(root, bundlePath, allowlistPath)
	if err != nil {
		return err
	}
	if err := checkExpectedBundle(verified); err != nil {
		return err
	}
	files, totalBytes, err := loadAllowlistedBytes(bundlePath, allowlistPath, verified)
	if err != nil {
		return err
	}
	if totalBytes > maxRawBundleBytes {
		return fmt.Errorf("verified bundle exceeds the publisher's bounded upload size (%d-byte request limit)", maxCommitBodyBytes)
	}

	token, err := readToken()
	if err != nil {
		return err
	}
	client := newHFClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	user, err := client.whoAmI(ctx)
	if err != nil {
		return err
	}
	if user != expectedNamespace {
		return fmt.Errorf("authenticated Hugging Face account %q does not match the fixed target namespace %q", user, expectedNamespace)
	}

	parentCommit, exists, err := client.mainCommit(ctx)
	if err != nil {
		return err
	}
	if exists {
		if err := client.requirePublicRepo(ctx); err != nil {
			return err
		}
	}
	bodyBytes, err := commitBodySize(files, parentCommit)
	if err != nil {
		return err
	}
	preflight := preflightResult{
		Schema:            "gooo/hf-publish-preflight/v1",
		Mode:              "dry_run",
		Decision:          "READY_TO_PUBLISH",
		Repository:        expectedNamespace + "/" + expectedRepoName,
		AuthenticatedUser: user,
		FileCount:         verified.FileCount,
		TotalBytes:        totalBytes,
		CommitBodyBytes:   bodyBytes,
		RepositoryExists:  exists,
		Roles:             verified.Roles,
		Models:            verified.Models,
		ParentCommit:      parentCommit,
		Files:             verified.Files,
		Limitations: []string{
			"Dry-run mode performs authenticated read-only identity and repository checks; it does not create or modify a Hub repository.",
			"The export verifier checks the declared bundle and provenance receipts but cannot establish source-data rights or model quality by itself.",
		},
	}
	if !*publish {
		return writeJSON(stdout, preflight)
	}

	created := false
	if !exists {
		created, err = client.createPublicRepo(ctx)
		if err != nil {
			return err
		}
		if created {
			parentCommit, _, err = client.mainCommit(ctx)
			if err != nil {
				return fmt.Errorf("repository was created, but its main reference could not be read: %w", err)
			}
			if err := client.requirePublicRepo(ctx); err != nil {
				return fmt.Errorf("repository was created public, but its unauthenticated visibility check failed: %w", err)
			}
		} else {
			// A concurrent creator may have won after the initial refs check.
			parentCommit, exists, err = client.mainCommit(ctx)
			if err != nil || !exists {
				return errors.New("repository creation raced with another writer; no commit was attempted")
			}
		}
		if !created {
			if err := client.requirePublicRepo(ctx); err != nil {
				return err
			}
		}
	}

	if !validCommitOIDOrEmpty(parentCommit) {
		return errors.New("Hugging Face main reference returned an invalid parent commit")
	}
	commit, err := client.commit(ctx, files, parentCommit)
	if err != nil {
		if created {
			return fmt.Errorf("public repository creation succeeded, but the bundle commit failed or its outcome is unknown; do not retry without checking the Hub: %w", err)
		}
		return fmt.Errorf("bundle commit failed or its outcome is unknown; do not retry without checking the Hub: %w", err)
	}
	publicFiles, err := client.verifyPublicFiles(ctx, commit.CommitOID, verified.Files)
	if err != nil {
		return fmt.Errorf("commit %s succeeded, but unauthenticated public file verification failed: %w", commit.CommitOID, err)
	}
	result := publishResult{
		Schema:        "gooo/hf-publish-result/v1",
		Mode:          "publish",
		Decision:      "PUBLISHED_AND_PUBLICLY_VERIFIED",
		Repository:    expectedNamespace + "/" + expectedRepoName,
		CommitOID:     commit.CommitOID,
		CommitURL:     commit.CommitURL,
		Created:       created,
		UploadedFiles: verified.Files,
		PublicFiles:   publicFiles,
		Limitations: []string{
			"Public resolution checks confirm that each allowlisted file is readable without authentication at the reported commit and has the expected SHA-256.",
			"The publisher does not independently establish the legal provenance of synthetic data or the validity of upstream rights claims.",
		},
	}
	return writeJSON(stdout, result)
}

func checkExpectedBundle(result verification) error {
	if len(result.Roles) != len(expectedBundleRoles) {
		return errors.New("public-export verifier returned an unexpected role inventory")
	}
	for role, expectedCount := range expectedBundleRoles {
		if result.Roles[role] != expectedCount {
			return fmt.Errorf("verified export role %q has count %d; expected %d", role, result.Roles[role], expectedCount)
		}
	}
	if len(result.Files) != len(expectedBundlePaths) {
		return errors.New("public-export verifier returned an unexpected file inventory")
	}
	actual := make(map[string]string, len(result.Files))
	for _, file := range result.Files {
		key := file.Role
		if key == "model_metadata" || key == "model_weights" {
			variant := ""
			switch {
			case strings.Contains(file.Path, "/fp32/"):
				variant = "fp32"
			case strings.Contains(file.Path, "/ptq_ternary/"):
				variant = "ptq"
			case strings.Contains(file.Path, "/qat_ternary/"):
				variant = "qat"
			default:
				return fmt.Errorf("verified model artifact has an unsupported variant path %q", file.Path)
			}
			key += "_" + variant
		}
		if _, duplicate := actual[key]; duplicate {
			return fmt.Errorf("public-export verifier returned duplicate file role %q", key)
		}
		actual[key] = file.Path
	}
	if len(actual) != len(expectedBundlePaths) {
		return errors.New("public-export verifier returned an unexpected role/path inventory")
	}
	for role, expectedPath := range expectedBundlePaths {
		if actual[role] != expectedPath {
			return fmt.Errorf("verified export role %q must use fixed path %q", role, expectedPath)
		}
	}
	if len(result.Models) != len(expectedModelVariants) {
		return errors.New("public-export verifier returned an unexpected model count")
	}
	for _, model := range result.Models {
		if expectedVariant, ok := expectedModelVariants[model.Path]; !ok || model.Variant != expectedVariant {
			return fmt.Errorf("verified model metadata %q has an unexpected variant", model.Path)
		}
	}
	return nil
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.New("could not determine working directory")
	}
	current := cwd
	for {
		moduleRaw, readErr := os.ReadFile(filepath.Join(current, "go.mod"))
		if readErr == nil && strings.Contains(string(moduleRaw), "module "+expectedModule) {
			if _, err := os.Stat(filepath.Join(current, "tools", "verify-public-export", "main.go")); err == nil {
				return current, nil
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", errors.New("run publish-hf from this neural experiment repository")
}

func pathWithin(directory, candidate string) bool {
	rel, err := filepath.Rel(directory, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func runVerifier(root, bundlePath, allowlistPath string) (verification, error) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return verification{}, errors.New("Go is required to run the public-export verifier")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, goPath, "run", "./tools/verify-public-export", bundlePath, allowlistPath)
	command.Dir = root
	command.Env = childEnvironmentWithoutHubSecrets(os.Environ())
	stdout := &boundedBuffer{limit: maxVerifierOutputBytes}
	stderr := &boundedBuffer{limit: 64 << 10}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if stdout.overflow || stderr.overflow {
			return verification{}, errors.New("public-export verifier output exceeded its safety limit")
		}
		return verification{}, errors.New("public-export verifier rejected the bundle or failed to run")
	}
	if stdout.overflow || stderr.overflow {
		return verification{}, errors.New("public-export verifier output exceeded its safety limit")
	}
	var result verification
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil || ensureEOF(decoder) != nil {
		return verification{}, errors.New("public-export verifier returned an invalid receipt")
	}
	if result.Schema != verificationSchema || result.Decision != "PASS" || result.FileCount <= 0 || result.TotalBytes < 0 {
		return verification{}, errors.New("public-export verifier did not return PASS")
	}
	return result, nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	remaining := buffer.limit - buffer.Len()
	if remaining <= 0 {
		buffer.overflow = true
		return len(value), nil
	}
	if len(value) > remaining {
		buffer.overflow = true
		_, _ = buffer.Buffer.Write(value[:remaining])
		return len(value), nil
	}
	return buffer.Buffer.Write(value)
}

func childEnvironmentWithoutHubSecrets(environment []string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(name)
		if strings.Contains(upper, "TOKEN") || upper == "HF_HOME" || upper == "HF_ENDPOINT" || strings.HasPrefix(upper, "HUGGINGFACE_") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func loadAllowlistedBytes(bundlePath, allowlistPath string, checked verification) ([]bundleFile, int64, error) {
	policyInfo, err := os.Lstat(allowlistPath)
	if err != nil || !policyInfo.Mode().IsRegular() || policyInfo.Mode()&os.ModeSymlink != 0 || policyInfo.Size() <= 0 || policyInfo.Size() > 1<<20 {
		return nil, 0, errors.New("allowlist must be a regular, non-symlink file within the size limit")
	}
	policyRaw, err := os.ReadFile(allowlistPath)
	if err != nil {
		return nil, 0, errors.New("could not read allowlist")
	}
	var policy allowlist
	decoder := json.NewDecoder(bytes.NewReader(policyRaw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil || ensureEOF(decoder) != nil || policy.Schema != allowlistSchema || len(policy.Files) == 0 {
		return nil, 0, errors.New("allowlist has an invalid schema or structure")
	}
	verifiedByPath := make(map[string]verifiedFile, len(checked.Files))
	for _, entry := range checked.Files {
		if _, duplicate := verifiedByPath[entry.Path]; duplicate {
			return nil, 0, errors.New("verifier returned duplicate file paths")
		}
		verifiedByPath[entry.Path] = entry
	}
	if len(verifiedByPath) != len(policy.Files) || checked.FileCount != len(policy.Files) {
		return nil, 0, errors.New("allowlist and verifier file inventories differ")
	}

	entries := append([]allowedArtifact(nil), policy.Files...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	files := make([]bundleFile, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if !safeRelativePath(entry.Path) || entry.Role == "" || !validSHA256(entry.SHA256) {
			return nil, 0, errors.New("allowlist contains an invalid path, role, or digest")
		}
		verifiedEntry, ok := verifiedByPath[entry.Path]
		if !ok || verifiedEntry.SHA256 != entry.SHA256 || verifiedEntry.Role != entry.Role {
			return nil, 0, errors.New("allowlist entries differ from the verifier receipt")
		}
		content, err := readRegularBundleFile(bundlePath, entry.Path)
		if err != nil {
			return nil, 0, err
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != entry.SHA256 {
			return nil, 0, fmt.Errorf("bundle bytes changed after verification for %q", entry.Path)
		}
		total += int64(len(content))
		if total > maxRawBundleBytes {
			return nil, 0, errors.New("publisher raw bundle size exceeds its 3 MiB limit")
		}
		files = append(files, bundleFile{Path: entry.Path, SHA256: entry.SHA256, Role: entry.Role, Content: content})
	}
	if total != checked.TotalBytes {
		return nil, 0, errors.New("bundle total differs from the verifier receipt")
	}
	return files, total, nil
}

func readRegularBundleFile(bundlePath, relative string) ([]byte, error) {
	rootInfo, err := os.Lstat(bundlePath)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("bundle must be a non-symlink directory")
	}
	current := bundlePath
	parts := strings.Split(relative, "/")
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("allowlisted bundle path is missing or contains a symlink: %q", relative)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("allowlisted bundle path traverses a non-directory: %q", relative)
		}
		if index == len(parts)-1 && (!info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxFileBytes) {
			return nil, fmt.Errorf("allowlisted bundle entry is not a regular file within the size limit: %q", relative)
		}
	}
	file, err := os.Open(current)
	if err != nil {
		return nil, fmt.Errorf("could not open allowlisted bundle file %q", relative)
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil || int64(len(content)) > maxFileBytes {
		return nil, fmt.Errorf("could not read allowlisted bundle file within size limit: %q", relative)
	}
	return content, nil
}

func safeRelativePath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00") || path.IsAbs(value) || path.Clean(value) != value {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && strings.ToLower(value) == value
}

func commitBodySize(files []bundleFile, parentCommit string) (int64, error) {
	headerJSON, err := json.Marshal(struct {
		Key   string       `json:"key"`
		Value commitHeader `json:"value"`
	}{Key: "header", Value: commitHeader{Summary: "Publish verified Gooo operator model bundle", ParentCommit: parentCommit}})
	if err != nil {
		return 0, errors.New("could not encode commit header")
	}
	total := int64(len(headerJSON) + 1)
	for _, file := range files {
		pathJSON, err := json.Marshal(file.Path)
		if err != nil {
			return 0, errors.New("could not encode bundle path")
		}
		prefix := []byte(`{"key":"file","value":{"path":`)
		separator := []byte(`,"encoding":"base64","content":"`)
		suffix := []byte(`"}}`)
		total += int64(len(prefix) + len(pathJSON) + len(separator) + base64.StdEncoding.EncodedLen(len(file.Content)) + len(suffix) + 1)
		if total > maxCommitBodyBytes {
			return 0, errors.New("base64 NDJSON commit body exceeds the 4 MiB upload limit")
		}
	}
	return total, nil
}

func readToken() (string, error) {
	if token := strings.TrimSpace(os.Getenv("HF_TOKEN")); token != "" {
		if !validToken(token) {
			return "", errors.New("HF_TOKEN contains invalid whitespace or control characters")
		}
		return token, nil
	}
	var tokenPath string
	if configured := strings.TrimSpace(os.Getenv("HF_TOKEN_PATH")); configured != "" {
		tokenPath = configured
	} else if hfHome := strings.TrimSpace(os.Getenv("HF_HOME")); hfHome != "" {
		tokenPath = filepath.Join(hfHome, "token")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.New("could not find the Hugging Face token file")
		}
		tokenPath = filepath.Join(home, ".cache", "huggingface", "token")
	}
	info, err := os.Lstat(tokenPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 16<<10 {
		return "", errors.New("set HF_TOKEN or configure a regular Hugging Face token file")
	}
	raw, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", errors.New("could not read the Hugging Face token file")
	}
	token := strings.TrimSpace(string(raw))
	if !validToken(token) {
		return "", errors.New("Hugging Face token file contains invalid whitespace or control characters")
	}
	return token, nil
}

func validToken(token string) bool {
	if token == "" || len(token) > 16<<10 {
		return false
	}
	for _, r := range token {
		if r <= 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func newHFClient(token string) *hfClient {
	api := &http.Client{Timeout: 45 * time.Second}
	api.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	public := &http.Client{Timeout: 45 * time.Second}
	public.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 || request.URL.Scheme != "https" || request.URL.User != nil {
			return errors.New("refusing an unsafe public file redirect")
		}
		return nil
	}
	return &hfClient{api: api, public: public, token: token}
}

func (client *hfClient) whoAmI(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, hubOrigin+"/api/whoami-v2", nil)
	if err != nil {
		return "", errors.New("could not create identity request")
	}
	response, err := client.doAuthenticated(request)
	if err != nil {
		return "", fmt.Errorf("Hugging Face identity check failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", apiStatusError("identity check", response.StatusCode)
	}
	body, err := readBounded(response.Body, maxAPIResponseBytes)
	if err != nil {
		return "", errors.New("Hugging Face identity response exceeded its safety limit")
	}
	var result whoAmI
	if err := json.Unmarshal(body, &result); err != nil || result.Name == "" {
		return "", errors.New("Hugging Face identity response was invalid")
	}
	return result.Name, nil
}

func (client *hfClient) mainCommit(ctx context.Context) (string, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, hubOrigin+"/api/models/"+expectedNamespace+"/"+expectedRepoName+"/refs", nil)
	if err != nil {
		return "", false, errors.New("could not create repository reference request")
	}
	response, err := client.doAuthenticated(request)
	if err != nil {
		return "", false, fmt.Errorf("repository reference check failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if response.StatusCode != http.StatusOK {
		return "", false, apiStatusError("repository reference check", response.StatusCode)
	}
	body, err := readBounded(response.Body, maxAPIResponseBytes)
	if err != nil {
		return "", false, errors.New("Hugging Face refs response exceeded its safety limit")
	}
	var result refsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, errors.New("Hugging Face refs response was invalid")
	}
	mainOID := ""
	for _, branch := range result.Branches {
		if branch.Name == "main" || branch.Ref == "refs/heads/main" {
			if mainOID != "" {
				return "", false, errors.New("Hugging Face refs response contains multiple main branches")
			}
			mainOID = branch.TargetCommit
		}
	}
	if mainOID != "" && !validCommitOIDOrEmpty(mainOID) {
		return "", false, errors.New("Hugging Face main reference returned an invalid commit")
	}
	return mainOID, true, nil
}

func (client *hfClient) requirePublicRepo(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, hubOrigin+"/api/models/"+expectedNamespace+"/"+expectedRepoName, nil)
	if err != nil {
		return errors.New("could not create public repository check")
	}
	response, err := client.public.Do(request)
	if err != nil {
		return fmt.Errorf("public repository check failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return apiStatusError("unauthenticated repository visibility check", response.StatusCode)
	}
	_, err = readBounded(response.Body, maxAPIResponseBytes)
	if err != nil {
		return errors.New("public repository metadata exceeded its safety limit")
	}
	return nil
}

func (client *hfClient) createPublicRepo(ctx context.Context) (bool, error) {
	body, err := json.Marshal(createRepoRequest{Type: "model", Name: expectedRepoName, Private: false})
	if err != nil {
		return false, errors.New("could not encode repository creation request")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, hubOrigin+"/api/repos/create", bytes.NewReader(body))
	if err != nil {
		return false, errors.New("could not create repository creation request")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.doAuthenticated(request)
	if err != nil {
		return false, fmt.Errorf("repository creation request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict {
		return false, nil
	}
	if response.StatusCode != http.StatusOK {
		return false, apiStatusError("repository creation", response.StatusCode)
	}
	if _, err := readBounded(response.Body, maxAPIResponseBytes); err != nil {
		return false, errors.New("repository creation response exceeded its safety limit")
	}
	return true, nil
}

func (client *hfClient) commit(ctx context.Context, files []bundleFile, parentCommit string) (commitResponse, error) {
	bodySize, err := commitBodySize(files, parentCommit)
	if err != nil {
		return commitResponse{}, err
	}
	pipeReader, pipeWriter := io.Pipe()
	writerResult := make(chan error, 1)
	go func() {
		writerResult <- writeCommitNDJSON(pipeWriter, files, parentCommit)
	}()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, hubOrigin+"/api/models/"+expectedNamespace+"/"+expectedRepoName+"/commit/main", pipeReader)
	if err != nil {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		return commitResponse{}, errors.New("could not create commit request")
	}
	request.ContentLength = bodySize
	request.Header.Set("Content-Type", "application/x-ndjson")
	response, requestErr := client.doAuthenticated(request)
	_ = pipeReader.Close()
	if requestErr != nil {
		_ = pipeWriter.CloseWithError(requestErr)
		<-writerResult
		return commitResponse{}, fmt.Errorf("commit request failed; it was not retried: %w", requestErr)
	}
	defer response.Body.Close()
	writerErr := <-writerResult
	if writerErr != nil {
		return commitResponse{}, errors.New("could not stream the complete allowlisted NDJSON commit body")
	}
	if response.StatusCode != http.StatusOK {
		return commitResponse{}, apiStatusError("bundle commit", response.StatusCode)
	}
	body, err := readBounded(response.Body, maxAPIResponseBytes)
	if err != nil {
		return commitResponse{}, errors.New("commit response exceeded its safety limit")
	}
	var result commitResponse
	if err := json.Unmarshal(body, &result); err != nil || !result.Success || !validCommitOIDOrEmpty(result.CommitOID) || result.CommitOID == "" {
		return commitResponse{}, errors.New("Hugging Face did not confirm a successful commit")
	}
	return result, nil
}

func writeCommitNDJSON(writer *io.PipeWriter, files []bundleFile, parentCommit string) error {
	finish := func(err error) error {
		if err != nil {
			_ = writer.CloseWithError(err)
			return err
		}
		return writer.Close()
	}
	encoder := json.NewEncoder(writer)
	header := struct {
		Key   string       `json:"key"`
		Value commitHeader `json:"value"`
	}{Key: "header", Value: commitHeader{Summary: "Publish verified Gooo operator model bundle", ParentCommit: parentCommit}}
	if err := encoder.Encode(header); err != nil {
		return finish(err)
	}
	buffer := bufio.NewWriter(writer)
	for _, file := range files {
		pathJSON, err := json.Marshal(file.Path)
		if err != nil {
			return finish(err)
		}
		if _, err := buffer.WriteString(`{"key":"file","value":{"path":`); err != nil {
			return finish(err)
		}
		if _, err := buffer.Write(pathJSON); err != nil {
			return finish(err)
		}
		if _, err := buffer.WriteString(`,"encoding":"base64","content":"`); err != nil {
			return finish(err)
		}
		base64Writer := base64.NewEncoder(base64.StdEncoding, buffer)
		if _, err := base64Writer.Write(file.Content); err != nil {
			_ = base64Writer.Close()
			return finish(err)
		}
		if err := base64Writer.Close(); err != nil {
			return finish(err)
		}
		if _, err := buffer.WriteString(`"}}` + "\n"); err != nil {
			return finish(err)
		}
	}
	if err := buffer.Flush(); err != nil {
		return finish(err)
	}
	return finish(nil)
}

func (client *hfClient) verifyPublicFiles(ctx context.Context, commitOID string, expected []verifiedFile) ([]verifiedFile, error) {
	verified := make([]verifiedFile, 0, len(expected))
	for _, file := range expected {
		segments := strings.Split(file.Path, "/")
		for i := range segments {
			segments[i] = url.PathEscape(segments[i])
		}
		target := hubOrigin + "/" + expectedNamespace + "/" + expectedRepoName + "/resolve/" + commitOID + "/" + strings.Join(segments, "/")
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, errors.New("could not create public file verification request")
		}
		response, err := client.public.Do(request)
		if err != nil {
			return nil, fmt.Errorf("public file resolution failed for %q: %w", file.Path, err)
		}
		content, readErr := readBounded(response.Body, file.Bytes+1)
		_ = response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK || int64(len(content)) != file.Bytes {
			return nil, fmt.Errorf("public file resolution did not return the expected bytes for %q", file.Path)
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != file.SHA256 {
			return nil, fmt.Errorf("public file digest differs from the allowlist for %q", file.Path)
		}
		verified = append(verified, file)
	}
	return verified, nil
}

func (client *hfClient) doAuthenticated(request *http.Request) (*http.Response, error) {
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("User-Agent", "gooo-neural-decision-publisher/1.0")
	return client.api.Do(request)
}

func apiStatusError(operation string, status int) error {
	switch status {
	case http.StatusUnauthorized:
		return fmt.Errorf("Hugging Face %s returned HTTP 401; check the configured token", operation)
	case http.StatusForbidden:
		return fmt.Errorf("Hugging Face %s returned HTTP 403; the token may lack write permission or be restricted by fine-grained token policy", operation)
	case http.StatusNotFound:
		return fmt.Errorf("Hugging Face %s returned HTTP 404", operation)
	case http.StatusConflict:
		return fmt.Errorf("Hugging Face %s returned HTTP 409; the target changed during the operation", operation)
	default:
		return fmt.Errorf("Hugging Face %s returned HTTP %d", operation, status)
	}
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 0 || limit > maxRawBundleBytes+1 {
		if limit != maxAPIResponseBytes {
			return nil, errors.New("invalid response size limit")
		}
	}
	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, errors.New("response exceeded size limit")
	}
	return content, nil
}

func validCommitOIDOrEmpty(value string) bool {
	if value == "" {
		return true
	}
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return false
		}
	}
	return true
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON content")
		}
		return err
	}
	return nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return errors.New("could not write publisher result")
	}
	return nil
}
