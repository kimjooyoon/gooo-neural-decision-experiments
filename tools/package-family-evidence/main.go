// package-family-evidence publishes a bounded, deterministic evidence appendix.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
)

const repository = "asketeddy/gooo-feedback-path-tiny-v1"
const prefix = "research/native-family-20261001/"
const maxFile = 4 << 20
const maxTotal = 40 << 20

var privacy = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)

// Every regex alternative requires one of these exact byte substrings. This
// necessary-prefix check avoids regex work on large ordinary evidence strings;
// suspected text still uses the original expression and rejection semantics.
func privateText(raw []byte) bool {
	if bytes.Contains(raw, []byte("/Users/")) || bytes.Contains(raw, []byte("/private/var/")) {
		return true
	}
	for _, prefix := range [5]string{"hf_", "ghp_", "github_pat_", "sk-", "Bearer"} {
		if bytes.Contains(raw, []byte(prefix)) {
			return privacy.Match(raw)
		}
	}
	return false
}

var fixed = map[string]string{
	"docs/retained-native-preregistration.md":                          "25b759ed3c054bdefad6a3263a36e02d07965aed0ac362b78ab3f6e0a373f959",
	"runs/retained-native-feature-pilot-20261001/metrics-summary.json": "2f27f9c6f27d350f4693ae541a3124db099d1e9dc207c082dada124be51bf452",
	"runs/retained-native-feature-pilot-20261001/report.json":          "581c09ebb1746505a636a44d8eb34cdeef333fd3204e6ad9120775ba1c5c7e9b",
	"runs/retained-native-feature-pilot-20261001/audit.json":           "581c09ebb1746505a636a44d8eb34cdeef333fd3204e6ad9120775ba1c5c7e9b",
	"runs/retained-native-feature-pilot-20261001/preexecution.json":    "d6afa79e902576da1ccf176b89395b842d5d0d793c248d2580ea561b9af2c734",
	"runs/unfixed-native-main-pilot-20261001/preexecution.json":        "333f5b1cf65a9c45f1c10ee01ce7a53e3dd989f915eec764cc2a7e30a5c09c39",
	"runs/unfixed-native-main-pilot-20261001/report.json":              "6bf4030ee7fe769fd3befe8b5bee66bb6fe00fe272037153f4197d59cc6818f6",
	"runs/unfixed-native-main-pilot-20261001/audit.json":               "6bf4030ee7fe769fd3befe8b5bee66bb6fe00fe272037153f4197d59cc6818f6",
	"runs/unfixed-native-feature-pilot-20261001/preexecution.json":     "b7fd3eedddabac4e37864b05a77d6f65e08dd60b1f626bb83134ab1126e8bdd0",
	"runs/unfixed-native-feature-pilot-20261001/report.json":           "4dba2dab42c17af2c069a442b18053fc804bc50a32f5b4276799ddf3f96d3962",
	"runs/unfixed-native-feature-pilot-20261001/audit.json":            "4dba2dab42c17af2c069a442b18053fc804bc50a32f5b4276799ddf3f96d3962",
	"studies/feedback-family-v1/cohort.jsonl":                          "0e9d5c2cb9a816ca4d05c2e6e2d94ceccb9a0d02912edd4cc1063c0adb648a70",
	"runs/feedback-family-pilot-20261001/report.json":                  "58090950d759d47d7c0c42ca9e6edd1929c50864cceb5a4b9f37fbc33c00da00",
	"runs/feedback-family-pilot-20261001/audit.json":                   "92884e427b9499b643f7a588e8eb39a7425f6d2d079edfbe3691fdf45e2103a8",
	"runs/feedback-family-matrix-20261001/report.json":                 "9c00cf48dae46bdc793b39c5ae94f15c825194d33eabe2bcf3002161aba87c2d",
	"runs/feedback-family-matrix-20261001/audit.json":                  "c732643f2312543a3bc669f878bef0d63ed8c9d671d6225db20d4b073512d0b2",
	"runs/feedback-family-optimized-feature-20261001/report.json":      "0346231d22f540a844fc9296b408a64c59bb33d538223603c10f65a281c63d62",
	"runs/feedback-family-optimized-feature-20261001/audit.json":       "8fcbc56ad12c812755d6925082a72baeab739c0e006d12fdfb27a56ac86adc47",
	"runs/feedback-family-optimized-main-20261001/report.json":         "4537080ed58d19f6744e6d615c5c6eb63632c9494d96637282c4f36a2990b083",
	"runs/feedback-family-optimized-main-20261001/audit.json":          "770b268ac84ab0fadb4a489b193f2008e3ec95fa027921b41da77742dd761be4",
	"studies/compound-path-v1/cohort.jsonl":                            "aefd16b22076c05b113e29df3e7788b06e60e0c5c14e45cc75547bd1e582a15a",
	"runs/compound-path-pilot-20261001/report.json":                    "11fa52d7fbc6c1bbd04eeacffd7ea82db440fd552ca03d0836de9b03cab44525",
	"runs/compound-path-pilot-20261001/audit.json":                     "20335902beda6880732df639a038d684a7b841b8f35e286a240d1e2d8acd24dc",
	"runs/compound-path-main-20261001/report.json":                     "dec30461109eafcea587e7d78a533087df21879fb23bb7edc68c5645aefca66e",
	"runs/compound-path-main-20261001/audit.json":                      "74d5ed1e3f36da15ad14c599a56c909d06a75b0411bdf795ad0fdfb44fb5d7c1",
	"runs/unfixed-feedback-sdk-20261001/report.json":                   "41b0e37c2ba99fa44d2173a25e4c2d7174e21115b9dff0a29d3c512664bf839e",
	"runs/unfixed-feedback-sdk-20261001/audit.json":                    "41b0e37c2ba99fa44d2173a25e4c2d7174e21115b9dff0a29d3c512664bf839e",
}

