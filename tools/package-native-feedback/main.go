// package-native-feedback publishes a small allowlisted synthetic evidence appendix.
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
	"time"
)

const repository = "asketeddy/gooo-typed-path-tiny-v1"
const prefix = "research/native-feedback-integration-20261001/"
const reportSHA = "f86a102955d5b8484a56c291d36d1a5af2a3c34d4a1b02acadc363e3912ca415"
const auditSHA = "0cf7940060235f789f717ac44e088cbbda2027939d082f52ab801932e1e0a2cd"
const coreSHA = "a320117c9d8567cbf0f15df0fb92fc0245c84787a49e1dff9f4cb775117894e2"

var privateText = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)
var files = map[string]string{
	"README.md":         "docs/native-feedback-results.md",
	"report.json":       "runs/native-feedback-integration-fixed-20261001/report.json",
	"preexecution.json": "runs/native-feedback-integration-fixed-20261001/preexecution.json",
	"audit.json":        "runs/native-feedback-integration-fixed-20261001/audit.json",
}

type entry struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema  string  `json:"schema"`
	Files   []entry `json:"files"`
	Core    string  `json:"unchanged_core_manifest_sha256"`
	Weights bool    `json:"weights_changed"`
	Scope   string  `json:"scope"`
}

func hash(raw []byte) string { value := sha256.Sum256(raw); return hex.EncodeToString(value[:]) }
func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 5<<20 {
		return nil, errors.New("bounded regular evidence file required")
	}
	return os.ReadFile(path)
}
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func pack(bundle string) error {
	if _, err := os.Lstat(bundle); !os.IsNotExist(err) {
		return errors.New("fresh bundle required")
	}
	var audit struct {
		Decision string `json:"decision"`
		Report   string `json:"report_sha256"`
		Calls    int    `json:"actual_local_predictions"`
	}
	raw, err := read(files["audit.json"])
	if err != nil || hash(raw) != auditSHA {
		return errors.New("fixed native audit required")
	}
	if err = json.Unmarshal(raw, &audit); err != nil || audit.Decision != "PASS" || audit.Report != reportSHA || audit.Calls != 246 {
		return errors.New("audit did not verify fixed study")
	}
	value := manifest{Schema: "gooo/native-feedback-evidence-allowlist/v1", Core: coreSHA, Scope: "Separate public synthetic native-integration evidence. No model weights, private host paths or credentials. Existing model card, weights and frozen evidence editions remain unchanged."}
	if err = os.Mkdir(bundle, 0755); err != nil {
		return err
	}
	for _, name := range []string{"README.md", "audit.json", "preexecution.json", "report.json"} {
		raw, err := read(files[name])
		if err != nil {
			return err
		}
		if privateText.Match(raw) || (name == "report.json" && hash(raw) != reportSHA) {
			return errors.New("changed or private appendix material")
		}
		if err = os.WriteFile(filepath.Join(bundle, name), raw, 0644); err != nil {
			return err
		}
		value.Files = append(value.Files, entry{name, hash(raw), len(raw)})
	}
	return save(filepath.Join(bundle, "publication-manifest.json"), value)
}
func verify(bundle, revision, output string) error {
	if revision != "" && !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable lowercase HF revision required")
	}
	manifestRaw, err := read(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var value manifest
	if err = json.Unmarshal(manifestRaw, &value); err != nil || value.Schema != "gooo/native-feedback-evidence-allowlist/v1" || len(value.Files) != 4 || value.Weights || value.Core != coreSHA {
		return errors.New("fixed appendix inventory required")
	}
	entries, err := os.ReadDir(bundle)
	if err != nil || len(entries) != 5 {
		return errors.New("extra or missing appendix file")
	}
	seen := map[string]bool{}
	for _, entry := range value.Files {
		if seen[entry.Path] || files[entry.Path] == "" {
			return errors.New("duplicate or unknown appendix file")
		}
		seen[entry.Path] = true
		raw, err := read(filepath.Join(bundle, entry.Path))
		if err != nil || privateText.Match(raw) || len(raw) != entry.Bytes || hash(raw) != entry.SHA {
			return errors.New("local evidence bytes or privacy differ")
		}
		if (entry.Path == "report.json" && entry.SHA != reportSHA) || (entry.Path == "audit.json" && entry.SHA != auditSHA) {
			return errors.New("fixed report or audit differs")
		}
	}
	requests := 0
	if revision != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		client := &http.Client{Timeout: 20 * time.Second}
		check := func(name, digest string) error {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://huggingface.co/"+repository+"/resolve/"+revision+"/"+name, nil)
			if err != nil {
				return err
			}
			response, err := client.Do(request)
			requests++
			if err != nil {
				return errors.New("anonymous public fetch failed")
			}
			defer response.Body.Close()
			raw, err := io.ReadAll(io.LimitReader(response.Body, (5<<20)+1))
			if err != nil || response.StatusCode != http.StatusOK || len(raw) > 5<<20 || hash(raw) != digest {
				return errors.New("anonymous immutable evidence bytes differ")
			}
			return nil
		}
		for _, entry := range value.Files {
			if err = check(prefix+entry.Path, entry.SHA); err != nil {
				return err
			}
		}
		if err = check(prefix+"publication-manifest.json", hash(manifestRaw)); err != nil {
			return err
		}
		if err = check("publication-manifest.json", coreSHA); err != nil {
			return err
		}
	}
	return save(output, map[string]any{"schema": "gooo/native-feedback-evidence-verification/v1", "decision": "PASS", "repository": repository, "revision": revision, "appendix_files": 5, "manifest_sha256": hash(manifestRaw), "anonymous_requests": requests, "credentials_sent": false, "new_model_predictions": 0, "scope": "Five allowlisted appendix files and unchanged core manifest. Prior weights, metadata, model card and earlier appendix are separately verified at the same immutable revision with package-feedback-pilot."})
}
func main() {
	mode := flag.String("mode", "pack", "pack or verify")
	bundle := flag.String("bundle", "", "new bundle or existing allowlisted bundle")
	revision := flag.String("revision", "", "immutable public HF revision")
	output := flag.String("output", "", "verification output")
	flag.Parse()
	var err error
	if *mode == "pack" {
		err = pack(*bundle)
	} else if *mode == "verify" {
		err = verify(*bundle, *revision, *output)
	} else {
		err = errors.New("unknown mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
