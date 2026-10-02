// package-compact-shared publishes a closed, bounded local study inventory.
package main

import (
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
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

type file struct {
	Name  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema       string `json:"schema"`
	Source       string `json:"audit_source_revision"`
	Files        []file `json:"files"`
	DecodedSHA   string `json:"decoded_comparisons_sha256"`
	DecodedBytes int    `json:"decoded_comparisons_bytes"`
	States       int    `json:"state_rows"`
}

func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func read(p string) ([]byte, error) {
	s, e := os.Lstat(p)
	if e != nil || !s.Mode().IsRegular() || s.Size() <= 0 || s.Size() > 16<<20 {
		return nil, errors.New("bounded regular publication input required")
	}
	return os.ReadFile(p)
}
func safeText(b []byte) error {
	for _, s := range []string{"/Users/", "/home/", "/private/", "Bearer ", "hf_", "ghp_", "github_pat_", "BEGIN PRIVATE KEY", "HF_TOKEN"} {
		if bytes.Contains(b, []byte(s)) {
			return errors.New("private text in publication")
		}
	}
	return nil
}
func run(source, output string) error {
	if source == "" || output == "" {
		return errors.New("source and fresh output required")
	}
	if _, e := os.Lstat(output); !os.IsNotExist(e) {
		return errors.New("fresh output required")
	}
	names := []string{"preexecution.json", "report.json"}
	for _, v := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		names = append(names, "models/"+v+"/model.json", "models/"+v+"/weights.bin")
	}
	sort.Strings(names)
	contents := map[string][]byte{}
	for _, n := range names {
		b, e := read(filepath.Join(source, n))
		if e != nil {
			return e
		}
		if strings.HasSuffix(n, ".json") {
			if e = decision.RejectDuplicateJSONKeys(b); e != nil {
				return e
			}
			if e = safeText(b); e != nil {
				return e
			}
		}
		contents[n] = b
	}
	var report struct {
		Status   string `json:"status"`
		Revision string `json:"source_revision"`
		Ledger   string `json:"comparison_ledger_sha256"`
		States   int    `json:"frozen_states"`
	}
	if e := json.Unmarshal(contents["report.json"], &report); e != nil {
		return e
	}
	if report.Status != "PASS" || len(report.Revision) != 40 || report.States != 10739 {
		return errors.New("complete frozen audit required")
	}
	for _, v := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		if _, e := jointdecision.LoadSharedThree(filepath.Join(source, "models", v, "model.json")); e != nil {
			return e
		}
	}
	ledger, e := read(filepath.Join(source, "comparisons.jsonl"))
	if e != nil {
		return e
	}
	if e = safeText(ledger); e != nil {
		return e
	}
	if hash(ledger) != report.Ledger || bytes.Count(ledger, []byte{'\n'}) != 10739 {
		return errors.New("complete ledger pin differs")
	}
	var compressed bytes.Buffer
	w, e := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if e != nil {
		return e
	}
	if _, e = w.Write(ledger); e != nil {
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	z, e := gzip.NewReader(bytes.NewReader(compressed.Bytes()))
	if e != nil {
		return e
	}
	decoded, e := io.ReadAll(io.LimitReader(z, 16<<20))
	if e != nil {
		return e
	}
	if e = z.Close(); e != nil {
		return e
	}
	if !bytes.Equal(decoded, ledger) {
		return errors.New("compressed ledger differs")
	}
	names = append(names, "comparisons.jsonl.gz")
	contents["comparisons.jsonl.gz"] = compressed.Bytes()
	sort.Strings(names)
	m := manifest{Schema: "gooo/shared-three-compact-publication/v1", Source: report.Revision, DecodedSHA: hash(ledger), DecodedBytes: len(ledger), States: 10739}
	for _, n := range names {
		b := contents[n]
		m.Files = append(m.Files, file{n, hash(b), len(b)})
	}
	b, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		return e
	}
	contents["manifest.json"] = append(b, '\n')
	names = append(names, "manifest.json")
	var total int
	for _, b := range contents {
		total += len(b)
	}
	if total > 8<<20 {
		return errors.New("bounded public output exceeded")
	}
	if e = os.MkdirAll(filepath.Dir(output), 0755); e != nil {
		return e
	}
	if e = os.Mkdir(output, 0755); e != nil {
		return e
	}
	for _, n := range names {
		p := filepath.Join(output, n)
		if e = os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			return e
		}
		f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		_, e = f.Write(contents[n])
		ce := f.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		actual, e := read(p)
		if e != nil || !bytes.Equal(actual, contents[n]) {
			return errors.New("written publication differs")
		}
	}
	fmt.Printf("PASS: %d public files, %d encoded bytes, all %d frozen state rows preserved\n", len(names), total, m.States)
	return nil
}
func main() {
	s := flag.String("source", "", "complete local compact audit")
	o := flag.String("output", "", "fresh public directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments")
		os.Exit(2)
	}
	if e := run(*s, *o); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
