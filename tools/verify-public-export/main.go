// verify-public-export checks a public model bundle against a separately
// reviewed, digest-pinned allowlist. It is a publication hygiene gate, not a
// proof that source material is public or that a model is correct.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const (
	allowlistSchema = "gooo/public-export-allowlist/v1"
	maxPolicyBytes  = 1 << 20
	maxFileBytes    = 64 << 20
	maxBundleBytes  = 256 << 20
)

type allowlist struct {
	Schema string            `json:"schema"`
	Files  []allowedArtifact `json:"files"`
}

type allowedArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Role   string `json:"role"`
	size   int64
}

type result struct {
	Schema      string           `json:"schema"`
	Decision    string           `json:"decision"`
	FileCount   int              `json:"file_count"`
	TotalBytes  int64            `json:"total_bytes"`
	Roles       map[string]int   `json:"roles"`
	Models      []modelResult    `json:"models"`
	Provenance  provenanceResult `json:"provenance"`
	Files       []fileResult     `json:"files"`
	Limitations []string         `json:"limitations"`
}

type modelResult struct {
	Path               string `json:"path"`
	Variant            string `json:"variant"`
	WeightsSHA256      string `json:"weights_sha256"`
	PackedFileBytes    int    `json:"packed_file_bytes"`
	DecodedWeightBytes int    `json:"decoded_weight_bytes"`
}

type fileResult struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	Role   string `json:"role"`
}

var sensitivePatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"private-key material", regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)},
	{"Hugging Face or source-control token", regexp.MustCompile(`\b(?:hf_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,})\b`)},
	{"API token", regexp.MustCompile(`\bsk-[A-Za-z0-9]{20,}\b`)},
	{"AWS access key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"bearer credential", regexp.MustCompile(`(?i)\bauthorization\s*:\s*bearer\s+[A-Za-z0-9._~+/=-]{20,}`)},
	{"credential assignment", regexp.MustCompile(`(?i)\b(?:password|secret|api[_-]?key|hf[_-]?token|github[_-]?token|aws[_-]?secret[_-]?access[_-]?key)\b\s*[:=]\s*["']?[A-Za-z0-9_./+=-]{8,}`)},
	{"local filesystem path", regexp.MustCompile(`(?:/Users/[^\s"'<>]+|/home/[^\s"'<>]+|/private/var/[^\s"'<>]+|[A-Za-z]:\\Users\\[^\s"'<>]+)`)},
	{"device or host identifier field", regexp.MustCompile(`(?i)["'](?:device_id|serial_number|machine_id|hardware_uuid|host_name|hostname)["']\s*:`)},
	{"MAC address", regexp.MustCompile(`(?i)\b(?:[0-9a-f]{2}:){5}[0-9a-f]{2}\b`)},
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: verify-public-export BUNDLE_DIR ALLOWLIST_JSON")
		os.Exit(2)
	}
	verified, err := verify(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "REJECTED: %v\n", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(verified); err != nil {
		fmt.Fprintln(os.Stderr, "could not encode verification result")
		os.Exit(1)
	}
}

