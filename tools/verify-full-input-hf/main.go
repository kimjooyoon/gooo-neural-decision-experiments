// Verify anonymous immutable downloads without retaining another model cache.
package main

import (
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
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func main() {
	revision := flag.String("revision", "", "immutable public Hugging Face revision")
	output := flag.String("output", "", "fresh transport verification report")
	separate := flag.Bool("include-separate", false, "also verify the explicit arithmetic appendix")
	flag.Parse()
	if err := verify(*revision, *output, *separate); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verify(revision, output string, includeSeparate bool) error {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(revision) || output == "" {
		return errors.New("immutable revision and fresh output required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh report required")
	}
	const bundle = "publication/full-input-initial-study-20261003"
	const appendix = "research/full-input-initial-20261003/"
	const repo = "asketeddy/gooo-shared-judgment-tiny-v1"
	raw, err := os.ReadFile(bundle + "/manifest.json")
	if err != nil {
		return err
	}
	var manifest struct {
		Files map[string]threestudent.Pin `json:"files"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil || len(manifest.Files) != 57 {
		return errors.New("complete local manifest required")
	}
	files := map[string]threestudent.Pin{}
	for name, pin := range manifest.Files {
		if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
			return errors.New("local member name required")
		}
		files[appendix+name] = pin
	}
	for remote, local := range map[string]string{appendix + "manifest.json": bundle + "/manifest.json", "README.md": "publication/shared-judgment-model-card-20261003.md"} {
		pin, e := threestudent.FilePin(local)
		if e != nil {
			return e
		}
		files[remote] = pin
	}
	const platformBundle = "publication/full-input-platform-diagnosis-20261003"
	const platformAppendix = "research/full-input-platform-20261003/"
	platformRaw, err := os.ReadFile(platformBundle + "/manifest.json")
	if err != nil {
		return err
	}
	var platform struct {
		Files map[string]threestudent.Pin `json:"files"`
	}
	if err = json.Unmarshal(platformRaw, &platform); err != nil || len(platform.Files) != 4 {
		return errors.New("complete platform manifest required")
	}
	for name, pin := range platform.Files {
		if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
			return errors.New("local platform member required")
		}
		files[platformAppendix+name] = pin
	}
	platformPin, err := threestudent.FilePin(platformBundle + "/manifest.json")
	if err != nil {
		return err
	}
	files[platformAppendix+"manifest.json"] = platformPin
	if includeSeparate {
		const root = "publication/full-input-separate-arithmetic-20261003"
		const prefix = "research/full-input-separate-20261003/"
		encoded, err := os.ReadFile(root + "/manifest.json")
		if err != nil {
			return err
		}
		var separate struct {
			Files map[string]threestudent.Pin `json:"files"`
		}
		if err = json.Unmarshal(encoded, &separate); err != nil || len(separate.Files) != 56 {
			return errors.New("complete separate manifest required")
		}
		for name, pin := range separate.Files {
			if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
				return errors.New("local separate member required")
			}
			files[prefix+name] = pin
		}
		pin, err := threestudent.FilePin(root + "/manifest.json")
		if err != nil {
			return err
		}
		files[prefix+"manifest.json"] = pin
	}
	client := &http.Client{Timeout: 120 * time.Second}
	metadata, err := client.Get("https://huggingface.co/api/models/" + repo + "/revision/" + revision)
	if err != nil {
		return err
	}
	var remoteInfo struct {
		SHA     string `json:"sha"`
		Private bool   `json:"private"`
	}
	if metadata.StatusCode != 200 {
		metadata.Body.Close()
		return errors.New("public revision metadata unavailable")
	}
	err = json.NewDecoder(io.LimitReader(metadata.Body, 2<<20)).Decode(&remoteInfo)
	metadata.Body.Close()
	if err != nil || remoteInfo.SHA != revision || remoteInfo.Private {
		return errors.New("public immutable revision required")
	}
	started := time.Now()
	var total int64
	for name, expected := range files {
		response, err := client.Get("https://huggingface.co/" + repo + "/resolve/" + revision + "/" + name)
		if err != nil {
			return err
		}
		if response.StatusCode != 200 {
			response.Body.Close()
			return fmt.Errorf("public file %s: HTTP %d", name, response.StatusCode)
		}
		digest := sha256.New()
		n, err := io.Copy(digest, io.LimitReader(response.Body, expected.Bytes+1))
		closeErr := response.Body.Close()
		if err != nil || closeErr != nil || n != expected.Bytes || hex.EncodeToString(digest.Sum(nil)) != expected.SHA {
			return fmt.Errorf("public bytes differ: %s", name)
		}
		total += n
	}
	source, err := threestudent.FilePin("tools/verify-full-input-hf/main.go")
	if err != nil {
		return err
	}
	value := map[string]any{"schema": "gooo/full-input-hf-transport/v1", "status": "PASS", "repository": repo,
		"revision": revision, "private": false, "anonymous": true, "public_files_verified": len(files),
		"downloaded_bytes": total, "wall_seconds": time.Since(started).Seconds(), "files": files,
		"local_bundle_manifest_sha256": threecohort.SHA(raw), "local_platform_manifest_sha256": threecohort.SHA(platformRaw), "verifier_source_sha256": source.SHA,
		"go_version": runtime.Version(), "new_optimizer_updates": 0, "new_model_predictions": 0,
		"scope": "Anonymous immutable downloads match the closed local bundle and current card byte-for-byte. No duplicate download cache retained."}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(encoded, '\n'))
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.Join(err, closeErr)
	}
	fmt.Printf("{\"status\":\"PASS\",\"files\":%d,\"bytes\":%d}\n", len(files), total)
	return nil
}
