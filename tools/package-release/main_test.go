package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidVersionAndCommit(t *testing.T) {
	for _, value := range []string{"v0.1.0-experimental", "v1.2.3", "v2.4.6-preview.1"} {
		if !validVersion(value) {
			t.Errorf("validVersion(%q) = false", value)
		}
	}
	for _, value := range []string{"", "0.1.0", "v1.2", "v1.2.3/../../tmp", "v1.2.3+build"} {
		if validVersion(value) {
			t.Errorf("validVersion(%q) = true", value)
		}
	}
	if !validCommit(strings.Repeat("a", 40)) {
		t.Fatal("valid lowercase commit SHA rejected")
	}
	for _, value := range []string{strings.Repeat("A", 40), strings.Repeat("g", 40), strings.Repeat("a", 39)} {
		if validCommit(value) {
			t.Errorf("validCommit(%q) = true", value)
		}
	}
}

func TestTargetBuildEnvironmentPinsOfflineReproducibilityControls(t *testing.T) {
	for _, target := range []buildTarget{{goos: "darwin", goarch: "arm64"}, {goos: "linux", goarch: "amd64"}, {goos: "linux", goarch: "arm64"}} {
		environment := strings.Join(targetBuildEnvironment(target), "\n")
		for _, required := range []string{"GOOS=" + target.goos, "GOARCH=" + target.goarch, "CGO_ENABLED=0", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOENV=off", "GOEXPERIMENT="} {
			if !strings.Contains(environment, required) {
				t.Errorf("%s/%s build environment missing %q", target.goos, target.goarch, required)
			}
		}
		if target.goarch == "amd64" && !strings.Contains(environment, "GOAMD64=v1") {
			t.Errorf("%s/%s missing GOAMD64 baseline", target.goos, target.goarch)
		}
		if target.goarch == "arm64" && !strings.Contains(environment, "GOARM64=v8.0") {
			t.Errorf("%s/%s missing GOARM64 baseline", target.goos, target.goarch)
		}
	}
}

func TestBuildEnvironmentOverridesAmbientGoSettingsWithoutDuplicates(t *testing.T) {
	for key, value := range map[string]string{
		"GOOS": "windows", "GOARCH": "386", "GOAMD64": "v3", "GOARM64": "v9.0",
		"CGO_ENABLED": "1", "GOFLAGS": "-tags=local", "GOTOOLCHAIN": "auto",
		"GOWORK": "workspace.work", "GOPROXY": "https://example.invalid", "GOSUMDB": "sum.golang.org",
		"GOENV": "/tmp/local-go-env", "GOEXPERIMENT": "rangefunc",
	} {
		t.Setenv(key, value)
	}
	values := make(map[string]string)
	for _, entry := range buildEnvironment(buildTarget{goos: "linux", goarch: "amd64"}) {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("invalid environment entry %q", entry)
		}
		if _, exists := values[key]; exists {
			t.Fatalf("duplicate environment key %q", key)
		}
		values[key] = value
	}
	for key, want := range map[string]string{
		"GOOS": "linux", "GOARCH": "amd64", "GOAMD64": "v1", "CGO_ENABLED": "0",
		"GOFLAGS": "", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOPROXY": "off",
		"GOSUMDB": "off", "GOENV": "off", "GOEXPERIMENT": "",
	} {
		if got := values[key]; got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestValidateOutputPathRequiresFreshExternalDirectory(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repository")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "new-release")
	got, err := validateOutputPath(root, outside)
	if err != nil || got != outside {
		t.Fatalf("validateOutputPath external fresh directory = %q, %v", got, err)
	}
	if _, err := validateOutputPath(root, filepath.Join(root, "release")); err == nil {
		t.Fatal("output inside repository was accepted")
	}
	existing := filepath.Join(parent, "existing")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := validateOutputPath(root, existing); err == nil {
		t.Fatal("existing output directory was accepted")
	}
	if _, err := validateOutputPath(root, filepath.Join(parent, "missing-parent", "release")); err == nil {
		t.Fatal("missing output parent was accepted")
	}
}

func TestReadAllowedFileRejectsSymlinkParentsAndOversizeFiles(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repository")
	external := filepath.Join(parent, "external")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(external, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "data.bin"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "models")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if _, err := readAllowedFile(root, "models/data.bin", 100); err == nil {
		t.Fatal("file reached through a symlink parent was accepted")
	}
	if err := os.Remove(filepath.Join(root, "models")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "models"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "models", "large.bin"), bytes.Repeat([]byte{'x'}, 8), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readAllowedFile(root, "models/large.bin", 7); err == nil {
		t.Fatal("file over the allowlist byte limit was accepted")
	}
}

func TestDeterministicArchiveHasNormalizedTarAndGzipMetadata(t *testing.T) {
	files := []payloadFile{
		{path: "z/data.txt", data: []byte("second"), mode: 0o644},
		{path: "bin/app", data: []byte("binary"), mode: 0o755},
		{path: "LICENSE", data: []byte("license"), mode: 0o644},
	}
	first, err := deterministicArchive(files)
	if err != nil {
		t.Fatal(err)
	}
	second, err := deterministicArchive([]payloadFile{files[2], files[0], files[1]})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("archive bytes depend on input file order")
	}
	zipReader, err := gzip.NewReader(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	if !zipReader.ModTime.IsZero() || zipReader.Name != "" || zipReader.Comment != "" {
		t.Fatalf("gzip metadata is not normalized: %+v", zipReader.Header)
	}
	tarReader := tar.NewReader(zipReader)
	var names []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
		if header.ModTime.Unix() != 0 || header.Uid != 0 || header.Gid != 0 || header.Format != tar.FormatUSTAR {
			t.Errorf("non-normalized tar header: %+v", header)
		}
		if _, err := io.Copy(io.Discard, tarReader); err != nil {
			t.Fatal(err)
		}
	}
	if err := zipReader.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"LICENSE", "bin/app", "z/data.txt"}) {
		t.Fatalf("archive entries = %v", names)
	}
}

