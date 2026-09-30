// package-compiler-model copies a fixed public allowlist, never an environment.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type artifact struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}

var privateText = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)

func digest(raw []byte) string { value := sha256.Sum256(raw); return hex.EncodeToString(value[:]) }

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	runDir := flag.String("run", "runs/compiler-prov-v3-mps-20261001", "validated training and dogfood run")
	output := flag.String("output", "publication/hf-compiler-prov-v2", "fresh public model folder")
	bodyReview := flag.Bool("body-review", false, "include the separately captured broader body comparison")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return errors.New("output must be fresh")
	}
	for _, path := range []string{"go-audit-v2.json", "native-improved/report.json", "native-offline/report.json", "parent-adjudication-v2/report.json"} {
		raw, err := os.ReadFile(filepath.Join(*runDir, path))
		if err != nil {
			return err
		}
		var report map[string]any
		if err := json.Unmarshal(raw, &report); err != nil {
			return err
		}
		if report["status"] != "PASS" {
			return fmt.Errorf("required report %s is not PASS", path)
		}
	}
	files := map[string]string{
		"training-report.json":       filepath.Join(*runDir, "report.json"),
		"training-preexecution.json": filepath.Join(*runDir, "preexecution.json"),
		"go-audit.json":              filepath.Join(*runDir, "go-audit-v2.json"),
		"native-improved.json":       filepath.Join(*runDir, "native-improved/report.json"),
		"native-offline.json":        filepath.Join(*runDir, "native-offline/report.json"),
		"parent-adjudication.json":   filepath.Join(*runDir, "parent-adjudication-v2/report.json"),
		"dataset-manifest.json":      "data/compiler-prov-v2/manifest.json",
		"model-contract.json":        "model-contract.json", "LICENSE": "LICENSE",
		"README.md": "docs/hf-compiler-prov-v2-model-card.md",
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		for _, name := range []string{"model.json", "weights.bin"} {
			path := filepath.Join("models", variant, name)
			files[path] = filepath.Join(*runDir, path)
		}
	}
	for _, id := range []string{"arithmetic-fp32", "arithmetic-ptq_ternary", "arithmetic-qat_ternary", "boolean-fp32", "boolean-ptq_ternary", "boolean-qat_ternary"} {
		files[filepath.Join("provenance", id+".ttl")] = filepath.Join(*runDir, "native-improved", id+".prov.ttl")
	}
	manifestSchema := "gooo/public-compiler-model-allowlist/v1"
	if *bodyReview {
		comparison := "runs/compiler-prov-v3-bodyplan-20261001/comparison.json"
		raw, err := os.ReadFile(comparison)
		if err != nil {
			return err
		}
		var report map[string]any
		if err := json.Unmarshal(raw, &report); err != nil {
			return err
		}
		if report["status"] != "PASS" {
			return errors.New("broader comparison is not complete and validated")
		}
		files["README.md"] = "docs/hf-compiler-prov-v2-reviewed-model-card.md"
		files["bodyplan-comparison.json"] = comparison
		files["bodyplan-preexecution.json"] = "runs/compiler-prov-v3-bodyplan-20261001/preexecution.json"
		files["bodyplan-evidence-manifest.json"] = "runs/compiler-prov-v3-bodyplan-20261001/artifact-manifest.json"
		manifestSchema = "gooo/public-compiler-model-allowlist/v2"
	}
	var manifest []artifact
	// Validate every source before creating any public output.
	contents := make(map[string][]byte, len(files))
	for name, source := range files {
		info, err := os.Lstat(source)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("allowlisted source is not regular")
		}
		raw, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if len(raw) > 4*1024*1024 {
			return errors.New("artifact exceeds 4 MiB")
		}
		if !strings.HasSuffix(name, ".bin") && privateText.Match(raw) {
			return fmt.Errorf("private path or credential-like content in %s", name)
		}
		contents[name] = raw
		manifest = append(manifest, artifact{Path: name, SHA: digest(raw), Bytes: len(raw)})
	}
	sort.Slice(manifest, func(i, j int) bool { return manifest[i].Path < manifest[j].Path })
	for name, raw := range contents {
		path := filepath.Join(*output, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(map[string]any{"schema": manifestSchema, "files": manifest, "credentials_and_host_paths_scanned": true,
		"binary_provenance": "Weights trained only on the disclosed public synthetic curriculum; binary files are not text secret-scanned"}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "publication-manifest.json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("prepared %d allowlisted public artifacts plus manifest\n", len(manifest))
	return nil
}