func verify(bundleArgument, policyArgument string) (result, error) {
	bundle, err := filepath.Abs(bundleArgument)
	if err != nil {
		return result{}, errors.New("could not resolve bundle directory")
	}
	info, err := os.Lstat(bundle)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return result{}, errors.New("bundle must be an existing, non-symlink directory")
	}
	policyRaw, err := readPolicyFile(policyArgument)
	if err != nil {
		return result{}, err
	}
	if err := scanSensitive(policyRaw); err != nil {
		return result{}, fmt.Errorf("allowlist: %w", err)
	}
	if err := rejectDuplicateJSONKeys(policyRaw); err != nil {
		return result{}, fmt.Errorf("allowlist: %w", err)
	}
	var policy allowlist
	decoder := json.NewDecoder(bytes.NewReader(policyRaw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return result{}, fmt.Errorf("allowlist JSON: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return result{}, fmt.Errorf("allowlist JSON: %w", err)
	}
	if policy.Schema != allowlistSchema || len(policy.Files) == 0 || len(policy.Files) > 5000 {
		return result{}, errors.New("allowlist schema or file count is invalid")
	}
	entries, err := validateEntries(policy.Files)
	if err != nil {
		return result{}, err
	}
	if err := verifyTree(bundle, entries); err != nil {
		return result{}, err
	}
	verified := result{
		Schema:      "gooo/public-export-verification/v1",
		Decision:    "PASS",
		FileCount:   len(entries),
		Roles:       make(map[string]int),
		Limitations: []string{"An allowlist is an explicit reviewer decision; hashes and scanning cannot prove data provenance or license rights.", "Passing does not establish model quality, safety, or correctness."},
	}
	for name, entry := range entries {
		verified.Roles[entry.Role]++
		verified.TotalBytes += entry.size
		verified.Files = append(verified.Files, fileResult{Path: name, SHA256: entry.SHA256, Bytes: entry.size, Role: entry.Role})
	}
	sort.Slice(verified.Files, func(i, j int) bool { return verified.Files[i].Path < verified.Files[j].Path })
	models, err := verifyModels(bundle, entries)
	if err != nil {
		return result{}, err
	}
	verified.Models = models
	verified.Provenance, err = verifyTrainingProvenance(bundle, entries, models)
	if err != nil {
		return result{}, err
	}
	if verified.FileCount == 0 || verified.TotalBytes > maxBundleBytes {
		return result{}, errors.New("bundle size is outside the allowed range")
	}
	return verified, nil
}

func readPolicyFile(name string) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maxPolicyBytes {
		return nil, errors.New("allowlist must be a regular, non-symlink file within the size limit")
	}
	raw, err := os.ReadFile(name)
	if err != nil {
		return nil, errors.New("could not read allowlist")
	}
	return raw, nil
}

func validateEntries(files []allowedArtifact) (map[string]allowedArtifact, error) {
	entries := make(map[string]allowedArtifact, len(files))
	for _, entry := range files {
		if !safeRelativePath(entry.Path) {
			return nil, fmt.Errorf("allowlist contains an unsafe relative path: %q", entry.Path)
		}
		if _, exists := entries[entry.Path]; exists {
			return nil, fmt.Errorf("allowlist repeats path %q", entry.Path)
		}
		digest, err := hex.DecodeString(entry.SHA256)
		if err != nil || len(digest) != sha256.Size || strings.ToLower(entry.SHA256) != entry.SHA256 {
			return nil, fmt.Errorf("allowlist has an invalid lowercase SHA-256 for %q", entry.Path)
		}
		if !allowedRole(entry.Role) {
			return nil, fmt.Errorf("allowlist has an unsupported role for %q", entry.Path)
		}
		entries[entry.Path] = entry
	}
	return entries, nil
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

func allowedRole(value string) bool {
	switch value {
	case "documentation", "license", "source", "synthetic_dataset", "dataset_manifest",
		"model_contract", "generator_source", "training_source", "training_record",
		"training_report", "model_card", "external_verification", "python_parity", "model_metadata", "model_weights":
		return true
	default:
		return false
	}
}

func verifyTree(bundle string, entries map[string]allowedArtifact) error {
	allowedDirs := map[string]bool{".": true}
	for name := range entries {
		for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
			allowedDirs[dir] = true
		}
	}
	seen := make(map[string]bool, len(entries))
	var total int64
	err := filepath.WalkDir(bundle, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.New("could not inspect every bundle entry")
		}
		if filename == bundle {
			return nil
		}
		relative, err := filepath.Rel(bundle, filename)
		if err != nil {
			return errors.New("could not normalize bundle entry")
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("bundle contains a symlink: %q", relative)
		}
		if entry.IsDir() {
			if !allowedDirs[relative] {
				return fmt.Errorf("bundle contains an unlisted directory: %q", relative)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("bundle contains a non-regular file: %q", relative)
		}
		allowed, ok := entries[relative]
		if !ok {
			return fmt.Errorf("bundle contains an unlisted file: %q", relative)
		}
		fileInfo, err := entry.Info()
		if err != nil || fileInfo.Size() < 0 || fileInfo.Size() > maxFileBytes {
			return fmt.Errorf("bundle file is outside the size limit: %q", relative)
		}
		raw, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("could not read bundle file %q", relative)
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != allowed.SHA256 {
			return fmt.Errorf("bundle digest differs from allowlist: %q", relative)
		}
		if err := scanSensitive(raw); err != nil {
			return fmt.Errorf("bundle file %q: %w", relative, err)
		}
		if strings.HasSuffix(relative, ".json") {
			if err := rejectDuplicateJSONKeys(raw); err != nil {
				return fmt.Errorf("bundle JSON %q: %w", relative, err)
			}
		}
		seen[relative] = true
		allowed.size = fileInfo.Size()
		entries[relative] = allowed
		total += fileInfo.Size()
		if total > maxBundleBytes {
			return errors.New("bundle exceeds the total size limit")
		}
		return nil
	})
	if err != nil {
		return err
	}
	for name := range entries {
		if !seen[name] {
			return fmt.Errorf("allowlisted file is missing from bundle: %q", name)
		}
	}
	return nil
}

