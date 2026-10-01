// Explicit own-model publication allowlist and anonymous digest verification.
package main

import (
	"archive/tar"
	"compress/gzip"
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

type artifact struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema  string     `json:"schema"`
	Repo    string     `json:"repo"`
	Prefix  string     `json:"prefix"`
	Files   []artifact `json:"files"`
	Archive []artifact `json:"archive_entries"`
}

var privateText = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)

func hash(raw []byte) string { v := sha256.Sum256(raw); return hex.EncodeToString(v[:]) }
func read(name string) ([]byte, error) {
	i, e := os.Lstat(name)
	if e != nil || !i.Mode().IsRegular() || i.Size() > 32<<20 {
		return nil, errors.New("bounded regular public artifact required")
	}
	return os.ReadFile(name)
}
func save(name string, value any) error {
	raw, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}

func allowlist() (map[string]string, []string, error) {
	files := map[string]string{"README.md": "docs/paired-compiler-context-results-20261002.md", "LICENSE": "LICENSE",
		"protocol.md":                "docs/paired-compiler-context-preregistration-20261002.md",
		"training-report.json":       "runs/paired-compiler-context-mps-20261002/report.json",
		"training-preexecution.json": "runs/paired-compiler-context-mps-20261002/preexecution.json",
		"go-audit.json":              "runs/paired-compiler-context-go-audit-20261002/report.json",
		"go-preexecution.json":       "runs/paired-compiler-context-go-audit-20261002/preexecution.json",
		"selection.json":             "runs/paired-compiler-context-go-audit-20261002/selection.json",
		"native-dogfood.json":        "runs/paired-compiler-context-native-20261002/report.json",
		"native-preexecution.json":   "runs/paired-compiler-context-native-20261002/preexecution.json",
		"curriculum-manifest.json":   "runs/compiler-context-curriculum-validated-20261001/manifest.json"}
	for _, arm := range []string{"js0", "js01", "js03"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, name := range []string{"model.json", "weights.bin"} {
				files[filepath.Join("models", arm, variant, name)] = filepath.Join("runs/paired-compiler-context-mps-20261002", arm, "models", variant, name)
			}
		}
	}
	var archive []string
	for _, root := range []string{"runs/compiler-context-curriculum-validated-20261001", "runs/paired-compiler-context-mps-20261002",
		"runs/paired-compiler-context-go-audit-20261002", "runs/paired-compiler-context-native-20261002", "runs/paired-compiler-ci-rejected-20261002",
		"tools/native-feedback-study", "internal/decision", "internal/pathplan", "internal/pathstudy"} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink forbidden")
			}
			if !d.IsDir() {
				archive = append(archive, path)
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	for _, name := range []string{"training/train_paired_compiler_context_v1.py", "training/train_bilingual_judgment_v1.py", "training/train_pilot_v2.py",
		"training/split_features_v2.py", "training/train_typed_path_v1.py", "docs/paired-compiler-context-preregistration-20261002.md",
		"docs/paired-compiler-context-results-20261002.md", "go.mod"} {
		archive = append(archive, name)
	}
	sort.Strings(archive)
	return files, archive, nil
}

