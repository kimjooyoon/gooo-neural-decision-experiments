package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	if len(os.Args) != 3 {
		panic("usage archive-check zip root")
	}
	bundle, root := os.Args[1], os.Args[2]
	r, e := zip.OpenReader(bundle)
	must(e)
	defer r.Close()
	files := map[string]bool{}
	roots := map[string]bool{}
	var count, size int64
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !f.Mode().IsRegular() || filepath.Clean(f.Name) != f.Name || !filepath.IsLocal(f.Name) ||
			f.UncompressedSize64 > 16<<20 || len(files) >= 4096 {
			panic("invalid member")
		}
		prefix := strings.SplitN(f.Name, "/", 2)[0]
		if prefix != "paired" && prefix != "paired-final" && prefix != "inputs" && prefix != "expected" {
			panic("unexpected archive root")
		}
		roots[prefix] = true
		if files[f.Name] {
			panic("duplicate member")
		}
		files[f.Name] = true
		a, e := f.Open()
		must(e)
		h := sha256.New()
		n, e := io.Copy(h, a)
		must(e)
		must(a.Close())
		b, e := os.Open(filepath.Join(root, f.Name))
		must(e)
		g := sha256.New()
		m, e := io.Copy(g, b)
		must(e)
		must(b.Close())
		if n != m || fmt.Sprintf("%x", h.Sum(nil)) != fmt.Sprintf("%x", g.Sum(nil)) {
			panic("member differs " + f.Name)
		}
		count++
		size += n
		if size > 64<<20 {
			panic("archive bytes exceed scope")
		}
	}
	for dir := range roots {
		must(filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			name, e := filepath.Rel(root, path)
			if e != nil {
				return e
			}
			if !files[name] {
				return fmt.Errorf("unarchived file %s", name)
			}
			return nil
		}))
	}
	f, e := os.Open(bundle)
	must(e)
	h := sha256.New()
	_, e = io.Copy(h, f)
	must(e)
	must(f.Close())
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "files": count, "regular_bytes": size, "zip_sha256": fmt.Sprintf("%x", h.Sum(nil)), "model_predictions": 0, "native_executions": 0}))
}
