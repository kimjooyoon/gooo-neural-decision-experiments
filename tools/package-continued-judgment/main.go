// package-continued-judgment publishes an allowlisted synthetic repair appendix.
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
const prefix = "research/continued-judgment-20261001/"
const reportSHA = "a5ceff7c5d373333a89e301e23a86a601a73d4fd39eaa28d093a9cdcb84d72fd"
const auditSHA = "1f100799ee9caa4637d359c2d7fd270121f88c2d704468e6ffa2a3f243cfff07"
const coreSHA = "a320117c9d8567cbf0f15df0fb92fc0245c84787a49e1dff9f4cb775117894e2"

var privateText = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)
var files = map[string]string{
	"README.md":         "docs/continued-judgment-results.md",
	"report.json":       "runs/feedback-context-decline-20261001/report.json",
	"preexecution.json": "runs/feedback-context-decline-20261001/preexecution.json",
	"audit.json":        "runs/feedback-context-decline-20261001/audit.json",
	"en-baseline.json":  "runs/feedback-context-decline-20261001/en-baseline.json",
	"en-feedback.json":  "runs/feedback-context-decline-20261001/en-feedback.json",
	"ko-baseline.json":  "runs/feedback-context-decline-20261001/ko-baseline.json",
	"ko-feedback.json":  "runs/feedback-context-decline-20261001/ko-feedback.json",
	"go-execution.json": "runs/feedback-context-decline-20261001/go-execution.json",
	"en-plan.json":      "runs/feedback-input-bound-20261001/en-plan.json",
	"ko-plan.json":      "runs/feedback-input-bound-20261001/ko-plan.json",
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
		Calls    int    `json:"recorded_local_predictions"`
	}
	raw, err := read(files["audit.json"])
	if err != nil || hash(raw) != auditSHA {
		return errors.New("fixed native audit required")
	}
	if err = json.Unmarshal(raw, &audit); err != nil || audit.Decision != "PASS" || audit.Report != reportSHA || audit.Calls != 24 {
		return errors.New("audit did not verify fixed study")
	}
	value := manifest{Schema: "gooo/continued-judgment-evidence-allowlist/v1", Core: coreSHA, Scope: "Separate public synthetic continued-construction evidence. No model weights, private host paths or credentials. Existing model card, weights and frozen evidence editions remain unchanged."}
	if err = os.Mkdir(bundle, 0755); err != nil {
		return err
	}
	for _, name := range []string{"README.md", "audit.json", "en-baseline.json", "en-feedback.json", "en-plan.json", "go-execution.json", "ko-baseline.json", "ko-feedback.json", "ko-plan.json", "preexecution.json", "report.json"} {
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
	if err = json.Unmarshal(manifestRaw, &value); err != nil || value.Schema != "gooo/continued-judgment-evidence-allowlist/v1" || len(value.Files) != 11 || value.Weights || value.Core != coreSHA {
		return errors.New("fixed appendix inventory required")
	}
	entries, err := os.ReadDir(bundle)
	if err != nil || len(entries) != 12 {
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
	return save(output, map[string]any{"schema": "gooo/continued-judgment-evidence-verification/v1", "decision": "PASS", "repository": repository, "revision": revision, "appendix_files": 12, "manifest_sha256": hash(manifestRaw), "anonymous_requests": requests, "credentials_sent": false, "new_model_predictions": 0, "scope": "Twelve allowlisted synthetic appendix files and unchanged core manifest. Prior weights, metadata, model card and earlier appendices are separately verified at the same immutable revision."})
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
