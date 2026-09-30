// Command package-release builds small, offline release archives from a clean
// committed source tree and the reviewed public model bundles.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultVersion        = "v0.2.0-experimental"
	modelRoot             = "runs/pilot-mps-20260930-v1/models"
	maxSmallFile          = 64 * 1024
	maxModelFile          = 1024 * 1024
	maxBinaryFile         = 16 * 1024 * 1024
	maxBundleBytes        = 32 * 1024 * 1024
	manifestSchema        = "gooo/experimental-release-source-manifest/v1"
	maxSnapshotEntries    = 20_000
	maxSnapshotListing    = 32 * 1024 * 1024
	maxSnapshotFileBytes  = 64 * 1024 * 1024
	maxSnapshotTotalBytes = 256 * 1024 * 1024
	modelSchema           = "gooo/tiny-ir-decision-model/v1"
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

type committedEntry struct {
	mode   string
	object string
	size   int64
}

type modelValidationResponse struct {
	Status        string `json:"status"`
	ModelVariant  string `json:"model_variant"`
	WeightsSHA256 string `json:"weights_sha256"`
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
		fmt.Fprintln(stderr, "usage: package-release --source-sha COMMIT --output NEW_DIR [--version v0.2.0-experimental]")
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
	snapshot, cleanupSnapshot, err := materializeGitSnapshot(root, *sourceSHA)
	if err != nil {
		fmt.Fprintf(stderr, "materialize committed source: %v\n", err)
		return 1
	}
	defer cleanupSnapshot()
	goVersion, err := goVersion(snapshot)
	if err != nil {
		fmt.Fprintf(stderr, "read Go version: %v\n", err)
		return 1
	}
	validator, cleanupValidator, err := buildModelValidator(snapshot)
	if err != nil {
		fmt.Fprintf(stderr, "build committed model validator: %v\n", err)
		return 1
	}
	defer cleanupValidator()
	models, err := loadModelPayloads(snapshot, validator)
	if err != nil {
		fmt.Fprintf(stderr, "validate model bundles: %v\n", err)
		return 1
	}
	archives, err := buildArchives(snapshot, *version, *sourceSHA, goVersion, models)
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

func materializeGitSnapshot(root, sourceSHA string) (string, func(), error) {
	entries, objectFormat, err := committedEntries(root, sourceSHA)
	if err != nil {
		return "", nil, err
	}
	snapshot, err := os.MkdirTemp("", "gooo-release-source-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(snapshot) }
	command := exec.Command("git", "archive", "--format=tar", sourceSHA)
	command.Dir = root
	stdout, err := command.StdoutPipe()
	if err != nil {
		cleanup()
		return "", nil, err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := extractCommittedArchive(stdout, snapshot, entries, objectFormat, sourceSHA); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		cleanup()
		return "", nil, fmt.Errorf("extract Git archive: %w", err)
	}
	if err := command.Wait(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git archive failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	for _, required := range []string{"go.mod", "LICENSE", "model-contract.json"} {
		if _, err := os.Stat(filepath.Join(snapshot, required)); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("committed snapshot is missing %s: %w", required, err)
		}
	}
	return snapshot, cleanup, nil
}

func committedEntries(root, sourceSHA string) (map[string]committedEntry, string, error) {
	formatOutput, err := commandOutput(root, nil, "git", "rev-parse", "--show-object-format")
	if err != nil {
		return nil, "", err
	}
	objectFormat := strings.TrimSpace(string(formatOutput))
	if objectFormat != "sha1" && objectFormat != "sha256" {
		return nil, "", fmt.Errorf("unsupported Git object format %q", objectFormat)
	}
	listing, err := commandOutputBounded(root, maxSnapshotListing, "git", "ls-tree", "-r", "-l", "-z", "--full-tree", sourceSHA)
	if err != nil {
		return nil, "", err
	}
	entries := make(map[string]committedEntry)
	directories := make(map[string]struct{})
	var totalBytes int64
	for _, record := range bytes.Split(listing, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		metadata, nameBytes, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return nil, "", errors.New("git ls-tree returned a malformed entry")
		}
		name := string(nameBytes)
		if !safeSnapshotPath(name) {
			return nil, "", fmt.Errorf("committed path is unsafe: %q", name)
		}
		fields := strings.Fields(string(metadata))
		if len(fields) != 4 || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
			return nil, "", fmt.Errorf("committed entry %q is not a regular source file", name)
		}
		objectID := fields[2]
		if (objectFormat == "sha1" && len(objectID) != 40) || (objectFormat == "sha256" && len(objectID) != 64) {
			return nil, "", fmt.Errorf("committed entry %q has an invalid object id", name)
		}
		for _, character := range objectID {
			if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
				return nil, "", fmt.Errorf("committed entry %q has an invalid object id", name)
			}
		}
		size, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil || size < 0 || size > maxSnapshotFileBytes {
			return nil, "", fmt.Errorf("committed entry %q has an invalid or oversized blob", name)
		}
		if _, exists := entries[name]; exists {
			return nil, "", fmt.Errorf("Git tree repeats path %q", name)
		}
		entries[name] = committedEntry{mode: fields[0], object: objectID, size: size}
		if len(entries)+len(directories) > maxSnapshotEntries {
			return nil, "", fmt.Errorf("committed snapshot exceeds %d entries", maxSnapshotEntries)
		}
		totalBytes += size
		if totalBytes > maxSnapshotTotalBytes {
			return nil, "", fmt.Errorf("committed source exceeds %d bytes", maxSnapshotTotalBytes)
		}
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			directories[parent] = struct{}{}
		}
	}
	for directory := range directories {
		if _, collides := entries[directory]; collides {
			return nil, "", fmt.Errorf("committed file %q is also a parent directory", directory)
		}
		entries[directory] = committedEntry{mode: "dir"}
	}
	if len(entries) == 0 || len(entries) > maxSnapshotEntries {
		return nil, "", fmt.Errorf("committed snapshot has %d entries; allowed range is 1..%d", len(entries), maxSnapshotEntries)
	}
	return entries, objectFormat, nil
}

