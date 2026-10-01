// package-feedback-model publishes only explicit synthetic experiment files.
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
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const repository = "asketeddy/gooo-feedback-path-tiny-v1"
const trainingRoot = "runs/feedback-path-soft-target-mps-20261001"
const nativeRoot = "runs/feedback-trained-native-20261001"
const maxFile = 16 << 20

var privacy = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)
var modelPins = map[string]string{
	"fp32":        "47bd3ed2037c8ba0af31ec4cad3a47fe1182af171845406aba01c5107f4b24b7",
	"ptq_ternary": "a7b66676af2eb5a7e6952f3c65c812f231865ac4974a3c3c1f329b125e55124c",
	"qat_ternary": "17dd98b2c05c7a9c203523a2e8e70ba0cbb523ae47715fdc0dc42acef81b0536",
}
var fixed = map[string]string{
	"data/dataset.jsonl":     "570d1d73bfea32662b869e5cfbf77e6d04df838b703191b076be58c1d2180dbf",
	"training/go-audit.json": "6b013256fb4d9ce7d044bcb640e15719de5908b21debc0b15be4cf4c7fb714c3",
	"native/report.json":     "cda64b53a48ff769a21319084a23239c26667e95dda449c89fa29f16a859303f",
	"native/audit.json":      "0a41bd75583097962b607b0afd2a8fee689bf8e60cb41e28c22ae5af79696741",
}

func inventory() map[string]string {
	files := map[string]string{"README.md": "docs/model-card-feedback-path-v1.md", "LICENSE": "LICENSE",
		"data/dataset.jsonl": "data/feedback-path-v1/dataset.jsonl", "data/manifest.json": "data/feedback-path-v1/manifest.json"}
	for variant := range modelPins {
		for _, name := range []string{"model.json", "weights.bin"} {
			files["models/"+variant+"/"+name] = trainingRoot + "/models/" + variant + "/" + name
		}
	}
	for _, name := range []string{"preexecution.json", "report.json", "go-parity.json", "go-audit.json"} {
		files["training/"+name] = trainingRoot + "/" + name
	}
	for _, name := range []string{"preexecution.json", "report.json", "audit.json", "en-fp32.json", "en-ptq_ternary.json", "en-qat_ternary.json", "ko-fp32.json", "ko-ptq_ternary.json", "ko-qat_ternary.json",
		"executions/0db2c6f0da2b2fbed66e56a7f5d9ae20f628167df67500b54ad73072d3cd1c72.json"} {
		files["native/"+name] = nativeRoot + "/" + name
	}
	return files
}

type entry struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema     string            `json:"schema"`
	Repository string            `json:"repository"`
	Models     map[string]string `json:"model_metadata_sha256"`
	Files      []entry           `json:"files"`
	Scope      string            `json:"scope"`
}

