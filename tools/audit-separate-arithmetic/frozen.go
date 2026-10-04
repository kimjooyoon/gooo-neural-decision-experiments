package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type frozen struct {
	Schema  string                                 `json:"schema"`
	Files   map[string]threestudent.Pin            `json:"files"`
	Members map[string]map[string]threestudent.Pin `json:"archive_members"`
	Decoded int64                                  `json:"decoded_archive_bytes"`
	Scope   string                                 `json:"scope"`
}

func pinned(name string, want threestudent.Pin) error {
	got, err := threestudent.FilePin(name)
	if err != nil || got != want || want.Bytes <= 0 || len(want.SHA) != 64 {
		return fmt.Errorf("pinned file differs: %s", filepath.Base(name))
	}
	return nil
}

func loadFrozen() (frozen, error) {
	raw, err := os.ReadFile(filepath.Join(bundle, "manifest.json"))
	var m frozen
	if err != nil || threecohort.SHA(raw) != manifestSHA {
		return m, errors.New("original publication manifest differs")
	}
	if err = threecohort.Decode(raw, &m); err != nil || len(m.Files) != 57 || len(m.Members) != 4 {
		return m, errors.New("closed original publication required")
	}
	for name, pin := range m.Files {
		if !filepath.IsLocal(name) {
			return m, errors.New("nonlocal original file")
		}
		if err = pinned(filepath.Join(bundle, name), pin); err != nil {
			return m, err
		}
	}
	return m, nil
}

func originalRows(z *zip.ReadCloser, name string, pin threestudent.Pin) ([]original, error) {
	var raw []byte
	for _, f := range z.File {
		if f.Name != name {
			continue
		}
		if raw != nil || pin.Bytes <= 0 || pin.Bytes > 1<<20 || int64(f.UncompressedSize64) != pin.Bytes {
			return nil, errors.New("duplicate or oversized journal")
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		raw, err = io.ReadAll(io.LimitReader(r, pin.Bytes+1))
		closed := r.Close()
		if err != nil || closed != nil {
			return nil, errors.Join(err, closed)
		}
	}
	if int64(len(raw)) != pin.Bytes || threecohort.SHA(raw) != pin.SHA {
		return nil, errors.New("original journal digest differs")
	}
	var rows []original
	err := scanGzip(bytes.NewReader(raw), func(line []byte) error {
		var row original
		if err := threecohort.Decode(line, &row); err != nil {
			return err
		}
		if len(row.Text) > jointdecision.ThreeInputMaxBytes || row.Input != threecohort.SHA([]byte(row.Text)) || row.Order != rank(row.Prediction) || row.Order[0] != int(row.Prediction.Mask) {
			return errors.New("original input/ranking differs")
		}
		for _, n := range row.Passed {
			if n < 0 || n > 16 {
				return errors.New("finite target extent differs")
			}
		}
		rows = append(rows, row)
		return nil
	})
	return rows, err
}

func scanGzip(r io.Reader, visit func([]byte) error) error {
	z, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer z.Close()
	bounded := &io.LimitedReader{R: z, N: (16 << 20) + 1}
	scan := bufio.NewScanner(bounded)
	scan.Buffer(make([]byte, 32768), 1<<20)
	count := 0
	for scan.Scan() {
		if count >= 512 {
			return errors.New("extra journal row")
		}
		if err := visit(scan.Bytes()); err != nil {
			return err
		}
		count++
	}
	if scan.Err() != nil || count != 512 || bounded.N == 0 {
		return errors.New("complete bounded 512-row journal required")
	}
	return nil
}

type budgetWriter struct {
	w     io.Writer
	bytes *int64
}

func (b budgetWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > outputCap-*b.bytes {
		return 0, errors.New("collection output cap exceeded")
	}
	n, err := b.w.Write(p)
	*b.bytes += int64(n)
	return n, err
}

func writeFresh(path string, raw []byte, used *int64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = (budgetWriter{f, used}).Write(raw)
	return errors.Join(err, f.Close())
}

func sourcePins(revision string) (map[string]threestudent.Pin, error) {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return nil, errors.New("exact source revision required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return nil, errors.New("clean collection source required")
	}
	files, err := exec.Command("git", "ls-files", "internal/decision", "internal/jointdecision", "internal/modelfile", "internal/pathplan", "internal/threecohort", "internal/threestudent", "tools/audit-separate-arithmetic", "go.mod", "go.sum").Output()
	if err != nil {
		return nil, err
	}
	pins := map[string]threestudent.Pin{}
	for name := range strings.FieldsSeq(string(files)) {
		pin, err := threestudent.FilePin(name)
		if err != nil {
			return nil, err
		}
		pins[name] = pin
	}
	if len(pins) < 20 {
		return nil, errors.New("computational source inventory incomplete")
	}
	return pins, nil
}

func loadFour(m frozen, output, arm, variant string, used *int64, pins map[string]threestudent.Pin) ([4]*jointdecision.ThreeModel, error) {
	var models [4]*jointdecision.ThreeModel
	for repr, layout := range []string{"expanded", "compact"} {
		rel := filepath.Join("models", layout, arm, variant)
		name := filepath.Join(bundle, rel, "model.json")
		raw, err := os.ReadFile(name)
		if err != nil {
			return models, err
		}
		var meta jointdecision.Metadata
		if err = threecohort.Decode(raw, &meta); err != nil || meta.Arithmetic != "" {
			return models, errors.New("legacy metadata required")
		}
		loader := jointdecision.LoadThree
		if strings.HasPrefix(arm, "bag-") {
			loader = jointdecision.LoadThreeBag
		}
		if repr == 1 {
			loader = jointdecision.LoadSharedThree
		}
		if models[repr], err = loader(name); err != nil {
			return models, err
		}
		meta.Arithmetic = jointdecision.SeparateArithmeticVersion
		encoded, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return models, err
		}
		if err = writeFresh(filepath.Join(output, rel, "model.json"), append(encoded, '\n'), used); err != nil {
			return models, err
		}
		weights, err := os.ReadFile(filepath.Join(bundle, rel, "weights.bin"))
		if err != nil {
			return models, err
		}
		if err = writeFresh(filepath.Join(output, rel, "weights.bin"), weights, used); err != nil {
			return models, err
		}
		if models[repr+2], err = loader(filepath.Join(output, rel, "model.json")); err != nil {
			return models, err
		}
		if models[repr].WeightsSHA256() != models[repr+2].WeightsSHA256() || models[repr+2].ArithmeticVersion() != meta.Arithmetic {
			return models, errors.New("weight or arithmetic identity changed")
		}
		for _, file := range []string{"model.json", "weights.bin"} {
			old := m.Files[filepath.ToSlash(filepath.Join(rel, file))]
			pins[filepath.ToSlash(filepath.Join("legacy", layout, arm, variant, file))] = old
			pin, err := threestudent.FilePin(filepath.Join(output, rel, file))
			if err != nil {
				return models, err
			}
			pins[filepath.ToSlash(filepath.Join("separate", layout, arm, variant, file))] = pin
		}
	}
	meta, weights, err := jointdecision.CompactThree(models[2])
	if err != nil {
		return models, err
	}
	expected, err := os.ReadFile(filepath.Join(output, "models", "compact", arm, variant, "model.json"))
	if err != nil {
		return models, err
	}
	encoded, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return models, err
	}
	if !bytes.Equal(expected, append(encoded, '\n')) || threecohort.SHA(weights) != models[3].WeightsSHA256() {
		return models, errors.New("compaction does not preserve explicit arithmetic artifact")
	}
	return models, nil
}
