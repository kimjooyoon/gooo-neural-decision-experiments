// package-three-curriculum streams frozen source/teacher evidence into a public
// ZIP and verifies every decoded member without operational model calls.
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

const decodedCap int64 = 768 << 20
const archiveCap int64 = 64 << 20
const lineCap = 1 << 20

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)
var phases = map[string][]string{
	"own-three-composition-curriculum-20261002":       {"collection-attempt.json", "dataset.jsonl", "exports.jsonl", "fixtures.jsonl", "preexecution.json"},
	"own-three-composition-curriculum-fixed-20261002": {"collection-attempt.json", "dataset.jsonl", "exports.jsonl", "fixtures.jsonl", "manifest.json", "preexecution.json"},
	"own-three-teacher-curriculum-20261002":           {"collection-attempt.json", "preexecution.json", "report.json", "states.jsonl", "teacher-sessions.jsonl"},
}
var supplements = map[string]string{
	"protocol.md":                              threefeedback.Protocol,
	"source-audit.json":                        "publication/own-three-choice-curriculum-audit-20261002.json",
	"teacher-audit.json":                       "publication/own-three-choice-teacher-audit-20261002.json",
	"first-source-negative.json":               "publication/own-three-choice-first-collection-negative-20261002.json",
	"reference-models/independent/model.json":  "models/joint-composition-v1/independent/models/fp32/model.json",
	"reference-models/independent/weights.bin": "models/joint-composition-v1/independent/models/fp32/weights.bin",
}

type pin struct {
	Name  string `json:"name"`
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}
type manifest struct {
	Schema          string `json:"schema"`
	Status          string `json:"status"`
	Dataset         string `json:"dataset_sha256"`
	Protocol        string `json:"protocol_sha256"`
	TeacherMetadata string `json:"teacher_metadata_sha256"`
	TeacherWeights  string `json:"teacher_weights_sha256"`
	ArchiveSHA      string `json:"archive_sha256"`
	ArchiveBytes    int64  `json:"archive_bytes"`
	DecodedBytes    int64  `json:"decoded_bytes"`
	Files           []pin  `json:"files"`
	Scope           string `json:"scope"`
}

