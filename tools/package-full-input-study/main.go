// Package the complete first full-input study as bounded, independently hashed
// archives. Preparation, training and initial Go observations keep separate roots.
package main

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const bank = "runs/own-three-full-input-bank-20261003"
const training = "runs/own-three-full-input-training-20261003"
const audit = "runs/own-three-full-input-initial-audit-20261003"

type manifest struct {
	Schema   string                                 `json:"schema"`
	Files    map[string]threestudent.Pin            `json:"files"`
	Archives map[string]map[string]threestudent.Pin `json:"archive_members"`
	Decoded  int64                                  `json:"decoded_archive_bytes"`
	Scope    string                                 `json:"scope"`
}

func main() {
	mode := flag.String("mode", "verify", "package or verify")
	output := flag.String("output", "publication/full-input-initial-study-20261003", "fresh public bundle")
	replay := flag.String("compare-audit", "", "compare a fresh Go audit to the published observations")
	flag.Parse()
	var err error
	switch *mode {
	case "package":
		err = pack(*output)
	case "verify":
		err = verify(*output)
	default:
		err = errors.New("unknown mode")
	}
	if err == nil && *replay != "" {
		err = compareAudit(filepath.Join(*output, "report.json"), *replay)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func compareAudit(expected, actual string) error {
	read := func(path string) (map[string]any, error) {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
			return nil, errors.New("bounded audit report required")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var result map[string]any
		err = threecohort.Decode(raw, &result)
		return result, err
	}
	a, err := read(expected)
	if err != nil {
		return err
	}
	b, err := read(actual)
	if err != nil {
		return err
	}
	if err = compareValue("", a, b); err != nil {
		return err
	}
	fmt.Println(`{"status":"PASS","comparison":"all model identities, counts, families, languages and bilingual outcomes exact; accumulated float64 mass/NLL tolerance 0.001; each export parity maximum <= 0.00001"}`)
	return nil
}

func compareValue(path string, a, b any) error {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return errors.New("audit object inventory differs: " + path)
		}
		for key, value := range x {
			other, ok := y[key]
			if !ok {
				return errors.New("audit key missing: " + path)
			}
			if err := compareValue(path+"/"+key, value, other); err != nil {
				return err
			}
		}
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return errors.New("audit array differs: " + path)
		}
		for i := range x {
			if err := compareValue(path, x[i], y[i]); err != nil {
				return err
			}
		}
	case float64:
		y, ok := b.(float64)
		if !ok || math.IsNaN(y) || math.IsInf(y, 0) {
			return errors.New("audit number differs: " + path)
		}
		if strings.HasSuffix(path, "/maximum_absolute_export_parity_error") {
			if x < 0 || y < 0 || x > 1e-5 || y > 1e-5 {
				return errors.New("export parity bound differs")
			}
			return nil
		}
		tolerance := 0.
		if strings.HasSuffix(path, "/summed_passing_set_mass") || strings.HasSuffix(path, "/summed_passing_set_nll") || strings.Contains(path, "/stable_float64_passing_set_nll_sum_by_condition/") {
			tolerance = .001
		}
		if math.Abs(x-y) > tolerance {
			return errors.New("audit number differs: " + path)
		}
	case string:
		y, ok := b.(string)
		if !ok {
			return errors.New("audit string differs: " + path)
		}
		if path == "/auditor_source_revision" && len(x) == 40 && len(y) == 40 {
			return nil
		}
		if x != y {
			return errors.New("audit identity differs: " + path)
		}
	default:
		if a != b {
			return errors.New("audit value differs: " + path)
		}
	}
	return nil
}

func tree(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("regular evidence files required")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = path
		return nil
	})
	return out, err
}

var sensitive = regexp.MustCompile(`/Users/|/home/|BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}`)

func privacy(path string) error {
	if strings.HasSuffix(path, ".bin") {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var input io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		z, e := gzip.NewReader(f)
		if e != nil {
			return e
		}
		defer z.Close()
		input = z
	}
	scan := bufio.NewScanner(input)
	scan.Buffer(make([]byte, 32768), 1<<20)
	for scan.Scan() {
		if sensitive.Match(scan.Bytes()) {
			return errors.New("public privacy scan rejected a record")
		}
	}
	return scan.Err()
}