func extractCommittedArchive(archive io.Reader, destination string, expected map[string]committedEntry, objectFormat, sourceSHA string) error {
	reader := tar.NewReader(archive)
	seen := make(map[string]struct{}, len(expected))
	var totalBytes int64
	globalHeaderSeen := false
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			if globalHeaderSeen || header.Name != "pax_global_header" || len(header.PAXRecords) != 1 || header.PAXRecords["comment"] != sourceSHA {
				return errors.New("archive has an unexpected global PAX header")
			}
			globalHeaderSeen = true
			continue
		}
		name := header.Name
		if header.Typeflag == tar.TypeDir {
			name = strings.TrimSuffix(name, "/")
		}
		if !safeSnapshotPath(name) {
			return fmt.Errorf("archive path is unsafe: %q", header.Name)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("archive repeats path %q", name)
		}
		entry, exists := expected[name]
		if !exists {
			return fmt.Errorf("archive contains uncommitted path %q", name)
		}
		seen[name] = struct{}{}
		path := filepath.Join(destination, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if entry.mode != "dir" || header.Size != 0 {
				return fmt.Errorf("unexpected directory entry %q", name)
			}
			if err := os.MkdirAll(path, 0o755); err != nil {
				return fmt.Errorf("create directory %q: %w", name, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if entry.mode == "dir" || header.Size != entry.size || header.Size < 0 || header.Size > maxSnapshotFileBytes {
				return fmt.Errorf("archive size or kind differs from committed entry %q", name)
			}
			permissions := int64(header.Mode & 0o777)
			if header.Mode&^0o777 != 0 || ((permissions&0o111 != 0) != (entry.mode == "100755")) {
				return fmt.Errorf("archive mode differs from committed entry %q", name)
			}
			totalBytes += header.Size
			if totalBytes > maxSnapshotTotalBytes {
				return fmt.Errorf("archive exceeds %d bytes", maxSnapshotTotalBytes)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("create parent for %q: %w", name, err)
			}
			committedMode := os.FileMode(0o644)
			if entry.mode == "100755" {
				committedMode = 0o755
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, committedMode)
			if err != nil {
				return fmt.Errorf("create committed file %q: %w", name, err)
			}
			hasher, err := gitObjectHasher(objectFormat)
			if err == nil {
				_, err = io.WriteString(hasher, fmt.Sprintf("blob %d\x00", header.Size))
			}
			if err == nil {
				_, err = io.CopyN(io.MultiWriter(file, hasher), reader, header.Size)
			}
			closeErr := file.Close()
			if err != nil {
				_ = os.Remove(path)
				return fmt.Errorf("write committed file %q: %w", name, err)
			}
			if closeErr != nil {
				_ = os.Remove(path)
				return fmt.Errorf("close committed file %q: %w", name, closeErr)
			}
			if hex.EncodeToString(hasher.Sum(nil)) != entry.object {
				return fmt.Errorf("archive content for %q does not match committed blob", name)
			}
		default:
			return fmt.Errorf("archive entry %q is not a regular file or directory", name)
		}
	}
	if len(seen) != len(expected) {
		for name := range expected {
			if _, exists := seen[name]; !exists {
				return fmt.Errorf("archive omitted committed path %q", name)
			}
		}
		return errors.New("archive omitted committed paths")
	}
	return nil
}

func gitObjectHasher(objectFormat string) (hash.Hash, error) {
	switch objectFormat {
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unsupported Git object format %q", objectFormat)
	}
}

func safeSnapshotPath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\:") || strings.ContainsRune(value, 0) || strings.HasPrefix(value, "/") || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	clean := path.Clean(value)
	if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
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

