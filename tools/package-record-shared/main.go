// Package the new shared pilot, reusing the immutable paired source bank.
package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func hash(raw []byte) string { sum := sha256.Sum256(raw); return fmt.Sprintf("%x", sum) }
func save(name string, value any) {
	raw, err := json.MarshalIndent(value, "", "  ")
	must(err)
	must(os.WriteFile(name, append(raw, '\n'), 0644))
}
func main() {
	if len(os.Args) != 3 {
		panic("new evidence root and publication directory required")
	}
	root, out := os.Args[1], os.Args[2]
	paths := []string{}
	var total int64
	for _, part := range []string{"initializer", "trained", "audit", "native"} {
		must(filepath.WalkDir(filepath.Join(root, part), func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("ordinary evidence file required")
			}
			relative, err := filepath.Rel(root, name)
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			total += info.Size()
			paths = append(paths, relative)
			return nil
		}))
	}
	if total > 64<<20 {
		panic("registered 64MiB new evidence cap exceeded")
	}
	slices.Sort(paths)
	f, err := os.OpenFile(filepath.Join(out, "evidence.zip"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	must(err)
	archive := zip.NewWriter(f)
	inventory := map[string]any{}
	for _, relative := range paths {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		must(err)
		header := &zip.FileHeader{Name: filepath.ToSlash(relative), Method: zip.Deflate}
		header.SetModTime(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
		header.SetMode(0644)
		member, err := archive.CreateHeader(header)
		must(err)
		_, err = member.Write(raw)
		must(err)
		inventory[filepath.ToSlash(relative)] = map[string]any{"sha256": hash(raw), "bytes": len(raw)}
	}
	must(archive.Close())
	must(f.Close())
	save(filepath.Join(out, "evidence-manifest.json"), map[string]any{"schema": "gooo/record-shared-evidence/v1", "files": inventory, "raw_bytes": total, "raw_cap_bytes": 64 << 20, "source_bank": "publication/paired-field-intents-20261005/evidence.zip at 05db7d6e13e66259d2d116ef5c97d36216b04c4b", "scope": "New initializer/training/prediction/native evidence only; original complete source bank is reused and independently replayed."})
	groups := map[string]map[string]float64{}
	durations := map[string][]float64{}
	graphs := 0
	input, err := os.Open(filepath.Join(root, "native", "summaries.jsonl"))
	must(err)
	scan := bufio.NewScanner(input)
	scan.Buffer(make([]byte, 4096), 1<<20)
	for scan.Scan() {
		var row map[string]any
		must(json.Unmarshal(scan.Bytes(), &row))
		graphs++
		axis := "source"
		if row["wording_variant"].(float64) > 0 {
			axis = "new_wording"
		}
		for _, key := range []string{fmt.Sprintf("%s/b%d", row["profile"], int(row["candidate_budget"].(float64))), fmt.Sprintf("%s/%s/b%d", axis, row["profile"], int(row["candidate_budget"].(float64)))} {
			if groups[key] == nil {
				groups[key] = map[string]float64{}
			}
			g := groups[key]
			g["graphs"]++
			for _, field := range []string{"active_fields_passed", "active_fields_total", "guard_fields_passed", "guard_fields_total", "required_changed_fields_passed", "required_changed_fields_total", "runtime_named_passed", "runtime_named_total", "selection_fields_passed", "selection_fields_total", "model_calls", "candidate_attempts", "process_cpu_seconds"} {
				g[field] += row[field].(float64)
			}
			durations[key] = append(durations[key], row["wall_ms"].(float64))
		}
	}
	must(scan.Err())
	must(input.Close())
	if graphs != 360 {
		panic("all 360 native graphs required")
	}
	for key, values := range durations {
		slices.Sort(values)
		groups[key]["whole_command_wall_ms_median"] = (values[(len(values)-1)/2] + values[len(values)/2]) / 2
	}
	save(filepath.Join(out, "native-summary.json"), map[string]any{"schema": "gooo/record-shared-native-summary/v1", "graphs": graphs, "runs": 2 * graphs, "groups": groups, "scope": "Active, unchanged branch and required-changed fields separate; sequential controls include generation/build/startup. All raw observations independently recounted before packaging."})
	names := []string{}
	must(filepath.WalkDir(out, func(name string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(out, name)
		if err != nil {
			return err
		}
		if relative != "SHA256SUMS" {
			names = append(names, relative)
		}
		return nil
	}))
	slices.Sort(names)
	var sums strings.Builder
	for _, relative := range names {
		raw, err := os.ReadFile(filepath.Join(out, relative))
		must(err)
		fmt.Fprintf(&sums, "%s  %s\n", hash(raw), filepath.ToSlash(relative))
	}
	must(os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(sums.String()), 0644))
	fmt.Printf("Packaged %d files, %d raw bytes, %d native graphs.\n", len(paths), total, graphs)
}