type bounded struct {
	writer io.Writer
	left   int64
}

func (b *bounded) Write(raw []byte) (int, error) {
	if int64(len(raw)) > b.left {
		return 0, errors.New("48-MiB archive cap; preserve prefix")
	}
	n, err := b.writer.Write(raw)
	b.left -= int64(n)
	return n, err
}

func pack(output string) error {
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh publication output required")
	}
	m := manifest{Schema: "gooo/full-input-initial-publication/v1", Files: map[string]threestudent.Pin{}, Archives: map[string]map[string]threestudent.Pin{}, Scope: "Complete Go preparation, offline optimization and initial Go audit. Original + two new full-input development conditions; further protocol conditions and native adoption remain pending."}
	groups := map[string]map[string]string{}
	for name, path := range map[string]string{"bank.zip": bank, "training.zip": training, "initial-audit.zip": audit} {
		x, err := tree(path)
		if err != nil {
			return err
		}
		groups[name] = x
	}
	groups["sources.zip"] = map[string]string{
		"dataset.jsonl":        "runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl",
		"teacher/states.jsonl": "runs/own-three-teacher-curriculum-20261002/states.jsonl",
		"teacher/report.json":  "runs/own-three-teacher-curriculum-20261002/report.json",
		"teacher/audit.json":   "publication/own-three-choice-teacher-audit-20261002.json",
		"protocol.md":          "docs/full-input-judgment-preregistration-20261003.md",
	}
	for _, name := range []string{"train_full_input_judgment_v1.py", "full_input_evidence_v1.py", "train_shared_three_judgment_v1.py", "shared_evidence_v1.py", "train_own_three_feedback_v1.py", "train_pilot_v2.py"} {
		groups["sources.zip"]["training/"+name] = "training/" + name
	}
	count := 0
	for archive, files := range groups {
		m.Archives[archive] = map[string]threestudent.Pin{}
		for name, path := range files {
			if err := privacy(path); err != nil {
				return fmt.Errorf("%s/%s: %w", archive, name, err)
			}
			pin, err := threestudent.FilePin(path)
			if err != nil {
				return err
			}
			m.Archives[archive][name] = pin
			m.Decoded += pin.Bytes
			count++
		}
	}
	if count > 1200 || m.Decoded+(128<<20) > 768<<20 {
		return errors.New("bounded complete archive and publication reserve required")
	}
	if err := os.Mkdir(output, 0755); err != nil {
		return err
	}
	for _, name := range []string{"bank.zip", "training.zip", "initial-audit.zip", "sources.zip"} {
		if err := archive(filepath.Join(output, name), groups[name], m.Archives[name]); err != nil {
			return err
		}
	}
	copyFiles := map[string]string{
		"README.md":               "docs/full-input-judgment-initial-results-20261003.md",
		"report.json":             audit + "/report.json",
		"optimizer-process.json":  training + "/optimizer-process.json",
		"preparation-replay.json": bank + "/replay.json",
		"order-control.json":      bank + "/order-control.json",
	}
	for prefix, root := range map[string]string{"compact": audit + "/compact", "expanded": training} {
		for _, arm := range []string{"positioned-original", "positioned-varied", "bag-original", "bag-varied"} {
			for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
				for _, file := range []string{"model.json", "weights.bin"} {
					from := filepath.Join(root, arm, variant, file)
					if prefix == "expanded" {
						from = filepath.Join(root, arm, "models", variant, file)
					}
					copyFiles[filepath.ToSlash(filepath.Join("models", prefix, arm, variant, file))] = from
				}
			}
		}
	}
	for name, from := range copyFiles {
		if err := privacy(from); err != nil {
			return err
		}
		raw, err := os.ReadFile(from)
		if err != nil {
			return err
		}
		if len(raw) > 4<<20 {
			return errors.New("bounded direct public file required")
		}
		path := filepath.Join(output, name)
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(path, raw, 0644); err != nil {
			return err
		}
	}
	files, err := tree(output)
	if err != nil {
		return err
	}
	var packed int64
	for name, path := range files {
		pin, e := threestudent.FilePin(path)
		if e != nil {
			return e
		}
		m.Files[name] = pin
		packed += pin.Bytes
	}
	if packed > 128<<20 {
		return errors.New("128-MiB complete publication cap")
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(output, "manifest.json"), append(raw, '\n'), 0644); err != nil {
		return err
	}
	return verify(output)
}

