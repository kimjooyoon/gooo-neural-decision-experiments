package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

func writeBounded(w io.Writer, raw []byte, used *int64) error {
	if len(raw) > 1<<20 || *used+int64(len(raw)) > threestudent.RawCap-(1<<20) {
		return errors.New("whole-study cap reached; prefix retained")
	}
	n, err := w.Write(raw)
	*used += int64(n)
	if err == nil && n != len(raw) {
		err = io.ErrShortWrite
	}
	return err
}
func saveRaw(path string, raw []byte, used *int64) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	err = writeBounded(f, raw, used)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func saveJSON(path string, v any, used *int64) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if privateText.Match(raw) {
		return errors.New("private metadata rejected")
	}
	return saveRaw(path, append(raw, '\n'), used)
}
func inventory(root string) (map[string]threestudent.Pin, int64, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, 0, err
	}
	pins := map[string]threestudent.Pin{}
	var total int64
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "own-three-") {
			continue
		}
		if !entry.IsDir() {
			return nil, 0, errors.New("regular study phase directory required")
		}
		err = filepath.WalkDir(filepath.Join(root, entry.Name()), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			p, e := threestudent.FilePin(path)
			if e != nil {
				return e
			}
			total += p.Bytes
			if total > threestudent.RawCap-(1<<20) {
				return errors.New("whole-study cap exceeded before preparation")
			}
			rel, e := filepath.Rel(root, path)
			if e != nil {
				return e
			}
			pins[filepath.ToSlash(rel)] = p
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	return pins, total, nil
}
