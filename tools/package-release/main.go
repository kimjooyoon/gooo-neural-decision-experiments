// Command package-release builds small, offline release archives from a clean
// committed source tree and the reviewed public model bundles.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const (
	defaultVersion = "v0.1.0-experimental"
	modelRoot      = "runs/pilot-mps-20260930-v1/models"
	maxSmallFile   = 64 * 1024
	maxModelFile   = 1024 * 1024
	maxBinaryFile  = 16 * 1024 * 1024
	maxBundleBytes = 32 * 1024 * 1024
	manifestSchema = "gooo/experimental-release-source-manifest/v1"
)

var (
	versionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)
	targets        = []buildTarget{
		{goos: "darwin", goarch: "arm64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "linux", goarch: "arm64"},
	}
	variants = []string{"fp32", "ptq_ternary", "qat_ternary"}
)

type buildTarget struct {
	goos   string
	goarch string
}

type payloadFile struct {
	path string
	data []byte
	mode int64
}

type fileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size_bytes"`
}

type manifestTarget struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

type sourceManifest struct {
	Schema           string         `json:"schema"`
	Version          string         `json:"version"`
	SourceRevision   string         `json:"source_revision"`
	GoVersion        string         `json:"go_version"`
	Target           manifestTarget `json:"target"`
	SourcePackages   []string       `json:"source_packages"`
	BuildFlags       []string       `json:"build_flags"`
	BuildEnvironment []string       `json:"build_environment"`
	Files            []fileDigest   `json:"files"`
}

type modelMetadata struct {
	Schema        string `json:"schema"`
	Variant       string `json:"variant"`
	WeightsFile   string `json:"weights_file"`
	WeightsSHA256 string `json:"weights_sha256"`
}

type releaseArchive struct {
	name string
	data []byte
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("package-release", flag.ContinueOnError)
	flags.SetOutput(stderr)
	version := flags.String("version", defaultVersion, "release version label")
	sourceSHA := flags.String("source-sha", "", "required clean Git commit to package")
	outputDir := flags.String("output", "", "new output directory outside the repository")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *sourceSHA == "" || *outputDir == "" {
		fmt.Fprintln(stderr, "usage: package-release --source-sha COMMIT --output NEW_DIR [--version v0.1.0-experimental]")
		return 2
	}
	if !validVersion(*version) {
		fmt.Fprintln(stderr, "version must be a safe vMAJOR.MINOR.PATCH label with an optional prerelease")
		return 2
	}
	if !validCommit(*sourceSHA) {
		fmt.Fprintln(stderr, "source-sha must be a lowercase 40-character Git commit SHA")
		return 2
	}
	root, err := repositoryRoot()
	if err != nil {
		fmt.Fprintf(stderr, "find repository root: %v\n", err)
		return 1
	}
	outputPath, err := validateOutputPath(root, *outputDir)
	if err != nil {
		fmt.Fprintf(stderr, "output path: %v\n", err)
		return 2
	}
	if err := verifyCleanSource(root, *sourceSHA); err != nil {
		fmt.Fprintf(stderr, "source check: %v\n", err)
		return 1
	}
	goVersion, err := goVersion(root)
	if err != nil {
		fmt.Fprintf(stderr, "read Go version: %v\n", err)
		return 1
	}
	models, err := loadModelPayloads(root)
	if err != nil {
		fmt.Fprintf(stderr, "validate model bundles: %v\n", err)
		return 1
	}
	archives, err := buildArchives(root, *version, *sourceSHA, goVersion, models)
	if err != nil {
		fmt.Fprintf(stderr, "build release archives: %v\n", err)
		return 1
	}
	if err := verifyCleanSource(root, *sourceSHA); err != nil {
		fmt.Fprintf(stderr, "source changed during build: %v\n", err)
		return 1
	}
	if err := installArchives(outputPath, archives); err != nil {
		fmt.Fprintf(stderr, "write release archives: %v\n", err)
		return 1
	}
	for _, archive := range archives {
		fmt.Fprintf(stdout, "%s  %s\n", hexDigest(archive.data), archive.name)
	}
	return 0
}

func validVersion(value string) bool { return versionPattern.MatchString(value) }

func validCommit(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func repositoryRoot() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	output, err := commandOutput(workingDirectory, nil, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", errors.New("Git returned an empty repository root")
	}
	return filepath.Abs(root)
}

func verifyCleanSource(root, expectedSHA string) error {
	head, err := commandOutput(root, nil, "git", "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != expectedSHA {
		return fmt.Errorf("HEAD is %s, expected %s", strings.TrimSpace(string(head)), expectedSHA)
	}
	status, err := commandOutput(root, nil, "git", "status", "--porcelain", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(status)) != 0 {
		return errors.New("working tree has staged, modified, or untracked files")
	}
	return nil
}

