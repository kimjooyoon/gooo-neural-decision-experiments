package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func preparePaired(compiler, revision, output string, limit int) {
	if compiler == "" || len(revision) != 40 || output == "" || limit < 1 || limit > pairedTotal {
		panic("explicit bounded paired preparation required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		panic("fresh paired directory required")
	}
	check(os.MkdirAll(filepath.Join(output, "sources"), 0755))
	check(os.Mkdir(filepath.Join(output, "context-captures"), 0755))
	version := invoke(compiler, "version", "--build", "--json")
	var identity struct {
		Source string `json:"compiler_source_sha"`
		State  string `json:"source_status"`
		Go     string `json:"go_version"`
	}
	check(json.Unmarshal(version, &identity))
	if identity.Source != revision || identity.State != "CLEAN_VCS" || identity.Go != "go1.27.1" {
		panic("exact clean paired compiler required")
	}
	check(os.WriteFile(filepath.Join(output, "compiler-build.json"), version, 0644))
	rowsFile, err := os.OpenFile(filepath.Join(output, "rows.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer rowsFile.Close()
	featureFile, err := os.OpenFile(filepath.Join(output, "features-fp32.bin"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer featureFile.Close()
	encoder := json.NewEncoder(rowsFile)
	rows := make([]PairedRow, 0, limit)
	start := time.Now()
	for _, spec := range pairedPlan()[:limit] {
		source := goalFixture(spec.Family, orders[spec.Permutation], spec.Orientation, spec.Language, spec.Goal, spec.Style)
		row, blob := observePairedSource(compiler, output, spec, source)
		check(encoder.Encode(row))
		_, err = featureFile.Write(blob[:])
		check(err)
		rows = append(rows, row)
		if len(rows)%512 == 0 {
			fmt.Printf("Prepared and compiler-observed %d paired source views\n", len(rows))
		}
	}
	check(rowsFile.Close())
	check(featureFile.Close())
	initial := make([]byte, 74624)
	rng := rand.New(rand.NewSource(20261051))
	at := 0
	for _, tensor := range [4][2]int{{768 * 24, 768}, {24, 768}, {24 * 8, 24}, {8, 24}} {
		bound := 1 / math.Sqrt(float64(tensor[1]))
		for range tensor[0] {
			value := float32((2*rng.Float64() - 1) * bound)
			binary.LittleEndian.PutUint32(initial[at:], math.Float32bits(value))
			at += 4
		}
	}
	check(os.WriteFile(filepath.Join(output, "initial-fp32.bin"), initial, 0644))
	pins := map[string]any{}
	for _, name := range []string{"rows.jsonl", "features-fp32.bin", "initial-fp32.bin", "compiler-build.json"} {
		raw, err := os.ReadFile(filepath.Join(output, name))
		check(err)
		pins[name] = map[string]any{"bytes": len(raw), "sha256": hash(raw)}
	}
	counts := map[string]int{}
	for _, row := range rows {
		counts[row.Split]++
	}
	save(filepath.Join(output, "manifest.json"), map[string]any{"schema": "gooo/record-field-paired-training-inputs/v2", "complete": len(rows) == pairedTotal, "compiler_source": revision, "feature_version": jointdecision.RecordFieldFeatureVersion, "total": len(rows), "rows": counts, "files": pins, "initial_seed": 20261051, "preparation_ms": float64(time.Since(start).Nanoseconds()) / 1e6, "split_context_overlap": pairedOverlap(rows), "scope": "All eight per-field goal combinations over unchanged permitted expressions; base wording whole-family split plus separate previously unseen wording diagnostics on seen and held-out source families. Three fields/eight authored body-expression families; repeated source views. V1 diagnostic source families are reused benchmarks."})
}

func observePairedSource(compiler, output string, spec PairedSpec, source []byte) (PairedRow, [768 * 4]byte) {
	name := filepath.Join("sources", spec.ID()+".gooo.fixture")
	path := filepath.Join(output, name)
	check(os.WriteFile(path, source, 0644))
	capture := invoke(compiler, "body-context", "--activity", "Select", "--feature-version", jointdecision.RecordFieldFeatureVersion, path)
	check(os.WriteFile(filepath.Join(output, "context-captures", spec.ID()+".json"), capture, 0644))
	var exported struct {
		Source      string `json:"original_source_sha256"`
		Predictions int    `json:"model_predictions"`
		Tests       int    `json:"candidate_tests"`
		Context     struct {
			Text    string `json:"text"`
			Status  string `json:"status"`
			Version string `json:"feature_version"`
		} `json:"context"`
	}
	check(json.Unmarshal(capture, &exported))
	if exported.Source != "sha256:"+hash(source) || exported.Predictions != 0 || exported.Tests != 0 || exported.Context.Status != "ENCODED" || exported.Context.Version != jointdecision.RecordFieldFeatureVersion {
		panic("paired source-only context differs")
	}
	var features [768]float32
	check(jointdecision.FeaturesIntoRecordThree(exported.Context.Text, &features))
	var blob [768 * 4]byte
	for i, value := range features {
		binary.LittleEndian.PutUint32(blob[4*i:], math.Float32bits(value))
	}
	raw := invoke(compiler, "body-codegen", "--json", "--activity", "Select", path)
	var generated struct {
		Report struct {
			Assembly struct {
				Mask       uint16 `json:"selected_mask"`
				Passed     int    `json:"passed"`
				Total      int    `json:"total"`
				Fields     int    `json:"fields_passed"`
				FieldTotal int    `json:"fields_total"`
				Calls      int    `json:"model_calls"`
			} `json:"record_assembly"`
		} `json:"report"`
	}
	check(json.Unmarshal(raw, &generated))
	a := generated.Report.Assembly
	if a.Mask != goalMask(orders[spec.Permutation], spec.Orientation, spec.Goal) || a.Passed != 5 || a.Total != 5 || a.Fields != 15 || a.FieldTotal != 15 || a.Calls != 0 {
		panic("paired goal differs from actual complete finite construction")
	}
	row := Row{spec.ID(), spec.Family, spec.Language, spec.Split, name, hash(source), exported.Context.Text, hash([]byte(exported.Context.Text)), hash(blob[:]), a.Mask, a.Passed, a.Total, a.Fields, a.FieldTotal}
	return PairedRow{row, spec.Goal, spec.Style, spec.Permutation, spec.Orientation}, blob
}

func pairedOverlap(rows []PairedRow) map[string]map[string]int {
	contexts := map[string]map[string]bool{}
	features := map[string]map[string]bool{}
	sources := map[string]map[string]bool{}
	for _, r := range rows {
		if contexts[r.Split] == nil {
			contexts[r.Split] = map[string]bool{}
			features[r.Split] = map[string]bool{}
			sources[r.Split] = map[string]bool{}
		}
		contexts[r.Split][r.ContextSHA] = true
		features[r.Split][r.FeatureSHA] = true
		sources[r.Split][r.SourceSHA] = true
	}
	out := map[string]map[string]int{}
	for _, left := range []string{"train", "calibration"} {
		for _, right := range []string{"calibration", "test_source", "test_wording_seen", "test_wording_new"} {
			if left == right {
				continue
			}
			c, f, s := 0, 0, 0
			for key := range contexts[left] {
				if contexts[right][key] {
					c++
				}
			}
			for key := range features[left] {
				if features[right][key] {
					f++
				}
			}
			for key := range sources[left] {
				if sources[right][key] {
					s++
				}
			}
			out[left+"/"+right] = map[string]int{"exact_sources": s, "exact_contexts": c, "feature_vectors": f}
		}
	}
	return out
}