type entry struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema     string  `json:"schema"`
	Repository string  `json:"repository"`
	Prefix     string  `json:"prefix"`
	Raw        []entry `json:"archive_entries"`
	Files      []entry `json:"publication_files"`
	Scope      string  `json:"scope"`
}

func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func read(path string) ([]byte, error) {
	i, err := os.Lstat(path)
	if err != nil || !i.Mode().IsRegular() || i.Size() <= 0 || i.Size() > maxFile {
		return nil, errors.New("bounded regular source required")
	}
	raw, err := os.ReadFile(path)
	if err != nil || privateText(raw) {
		return nil, errors.New("private or unreadable publication text rejected")
	}
	if pin := fixed[path]; pin != "" && hash(raw) != pin {
		return nil, errors.New("frozen evidence digest differs")
	}
	return raw, nil
}
func names() ([]string, error) {
	return namesKind("")
}
func namesKind(kind string) ([]string, error) {
	if kind == "retained-native-feature" {
		return retainedNativeNames()
	}
	if kind == "unfixed-native-feature" || kind == "unfixed-native-main" {
		return nativeUnfixedNames(kind)
	}
	if kind == "unfixed-sdk" {
		return unfixedNames()
	}
	names := []string{"studies/feedback-family-v1/cohort.jsonl", "studies/feedback-family-v1/manifest.json"}
	idRE := regexp.MustCompile(`^[a-z0-9_-]+$`)
	shaRE := regexp.MustCompile(`^[a-f0-9]{64}$`)
	roots := []string{"runs/feedback-family-pilot-20261001", "runs/feedback-family-matrix-20261001"}
	expected := 1128
	if kind == "optimized-feature" {
		roots = []string{"runs/feedback-family-optimized-feature-20261001"}
		expected = 1105
	} else if kind == "optimized-main" {
		roots = []string{"runs/feedback-family-optimized-main-20261001"}
		expected = 1105
	} else if kind == "compound-main" {
		names = []string{"studies/compound-path-v1/cohort.jsonl", "studies/compound-path-v1/manifest.json"}
		roots = []string{"runs/compound-path-pilot-20261001", "runs/compound-path-main-20261001"}
		expected = 674
	} else if kind != "" {
		return nil, errors.New("unknown frozen evidence kind")
	}
	for _, root := range roots {
		for _, name := range []string{"report.json", "preexecution.json", "audit.json"} {
			names = append(names, root+"/"+name)
		}
		raw, err := read(root + "/report.json")
		if err != nil {
			return nil, err
		}
		var report struct {
			Observations []struct {
				ID      string `json:"id"`
				Capture string `json:"capture_sha256"`
			} `json:"observations"`
		}
		if err = json.Unmarshal(raw, &report); err != nil {
			return nil, err
		}
		for _, o := range report.Observations {
			if !idRE.MatchString(o.ID) || !shaRE.MatchString(o.Capture) {
				return nil, errors.New("unsafe capture name")
			}
			path := root + "/captures/" + o.ID + ".json"
			data, err := read(path)
			if err != nil || hash(data) != o.Capture {
				return nil, errors.New("capture bytes differ from frozen native report")
			}
			names = append(names, path)
		}
		raw, err = read(root + "/audit.json")
		if err != nil {
			return nil, err
		}
		var audit struct {
			Executions map[string]string `json:"execution_sha256"`
		}
		if err = json.Unmarshal(raw, &audit); err != nil {
			return nil, err
		}
		for sha, pin := range audit.Executions {
			if !shaRE.MatchString(sha) || !shaRE.MatchString(pin) {
				return nil, errors.New("unsafe execution digest")
			}
			path := root + "/executions/" + sha + ".json"
			data, err := read(path)
			if err != nil || hash(data) != pin {
				return nil, errors.New("execution evidence changed")
			}
			names = append(names, path)
		}
	}
	sort.Strings(names)
	if len(names) != expected {
		return nil, errors.New("fixed raw inventory count differs")
	}
	for i, name := range names {
		if i > 0 && name == names[i-1] {
			return nil, errors.New("duplicate archive name")
		}
	}
	return names, nil
}
func build() (manifest, map[string][]byte, error) {
	return buildKind("")
}
func buildKind(kind string) (manifest, map[string][]byte, error) {
	m := manifest{Schema: "gooo/native-family-publication/v1", Repository: repository, Prefix: prefix,
		Scope: "Appendix only; existing model weights/card unchanged. 1080 native calls, 1204 predictions including 244 redundant feedback predictions, 1638 candidates, 20 actual Go processes and 440 actual function evaluations. 120 contract/language views are not independent experiments. Sparse tests do not establish complete intent; new models underperform the parent. No new training/model/native/Go calls during packaging; no host utilization or causal speedup claim."}
	if kind == "optimized-feature" {
		m.Prefix = "research/native-family-optimized-feature-20261001/"
		m.Scope = "Frozen feature SDK v0.2.5 evidence only, not main deployment. 1080 native calls, 960 initial predictions, zero feedback predictions, 244 zero-call ranking_unnecessary receipts, 1638 candidates, 20 actual Go processes and 440 function evaluations. All 1080 baseline pairs preserve selected label/Go/finite/separate outcomes and attempts. Original feature CI failed obsolete module sums, then was canceled by the checksum repair; caller PASS hint was not authenticated CI. Existing core weights and card remain unchanged. No new training/model/native/Go calls during packaging. No host utilization or causal wall-speedup claim."
	} else if kind == "optimized-main" {
		m.Prefix = "research/native-family-optimized-main-20261001/"
		m.Scope = "Frozen SDK v0.2.5 main ef63060ed1aebd9d92a2fe4cf24ce9c5929b8726 evidence. 1080 native calls, 960 initial predictions, zero feedback predictions, 244 zero-call ranking_unnecessary receipts, 1638 candidates, 20 actual Go processes and 440 function evaluations. All 1080 baseline pairs preserve selected label/Go/finite/separate outcomes and attempts. Captured caller CI status is UNKNOWN because post-push CI was pending at execution; this unauthenticated hint is retained unchanged. Existing core weights and card remain unchanged. No new training/model/native/Go calls during packaging. Repeated contract/language/arm views are not independent experiments. No host utilization or causal wall-speedup claim."
	} else if kind == "compound-main" {
		m.Schema = "gooo/compound-path-publication/v1"
		m.Prefix = "research/compound-path-main-20261001/"
		m.Scope = "Three interacting body templates, twelve structural intention groups, two decisions and four whole-body paths. Main matrix: 648 native calls, 1590 predictions including 438 feedback predictions, 1405 candidates, 12 actual Go processes and 192 actual function evaluations. Pilot: three disconnected native calls, three Go processes and 42 evaluations. All 288 ranking/feedback pairs retain emitted Go and finite/separate outcomes; 16 candidate sequences differ. QAT uses two fewer candidate attempts while the parent uses one more. 97 feedback predictions concern coordinates constant among remaining paths; skipping them has not been implemented or measured. Exact typed function AST is reconciled modulo in-range int64 literal wrappers. One synthetic configuration; repeated policy views are not independent experiments or untouched language accuracy. Core weights/card and earlier appendices are unchanged. No training, model/native/Go calls during packaging; no upstream Laya call, host utilization or causal timing claim."
	} else if kind == "unfixed-sdk" {
		m.Schema = "gooo/unfixed-feedback-publication/v1"
		m.Prefix = "research/unfixed-feedback-sdk-20261001/"
		m.Scope = "Go SDK opt-in ReconsiderUnfixed; native main still uses SDK 2.5/default feedback. 576 actual SDK sessions, 1931 own frozen tiny-model predictions: 1014 legacy and 917 optimized, skipping 97 fixed-coordinate predictions. All 288 pairs retain candidate sequence and selected body/finite results; legacy matches 288 frozen native references. 12 actual generated Go processes and 192 function evaluations match an independent integer-state oracle. Counts use fixed 64-byte arrays. Removing common log factors can alter floating-point near ties on other data. Reused bilingual development views are not independent new tasks or untouched accuracy. Existing core model and earlier appendix bytes remain unchanged. No training/model/native/Go calls while packaging; no causal timing or host CPU improvement claim."
	}
	if kind == "unfixed-native-feature" || kind == "unfixed-native-main" {
		m.Schema = "gooo/native-unfixed-publication/v1"
		m.Prefix = "research/" + nativeUnfixedRoot(kind)[len("runs/"):] + "/"
		m.Scope = "Actual native SDK 0.2.7 pilot using own frozen models, six reused bilingual contradictory views and 24 legacy-first pairs. The explicit --path-feedback-unfixed option ranks varying typed coordinates. Raw native bodies, resource sidecars, derived skip receipts, partial finite outcomes and actual emitted-Go values are retained. Caller PASS is unauthenticated context; source CI/GoProof is separate. Feature/main edition is explicit in the prefix and captured compiler SHA. This does not establish arbitrary natural-language accuracy, causal wall speedup, host CPU growth or all-input ordering equivalence. Core weights/card and prior appendices are preserved. No training, GPU work, upstream Laya calls or model/native/Go calls during packaging or offline verification."
	}
	if kind == "retained-native-feature" {
		m.Schema = "gooo/retained-native-publication/v1"
		m.Prefix = "research/retained-native-feature-pilot-20261001/"
		m.Scope = "Actual native compiler/retained Go worker with identical clean feature source, five frozen own-model/disconnected arms, six reused bilingual contradictory views repeated forward/reverse. 180 valid constructions, 10 source rejections with zero predictions, 70 native processes, 720 predictions, 120 pairs with equal candidate sequences/bodies/actuals, 87.5% finite completeness, three actual Go executions and 42 function values against an independent state oracle. Default one worker; four-worker throughput costs more per-request CPU and resident process memory than sequential retained mode. Fresh response includes startup/load; sequential retained response excludes one startup/setup; parallel latency includes queueing. First disconnected binary invocations have large startup/wall outliers and are preserved. Original UNKNOWN CI hint is not authority. Feature evidence is distinct from main deployment. Core weights/card and earlier appendices unchanged; no training/GPU/upstream Laya or model/native/Go calls during packaging/audit. No causal speedup, host CPU, arbitrary text-generation or all-input correctness claim."
	}
	n, err := namesKind(kind)
	if err != nil {
		return m, nil, err
	}
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	z.Header.ModTime = time.Unix(0, 0)
	z.Header.OS = 255
	t := tar.NewWriter(z)
	total := 0
	files := map[string][]byte{}
	for _, name := range n {
		raw, err := read(name)
		if err != nil {
			return m, nil, err
		}
		total += len(raw)
		if total > maxTotal {
			return m, nil, errors.New("raw archive budget exceeded")
		}
		m.Raw = append(m.Raw, entry{name, hash(raw), len(raw)})
		if err = t.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(raw)), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}); err != nil {
			return m, nil, err
		}
		if _, err = t.Write(raw); err != nil {
			return m, nil, err
		}
		if strings.HasSuffix(name, "/report.json") || strings.HasSuffix(name, "/audit.json") || strings.HasPrefix(name, "studies/") {
			files[name] = raw
		}
	}
	if err = t.Close(); err != nil {
		return m, nil, err
	}
	if err = z.Close(); err != nil {
		return m, nil, err
	}
	files["evidence.tar.gz"] = append([]byte(nil), buf.Bytes()...)
	files["README.md"] = []byte("# Native Gooo five-family evidence\n\nFrozen SDK v0.2.4 matrix and independent replay audit. The archive contains the complete 1,128-file allowlist; hashes and byte sizes are in publication-manifest.json. Core model artifacts remain unchanged.\n\nRead the methods, negative results, finite completion and resource limitations at https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/native-feedback-families-study.md .\n\nThis appendix is evidence of bounded bilingual path construction, not arbitrary natural-language-to-code correctness. Separate-input observations reuse 20 actual Go executions; 120 views are not 120 independent experiments.\n")
	var payload []string
	if kind == "optimized-feature" {
		files["README.md"] = []byte("# Gooo sole-remaining-path feature evidence\n\n" + m.Scope + "\n\nThe deterministic archive contains 1,105 allowlisted files. See per-entry hashes and sizes in publication-manifest.json and methods/failure limitations at https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/native-feedback-families-study.md .\n")
	} else if kind == "optimized-main" {
		files["README.md"] = []byte("# Gooo sole-remaining-path main evidence\n\n" + m.Scope + "\n\nThe deterministic archive contains 1,105 allowlisted files. See per-entry hashes and sizes in publication-manifest.json and methods/failure limitations at https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/native-feedback-families-study.md .\n\nVerified main promotion: https://github.com/kimjooyoon/meta-ontology-go/pull/1123 . Subsequent CI evidence is recorded separately without rewriting the original UNKNOWN hints.\n")
	} else if kind == "compound-main" {
		files["README.md"] = []byte("# Interacting Gooo four-path main evidence\n\n" + m.Scope + "\n\nThe deterministic archive contains 674 allowlisted files. Per-entry hashes and sizes are in publication-manifest.json. Methods, distinct structural and functional denominators, finite ambiguity, resource observations and negative results are described at https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/compound-path-study.md .\n")
	} else if kind == "unfixed-sdk" {
		files["README.md"] = []byte("# Gooo opt-in unfixed feedback SDK evidence\n\n" + m.Scope + "\n\nThe deterministic archive contains 593 allowlisted files. Per-entry hashes and sizes are in publication-manifest.json. Methods and limitations: https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/unfixed-feedback-study.md .\n")
	}
	if kind == "unfixed-native-feature" || kind == "unfixed-native-main" {
		files["README.md"] = []byte("# Native Gooo varying-coordinate feedback pilot\n\n" + m.Scope + "\n\nSee report.json for actual measured counts. The deterministic archive preserves raw captures, resource sidecars and actual Go execution values; per-entry hashes and byte sizes are in publication-manifest.json. Methods and limitations: https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/native-unfixed-pilot.md .\n")
	}
	if kind == "retained-native-feature" {
		files["README.md"] = []byte("# Retained native Gooo construction evidence\n\n" + m.Scope + "\n\nThe deterministic archive preserves original JSON/NDJSON, setup/latency/CPU/RSS sidecars and actual generated-Go execution values. Per-entry digests and byte sizes are in publication-manifest.json. Methods, measurement boundaries and increased costs: https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/retained-native-study.md .\n")
		for _, path := range []string{"docs/retained-native-preregistration.md", "runs/retained-native-feature-pilot-20261001/metrics-summary.json"} {
			raw, err := read(path)
			if err != nil {
				return m, nil, err
			}
			files[path] = raw
		}
	}
	for name := range files {
		payload = append(payload, name)
	}
	sort.Strings(payload)
	for _, name := range payload {
		raw := files[name]
		m.Files = append(m.Files, entry{name, hash(raw), len(raw)})
	}
	return m, files, nil
}

