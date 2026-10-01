// package-feedback-pilot publishes only a separate synthetic evidence appendix.
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

const repo = "asketeddy/gooo-typed-path-tiny-v1"
const prefix = "research/feedback-path-pilot-20261001/"

var privateText = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)
var sources = map[string]string{
	"report.json":            "runs/feedback-path-pilot-fixed-20261001/report.json",
	"preexecution.json":      "runs/feedback-path-pilot-fixed-20261001/preexecution.json",
	"source-input-pins.json": "runs/feedback-path-pilot-fixed-20261001/source-input-pins.json",
	"audit.json":             "runs/feedback-path-pilot-fixed-20261001/audit.json",
	"README.md":              "docs/feedback-path-pilot-results.md",
}

type file struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema         string `json:"schema"`
	Files          []file `json:"files"`
	OldManifest    string `json:"unchanged_core_manifest_sha256"`
	WeightsChanged bool   `json:"weights_changed"`
	Scope          string `json:"scope"`
}

func hash(raw []byte) string { v := sha256.Sum256(raw); return hex.EncodeToString(v[:]) }
func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 5<<20 {
		return nil, errors.New("bounded regular public artifact required")
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
		return errors.New("new appendix directory required")
	}
	var audit struct {
		Decision string `json:"decision"`
		Report   string `json:"report_sha256"`
		Calls    int    `json:"actual_model_predictions"`
		Feedback int    `json:"feedback_model_predictions"`
	}
	raw, err := read(sources["audit.json"])
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &audit); err != nil || audit.Decision != "PASS" || audit.Report != "050f288116b973aeb5198d35fcf9c085b178189868243eceb67a86cb0d32048e" || audit.Calls != 246 || audit.Feedback != 102 {
		return errors.New("fixed independently audited study required")
	}
	if err = os.Mkdir(bundle, 0755); err != nil {
		return err
	}
	value := manifest{Schema: "gooo/public-feedback-pilot-allowlist/v1", OldManifest: "a320117c9d8567cbf0f15df0fb92fc0245c84787a49e1dff9f4cb775117894e2", Scope: "Separate evidence appendix to unchanged v5 weights and core files. One fixed compound intent; not model training or general language accuracy. Raw native/Go captures and source live in the public GitHub research repository."}
	for _, name := range []string{"README.md", "audit.json", "preexecution.json", "report.json", "source-input-pins.json"} {
		raw, err := read(sources[name])
		if err != nil {
			return err
		}
		if privateText.Match(raw) {
			return errors.New("private text in appendix")
		}
		if name == "report.json" && hash(raw) != audit.Report {
			return errors.New("changed study report")
		}
		if err = os.WriteFile(filepath.Join(bundle, name), raw, 0644); err != nil {
			return err
		}
		value.Files = append(value.Files, file{name, hash(raw), len(raw)})
	}
	return save(filepath.Join(bundle, "publication-manifest.json"), value)
}
func verify(bundle, revision, output string) error {
	if revision != "" && !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable lowercase revision required")
	}
	manifestRaw, err := read(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var value manifest
	if err = json.Unmarshal(manifestRaw, &value); err != nil || value.Schema != "gooo/public-feedback-pilot-allowlist/v1" || len(value.Files) != 5 || value.WeightsChanged || value.OldManifest != "a320117c9d8567cbf0f15df0fb92fc0245c84787a49e1dff9f4cb775117894e2" {
		return errors.New("appendix manifest differs")
	}
	entries, err := os.ReadDir(bundle)
	if err != nil || len(entries) != 6 {
		return errors.New("appendix inventory differs")
	}
	seen := map[string]bool{}
	for _, entry := range value.Files {
		if seen[entry.Path] || sources[entry.Path] == "" {
			return errors.New("unknown appendix path")
		}
		seen[entry.Path] = true
		raw, err := read(filepath.Join(bundle, entry.Path))
		if err != nil || hash(raw) != entry.SHA || len(raw) != entry.Bytes || privateText.Match(raw) {
			return errors.New("appendix hash or privacy check failed")
		}
	}
	requests := 0
	verifiedWeights := 0
	verifiedMetadata := 0
	verifiedCard := false
	if revision != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		client := &http.Client{Timeout: 20 * time.Second}
		get := func(url string) ([]byte, error) {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			response, err := client.Do(request)
			requests++
			if err != nil {
				return nil, err
			}
			defer response.Body.Close()
			if response.StatusCode != 200 {
				return nil, errors.New("public HTTP response failed")
			}
			raw, err := io.ReadAll(io.LimitReader(response.Body, 5<<20+1))
			if err != nil || len(raw) > 5<<20 {
				return nil, errors.New("public response bound exceeded")
			}
			return raw, nil
		}
		infoRaw, err := get("https://huggingface.co/api/models/" + repo + "/revision/" + revision)
		if err != nil {
			return err
		}
		var info struct {
			SHA     string `json:"sha"`
			Private bool   `json:"private"`
		}
		if err = json.Unmarshal(infoRaw, &info); err != nil || info.SHA != revision || info.Private {
			return errors.New("public immutable model revision required")
		}
		for _, entry := range append(append([]file(nil), value.Files...), file{"publication-manifest.json", hash(manifestRaw), len(manifestRaw)}) {
			raw, err := get("https://huggingface.co/" + repo + "/resolve/" + revision + "/" + prefix + entry.Path)
			if err != nil || hash(raw) != entry.SHA || len(raw) != entry.Bytes {
				return errors.New("public appendix differs")
			}
		}
		coreRaw, err := get("https://huggingface.co/" + repo + "/resolve/" + revision + "/publication-manifest.json")
		if err != nil || hash(coreRaw) != value.OldManifest {
			return errors.New("core v5 manifest changed")
		}
		var core struct {
			Files []file `json:"files"`
		}
		if err = json.Unmarshal(coreRaw, &core); err != nil {
			return err
		}
		for _, entry := range core.Files {
			if filepath.Base(entry.Path) != "weights.bin" {
				continue
			}
			raw, err := get("https://huggingface.co/" + repo + "/resolve/" + revision + "/" + entry.Path)
			if err != nil || hash(raw) != entry.SHA || len(raw) != entry.Bytes {
				return errors.New("previous weights changed")
			}
			verifiedWeights++
		}
		if verifiedWeights != 9 {
			return errors.New("nine old weight files required")
		}
		for _, entry := range core.Files {
			if filepath.Base(entry.Path) != "model.json" && entry.Path != "README.md" {
				continue
			}
			raw, err := get("https://huggingface.co/" + repo + "/resolve/" + revision + "/" + entry.Path)
			if err != nil || hash(raw) != entry.SHA || len(raw) != entry.Bytes {
				return errors.New("previous metadata or model card changed")
			}
			if entry.Path == "README.md" {
				verifiedCard = true
			} else {
				verifiedMetadata++
			}
		}
		if verifiedMetadata != 9 || !verifiedCard {
			return errors.New("nine model metadata files and prior model card required")
		}
	}
	result := map[string]any{"schema": "gooo/public-feedback-pilot-verification/v1", "decision": "PASS", "repository": repo, "revision": revision, "appendix_files_verified": 6, "appendix_manifest_sha256": hash(manifestRaw), "anonymous_requests": requests, "credentials_sent": false, "previous_weights_verified": verifiedWeights, "previous_model_metadata_verified": verifiedMetadata, "previous_model_card_verified": verifiedCard, "new_model_predictions": 0, "scope": "Local appendix digests/privacy and, when revision is supplied, six public appendix files, unchanged core manifest, all nine old weight/metadata files and prior model card. This does not re-download every previous v5 evidence file."}
	return save(output, result)
}
func main() {
	mode := flag.String("mode", "package", "package or verify")
	bundle := flag.String("bundle", "publication/hf-feedback-path-pilot-20261001", "fixed synthetic appendix directory")
	revision := flag.String("revision", "", "optional immutable public HF revision")
	output := flag.String("output", "", "verification receipt")
	flag.Parse()
	var err error
	if *mode == "package" {
		err = pack(*bundle)
	} else if *mode == "verify" && *output != "" {
		err = verify(*bundle, *revision, *output)
	} else {
		err = errors.New("invalid mode or missing verification output")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
