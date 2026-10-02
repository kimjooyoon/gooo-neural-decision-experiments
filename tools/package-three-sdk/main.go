// package-three-sdk preserves every original byte of the cap-stopped study.
// Packaging, byte verification and transport never call a model or optimizer.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const phase = "own-three-sdk-study-20261002"
const producer = "b8c515fcd5a5e623aae5fd3b93c3b8f69f9f4608"
const baseHF = "a7a9170370da3e183c1831ae4442e88870ea3a0d"
const repository = "asketeddy/gooo-three-choice-feedback-tiny-v1"
const remotePrefix = "research/sdk-prefix-20261002"

type member struct {
	Name string           `json:"name"`
	Pin  threestudent.Pin `json:"pin"`
}
type manifest struct {
	Schema         string           `json:"schema"`
	Status         string           `json:"status"`
	Producer       string           `json:"producer_source_revision"`
	Files          []member         `json:"files"`
	Bytes          int64            `json:"decoded_bytes"`
	Archive        threestudent.Pin `json:"archive"`
	Sessions       int              `json:"actual_sdk_sessions"`
	Predictions    int              `json:"actual_model_predictions"`
	Values         int              `json:"independently_verified_ordered_values"`
	ModelsRevision string           `json:"existing_hf_models_revision"`
	Protocol       string           `json:"protocol_sha256"`
	Scope          string           `json:"scope"`
}

