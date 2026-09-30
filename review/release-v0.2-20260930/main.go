// Command main independently validates deterministic experimental release bundles.
// It never writes to the input build directories or contacts a network service.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	verificationSchema = "gooo/release-artifact-independent-verification/v1"
	maxArchiveBytes    = 32 << 20
	maxTarBytes        = 64 << 20
	maxSmallFile       = 64 << 10
	maxModelBytes      = 1 << 20
	maxBinaryBytes     = 16 << 20
	contractSHA        = "e1926dc964c6fd0fa671a68861190afe4ec059ec5d4e4e66f877ba119ef8e6e2"
	licenseSHA         = "3ad2cd8fe84a937a0005a2934e377432f2f86fe10ff86c4242cc48f49ebf4947"
)

var (
	versionRE = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)
	commitRE  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	targets   = []target{{"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}}
	variants  = []string{"fp32", "ptq_ternary", "qat_ternary"}
	labels    = []string{"add", "subtract", "multiply", "less_than", "less_equal", "equal", "and", "or"}
)

type target struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

type fileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size_bytes"`
}

type sourceManifest struct {
	Schema           string       `json:"schema"`
	Version          string       `json:"version"`
	SourceRevision   string       `json:"source_revision"`
	GoVersion        string       `json:"go_version"`
	Target           target       `json:"target"`
	SourcePackages   []string     `json:"source_packages"`
	BuildFlags       []string     `json:"build_flags"`
	BuildEnvironment []string     `json:"build_environment"`
	Files            []fileDigest `json:"files"`
}

type tensor struct {
	Bytes    int     `json:"bytes"`
	Cols     int     `json:"cols"`
	Count    int     `json:"count"`
	Encoding string  `json:"encoding"`
	Name     string  `json:"name"`
	Offset   int     `json:"offset"`
	Rows     int     `json:"rows"`
	Scale    float64 `json:"scale"`
}

type modelMetadata struct {
	ConfidenceThreshold float64  `json:"confidence_threshold"`
	FeatureDim          int      `json:"feature_dim"`
	HiddenDim           int      `json:"hidden_dim"`
	Labels              []string `json:"labels"`
	MaxBytes            int      `json:"max_bytes"`
	Schema              string   `json:"schema"`
	Temperature         float64  `json:"temperature"`
	Tensors             []tensor `json:"tensors"`
	Variant             string   `json:"variant"`
	WeightsFile         string   `json:"weights_file"`
	WeightsSHA256       string   `json:"weights_sha256"`
}

type modelContract struct {
	Schema       string   `json:"schema"`
	FeatureDim   int      `json:"feature_dim"`
	HiddenDim    int      `json:"hidden_dim"`
	LabelCount   int      `json:"label_count"`
	Labels       []string `json:"labels"`
	Input        string   `json:"input"`
	MaxBytes     int      `json:"max_bytes"`
	Network      string   `json:"network"`
	WeightLayout string   `json:"weight_layout"`
	Quantization string   `json:"quantization"`
	Runtime      string   `json:"runtime"`
	Training     string   `json:"training"`
	Scope        string   `json:"scope"`
}

type archiveReceipt struct {
	Filename             string            `json:"filename"`
	SHA256               string            `json:"sha256"`
	Target               target            `json:"target"`
	GoVersion            string            `json:"go_version"`
	PayloadFiles         int               `json:"payload_file_count"`
	PayloadSHA256        map[string]string `json:"payload_sha256"`
	SourceManifestSHA256 string            `json:"source_manifest_sha256"`
}

type receipt struct {
	Schema                  string           `json:"schema"`
	Decision                string           `json:"decision"`
	Version                 string           `json:"version"`
	SourceRevision          string           `json:"source_revision"`
	IndependentBuildsEqual  bool             `json:"independent_builds_byte_identical"`
	ArchiveReceipts         []archiveReceipt `json:"archives"`
	DarwinSmoke             map[string]any   `json:"darwin_smoke"`
	LinuxExecutionPerformed bool             `json:"linux_execution_performed"`
	Caveats                 []string         `json:"caveats"`
}

