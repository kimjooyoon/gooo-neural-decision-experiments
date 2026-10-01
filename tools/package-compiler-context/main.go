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
	files := map[string]string{"README.md": "docs/hf-compiler-context-tiny-v1-model-card.md", "LICENSE": "LICENSE",
		"results.md": "docs/compiler-context-training-results.md", "protocol.md": "docs/compiler-context-training-preregistration.md",
		"training-report.json":       "runs/compiler-context-training-mps-20261001/report.json",
		"training-preexecution.json": "runs/compiler-context-training-mps-20261001/preexecution.json",
		"go-audit.json":              "runs/compiler-context-model-audit-20261001/report.json",
		"native-dogfood.json":        "runs/compiler-context-native-dogfood-20261001/report.json",
		"curriculum-manifest.json":   "runs/compiler-context-curriculum-validated-20261001/manifest.json"}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		for _, name := range []string{"model.json", "weights.bin"} {
			path := filepath.Join("models", variant, name)
			files[path] = filepath.Join("runs/compiler-context-training-mps-20261001/compiler_context", path)
		}
	}
	var archive []string
	for _, root := range []string{"runs/compiler-context-curriculum-20261001", "runs/compiler-context-curriculum-validated-20261001",
		"runs/compiler-context-training-mps-20261001", "runs/compiler-context-model-audit-20261001", "runs/compiler-context-native-dogfood-20261001"} {
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
	for _, name := range []string{"training/train_compiler_context_v1.py", "training/train_bilingual_judgment_v1.py", "training/train_pilot_v2.py",
		"training/split_features_v2.py", "training/train_typed_path_v1.py", "tools/native-feedback-study/compiler_curriculum.go",
		"tools/native-feedback-study/compiler_model_audit.go", "tools/native-feedback-study/compiler_native_dogfood.go"} {
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
	value := manifest{Schema: "gooo/compiler-context-public-model/v1", Repo: "asketeddy/gooo-compiler-context-tiny-v1"}
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
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("public GET status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (32<<20)+1))
	if len(raw) > 32<<20 {
		return nil, errors.New("public response too large")
	}
	return raw, err
}
func verify(bundle, revision, output string, includeAppendices, includeMain bool) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable HF revision required")
	}
	raw, err := read(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var value manifest
	if err = json.Unmarshal(raw, &value); err != nil {
		return err
	}
	if value.Schema != "gooo/compiler-context-public-model/v1" || value.Repo != "asketeddy/gooo-compiler-context-tiny-v1" || len(value.Files) != 16 {
		return errors.New("public allowlist differs")
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
		return errors.New("public exact revision mismatch")
	}
	base := "https://huggingface.co/" + value.Repo + "/resolve/" + revision + "/"
	remote, err := get(base + "publication-manifest.json")
	if err != nil || hash(remote) != hash(raw) {
		return errors.New("public manifest differs")
	}
	for _, file := range value.Files {
		if filepath.Clean(file.Path) != file.Path || strings.Contains(file.Path, "..") || filepath.IsAbs(file.Path) {
			return errors.New("unsafe public path")
		}
		remote, err = get(base + file.Path)
		if err != nil {
			return err
		}
		if hash(remote) != file.SHA || len(remote) != file.Bytes {
			return fmt.Errorf("public hash differs: %s", file.Path)
		}
	}
	var appendices []artifact
	if includeAppendices {
		for _, pair := range [][2]string{
			{"publication/compiler-context-bottlenecks-20261002.json", "research/bottlenecks-20261002.json"},
			{"docs/own-gooo-model-next-study-plan.md", "research/next-study-plan-20261002.md"},
		} {
			local, e := read(pair[0])
			if e != nil {
				return e
			}
			remote, e := get(base + pair[1])
			if e != nil {
				return e
			}
			if hash(remote) != hash(local) {
				return errors.New("public appendix hash differs")
			}
			appendices = append(appendices, artifact{pair[1], hash(remote), len(remote)})
		}
	}
	if includeMain {
		files := map[string]string{
			"publication/compiler-context-feature-main-comparison-20261002.json": "research/main-comparison-20261002.json",
			"publication/compiler-context-main-adoption-20261002.json":           "research/main-adoption-20261002.json",
			"docs/compiler-context-main-results.md":                              "research/main-results-20261002.md",
		}
		root := "runs/compiler-context-native-main-replay-20261002"
		entries, e := os.ReadDir(root)
		if e != nil {
			return e
		}
		if len(entries) != 162 {
			return errors.New("162 frozen main replay files required")
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
				return errors.New("unexpected main replay file")
			}
			files[filepath.Join(root, entry.Name())] = "research/main-replay-20261002/" + entry.Name()
		}
		var paths []string
		for path := range files {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			local, e := read(path)
			if e != nil {
				return e
			}
			remote, e := get(base + files[path])
			if e != nil {
				return e
			}
			if hash(remote) != hash(local) {
				return errors.New("public main replay hash differs")
			}
			appendices = append(appendices, artifact{files[path], hash(remote), len(remote)})
		}
	}
	return save(output, map[string]any{"schema": "gooo/compiler-context-public-verification/v1", "status": "PASS", "repo": value.Repo, "revision": revision,
		"manifest_sha256": hash(raw), "verified_payload_files": len(value.Files) + len(appendices), "anonymous_payload_gets": len(value.Files) + 1 + len(appendices), "appendices": appendices, "archive_entries": len(value.Archive), "api_discovery_gets": 1, "scope": "anonymous immutable-revision byte hashes; selected allowlist only"})
}
func main() {
	mode := flag.String("mode", "pack", "pack or verify")
	bundle := flag.String("bundle", "", "fresh model publication directory")
	revision := flag.String("revision", "", "HF commit SHA")
	output := flag.String("output", "", "verification receipt")
	appendices := flag.Bool("appendices", false, "verify source-bound diagnostic and next-study appendices")
	mainReplay := flag.Bool("main-replay", false, "verify installed-main captures and adoption appendices")
	flag.Parse()
	var err error
	if *mode == "pack" {
		err = pack(*bundle)
	} else if *mode == "verify" {
		err = verify(*bundle, *revision, *output, *appendices, *mainReplay)
	} else {
		err = errors.New("unknown publication mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