func validateOutputPath(root, requested string) (string, error) {
	absolute, err := filepath.Abs(requested)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", errors.New("output directory already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent := filepath.Dir(absolute)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("output parent must already exist: %w", err)
	}
	if !parentInfo.IsDir() {
		return "", errors.New("output parent is not a directory")
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	parentReal, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootReal, parentReal)
	if err != nil {
		return "", err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return "", errors.New("output directory must be outside the repository")
	}
	return absolute, nil
}

func loadModelPayloads(root string) ([]payloadFile, error) {
	files := make([]payloadFile, 0, len(variants)*2)
	for _, variant := range variants {
		directory := filepath.Join(root, filepath.FromSlash(modelRoot), variant)
		metadataPath := filepath.Join(directory, "model.json")
		if _, err := decision.Load(metadataPath); err != nil {
			return nil, fmt.Errorf("%s: %w", variant, err)
		}
		metadataBytes, err := readAllowedFile(root, filepath.ToSlash(filepath.Join(modelRoot, variant, "model.json")), maxSmallFile)
		if err != nil {
			return nil, err
		}
		weightBytes, err := readAllowedFile(root, filepath.ToSlash(filepath.Join(modelRoot, variant, "weights.bin")), maxModelFile)
		if err != nil {
			return nil, err
		}
		var metadata modelMetadata
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, fmt.Errorf("decode %s model metadata: %w", variant, err)
		}
		if metadata.Schema != decision.MetadataSchema || metadata.Variant != variant || metadata.WeightsFile != "weights.bin" || metadata.WeightsSHA256 != hexDigest(weightBytes) {
			return nil, fmt.Errorf("%s model metadata does not bind the allowed weights file", variant)
		}
		files = append(files,
			payloadFile{path: filepath.ToSlash(filepath.Join("models", variant, "model.json")), data: metadataBytes, mode: 0o644},
			payloadFile{path: filepath.ToSlash(filepath.Join("models", variant, "weights.bin")), data: weightBytes, mode: 0o644},
		)
	}
	return files, nil
}

func readAllowedFile(root, relative string, maximum int64) ([]byte, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return nil, fmt.Errorf("path is outside the fixed allowlist: %q", relative)
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	path, err := regularPathWithoutSymlinkParents(rootReal, clean)
	if err != nil {
		return nil, fmt.Errorf("validate %s: %w", relative, err)
	}
	return readRegularBoundedFile(path, relative, maximum)
}

func regularPathWithoutSymlinkParents(root, relative string) (string, error) {
	components := strings.Split(relative, string(filepath.Separator))
	path := root
	for index, component := range components {
		path = filepath.Join(path, component)
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%s is a symbolic link", filepath.ToSlash(path))
		}
		last := index == len(components)-1
		if last && !info.Mode().IsRegular() {
			return "", fmt.Errorf("%s is not a regular file", filepath.ToSlash(path))
		}
		if !last && !info.IsDir() {
			return "", fmt.Errorf("%s is not a directory", filepath.ToSlash(path))
		}
	}
	return path, nil
}

func readRegularBoundedFile(path, displayPath string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", displayPath, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s must be a regular non-symlink file", displayPath)
	}
	if info.Size() < 0 || info.Size() > maximum {
		return nil, fmt.Errorf("%s exceeds its %d-byte limit", displayPath, maximum)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", displayPath, err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat opened %s: %w", displayPath, err)
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return nil, fmt.Errorf("%s changed while being opened", displayPath)
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", displayPath, err)
	}
	if int64(len(data)) > maximum || int64(len(data)) != openedInfo.Size() {
		return nil, fmt.Errorf("%s changed size while being read", displayPath)
	}
	return data, nil
}