func main() {
	mode := flag.String("mode", "", "pack, verify or verify-remote")
	root := flag.String("root", "runs", "phase directory for pack")
	bundle := flag.String("bundle", "", "public ZIP")
	manifestPath := flag.String("manifest", "", "fresh pack manifest or existing verify manifest")
	dest := flag.String("destination", "", "fresh directory for verified decoded bytes")
	publicRevision := flag.String("public-revision", "", "immutable GitHub source for anonymous verify-remote")
	verificationReport := flag.String("verification-report", "", "fresh anonymous byte-verification receipt")
	flag.Parse()
	var err error
	if flag.NArg() != 0 || *manifestPath == "" {
		err = errors.New("manifest required")
	} else {
		switch *mode {
		case "pack":
			if *dest != "" || *bundle == "" || *publicRevision != "" || *verificationReport != "" {
				err = errors.New("pack cannot extract")
			} else {
				err = pack(*root, *bundle, *manifestPath)
			}
		case "verify":
			if *dest == "" || *bundle == "" || *publicRevision != "" || *verificationReport != "" {
				err = errors.New("fresh verify destination required")
			} else {
				err = verify(*bundle, *manifestPath, *dest)
			}
		case "verify-remote":
			if *dest == "" || *bundle != "" || *publicRevision == "" || *verificationReport == "" {
				err = errors.New("verify-remote requires immutable public revision, fresh destination and verification report")
			} else {
				err = verifyRemote(*publicRevision, *manifestPath, *dest, *verificationReport)
			}
		default:
			err = errors.New("mode must be pack, verify or verify-remote")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "package-three-curriculum:", err)
		os.Exit(1)
	}
}
func fileHash(path string) (pin, error) {
	s, err := os.Lstat(path)
	if err != nil || !s.Mode().IsRegular() || s.Size() > decodedCap {
		return pin{}, errors.New("bounded regular evidence required")
	}
	f, err := os.Open(path)
	if err != nil {
		return pin{}, err
	}
	defer f.Close()
	h := sha256.New()
	var buffer [32768]byte
	if _, err = io.CopyBuffer(h, f, buffer[:]); err != nil {
		return pin{}, err
	}
	return pin{SHA: hex.EncodeToString(h.Sum(nil)), Bytes: s.Size()}, nil
}
func names() []string {
	result := []string{}
	for phase, files := range phases {
		for _, file := range files {
			result = append(result, phase+"/"+file)
		}
	}
	for name := range supplements {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}
func sourcePath(root, name string) string {
	if s, ok := supplements[name]; ok {
		return s
	}
	return filepath.Join(root, filepath.FromSlash(name))
}
func stringPrivacy(v any) error {
	switch v := v.(type) {
	case string:
		if privateText.MatchString(v) {
			return errors.New("private string rejected")
		}
	case map[string]any:
		for k, item := range v {
			if privateText.MatchString(k) {
				return errors.New("private key rejected")
			}
			if err := stringPrivacy(item); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			if err := stringPrivacy(item); err != nil {
				return err
			}
		}
	}
	return nil
}
func jsonPrivacy(raw []byte) error {
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return err
	}
	if err := stringPrivacy(v); err != nil {
		return err
	}
	if object, ok := v.(map[string]any); ok {
		if _, ok = object["native_receipt"]; ok {
			var export struct {
				Raw []byte `json:"native_receipt"`
			}
			if err := json.Unmarshal(raw, &export); err != nil {
				return err
			}
			if err := jsonPrivacy(export.Raw); err != nil {
				return err
			}
		}
	}
	return nil
}
func privacy(path, name string) error {
	if strings.HasSuffix(name, "weights.bin") {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if strings.HasSuffix(name, ".jsonl") {
		r := bufio.NewScanner(f)
		r.Buffer(make([]byte, 32768), lineCap)
		for r.Scan() {
			if err := jsonPrivacy(r.Bytes()); err != nil {
				return err
			}
		}
		return r.Err()
	}
	raw, err := io.ReadAll(io.LimitReader(f, lineCap+1))
	if err != nil {
		return err
	}
	if len(raw) > lineCap {
		return errors.New("metadata/text cap exceeded")
	}
	if strings.HasSuffix(name, ".json") {
		return jsonPrivacy(raw)
	}
	if privateText.Match(raw) {
		return errors.New("private text rejected")
	}
	return nil
}
func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func pack(root, bundle, manifestPath string) error {
	if _, err := os.Lstat(manifestPath); !os.IsNotExist(err) {
		return errors.New("fresh public manifest required")
	}
	m := manifest{Schema: "gooo/three-choice-source-teacher-public-bundle/v1", Status: "SOURCE_AND_TEACHER_ONLY", Dataset: threecohort.DatasetSHA, Protocol: threefeedback.ProtocolSHA, TeacherMetadata: threefeedback.TeacherMetadata, TeacherWeights: threefeedback.TeacherWeights, Scope: "All actual native source exports, first failed prefix, 4096 independent-teacher sessions, full input reconstructions, student states, audits and frozen teacher bytes. No new three-choice student weights, training, native generative study or default promotion."}
	for phase, files := range phases {
		entries, err := os.ReadDir(filepath.Join(root, phase))
		if err != nil || len(entries) != len(files) {
			return errors.New("closed source/teacher phase inventory differs")
		}
		for _, entry := range entries {
			if !slices.Contains(files, entry.Name()) {
				return errors.New("unexpected source/teacher phase file")
			}
		}
	}
	for _, name := range names() {
		path := sourcePath(root, name)
		p, err := fileHash(path)
		if err != nil {
			return err
		}
		p.Name = name
		if err = privacy(path, name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		m.DecodedBytes += p.Bytes
		if m.DecodedBytes > decodedCap {
			return errors.New("whole-study decoded evidence cap exceeded")
		}
		m.Files = append(m.Files, p)
	}
	if err := validate(m); err != nil {
		return err
	}
	f, err := os.OpenFile(bundle, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	w := zip.NewWriter(f)
	for _, p := range m.Files {
		h := &zip.FileHeader{Name: p.Name, Method: zip.Deflate, Modified: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)}
		h.SetMode(0644)
		entry, e := w.CreateHeader(h)
		if e != nil {
			f.Close()
			return e
		}
		source, e := os.Open(sourcePath(root, p.Name))
		if e != nil {
			f.Close()
			return e
		}
		var buffer [32768]byte
		_, e = io.CopyBuffer(entry, source, buffer[:])
		closeErr := source.Close()
		if e != nil {
			f.Close()
			return e
		}
		if closeErr != nil {
			f.Close()
			return closeErr
		}
	}
	if err = w.Close(); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	p, err := fileHash(bundle)
	if err != nil {
		return err
	}
	if p.Bytes > archiveCap {
		return errors.New("public archive cap exceeded; retained source is unchanged")
	}
	m.ArchiveSHA, m.ArchiveBytes = p.SHA, p.Bytes
	if err = writeJSON(manifestPath, m); err != nil {
		return err
	}
	fmt.Printf("Packed %d files: decoded=%d archive=%d bytes\n", len(m.Files), m.DecodedBytes, m.ArchiveBytes)
	return nil
}
func validate(m manifest) error {
	if m.Schema != "gooo/three-choice-source-teacher-public-bundle/v1" || m.Status != "SOURCE_AND_TEACHER_ONLY" || m.Dataset != threecohort.DatasetSHA || m.Protocol != threefeedback.ProtocolSHA || m.TeacherMetadata != threefeedback.TeacherMetadata || m.TeacherWeights != threefeedback.TeacherWeights || len(m.Files) != len(names()) {
		return errors.New("frozen public source/teacher manifest differs")
	}
	expected := names()
	var total int64
	for i, p := range m.Files {
		decoded, e := hex.DecodeString(p.SHA)
		if e != nil || len(decoded) != 32 || p.SHA != hex.EncodeToString(decoded) || p.Name != expected[i] || p.Bytes < 0 || p.Bytes > decodedCap {
			return errors.New("closed ordered bounded archive member pins required")
		}
		total += p.Bytes
	}
	if total != m.DecodedBytes || total > decodedCap {
		return errors.New("decoded archive denominator differs")
	}
	pinned := map[string]string{"protocol.md": threefeedback.ProtocolSHA, "reference-models/independent/model.json": threefeedback.TeacherMetadata, "reference-models/independent/weights.bin": threefeedback.TeacherWeights, "own-three-composition-curriculum-fixed-20261002/dataset.jsonl": threecohort.DatasetSHA, "source-audit.json": threecohort.AuditSHA, "teacher-audit.json": "dc1bc90a5468b2f45a4e32501f0814a73ee2be7de56481c1edea3b7d414bfc6a"}
	for _, p := range m.Files {
		if sha, ok := pinned[p.Name]; ok && sha != p.SHA {
			return errors.New("immutable protocol/source/teacher/audit member changed")
		}
	}
	return nil
}
func verify(bundle, manifestPath, destination string) error {
	s, err := os.Lstat(manifestPath)
	if err != nil || !s.Mode().IsRegular() || s.Size() > lineCap {
		return errors.New("bounded regular public manifest required")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var m manifest
	if err = threecohort.Decode(raw, &m); err != nil {
		return err
	}
	if err = validate(m); err != nil {
		return err
	}
	p, err := fileHash(bundle)
	if err != nil || p.SHA != m.ArchiveSHA || p.Bytes != m.ArchiveBytes || p.Bytes > archiveCap {
		return errors.New("public ZIP byte pin differs")
	}
	if _, err = os.Lstat(destination); !os.IsNotExist(err) {
		return errors.New("fresh extraction destination required")
	}
	reader, err := zip.OpenReader(bundle)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) != len(m.Files) {
		return errors.New("ZIP closed member denominator differs")
	}
	if err = os.Mkdir(destination, 0755); err != nil {
		return err
	}
	for i, entry := range reader.File {
		p := m.Files[i]
		if entry.Name != p.Name || !entry.Mode().IsRegular() || entry.UncompressedSize64 != uint64(p.Bytes) || entry.Method != zip.Deflate {
			return errors.New("ZIP member identity/mode/size differs")
		}
		path := filepath.Join(destination, filepath.FromSlash(p.Name))
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			f.Close()
			return err
		}
		h := sha256.New()
		var buffer [32768]byte
		n, e := io.CopyBuffer(io.MultiWriter(f, h), io.LimitReader(source, p.Bytes+1), buffer[:])
		closeErr := source.Close()
		fileErr := f.Close()
		if e != nil || closeErr != nil || fileErr != nil || n != p.Bytes || hex.EncodeToString(h.Sum(nil)) != p.SHA {
			return errors.New("decoded ZIP SHA/CRC/content differs")
		}
		if err = privacy(path, p.Name); err != nil {
			return err
		}
	}
	fmt.Printf("PASS: %d decoded member SHA/CRC/privacy checks, %d bytes; no model/native/optimizer calls\n", len(m.Files), m.DecodedBytes)
	return nil
}