func TestManifestAndChecksumsBindPayloadAndManifest(t *testing.T) {
	files := []payloadFile{
		{path: "README.md", data: []byte("readme"), mode: 0o644},
		{path: "bin/gooo-decision", data: []byte("binary"), mode: 0o755},
	}
	manifest := makeManifest("v0.1.0-experimental", strings.Repeat("a", 40), "go version go1.27.0 darwin/arm64", buildTarget{goos: "darwin", goarch: "arm64"}, files)
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded sourceManifest
	if err := json.Unmarshal(manifestBytes, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SourceRevision != strings.Repeat("a", 40) || decoded.Target != (manifestTarget{GOOS: "darwin", GOARCH: "arm64"}) {
		t.Fatalf("manifest binding = %+v", decoded)
	}
	if !reflect.DeepEqual(decoded.SourcePackages, []string{"cmd/gooo-decision", "cmd/gooo-decision-stream"}) || !reflect.DeepEqual(decoded.BuildEnvironment, targetBuildEnvironment(buildTarget{goos: "darwin", goarch: "arm64"})) {
		t.Fatalf("manifest build identity = %+v", decoded)
	}
	if len(decoded.Files) != 2 || decoded.Files[0].Path != "README.md" || decoded.Files[1].Path != "bin/gooo-decision" {
		t.Fatalf("manifest files = %+v", decoded.Files)
	}
	manifestFile := payloadFile{path: "SOURCE-MANIFEST.json", data: append(manifestBytes, '\n'), mode: 0o644}
	sums := string(makeSumsFile(append(files, manifestFile)))
	for _, file := range append(files, manifestFile) {
		want := hexDigest(file.data) + "  " + file.path + "\n"
		if !strings.Contains(sums, want) {
			t.Errorf("SHA256SUMS missing %q", file.path)
		}
	}
	if strings.Contains(sums, "SHA256SUMS") {
		t.Fatal("checksum file attempted to include its own digest")
	}
	got := sha256.Sum256(files[0].data)
	if hexDigest(files[0].data) != hex.EncodeToString(got[:]) {
		t.Fatal("hexDigest does not match SHA-256")
	}
}

func TestSafeArchivePathRejectsTraversalAndAmbiguousNames(t *testing.T) {
	for _, path := range []string{"", "/absolute", "C:/absolute", "../parent", "a/../b", `models\fp32\model.json`, "./relative"} {
		if safeArchivePath(path) {
			t.Errorf("safeArchivePath(%q) = true", path)
		}
	}
	for _, path := range []string{"LICENSE", "models/fp32/model.json", "bin/gooo-decision"} {
		if !safeArchivePath(path) {
			t.Errorf("safeArchivePath(%q) = false", path)
		}
	}
}
