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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	runPath = "runs/native-integer-local-6a8011bf-20260930"
	outPath = "review/body-plan-publication-v1-20260930/native-integer"
)

var files = []string{
	"README.md", "native-result-before-fix.json", "native-result-source-bound.json", "native-result.json",
}

var knownKeys = words(`FAIL_CLOSED PASS PROGRESS UNKNOWN activity activity_id aggregate_completeness_score allowed_investment boundary candidate_routes compiler_source_sha completeness_percent completeness_receipt core_dimensions decision decision_basis denominator deterministic_replay dimensions domain_scope equivalence_rule equivalent error evidence excluded_effects excluded_scope execution_environment fail_closed_reason fallback_reason final_route first_unresolved generated_digest generated_semantic_digest human_actions id input_type laya_mode laya_provider lowered_constructs lowered_semantic_units maximum_human_actions maximum_repository_writes method mode next_operation non_authorizing non_executing not_claimed numerator observed_system_cost output_type plan_sha256 profile_id program_digest proposed_route provider reason replay_digest report repository_writes request_sha256 route route_decision route_decision_latency_ms route_equivalence route_selection rule schema scope selected source source_constructs source_digest source_semantic_digest source_semantic_units status status_counts system_budget toolchain typecheck_passed unit unresolved_claims unsupported_constructs`)

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(/Users/|/home/|/private/|/var/folders/|/tmp/|[A-Z]:\\Users\\|/opt/homebrew/|/Applications/)`),
	regexp.MustCompile(`(?i)\b(?:Bearer\s+[A-Za-z0-9._~+/=-]+|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|hf_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9]{20,}|AKIA[A-Z0-9]{16}|AIza[0-9A-Za-z_-]{35}|xox[baprs]-[A-Za-z0-9-]{10,})\b`),
	regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`),
	regexp.MustCompile(`(?i)\b(?:host|machine|session)[_-]?id\b`),
	regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`),
}

type entry struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}

type result struct {
	Schema              string   `json:"schema"`
	AuditedUTC          string   `json:"audited_utc"`
	Decision            string   `json:"decision"`
	RunPath             string   `json:"run_path"`
	ExpectedFiles       int      `json:"expected_files"`
	ObservedFiles       int      `json:"observed_files"`
	TotalBytes          int64    `json:"total_bytes"`
	ManifestFile        string   `json:"manifest_file"`
	ManifestSHA256      string   `json:"manifest_sha256"`
	AuditorSourceSHA256 string   `json:"auditor_source_sha256"`
	JSONFiles           int      `json:"json_files"`
	PrivacyMatches      int      `json:"privacy_matches"`
	UnknownJSONKeys     int      `json:"unknown_json_keys"`
	DuplicateJSONKeys   int      `json:"duplicate_json_keys"`
	BinaryFiles         int      `json:"binary_files"`
	UnexpectedPaths     int      `json:"unexpected_paths"`
	Findings            []string `json:"findings"`
	Notes               []string `json:"notes"`
}

func main() {
	root, err := filepath.Abs(runPath)
	must(err)
	allow := map[string]bool{}
	for _, name := range files {
		allow[name] = true
	}

	actual := map[string]bool{}
	findings := make([]string, 0)
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if info.IsDir() {
			findings = append(findings, "unexpected directory: "+path)
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		actual[rel] = true
		if !allow[rel] || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			findings = append(findings, "unexpected file path or file type: "+rel)
		}
		return nil
	})
	must(err)
	for _, name := range files {
		if !actual[name] {
			findings = append(findings, "missing allowlisted file: "+name)
		}
	}

	paths := append([]string(nil), files...)
	sort.Strings(paths)
	manifest := make([]entry, 0, len(paths))
	var bytesTotal int64
	privacy, unknown, duplicates, binary, jsonFiles := 0, 0, 0, 0, 0
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			findings = append(findings, "unreadable file: "+rel)
			continue
		}
		if !utf8.Valid(data) || hasBinaryControl(data) {
			binary++
			findings = append(findings, "non-text file: "+rel)
		}
		privacy += scanSensitive(data)
		sum := sha256.Sum256(data)
		manifest = append(manifest, entry{Path: rel, Bytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:]), Kind: "UTF-8 text"})
		bytesTotal += int64(len(data))
		if strings.HasSuffix(rel, ".json") {
			jsonFiles++
			values, extra, repeated, err := scanJSON(data)
			if err != nil || values != 1 {
				findings = append(findings, "invalid JSON: "+rel)
			}
			unknown += extra
			duplicates += repeated
		}
	}
	if privacy != 0 {
		findings = append(findings, fmt.Sprintf("sensitive-content-pattern matches: %d", privacy))
	}
	if unknown != 0 {
		findings = append(findings, fmt.Sprintf("unknown JSON key occurrences: %d", unknown))
	}
	if duplicates != 0 {
		findings = append(findings, fmt.Sprintf("duplicate JSON key occurrences: %d", duplicates))
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	must(err)
	manifestBytes = append(manifestBytes, '\n')
	manifestSum := sha256.Sum256(manifestBytes)
	auditorBytes, err := os.ReadFile("review/body-plan-publication-v1-20260930/tools/audit-native-integer/main.go")
	must(err)
	auditorSum := sha256.Sum256(auditorBytes)
	must(os.MkdirAll(outPath, 0o755))
	manifestFile := filepath.Join(outPath, "file-manifest.json")
	must(os.WriteFile(manifestFile, manifestBytes, 0o644))

	decision := "PASS"
	if len(findings) != 0 || len(actual) != len(allow) {
		decision = "FAIL"
	}
	r := result{
		Schema: "gooo/native-integer-local-privacy-audit/v1", AuditedUTC: time.Now().UTC().Format(time.RFC3339Nano),
		Decision: decision, RunPath: runPath, ExpectedFiles: len(allow), ObservedFiles: len(actual), TotalBytes: bytesTotal,
		ManifestFile: filepath.ToSlash(filepath.Join(outPath, "file-manifest.json")), ManifestSHA256: hex.EncodeToString(manifestSum[:]),
		AuditorSourceSHA256: hex.EncodeToString(auditorSum[:]), JSONFiles: jsonFiles, PrivacyMatches: privacy,
		UnknownJSONKeys: unknown, DuplicateJSONKeys: duplicates, BinaryFiles: binary,
		UnexpectedPaths: len(findings), Findings: findings,
		Notes: []string{
			"The exact four-file allowlist was scanned, including the source-bound compiler report source string and the preserved pre-fix failure report.",
			"This is a publication/privacy scan only; it does not independently validate the compiler change or semantic result.",
		},
	}
	receiptBytes, err := json.MarshalIndent(r, "", "  ")
	must(err)
	receiptBytes = append(receiptBytes, '\n')
	must(os.WriteFile(filepath.Join(outPath, "receipt.json"), receiptBytes, 0o644))
	if decision != "PASS" {
		fail("native Integer local privacy audit failed")
	}
	fmt.Printf("PASS files=%d bytes=%d manifest_sha256=%s\n", len(manifest), bytesTotal, hex.EncodeToString(manifestSum[:]))
}

func scanJSON(data []byte) (values, unknown, duplicates int, err error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := scanValue(d, &unknown, &duplicates); err != nil {
		return 0, unknown, duplicates, err
	}
	if _, err := d.Token(); err != io.EOF {
		return 0, unknown, duplicates, errors.New("extra JSON tokens")
	}
	return 1, unknown, duplicates, nil
}

func scanValue(d *json.Decoder, unknown, duplicates *int) error {
	tok, err := d.Token()
	if err != nil {
		return err
	}
	switch delim := tok.(type) {
	case json.Delim:
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				keyToken, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("non-string key")
				}
				if seen[key] {
					*duplicates++
				}
				seen[key] = true
				if !knownKeys[key] {
					*unknown++
				}
				if err := scanValue(d, unknown, duplicates); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return errors.New("object terminator expected")
			}
		case '[':
			for d.More() {
				if err := scanValue(d, unknown, duplicates); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return errors.New("array terminator expected")
			}
		default:
			return errors.New("invalid delimiter")
		}
	}
	return nil
}

func scanSensitive(data []byte) int {
	count := 0
	for _, pattern := range sensitivePatterns {
		count += len(pattern.FindAll(data, -1))
	}
	return count
}

func hasBinaryControl(data []byte) bool {
	for _, b := range data {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			return true
		}
	}
	return false
}

func words(value string) map[string]bool {
	result := map[string]bool{}
	for _, word := range strings.Fields(value) {
		result[word] = true
	}
	return result
}

func must(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
