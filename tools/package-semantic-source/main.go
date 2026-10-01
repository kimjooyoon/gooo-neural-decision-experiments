// package-semantic-source publishes source ABI evidence, never new model weights.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
)

const sdkRevision = "25c6911359a2009381f5e4388fb0f1ff60b279ad"
const researchRevision = "d9fc4d1c8140c278794c6aabde3a7585af4692c0"
const maxPublicBytes = 4 << 20

var privatePattern = regexp.MustCompile(`/Users/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func gitBytes(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	raw, err := command.Output()
	if err != nil || len(raw) > maxPublicBytes {
		return nil, errors.New("bounded pinned Git object required")
	}
	return raw, nil
}

func validateEvidence(raw []byte) error {
	var evidence struct {
		Schema     string `json:"schema"`
		Status     string `json:"status"`
		NativeMain string `json:"native_main_revision"`
		Feature    string `json:"feature_version"`
		Updates    *int   `json:"new_optimizer_updates"`
		Weights    *int   `json:"new_model_exports"`
	}
	if len(raw) == 0 || len(raw) > maxPublicBytes || privatePattern.Match(raw) || json.Unmarshal(raw, &evidence) != nil {
		return errors.New("public implementation JSON required")
	}
	mainSHA, err := hex.DecodeString(evidence.NativeMain)
	if err != nil || len(mainSHA) != 20 || evidence.NativeMain != hex.EncodeToString(mainSHA) ||
		evidence.Schema != "gooo/semantic-source-v3-implementation/v1" || evidence.Status != "NATIVE_MAIN_VERIFIED" ||
		evidence.Feature != "semantic_context_intent_v3" || evidence.Updates == nil || evidence.Weights == nil || *evidence.Updates != 0 || *evidence.Weights != 0 {
		return errors.New("source-only implementation evidence contract mismatch")
	}
	return nil
}

func validateSourceArchive(raw []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(reader.File) == 0 || len(reader.File) > 128 {
		return errors.New("bounded SDK source archive required")
	}
	seen := map[string]bool{}
	var total uint64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if !file.Mode().IsRegular() || seen[file.Name] || file.Name != path.Clean(file.Name) || path.IsAbs(file.Name) ||
			regexp.MustCompile(`(^|/)\.\.(/|$)|[\\:]`).MatchString(file.Name) || file.UncompressedSize64 > maxPublicBytes {
			return errors.New("unsafe source archive entry")
		}
		seen[file.Name] = true
		total += file.UncompressedSize64
		if total > maxPublicBytes {
			return errors.New("source archive exceeds byte budget")
		}
		stream, err := file.Open()
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, maxPublicBytes+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || uint64(len(data)) != file.UncompressedSize64 || privatePattern.Match(data) {
			return errors.New("source archive entry failed public byte preflight")
		}
	}
	if !seen["go.mod"] || !seen["source-provenance.json"] || !seen["semantic_features.go"] || !seen["pathplan/source_features.go"] {
		return errors.New("SDK archive lacks required direct source ABI")
	}
	return nil
}

func packageSource(sdk, evidenceFile, output string) error {
	evidence, err := os.ReadFile(evidenceFile)
	if err != nil {
		return err
	}
	if err = validateEvidence(evidence); err != nil {
		return err
	}
	archive, err := gitBytes(sdk, "archive", "--format=zip", sdkRevision)
	if err != nil {
		return err
	}
	if err = validateSourceArchive(archive); err != nil {
		return err
	}
	payloads := map[string][]byte{"implementation.json": evidence, "go-sdk-source.zip": archive}
	for name, origin := range map[string]string{
		"protocol.md":                          "docs/semantic-context-v3-protocol-20261002.md",
		"fresh-composition-preregistration.md": "docs/fresh-composition-semantic-v3-preregistration-20261002.md",
	} {
		payloads[name], err = gitBytes(".", "show", researchRevision+":"+origin)
		if err != nil {
			return err
		}
	}
	payloads["sdk-source-provenance.json"], err = gitBytes(sdk, "show", sdkRevision+":source-provenance.json")
	if err != nil {
		return err
	}
	payloads["README.md"] = []byte("# Own Gooo model: direct source ABI v3\n\nThis appendix publishes Go runtime source, direct source feature/feedback contracts and native implementation evidence. It contains zero new optimizer updates or model weight exports. Existing model bundles retain their feature versions. The fresh matched composition study is preregistered; its training/results are subsequent work.\n\nThe independently initialized own model direction uses Gooo structure and Korean/English bounded judgments, without Laya weights or derivatives. Gooo owns typed fragments and finite acceptance. Optional representation decline retains deterministic assembly. Five trits per byte are disk packing, not packed runtime arithmetic.\n")
	return writePayloads(output, payloads)
}

func writePayloads(output string, payloads map[string][]byte) error {
	var names []string
	for name, raw := range payloads {
		if len(raw) > maxPublicBytes || name != "go-sdk-source.zip" && privatePattern.Match(raw) {
			return errors.New("payload failed public byte preflight")
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var files []map[string]any
	for _, name := range names {
		files = append(files, map[string]any{"path": name, "bytes": len(payloads[name]), "sha256": digest(payloads[name])})
	}
	manifest := map[string]any{"schema": "gooo/semantic-source-publication/v1", "repository": "asketeddy/gooo-compiler-context-tiny-v1", "prefix": "research/semantic-source-v3-20261002", "sdk_revision": sdkRevision, "protocol_revision": researchRevision, "files": files, "new_optimizer_updates": 0, "new_model_exports": 0, "scope": "source ABI and implementation evidence; no learned improvement claim"}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return errors.New("fresh output directory required")
	}
	for _, name := range names {
		if err = os.WriteFile(filepath.Join(output, name), payloads[name], 0644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(output, "publication-manifest.json"), append(raw, '\n'), 0644)
}

func main() {
	sdk := flag.String("sdk", "", "pinned SDK checkout")
	evidence := flag.String("evidence", "", "public main implementation evidence")
	output := flag.String("output", "", "fresh publication directory")
	flag.Parse()
	if flag.NArg() != 0 || *sdk == "" || *evidence == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "sdk, evidence and output required")
		os.Exit(2)
	}
	if err := packageSource(*sdk, *evidence, *output); err != nil {
		fmt.Fprintln(os.Stderr, "package-semantic-source:", err)
		os.Exit(1)
	}
	fmt.Println(`{"status":"PACKAGED_SOURCE_ABI","public_payloads":6,"new_optimizer_updates":0,"new_model_exports":0}`)
}
