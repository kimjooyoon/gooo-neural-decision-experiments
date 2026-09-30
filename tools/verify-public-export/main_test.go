package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func verifierRepoRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate verifier test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
}

func makeVerifiedFixture(t *testing.T) (bundle, policy string) {
	t.Helper()
	root := verifierRepoRoot(t)
	temporary := t.TempDir()
	bundle = filepath.Join(temporary, "bundle")
	policy = filepath.Join(temporary, "allowlist.json")
	if err := os.Mkdir(bundle, 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "publication/public-export-allowlist-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var allowlist allowlist
	if err := json.Unmarshal(raw, &allowlist); err != nil {
		t.Fatal(err)
	}
	for _, item := range allowlist.Files {
		sourcePath := item.Path
		if sourcePath == "README.md" {
			sourcePath = "HF-MODEL-CARD.md"
		}
		source := filepath.Join(root, filepath.FromSlash(sourcePath))
		destination := filepath.Join(bundle, filepath.FromSlash(item.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read fixture source %s: %v", sourcePath, err)
		}
		if err := os.WriteFile(destination, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(policy, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return bundle, policy
}

func readTestAllowlist(t *testing.T, policyPath string) allowlist {
	t.Helper()
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy allowlist
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	return policy
}

func writeTestAllowlist(t *testing.T, policyPath string, policy allowlist) {
	t.Helper()
	raw, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(policyPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func rehashAllowlistedFile(t *testing.T, bundle, policyPath, name string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(bundle, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	policy := readTestAllowlist(t, policyPath)
	for i := range policy.Files {
		if policy.Files[i].Path == name {
			policy.Files[i].SHA256 = hex.EncodeToString(digest[:])
			writeTestAllowlist(t, policyPath, policy)
			return
		}
	}
	t.Fatalf("allowlist does not contain %q", name)
}

func TestVerifyPublicBundleHasExactReviewedSeventeenFileInventory(t *testing.T) {
	bundle, policyPath := makeVerifiedFixture(t)
	verified, err := verify(bundle, policyPath)
	if err != nil {
		t.Fatalf("verify reviewed bundle: %v", err)
	}
	if verified.Decision != "PASS" || verified.FileCount != 17 {
		t.Fatalf("unexpected verification result: decision=%q files=%d", verified.Decision, verified.FileCount)
	}
	wantRoles := map[string]int{
		"documentation": 1, "dataset_manifest": 1, "synthetic_dataset": 1,
		"model_contract": 1, "generator_source": 1, "training_source": 1,
		"training_record": 1, "training_report": 1, "model_card": 1,
		"external_verification": 1, "python_parity": 1, "model_metadata": 3, "model_weights": 3,
	}
	if len(verified.Roles) != len(wantRoles) {
		t.Fatalf("role inventory has %d roles, want %d: %#v", len(verified.Roles), len(wantRoles), verified.Roles)
	}
	for role, count := range wantRoles {
		if verified.Roles[role] != count {
			t.Errorf("role %q count=%d, want %d", role, verified.Roles[role], count)
		}
	}
	if len(verified.Models) != 3 || verified.Provenance.ExternalParityStatus != "PASS" ||
		verified.Provenance.ParityRowsPerVariant != 32 || verified.Provenance.TestRowsPerVariant != 256 {
		t.Fatalf("model/provenance inventory is incomplete: models=%d provenance=%+v", len(verified.Models), verified.Provenance)
	}
}

func TestAllowlistCannotAddAnEighteenthArtifact(t *testing.T) {
	_, policyPath := makeVerifiedFixture(t)
	policy := readTestAllowlist(t, policyPath)
	policy.Files = append(policy.Files, allowedArtifact{
		Path: "extra.txt", SHA256: strings.Repeat("0", 64), Role: "documentation",
	})
	if err := requireRolePaths(mustValidateEntries(t, policy.Files)); err == nil || !strings.Contains(err.Error(), "exactly the 17") {
		t.Fatalf("18-file allowlist was not rejected by the fixed inventory check: %v", err)
	}
}

func mustValidateEntries(t *testing.T, entries []allowedArtifact) map[string]allowedArtifact {
	t.Helper()
	validated, err := validateEntries(entries)
	if err != nil {
		t.Fatalf("validate fixture entries: %v", err)
	}
	return validated
}

func TestRejectsDuplicateJSONKeysInAllowlist(t *testing.T) {
	bundle, _ := makeVerifiedFixture(t)
	policyPath := filepath.Join(t.TempDir(), "duplicate.json")
	policy := `{"schema":"gooo/public-export-allowlist/v1","schema":"gooo/public-export-allowlist/v1","files":[]}`
	if err := os.WriteFile(policyPath, []byte(policy), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := verify(bundle, policyPath); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate allowlist key was not rejected: %v", err)
	}
}

func TestRejectsDuplicateJSONKeysInBundledModelCard(t *testing.T) {
	bundle, policyPath := makeVerifiedFixture(t)
	cardPath := filepath.Join(bundle, "model-card.json")
	duplicate := []byte(`{"schema":"gooo/tiny-ir-decision-model-card/v1","schema":"gooo/tiny-ir-decision-model-card/v1"}`)
	if err := os.WriteFile(cardPath, duplicate, 0644); err != nil {
		t.Fatal(err)
	}
	rehashAllowlistedFile(t, bundle, policyPath, "model-card.json")
	if _, err := verify(bundle, policyPath); err == nil || !strings.Contains(err.Error(), "duplicate or invalid JSON object key") {
		t.Fatalf("duplicate model-card key was not rejected: %v", err)
	}
}

func TestRejectsSymlinkedAndUnlistedBundleEntries(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		bundle, policyPath := makeVerifiedFixture(t)
		outside := filepath.Join(t.TempDir(), "outside.txt")
		if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(bundle, "extra-link.txt")); err != nil {
			t.Skipf("symlink creation is not supported: %v", err)
		}
		if _, err := verify(bundle, policyPath); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlinked bundle entry was not rejected: %v", err)
		}
	})
	t.Run("unlisted file", func(t *testing.T) {
		bundle, policyPath := makeVerifiedFixture(t)
		if err := os.WriteFile(filepath.Join(bundle, "extra.txt"), []byte("unlisted"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := verify(bundle, policyPath); err == nil || !strings.Contains(err.Error(), "unlisted file") {
			t.Fatalf("unlisted bundle file was not rejected: %v", err)
		}
	})
}

func TestRejectsAllowlistDigestMismatch(t *testing.T) {
	bundle, policyPath := makeVerifiedFixture(t)
	name := "model-contract.json"
	if err := os.WriteFile(filepath.Join(bundle, name), []byte(`{"tampered":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := verify(bundle, policyPath); err == nil || !strings.Contains(err.Error(), "digest differs") {
		t.Fatalf("tampered bundle file was not rejected: %v", err)
	}
}

func TestManifestRequiresCompletePublicSourceClaim(t *testing.T) {
	root := verifierRepoRoot(t)
	manifestRaw, err := os.ReadFile(filepath.Join(root, "data/synthetic-ops-v1/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodeManifest(manifestRaw)
	if err != nil {
		t.Fatal(err)
	}
	datasetRaw, err := os.ReadFile(filepath.Join(root, "data/synthetic-ops-v1/dataset.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	counts, datasetDigest, splitDigests, err := inspectDataset(datasetRaw)
	if err != nil {
		t.Fatal(err)
	}
	contractRaw, err := os.ReadFile(filepath.Join(root, "model-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	generatorRaw, err := os.ReadFile(filepath.Join(root, "cmd/dataset/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkManifest(manifest, datasetDigest, splitDigests, counts,
		digestHex(contractRaw), digestHex(generatorRaw)); err != nil {
		t.Fatalf("published manifest source claim should pass: %v", err)
	}
	manifest.SyntheticProvenance.SourceMaterial = "No repository source or user content is used."
	if err := checkManifest(manifest, datasetDigest, splitDigests, counts,
		digestHex(contractRaw), digestHex(generatorRaw)); err == nil {
		t.Fatal("incomplete source claim omitted model output, external dataset, and private text but was accepted")
	}
}

func TestRejectsModelCardWithMismatchedExternalAuditBinding(t *testing.T) {
	bundle, policyPath := makeVerifiedFixture(t)
	cardPath := filepath.Join(bundle, "model-card.json")
	var card map[string]any
	raw, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &card); err != nil {
		t.Fatal(err)
	}
	card["external_verification_sha256"] = strings.Repeat("0", 64)
	modified, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cardPath, append(modified, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	rehashAllowlistedFile(t, bundle, policyPath, "model-card.json")
	if _, err := verify(bundle, policyPath); err == nil || !strings.Contains(err.Error(), "model card scope/origin or external Go verification binding") {
		t.Fatalf("model card with mismatched external audit provenance was not rejected: %v", err)
	}
}
