package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const fullPhaseCap int64 = 128 << 20
const fullFailureReserve int64 = 8 << 20
const fullInvocationReserve int64 = 4 << 20

type fullStorage struct {
	Available uint64           `json:"available_bytes"`
	Whole     int64            `json:"whole_own_three_bytes"`
	Study     int64            `json:"full_input_and_arithmetic_study_bytes"`
	Phase     int64            `json:"current_phase_bytes"`
	Inventory map[string]int64 `json:"logical_bytes_by_phase"`
}

func boundedFullFile(path string, maximum int64) []byte {
	info, err := os.Lstat(path)
	must(err)
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximum {
		panic("bounded regular input required")
	}
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maximum+1))
	must(err)
	if int64(len(b)) != info.Size() {
		panic("input changed while reading")
	}
	return b
}

func fullDirectorySize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular evidence at %s", p)
		}
		if info.Size() > 3<<30 || total > (3<<30)-info.Size() {
			return fmt.Errorf("own-three evidence extent exceeds limit")
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func inspectFullStorage(output string, fresh bool) fullStorage {
	if filepath.Dir(filepath.Clean(output)) != "runs" || !strings.HasPrefix(filepath.Base(output), "own-three-full-input-native-") {
		panic("registered full-input native output prefix required")
	}
	if fresh {
		if _, err := os.Lstat(output); !os.IsNotExist(err) {
			panic("fresh output required; retain the original attempt")
		}
	}
	s := fullStorage{Inventory: map[string]int64{}}
	entries, err := os.ReadDir("runs")
	must(err)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "own-three-") {
			continue
		}
		size, err := fullDirectorySize(filepath.Join("runs", entry.Name()))
		must(err)
		s.Inventory[entry.Name()] = size
		s.Whole += size
		if strings.HasPrefix(entry.Name(), "own-three-full-input-") || strings.HasPrefix(entry.Name(), "own-three-separate-") {
			s.Study += size
		}
		if entry.Name() == filepath.Base(output) {
			s.Phase = size
		}
	}
	var disk syscall.Statfs_t
	must(syscall.Statfs("runs", &disk))
	s.Available = disk.Bavail * uint64(disk.Bsize)
	if !fullStorageFits(s, fullInvocationReserve) {
		panic("native evidence storage bound; preserve completed prefix")
	}
	return s
}

func fullStorageFits(s fullStorage, next int64) bool {
	return next >= 0 && next <= fullPhaseCap-fullFailureReserve && s.Available >= 4<<30 &&
		s.Phase >= 0 && s.Study >= s.Phase && s.Whole >= s.Study &&
		s.Phase <= fullPhaseCap-next-fullFailureReserve && s.Study <= (768<<20)-next-fullFailureReserve &&
		s.Whole <= (3<<30)-next-fullFailureReserve
}

func validateFullRuntime(raw, parent []byte, cases []pathplan.TestCase, revision string) int {
	var r runtimeResult
	must(json.Unmarshal(raw, &r))
	if r.Observation.Stage != "COMPLETE" || r.Observation.Producer != revision || !bytes.Equal(r.Parent, parent) || r.Observation.ParentSHA != sha(parent) || len(cases) != 24 || len(r.Observation.Cases) != len(cases) {
		panic("native source/runtime binding differs")
	}
	var detail struct {
		Observation struct {
			Build struct {
				Started   bool `json:"started"`
				Completed bool `json:"completed"`
				Exit      *int `json:"exit_code"`
			} `json:"build"`
			Runs []struct {
				Started   bool `json:"started"`
				Completed bool `json:"completed"`
				Exit      *int `json:"exit_code"`
			} `json:"runs"`
		} `json:"observation"`
	}
	must(json.Unmarshal(raw, &detail))
	if !detail.Observation.Build.Started || !detail.Observation.Build.Completed || detail.Observation.Build.Exit == nil || *detail.Observation.Build.Exit != 0 || len(detail.Observation.Runs) != 2 {
		panic("native build/run denominator differs")
	}
	for _, run := range detail.Observation.Runs {
		if !run.Started || !run.Completed || run.Exit == nil || *run.Exit != 0 {
			panic("native execution incomplete")
		}
	}
	passed := 0
	for i, c := range r.Observation.Cases {
		if c.Input != cases[i].Input || c.Expected != cases[i].Expected || c.Passed != (c.Actual == c.Expected) {
			panic("independent native finite oracle differs")
		}
		if c.Passed {
			passed++
		}
	}
	want := "PROGRESS"
	if passed == 24 {
		want = "PASS"
	}
	a := runtimeAxis(r, "runtime_finite_accuracy")
	if a.Numerator != passed || a.Denominator != 24 || a.Status != want {
		panic("finite accuracy accounting differs")
	}
	for _, id := range []string{"runtime_source_replay", "runtime_build", "execution_boundary", "runtime_deterministic_replay", "reverse_observation_coverage"} {
		if runtimeAxis(r, id).Status != "PASS" {
			panic("native axis incomplete: " + id)
		}
	}
	disjoint := runtimeAxis(r, "runtime_selection_disjointness")
	if disjoint.Numerator != 8 || disjoint.Denominator != 24 || r.Receipt.Aggregate != nil || r.Receipt.First.ID != "permission_boundary" {
		panic("finite scope or unresolved boundary differs")
	}
	return passed
}
