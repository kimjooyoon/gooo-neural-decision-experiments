package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	root        = "review/body-plan-publication-v1-20260930"
	archiveDir  = "review/body-plan-publication-v1-20260930/v1-initial"
	manifestPin = "38b57de364a142ac3ad00e53ccdcea1f92cbe4960376b2ee6172f9915d44e6a6"
	receiptPin  = "1086069085e83dc51104bd81324dd2483ad247405bd87338ea280ebe6c1d24e7"
)

type manifestEntry struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}

type receipt struct {
	Schema                 string   `json:"schema"`
	AuditedUTC             string   `json:"audited_utc"`
	RunPath                string   `json:"run_path"`
	Decision               string   `json:"decision"`
	Allowlist              string   `json:"allowlist"`
	ExpectedFiles          int      `json:"expected_files"`
	ObservedFiles          int      `json:"observed_files"`
	ExpectedDirectories    int      `json:"expected_directories"`
	ObservedDirectories    int      `json:"observed_directories"`
	TotalBytes             int64    `json:"total_bytes"`
	ManifestFile           string   `json:"manifest_file"`
	ManifestSHA256         string   `json:"manifest_sha256"`
	RawExchangeBundles     int      `json:"raw_exchange_bundles"`
	DecodedProviderReplies int      `json:"decoded_provider_replies"`
	DecodedReplyBytes      int64    `json:"decoded_reply_bytes"`
	ValidatedJSONValues    int      `json:"validated_json_values"`
	TextFileCount          int      `json:"text_file_count"`
	BinaryFileCount        int      `json:"binary_file_count"`
	UnexpectedPaths        int      `json:"unexpected_paths"`
	UnknownJSONKeys        int      `json:"unknown_json_keys"`
	DuplicateJSONKeys      int      `json:"duplicate_json_keys"`
	PrivacyMatches         int      `json:"privacy_matches"`
	Findings               []any    `json:"findings"`
	Notes                  []string `json:"notes"`
}

func main() {
	currentReadme, err := os.ReadFile(filepath.Join("runs/body-plan-v1-laya-7d626b9-20260930", "README.md"))
	must(err)
	oldReadmeText := strings.Replace(string(currentReadme),
		"was 36.2 ms minimum, 40.9 ms median and 184.3 ms maximum. For 222 samples, the\np95 is 46.8 ms at zero-based index `floor(n*0.95)-1`, or 48.0 ms using the\nnearest-rank index `ceil(n*0.95)-1`. This is not GPU or model-kernel time.",
		"was 36.2 ms minimum, 40.9 ms median, 46.8 ms p95 and 184.3 ms maximum. The p95\nuses the floor-index order statistic. This is not GPU or model-kernel time.", 1)
	if oldReadmeText == string(currentReadme) {
		fail("could not uniquely reconstruct the prior README text")
	}
	oldReadme := []byte(oldReadmeText)
	if len(oldReadme) != 5535 {
		fail(fmt.Sprintf("reconstructed README size %d, want 5535", len(oldReadme)))
	}
	oldReadmeHash := sha256.Sum256(oldReadme)

	currentManifest, err := os.ReadFile(filepath.Join(root, "file-manifest.json"))
	must(err)
	var entries []manifestEntry
	must(json.Unmarshal(currentManifest, &entries))
	foundReadme := false
	for i := range entries {
		if entries[i].Path == "README.md" {
			entries[i].Bytes = int64(len(oldReadme))
			entries[i].SHA256 = hex.EncodeToString(oldReadmeHash[:])
			foundReadme = true
		}
	}
	if !foundReadme {
		fail("current manifest has no README entry")
	}
	manifestBytes, err := json.MarshalIndent(entries, "", "  ")
	must(err)
	manifestBytes = append(manifestBytes, '\n')
	manifestHash := sha256.Sum256(manifestBytes)
	if hex.EncodeToString(manifestHash[:]) != manifestPin {
		fail("reconstructed old README and manifest do not match the original pinned manifest SHA-256")
	}

	oldReceipt := receipt{
		Schema:        "gooo/body-plan-publication-privacy-audit/v1",
		AuditedUTC:    "2026-09-30T13:52:21.103903Z",
		RunPath:       "runs/body-plan-v1-laya-7d626b9-20260930",
		Decision:      "PASS",
		Allowlist:     "10 arms x 128 case directories x 5 files; 10 compiled-replay directories x 4 files; pinned root evidence set",
		ExpectedFiles: 6575, ObservedFiles: 6575, ExpectedDirectories: 1300, ObservedDirectories: 1300,
		TotalBytes:   36493493,
		ManifestFile: filepath.ToSlash(filepath.Join(root, "file-manifest.json")), ManifestSHA256: manifestPin,
		RawExchangeBundles: 128, DecodedProviderReplies: 222, DecodedReplyBytes: 95715,
		ValidatedJSONValues: 3453, TextFileCount: 6575, BinaryFileCount: 0,
		UnexpectedPaths: 0, UnknownJSONKeys: 0, DuplicateJSONKeys: 0, PrivacyMatches: 0,
		Notes: []string{
			"Raw exchange request objects and all base64 provider replies were decoded and scanned; captured provider replies are retained byte-for-byte in the run directory.",
			"The resource-monitor source accepts a PID argument generically; the run record contains no PID field or process identifier.",
			"This privacy audit does not validate model quality, training/held-out scoring, or compiler semantics; those are separate reviews.",
			"The manifest hashes every allowlisted payload file; the manifest itself is bound by manifest_sha256.",
		},
	}
	receiptBytes, err := json.MarshalIndent(oldReceipt, "", "  ")
	must(err)
	receiptBytes = append(receiptBytes, '\n')
	receiptHash := sha256.Sum256(receiptBytes)
	if hex.EncodeToString(receiptHash[:]) != receiptPin {
		fail("reconstructed initial receipt bytes do not match the captured receipt SHA-256")
	}

	must(os.MkdirAll(archiveDir, 0o755))
	must(os.WriteFile(filepath.Join(archiveDir, "README.md"), oldReadme, 0o644))
	must(os.WriteFile(filepath.Join(archiveDir, "file-manifest.json"), manifestBytes, 0o644))
	must(os.WriteFile(filepath.Join(archiveDir, "receipt.json"), receiptBytes, 0o644))
	metadata, err := json.MarshalIndent(map[string]any{
		"schema":                      "gooo/body-plan-publication-initial-snapshot/v1",
		"reconstructed_readme_sha256": hex.EncodeToString(oldReadmeHash[:]),
		"manifest_sha256":             manifestPin,
		"receipt_sha256":              receiptPin,
		"verification":                "both manifest and receipt bytes matched the SHA-256 values captured before README correction and scanner v2 changes",
	}, "", "  ")
	must(err)
	metadata = append(metadata, '\n')
	must(os.WriteFile(filepath.Join(archiveDir, "snapshot-metadata.json"), metadata, 0o644))
	fmt.Printf("PASS initial snapshot manifest=%s receipt=%s README=%s bytes=%d\n", manifestPin, receiptPin, hex.EncodeToString(oldReadmeHash[:]), len(oldReadme))
}

func must(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
