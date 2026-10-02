package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

type filePin struct {
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
	Lines int    `json:"lines"`
}

func readJSON(name string, value any) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > lineCap {
		return nil, errors.New("bounded regular metadata required")
	}
	raw, err := os.ReadFile(name)
	if err != nil || privateText.Match(raw) || decision.RejectDuplicateJSONKeys(raw) != nil {
		return nil, errors.New("public nonduplicate metadata required")
	}
	return raw, json.Unmarshal(raw, value)
}

func audit(dir, output string) error {
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh audit report required")
	}
	var manifest struct {
		Schema   string             `json:"schema"`
		Status   string             `json:"status"`
		Files    map[string]filePin `json:"files"`
		Calls    int                `json:"actual_native_export_calls"`
		Views    int                `json:"function_views"`
		Protocol string             `json:"protocol_sha256"`
		Runner   string             `json:"runner_revision"`
		Native   string             `json:"native_main_revision"`
		Binary   string             `json:"native_binary_sha256"`
		SDK      string             `json:"sdk_version"`
		RawBytes int64              `json:"raw_bytes"`
	}
	manifestRaw, err := readJSON(filepath.Join(dir, "manifest.json"), &manifest)
	if err != nil || manifest.Schema != "gooo/three-composition-curriculum/v1" ||
		manifest.Status != "SOURCE_BOUND_EXPORTED_PENDING_INDEPENDENT_AUDIT" || manifest.Calls != 3072 ||
		manifest.Views != 3072 || manifest.Protocol != protocolSHA || len(manifest.Files) != 3 ||
		manifest.SDK != sdkVersion || len(manifest.Runner) != 40 || len(manifest.Native) != 40 || len(manifest.Binary) != 64 {
		return errors.New("complete fixed source manifest required")
	}
	var preexecution struct {
		Schema   string `json:"schema"`
		Runner   string `json:"runner_revision"`
		Native   string `json:"native_main_revision"`
		Binary   string `json:"native_binary_sha256"`
		SDK      string `json:"sdk_version"`
		Protocol string `json:"protocol_sha256"`
		Calls    int    `json:"planned_native_export_calls"`
	}
	if _, err := readJSON(filepath.Join(dir, "preexecution.json"), &preexecution); err != nil ||
		preexecution.Schema != "gooo/three-composition-collection-preexecution/v1" || preexecution.Runner != manifest.Runner ||
		preexecution.Native != manifest.Native || preexecution.Binary != manifest.Binary || preexecution.SDK != manifest.SDK ||
		preexecution.Protocol != protocolSHA || preexecution.Calls != 3072 {
		return errors.New("preexecution source/binary/SDK/protocol pins differ")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 6 {
		return errors.New("closed complete collection directory required")
	}
	allowed := map[string]bool{"fixtures.jsonl": true, "exports.jsonl": true, "dataset.jsonl": true,
		"preexecution.json": true, "manifest.json": true, "collection-attempt.json": true}
	for _, entry := range entries {
		if !allowed[entry.Name()] || entry.Type() != 0 {
			return errors.New("unexpected collection member or file type")
		}
	}
	var attempt struct {
		Status string             `json:"status"`
		Calls  int                `json:"actual_native_export_calls"`
		Views  int                `json:"retained_function_views"`
		Files  map[string]filePin `json:"files"`
	}
	if _, err := readJSON(filepath.Join(dir, "collection-attempt.json"), &attempt); err != nil ||
		attempt.Status != "COMPLETE_FIXED_COLLECTION" || attempt.Calls != 3072 || attempt.Views != 3072 ||
		!reflect.DeepEqual(attempt.Files, manifest.Files) {
		return errors.New("completed collection-attempt pins required")
	}
	names := [3]string{"fixtures.jsonl", "exports.jsonl", "dataset.jsonl"}
	var scanners [3]*bufio.Scanner
	var total int64
	for i, name := range names {
		pin := manifest.Files[name]
		if pin.Lines != 3072 || pin.Bytes <= 0 || pin.Bytes > rawCap {
			return errors.New("closed complete journal pins required")
		}
		fileInfo, err := os.Lstat(filepath.Join(dir, name))
		if err != nil || !fileInfo.Mode().IsRegular() {
			return errors.New("regular nonsymlink journal required")
		}
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() != pin.Bytes {
			return errors.New("journal type/bytes differ")
		}
		h := sha256.New()
		if _, err := io.Copy(h, io.LimitReader(f, pin.Bytes+1)); err != nil || hex.EncodeToString(h.Sum(nil)) != pin.SHA {
			return errors.New("journal hash differs")
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		scanners[i] = bufio.NewScanner(f)
		scanners[i].Buffer(make([]byte, 32768), lineCap)
		total += pin.Bytes
	}
	if total > rawCap || total != manifest.RawBytes {
		return errors.New("combined raw journal cap exceeded")
	}
	views, ties := 0, 0
	for _, family := range threecompositionstudy.Families {
		for config := range 24 {
			for goal := range 8 {
				for _, language := range [2]string{"en", "ko"} {
					for _, scanner := range scanners {
						if !scanner.Scan() || privateText.Match(scanner.Bytes()) || decision.RejectDuplicateJSONKeys(scanner.Bytes()) != nil {
							return errors.New("complete public ordered journals required")
						}
					}
					id := fmt.Sprintf("%s-c%02d-goal%d-%s", family, config, goal, language)
					doc, source, target, err := fixture(family, config, goal, language)
					if err != nil {
						return err
					}
					goldFixture, _ := json.Marshal(map[string]any{"id": id, "source": string(source), "document": doc,
						"finite_target": target, "source_sha256": hash(source)})
					if !bytes.Equal(scanners[0].Bytes(), goldFixture) {
						return errors.New("frozen concrete fixture differs from independently rebuilt oracle/source")
					}
					var capture struct {
						ID   string `json:"id"`
						Raw  []byte `json:"native_receipt"`
						Wall int64  `json:"wall_ns"`
						OK   bool   `json:"child_succeeded"`
					}
					if json.Unmarshal(scanners[1].Bytes(), &capture) != nil || capture.ID != id || !capture.OK || capture.Wall < 0 {
						return errors.New("actual successful native capture required")
					}
					value, text, err := inspect(capture.Raw, doc, source)
					if err != nil {
						return err
					}
					goldRow, _ := json.Marshal(datasetRow(id, family, config, goal, language, source, capture.Raw, value, text, target))
					if !bytes.Equal(scanners[2].Bytes(), goldRow) {
						return errors.New("frozen complete three-input dataset differs from raw native capture")
					}
					views++
					if len(target.BestMasks) > 1 {
						ties++
					}
				}
			}
		}
	}
	for _, scanner := range scanners {
		if scanner.Scan() || scanner.Err() != nil {
			return errors.New("extra or invalid journal suffix")
		}
	}
	if views != 3072 || ties != 878 {
		return errors.New("fixed independently audited denominators differ")
	}
	return save(output, map[string]any{"schema": "gooo/three-composition-curriculum-audit/v1", "status": "PASS",
		"manifest_sha256": hash(manifestRaw), "files": manifest.Files, "raw_bytes": total, "function_views": views,
		"bilingual_contract_groups": 1536, "source_input_rows": 9216, "tied_function_views": ties,
		"recorded_native_export_calls": 3072, "independent_typed_oracle_comparisons": 393216,
		"new_native_calls": 0, "new_model_predictions": 0, "new_optimizer_updates": 0,
		"scope": "Offline full-byte/oracle/source-context reconciliation of ordered actual native exports; no learning or generative-quality claim."})
}
