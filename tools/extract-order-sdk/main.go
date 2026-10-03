// Extract the additive whole-candidate runtime without modifying old SDK files.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type entry struct {
	Origin    string `json:"origin_path"`
	Copied    string `json:"copied_path"`
	OriginSHA string `json:"origin_sha256"`
	SHA       string `json:"sha256"`
	Bytes     int    `json:"bytes"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func hash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

func main() {
	out := flag.String("output", "", "fresh extraction directory")
	flag.Parse()
	if *out == "" || flag.NArg() != 0 {
		panic("output required")
	}
	if _, err := os.Stat(*out); !os.IsNotExist(err) {
		panic("output must be fresh")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	must(err)
	if len(dirty) != 0 {
		panic("clean committed source required")
	}
	paths := []string{
		"orderfacts/candidates.go", "orderfacts/candidates_test.go", "orderfacts/facts.go", "orderfacts/facts_test.go",
		"orderjudge/artifact.go", "orderjudge/features.go", "orderjudge/fit.go", "orderjudge/model.go",
		"orderjudge/model_test.go", "orderjudge/search.go", "orderjudge/search_test.go",
		"examples/order-replay/main.go",
	}
	replacements := strings.NewReplacer(
		"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision", "github.com/kimjooyoon/gooo-decision-runtime",
		"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson", "github.com/kimjooyoon/gooo-decision-runtime/internal/strictjson",
		"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/", "github.com/kimjooyoon/gooo-decision-runtime/",
	)
	var entries []entry
	for _, path := range paths {
		origin := "internal/" + path
		if strings.HasPrefix(path, "examples/") {
			origin = "cmd/order-judge-replay/main.go"
		}
		raw, err := exec.Command("git", "show", strings.TrimSpace(string(head))+":"+origin).Output()
		must(err)
		copied, err := format.Source([]byte(replacements.Replace(string(raw))))
		must(err)
		name := filepath.Join(*out, path)
		must(os.MkdirAll(filepath.Dir(name), 0755))
		must(os.WriteFile(name, copied, 0644))
		entries = append(entries, entry{origin, path, hash(raw), hash(copied), len(copied)})
	}
	manifest := map[string]any{"schema": "gooo/order-judge-sdk-source/v1", "origin_repository": "github.com/kimjooyoon/gooo-neural-decision-experiments",
		"origin_revision": strings.TrimSpace(string(head)), "files": entries, "transformation": "fixed_module_import_relocation_and_gofmt",
		"scope": "Additive orderfacts/orderjudge packages, unit tests and complete frozen-evidence replay. Existing SDK sources, manifests and models are unchanged."}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(*out, "source-provenance-order-judge-v1.json"), append(raw, '\n'), 0644))
	fmt.Printf("extracted %d source/test/example files\n", len(entries))
}