func scanSensitive(raw []byte) error {
	for _, pattern := range sensitivePatterns {
		if pattern.re.Find(raw) != nil {
			return fmt.Errorf("possible %s detected", pattern.name)
		}
	}
	return nil
}

func verifyModels(bundle string, entries map[string]allowedArtifact) ([]modelResult, error) {
	var models []modelResult
	pairedWeights := make(map[string]bool)
	expectedPaths := map[string]string{
		"runs/pilot-mps-20260930-v1/models/fp32/model.json":        "fp32",
		"runs/pilot-mps-20260930-v1/models/ptq_ternary/model.json": "ptq_ternary",
		"runs/pilot-mps-20260930-v1/models/qat_ternary/model.json": "qat_ternary",
	}
	for name, entry := range entries {
		if entry.Role != "model_metadata" {
			continue
		}
		expectedVariant, expected := expectedPaths[name]
		if path.Base(name) != "model.json" || !expected {
			return nil, fmt.Errorf("model metadata is outside the frozen v1 variant paths: %q", name)
		}
		metadataPath := filepath.Join(bundle, filepath.FromSlash(name))
		model, err := decision.Load(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("model metadata is invalid: %q: %w", name, err)
		}
		if model.Variant() != expectedVariant {
			return nil, fmt.Errorf("model metadata path and declared variant differ: %q", name)
		}
		raw, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("could not read model metadata: %q", name)
		}
		var artifact struct {
			WeightsFile string `json:"weights_file"`
		}
		if err := json.Unmarshal(raw, &artifact); err != nil {
			return nil, fmt.Errorf("model metadata has invalid JSON: %q", name)
		}
		weightName := path.Join(path.Dir(name), artifact.WeightsFile)
		weightEntry, ok := entries[weightName]
		if !ok || weightEntry.Role != "model_weights" {
			return nil, fmt.Errorf("model weights are not explicitly allowlisted: %q", weightName)
		}
		pairedWeights[weightName] = true
		models = append(models, modelResult{
			Path: name, Variant: model.Variant(), WeightsSHA256: model.WeightsSHA256(),
			PackedFileBytes: model.PackedFileBytes(), DecodedWeightBytes: model.DecodedTensorBytes(),
		})
	}
	if len(models) != len(expectedPaths) {
		return nil, errors.New("allowlist must contain exactly the three frozen v1 model variants")
	}
	for name, entry := range entries {
		if entry.Role == "model_weights" && !pairedWeights[name] {
			return nil, fmt.Errorf("unpaired model weight artifact: %q", name)
		}
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Path < models[j].Path })
	return models, nil
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON content")
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || duplicateFoldedKey(seen, name) {
				return errors.New("duplicate or invalid JSON object key")
			}
			seen[name] = true
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return errors.New("unexpected JSON delimiter")
	}
}

func duplicateFoldedKey(seen map[string]bool, name string) bool {
	for previous := range seen {
		if strings.EqualFold(previous, name) {
			return true
		}
	}
	return false
}

func ensureEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON content")
		}
		return err
	}
	return nil
}
