// package-three-training retains every prepared tensor and completed optimizer
// receipt in a compressed public bundle. It never trims negative variants.
package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const prepared = "own-three-training-inputs-20261002"
const trained = "own-three-training-mps-20261002"

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

type member struct {
	Name  string `json:"name"`
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}
type manifest struct {
	Schema       string           `json:"schema"`
	Status       string           `json:"status"`
	Files        []member         `json:"files"`
	Bytes        int64            `json:"decoded_bytes"`
	Archive      threestudent.Pin `json:"archive"`
	Protocol     string           `json:"protocol_sha256"`
	TeacherAudit string           `json:"teacher_audit_sha256"`
	States       string           `json:"student_states_sha256"`
	Initial      string           `json:"initial_state_sha256"`
	Updates      int              `json:"recorded_optimizer_updates"`
	Models       int              `json:"model_exports"`
	Scope        string           `json:"scope"`
}

func main() {
	mode := flag.String("mode", "verify", "pack or verify")
	bundle := flag.String("bundle", "", "public ZIP")
	index := flag.String("manifest", "", "public byte manifest")
	prepAudit := flag.String("preparation-audit", "", "complete independent input replay")
	modelAudit := flag.String("model-audit", "", "actual Go model audit")
	destination := flag.String("destination", "", "fresh expanded evidence directory")
	flag.Parse()
	var err error
	if *mode == "pack" {
		err = pack(*bundle, *index, *prepAudit, *modelAudit)
	} else if *mode == "verify" {
		err = verify(*bundle, *index, *destination)
	} else {
		err = errors.New("unknown bundle mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func expectedNames() []string {
	names := []string{"protocol.md", "preparation-audit.json", "go-model-audit.json"}
	for _, n := range []string{"preexecution.json", "initial-fp32.bin", "features-f32le.bin", "rows.jsonl", "parity-inputs.json", "manifest.json", "preparation-attempt.json"} {
		names = append(names, prepared+"/"+n)
	}
	for _, n := range []string{"preexecution.json", "report.json"} {
		names = append(names, trained+"/"+n)
	}
	for _, arm := range []string{"uniform-initial", "set-initial", "set-feedback"} {
		for _, n := range []string{"training-report.json", "go-parity.jsonl"} {
			names = append(names, trained+"/"+arm+"/"+n)
		}
		for _, stage := range []string{"fp32", "qat"} {
			names = append(names, trained+"/"+arm+"/"+stage+"/optimizer-updates.jsonl")
			for epoch := 1; epoch <= 100; epoch++ {
				names = append(names, fmt.Sprintf("%s/%s/%s/epoch-%03d.json", trained, arm, stage, epoch))
			}
		}
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, n := range []string{"model.json", "weights.bin"} {
				names = append(names, trained+"/"+arm+"/models/"+variant+"/"+n)
			}
		}
	}
	sort.Strings(names)
	return names
}

func publicJSON(raw []byte) error {
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	var walk func(any) error
	walk = func(v any) error {
		switch x := v.(type) {
		case string:
			if privateText.MatchString(x) {
				return errors.New("decoded private text rejected")
			}
		case []any:
			for _, y := range x {
				if err := walk(y); err != nil {
					return err
				}
			}
		case map[string]any:
			for key, y := range x {
				if privateText.MatchString(key) {
					return errors.New("private JSON key")
				}
				if err := walk(y); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(v)
}

func privacy(path, name string) error {
	if strings.HasSuffix(name, ".bin") {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if strings.HasSuffix(name, ".jsonl") {
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 32768), 1<<20)
		for s.Scan() {
			if err = publicJSON(s.Bytes()); err != nil {
				return err
			}
		}
		return s.Err()
	}
	raw, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("bounded metadata required")
	}
	if strings.HasSuffix(name, ".json") {
		return publicJSON(raw)
	}
	if privateText.Match(raw) {
		return errors.New("private text rejected")
	}
	return nil
}

func save(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 || publicJSON(raw) != nil {
		return errors.New("bounded public manifest required")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func pack(bundle, index, prepAudit, modelAudit string) error {
	for _, name := range []string{bundle, index} {
		if _, err := os.Lstat(name); !os.IsNotExist(err) {
			return errors.New("fresh public bundle and manifest required")
		}
	}
	sources := map[string]string{"protocol.md": threefeedback.Protocol, "preparation-audit.json": prepAudit, "go-model-audit.json": modelAudit}
	expected := expectedNames()
	allowed := map[string]bool{}
	for _, n := range expected {
		allowed[n] = true
	}
	for _, phase := range []string{prepared, trained} {
		actual := 0
		err := filepath.WalkDir(filepath.Join("runs", phase), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, e := filepath.Rel("runs", path)
			if e != nil {
				return e
			}
			rel = filepath.ToSlash(rel)
			if !allowed[rel] {
				return errors.New("unexpected retained phase file")
			}
			sources[rel] = path
			actual++
			return nil
		})
		if err != nil {
			return err
		}
		count := 0
		for _, n := range expected {
			if strings.HasPrefix(n, phase+"/") {
				count++
			}
		}
		if count != actual {
			return errors.New("missing original raw phase file")
		}
	}
	var actualAudit struct {
		Status  string `json:"status"`
		Calls   int    `json:"actual_go_model_predictions"`
		Models  int    `json:"model_exports"`
		Updates int    `json:"recorded_optimizer_updates_reconciled"`
	}
	var preparedAudit struct {
		Schema   string `json:"schema"`
		Status   string `json:"status"`
		States   string `json:"student_states_sha256"`
		Initial  string `json:"initial_state_sha256"`
		Manifest string `json:"manifest_sha256"`
		Values   int    `json:"feature_float32_values_recomputed"`
		Calls    int    `json:"model_predictions"`
		Updates  int    `json:"optimizer_updates"`
		Native   int    `json:"native_calls"`
	}
	rawPrep, err := os.ReadFile(prepAudit)
	if err != nil {
		return err
	}
	if err = publicJSON(rawPrep); err != nil {
		return err
	}
	pinPrep, err := threestudent.FilePin(filepath.Join("runs", prepared, "manifest.json"))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(rawPrep, &preparedAudit); err != nil || preparedAudit.Schema != "gooo/own-three-training-input-independent-replay/v1" || preparedAudit.Status != "PASS" || preparedAudit.States != threestudent.StatesSHA || preparedAudit.Initial != threestudent.InitialSHA || preparedAudit.Manifest != pinPrep.SHA || preparedAudit.Values != 8247552 || preparedAudit.Calls != 0 || preparedAudit.Updates != 0 || preparedAudit.Native != 0 {
		return errors.New("complete independently replayed Go optimizer inputs required")
	}
	raw, err := os.ReadFile(modelAudit)
	if err != nil {
		return err
	}
	if err = publicJSON(raw); err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &actualAudit); err != nil || actualAudit.Status != "PASS" || actualAudit.Calls != 18450 || actualAudit.Models != 9 || actualAudit.Updates != 4800 {
		return errors.New("actual full Go model audit required")
	}
	m := manifest{Schema: "gooo/three-choice-training-public-bundle/v1", Status: "TRAINED_GO_KERNEL_AUDITED", Protocol: threefeedback.ProtocolSHA, TeacherAudit: threestudent.TeacherAuditSHA, States: threestudent.StatesSHA, Initial: threestudent.InitialSHA, Updates: 4800, Models: 9, Scope: "Every original Go feature/initializer file, all 4800 update receipts, all 600 epoch receipts, all nine models including negative PTQ variants, complete parity rows and independent input/kernel audits. Full SDK/native completeness study and default promotion remain pending. Source/teacher raw evidence is retained in its separate immutable public bundle."}
	for _, name := range expected {
		p, err := threestudent.FilePin(sources[name])
		if err != nil {
			return err
		}
		if err = privacy(sources[name], name); err != nil {
			return err
		}
		m.Files = append(m.Files, member{name, p.SHA, p.Bytes})
		m.Bytes += p.Bytes
	}
	if err = validate(m); err != nil {
		return err
	}
	f, err := os.OpenFile(bundle, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w := zip.NewWriter(f)
	var buffer [32768]byte
	for _, p := range m.Files {
		header := zip.FileHeader{Name: p.Name, Method: zip.Deflate}
		header.SetMode(0644)
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		entry, e := w.CreateHeader(&header)
		if e != nil {
			w.Close()
			f.Close()
			return e
		}
		source, e := os.Open(sources[p.Name])
		if e != nil {
			w.Close()
			f.Close()
			return e
		}
		n, e := io.CopyBuffer(entry, source, buffer[:])
		closeErr := source.Close()
		if e != nil || closeErr != nil || n != p.Bytes {
			w.Close()
			f.Close()
			return errors.New("source changed during ZIP creation")
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
	m.Archive, err = threestudent.FilePin(bundle)
	if err != nil {
		return err
	}
	if m.Archive.Bytes > 64<<20 {
		return errors.New("compressed publication cap exceeded; originals retained")
	}
	if err = save(index, m); err != nil {
		return err
	}
	fmt.Printf("PACKED: %d original/supplement members, %d decoded bytes, %d compressed bytes\n", len(m.Files), m.Bytes, m.Archive.Bytes)
	return nil
}

func validate(m manifest) error {
	expected := expectedNames()
	if m.Schema != "gooo/three-choice-training-public-bundle/v1" || m.Status != "TRAINED_GO_KERNEL_AUDITED" || m.Protocol != threefeedback.ProtocolSHA || m.TeacherAudit != threestudent.TeacherAuditSHA || m.States != threestudent.StatesSHA || m.Initial != threestudent.InitialSHA || m.Updates != 4800 || m.Models != 9 || len(m.Files) != len(expected) || len(expected) != 642 {
		return errors.New("frozen student public inventory differs")
	}
	var total int64
	for i, p := range m.Files {
		if p.Name != expected[i] || p.Bytes <= 0 || p.Bytes > threestudent.RawCap || len(p.SHA) != 64 {
			return errors.New("closed file name/size/pin differs")
		}
		total += p.Bytes
		if p.Name == "protocol.md" && p.SHA != threefeedback.ProtocolSHA || p.Name == prepared+"/initial-fp32.bin" && p.SHA != threestudent.InitialSHA {
			return errors.New("immutable protocol/initializer differs")
		}
		if p.Name == prepared+"/features-f32le.bin" && p.SHA != "4143e4f57936427faeee5b46b91374511efdf20dec5c4c15f70fd9af6d893167" || p.Name == prepared+"/rows.jsonl" && p.SHA != "2b23a6204c5631074fd763c45d1c5cee79f504946d8591ddd4f3e4e6d2ab5d3b" {
			return errors.New("frozen source features or balanced row metadata differs")
		}
	}
	if m.Bytes != total || total > threestudent.RawCap {
		return errors.New("decoded public evidence cap exceeded")
	}
	return nil
}

func verify(bundle, index, destination string) error {
	raw, err := os.ReadFile(index)
	if err != nil || len(raw) > 1<<20 {
		return errors.New("bounded manifest required")
	}
	var m manifest
	if err = threecohort.Decode(raw, &m); err != nil {
		return err
	}
	if err = validate(m); err != nil {
		return err
	}
	p, err := threestudent.FilePin(bundle)
	if err != nil || p != m.Archive || p.Bytes > 64<<20 {
		return errors.New("exact public ZIP bytes required")
	}
	if _, err = os.Lstat(destination); !os.IsNotExist(err) {
		return errors.New("fresh expanded destination required")
	}
	if err = os.Mkdir(destination, 0755); err != nil {
		return err
	}
	r, err := zip.OpenReader(bundle)
	if err != nil {
		return err
	}
	defer r.Close()
	if len(r.File) != len(m.Files) {
		return errors.New("ZIP closed inventory differs")
	}
	var buffer [32768]byte
	for i, z := range r.File {
		expected := m.Files[i]
		if z.Name != expected.Name || z.Mode() != 0644 || z.UncompressedSize64 != uint64(expected.Bytes) {
			return errors.New("ZIP path/mode/size differs")
		}
		path := filepath.Join(destination, filepath.FromSlash(z.Name))
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		source, err := z.Open()
		if err != nil {
			f.Close()
			return err
		}
		n, e := io.CopyBuffer(f, io.LimitReader(source, expected.Bytes+1), buffer[:])
		sourceErr := source.Close()
		fileErr := f.Close()
		if e != nil || sourceErr != nil || fileErr != nil || n != expected.Bytes {
			return errors.New("ZIP decoded bytes/CRC differ")
		}
		p, err := threestudent.FilePin(path)
		if err != nil || p.SHA != expected.SHA || p.Bytes != expected.Bytes {
			return errors.New("decoded member SHA differs")
		}
		if err = privacy(path, z.Name); err != nil {
			return err
		}
	}
	fmt.Printf("PASS: all %d decoded SHA/CRC/mode/privacy checks, %d bytes\n", len(m.Files), m.Bytes)
	return nil
}