type archivedFile struct {
	Data []byte
	Mode int64
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("release-artifact-review", flag.ContinueOnError)
	first := flags.String("first", "", "first completed release output directory")
	second := flags.String("second", "", "independent second release output directory")
	source := flags.String("source-sha", "", "expected lowercase 40-character source commit")
	version := flags.String("version", "v0.2.0-experimental", "expected release version")
	output := flags.String("out", "", "new verification receipt path; it must not exist")
	smoke := flags.Bool("smoke-darwin", false, "run the three packaged darwin/arm64 CLIs offline on public synthetic examples")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *first == "" || *second == "" || *source == "" || *output == "" {
		return errors.New("usage: go run ./review/release-v0.2-20260930 --first DIR --second DIR --source-sha COMMIT --version VERSION --out NEW_FILE [--smoke-darwin]")
	}
	if !commitRE.MatchString(*source) || !versionRE.MatchString(*version) {
		return errors.New("source SHA or version is malformed")
	}
	firstPath, err := filepath.Abs(*first)
	if err != nil {
		return err
	}
	secondPath, err := filepath.Abs(*second)
	if err != nil {
		return err
	}
	firstPath, err = filepath.EvalSymlinks(firstPath)
	if err != nil {
		return err
	}
	secondPath, err = filepath.EvalSymlinks(secondPath)
	if err != nil {
		return err
	}
	outputPath, err := filepath.Abs(*output)
	if err != nil {
		return err
	}
	outputParent, err := filepath.EvalSymlinks(filepath.Dir(outputPath))
	if err != nil {
		return err
	}
	outputPath = filepath.Join(outputParent, filepath.Base(outputPath))
	if firstPath == secondPath {
		return errors.New("independent build directories must be different")
	}
	if err := requireFreshOutput(outputPath, firstPath, secondPath); err != nil {
		return err
	}
	firstFiles, firstSums, err := readBuildDirectory(firstPath, *version)
	if err != nil {
		return fmt.Errorf("first build: %w", err)
	}
	secondFiles, secondSums, err := readBuildDirectory(secondPath, *version)
	if err != nil {
		return fmt.Errorf("second build: %w", err)
	}
	if !bytes.Equal(firstSums, secondSums) || !sameBuildBytes(firstFiles, secondFiles) {
		return errors.New("two independently packaged directories are not byte-identical")
	}
	receipts := make([]archiveReceipt, 0, len(targets))
	var darwinFiles map[string]archivedFile
	goVersion := ""
	for _, item := range targets {
		name := archiveName(*version, item)
		verified, err := verifyArchive(firstFiles[name], *version, *source, item)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		receipts = append(receipts, verified.receipt)
		if goVersion == "" {
			goVersion = verified.receipt.GoVersion
		} else if verified.receipt.GoVersion != goVersion {
			return errors.New("target archives record different Go toolchain versions")
		}
		if item.GOOS == "darwin" && item.GOARCH == "arm64" {
			darwinFiles = verified.files
		}
	}
	smokeReceipt := map[string]any{"attempted": false, "reason": "not requested"}
	if *smoke {
		smokeReceipt, err = smokeDarwin(darwinFiles)
		if err != nil {
			return fmt.Errorf("Darwin CLI smoke: %w", err)
		}
	}
	result := receipt{
		Schema: verificationSchema, Decision: "PASS", Version: *version, SourceRevision: *source,
		IndependentBuildsEqual: true, ArchiveReceipts: receipts, DarwinSmoke: smokeReceipt,
		LinuxExecutionPerformed: false,
		Caveats: []string{
			"The source commit value and release version are caller-pinned and checked against every archive manifest and generated README. This verifier does not establish the repository commit's authenticity or rebuild the source itself.",
			"Linux binaries are validated by archive structure, source manifest, per-file hashes and independent byte-for-byte rebuild comparison only; they are not executed on this host.",
			"Archive reproducibility confirms identical outputs from the two supplied directories. The caller must establish that the directories came from separate clean builds of the pinned source and toolchain.",
		},
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if err := writeFresh(outputPath, encoded); err != nil {
		return err
	}
	fmt.Printf("PASS: %d target archives; independent outputs byte-identical; receipt %s\n", len(receipts), digest(encoded))
	return nil
}

func requireFreshOutput(output string, inputs ...string) error {
	if _, err := os.Lstat(output); err == nil {
		return errors.New("verification output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parentInfo, err := os.Stat(filepath.Dir(output))
	if err != nil || !parentInfo.IsDir() {
		return errors.New("verification output parent must already exist")
	}
	for _, input := range inputs {
		relative, err := filepath.Rel(input, output)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("verification output must be outside both input directories")
		}
	}
	return nil
}

func readBuildDirectory(directory, version string) (map[string][]byte, []byte, error) {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, errors.New("build path must be a real directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, nil, err
	}
	expected := map[string]bool{"SHA256SUMS": true}
	for _, item := range targets {
		expected[archiveName(version, item)] = true
	}
	if len(entries) != len(expected) {
		return nil, nil, fmt.Errorf("build directory has %d entries, expected %d", len(entries), len(expected))
	}
	files := make(map[string][]byte, len(targets))
	var sums []byte
	for _, entry := range entries {
		name := entry.Name()
		if !expected[name] || entry.IsDir() {
			return nil, nil, fmt.Errorf("unexpected build output %q", name)
		}
		mode := entry.Type()
		if mode&os.ModeSymlink != 0 || mode.IsDir() || !mode.IsRegular() {
			return nil, nil, fmt.Errorf("build output %q is not a regular file", name)
		}
		maximum := int64(maxArchiveBytes)
		if name == "SHA256SUMS" {
			maximum = 4096
		}
		data, err := readBounded(filepath.Join(directory, name), maximum)
		if err != nil {
			return nil, nil, err
		}
		if name == "SHA256SUMS" {
			sums = data
		} else {
			files[name] = data
		}
	}
	if len(files) != len(targets) || len(sums) == 0 {
		return nil, nil, errors.New("build directory lacks expected archives or outer checksums")
	}
	if !bytes.Equal(sums, checksumBytes(files)) {
		return nil, nil, errors.New("outer SHA256SUMS is not the exact sorted digest list for the three archives")
	}
	return files, sums, nil
}

func readBounded(filename string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 0 || info.Size() > maximum {
		return nil, fmt.Errorf("file %s is non-regular or outside its size bound", filepath.Base(filename))
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, errors.New("file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(data)) > maximum || int64(len(data)) != opened.Size() {
		return nil, errors.New("file changed size while reading or exceeded limit")
	}
	return data, nil
}

func sameBuildBytes(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for name, content := range a {
		if !bytes.Equal(content, b[name]) {
			return false
		}
	}
	return true
}

func verifyArchive(archive []byte, version, sourceSHA string, expectedTarget target) (verifiedArchive, error) {
	if len(archive) == 0 || len(archive) > maxArchiveBytes {
		return verifiedArchive{}, errors.New("archive size outside bound")
	}
	files, err := unpack(archive)
	if err != nil {
		return verifiedArchive{}, err
	}
	wanted := archivePaths()
	if len(files) != len(wanted) {
		return verifiedArchive{}, fmt.Errorf("archive has %d files; expected exactly %d", len(files), len(wanted))
	}
	for _, name := range wanted {
		if _, exists := files[name]; !exists {
			return verifiedArchive{}, fmt.Errorf("archive is missing %s", name)
		}
	}
	for name := range files {
		if !contains(wanted, name) {
			return verifiedArchive{}, fmt.Errorf("archive includes non-allowlisted path %q", name)
		}
	}
	if err := checkNoLocalPaths(files); err != nil {
		return verifiedArchive{}, err
	}
	if err := checkModes(files); err != nil {
		return verifiedArchive{}, err
	}
	for _, name := range []string{"gooo-decision", "gooo-decision-stream", "gooo-body-compose"} {
		if err := validateBinaryFormat(files["bin/"+name].Data, expectedTarget); err != nil {
			return verifiedArchive{}, fmt.Errorf("%s: %w", name, err)
		}
	}
	manifestBytes := files["SOURCE-MANIFEST.json"].Data
	var manifest sourceManifest
	if err := decodeStrict(manifestBytes, &manifest); err != nil {
		return verifiedArchive{}, fmt.Errorf("source manifest: %w", err)
	}
	canonical, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return verifiedArchive{}, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(manifestBytes, canonical) {
		return verifiedArchive{}, errors.New("source manifest JSON is not in normalized release form")
	}
	if err := validateManifest(manifest, version, sourceSHA, expectedTarget, files); err != nil {
		return verifiedArchive{}, err
	}
	if !bytes.Equal(files["SHA256SUMS"].Data, checksumArchivePayloads(files)) {
		return verifiedArchive{}, errors.New("inner SHA256SUMS does not exactly cover sorted payload and source manifest files")
	}
	if digest(files["LICENSE"].Data) != licenseSHA {
		return verifiedArchive{}, errors.New("LICENSE digest differs from reviewed MIT license")
	}
	if err := validateReadmeExact(files["README.md"].Data, version, sourceSHA, expectedTarget); err != nil {
		return verifiedArchive{}, err
	}
	if digest(files["model-contract.json"].Data) != contractSHA {
		return verifiedArchive{}, errors.New("model contract digest differs from reviewed public contract")
	}
	if err := validateContract(files["model-contract.json"].Data); err != nil {
		return verifiedArchive{}, err
	}
	payloadSHA := make(map[string]string, len(manifest.Files))
	for _, entry := range manifest.Files {
		payloadSHA[entry.Path] = entry.SHA256
	}
	return verifiedArchive{
		files: files,
		receipt: archiveReceipt{
			Filename: archiveName(version, expectedTarget), SHA256: digest(archive), Target: expectedTarget,
			GoVersion: manifest.GoVersion, PayloadFiles: len(manifest.Files), PayloadSHA256: payloadSHA,
			SourceManifestSHA256: digest(manifestBytes),
		},
	}, nil
}

type verifiedArchive struct {
	files   map[string]archivedFile
	receipt archiveReceipt
}

func unpack(archive []byte) (map[string]archivedFile, error) {
	source := bytes.NewReader(archive)
	gz, err := gzip.NewReader(source)
	if err != nil {
		return nil, err
	}
	gz.Multistream(false)
	if !gz.ModTime.IsZero() || gz.OS != 255 || gz.Name != "" || gz.Comment != "" || len(gz.Extra) != 0 {
		_ = gz.Close()
		return nil, errors.New("gzip header is not normalized")
	}
	decompressed, err := io.ReadAll(io.LimitReader(gz, maxTarBytes+1))
	closeErr := gz.Close()
	if err != nil || closeErr != nil || int64(len(decompressed)) > maxTarBytes {
		return nil, errors.New("gzip stream is invalid or exceeds decompressed size limit")
	}
	if source.Len() != 0 {
		return nil, errors.New("archive contains trailing compressed data or multiple gzip members")
	}
	reader := bytes.NewReader(decompressed)
	tarReader := tar.NewReader(reader)
	files := make(map[string]archivedFile)
	lastName := ""
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if !safePath(header.Name) || header.Typeflag != tar.TypeReg || header.Format != tar.FormatUSTAR || header.Uid != 0 || header.Gid != 0 || header.Uname != "" || header.Gname != "" || !header.ModTime.Equal(time.Unix(0, 0).UTC()) || !header.AccessTime.IsZero() || !header.ChangeTime.IsZero() || header.Linkname != "" || header.PAXRecords != nil || header.Devmajor != 0 || header.Devminor != 0 {
			return nil, fmt.Errorf("archive member %q is not a normalized regular USTAR file", header.Name)
		}
		if header.Name <= lastName {
			return nil, errors.New("archive paths are not in strict lexical order")
		}
		lastName = header.Name
		if _, duplicate := files[header.Name]; duplicate || header.Size < 0 || header.Size > maxBinaryBytes {
			return nil, fmt.Errorf("archive member %q duplicates or exceeds size limit", header.Name)
		}
		data, err := io.ReadAll(io.LimitReader(tarReader, maxBinaryBytes+1))
		if err != nil || int64(len(data)) != header.Size || int64(len(data)) > maxBinaryBytes {
			return nil, fmt.Errorf("archive member %q has invalid content size", header.Name)
		}
		files[header.Name] = archivedFile{Data: data, Mode: header.Mode}
	}
	trailing, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	for _, value := range trailing {
		if value != 0 {
			return nil, errors.New("tar archive has nonzero trailing bytes")
		}
	}
	return files, nil
}

func validateManifest(manifest sourceManifest, version, sourceSHA string, targetExpected target, files map[string]archivedFile) error {
	if manifest.Schema != "gooo/experimental-release-source-manifest/v1" || manifest.Version != version || manifest.SourceRevision != sourceSHA || manifest.Target != targetExpected {
		return errors.New("source manifest schema, version, commit or target does not match requested pins")
	}
	if !strings.HasPrefix(manifest.GoVersion, "go version go") || !strings.HasSuffix(manifest.GoVersion, " darwin/arm64") {
		return errors.New("Go toolchain version field is malformed or not the packager's pinned query target")
	}
	if !equalStrings(manifest.SourcePackages, []string{"cmd/gooo-decision", "cmd/gooo-decision-stream", "cmd/gooo-body-compose"}) || !equalStrings(manifest.BuildFlags, []string{"-trimpath", "-buildvcs=false"}) {
		return errors.New("source package list or normalized build flags differ from release contract")
	}
	if !equalStrings(manifest.BuildEnvironment, environmentFor(targetExpected)) {
		return errors.New("build environment fields differ from the fixed target policy")
	}
	wanted := payloadPaths()
	if len(manifest.Files) != len(wanted) {
		return fmt.Errorf("source manifest has %d file entries; expected %d payload files", len(manifest.Files), len(wanted))
	}
	for index, filename := range wanted {
		entry := manifest.Files[index]
		file, ok := files[filename]
		if !ok || entry.Path != filename || entry.Size != len(file.Data) || entry.SHA256 != digest(file.Data) {
			return fmt.Errorf("source manifest entry %d does not bind allowlisted payload %s", index, filename)
		}
	}
	for _, variant := range variants {
		if err := validateModel(files["models/"+variant+"/model.json"].Data, files["models/"+variant+"/weights.bin"].Data, variant); err != nil {
			return err
		}
	}
	return nil
}

func validateModel(metadataBytes, weights []byte, variant string) error {
	var metadata modelMetadata
	if err := decodeStrict(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("%s model metadata: %w", variant, err)
	}
	expectedModels := map[string][2]string{
		"fp32":        {"c5867ee872cbf0fb8227015bd3e94540415786437704ed01b66e6bb0b421c930", "854da0f8fc0dcf5560d1f1b8838ea30081752020dbc2005772207217c5d1d17a"},
		"ptq_ternary": {"5766a32a9d70ef16c895e7cf8765d7884d59ab6241fcfa225ac2291731a77a4e", "3c60ab7154f02f04a94b1bc68274e05c4ae86f10185b19dbe7251c26610a5c8b"},
		"qat_ternary": {"4746fc7cbd2864e9536b43d779c7ecc39aa35f8f6f64943c73b951f9e967c8a7", "bd4444d5233f07c18168a4dfa0053afde180ce97e390a993e4e0fbb33fb88337"},
	}
	pins, ok := expectedModels[variant]
	if !ok || digest(metadataBytes) != pins[0] || digest(weights) != pins[1] || metadata.Schema != "gooo/tiny-ir-decision-model/v1" || metadata.Variant != variant || metadata.WeightsFile != "weights.bin" || metadata.WeightsSHA256 != digest(weights) {
		return fmt.Errorf("%s model metadata or weights differ from public model pins", variant)
	}
	if metadata.FeatureDim != 256 || metadata.HiddenDim != 48 || metadata.MaxBytes != 512 || metadata.ConfidenceThreshold != 0.125 || len(metadata.Labels) != len(labels) || !equalStrings(metadata.Labels, labels) || len(metadata.Tensors) != 4 || !positiveFinite(metadata.Temperature) {
		return fmt.Errorf("%s model dimensions, labels or numeric parameters are invalid", variant)
	}
	wantedTensorShapes := []tensor{{Name: "w1", Rows: 48, Cols: 256, Count: 12288}, {Name: "b1", Rows: 1, Cols: 48, Count: 48}, {Name: "w2", Rows: 8, Cols: 48, Count: 384}, {Name: "b2", Rows: 1, Cols: 8, Count: 8}}
	offset := 0
	for index, got := range metadata.Tensors {
		shape := wantedTensorShapes[index]
		if got.Name != shape.Name || got.Rows != shape.Rows || got.Cols != shape.Cols || got.Count != shape.Count || got.Offset != offset || !positiveFinite(got.Scale) {
			return fmt.Errorf("%s model tensor %d has invalid name, dimensions, offset or scale", variant, index)
		}
		encoding := "float32_le"
		wantBytes := got.Count * 4
		if variant != "fp32" && (got.Name == "w1" || got.Name == "w2") {
			encoding = "ternary_base3_5"
			wantBytes = (got.Count + 4) / 5
		} else if got.Scale != 1 {
			return fmt.Errorf("%s unquantized tensor scale must equal one", variant)
		}
		if got.Encoding != encoding || got.Bytes != wantBytes {
			return fmt.Errorf("%s tensor %s encoding or byte count invalid", variant, got.Name)
		}
		offset += got.Bytes
	}
	if offset != len(weights) {
		return fmt.Errorf("%s model tensor layout size does not equal weights length", variant)
	}
	return nil
}

func validateContract(data []byte) error {
	var contract modelContract
	if err := decodeStrict(data, &contract); err != nil {
		return err
	}
	if contract.Schema != "gooo/tiny-ir-decision-model-contract/v1" || contract.FeatureDim != 256 || contract.HiddenDim != 48 || contract.LabelCount != 8 || contract.MaxBytes != 512 || !equalStrings(contract.Labels, labels) || !strings.Contains(contract.Scope, "cannot generate arbitrary Gooo source") || !strings.Contains(contract.Runtime, "Go only") {
		return errors.New("model contract metadata differs from the reviewed typed decision contract")
	}
	return nil
}

func checkModes(files map[string]archivedFile) error {
	for name, file := range files {
		want := int64(0o644)
		if strings.HasPrefix(name, "bin/") {
			want = 0o755
			if len(file.Data) == 0 || len(file.Data) > maxBinaryBytes {
				return fmt.Errorf("binary %s is empty or exceeds size cap", name)
			}
		} else if len(file.Data) > maxSmallFile && !strings.HasPrefix(name, "models/") {
			return fmt.Errorf("small payload %s exceeds size cap", name)
		}
		if strings.HasPrefix(name, "models/") && len(file.Data) > maxModelBytes {
			return fmt.Errorf("model file %s exceeds size cap", name)
		}
		if file.Mode != want {
			return fmt.Errorf("archive mode for %s is %o, expected %o", name, file.Mode, want)
		}
	}
	return nil
}

func checkNoLocalPaths(files map[string]archivedFile) error {
	patterns := [][]byte{[]byte("/Users/"), []byte("/home/"), []byte("/tmp/"), []byte("/private/var/"), []byte("/var/folders/"), []byte(`C:\Users\`)}
	for name, file := range files {
		if strings.HasPrefix(name, "models/") && strings.HasSuffix(name, "weights.bin") {
			continue
		}
		for _, pattern := range patterns {
			if bytes.Contains(file.Data, pattern) {
				return fmt.Errorf("payload %s contains a local filesystem path marker", name)
			}
		}
	}
	return nil
}

func validateBinaryFormat(data []byte, item target) error {
	switch item.GOOS {
	case "darwin":
		binary, err := macho.NewFile(bytes.NewReader(data))
		if err != nil || binary.Cpu != macho.CpuArm64 {
			return errors.New("binary is not a valid darwin/arm64 Mach-O executable")
		}
	case "linux":
		binary, err := elf.NewFile(bytes.NewReader(data))
		if err != nil || binary.Class != elf.ELFCLASS64 {
			return errors.New("binary is not a valid 64-bit Linux ELF executable")
		}
		if item.GOARCH == "amd64" && binary.Machine != elf.EM_X86_64 {
			return errors.New("Linux/amd64 binary has the wrong ELF machine")
		}
		if item.GOARCH == "arm64" && binary.Machine != elf.EM_AARCH64 {
			return errors.New("Linux/arm64 binary has the wrong ELF machine")
		}
	default:
		return errors.New("unsupported binary target")
	}
	return nil
}

func smokeDarwin(files map[string]archivedFile) (map[string]any, error) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return nil, errors.New("Darwin smoke can run only on a darwin/arm64 host")
	}
	root, err := os.MkdirTemp("", "gooo-release-v02-smoke-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)
	for name, file := range files {
		if err := writeWithin(root, name, file.Data, os.FileMode(file.Mode)); err != nil {
			return nil, err
		}
	}
	request := []byte(`{"schema":"gooo/tiny-ir-decision-request/v1","text":"Add the left and right values to make a total.","left":{"name":"left","type":"Int"},"right":{"name":"right","type":"Int"}}`)
	streamRequest := []byte(`{"schema":"gooo/tiny-ir-decision-stream-request/v1","correlation_id":"public-smoke","request":{"schema":"gooo/tiny-ir-decision-request/v1","text":"Add the left and right values to make a total.","left":{"name":"left","type":"Int"},"right":{"name":"right","type":"Int"}}}`)
	plan := []byte(`{"schema":"gooo/typed-body-plan/v1","id":"release-smoke","name":"ReleaseSmoke","result_type":"Int","expressions":[{"kind":"input","name":"input"},{"kind":"int","int":1},{"kind":"hole","left":0,"right":1,"hole_id":"op","text":"Add the input and one to make a total.","allowed":["add","subtract","multiply"],"fallback":"multiply"}],"statements":[{"kind":"return","expr":2}],"root":[0]}`)
	tests := 0
	for _, variant := range variants {
		modelPath := filepath.Join(root, "models", variant, "model.json")
		if err := runCLI(root, "bin/gooo-decision", []string{"--model", modelPath}, request, func(output []byte) error {
			var response struct {
				Schema       string `json:"schema"`
				ModelVariant string `json:"model_variant"`
				Status       string `json:"status"`
			}
			if err := decodeEvidence(output, &response); err != nil {
				return err
			}
			if response.Schema != "gooo/tiny-ir-decision-response/v1" || response.ModelVariant != variant || (response.Status != "decision" && response.Status != "abstained") {
				return errors.New("gooo-decision returned unexpected response metadata")
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("gooo-decision/%s: %w", variant, err)
		}
		tests++
		if err := runCLI(root, "bin/gooo-decision-stream", []string{"--model", modelPath, "--workers", "1"}, append(streamRequest, '\n'), func(output []byte) error {
			lines := bytes.Split(bytes.TrimSpace(output), []byte{'\n'})
			if len(lines) != 1 {
				return errors.New("decision stream did not emit exactly one result")
			}
			var response struct {
				Schema        string `json:"schema"`
				CorrelationID string `json:"correlation_id"`
				Status        string `json:"status"`
				Response      *struct {
					ModelVariant string `json:"model_variant"`
				} `json:"response"`
			}
			if err := decodeEvidence(lines[0], &response); err != nil {
				return err
			}
			if response.Schema != "gooo/tiny-ir-decision-stream-result/v1" || response.CorrelationID != "public-smoke" || response.Status != "completed" || response.Response == nil || response.Response.ModelVariant != variant {
				return errors.New("decision stream returned unexpected response metadata")
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("gooo-decision-stream/%s: %w", variant, err)
		}
		tests++
		if err := runCLI(root, "bin/gooo-body-compose", []string{"--model", modelPath}, plan, func(output []byte) error {
			var response struct {
				Schema    string `json:"schema"`
				Selection struct {
					ModelVariant string `json:"model_variant"`
				} `json:"initial_selection"`
				GoSource   string `json:"go_source"`
				GoooSource string `json:"gooo_source"`
			}
			if err := decodeEvidence(output, &response); err != nil {
				return err
			}
			if response.Schema != "gooo/typed-body-composition-result/v1" || response.Selection.ModelVariant != variant || response.GoSource == "" || response.GoooSource == "" {
				return errors.New("body compose returned unexpected output metadata")
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("gooo-body-compose/%s: %w", variant, err)
		}
		tests++
	}
	return map[string]any{"attempted": true, "host": "darwin/arm64", "cli_invocations_passed": tests, "cli_invocations_planned": 9, "models_loaded_by_each_cli": variants, "network_used": false, "linux_execution_performed": false}, nil
}

func runCLI(root, relative string, args []string, input []byte, validate func([]byte) error) error {
	filename := filepath.Join(root, filepath.FromSlash(relative))
	data, err := readBounded(filename, maxBinaryBytes)
	if err != nil {
		return err
	}
	file, err := macho.NewFile(bytes.NewReader(data))
	if err != nil || file.Cpu != macho.CpuArm64 {
		return errors.New("smoke executable is not a darwin/arm64 Mach-O binary")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, filename, args...)
	command.Dir = root
	command.Env = []string{"PATH=" + os.Getenv("PATH")}
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr limitedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("command failed (stderr=%q): %w", stderr.String(), err)
	}
	if err := validate(stdout.Bytes()); err != nil {
		return fmt.Errorf("invalid CLI response: %w", err)
	}
	return nil
}

type limitedBuffer struct {
	buffer bytes.Buffer
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if b.buffer.Len()+len(data) > 1<<20 {
		return 0, errors.New("CLI output exceeds one mebibyte")
	}
	return b.buffer.Write(data)
}

func (b *limitedBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *limitedBuffer) String() string { return b.buffer.String() }

func writeWithin(root, relative string, data []byte, mode os.FileMode) error {
	if !safePath(relative) {
		return errors.New("unsafe archive path while preparing smoke")
	}
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func decodeStrict(data []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("JSON contains trailing data")
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("JSON contains trailing value")
	}
	return nil
}

func decodeEvidence(data []byte, output any) error {
	if err := checkJSON(data); err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}

func checkJSON(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("JSON is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("JSON contains trailing data")
	}
	return nil
}

func scanValue(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("JSON nesting exceeds 32 levels")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		var keys []string
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid JSON object key")
			}
			for _, prior := range keys {
				if strings.EqualFold(prior, key) {
					return fmt.Errorf("duplicate or case-folded JSON key %q", key)
				}
			}
			keys = append(keys, key)
			if err := scanValue(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := scanValue(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	_, err = decoder.Token()
	return err
}

func archivePaths() []string {
	paths := append(payloadPaths(), "SHA256SUMS", "SOURCE-MANIFEST.json")
	sort.Strings(paths)
	return paths
}

func payloadPaths() []string {
	paths := []string{"LICENSE", "README.md", "model-contract.json", "bin/gooo-decision", "bin/gooo-decision-stream", "bin/gooo-body-compose"}
	for _, variant := range variants {
		paths = append(paths, "models/"+variant+"/model.json", "models/"+variant+"/weights.bin")
	}
	sort.Strings(paths)
	return paths
}

func archiveName(version string, item target) string {
	return fmt.Sprintf("gooo-ir-decision-%s-%s-%s.tar.gz", version, item.GOOS, item.GOARCH)
}

func environmentFor(item target) []string {
	values := []string{"GOOS=" + item.GOOS, "GOARCH=" + item.GOARCH, "CGO_ENABLED=0", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOENV=off", "GOEXPERIMENT="}
	if item.GOARCH == "amd64" {
		values = append(values, "GOAMD64=v1")
	} else {
		values = append(values, "GOARM64=v8.0")
	}
	return values
}

func checksumBytes(files map[string][]byte) []byte {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var result strings.Builder
	for _, name := range names {
		fmt.Fprintf(&result, "%s  %s\n", digest(files[name]), name)
	}
	return []byte(result.String())
}

func checksumArchivePayloads(files map[string]archivedFile) []byte {
	data := make(map[string][]byte, len(files)-1)
	for name, file := range files {
		if name != "SHA256SUMS" {
			data[name] = file.Data
		}
	}
	return checksumBytes(data)
}

func validateReadmeExact(data []byte, version, sourceSHA string, item target) error {
	expected := releaseReadme(version, sourceSHA, item)
	if !bytes.Equal(data, []byte(expected)) {
		return errors.New("README differs from normalized release instructions or includes unreviewed text")
	}
	return nil
}

func releaseReadme(version, sourceSHA string, target target) string {
	lines := []string{
		"# Gooo IR decision binaries (" + version + ")", "",
		fmt.Sprintf("Experimental, standalone command line tools for %s/%s. Built from source", target.GOOS, target.GOARCH),
		"revision " + sourceSHA + " with CGO disabled and Go build path trimming enabled.", "",
		"The model selects among eight supported typed binary IR operations. It does",
		"not generate arbitrary source code. A model may abstain; check the response",
		"status and typed IR before using a result.", "", "Included files:", "",
		"- bin/gooo-decision: one-request JSON interface", "- bin/gooo-decision-stream: bounded NDJSON stream interface",
		"- bin/gooo-body-compose: typed multi-node plan to Gooo/Go source, emitted as data",
		"- models/{fp32,ptq_ternary,qat_ternary}/: validated model bundles",
		"- model-contract.json: input and model contract", "- LICENSE: MIT license", "",
		"Examples (run from this directory):", "", "```sh",
		"./bin/gooo-decision --model models/fp32/model.json < request.json",
		"./bin/gooo-decision-stream --model models/qat_ternary/model.json --workers 4 < requests.ndjson",
		"./bin/gooo-body-compose --model models/fp32/model.json < authored-plan.json",
		"./bin/gooo-body-compose < authored-plan.json # deterministic declared fallbacks", "```", "",
		"Body composition supports authored locals, assignment, branches and returns with",
		"finite typed operation holes. It does not execute the emitted code. Compilation",
		"through native Gooo requires a separate compiler. Laya is not bundled or required.", "",
		"SHA256SUMS lists payload and source manifest digests. Verify with sha256sum -c SHA256SUMS on Linux.",
		"SOURCE-MANIFEST.json lists the payload file paths and SHA-256 digests. SHA256SUMS also covers that manifest.",
		"These binaries are not code-signed or notarized.",
	}
	return strings.Join(lines, "\n") + "\n"
}

func safePath(name string) bool {
	if name == "" || strings.ContainsAny(name, "\\:") || strings.HasPrefix(name, "/") || path.Clean(name) != name {
		return false
	}
	return name != ".." && !strings.HasPrefix(name, "../") && !strings.Contains(name, "//")
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func writeFresh(filename string, data []byte) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(filename)
		return err
	}
	return file.Close()
}