func buildArchives(root, version, sourceSHA, goVersion string, models []payloadFile) ([]releaseArchive, error) {
	license, err := readAllowedFile(root, "LICENSE", maxSmallFile)
	if err != nil {
		return nil, err
	}
	contract, err := readAllowedFile(root, "model-contract.json", maxSmallFile)
	if err != nil {
		return nil, err
	}
	archives := make([]releaseArchive, 0, len(targets))
	for _, target := range targets {
		binaries, err := buildTargetBinaries(root, target)
		if err != nil {
			return nil, fmt.Errorf("%s/%s: %w", target.goos, target.goarch, err)
		}
		files := []payloadFile{
			{path: "LICENSE", data: license, mode: 0o644},
			{path: "README.md", data: []byte(releaseReadme(version, sourceSHA, target)), mode: 0o644},
			{path: "model-contract.json", data: contract, mode: 0o644},
			{path: "bin/gooo-decision", data: binaries["gooo-decision"], mode: 0o755},
			{path: "bin/gooo-decision-stream", data: binaries["gooo-decision-stream"], mode: 0o755},
		}
		files = append(files, models...)
		manifest := makeManifest(version, sourceSHA, goVersion, target, files)
		manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return nil, err
		}
		manifestBytes = append(manifestBytes, '\n')
		files = append(files, payloadFile{path: "SOURCE-MANIFEST.json", data: manifestBytes, mode: 0o644})
		sums := makeSumsFile(files)
		files = append(files, payloadFile{path: "SHA256SUMS", data: sums, mode: 0o644})
		archive, err := deterministicArchive(files)
		if err != nil {
			return nil, fmt.Errorf("%s/%s: %w", target.goos, target.goarch, err)
		}
		name := fmt.Sprintf("gooo-ir-decision-%s-%s-%s.tar.gz", version, target.goos, target.goarch)
		archives = append(archives, releaseArchive{name: name, data: archive})
	}
	return archives, nil
}

func buildTargetBinaries(root string, target buildTarget) (map[string][]byte, error) {
	work, err := os.MkdirTemp("", "gooo-release-build-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)
	goEnv := buildEnvironment(target)
	listed, err := commandOutput(root, goEnv, "go", "list", "-f", "{{.ImportPath}}", "./cmd/gooo-decision", "./cmd/gooo-decision-stream")
	if err != nil {
		return nil, fmt.Errorf("go list %s/%s: %w", target.goos, target.goarch, err)
	}
	packages := strings.Fields(string(listed))
	expected := []string{
		"github.com/kimjooyoon/gooo-neural-decision-experiments/cmd/gooo-decision",
		"github.com/kimjooyoon/gooo-neural-decision-experiments/cmd/gooo-decision-stream",
	}
	if len(packages) != len(expected) || packages[0] != expected[0] || packages[1] != expected[1] {
		return nil, errors.New("go list returned an unexpected command set")
	}
	binaries := make(map[string][]byte, len(expected))
	commands := []struct{ name, packagePath string }{
		{name: "gooo-decision", packagePath: "./cmd/gooo-decision"},
		{name: "gooo-decision-stream", packagePath: "./cmd/gooo-decision-stream"},
	}
	for _, item := range commands {
		path := filepath.Join(work, item.name)
		if _, err := commandOutput(root, goEnv, "go", "build", "-trimpath", "-buildvcs=false", "-o", path, item.packagePath); err != nil {
			return nil, fmt.Errorf("build %s for %s/%s: %w", item.name, target.goos, target.goarch, err)
		}
		data, err := readRegularBoundedFile(path, item.name, maxBinaryFile)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("build %s produced an empty binary", item.name)
		}
		binaries[item.name] = data
	}
	return binaries, nil
}

func buildEnvironment(target buildTarget) []string {
	environment := make([]string, 0, len(os.Environ())+7)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "GOOS", "GOARCH", "GOARM", "GOARM64", "GOAMD64", "GO386", "GOMIPS", "GOMIPS64", "GOPPC64", "GORISCV64", "CGO_ENABLED", "GOFLAGS", "GOTOOLCHAIN", "GOWORK", "GOPROXY", "GOSUMDB", "GOENV", "GOEXPERIMENT":
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, targetBuildEnvironment(target)...)
}

func targetBuildEnvironment(target buildTarget) []string {
	environment := []string{
		"GOOS=" + target.goos,
		"GOARCH=" + target.goarch,
		"CGO_ENABLED=0",
		"GOFLAGS=",
		"GOTOOLCHAIN=local",
		"GOWORK=off",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOENV=off",
		"GOEXPERIMENT=",
	}
	if target.goarch == "amd64" {
		environment = append(environment, "GOAMD64=v1")
	} else if target.goarch == "arm64" {
		environment = append(environment, "GOARM64=v8.0")
	}
	return environment
}

func goVersion(root string) (string, error) {
	output, err := commandOutput(root, buildEnvironment(buildTarget{goos: "darwin", goarch: "arm64"}), "go", "version")
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(output))
	if !strings.HasPrefix(version, "go version go") {
		return "", fmt.Errorf("unexpected Go version output %q", version)
	}
	return version, nil
}