func pack(output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh publication directory required")
	}
	files, archive, err := allowlist()
	if err != nil {
		return err
	}
	all := map[string][]byte{}
	for _, name := range archive {
		raw, err := read(name)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(name, "weights.bin") && privateText.Match(raw) {
			return fmt.Errorf("private content in %s", name)
		}
		all[name] = raw
	}
	for _, name := range files {
		raw, err := read(name)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(name, "weights.bin") && privateText.Match(raw) {
			return fmt.Errorf("private content in %s", name)
		}
		all[name] = raw
	}
	for _, name := range []string{files["go-audit.json"], files["native-dogfood.json"]} {
		var value struct {
			Status string `json:"status"`
		}
		if err = json.Unmarshal(all[name], &value); err != nil || value.Status != "PASS" {
			return errors.New("passing frozen audit/dogfood required")
		}
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	value := manifest{Schema: "gooo/paired-compiler-public-appendix/v1", Repo: "asketeddy/gooo-compiler-context-tiny-v1", Prefix: "research/paired-compiler-20261002"}
	f, err := os.OpenFile(filepath.Join(output, "evidence.tar.gz"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, name := range archive {
		raw := all[name]
		if err = tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(raw)), ModTime: time.Unix(0, 0)}); err != nil {
			return err
		}
		if _, err = tw.Write(raw); err != nil {
			return err
		}
		value.Archive = append(value.Archive, artifact{name, hash(raw), len(raw)})
	}
	if err = tw.Close(); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	var paths []string
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		raw := all[files[path]]
		if err = os.MkdirAll(filepath.Dir(filepath.Join(output, path)), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, path), raw, 0644); err != nil {
			return err
		}
		value.Files = append(value.Files, artifact{path, hash(raw), len(raw)})
	}
	raw, err := read(filepath.Join(output, "evidence.tar.gz"))
	if err != nil {
		return err
	}
	value.Files = append(value.Files, artifact{"evidence.tar.gz", hash(raw), len(raw)})
	return save(filepath.Join(output, "publication-manifest.json"), value)
}

func get(url string) ([]byte, error) {
	client := http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anonymous GET status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (32<<20)+1))
	if len(raw) > 32<<20 {
		return nil, errors.New("bounded public response required")
	}
	return raw, err
}

func verify(bundle, revision, output string) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable HF revision required")
	}
	raw, err := read(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var value manifest
	if err = json.Unmarshal(raw, &value); err != nil || value.Schema != "gooo/paired-compiler-public-appendix/v1" || value.Repo != "asketeddy/gooo-compiler-context-tiny-v1" || value.Prefix != "research/paired-compiler-20261002" || len(value.Files) != 30 {
		return errors.New("own paired public allowlist differs")
	}
	api, err := get("https://huggingface.co/api/models/" + value.Repo + "/revision/" + revision)
	if err != nil {
		return err
	}
	var info struct {
		SHA     string `json:"sha"`
		Private bool   `json:"private"`
	}
	if err = json.Unmarshal(api, &info); err != nil || info.Private || info.SHA != revision {
		return errors.New("public immutable revision differs")
	}
	base := "https://huggingface.co/" + value.Repo + "/resolve/" + revision + "/" + value.Prefix + "/"
	remote, err := get(base + "publication-manifest.json")
	if err != nil || hash(remote) != hash(raw) {
		return errors.New("public appendix manifest differs")
	}
	for _, file := range value.Files {
		if filepath.Clean(file.Path) != file.Path || strings.Contains(file.Path, "..") || filepath.IsAbs(file.Path) {
			return errors.New("unsafe payload path")
		}
		remote, err = get(base + file.Path)
		if err != nil {
			return err
		}
		if hash(remote) != file.SHA || len(remote) != file.Bytes {
			return fmt.Errorf("public appendix hash differs: %s", file.Path)
		}
	}
	return save(output, map[string]any{"schema": "gooo/paired-compiler-public-verification/v1", "status": "PASS", "repo": value.Repo, "prefix": value.Prefix, "revision": revision, "manifest_sha256": hash(raw), "verified_payload_files": len(value.Files), "anonymous_payload_gets": len(value.Files) + 1, "api_discovery_gets": 1, "archive_entries": len(value.Archive), "scope": "anonymous exact-revision allowlist bytes, including all nine own models and raw source-bound evidence archive; no root model replacement"})
}

func main() {
	mode := flag.String("mode", "pack", "pack or verify")
	bundle := flag.String("bundle", "", "fresh appendix publication directory")
	revision := flag.String("revision", "", "immutable Hugging Face revision")
	output := flag.String("output", "", "public verification receipt")
	flag.Parse()
	var err error
	switch *mode {
	case "pack":
		err = pack(*bundle)
	case "verify":
		err = verify(*bundle, *revision, *output)
	default:
		err = errors.New("unknown publication mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
