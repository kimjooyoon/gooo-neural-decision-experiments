// assemble-public-export copies only the fixed reviewed model publication set.
// It never recursively collects workspace files or reads authentication.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type entry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Role   string `json:"role"`
}

var sources = map[string]string{
	"README.md":                                               "HF-MODEL-CARD.md",
	"model-card.json":                                         "model-card.json",
	"model-contract.json":                                     "model-contract.json",
	"cmd/dataset/main.go":                                     "cmd/dataset/main.go",
	"data/synthetic-ops-v1/dataset.jsonl":                     "data/synthetic-ops-v1/dataset.jsonl",
	"data/synthetic-ops-v1/manifest.json":                     "data/synthetic-ops-v1/manifest.json",
	"runs/pilot-mps-20260930-v1/training-source.py":           "runs/pilot-mps-20260930-v1/training-source.py",
	"runs/pilot-mps-20260930-v1/preexecution.json":            "runs/pilot-mps-20260930-v1/preexecution.json",
	"runs/pilot-mps-20260930-v1/public-training-summary.json": "runs/pilot-mps-20260930-v1/public-training-summary.json",
	"runs/pilot-mps-20260930-v1/go-audit.json":                "runs/pilot-mps-20260930-v1/go-audit.json",
	"runs/pilot-mps-20260930-v1/go-parity.json":               "runs/pilot-mps-20260930-v1/go-parity.json",
}

var roles = map[string]string{
	"README.md": "documentation", "model-card.json": "model_card",
	"model-contract.json": "model_contract", "cmd/dataset/main.go": "generator_source",
	"data/synthetic-ops-v1/dataset.jsonl":                     "synthetic_dataset",
	"data/synthetic-ops-v1/manifest.json":                     "dataset_manifest",
	"runs/pilot-mps-20260930-v1/training-source.py":           "training_source",
	"runs/pilot-mps-20260930-v1/preexecution.json":            "training_record",
	"runs/pilot-mps-20260930-v1/public-training-summary.json": "training_report",
	"runs/pilot-mps-20260930-v1/go-audit.json":                "external_verification",
	"runs/pilot-mps-20260930-v1/go-parity.json":               "python_parity",
}

func main() {
	output := flag.String("output", "", "new bundle directory")
	policy := flag.String("allowlist", "", "separate allowlist JSON output")
	flag.Parse()
	if err := assemble(*output, *policy); err != nil {
		fmt.Fprintln(os.Stderr, "assemble-public-export:", err)
		os.Exit(1)
	}
}

func assemble(output, policy string) error {
	if output == "" || policy == "" || flag.NArg() != 0 {
		return errors.New("--output and --allowlist required")
	}
	bundleAbs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	policyAbs, err := filepath.Abs(policy)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(bundleAbs, policyAbs)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return errors.New("allowlist must be outside bundle")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("bundle output already exists or cannot be inspected")
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		prefix := "runs/pilot-mps-20260930-v1/models/" + variant + "/"
		for _, filename := range []string{"model.json", "weights.bin"} {
			name := prefix + filename
			sources[name] = name
			if filename == "model.json" {
				roles[name] = "model_metadata"
			} else {
				roles[name] = "model_weights"
			}
		}
	}
	var names []string
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	var entries []entry
	contents := make(map[string][]byte)
	var total int
	for _, name := range names {
		info, err := os.Lstat(sources[name])
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("invalid source %q", sources[name])
		}
		if info.Size() <= 0 || info.Size() > 3<<20 {
			return fmt.Errorf("source outside size bound: %q", name)
		}
		data, err := os.ReadFile(sources[name])
		if err != nil {
			return err
		}
		total += len(data)
		if total > 3<<20 {
			return errors.New("bundle exceeds 3 MiB")
		}
		contents[name] = data
		digest := sha256.Sum256(data)
		entries = append(entries, entry{name, hex.EncodeToString(digest[:]), roles[name]})
	}
	encoded, err := json.MarshalIndent(struct {
		Schema string  `json:"schema"`
		Files  []entry `json:"files"`
	}{"gooo/public-export-allowlist/v1", entries}, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	// An existing exact allowlist is reusable, but a changed policy is never overwritten.
	if existing, err := os.ReadFile(policy); err == nil {
		if !bytes.Equal(existing, encoded) {
			return errors.New("existing allowlist differs; use a new versioned path")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	for _, name := range names {
		filename := filepath.Join(output, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filename, contents[name], 0644); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(policy), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(policy, encoded, 0644); err != nil {
		return err
	}
	fmt.Printf("Assembled %d explicitly selected files (%d bytes); run the independent export verifier before publication.\n", len(entries), total)
	return nil
}
