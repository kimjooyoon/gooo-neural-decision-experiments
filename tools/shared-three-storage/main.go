// shared-three-storage hashes retained evidence before and after offline training.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

type phase struct {
	Name    string `json:"name"`
	Files   int    `json:"files"`
	Bytes   int64  `json:"bytes"`
	TreeSHA string `json:"tree_sha256"`
}
type report struct {
	Schema           string  `json:"schema"`
	Status           string  `json:"status"`
	Phases           []phase `json:"phases"`
	LegacyBytes      int64   `json:"legacy_bytes"`
	NewCap           int64   `json:"new_phase_cap_bytes"`
	WholeCap         int64   `json:"amended_whole_cap_bytes"`
	OptimizerUpdates int     `json:"optimizer_updates"`
	Scope            string  `json:"scope"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func scan(root, name string) phase {
	r := phase{Name: name}
	h := sha256.New()
	base := filepath.Join(root, "runs", name)
	must(filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("evidence symlink")
		}
		if d.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular evidence")
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		fileHash := sha256.New()
		n, err := io.Copy(fileHash, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if n != info.Size() {
			return fmt.Errorf("changing evidence")
		}
		fmt.Fprintf(h, "%q %d %x\n", filepath.ToSlash(rel), n, fileHash.Sum(nil))
		r.Files++
		r.Bytes += n
		return nil
	}))
	r.TreeSHA = fmt.Sprintf("%x", h.Sum(nil))
	return r
}
func main() {
	root := flag.String("root", ".", "research repository")
	output := flag.String("output", "", "fresh output JSON")
	baseline := flag.String("baseline", "", "compare every retained phase to original preflight")
	flag.Parse()
	if *output == "" || flag.NArg() != 0 {
		panic("--output required")
	}
	r := report{Schema: "gooo/shared-three-storage-preflight/v1", Status: "PASS", NewCap: 64 << 20, WholeCap: 2 << 30,
		Scope: "All original own-three phases retained byte-for-byte; shared experiment phases excluded from legacy baseline. Original 768-MiB failed outcome unchanged; existing separate 2-GiB amendment applies. Zero optimizer/model/native calls."}
	entries, err := os.ReadDir(filepath.Join(*root, "runs"))
	must(err)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "own-three-") && !strings.HasPrefix(e.Name(), "own-three-shared-") {
			if !e.IsDir() {
				panic("phase is not directory")
			}
			p := scan(*root, e.Name())
			r.Phases = append(r.Phases, p)
			r.LegacyBytes += p.Bytes
		}
	}
	sort.Slice(r.Phases, func(i, j int) bool { return r.Phases[i].Name < r.Phases[j].Name })
	if len(r.Phases) == 0 || r.LegacyBytes+r.NewCap > r.WholeCap {
		panic("no room within amended bound")
	}
	if *baseline != "" {
		raw, err := os.ReadFile(*baseline)
		must(err)
		var old report
		must(json.Unmarshal(raw, &old))
		if !reflect.DeepEqual(r, old) {
			panic("legacy evidence or budget changed")
		}
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	must(err)
	f, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(err)
	_, err = f.Write(append(raw, '\n'))
	must(err)
	must(f.Sync())
	must(f.Close())
	fmt.Printf("PASS: %d original phases, %d retained bytes, %d new-byte cap\n", len(r.Phases), r.LegacyBytes, r.NewCap)
}