func commandOutput(directory string, environment []string, name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	command.Dir = directory
	if environment != nil {
		command.Env = environment
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func releaseReadme(version, sourceSHA string, target buildTarget) string {
	lines := []string{
		"# Gooo IR decision binaries (" + version + ")",
		"",
		fmt.Sprintf("Experimental, standalone command line tools for %s/%s. Built from source", target.goos, target.goarch),
		"revision " + sourceSHA + " with CGO disabled and Go build path trimming enabled.",
		"",
		"The model selects among eight supported typed binary IR operations. It does",
		"not generate arbitrary source code. A model may abstain; check the response",
		"status and typed IR before using a result.",
		"",
		"Included files:",
		"",
		"- bin/gooo-decision: one-request JSON interface",
		"- bin/gooo-decision-stream: bounded NDJSON stream interface",
		"- models/{fp32,ptq_ternary,qat_ternary}/: validated model bundles",
		"- model-contract.json: input and model contract",
		"- LICENSE: MIT license",
		"",
		"Examples (run from this directory):",
		"",
		"```sh",
		"./bin/gooo-decision --model models/fp32/model.json < request.json",
		"./bin/gooo-decision-stream --model models/qat_ternary/model.json --workers 4 < requests.ndjson",
		"```",
		"",
		"SHA256SUMS lists payload and source manifest digests. Verify with sha256sum -c SHA256SUMS on Linux.",
		"SOURCE-MANIFEST.json lists the payload file paths and SHA-256 digests. SHA256SUMS also covers that manifest.",
		"These binaries are not code-signed or notarized.",
	}
	return strings.Join(lines, "\n") + "\n"
}

func makeManifest(version, sourceSHA, goVersion string, target buildTarget, files []payloadFile) sourceManifest {
	ordered := append([]payloadFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].path < ordered[j].path })
	entries := make([]fileDigest, 0, len(ordered))
	for _, file := range ordered {
		entries = append(entries, fileDigest{Path: file.path, SHA256: hexDigest(file.data), Size: len(file.data)})
	}
	return sourceManifest{
		Schema: manifestSchema, Version: version, SourceRevision: sourceSHA,
		GoVersion: goVersion, Target: manifestTarget{GOOS: target.goos, GOARCH: target.goarch},
		SourcePackages:   []string{"cmd/gooo-decision", "cmd/gooo-decision-stream"},
		BuildFlags:       []string{"-trimpath", "-buildvcs=false"},
		BuildEnvironment: targetBuildEnvironment(target),
		Files:            entries,
	}
}

func makeSumsFile(files []payloadFile) []byte {
	ordered := append([]payloadFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].path < ordered[j].path })
	var sums strings.Builder
	for _, file := range ordered {
		fmt.Fprintf(&sums, "%s  %s\n", hexDigest(file.data), file.path)
	}
	return []byte(sums.String())
}

func deterministicArchive(files []payloadFile) ([]byte, error) {
	ordered := append([]payloadFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].path < ordered[j].path })
	for index, file := range ordered {
		if !safeArchivePath(file.path) || (index > 0 && ordered[index-1].path == file.path) {
			return nil, fmt.Errorf("unsafe or repeated archive path %q", file.path)
		}
	}
	var compressed bytes.Buffer
	zipWriter, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	zipWriter.Header.ModTime = time.Unix(0, 0).UTC()
	zipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(zipWriter)
	for _, file := range ordered {
		header := &tar.Header{
			Name: file.path, Mode: file.mode, Size: int64(len(file.data)), Typeflag: tar.TypeReg,
			ModTime: time.Unix(0, 0).UTC(), Uid: 0, Gid: 0, Format: tar.FormatUSTAR,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tarWriter.Write(file.data); err != nil {
			return nil, err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	if int64(compressed.Len()) > maxBundleBytes {
		return nil, fmt.Errorf("archive exceeds %d-byte limit", maxBundleBytes)
	}
	return compressed.Bytes(), nil
}

func safeArchivePath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\:") || strings.HasPrefix(value, "/") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	return clean == value && clean != ".." && !strings.HasPrefix(clean, "../")
}

func installArchives(outputPath string, archives []releaseArchive) (err error) {
	if err := os.Mkdir(outputPath, 0o755); err != nil {
		return err
	}
	installed := true
	defer func() {
		if err != nil && installed {
			_ = os.RemoveAll(outputPath)
		}
	}()
	ordered := append([]releaseArchive(nil), archives...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].name < ordered[j].name })
	for _, archive := range ordered {
		if filepath.Base(archive.name) != archive.name || !strings.HasSuffix(archive.name, ".tar.gz") {
			err = fmt.Errorf("invalid archive name %q", archive.name)
			return err
		}
		if err = writeExclusive(filepath.Join(outputPath, archive.name), archive.data); err != nil {
			return err
		}
	}
	var sums strings.Builder
	for _, archive := range ordered {
		fmt.Fprintf(&sums, "%s  %s\n", hexDigest(archive.data), archive.name)
	}
	if err = writeExclusive(filepath.Join(outputPath, "SHA256SUMS"), []byte(sums.String())); err != nil {
		return err
	}
	installed = false
	return nil
}

func writeExclusive(path string, data []byte) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()
	if _, err = file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}

func hexDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
