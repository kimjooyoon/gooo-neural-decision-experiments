package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const studyCap int64 = 768 << 20
const wholeCap int64 = 3 << 30
const reserve int64 = 16 << 20

type storage struct {
	output      string
	prior, used int64
	available   uint64
}

func exactSource(revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision || runtime.Version() != "go1.27.1" {
		return errors.New("exact Go 1.27.1 source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean published source required")
	}
	return nil
}

func newStorage(output string) (*storage, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(output)
	if err != nil || filepath.Dir(absolute) != filepath.Join(cwd, "runs") || !strings.HasPrefix(filepath.Base(absolute), "own-three-full-input-") {
		return nil, errors.New("fresh full-input phase under runs required")
	}
	if _, err = os.Lstat(absolute); !os.IsNotExist(err) {
		return nil, errors.New("phase exists; preserve previous attempt")
	}
	s := &storage{output: absolute}
	entries, err := os.ReadDir(filepath.Join(cwd, "runs"))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "own-three-") {
			continue
		}
		var size int64
		err = filepath.WalkDir(filepath.Join(cwd, "runs", entry.Name()), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.Type()&os.ModeSymlink != 0 {
				return errors.New("raw evidence symlink")
			}
			if d.IsDir() {
				return nil
			}
			info, e := d.Info()
			if e != nil {
				return e
			}
			if !info.Mode().IsRegular() {
				return errors.New("nonregular raw evidence")
			}
			size += info.Size()
			return nil
		})
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(entry.Name(), "own-three-full-input-") {
			s.used += size
		} else {
			s.prior += size
		}
	}
	var disk syscall.Statfs_t
	if err = syscall.Statfs(cwd, &disk); err != nil {
		return nil, err
	}
	s.available = disk.Bavail * uint64(disk.Bsize)
	if s.available < 4<<30 || s.used >= studyCap-reserve || s.prior+s.used >= wholeCap-reserve {
		return nil, errors.New("preparation storage bound")
	}
	return s, nil
}

func (s *storage) write(name string, raw []byte, appendMode bool) error {
	margin := reserve
	if name == "failure.json" {
		margin = 0
	}
	if filepath.Base(name) != name || s.used+int64(len(raw)) > studyCap-margin || s.prior+s.used+int64(len(raw)) > wholeCap-margin {
		return errors.New("bounded retained evidence; stop and preserve prefix")
	}
	flag := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if appendMode {
		flag = os.O_WRONLY | os.O_APPEND
	}
	f, err := os.OpenFile(filepath.Join(s.output, name), flag, 0644)
	if err != nil {
		return err
	}
	n, err := f.Write(raw)
	s.used += int64(n)
	closed := f.Close()
	if err != nil {
		return err
	}
	if n != len(raw) {
		return errors.New("partial evidence write")
	}
	return closed
}
func (s *storage) line(name string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("bounded record required")
	}
	return s.write(name, append(raw, '\n'), true)
}
func (s *storage) json(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("bounded metadata required")
	}
	return s.write(name, append(raw, '\n'), false)
}

func writeStoragePreflight(future, revision, output string) error {
	if err := exactSource(revision); err != nil {
		return err
	}
	s, err := newStorage(future)
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(output)
	if err != nil || filepath.Dir(filepath.Dir(absolute)) != filepath.Dir(s.output) || !strings.HasPrefix(filepath.Base(filepath.Dir(absolute)), "own-three-full-input-") {
		return errors.New("storage report must belong to an existing full-input phase")
	}
	info, err := os.Lstat(filepath.Dir(absolute))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("regular existing storage report directory required")
	}
	value := map[string]any{"schema": "gooo/full-input-storage-preflight/v1", "status": "PASS", "source_revision": revision, "future_phase": filepath.Base(future), "prior_own_three_bytes": s.prior, "existing_full_input_bytes": s.used, "new_study_cap_bytes": studyCap, "whole_own_three_cap_bytes": wholeCap, "failure_reserve_bytes": reserve, "available_bytes": s.available, "optimizer_updates": 0, "model_predictions": 0}
	raw, err := storageReportBytes(value, s.used)
	if err != nil {
		return err
	}
	if s.used+int64(len(raw)) > studyCap-reserve || s.prior+s.used+int64(len(raw)) > wholeCap-reserve {
		return errors.New("storage report exceeds evidence reserve")
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(raw)
	return err
}

// Include this report's own bytes so the optimizer can compare a fresh inventory.
func storageReportBytes(value map[string]any, before int64) ([]byte, error) {
	for attempt := 0; attempt < 10; attempt++ {
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, err
		}
		raw = append(raw, '\n')
		next := before + int64(len(raw))
		if value["existing_full_input_bytes"] == next {
			return raw, nil
		}
		value["existing_full_input_bytes"] = next
	}
	return nil, errors.New("storage report extent did not converge")
}
