// record-field-curriculum prepares source-bound examples using the real Gooo
// compiler. Offline optimization consumes only Go features and observed masks.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

type Row struct {
	ID           string `json:"id"`
	Family       int    `json:"source_family"`
	Language     string `json:"language"`
	Split        string `json:"split"`
	SourceFile   string `json:"source_file"`
	SourceSHA    string `json:"source_sha256"`
	Context      string `json:"text"`
	ContextSHA   string `json:"context_sha256"`
	FeatureSHA   string `json:"feature_sha256"`
	Target       uint16 `json:"observed_complete_mask"`
	Passed       int    `json:"selection_cases_passed"`
	Total        int    `json:"selection_cases_total"`
	FieldsPassed int    `json:"selection_fields_passed"`
	FieldsTotal  int    `json:"selection_fields_total"`
}

func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func check(err error) {
	if err != nil {
		panic(err)
	}
}
func save(name string, value any) {
	raw, err := json.MarshalIndent(value, "", "  ")
	check(err)
	check(os.WriteFile(name, append(raw, '\n'), 0644))
}

func invoke(compiler string, args ...string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	raw, err := exec.CommandContext(ctx, compiler, args...).CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Gooo invocation failed: %v: %.2000s", err, raw))
	}
	return raw
}

func main() {
	output := flag.String("output", "", "fresh preparation directory")
	compiler := flag.String("compiler", "", "explicit clean Gooo executable")
	sourceRevision := flag.String("compiler-source", "", "exact compiler source revision")
	limit := flag.Int("limit", 768, "development cap; training requires all 768")
	prepared := flag.String("audit-prepared", "", "audit existing source/feature preparation")
	models := flag.String("models", "", "record-specific model variant directory")
	nativePrepared := flag.String("native-prepared", "", "execute existing source views on fresh runtime inputs")
	oldModel := flag.String("frozen-model", "", "explicit frozen ordinal model control")
	verifyNative := flag.String("verify-native", "", "recount captured native observations against current inputs")
	flag.Parse()
	if *verifyNative != "" {
		verifyNativeDirectory(*prepared, *verifyNative, *output)
		return
	}
	if *nativePrepared != "" {
		nativeStudy(*nativePrepared, *models, *compiler, *sourceRevision, *oldModel, *output)
		return
	}
	if *prepared != "" {
		audit(*prepared, *models, *output)
		return
	}
	if *output == "" || *compiler == "" || len(*sourceRevision) != 40 || *limit < 1 || *limit > 768 {
		panic("explicit output, compiler, source and bounded row count required")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		panic("fresh output directory required")
	}
	check(os.MkdirAll(filepath.Join(*output, "sources"), 0755))
	check(os.Mkdir(filepath.Join(*output, "context-captures"), 0755))
	version := invoke(*compiler, "version", "--build", "--json")
	check(os.WriteFile(filepath.Join(*output, "compiler-build.json"), version, 0644))
	var identity struct {
		Source string `json:"compiler_source_sha"`
		State  string `json:"source_status"`
		Go     string `json:"go_version"`
	}
	check(json.Unmarshal(version, &identity))
	if identity.Source != *sourceRevision || identity.State != "CLEAN_VCS" || identity.Go != "go1.27.1" {
		panic("clean exact Go 1.27.1 compiler required")
	}
	rowsFile, err := os.OpenFile(filepath.Join(*output, "rows.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer rowsFile.Close()
	featureFile, err := os.OpenFile(filepath.Join(*output, "features-fp32.bin"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	check(err)
	defer featureFile.Close()
	encoder := json.NewEncoder(rowsFile)
	rows := []Row{}
	started := time.Now()
	for family := 0; family < 8; family++ {
		for permutation, order := range orders {
			for mask := uint16(0); mask < 8; mask++ {
				for _, language := range []string{"ko", "en"} {
					if len(rows) >= *limit {
						break
					}
					id := fmt.Sprintf("f%d-p%d-m%d-%s", family, permutation, mask, language)
					name := filepath.Join("sources", id+".gooo.fixture")
					source := fixture(family, order, mask, language)
					check(os.WriteFile(filepath.Join(*output, name), source, 0644))
					contextRaw := invoke(*compiler, "body-context", "--activity", "Select", "--feature-version", jointdecision.RecordFieldFeatureVersion, filepath.Join(*output, name))
					check(os.WriteFile(filepath.Join(*output, "context-captures", id+".json"), contextRaw, 0644))
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
					check(json.Unmarshal(contextRaw, &exported))
					if exported.Predictions != 0 || exported.Tests != 0 || exported.Context.Status != "ENCODED" || exported.Context.Version != jointdecision.RecordFieldFeatureVersion || exported.Source != "sha256:"+hash(source) {
						panic("source-bound context export differs")
					}
					var features [jointdecision.ThreeFeatureDim]float32
					check(jointdecision.FeaturesIntoRecordThree(exported.Context.Text, &features))
					var blob [jointdecision.ThreeFeatureDim * 4]byte
					for i, value := range features {
						binary.LittleEndian.PutUint32(blob[4*i:], math.Float32bits(value))
					}
					_, err = featureFile.Write(blob[:])
					check(err)
					observedRaw := invoke(*compiler, "body-codegen", "--json", "--activity", "Select", filepath.Join(*output, name))
					var observed struct {
						Report struct {
							Assembly struct {
								Target       uint16 `json:"selected_mask"`
								Passed       int    `json:"passed"`
								Total        int    `json:"total"`
								FieldsPassed int    `json:"fields_passed"`
								FieldsTotal  int    `json:"fields_total"`
								Calls        int    `json:"model_calls"`
							} `json:"record_assembly"`
						} `json:"report"`
					}
					check(json.Unmarshal(observedRaw, &observed))
					a := observed.Report.Assembly
					if a.Target != mask || a.Passed != 5 || a.Total != 5 || a.FieldsPassed != 15 || a.FieldsTotal != 15 || a.Calls != 0 {
						panic("authored role target differs from actual finite compiler observation")
					}
					row := Row{id, family, language, split(family), name, hash(source), exported.Context.Text, hash([]byte(exported.Context.Text)), hash(blob[:]), a.Target, a.Passed, a.Total, a.FieldsPassed, a.FieldsTotal}
					check(encoder.Encode(row))
					rows = append(rows, row)
					if len(rows)%96 == 0 {
						fmt.Printf("Prepared and observed %d complete source views\n", len(rows))
					}
				}
			}
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
	check(os.WriteFile(filepath.Join(*output, "initial-fp32.bin"), initial, 0644))
	counts := map[string]int{}
	contexts := map[string]map[string]bool{}
	featuresSeen := map[string]map[string]bool{}
	for _, row := range rows {
		counts[row.Split]++
		if contexts[row.Split] == nil {
			contexts[row.Split] = map[string]bool{}
			featuresSeen[row.Split] = map[string]bool{}
		}
		contexts[row.Split][row.ContextSHA] = true
		featuresSeen[row.Split][row.FeatureSHA] = true
	}
	overlaps := map[string]map[string]int{}
	for _, pair := range [][2]string{{"train", "calibration"}, {"train", "test"}, {"calibration", "test"}} {
		c, f := 0, 0
		for value := range contexts[pair[0]] {
			if contexts[pair[1]][value] {
				c++
			}
		}
		for value := range featuresSeen[pair[0]] {
			if featuresSeen[pair[1]][value] {
				f++
			}
		}
		overlaps[pair[0]+"/"+pair[1]] = map[string]int{"exact_contexts": c, "feature_vectors": f}
	}
	pins := map[string]any{}
	for _, name := range []string{"rows.jsonl", "features-fp32.bin", "initial-fp32.bin", "compiler-build.json"} {
		raw, err := os.ReadFile(filepath.Join(*output, name))
		check(err)
		pins[name] = map[string]any{"sha256": hash(raw), "bytes": len(raw)}
	}
	save(filepath.Join(*output, "manifest.json"), map[string]any{"schema": "gooo/record-field-training-inputs/v1", "complete": len(rows) == 768, "compiler_source": *sourceRevision, "feature_version": jointdecision.RecordFieldFeatureVersion, "rows": counts, "total": len(rows), "split_context_overlap": overlaps, "files": pins, "initial_seed": 20261051, "preparation_ms": float64(time.Since(started).Nanoseconds()) / 1e6, "scope": "Eight authored source/expression families; six choice permutations and eight binary orientations; paired Korean/English views. Actual finite source observations provide masks. These are three field roles, not 768 independent semantic algorithms."})
}

func split(family int) string {
	if family < 4 {
		return "train"
	}
	if family < 6 {
		return "calibration"
	}
	return "test"
}
