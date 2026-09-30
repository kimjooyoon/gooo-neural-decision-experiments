// verify-compiler-model verifies the fixed model bundle locally and, optionally,
// at one immutable public Hugging Face revision without sending credentials.
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
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const repository = "asketeddy/gooo-compiler-prov-tiny-v2"

type artifact struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}

type manifest struct {
	Schema             string     `json:"schema"`
	Files              []artifact `json:"files"`
	PrivateTextScanned bool       `json:"credentials_and_host_paths_scanned"`
	BinaryProvenance   string     `json:"binary_provenance"`
}

func digest(raw []byte) string {
	value := sha256.Sum256(raw)
	return hex.EncodeToString(value[:])
}

func fixedPaths() map[string]bool {
	out := map[string]bool{}
	for _, name := range []string{"README.md", "LICENSE", "model-contract.json", "dataset-manifest.json", "training-report.json", "training-preexecution.json", "go-audit.json", "native-improved.json", "native-offline.json", "parent-adjudication.json"} {
		out[name] = true
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		out["models/"+variant+"/model.json"] = true
		out["models/"+variant+"/weights.bin"] = true
		for _, family := range []string{"arithmetic", "boolean"} {
			out["provenance/"+family+"-"+variant+".ttl"] = true
		}
	}
	return out
}

func readLocal(root string) ([]artifact, []byte, error) {
	raw, err := os.ReadFile(filepath.Join(root, "publication-manifest.json"))
	if err != nil {
		return nil, nil, err
	}
	var value manifest
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, nil, err
	}
	expected := fixedPaths()
	if value.Schema != "gooo/public-compiler-model-allowlist/v1" || !value.PrivateTextScanned || value.BinaryProvenance == "" || len(value.Files) != len(expected) {
		return nil, nil, errors.New("invalid fixed publication manifest")
	}
	seen := map[string]bool{}
	for _, file := range value.Files {
		if !expected[file.Path] || seen[file.Path] || file.Bytes < 0 || file.Bytes > 4<<20 || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(file.SHA) {
			return nil, nil, errors.New("unexpected or duplicate artifact")
		}
		seen[file.Path] = true
		info, err := os.Lstat(filepath.Join(root, file.Path))
		if err != nil {
			return nil, nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, nil, errors.New("nonregular artifact")
		}
		content, err := os.ReadFile(filepath.Join(root, file.Path))
		if err != nil {
			return nil, nil, err
		}
		if len(content) != file.Bytes || digest(content) != file.SHA {
			return nil, nil, fmt.Errorf("local digest mismatch: %s", file.Path)
		}
	}
	seen["publication-manifest.json"] = true
	err = filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		if !seen[filepath.ToSlash(relative)] {
			return errors.New("extra bundle file")
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	value.Files = append(value.Files, artifact{"publication-manifest.json", digest(raw), len(raw)})
	sort.Slice(value.Files, func(i, j int) bool { return value.Files[i].Path < value.Files[j].Path })
	return value.Files, raw, nil
}

func publicGet(ctx context.Context, client *http.Client, target string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	// No token lookup, authorization header, cookie jar, or authenticated CLI.
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anonymous GET returned %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, errors.New("public response exceeds limit")
	}
	return raw, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root := flag.String("bundle", "publication/hf-compiler-prov-v2", "fixed local bundle")
	revision := flag.String("revision", "", "immutable public revision; omit for local verification")
	output := flag.String("output", "", "fresh verification receipt")
	flag.Parse()
	if flag.NArg() != 0 || *output == "" {
		return errors.New("a fresh --output is required")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return errors.New("receipt output must be fresh")
	}
	files, raw, err := readLocal(*root)
	if err != nil {
		return err
	}
	receipt := map[string]any{"schema": "gooo/compiler-model-public-verification/v1", "status": "PASS", "local_files_verified": files, "manifest_sha256": digest(raw), "credentials_sent": false, "network_requests": 0}
	if *revision != "" {
		if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(*revision) {
			return errors.New("revision must be a full immutable commit")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 8 || req.URL.Scheme != "https" || req.URL.User != nil {
				return errors.New("unsafe public redirect")
			}
			return nil
		}}
		infoRaw, err := publicGet(ctx, client, "https://huggingface.co/api/models/"+repository+"/revision/"+*revision, 1<<20)
		if err != nil {
			return err
		}
		var info struct {
			SHA      string `json:"sha"`
			Private  bool   `json:"private"`
			Siblings []struct {
				Path string `json:"rfilename"`
			} `json:"siblings"`
		}
		if err := json.Unmarshal(infoRaw, &info); err != nil {
			return err
		}
		if info.SHA != *revision || info.Private {
			return errors.New("public revision mismatch")
		}
		expected := map[string]bool{".gitattributes": true}
		for _, file := range files {
			expected[file.Path] = true
		}
		seen := map[string]bool{}
		for _, sibling := range info.Siblings {
			if !expected[sibling.Path] || seen[sibling.Path] {
				return errors.New("unexpected public repository file")
			}
			seen[sibling.Path] = true
		}
		for _, file := range files {
			if !seen[file.Path] || path.Clean(file.Path) != file.Path || strings.Contains(file.Path, "\\") {
				return errors.New("missing or unsafe public path")
			}
			content, err := publicGet(ctx, client, "https://huggingface.co/"+repository+"/resolve/"+*revision+"/"+file.Path, 4<<20)
			if err != nil {
				return fmt.Errorf("%s: %w", file.Path, err)
			}
			if len(content) != file.Bytes || digest(content) != file.SHA {
				return fmt.Errorf("public digest mismatch: %s", file.Path)
			}
		}
		receipt["repository"] = repository
		receipt["commit_oid"] = *revision
		receipt["public_files_verified"] = files
		receipt["network_requests"] = len(files) + 1
	}
	receipt["recorded_at_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
	result, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(*output, append(result, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("PASS: verified %d files; public revision %s\n", len(files), *revision)
	return nil
}