func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func read(path string) ([]byte, error) {
	i, err := os.Lstat(path)
	if err != nil || !i.Mode().IsRegular() || i.Size() <= 0 || i.Size() > maxFile {
		return nil, errors.New("bounded regular publication file required")
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
func sources() (manifest, error) {
	value := manifest{Schema: "gooo/feedback-path-model-publication/v1", Repository: repository, Models: modelPins,
		Scope: "New experimental own-model weights and synthetic bilingual finite-feedback data/evidence. Negative parent comparison retained; feature native evidence is separate from main deployment. No upstream Laya weights, credentials or private host material."}
	for variant, pin := range modelPins {
		model, err := decision.LoadPath(trainingRoot + "/models/" + variant + "/model.json")
		if err != nil || model.MetadataSHA256() != pin {
			return value, errors.New("frozen trained model metadata or weight digest differs")
		}
	}
	files := inventory()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	total := 0
	for _, name := range names {
		raw, err := read(files[name])
		if err != nil {
			return value, err
		}
		if !strings.HasSuffix(name, "/weights.bin") && privacy.Match(raw) {
			return value, errors.New("private publication text rejected")
		}
		if pin := fixed[name]; pin != "" && hash(raw) != pin {
			return value, errors.New("fixed evidence digest differs")
		}
		total += len(raw)
		if total > 20<<20 {
			return value, errors.New("publication exceeds total budget")
		}
		value.Files = append(value.Files, entry{name, hash(raw), len(raw)})
	}
	return value, nil
}
func pack(bundle string) error {
	if bundle == "" {
		return errors.New("bundle path required")
	}
	if _, err := os.Lstat(bundle); !os.IsNotExist(err) {
		return errors.New("fresh publication directory required")
	}
	value, err := sources()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(bundle, 0755); err != nil {
		return err
	}
	for _, e := range value.Files {
		raw, err := read(inventory()[e.Path])
		if err != nil || hash(raw) != e.SHA {
			return errors.New("publication source changed during packaging")
		}
		target := filepath.Join(bundle, e.Path)
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(target, raw, 0644); err != nil {
			return err
		}
	}
	return save(filepath.Join(bundle, "publication-manifest.json"), value)
}
func verify(bundle, revision, output string) error {
	if revision != "" && !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable HF revision required")
	}
	expected, err := sources()
	if err != nil {
		return err
	}
	raw, err := read(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var actual manifest
	if err = json.Unmarshal(raw, &actual); err != nil || !reflect.DeepEqual(actual, expected) {
		return errors.New("manifest differs from fixed source allowlist")
	}
	allowed := map[string]entry{}
	for _, e := range actual.Files {
		allowed[e.Path] = e
	}
	allowed["publication-manifest.json"] = entry{"publication-manifest.json", hash(raw), len(raw)}
	count := 0
	if err = filepath.WalkDir(bundle, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("publication symlink rejected")
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(bundle, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		e, ok := allowed[rel]
		if !ok {
			return errors.New("unlisted publication file rejected")
		}
		data, err := read(path)
		if err != nil || len(data) != e.Bytes || hash(data) != e.SHA {
			return errors.New("publication bytes differ")
		}
		count++
		return nil
	}); err != nil {
		return err
	}
	if count != len(allowed) {
		return errors.New("missing publication file")
	}
	requests := 0
	if revision != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		client := &http.Client{Timeout: 20 * time.Second}
		fetch := func(url string) ([]byte, error) {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			response, err := client.Do(request)
			requests++
			if err != nil {
				return nil, errors.New("anonymous public request failed")
			}
			defer response.Body.Close()
			data, err := io.ReadAll(io.LimitReader(response.Body, maxFile+1))
			if err != nil || response.StatusCode != http.StatusOK || len(data) > maxFile {
				return nil, errors.New("bounded public response required")
			}
			return data, nil
		}
		metaRaw, err := fetch("https://huggingface.co/api/models/" + repository + "/revision/" + revision)
		if err != nil {
			return err
		}
		var meta struct {
			ID      string `json:"id"`
			SHA     string `json:"sha"`
			Private *bool  `json:"private"`
		}
		if err = json.Unmarshal(metaRaw, &meta); err != nil || meta.ID != repository || meta.SHA != revision || meta.Private == nil || *meta.Private {
			return errors.New("public immutable repository identity differs")
		}
		names := make([]string, 0, len(allowed))
		for name := range allowed {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			data, err := fetch("https://huggingface.co/" + repository + "/resolve/" + revision + "/" + name)
			if err != nil || len(data) != allowed[name].Bytes || hash(data) != allowed[name].SHA {
				return errors.New("anonymous immutable publication digest differs")
			}
		}
	}
	return save(output, map[string]any{"schema": "gooo/feedback-path-model-verification/v1", "decision": "PASS", "repository": repository, "revision": revision, "publication_files": count, "anonymous_requests": requests, "credentials_sent": false, "manifest_sha256": hash(raw), "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_steps": 0})
}
func main() {
	mode := flag.String("mode", "pack", "pack or verify")
	bundle := flag.String("bundle", "", "allowlisted bundle directory")
	revision := flag.String("revision", "", "immutable public revision")
	output := flag.String("output", "", "verification receipt")
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
