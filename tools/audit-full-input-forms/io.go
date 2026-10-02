package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const rawCap int64 = 224 << 20
const failureReserve int64 = 16 << 20
const publicAllowance int64 = 64 << 20
const maxRow = 64 << 10

type storageState struct {
	Available uint64           `json:"available_bytes"`
	Whole     int64            `json:"whole_own_three_bytes"`
	Study     int64            `json:"full_input_and_arithmetic_bytes"`
	Phase     int64            `json:"phase_bytes"`
	Inventory map[string]int64 `json:"logical_bytes_by_phase"`
}

func outputPath(output string) error {
	if filepath.Clean(output) != output || filepath.Dir(output) != "runs" || !strings.HasPrefix(filepath.Base(output), phasePrefix) || filepath.Base(output) == phasePrefix {
		return errors.New("registered direct-child output path required")
	}
	info, err := os.Lstat("runs")
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("regular runs directory required")
	}
	return nil
}

func directoryBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() || info.Size() > 3<<30 || total > (3<<30)-info.Size() {
			return fmt.Errorf("invalid evidence extent: %s", path)
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func storageFits(s storageState, fresh bool) bool {
	if s.Available < 4<<30 || s.Phase < 0 || s.Study < s.Phase || s.Whole < s.Study || s.Phase > rawCap {
		return false
	}
	// Reserve the entire remaining raw allowance and public bundle, rather than
	// counting only the next write. The reserve includes failure diagnostics.
	reserve := rawCap + publicAllowance - s.Phase
	if fresh && s.Phase != 0 {
		return false
	}
	return s.Study <= (768<<20)-reserve && s.Whole <= (3<<30)-reserve
}

func storagePreflight(output string, fresh bool) (storageState, error) {
	s := storageState{Inventory: map[string]int64{}}
	if err := outputPath(output); err != nil {
		return s, err
	}
	info, err := os.Lstat(output)
	if fresh {
		if !os.IsNotExist(err) {
			return s, errors.New("fresh output required; preserve previous attempt")
		}
	} else if err != nil || !info.IsDir() {
		return s, errors.New("existing evidence directory required")
	}
	entries, err := os.ReadDir("runs")
	if err != nil {
		return s, err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "own-three-") {
			continue
		}
		n, err := directoryBytes(filepath.Join("runs", entry.Name()))
		if err != nil {
			return s, err
		}
		s.Inventory[entry.Name()] = n
		s.Whole += n
		if strings.HasPrefix(entry.Name(), "own-three-full-input-") || strings.HasPrefix(entry.Name(), "own-three-separate-") {
			s.Study += n
		}
		if entry.Name() == filepath.Base(output) {
			s.Phase = n
		}
	}
	var disk syscall.Statfs_t
	if err := syscall.Statfs("runs", &disk); err != nil {
		return s, err
	}
	s.Available = uint64(disk.Bavail) * uint64(disk.Bsize)
	if !storageFits(s, fresh) {
		return s, errors.New("registered combined storage budget unavailable")
	}
	return s, nil
}

func boundedFile(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum {
		return nil, errors.New("bounded regular input required")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maximum+1))
	if err == nil && int64(len(b)) != info.Size() {
		err = errors.New("input extent changed")
	}
	return b, err
}

type fileDigest struct {
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}

func filePin(path string) (fileDigest, error) {
	var p fileDigest
	info, err := os.Lstat(path)
	if err != nil {
		return p, err
	}
	if !info.Mode().IsRegular() || info.Size() > rawCap {
		return p, errors.New("bounded regular evidence required")
	}
	f, err := os.Open(path)
	if err != nil {
		return p, err
	}
	defer f.Close()
	h := sha256.New()
	p.Bytes, err = io.Copy(h, io.LimitReader(f, rawCap+1))
	if err != nil {
		return p, err
	}
	if p.Bytes != info.Size() {
		return p, errors.New("evidence changed while hashing")
	}
	p.SHA = hex.EncodeToString(h.Sum(nil))
	return p, nil
}

func saveFresh(root, name string, value any) error {
	if filepath.Base(name) != name {
		return errors.New("direct file name required")
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if len(b) > 1<<20 {
		return errors.New("metadata extent exceeds bound")
	}
	current, err := directoryBytes(root)
	if err != nil {
		return err
	}
	cap := rawCap - failureReserve
	if name == "failure.json" {
		cap = rawCap
	}
	if current > cap-int64(len(b)) {
		return errors.New("metadata storage bound")
	}
	f, err := os.OpenFile(filepath.Join(root, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, writeErr := f.Write(b)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return closeErr
}

type journal struct {
	file   *os.File
	used   *int64
	closed bool
}

func newJournal(root, name string, used *int64) (*journal, error) {
	if filepath.Base(name) != name || used == nil {
		return nil, errors.New("bounded journal arguments required")
	}
	n, err := directoryBytes(root)
	if err != nil {
		return nil, err
	}
	*used = n
	f, err := os.OpenFile(filepath.Join(root, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	return &journal{file: f, used: used}, nil
}
func (w *journal) Append(value any) error {
	if w.closed {
		return errors.New("journal closed")
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if len(b) > maxRow || *w.used < 0 || *w.used > rawCap-failureReserve-int64(len(b)) {
		return errors.New("journal row or phase bound; prefix retained")
	}
	n, err := w.file.Write(b)
	*w.used += int64(n)
	if err == nil && n != len(b) {
		return io.ErrShortWrite
	}
	return err
}
func (w *journal) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	return w.file.Close()
}
