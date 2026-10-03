package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type pin struct {
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}

func filePin(path string) pin {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	h := sha256.Sum256(b)
	return pin{hex.EncodeToString(h[:]), int64(len(b))}
}
func main() {
	var r struct {
		Status  string            `json:"status"`
		Source  string            `json:"source_revision"`
		Files   map[string][3]pin `json:"files"`
		Sources map[string]pin    `json:"computational_sources"`
	}
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(b, &r); err != nil {
		panic(err)
	}
	if r.Status != "PASS" || len(r.Files) != 61 || len(r.Sources) < 20 {
		panic("closed reader result required")
	}
	for name, p := range r.Files {
		if !filepath.IsLocal(name) || filePin(filepath.Join(os.Args[2], name)) != p[2] {
			panic("downloaded file differs: " + name)
		}
	}
	for name, p := range r.Sources {
		if !filepath.IsLocal(name) || filePin(filepath.Join(os.Args[3], name)) != p {
			panic("source bytes differ: " + name)
		}
	}
	var audit struct {
		Source string `json:"auditor_source_revision"`
	}
	b, err = os.ReadFile(filepath.Join(os.Args[2], "report.json"))
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(b, &audit); err != nil || audit.Source != r.Source {
		panic("reader/audit source differs")
	}
	fmt.Printf("{\"status\":\"PASS\",\"downloaded_files\":%d,\"exact_source_files\":%d,\"reader_audit_source\":%q,\"model_predictions\":0}\n", len(r.Files), len(r.Sources), r.Source)
}
