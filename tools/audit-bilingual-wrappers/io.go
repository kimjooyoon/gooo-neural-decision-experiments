package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const capBytes = 96 << 20

var privateText = regexp.MustCompile(`/Users/|/home/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_`)

type journal struct {
	file *os.File
	used *int64
}

func openJournal(root, name string, used *int64) (*journal, error) {
	f, e := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, e
	}
	return &journal{f, used}, nil
}
func (j *journal) append(v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	b = append(b, '\n')
	if len(b) > 1<<20 || *j.used+int64(len(b)) > capBytes-(4<<20) || privateText.Match(b) {
		return fmt.Errorf("bounded public journal required")
	}
	n, e := j.file.Write(b)
	*j.used += int64(n)
	return e
}
func save(root, name string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	b = append(b, '\n')
	if privateText.Match(b) || len(b) > 2<<20 {
		return fmt.Errorf("bounded public metadata required")
	}
	f, e := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	closed := f.Close()
	if e != nil {
		return e
	}
	return closed
}
func pin(path string) (map[string]any, error) {
	s, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !s.Mode().IsRegular() {
		return nil, fmt.Errorf("regular file required")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, f)
	if e != nil {
		return nil, e
	}
	return map[string]any{"bytes": n, "sha256": fmt.Sprintf("%x", h.Sum(nil))}, nil
}