func unfixedNames() ([]string, error) {
	root := "runs/unfixed-feedback-sdk-20261001"
	names := []string{"studies/compound-path-v1/cohort.jsonl", "studies/compound-path-v1/manifest.json",
		root + "/report.json", root + "/audit.json", root + "/preexecution.json"}
	raw, err := read(root + "/report.json")
	var report struct {
		Files map[string]string `json:"files_sha256"`
	}
	if err != nil || json.Unmarshal(raw, &report) != nil || len(report.Files) != 588 {
		return nil, errors.New("fixed unfixed SDK inventory differs")
	}
	nameRE := regexp.MustCompile(`^(captures/[a-z0-9_-]+|executions/[a-f0-9]{64})\.json$`)
	for path, pin := range report.Files {
		if !nameRE.MatchString(path) || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(pin) {
			return nil, errors.New("unsafe unfixed evidence name/hash")
		}
		full := root + "/" + path
		data, err := read(full)
		if err != nil || hash(data) != pin {
			return nil, errors.New("unfixed evidence bytes differ")
		}
		names = append(names, full)
	}
	sort.Strings(names)
	return names, nil
}

func nativeUnfixedRoot(kind string) string {
	if kind == "unfixed-native-feature" {
		return "runs/unfixed-native-feature-pilot-20261001"
	}
	return "runs/unfixed-native-main-pilot-20261001"
}
func nativeUnfixedNames(kind string) ([]string, error) {
	root := nativeUnfixedRoot(kind)
	if fixed[root+"/report.json"] == "" || fixed[root+"/preexecution.json"] == "" {
		return nil, errors.New("frozen native pilot pins required")
	}
	raw, err := read(root + "/report.json")
	var report struct {
		Schema     string            `json:"schema"`
		Decision   string            `json:"decision"`
		Calls      int               `json:"native_calls"`
		Pairs      int               `json:"paired_views"`
		Executions int               `json:"actual_go_processes"`
		Files      map[string]string `json:"files_sha256"`
	}
	if err != nil || json.Unmarshal(raw, &report) != nil || report.Schema != "gooo/native-unfixed-pilot/v1" ||
		report.Decision != "PASS" || report.Calls != 48 || report.Pairs != 24 || report.Executions < 1 || report.Executions > 48 ||
		len(report.Files) != 96+report.Executions {
		return nil, errors.New("frozen native pilot inventory differs")
	}
	audit, err := read(root + "/audit.json")
	if err != nil || !bytes.Equal(raw, audit) {
		return nil, errors.New("native pilot audit differs")
	}
	names := []string{"studies/compound-path-v1/cohort.jsonl", "studies/compound-path-v1/manifest.json", root + "/report.json", root + "/audit.json", root + "/preexecution.json"}
	nameRE := regexp.MustCompile(`^(captures/[a-z0-9_-]+|metrics/[a-z0-9_-]+|executions/[a-f0-9]{64})\.json$`)
	counts := map[string]int{}
	for path, pin := range report.Files {
		if !nameRE.MatchString(path) || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(pin) {
			return nil, errors.New("unsafe native pilot path/hash")
		}
		data, err := read(root + "/" + path)
		if err != nil || hash(data) != pin {
			return nil, errors.New("native pilot evidence changed")
		}
		counts[strings.Split(path, "/")[0]]++
		names = append(names, root+"/"+path)
	}
	if counts["captures"] != 48 || counts["metrics"] != 48 || counts["executions"] != report.Executions {
		return nil, errors.New("native pilot path categories differ")
	}
	sort.Strings(names)
	return names, nil
}
func checkArchive(raw []byte, entries []entry) error {
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer z.Close()
	t := tar.NewReader(io.LimitReader(z, maxTotal+2<<20))
	for _, e := range entries {
		h, err := t.Next()
		if err != nil || h.Name != e.Path || h.Typeflag != tar.TypeReg || h.Size != int64(e.Bytes) {
			return errors.New("archive entry differs")
		}
		data, err := io.ReadAll(io.LimitReader(t, int64(e.Bytes)+1))
		if err != nil || len(data) != e.Bytes || hash(data) != e.SHA || privateText(data) {
			return errors.New("archive bytes differ")
		}
	}
	if _, err := t.Next(); err != io.EOF {
		return errors.New("extra archive entry")
	}
	return nil
}
func execute(bundle, revision, output string, pack bool) error {
	return executeKind(bundle, revision, output, pack, "")
}
func executeKind(bundle, revision, output string, pack bool, kind string) error {
	m, files, err := buildKind(kind)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	files["publication-manifest.json"] = append(raw, '\n')
	if err = checkArchive(files["evidence.tar.gz"], m.Raw); err != nil {
		return err
	}
	if pack {
		if _, err = os.Lstat(bundle); !os.IsNotExist(err) {
			return errors.New("fresh bundle required")
		}
		for name, data := range files {
			path := filepath.Join(bundle, name)
			if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err = os.WriteFile(path, data, 0644); err != nil {
				return err
			}
		}
		return nil
	}
	count, err := checkLocal(bundle, files)
	if err != nil {
		return err
	}
	gets := 0
	if revision != "" {
		if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
			return errors.New("immutable revision required")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		client := &http.Client{Timeout: 30 * time.Second}
		fetch := func(url string, limit int) ([]byte, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			gets++
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New("anonymous publication unavailable")
			}
			data, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
			if err != nil || len(data) > limit {
				return nil, errors.New("HTTP budget exceeded")
			}
			return data, nil
		}
		api, err := fetch("https://huggingface.co/api/models/"+repository+"/revision/"+revision, 1<<20)
		if err != nil {
			return err
		}
		var identity struct {
			SHA     string `json:"sha"`
			Private bool   `json:"private"`
		}
		if json.Unmarshal(api, &identity) != nil || identity.SHA != revision || identity.Private {
			return errors.New("public revision differs")
		}
		var sorted []string
		for name := range files {
			sorted = append(sorted, name)
		}
		sort.Strings(sorted)
		for _, name := range sorted {
			data, err := fetch("https://huggingface.co/"+repository+"/resolve/"+revision+"/"+m.Prefix+name, len(files[name]))
			if err != nil || !bytes.Equal(data, files[name]) {
				return errors.New("anonymous immutable payload differs")
			}
		}
	}
	value := map[string]any{"schema": "gooo/native-family-publication-verification/v1", "decision": "PASS", "repository": repository, "revision": revision, "prefix": m.Prefix, "files_verified": count, "archive_entries_verified": len(m.Raw), "archive_bytes": len(files["evidence.tar.gz"]), "manifest_sha256": hash(files["publication-manifest.json"]), "anonymous_http_gets": gets, "credentials_sent": false, "new_model_predictions": 0, "new_native_calls": 0, "new_go_processes": 0}
	if kind == "compound-main" {
		value["schema"] = "gooo/compound-path-publication-verification/v1"
	} else if kind == "unfixed-sdk" {
		value["schema"] = "gooo/unfixed-feedback-publication-verification/v1"
	}
	if kind == "unfixed-native-feature" || kind == "unfixed-native-main" {
		value["schema"] = "gooo/native-unfixed-publication-verification/v1"
	}
	if kind == "retained-native-feature" {
		value["schema"] = "gooo/retained-native-publication-verification/v1"
	}
	if output == "" {
		return errors.New("verification output required")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(data, '\n'), 0644)
}
func main() {
	bundle := flag.String("bundle", "", "fresh/local bundle")
	revision := flag.String("revision", "", "immutable public revision")
	output := flag.String("output", "", "verification record")
	pack := flag.Bool("pack", false, "build deterministic archive")
	kind := flag.String("kind", "", "frozen evidence kind: empty, optimized-feature, optimized-main, compound-main, unfixed-sdk or unfixed-native-feature/main")
	flag.Parse()
	if flag.NArg() != 0 || *bundle == "" {
		fmt.Fprintln(os.Stderr, "package-family-evidence: bundle required")
		os.Exit(1)
	}
	if err := executeKind(*bundle, *revision, *output, *pack, *kind); err != nil {
		fmt.Fprintln(os.Stderr, "package-family-evidence:", err)
		os.Exit(1)
	}
}

func checkLocal(bundle string, files map[string][]byte) (int, error) {
	count := 0
	err := filepath.WalkDir(bundle, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink rejected")
		}
		if d.IsDir() {
			return nil
		}
		name, err := filepath.Rel(bundle, path)
		if err != nil {
			return err
		}
		expected, ok := files[name]
		if !ok {
			return errors.New("extra payload rejected")
		}
		i, err := os.Lstat(path)
		if err != nil || !i.Mode().IsRegular() || i.Size() != int64(len(expected)) {
			return errors.New("payload size differs")
		}
		data, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, expected) {
			return errors.New("payload bytes differ")
		}
		count++
		return nil
	})
	if err != nil || count != len(files) {
		return count, errors.New("local payload inventory differs")
	}
	return count, nil
}