func archive(output string, files map[string]string, pins map[string]threestudent.Pin) error {
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	z := zip.NewWriter(&bounded{f, 48 << 20})
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		out, e := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if e != nil {
			err = e
			break
		}
		in, e := os.Open(files[name])
		if e != nil {
			err = e
			break
		}
		h := sha256.New()
		n, e := io.Copy(io.MultiWriter(out, h), in)
		closed := in.Close()
		if e != nil || closed != nil {
			err = errors.Join(e, closed)
			break
		}
		if n != pins[name].Bytes || hex.EncodeToString(h.Sum(nil)) != pins[name].SHA {
			err = errors.New("source changed during packing")
			break
		}
	}
	return errors.Join(err, z.Close(), f.Close())
}

func verify(output string) error {
	raw, err := os.ReadFile(filepath.Join(output, "manifest.json"))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("bounded public manifest required")
	}
	var m manifest
	if err = threecohort.Decode(raw, &m); err != nil {
		return err
	}
	if m.Schema != "gooo/full-input-initial-publication/v1" || len(m.Archives) != 4 || m.Decoded <= 0 || m.Decoded > 768<<20 {
		return errors.New("closed archive contract required")
	}
	var expectedBytes int64
	expectedCount := 0
	for _, members := range m.Archives {
		for _, pin := range members {
			if pin.Bytes < 0 || pin.Bytes > 768<<20 {
				return errors.New("bounded decoded member required")
			}
			expectedBytes += pin.Bytes
			expectedCount++
			if expectedBytes > 768<<20 || expectedCount > 1200 {
				return errors.New("bounded decoded inventory required")
			}
		}
	}
	if expectedBytes != m.Decoded {
		return errors.New("declared decoded total differs")
	}
	files, err := tree(output)
	if err != nil {
		return err
	}
	if len(files) != len(m.Files)+1 {
		return errors.New("unexpected public file inventory")
	}
	var total int64
	for name, pin := range m.Files {
		path, ok := files[name]
		if !ok {
			return errors.New("public file missing")
		}
		p, e := threestudent.FilePin(path)
		if e != nil || p != pin {
			return errors.New("public file bytes differ")
		}
		total += p.Bytes
	}
	if total > 128<<20 {
		return errors.New("public bundle cap")
	}
	var decoded int64
	count := 0
	for archive, pins := range m.Archives {
		if filepath.Base(archive) != archive || !strings.HasSuffix(archive, ".zip") || m.Files[archive].Bytes > 48<<20 {
			return errors.New("bounded archive name required")
		}
		z, e := zip.OpenReader(filepath.Join(output, archive))
		if e != nil {
			return e
		}
		seen := map[string]bool{}
		for _, member := range z.File {
			pin, ok := pins[member.Name]
			if !ok || seen[member.Name] || !filepath.IsLocal(member.Name) || strings.Contains(member.Name, "\\") || member.UncompressedSize64 != uint64(pin.Bytes) || pin.Bytes < 0 || pin.Bytes > 768<<20 {
				z.Close()
				return errors.New("archive member inventory differs")
			}
			seen[member.Name] = true
			in, e := member.Open()
			if e != nil {
				z.Close()
				return e
			}
			h := sha256.New()
			n, e := io.Copy(h, io.LimitReader(in, pin.Bytes+1))
			closeErr := in.Close()
			if e != nil || closeErr != nil || n != pin.Bytes || hex.EncodeToString(h.Sum(nil)) != pin.SHA {
				z.Close()
				return errors.New("archive member digest/CRC differs")
			}
			decoded += n
			count++
		}
		z.Close()
		if len(seen) != len(pins) {
			return errors.New("missing archive member")
		}
	}
	if decoded != m.Decoded || count > 1200 {
		return errors.New("decoded archive inventory differs")
	}
	fmt.Printf("{\"status\":\"PASS\",\"public_files\":%d,\"archive_members\":%d,\"decoded_bytes\":%d,\"public_bytes\":%d}\n", len(m.Files)+1, count, decoded, total+int64(len(raw)))
	return nil
}
