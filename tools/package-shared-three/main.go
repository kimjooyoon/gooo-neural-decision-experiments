// package-shared-three publishes a closed evidence inventory and verifies every byte.
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

type member struct {
	Name  string `json:"name"`
	Bytes int    `json:"bytes"`
	SHA   string `json:"sha256"`
}
type manifest struct {
	Schema       string   `json:"schema"`
	Files        []member `json:"files"`
	Bytes        int      `json:"decoded_bytes"`
	ArchiveSHA   string   `json:"archive_sha256"`
	ArchiveBytes int      `json:"archive_bytes"`
	Scope        string   `json:"scope"`
}

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|-----BEGIN .*PRIVATE KEY-----`)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(path string) []byte {
	s, err := os.Lstat(path)
	must(err)
	if !s.Mode().IsRegular() || s.Size() > 8<<20 {
		panic("bounded regular publication file required")
	}
	b, err := os.ReadFile(path)
	must(err)
	return b
}
func walkJSON(v any) error {
	switch x := v.(type) {
	case string:
		if privateText.MatchString(x) {
			return errors.New("decoded private text")
		}
	case []any:
		for _, v := range x {
			if e := walkJSON(v); e != nil {
				return e
			}
		}
	case map[string]any:
		for k, v := range x {
			if privateText.MatchString(k) {
				return errors.New("private key")
			}
			if e := walkJSON(v); e != nil {
				return e
			}
		}
	}
	return nil
}
func privacy(name string, raw []byte) {
	if strings.HasSuffix(name, "/weights.bin") {
		return
	}
	if !utf8.Valid(raw) || privateText.Match(raw) {
		panic("private or nontext public material: " + name)
	}
	check := func(b []byte) {
		must(decision.RejectDuplicateJSONKeys(b))
		var v any
		must(json.Unmarshal(b, &v))
		must(walkJSON(v))
	}
	if strings.HasSuffix(name, ".json") {
		check(raw)
	}
	if strings.HasSuffix(name, ".jsonl") {
		s := bufio.NewScanner(bytes.NewReader(raw))
		s.Buffer(make([]byte, 32768), 1<<20)
		for s.Scan() {
			check(s.Bytes())
		}
		must(s.Err())
	}
}
func files(training, native string) map[string]string {
	m := map[string]string{
		"README.md": "docs/shared-three-judgment-results-20261003.md", "LICENSE": "LICENSE",
		"protocol.md":            "docs/shared-three-judgment-preregistration-20261002.md",
		"storage-amendment.md":   "docs/shared-three-storage-amendment-20261003.md",
		"storage-preflight.json": "preexecution/shared-three-storage-preflight-20261003.json",
		"original-failure.json":  "preexecution/shared-three-storage-failure-20261003.json",
	}
	for _, name := range []string{"preexecution.json", "report.json", "legacy-preservation.json", "go-audit.json"} {
		m["training/"+name] = filepath.Join(training, name)
	}
	for _, arm := range []string{"dense", "shared-local"} {
		for _, name := range []string{"training-report.json", "go-parity.jsonl"} {
			m["training/"+arm+"/"+name] = filepath.Join(training, arm, name)
		}
		for _, stage := range []string{"fp32", "qat"} {
			base := "training/" + arm + "/" + stage + "/"
			m[base+"optimizer-updates.jsonl"] = filepath.Join(training, arm, stage, "optimizer-updates.jsonl")
			for epoch := 1; epoch <= 100; epoch++ {
				name := fmt.Sprintf("epoch-%03d.json", epoch)
				m[base+name] = filepath.Join(training, arm, stage, name)
			}
		}
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, name := range []string{"model.json", "weights.bin"} {
				m["models/"+arm+"/"+variant+"/"+name] = filepath.Join(training, arm, "models", variant, name)
			}
		}
	}
	for _, name := range []string{"preexecution.json", "report.json", "independent-consumption.json", "independent-consumer.go"} {
		m["native/"+name] = filepath.Join(native, name)
	}
	for _, family := range threecompositionstudy.Families {
		for _, language := range []string{"en", "ko"} {
			for _, arm := range []string{"dense", "shared-local"} {
				for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
					name := family + "-" + language + "-" + arm + "-" + variant
					for _, file := range []string{"input.gooo", "plan.json", "generation.json", "generated.go", "runtime-cases.json", "runtime.json", "observation.json"} {
						m["native/"+name+"/"+file] = filepath.Join(native, name, file)
					}
				}
			}
		}
	}
	for _, name := range []string{"train_shared_three_judgment_v1.py", "shared_evidence_v1.py", "train_own_three_feedback_v1.py", "train_pilot_v2.py"} {
		m["source/training/"+name] = filepath.Join("training", name)
	}
	for _, name := range []string{"main.go", "training.go", "shared.go", "shared_development.go"} {
		m["source/auditor/"+name] = filepath.Join("tools/audit-own-three-models", name)
	}
	for _, name := range []string{"main.go", "shared.go"} {
		m["source/native-runner/"+name] = filepath.Join("cmd/native-runtime-observe", name)
	}
	return m
}
func pack(training, native, out string) {
	all := files(training, native)
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	m := manifest{Schema: "gooo/shared-three-public-evidence/v1", Scope: "Closed six-model, all-epoch/update, full Go-development and 96 native/runtime evidence inventory; original failure preserved. Text and decoded JSON scanned for private host paths and credential patterns. No environment dump or external/pretrained weights."}
	must(os.Mkdir(out, 0700))
	f, err := os.OpenFile(filepath.Join(out, "evidence.zip"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(err)
	z := zip.NewWriter(f)
	for _, name := range names {
		raw := read(all[name])
		privacy(name, raw)
		m.Bytes += len(raw)
		if m.Bytes > 64<<20 {
			panic("public decoded cap")
		}
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0600)
		w, err := z.CreateHeader(h)
		must(err)
		_, err = w.Write(raw)
		must(err)
		m.Files = append(m.Files, member{name, len(raw), threecohort.SHA(raw)})
		if strings.HasPrefix(name, "models/") || name == "README.md" || name == "LICENSE" || name == "training/go-audit.json" || name == "native/independent-consumption.json" {
			path := filepath.Join(out, name)
			must(os.MkdirAll(filepath.Dir(path), 0700))
			must(os.WriteFile(path, raw, 0600))
		}
	}
	must(z.Close())
	must(f.Sync())
	must(f.Close())
	archive, err := os.ReadFile(filepath.Join(out, "evidence.zip"))
	must(err)
	m.ArchiveBytes = len(archive)
	m.ArchiveSHA = threecohort.SHA(archive)
	raw, err := json.MarshalIndent(m, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(out, "manifest.json"), append(raw, '\n'), 0600))
	verify(out, true)
}
func verify(root string, standalone bool) {
	var m manifest
	must(json.Unmarshal(read(filepath.Join(root, "manifest.json")), &m))
	if m.Schema != "gooo/shared-three-public-evidence/v1" || len(m.Files) != 1116 || m.Bytes > 64<<20 || m.ArchiveBytes <= 0 || m.ArchiveBytes > 64<<20 {
		panic("closed shared bundle extent")
	}
	info, err := os.Lstat(filepath.Join(root, "evidence.zip"))
	must(err)
	if !info.Mode().IsRegular() || info.Size() != int64(m.ArchiveBytes) {
		panic("regular bounded archive required")
	}
	raw, err := os.ReadFile(filepath.Join(root, "evidence.zip"))
	must(err)
	if len(raw) != m.ArchiveBytes || threecohort.SHA(raw) != m.ArchiveSHA {
		panic("archive byte mismatch")
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	must(err)
	if len(z.File) != len(m.Files) {
		panic("archive member count")
	}
	total := 0
	for i, f := range z.File {
		pin := m.Files[i]
		if f.Name != pin.Name || int(f.UncompressedSize64) != pin.Bytes || pin.Bytes > 8<<20 || filepath.IsAbs(pin.Name) || strings.Contains(pin.Name, "..") || (i > 0 && m.Files[i-1].Name >= pin.Name) {
			panic("closed archive member identity")
		}
		in, err := f.Open()
		must(err)
		b, err := io.ReadAll(io.LimitReader(in, int64(pin.Bytes)+1))
		must(err)
		must(in.Close())
		if len(b) != pin.Bytes || threecohort.SHA(b) != pin.SHA {
			panic("member SHA/size/CRC mismatch")
		}
		privacy(pin.Name, b)
		total += len(b)
		if standalone && (strings.HasPrefix(pin.Name, "models/") || pin.Name == "README.md" || pin.Name == "LICENSE" || pin.Name == "training/go-audit.json" || pin.Name == "native/independent-consumption.json") {
			if !bytes.Equal(read(filepath.Join(root, pin.Name)), b) {
				panic("standalone export differs")
			}
		}
	}
	if total != m.Bytes {
		panic("decoded extent mismatch")
	}
	fmt.Printf("PASS: %d members, %d decoded bytes, %d ZIP bytes\n", len(m.Files), total, len(raw))
}
func main() {
	mode := flag.String("mode", "verify", "pack, verify, or verify-bundle")
	training := flag.String("training", "", "training capture")
	native := flag.String("native", "", "native capture")
	out := flag.String("output", "", "fresh stage or existing stage for verify")
	flag.Parse()
	if *out == "" {
		panic("output required")
	}
	switch *mode {
	case "pack":
		pack(*training, *native, *out)
	case "verify":
		verify(*out, true)
	case "verify-bundle":
		verify(*out, false)
	default:
		panic("unknown mode")
	}
}
