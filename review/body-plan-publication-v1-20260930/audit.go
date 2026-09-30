package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
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
	runRelative  = "runs/body-plan-v1-laya-7d626b9-20260930"
	outputDir    = "review/body-plan-publication-v1-20260930"
	manifestName = "file-manifest-v2.json"
	receiptName  = "receipt-v2.json"
	armCount     = 10
	caseCount    = 128
)

var arms = []string{
	"deterministic", "deterministic_search", "fp32", "fp32_search",
	"laya_multilingual", "laya_multilingual_search", "ptq_ternary",
	"ptq_ternary_search", "qat_ternary", "qat_ternary_search",
}

// This allowlist is the union of field names in the frozen public evidence
// schemas. A new field name fails the audit, even when its value is empty.
var publicJSONKeys = words(`
FAIL_CLOSED PASS PROGRESS UNKNOWN act_probability action activity activity_id add aggregate_completeness_score allowed_investment and answer_confidence answers arithmetic-pipeline arm attempts boolean-composition boundary candidate_routes cells choice choices cohort_sha256 compiled_go_cases_observed compiled_go_cases_planned compiled_go_cases_unknown compiled_go_heldout compiled_go_training compiler_source_sha completeness_percent completeness_receipt confidence core_dimensions correct criteria cumulative_cpu_seconds decision decision_basis denominator detection deterministic_replay dimensions domain_scope elapsed_ms emitted_choices equal equivalence_rule equivalent evidence excluded_effects excluded_scope executable_pins_stable_after_execution execution_environment external_provider_calls fail_closed_reason fallback_reason family family_counts feedback_model_calls final_route final_training_score first_unresolved fp32 generated_digest generated_semantic_digest global_confidence go_tool_sha256 gooo_binary_sha256 gooo_source_sha256 h1 h2 h3 h4 heldout hole_id holes human_actions id initial_selection initial_training_score input_tokens input_type instructions label laya_mode laya_posts_attempted laya_posts_planned laya_provider laya_revision less_equal less_than limits lowered_constructs lowered_semantic_units maximum_attempts maximum_human_actions maximum_repository_writes method mode model model_artifact_sha256 model_json model_prediction_calls model_variant multiply native_failed_or_unknown_cells native_generated_cells native_go_source_sha256 native_gooo_generation_pass nested-branch next_operation non_authorizing non_executing not_claimed numerator observed_selected_cells observed_system_cost operation or oracle_correct_holes output_tokens output_type piecewise-branch plan_sha256 planned_cells planned_intents polynomial-order predict_ns probabilities probability profile_id program_digest proposed proposed_route provider ptq_ternary qat_ternary questions raw_probabilities reason reassignment replay_digest repo report repository_writes request request_sha256 response_bytes_base64 rolling_pcpu_one_core_percent route route_decision route_decision_latency_ms route_equivalence route_selection routing rss_kib rule runner_binary_sha256 schema scope selected sequence signed-boundary source source_constructs source_digest source_revision source_semantic_digest source_semantic_units state status status_counts stop_reason subtract system_budget text_sha256 threshold-comparison tiny_predictions_observed toolchain total total_holes training training_candidate_attempt_limit training_score training_search type typecheck_passed unit unobserved_selection_cells unresolved_claims unsupported_constructs usage wall_ms wall_ns weights_bin weights_sha256 workflow
`)

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(/Users/|/home/|/private/|/var/folders/|/tmp/|[A-Z]:\\Users\\|/opt/homebrew/|/Applications/)`),
	regexp.MustCompile(`(?i)\b(?:Bearer\s+[A-Za-z0-9._~+/=-]+|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|hf_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9]{20,}|AKIA[A-Z0-9]{16}|AIza[0-9A-Za-z_-]{35}|xox[baprs]-[A-Za-z0-9-]{10,})\b`),
	regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`),
	regexp.MustCompile(`(?i)\b(?:host|machine|session)[_-]?id\b`),
	regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`),
}

var sensitiveKeys = map[string]bool{
	"api_key": true, "apikey": true, "access_token": true, "refresh_token": true,
	"authorization": true, "credential": true, "credentials": true,
	"password": true, "secret": true, "host_id": true, "machine_id": true,
	"session_id": true, "user_id": true, "username": true,
}

type manifestEntry struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}

type finding struct {
	Path     string `json:"path"`
	Category string `json:"category"`
	Count    int    `json:"count,omitempty"`
}

type receipt struct {
	Schema                     string    `json:"schema"`
	AuditedUTC                 string    `json:"audited_utc"`
	RunPath                    string    `json:"run_path"`
	Decision                   string    `json:"decision"`
	Allowlist                  string    `json:"allowlist"`
	ExpectedFiles              int       `json:"expected_files"`
	ObservedFiles              int       `json:"observed_files"`
	ExpectedDirectories        int       `json:"expected_directories"`
	ObservedDirectories        int       `json:"observed_directories"`
	TotalBytes                 int64     `json:"total_bytes"`
	ManifestFile               string    `json:"manifest_file"`
	ManifestSHA256             string    `json:"manifest_sha256"`
	AuditSourceSHA256          string    `json:"audit_source_sha256"`
	RawExchangeBundles         int       `json:"raw_exchange_bundles"`
	RawProviderRequests        int       `json:"raw_provider_requests"`
	DecodedProviderReplies     int       `json:"decoded_provider_replies"`
	DecodedReplyBytes          int64     `json:"decoded_reply_bytes"`
	DecodedReplyPrivacyMatches int       `json:"decoded_reply_privacy_matches"`
	ValidatedJSONValues        int       `json:"validated_json_values"`
	TextFileCount              int       `json:"text_file_count"`
	BinaryFileCount            int       `json:"binary_file_count"`
	UnexpectedPaths            int       `json:"unexpected_paths"`
	UnknownJSONKeys            int       `json:"unknown_json_keys"`
	DuplicateJSONKeys          int       `json:"duplicate_json_keys"`
	PrivacyMatches             int       `json:"privacy_matches"`
	Findings                   []finding `json:"findings"`
	Notes                      []string  `json:"notes"`
}

func main() {
	runArg := flag.String("run", runRelative, "run directory relative to repository root")
	outArg := flag.String("out", outputDir, "review output directory relative to repository root")
	flag.Parse()

	if *runArg != runRelative || *outArg != outputDir || filepath.IsAbs(*runArg) || filepath.IsAbs(*outArg) {
		fmt.Fprintln(os.Stderr, "audit accepts only the pinned run path and a repository-relative output path")
		os.Exit(2)
	}
	if err := audit(*runArg, *outArg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func audit(runPath, outPath string) error {
	_, expected := collectExpectedFiles()
	expectedDirs := collectExpectedDirectories()
	root, err := filepath.Abs(runPath)
	if err != nil {
		return err
	}
	actual := make(map[string]os.FileInfo)
	actualDirs := make(map[string]bool)
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.Mode()&os.ModeSymlink != 0 {
			actual[rel] = info
			return nil
		}
		if info.IsDir() {
			actualDirs[rel] = true
			return nil
		}
		actual[rel] = info
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk run payload: %w", err)
	}

	findings := make([]finding, 0)
	for path := range actual {
		if _, ok := expected[path]; !ok {
			findings = append(findings, finding{Path: path, Category: "unexpected_path_or_symlink"})
		}
	}
	for path := range expected {
		if _, ok := actual[path]; !ok {
			findings = append(findings, finding{Path: path, Category: "missing_allowlisted_file"})
		}
	}
	for path := range actualDirs {
		if !expectedDirs[path] {
			findings = append(findings, finding{Path: path, Category: "unexpected_directory"})
		}
	}
	for path := range expectedDirs {
		if !actualDirs[path] {
			findings = append(findings, finding{Path: path, Category: "missing_allowlisted_directory"})
		}
	}

	manifest := make([]manifestEntry, 0, len(actual))
	var totalBytes int64
	textCount, binaryCount := 0, 0
	privacyMatches, unknownKeys, duplicateKeys, validatedJSON := 0, 0, 0, 0
	rawBundles, rawRequests, decodedReplies := 0, 0, 0
	decodedReplyPrivacy := 0
	var decodedBytes int64

	paths := make([]string, 0, len(actual))
	for p := range actual {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		info := actual[rel]
		full := filepath.Join(root, filepath.FromSlash(rel))
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if !info.Mode().IsRegular() {
			findings = append(findings, finding{Path: rel, Category: "non_regular_file"})
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		sum := sha256.Sum256(data)
		kind := "text"
		if !utf8.Valid(data) || containsBinaryControl(data) {
			kind = "binary_or_invalid_text"
			binaryCount++
			findings = append(findings, finding{Path: rel, Category: "binary_or_invalid_text"})
		} else {
			textCount++
		}
		manifest = append(manifest, manifestEntry{Path: rel, Bytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:]), Kind: kind})
		totalBytes += int64(len(data))

		if matches := scanSensitive(data); matches > 0 {
			privacyMatches += matches
			findings = append(findings, finding{Path: rel, Category: "sensitive_content_pattern", Count: matches})
		}

		if strings.HasSuffix(rel, ".json") || strings.HasSuffix(rel, ".jsonl") {
			values, keys, duplicates, err := scanJSONFile(data, strings.HasSuffix(rel, ".jsonl"))
			if err != nil {
				findings = append(findings, finding{Path: rel, Category: "invalid_or_duplicate_json"})
			} else {
				validatedJSON += values
				unknownKeys += keys
				duplicateKeys += duplicates
				if keys > 0 {
					findings = append(findings, finding{Path: rel, Category: "unknown_json_field", Count: keys})
				}
				if duplicates > 0 {
					findings = append(findings, finding{Path: rel, Category: "duplicate_json_field", Count: duplicates})
				}
			}
		}

		if strings.HasSuffix(rel, "-laya-exchanges.json") {
			n, replyCount, replyBytes, replyPrivacy, err := scanExchanges(data)
			if err != nil {
				findings = append(findings, finding{Path: rel, Category: "invalid_exchange_or_provider_reply"})
			} else {
				rawBundles++
				rawRequests += n
				decodedReplies += replyCount
				decodedBytes += replyBytes
				decodedReplyPrivacy += replyPrivacy
				if replyPrivacy > 0 {
					findings = append(findings, finding{Path: rel, Category: "decoded_reply_sensitive_content_pattern", Count: replyPrivacy})
				}
				if n == 0 {
					findings = append(findings, finding{Path: rel, Category: "empty_exchange_bundle"})
				}
			}
		}
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestSum := sha256.Sum256(manifestBytes)
	auditSource, err := os.ReadFile(filepath.Join(outPath, "audit.go"))
	if err != nil {
		return fmt.Errorf("read archived audit source: %w", err)
	}
	auditSourceSum := sha256.Sum256(auditSource)
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		return err
	}
	manifestPath := filepath.Join(outPath, manifestName)
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return err
	}

	decision := "PASS"
	if len(findings) > 0 || len(actual) != len(expected) || rawBundles != caseCount || rawRequests != 222 || decodedReplies != 222 {
		decision = "FAIL"
	}
	r := receipt{
		Schema: "gooo/body-plan-publication-privacy-audit/v1", AuditedUTC: time.Now().UTC().Format(time.RFC3339Nano),
		RunPath: runPath, Decision: decision, Allowlist: "10 arms x 128 case directories x 5 files; 10 compiled-replay directories x 4 files; pinned root evidence set",
		ExpectedFiles: len(expected), ObservedFiles: len(actual), TotalBytes: totalBytes,
		ExpectedDirectories: len(expectedDirs), ObservedDirectories: len(actualDirs),
		ManifestFile: filepath.ToSlash(filepath.Join(outPath, manifestName)), ManifestSHA256: hex.EncodeToString(manifestSum[:]),
		AuditSourceSHA256:  hex.EncodeToString(auditSourceSum[:]),
		RawExchangeBundles: rawBundles, RawProviderRequests: rawRequests,
		DecodedProviderReplies: decodedReplies, DecodedReplyBytes: decodedBytes,
		DecodedReplyPrivacyMatches: decodedReplyPrivacy,
		ValidatedJSONValues:        validatedJSON, TextFileCount: textCount, BinaryFileCount: binaryCount,
		UnexpectedPaths: countFindings(findings, "unexpected_path_or_symlink") + countFindings(findings, "missing_allowlisted_file") +
			countFindings(findings, "unexpected_directory") + countFindings(findings, "missing_allowlisted_directory"),
		UnknownJSONKeys: unknownKeys, DuplicateJSONKeys: duplicateKeys,
		PrivacyMatches: privacyMatches + decodedReplyPrivacy,
		Findings:       findings,
		Notes: []string{
			"Raw exchange request objects and all base64 provider replies were decoded and scanned; captured provider replies are retained byte-for-byte in the run directory.",
			"The resource-monitor source accepts a PID argument generically; the run record contains no PID field or process identifier.",
			"This privacy audit does not validate model quality, training/held-out scoring, or compiler semantics; those are separate reviews.",
			"The manifest hashes every allowlisted payload file; the manifest itself is bound by manifest_sha256.",
		},
	}
	receiptBytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	receiptBytes = append(receiptBytes, '\n')
	if err := os.WriteFile(filepath.Join(outPath, receiptName), receiptBytes, 0o644); err != nil {
		return err
	}
	if decision != "PASS" {
		return fmt.Errorf("privacy audit failed with %d findings", len(findings))
	}
	fmt.Printf("PASS files=%d bytes=%d exchanges=%d replies=%d manifest_sha256=%s\n", len(manifest), totalBytes, rawBundles, decodedReplies, hex.EncodeToString(manifestSum[:]))
	return nil
}

func collectExpectedFiles() ([]string, map[string]bool) {
	var paths []string
	for _, name := range []string{
		"README.md", "laya-process-observations.jsonl", "preexecution.json", "progress.json", "report.json",
		"resource-monitor.go.txt", "selected-cells.jsonl",
	} {
		paths = append(paths, name)
	}
	for i := 1; i <= caseCount; i++ {
		paths = append(paths, fmt.Sprintf("body-%03d-laya-exchanges.json", i))
	}
	for _, arm := range arms {
		for i := 1; i <= caseCount; i++ {
			base := fmt.Sprintf("%s/body-%03d/", arm, i)
			for _, name := range []string{"input.gooo", "native-generated.go.txt", "native-stderr.txt", "native-stdout.json", "plan-generated.go.txt"} {
				paths = append(paths, base+name)
			}
		}
		for _, name := range []string{"generated.go", "generated_test.go", "go-test-output.txt", "go.mod"} {
			paths = append(paths, arm+"/compiled-replay/"+name)
		}
	}
	set := make(map[string]bool, len(paths))
	for _, p := range paths {
		set[p] = true
	}
	return paths, set
}

func collectExpectedDirectories() map[string]bool {
	set := make(map[string]bool, armCount*(caseCount+1))
	for _, arm := range arms {
		set[arm] = true
		set[arm+"/compiled-replay"] = true
		for i := 1; i <= caseCount; i++ {
			set[fmt.Sprintf("%s/body-%03d", arm, i)] = true
		}
	}
	return set
}

func scanExchanges(data []byte) (int, int, int64, int, error) {
	var rows []map[string]any
	if err := decodeJSON(data, &rows); err != nil {
		return 0, 0, 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, 0, 0, errors.New("empty exchange array")
	}
	var replyBytes int64
	privacyMatches := 0
	for _, row := range rows {
		if !hasExactKeys(row, "hole_id", "request", "response_bytes_base64", "status", "wall_ns") {
			return 0, 0, 0, 0, errors.New("exchange row key mismatch")
		}
		if !isString(row["hole_id"]) || !isString(row["status"]) || !isNumber(row["wall_ns"]) {
			return 0, 0, 0, 0, errors.New("exchange scalar type mismatch")
		}
		request, ok := row["request"].(map[string]any)
		if !ok || !hasExactKeys(request, "model", "questions", "state") {
			return 0, 0, 0, 0, errors.New("request field mismatch")
		}
		if !isString(request["model"]) {
			return 0, 0, 0, 0, errors.New("request model type mismatch")
		}
		questionSet, ok := request["questions"].(map[string]any)
		if !ok || !hasExactKeys(questionSet, "operation") {
			return 0, 0, 0, 0, errors.New("question set mismatch")
		}
		question, ok := questionSet["operation"].(map[string]any)
		if !ok || !hasExactKeys(question, "criteria", "instructions", "type") {
			return 0, 0, 0, 0, errors.New("question field mismatch")
		}
		if !isString(question["instructions"]) || !isString(question["type"]) {
			return 0, 0, 0, 0, errors.New("question scalar type mismatch")
		}
		criteria, ok := question["criteria"].(map[string]any)
		if !ok || len(criteria) == 0 {
			return 0, 0, 0, 0, errors.New("criteria mismatch")
		}
		for key := range criteria {
			if !map[string]bool{"add": true, "and": true, "equal": true, "less_equal": true, "less_than": true, "multiply": true, "or": true, "subtract": true}[key] {
				return 0, 0, 0, 0, errors.New("unknown operation criterion")
			}
			if !isString(criteria[key]) {
				return 0, 0, 0, 0, errors.New("criterion description is not text")
			}
		}
		state, ok := request["state"].(map[string]any)
		if !ok || !hasExactKeys(state, "request") {
			return 0, 0, 0, 0, errors.New("state field mismatch")
		}
		if !isString(state["request"]) {
			return 0, 0, 0, 0, errors.New("request state is not text")
		}
		encoded, ok := row["response_bytes_base64"].(string)
		if !ok {
			return 0, 0, 0, 0, errors.New("reply is not base64 text")
		}
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || !utf8.Valid(raw) {
			return 0, 0, 0, 0, errors.New("reply is invalid base64 or UTF-8")
		}
		var reply map[string]any
		if err := decodeJSON(raw, &reply); err != nil {
			return 0, 0, 0, 0, err
		}
		_, unknown, duplicates, err := scanJSONFile(raw, false)
		if err != nil || unknown != 0 || duplicates != 0 {
			return 0, 0, 0, 0, errors.New("provider reply has invalid, duplicate, or unknown JSON fields")
		}
		if !hasExactKeys(reply, "answers", "model", "routing", "usage") {
			return 0, 0, 0, 0, errors.New("provider reply top-level field mismatch")
		}
		if !isString(reply["model"]) {
			return 0, 0, 0, 0, errors.New("provider model field is not text")
		}
		routing, ok := reply["routing"].(map[string]any)
		if !ok || !hasExactKeys(routing, "detection", "model", "reason", "repo", "workflow") {
			return 0, 0, 0, 0, errors.New("provider reply routing fields mismatch")
		}
		if !isNullableString(routing["detection"]) || !isString(routing["model"]) ||
			!isString(routing["reason"]) || !isString(routing["repo"]) || !isNullableString(routing["workflow"]) {
			return 0, 0, 0, 0, errors.New("provider reply routing scalar type mismatch")
		}
		answers, ok := reply["answers"].(map[string]any)
		if !ok || !hasExactKeys(answers, "operation") {
			return 0, 0, 0, 0, errors.New("provider reply answer fields mismatch")
		}
		answer, ok := answers["operation"].(map[string]any)
		if !ok || !hasExactKeys(answer, "action", "answer_confidence", "choice", "confidence", "probabilities", "type") {
			return 0, 0, 0, 0, errors.New("provider answer fields mismatch")
		}
		if !isNumber(answer["answer_confidence"]) || !isString(answer["choice"]) ||
			!isNumber(answer["confidence"]) || !isString(answer["type"]) {
			return 0, 0, 0, 0, errors.New("provider answer scalar type mismatch")
		}
		action, ok := answer["action"].(map[string]any)
		if !ok || !hasExactKeys(action, "act_probability") || !isNumber(action["act_probability"]) {
			return 0, 0, 0, 0, errors.New("provider action fields mismatch")
		}
		probabilities, ok := answer["probabilities"].(map[string]any)
		if !ok || len(probabilities) == 0 {
			return 0, 0, 0, 0, errors.New("provider probabilities mismatch")
		}
		for key, value := range probabilities {
			if !map[string]bool{"add": true, "and": true, "equal": true, "less_equal": true, "less_than": true, "multiply": true, "or": true, "subtract": true}[key] || !isNumber(value) {
				return 0, 0, 0, 0, errors.New("provider probability entry mismatch")
			}
		}
		usage, ok := reply["usage"].(map[string]any)
		if !ok || !hasExactKeys(usage, "input_tokens", "output_tokens") {
			return 0, 0, 0, 0, errors.New("provider usage fields mismatch")
		}
		if !isNumber(usage["input_tokens"]) || !isNumber(usage["output_tokens"]) {
			return 0, 0, 0, 0, errors.New("provider usage fields have wrong type")
		}
		privacyMatches += scanSensitive(raw)
		replyBytes += int64(len(raw))
	}
	return len(rows), len(rows), replyBytes, privacyMatches, nil
}

func decodeJSON(data []byte, target any) error {
	if !utf8.Valid(data) {
		return errors.New("invalid UTF-8 JSON")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func scanJSONFile(data []byte, jsonl bool) (values, unknown, duplicates int, err error) {
	lines := [][]byte{data}
	if jsonl {
		lines = bytes.Split(data, []byte("\n"))
	}
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.UseNumber()
		if e := scanJSONValue(dec, &unknown, &duplicates); e != nil {
			return values, unknown, duplicates, e
		}
		if tok, e := dec.Token(); e != io.EOF {
			if e != nil {
				return values, unknown, duplicates, e
			}
			return values, unknown, duplicates, fmt.Errorf("trailing JSON token %v", tok)
		}
		values++
	}
	return values, unknown, duplicates, nil
}

func scanJSONValue(dec *json.Decoder, unknown, duplicates *int) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("non-string object key")
				}
				if seen[key] {
					*duplicates++
				}
				seen[key] = true
				if !publicJSONKeys[key] {
					*unknown++
				}
				if sensitiveKeys[strings.ToLower(key)] {
					*unknown++
				}
				if err := scanJSONValue(dec, unknown, duplicates); err != nil {
					return err
				}
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim('}') {
				return errors.New("unterminated JSON object")
			}
		case '[':
			for dec.More() {
				if err := scanJSONValue(dec, unknown, duplicates); err != nil {
					return err
				}
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim(']') {
				return errors.New("unterminated JSON array")
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", value)
		}
	}
	return nil
}

func containsBinaryControl(data []byte) bool {
	for _, b := range data {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			return true
		}
	}
	return false
}

func scanSensitive(data []byte) int {
	count := 0
	for _, re := range sensitivePatterns {
		count += len(re.FindAll(data, -1))
	}
	return count
}

func hasExactKeys(value map[string]any, expected ...string) bool {
	if len(value) != len(expected) {
		return false
	}
	for _, key := range expected {
		if _, ok := value[key]; !ok {
			return false
		}
	}
	return true
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func isNullableString(value any) bool {
	return value == nil || isString(value)
}

func isNumber(value any) bool {
	_, ok := value.(json.Number)
	return ok
}

func countFindings(findings []finding, category string) int {
	count := 0
	for _, f := range findings {
		if f.Category == category {
			count++
		}
	}
	return count
}

func words(value string) map[string]bool {
	out := map[string]bool{}
	for _, word := range strings.Fields(value) {
		out[word] = true
	}
	return out
}
