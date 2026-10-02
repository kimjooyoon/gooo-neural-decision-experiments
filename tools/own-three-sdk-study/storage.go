package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const lineCap = 1 << 20
const receiptReserve = 2 << 20

type storage struct {
	Used  int64
	Prior int64
}

func inventory() (map[string]threestudent.Pin, int64, error) {
	all := map[string]threestudent.Pin{}
	var used int64
	entries, err := os.ReadDir("runs")
	if err != nil {
		return nil, 0, err
	}
	for _, d := range entries {
		if !strings.HasPrefix(d.Name(), "own-three-") {
			continue
		}
		if !d.IsDir() {
			return nil, 0, errors.New("regular own-three phase directory required")
		}
		err = filepath.WalkDir(filepath.Join("runs", d.Name()), func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			pin, e := threestudent.FilePin(p)
			if e != nil {
				return e
			}
			all[filepath.ToSlash(p)] = pin
			used += pin.Bytes
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	return all, used, nil
}
func (s *storage) beforeCall() error {
	if s.Used+receiptReserve+lineCap > threestudent.RawCap {
		return errors.New("whole-study 768 MiB raw cap: stop before another session; completed prefix retained")
	}
	return nil
}
func (s *storage) write(f *os.File, raw []byte, reserve int64) error {
	if s.Used+int64(len(raw))+reserve > threestudent.RawCap {
		return errors.New("whole-study raw cap reached; earlier complete bytes retained")
	}
	n, err := f.Write(raw)
	s.Used += int64(n)
	if err != nil {
		return err
	}
	if n != len(raw) {
		return errors.New("short retained evidence write")
	}
	return nil
}
func (s *storage) save(name string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = s.write(f, append(raw, '\n'), 0); err != nil {
		return err
	}
	return f.Sync()
}
