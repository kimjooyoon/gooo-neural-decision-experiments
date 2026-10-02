package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	stdhash "hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

const rawCap int64 = 768 << 20
const lineCap = 1 << 20
const metadataReserve int64 = 1 << 20

var hex40 = regexp.MustCompile(`^[0-9a-f]{40}$`)
var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

type filePin struct {
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
	Lines int    `json:"lines,omitempty"`
}

func fileSHA(path string) (string, error) {
	s, err := os.Lstat(path)
	if err != nil || !s.Mode().IsRegular() {
		return "", errors.New("regular evidence file required")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	var buffer [32768]byte
	if _, err = io.CopyBuffer(h, f, buffer[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func save(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if len(raw) > lineCap || privateText.Match(raw) {
		return errors.New("bounded public metadata required")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func readJSON(path string, v any) error {
	s, err := os.Lstat(path)
	if err != nil || !s.Mode().IsRegular() || s.Size() > lineCap {
		return errors.New("bounded regular metadata required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if privateText.Match(raw) {
		return errors.New("private metadata rejected")
	}
	return threecohort.Decode(raw, v)
}
func inventory(root string) (map[string]filePin, int64, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, 0, err
	}
	pins := map[string]filePin{}
	var total int64
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "own-three-") {
			continue
		}
		if !entry.IsDir() {
			return nil, 0, errors.New("regular three-choice phase directory required")
		}
		err = filepath.WalkDir(filepath.Join(root, entry.Name()), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			s, e := d.Info()
			if e != nil {
				return e
			}
			if !s.Mode().IsRegular() {
				return errors.New("nonregular raw evidence rejected")
			}
			total += s.Size()
			if total > rawCap-metadataReserve {
				return errors.New("whole-study evidence cap reached before teacher")
			}
			sha, e := fileSHA(path)
			if e != nil {
				return e
			}
			rel, e := filepath.Rel(root, path)
			if e != nil {
				return e
			}
			pins[filepath.ToSlash(rel)] = filePin{SHA: sha, Bytes: s.Size()}
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	return pins, total, nil
}

type journal struct {
	file  *os.File
	hash  stdhash.Hash
	bytes int64
	lines int
	used  *int64
}

func openJournal(path string, used *int64) (*journal, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}
	return &journal{file: f, hash: sha256.New(), used: used}, nil
}
func (j *journal) appendRaw(raw []byte) error {
	if len(raw)+1 > lineCap || *j.used+int64(len(raw)+1) > rawCap-metadataReserve {
		return errors.New("whole-study evidence cap reached; actual prefix retained")
	}
	if privateText.Match(raw) {
		return errors.New("private text in journal rejected")
	}
	raw = append(raw, '\n')
	n, err := j.file.Write(raw)
	_, _ = j.hash.Write(raw[:n])
	j.bytes += int64(n)
	*j.used += int64(n)
	if err == nil && n != len(raw) {
		err = io.ErrShortWrite
	}
	if err == nil {
		j.lines++
	}
	return err
}
func (j *journal) pin() filePin {
	return filePin{hex.EncodeToString(j.hash.Sum(nil)), j.bytes, j.lines}
}
func scan(path string, visit func([]byte) error) (filePin, error) {
	s, err := os.Lstat(path)
	if err != nil || !s.Mode().IsRegular() || s.Size() > rawCap {
		return filePin{}, errors.New("bounded regular journal required")
	}
	f, err := os.Open(path)
	if err != nil {
		return filePin{}, err
	}
	defer f.Close()
	h := sha256.New()
	r := bufio.NewScanner(io.TeeReader(f, h))
	r.Buffer(make([]byte, 32768), lineCap)
	lines := 0
	for r.Scan() {
		if privateText.Match(r.Bytes()) {
			return filePin{}, errors.New("private raw evidence rejected")
		}
		if err = visit(r.Bytes()); err != nil {
			return filePin{}, err
		}
		lines++
	}
	if r.Err() != nil {
		return filePin{}, r.Err()
	}
	return filePin{hex.EncodeToString(h.Sum(nil)), s.Size(), lines}, nil
}
