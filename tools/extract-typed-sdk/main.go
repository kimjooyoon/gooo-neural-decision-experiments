// extract-typed-sdk copies a fixed source/test inventory with recorded import
// relocation. It never copies weights, raw studies, environments or caches.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type entry struct {
	Origin      string `json:"origin_path"`
	Copied      string `json:"copied_path"`
	OriginSHA   string `json:"origin_sha256"`
	SHA         string `json:"sha256"`
	OriginBytes int    `json:"origin_bytes"`
	Bytes       int    `json:"bytes"`
	Transform   string `json:"transformation"`
}

func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func inventory() map[string]string {
	return map[string]string{
		"model.go": "internal/decision/model.go", "bridge.go": "internal/decision/bridge.go", "ir.go": "internal/decision/ir.go", "LICENSE": "LICENSE",
		"model_test.go": "internal/decision/model_test.go", "ir_test.go": "internal/decision/ir_test.go", "path_model_test.go": "internal/decision/path_model_test.go", "path_features_test.go": "internal/decision/path_features_test.go",
		"bodyplan/bodyplan.go": "internal/bodyplan/bodyplan.go", "bodyplan/bodyplan_test.go": "internal/bodyplan/bodyplan_test.go",
		"pathplan/pathplan.go": "internal/pathplan/pathplan.go", "pathplan/search.go": "internal/pathplan/search.go", "pathplan/pathplan_test.go": "internal/pathplan/pathplan_test.go", "pathplan/search_test.go": "internal/pathplan/search_test.go",
		"pathplan/prepared.go": "internal/pathplan/prepared.go", "pathplan/prepared_test.go": "internal/pathplan/prepared_test.go",
		"pathplan/session.go": "internal/pathplan/session.go", "pathplan/session_test.go": "internal/pathplan/session_test.go",
		"internal/strictjson/decode.go": "internal/strictjson/decode.go", "internal/strictjson/decode_test.go": "internal/strictjson/decode_test.go",
	}
}
func run(output string) error {
	if output == "" {
		return errors.New("SDK output required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return err
	}
	paths := []string{"diff", "--quiet", "HEAD", "--", "internal/decision", "internal/bodyplan", "internal/pathplan", "internal/strictjson", "LICENSE"}
	if err := exec.Command("git", paths...).Run(); err != nil {
		return errors.New("origin source must be committed before extraction")
	}
	contents := map[string][]byte{}
	var files []entry
	for copied, origin := range inventory() {
		info, err := os.Lstat(origin)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 2<<20 {
			return errors.New("bounded regular origin required")
		}
		raw, err := os.ReadFile(origin)
		if err != nil {
			return err
		}
		transformed := string(raw)
		for old, newName := range map[string]string{
			"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision":   "github.com/kimjooyoon/gooo-decision-runtime",
			"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan":   "github.com/kimjooyoon/gooo-decision-runtime/bodyplan",
			"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan":   "github.com/kimjooyoon/gooo-decision-runtime/pathplan",
			"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson": "github.com/kimjooyoon/gooo-decision-runtime/internal/strictjson",
		} {
			transformed = strings.ReplaceAll(transformed, old, newName)
		}
		data := []byte(transformed)
		if strings.HasSuffix(copied, ".go") {
			data, err = format.Source(data)
			if err != nil {
				return err
			}
		}
		method := "exact"
		if string(data) != string(raw) {
			method = "fixed_module_import_relocation_and_gofmt"
		}
		contents[copied] = data
		files = append(files, entry{origin, copied, hash(raw), hash(data), len(raw), len(data), method})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Copied < files[j].Copied })
	for copied, raw := range contents {
		path := filepath.Join(output, copied)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, raw, 0644); err != nil {
			return err
		}
	}
	manifest := map[string]any{"schema": "gooo/decision-runtime-source-provenance/v2", "module": "github.com/kimjooyoon/gooo-decision-runtime", "origin_repository": "github.com/kimjooyoon/gooo-neural-decision-experiments", "origin_revision": strings.TrimSpace(string(head)), "files": files, "scope": "Fixed source and test extraction; import relocation is recorded with separate origin and copied byte digests. No weights or raw experiment directories are copied."}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "source-provenance.json"), append(raw, '\n'), 0644)
}
func main() {
	output := flag.String("output", "", "SDK checkout to update")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments")
		os.Exit(2)
	}
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, "extract-typed-sdk:", err)
		os.Exit(1)
	}
	fmt.Printf("{\"status\":\"EXTRACTED\",\"source_and_test_files\":%d,\"model_predictions\":0}\n", len(inventory()))
}