func loadModelPayloads(root, validatorBinary string) ([]payloadFile, error) {
	files := make([]payloadFile, 0, len(variants)*2)
	for _, variant := range variants {
		directory := filepath.Join(root, filepath.FromSlash(modelRoot), variant)
		metadataPath := filepath.Join(directory, "model.json")
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
		if metadata.Schema != modelSchema || metadata.Variant != variant || metadata.WeightsFile != "weights.bin" || metadata.WeightsSHA256 != hexDigest(weightBytes) {
			return nil, fmt.Errorf("%s model metadata does not bind the allowed weights file", variant)
		}
		if err := validateModelWithBinary(validatorBinary, metadataPath, variant, metadata.WeightsSHA256); err != nil {
			return nil, fmt.Errorf("%s: %w", variant, err)
		}
		files = append(files,
			payloadFile{path: filepath.ToSlash(filepath.Join("models", variant, "model.json")), data: metadataBytes, mode: 0o644},
			payloadFile{path: filepath.ToSlash(filepath.Join("models", variant, "weights.bin")), data: weightBytes, mode: 0o644},
		)
	}
	return files, nil
}

func buildModelValidator(snapshot string) (string, func(), error) {
	work, err := os.MkdirTemp("", "gooo-release-validator-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(work) }
	binary := filepath.Join(work, "gooo-decision")
	target := buildTarget{goos: runtime.GOOS, goarch: runtime.GOARCH}
	if _, err := commandOutput(snapshot, buildEnvironment(target), "go", "build", "-trimpath", "-buildvcs=false", "-o", binary, "./cmd/gooo-decision"); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("build native validation binary: %w", err)
	}
	if _, err := readRegularBoundedFile(binary, "gooo-decision validator", maxBinaryFile); err != nil {
		cleanup()
		return "", nil, err
	}
	return binary, cleanup, nil
}

func validateModelWithBinary(binary, metadataPath, variant, weightsSHA string) error {
	request := `{"schema":"gooo/tiny-ir-decision-request/v1","text":"add left and right values","left":{"name":"left","type":"Int"},"right":{"name":"right","type":"Int"}}` + "\n"
	command := exec.Command(binary, "--model", metadataPath)
	command.Stdin = strings.NewReader(request)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("committed CLI rejected the model bundle: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var response modelValidationResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return fmt.Errorf("decode committed CLI model-validation response: %w", err)
	}
	if (response.Status != "decision" && response.Status != "abstained") || response.ModelVariant != variant || response.WeightsSHA256 != weightsSHA {
		return fmt.Errorf("committed CLI response does not bind variant and weights: status=%q variant=%q weights=%q", response.Status, response.ModelVariant, response.WeightsSHA256)
	}
	return nil
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
			{path: "bin/gooo-body-compose", data: binaries["gooo-body-compose"], mode: 0o755},
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
	listed, err := commandOutput(root, goEnv, "go", "list", "-f", "{{.ImportPath}}",
		"./cmd/gooo-decision", "./cmd/gooo-decision-stream", "./cmd/gooo-body-compose")
	if err != nil {
		return nil, fmt.Errorf("go list %s/%s: %w", target.goos, target.goarch, err)
	}
	packages := strings.Fields(string(listed))
	expected := []string{
		"github.com/kimjooyoon/gooo-neural-decision-experiments/cmd/gooo-decision",
		"github.com/kimjooyoon/gooo-neural-decision-experiments/cmd/gooo-decision-stream",
		"github.com/kimjooyoon/gooo-neural-decision-experiments/cmd/gooo-body-compose",
	}
	if len(packages) != len(expected) {
		return nil, errors.New("go list returned an unexpected command set")
	}
	for index, packageName := range packages {
		if packageName != expected[index] {
			return nil, errors.New("go list returned an unexpected command set")
		}
	}
	binaries := make(map[string][]byte, len(expected))
	commands := []struct{ name, packagePath string }{
		{name: "gooo-decision", packagePath: "./cmd/gooo-decision"},
		{name: "gooo-decision-stream", packagePath: "./cmd/gooo-decision-stream"},
		{name: "gooo-body-compose", packagePath: "./cmd/gooo-body-compose"},
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

func commandOutputBounded(directory string, maximum int64, name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	command.Dir = directory
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	output, readErr := io.ReadAll(io.LimitReader(stdout, maximum+1))
	if readErr != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, readErr
	}
	if int64(len(output)) > maximum {
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, fmt.Errorf("%s output exceeds %d bytes", name, maximum)
	}
	if err := command.Wait(); err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
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
		"- bin/gooo-body-compose: typed multi-node plan to Gooo/Go source, emitted as data",
		"- models/{fp32,ptq_ternary,qat_ternary}/: validated model bundles",
		"- model-contract.json: input and model contract",
		"- LICENSE: MIT license",
		"",
		"Examples (run from this directory):",
		"",
		"```sh",
		"./bin/gooo-decision --model models/fp32/model.json < request.json",
		"./bin/gooo-decision-stream --model models/qat_ternary/model.json --workers 4 < requests.ndjson",
		"./bin/gooo-body-compose --model models/fp32/model.json < authored-plan.json",
		"./bin/gooo-body-compose < authored-plan.json # deterministic declared fallbacks",
		"```",
		"",
		"Body composition supports authored locals, assignment, branches and returns with",
		"finite typed operation holes. It does not execute the emitted code. Compilation",
		"through native Gooo requires a separate compiler. Laya is not bundled or required.",
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
		SourcePackages:   []string{"cmd/gooo-decision", "cmd/gooo-decision-stream", "cmd/gooo-body-compose"},
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
