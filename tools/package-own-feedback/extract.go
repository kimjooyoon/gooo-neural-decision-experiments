package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func pinManifest(dir, expected string) (string, error) {
	if expected != "" && !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(expected) {
		return "", errors.New("manifest pin must be an exact SHA-256")
	}
	p := filepath.Join(dir, "publication-manifest.json")
	s, err := os.Lstat(p)
	if err != nil || !s.Mode().IsRegular() || s.Size() > 1<<20 {
		return "", errors.New("bounded regular manifest required")
	}
	a, err := hashFile(p)
	if err != nil {
		return "", err
	}
	if expected != "" && a.SHA != expected {
		return "", errors.New("independent manifest pin differs")
	}
	return a.SHA, nil
}

func safeMemberPath(name string) bool {
	return name != "" && name != "." && !path.IsAbs(name) && path.Clean(name) == name &&
		!strings.Contains(name, "\\") && !strings.Contains(name, ":") &&
		name != ".." && !strings.HasPrefix(name, "../")
}

// Only called after the entire allowlisted archive has passed verification.
// Rehash each streamed copy so the extracted bytes retain the same binding.
func extractEvidence(files []*zip.File, expected map[string]entry, dest string) error {
	if dest == "" || filepath.Clean(dest) == "." || len(files) != len(expected) {
		return errors.New("fresh exact raw evidence target required")
	}
	seen := map[string]bool{}
	var total int64
	for _, f := range files {
		a, ok := expected[f.Name]
		if !ok || seen[f.Name] || !safeMemberPath(f.Name) || !f.Mode().IsRegular() ||
			a.Path != f.Name || a.Bytes < 0 || a.Bytes > 256<<20 || uint64(a.Bytes) != f.UncompressedSize64 {
			return errors.New("unsafe raw extraction entry")
		}
		seen[f.Name] = true
		total += a.Bytes
		if total > 640<<20 {
			return errors.New("raw extraction expanded byte cap exceeded")
		}
	}
	if err := os.Mkdir(dest, 0755); err != nil {
		return errors.New("raw evidence target must not already exist")
	}
	for _, f := range files {
		a := expected[f.Name]
		to := filepath.Join(dest, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
			return err
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			r.Close()
			return err
		}
		h := sha256.New()
		var buffer [32768]byte
		n, copyErr := io.CopyBuffer(io.MultiWriter(w, h), io.LimitReader(r, a.Bytes+1), buffer[:])
		readErr, writeErr := r.Close(), w.Close()
		if copyErr != nil || readErr != nil || writeErr != nil || n != a.Bytes || hex.EncodeToString(h.Sum(nil)) != a.SHA {
			return errors.New("raw extraction hash, CRC or size differs")
		}
	}
	return nil
}