func main() {
	mode := flag.String("mode", "verify", "pack, verify, stage-hf or verify-hf")
	bundle := flag.String("bundle", "publication/own-three-choice-sdk-prefix-20261002.zip", "complete public prefix ZIP")
	index := flag.String("manifest", "publication/own-three-choice-sdk-prefix-bundle-20261002.json", "closed ZIP byte manifest")
	audit := flag.String("audit", "publication/own-three-choice-sdk-prefix-audit-20261002.json", "actual independent prefix audit")
	stage := flag.String("stage", "publication/hf-own-three-sdk-prefix-20261002", "fresh seven-file HF appendix")
	revision := flag.String("revision", "", "immutable public HF appendix commit")
	receipt := flag.String("receipt", "", "fresh anonymous byte receipt")
	flag.Parse()
	var err error
	switch *mode {
	case "pack":
		err = pack(*bundle, *index, *audit)
	case "verify":
		err = verify(*bundle, *index)
	case "stage-hf":
		err = stageHF(*bundle, *index, *audit, *stage)
	case "verify-hf":
		err = verifyHF(*stage, *revision, *receipt)
	default:
		err = errors.New("unknown prefix publication mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func read(name string, v any) error {
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}
func save(name string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(append(raw, '\n')); err != nil {
		return err
	}
	return f.Sync()
}
func sources(audit string) (map[string]string, error) {
	root := filepath.Join("runs", phase)
	var attempt struct {
		Status      string                      `json:"status"`
		Sessions    int                         `json:"actual_sdk_sessions"`
		Predictions int                         `json:"actual_model_predictions"`
		Files       map[string]threestudent.Pin `json:"retained_files_before_attempt"`
	}
	if err := read(filepath.Join(root, "collection-attempt.json"), &attempt); err != nil {
		return nil, err
	}
	if attempt.Status != "FAILED_PREFIX_RETAINED" || attempt.Sessions != 8779 || attempt.Predictions != 33388 || len(attempt.Files) != 20 {
		return nil, errors.New("exact original cap-stopped prefix required")
	}
	all := map[string]string{"protocol.md": threefeedback.Protocol, "design.md": "docs/own-three-choice-sdk-design-20261002.md", "audit.json": audit, "dataset.jsonl": "runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl"}
	for name, want := range attempt.Files {
		if filepath.Base(name) != name || !(strings.HasPrefix(name, "calibration-") || strings.HasPrefix(name, "development-") || name == "preexecution.json" || name == "selection.json") {
			return nil, errors.New("closed original phase names required")
		}
		p, err := threestudent.FilePin(filepath.Join(root, name))
		if err != nil || p != want {
			return nil, errors.New("original retained member changed")
		}
		all[phase+"/"+name] = filepath.Join(root, name)
	}
	all[phase+"/collection-attempt.json"] = filepath.Join(root, "collection-attempt.json")
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 21 {
		return nil, errors.New("exact original phase inventory required")
	}
	return all, nil
}
func pack(bundle, index, audit string) error {
	var proof struct {
		Schema      string `json:"schema"`
		Status      string `json:"status"`
		Producer    string `json:"producer_source_revision"`
		Sessions    int    `json:"actual_captured_sdk_sessions"`
		Predictions int    `json:"actual_captured_model_predictions"`
		Values      int    `json:"actual_independently_verified_ordered_values"`
	}
	if err := read(audit, &proof); err != nil {
		return err
	}
	if proof.Schema != "gooo/own-three-sdk-independent-audit/v1" || proof.Status != "PREFIX_VERIFIED_INCOMPLETE" || proof.Producer != producer || proof.Sessions != 8779 || proof.Predictions != 33388 || proof.Values != 510256 {
		return errors.New("actual original independent prefix audit required")
	}
	all, err := sources(audit)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(all))
	for n := range all {
		names = append(names, n)
	}
	slices.Sort(names)
	m := manifest{Schema: "gooo/own-three-sdk-prefix-public-bundle/v1", Status: "PREFIX_VERIFIED_INCOMPLETE", Producer: producer, Sessions: 8779, Predictions: 33388, Values: 510256, ModelsRevision: baseHF, Protocol: threefeedback.ProtocolSHA, Scope: "Complete original bytes of the 768-MiB-cap stopped study, all completed/negative policies and partial final cell. All eleven calibration cells and six development cells complete; the seventh development cell is an incomplete retained prefix. No full 11264-session claim, native calls or default promotion. Frozen nine own models and independent reference remain in the existing immutable model/source/training publication."}
	f, err := os.OpenFile(bundle, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	z := zip.NewWriter(f)
	for _, name := range names {
		p, err := threestudent.FilePin(all[name])
		if err != nil {
			z.Close()
			f.Close()
			return err
		}
		in, err := os.Open(all[name])
		if err != nil {
			z.Close()
			f.Close()
			return err
		}
		err = privacy(in, name)
		in.Close()
		if err != nil {
			z.Close()
			f.Close()
			return err
		}
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0644)
		out, err := z.CreateHeader(h)
		if err != nil {
			z.Close()
			f.Close()
			return err
		}
		in, err = os.Open(all[name])
		if err != nil {
			z.Close()
			f.Close()
			return err
		}
		hash := sha256.New()
		n, err := io.Copy(io.MultiWriter(out, hash), in)
		in.Close()
		if err != nil || n != p.Bytes || hex.EncodeToString(hash.Sum(nil)) != p.SHA {
			z.Close()
			f.Close()
			return errors.New("source changed during complete byte packaging")
		}
		m.Files = append(m.Files, member{name, p})
		m.Bytes += p.Bytes
	}
	if err = z.Close(); err != nil {
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
	m.Archive, err = threestudent.FilePin(bundle)
	if err != nil {
		return err
	}
	if m.Archive.Bytes > 64<<20 {
		return errors.New("compressed prefix exceeds public bundle bound; prefix retained")
	}
	if err = save(index, m); err != nil {
		return err
	}
	return verify(bundle, index)
}
func validate(m manifest) error {
	if m.Schema != "gooo/own-three-sdk-prefix-public-bundle/v1" || m.Status != "PREFIX_VERIFIED_INCOMPLETE" || m.Producer != producer || m.Sessions != 8779 || m.Predictions != 33388 || m.Values != 510256 || m.ModelsRevision != baseHF || m.Protocol != threefeedback.ProtocolSHA || len(m.Files) != 25 || m.Bytes > 768<<20 || m.Archive.Bytes > 64<<20 {
		return errors.New("closed original prefix manifest differs")
	}
	seen := map[string]bool{}
	var n int64
	for _, f := range m.Files {
		if seen[f.Name] || filepath.Clean(f.Name) != f.Name || strings.HasPrefix(f.Name, "/") || strings.Contains(f.Name, "..") || !(strings.HasPrefix(f.Name, phase+"/") || f.Name == "protocol.md" || f.Name == "design.md" || f.Name == "audit.json" || f.Name == "dataset.jsonl") || len(f.Pin.SHA) != 64 || f.Pin.Bytes < 0 {
			return errors.New("closed unique public member required")
		}
		seen[f.Name] = true
		n += f.Pin.Bytes
	}
	if n != m.Bytes {
		return errors.New("public decoded byte sum differs")
	}
	return nil
}

type counted struct {
	Reader io.Reader
	Bytes  int64
}

func (c *counted) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	c.Bytes += int64(n)
	return n, err
}
func verify(bundle, index string) error {
	var m manifest
	if err := read(index, &m); err != nil {
		return err
	}
	if err := validate(m); err != nil {
		return err
	}
	p, err := threestudent.FilePin(bundle)
	if err != nil || p != m.Archive {
		return errors.New("actual complete ZIP bytes differ")
	}
	z, err := zip.OpenReader(bundle)
	if err != nil {
		return err
	}
	defer z.Close()
	if len(z.File) != len(m.Files) {
		return errors.New("closed ZIP inventory differs")
	}
	for i, entry := range z.File {
		want := m.Files[i]
		if entry.Name != want.Name || entry.Mode() != 0644 || entry.UncompressedSize64 != uint64(want.Pin.Bytes) {
			return errors.New("ZIP member name/mode/size differs")
		}
		r, err := entry.Open()
		if err != nil {
			return err
		}
		hash := sha256.New()
		in := &counted{Reader: io.TeeReader(io.LimitReader(r, want.Pin.Bytes+1), hash)}
		err = privacy(in, entry.Name)
		ce := r.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
		if in.Bytes != want.Pin.Bytes || hex.EncodeToString(hash.Sum(nil)) != want.Pin.SHA {
			return errors.New("decoded original SHA/CRC bytes differ")
		}
		if entry.Name == "protocol.md" && want.Pin.SHA != threefeedback.ProtocolSHA {
			return errors.New("immutable protocol changed")
		}
		if entry.Name == "dataset.jsonl" && (want.Pin.SHA != threecohort.DatasetSHA || want.Pin.Bytes != threecohort.DatasetBytes) {
			return errors.New("frozen corpus bytes changed")
		}
	}
	fmt.Printf("PASS: all %d complete original decoded files, %d bytes; ZIP %d bytes; private-text and SHA/CRC checks; zero new model calls\n", len(m.Files), m.Bytes, m.Archive.Bytes)
	return nil
}
