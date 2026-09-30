package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"
)

const (
	datasetSchema        = "gooo/synthetic-operation-choice-dataset/v1"
	manifestSchema       = "gooo/synthetic-operation-choice-manifest/v1"
	contractSchema       = "gooo/tiny-ir-decision-model-contract/v1"
	rowsPerTemplate      = 8
	templatesPerGroup    = 16
	trainTemplates       = 12
	calibrationTemplates = 2
	testTemplates        = 2
)

var operationOrder = []string{
	"add",
	"subtract",
	"multiply",
	"less_than",
	"less_equal",
	"equal",
	"and",
	"or",
}

var languages = []string{"en", "ko"}

type variablePair struct {
	Left  string
	Right string
}

type operation struct {
	Label   string
	Boolean bool
	English [4]string
	Korean  [4]string
}

type promptTemplate struct {
	ID       string
	Language string
	Label    string
	Ordinal  int
	Text     string
}

type row struct {
	ID              string `json:"id"`
	TemplateID      string `json:"template_id"`
	ConfigurationID string `json:"configuration_id"`
	Language        string `json:"language"`
	Split           string `json:"split"`
	Text            string `json:"text"`
	Label           string `json:"label"`
}

type splitCounts struct {
	Total       int `json:"total"`
	Train       int `json:"train"`
	Calibration int `json:"calibration"`
	Test        int `json:"test"`
}

type groupCounts struct {
	Label     string      `json:"label"`
	Language  string      `json:"language"`
	Templates int         `json:"templates"`
	Rows      splitCounts `json:"rows"`
}

type manifest struct {
	Schema                   string                 `json:"schema"`
	DatasetSchema            string                 `json:"dataset_schema"`
	GeneratorVersion         string                 `json:"generator_version"`
	Seed                     int64                  `json:"seed"`
	ModelContractSHA256      string                 `json:"model_contract_sha256"`
	GeneratorSourceSHA256    string                 `json:"generator_source_sha256,omitempty"`
	DatasetSHA256            string                 `json:"dataset_sha256"`
	SplitSHA256              map[string]string      `json:"split_sha256"`
	Rows                     splitCounts            `json:"rows"`
	RowsByLabel              map[string]splitCounts `json:"rows_by_label"`
	RowsByLanguage           map[string]splitCounts `json:"rows_by_language"`
	RowsByLabelAndLanguage   []groupCounts          `json:"rows_by_label_and_language"`
	OperationLabels          []string               `json:"operation_labels"`
	SplitPolicy              splitPolicy            `json:"split_policy"`
	VariableConfigurations   int                    `json:"variable_configurations_per_template"`
	SyntheticProvenance      syntheticProvenance    `json:"synthetic_provenance"`
	PromptInputPolicy        string                 `json:"prompt_input_policy"`
	FeatureInputField        string                 `json:"feature_input_field"`
	NoGoldLabelInPromptCheck bool                   `json:"no_gold_label_in_prompt_check"`
}

type splitPolicy struct {
	Unit                   string `json:"unit"`
	TemplatesPerLabelLang  int    `json:"templates_per_label_language"`
	TrainTemplates         int    `json:"train_templates_per_label_language"`
	CalibrationTemplates   int    `json:"calibration_templates_per_label_language"`
	TestTemplates          int    `json:"test_templates_per_label_language"`
	RowsPerTemplate        int    `json:"rows_per_template"`
	SameTemplateAcrossSets bool   `json:"same_template_crosses_splits"`
	ShuffledWithSeed       bool   `json:"template_assignment_shuffled_with_seed"`
}

type syntheticProvenance struct {
	Kind           string `json:"kind"`
	SourceMaterial string `json:"source_material"`
	SensitiveData  string `json:"sensitive_data"`
	Values         string `json:"values"`
	IntendedUse    string `json:"intended_use"`
}

type contract struct {
	Schema       string   `json:"schema"`
	FeatureDim   int      `json:"feature_dim"`
	HiddenDim    int      `json:"hidden_dim"`
	LabelCount   int      `json:"label_count"`
	Labels       []string `json:"labels"`
	Input        string   `json:"input"`
	MaxBytes     int      `json:"max_bytes"`
	Network      string   `json:"network"`
	WeightLayout string   `json:"weight_layout"`
	Quantization string   `json:"quantization"`
	Runtime      string   `json:"runtime"`
	Training     string   `json:"training"`
	Scope        string   `json:"scope"`
}

var operations = []operation{
	{
		Label: "add",
		English: [4]string{
			"Compute the sum of the two values.",
			"Return the total value.",
			"Combine both values by summation.",
			"Produce the result of adding the values together.",
		},
		Korean: [4]string{
			"두 값의 합을 구하라.",
			"두 값이 합쳐진 전체를 반환하라.",
			"두 수를 더한 결과를 산출하라.",
			"두 값을 합산한 결과를 내라.",
		},
	},
	{
		Label: "subtract",
		English: [4]string{
			"Find the first value after taking away the second.",
			"Return the ordered difference.",
			"Remove the second value from the first.",
			"Compute what remains when the second is removed from the first.",
		},
		Korean: [4]string{
			"첫 값에서 다음 값을 뺀 결과를 구하라.",
			"입력 순서를 반영한 차이를 반환하라.",
			"첫 번째 수에서 두 번째 수를 제거한 값을 내라.",
			"두 번째 값을 첫 값에서 제외하고 남은 값을 계산하라.",
		},
	},
	{
		Label: "multiply",
		English: [4]string{
			"Return the product of the values.",
			"Scale the first value by the second.",
			"Compute their repeated product.",
			"Use the second as a repeated scale factor for the first.",
		},
		Korean: [4]string{
			"두 값의 곱을 반환하라.",
			"첫 값을 다음 값의 배수만큼 늘린 결과를 구하라.",
			"두 수를 곱한 결과를 계산하라.",
			"두 입력의 곱셈 결과를 산출하라.",
		},
	},
	{
		Label: "less_than",
		English: [4]string{
			"Return whether the first value is strictly smaller.",
			"Check if the first lies below the second.",
			"Decide if the first has the lower numeric value.",
			"Mark true when the first precedes the second in numeric order.",
		},
		Korean: [4]string{
			"첫 값이 다음 값보다 엄격히 작은지 답하라.",
			"첫 입력이 두 번째 입력 아래에 있는지 확인하라.",
			"첫 값의 크기가 더 낮은지 판정하라.",
			"수의 순서에서 첫 값이 앞서는지 표시하라.",
		},
	},
	{
		Label: "less_equal",
		English: [4]string{
			"Return whether the first value does not exceed the second.",
			"Check if the first is at most the second.",
			"Decide whether the first is no greater than the second.",
			"Mark true when the first sits at or below the second.",
		},
		Korean: [4]string{
			"첫 값이 다음 값보다 크지 않은지 답하라.",
			"첫 값이 두 번째 값을 넘지 않는지 확인하라.",
			"첫 값이 최대한 두 번째 값과 같은 범위인지 판정하라.",
			"첫 입력이 두 번째 입력 이하인지 표시하라.",
		},
	},
	{
		Label: "equal",
		English: [4]string{
			"Return whether the two values match exactly.",
			"Check if both values are identical.",
			"Decide whether their numeric values coincide.",
			"Mark true when both inputs share one value.",
		},
		Korean: [4]string{
			"두 값이 정확히 같은지 답하라.",
			"두 입력이 서로 일치하는지 확인하라.",
			"두 수의 값이 동일한지 판정하라.",
			"첫 값과 다음 값이 같은 값을 갖는지 표시하라.",
		},
	},
	{
		Label:   "and",
		Boolean: true,
		English: [4]string{
			"Return true only when each boolean input holds.",
			"Mark true if both flags are true.",
			"The result is true when every condition holds.",
			"Return false when at least one input is false.",
		},
		Korean: [4]string{
			"각 불리언 입력이 모두 참일 때만 참을 반환하라.",
			"두 플래그가 참이면 참으로 표시하라.",
			"모든 조건이 성립할 때 참을 내라.",
			"입력 중 하나라도 거짓이면 거짓을 반환하라.",
		},
	},
	{
		Label:   "or",
		Boolean: true,
		English: [4]string{
			"Return true if at least one boolean input holds.",
			"Mark false only when each flag is false.",
			"The result is true when any condition holds.",
			"Return true unless both inputs are false.",
		},
		Korean: [4]string{
			"불리언 입력 중 하나 이상이 참이면 참을 반환하라.",
			"두 플래그가 모두 거짓인 경우에만 거짓으로 표시하라.",
			"어떤 조건이든 성립하면 참을 내라.",
			"두 입력이 모두 거짓이 아니라면 참을 반환하라.",
		},
	},
}

var numericPairs = []variablePair{
	{"a", "b"},
	{"x", "y"},
	{"left", "right"},
	{"first", "second"},
	{"m", "n"},
	{"p", "q"},
	{"u", "v"},
	{"lhs", "rhs"},
}

var booleanPairs = []variablePair{
	{"flag_a", "flag_b"},
	{"is_open", "is_valid"},
	{"enabled", "ready"},
	{"condition_x", "condition_y"},
	{"check_one", "check_two"},
	{"feature_p", "feature_q"},
	{"signal_a", "signal_b"},
	{"state_left", "state_right"},
}

var englishPrefixes = [4]string{
	"Operands: %s, %s. ",
	"Read the ordered pair (%s, %s). ",
	"Use %s as the first input; use %s as the second. ",
	"Inputs are left=%s; right=%s. ",
}

var englishBooleanPrefixes = [4]string{
	"Boolean inputs: %s, %s. ",
	"Read the ordered flags (%s, %s). ",
	"Use %s as the first flag; use %s as the second. ",
	"Conditions are first=%s; next=%s. ",
}

var koreanPrefixes = [4]string{
	"피연산자는 %s, %s이다. ",
	"순서가 있는 입력 (%s, %s)을 읽어라. ",
	"%s를 첫 입력으로, %s를 다음 입력으로 사용하라. ",
	"입력 순서는 첫 값=%s, 다음 값=%s이다. ",
}

var koreanBooleanPrefixes = [4]string{
	"불리언 입력은 %s, %s이다. ",
	"순서가 있는 플래그 (%s, %s)를 읽어라. ",
	"%s를 첫 플래그로, %s를 다음 플래그로 사용하라. ",
	"조건 순서는 첫 값=%s, 다음 값=%s이다. ",
}

func main() {
	seed := flag.Int64("seed", 20260930, "seed used to assign whole prompt templates to splits")
	outDir := flag.String("out-dir", "data/synthetic-ops-v1", "directory for dataset.jsonl and manifest.json")
	contractPath := flag.String("contract", "model-contract.json", "model contract to validate and hash")
	flag.Parse()

	if err := run(*seed, *outDir, *contractPath); err != nil {
		fmt.Fprintln(os.Stderr, "dataset generation failed:", err)
		os.Exit(1)
	}
}

func run(seed int64, outDir, contractPath string) error {
	if err := validateDefinitions(); err != nil {
		return err
	}
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		return fmt.Errorf("read model contract %q: %w", contractPath, err)
	}
	if err := validateContract(contractBytes); err != nil {
		return err
	}

	rows, groups, err := generateRows(seed)
	if err != nil {
		return err
	}
	dataBytes, splitBytes, err := encodeRows(rows)
	if err != nil {
		return err
	}
	if err := validateRows(rows, groups); err != nil {
		return err
	}

	byLabel := make(map[string]splitCounts, len(operationOrder))
	byLanguage := make(map[string]splitCounts, len(languages))
	byLabelAndLanguage := make([]groupCounts, 0, len(operationOrder)*len(languages))
	for _, label := range operationOrder {
		byLabel[label] = countRows(rows, func(r row) bool { return r.Label == label })
		for _, lang := range languages {
			c := countRows(rows, func(r row) bool { return r.Label == label && r.Language == lang })
			byLabelAndLanguage = append(byLabelAndLanguage, groupCounts{
				Label: label, Language: lang, Templates: templatesPerGroup, Rows: c,
			})
		}
	}
	for _, lang := range languages {
		byLanguage[lang] = countRows(rows, func(r row) bool { return r.Language == lang })
	}

	sourceHash := generatorSourceHash()
	manifestValue := manifest{
		Schema:                manifestSchema,
		DatasetSchema:         datasetSchema,
		GeneratorVersion:      "1.0.0",
		Seed:                  seed,
		ModelContractSHA256:   hashHex(contractBytes),
		GeneratorSourceSHA256: sourceHash,
		DatasetSHA256:         hashHex(dataBytes),
		SplitSHA256: map[string]string{
			"train":       hashHex(splitBytes["train"]),
			"calibration": hashHex(splitBytes["calibration"]),
			"test":        hashHex(splitBytes["test"]),
		},
		Rows:                   countRows(rows, func(row) bool { return true }),
		RowsByLabel:            byLabel,
		RowsByLanguage:         byLanguage,
		RowsByLabelAndLanguage: byLabelAndLanguage,
		OperationLabels:        append([]string(nil), operationOrder...),
		SplitPolicy: splitPolicy{
			Unit:                   "whole text template; all eight variable configurations for a template stay in one split",
			TemplatesPerLabelLang:  templatesPerGroup,
			TrainTemplates:         trainTemplates,
			CalibrationTemplates:   calibrationTemplates,
			TestTemplates:          testTemplates,
			RowsPerTemplate:        rowsPerTemplate,
			SameTemplateAcrossSets: false,
			ShuffledWithSeed:       true,
		},
		VariableConfigurations: rowsPerTemplate,
		SyntheticProvenance: syntheticProvenance{
			Kind:           "synthetic, hand-authored English and Korean operation instructions, expanded with generated variable names",
			SourceMaterial: "No repository source, user content, model output, external dataset, or private text is used.",
			SensitiveData:  "No personal identifiers, customer content, credentials, local paths, or real project identifiers are present.",
			Values:         "Only generated variable names appear; no expected numeric or boolean result is included in the prompt.",
			IntendedUse:    "Finite eight-class typed operation choice for an IR hole; not arbitrary source-code generation.",
		},
		PromptInputPolicy:        "The text field contains only a synthetic natural-language instruction and generated variable names. Gold labels exist only in the separate label target field; labels are not embedded in prompt text.",
		FeatureInputField:        "text",
		NoGoldLabelInPromptCheck: true,
	}
	manifestBytes, err := json.MarshalIndent(manifestValue, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')

	if err := writeNewOrIdentical(filepath.Join(outDir, "dataset.jsonl"), dataBytes); err != nil {
		return err
	}
	if err := writeNewOrIdentical(filepath.Join(outDir, "manifest.json"), manifestBytes); err != nil {
		return err
	}
	fmt.Printf("wrote %d synthetic rows (%d train, %d calibration, %d test) to %s\n",
		len(rows), manifestValue.Rows.Train, manifestValue.Rows.Calibration, manifestValue.Rows.Test, outDir)
	fmt.Printf("dataset sha256: %s\n", manifestValue.DatasetSHA256)
	return nil
}

func validateDefinitions() error {
	if len(operationOrder) != 8 || len(operations) != len(operationOrder) {
		return fmt.Errorf("expected exactly eight operation definitions")
	}
	for i, op := range operations {
		if op.Label != operationOrder[i] {
			return fmt.Errorf("operation order mismatch at %d: got %q, want %q", i, op.Label, operationOrder[i])
		}
	}
	if len(numericPairs) != rowsPerTemplate || len(booleanPairs) != rowsPerTemplate {
		return fmt.Errorf("each variable-pair table must have exactly %d entries", rowsPerTemplate)
	}
	if trainTemplates+calibrationTemplates+testTemplates != templatesPerGroup {
		return errors.New("split template counts do not sum to templates per group")
	}
	return nil
}

func validateContract(raw []byte) error {
	var c contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return fmt.Errorf("decode model contract: %w", err)
	}
	if c.Schema != contractSchema {
		return fmt.Errorf("unsupported model contract schema %q", c.Schema)
	}
	if c.LabelCount != len(operationOrder) || !equalStrings(c.Labels, operationOrder) {
		return errors.New("model contract operation labels do not match the synthetic dataset generator")
	}
	return nil
}

func generateRows(seed int64) ([]row, []promptTemplate, error) {
	rows := make([]row, 0, 2048)
	templates := make([]promptTemplate, 0, len(operations)*len(languages)*templatesPerGroup)
	for _, op := range operations {
		for _, lang := range languages {
			opTemplates := buildTemplates(op, lang)
			if len(opTemplates) != templatesPerGroup {
				return nil, nil, fmt.Errorf("%s/%s has %d templates, expected %d", op.Label, lang, len(opTemplates), templatesPerGroup)
			}
			assignment := make([]int, templatesPerGroup)
			for i := range assignment {
				assignment[i] = i
			}
			groupSeed := seed + int64(labelIndex(op.Label)*len(languages)+languageIndex(lang))*0x1f123bb5
			rand.New(rand.NewSource(groupSeed)).Shuffle(len(assignment), func(i, j int) {
				assignment[i], assignment[j] = assignment[j], assignment[i]
			})
			splitByOrdinal := make(map[int]string, templatesPerGroup)
			for i, ordinal := range assignment {
				switch {
				case i < trainTemplates:
					splitByOrdinal[ordinal] = "train"
				case i < trainTemplates+calibrationTemplates:
					splitByOrdinal[ordinal] = "calibration"
				default:
					splitByOrdinal[ordinal] = "test"
				}
			}
			for _, tmpl := range opTemplates {
				templates = append(templates, tmpl)
				pairs := numericPairs
				if op.Boolean {
					pairs = booleanPairs
				}
				for configIndex, pair := range pairs {
					text := fmt.Sprintf(tmpl.Text, pair.Left, pair.Right)
					if containsGoldLabel(op.Label, text) {
						return nil, nil, fmt.Errorf("template %s spells its gold label in prompt text", tmpl.ID)
					}
					configurationID := fmt.Sprintf("cfg-%02d", configIndex+1)
					id := opaqueID("example", tmpl.ID+"\x00"+configurationID)
					rows = append(rows, row{
						ID:              id,
						TemplateID:      tmpl.ID,
						ConfigurationID: configurationID,
						Language:        lang,
						Split:           splitByOrdinal[tmpl.Ordinal],
						Text:            text,
						Label:           op.Label,
					})
				}
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Label != rows[j].Label {
			return labelIndex(rows[i].Label) < labelIndex(rows[j].Label)
		}
		if rows[i].Language != rows[j].Language {
			return languageIndex(rows[i].Language) < languageIndex(rows[j].Language)
		}
		if rows[i].TemplateID != rows[j].TemplateID {
			return rows[i].TemplateID < rows[j].TemplateID
		}
		return rows[i].ConfigurationID < rows[j].ConfigurationID
	})
	return rows, templates, nil
}

func buildTemplates(op operation, lang string) []promptTemplate {
	var prefixes [4]string
	var actions [4]string
	if lang == "en" {
		prefixes = englishPrefixes
		actions = op.English
		if op.Boolean {
			prefixes = englishBooleanPrefixes
		}
	} else {
		prefixes = koreanPrefixes
		actions = op.Korean
		if op.Boolean {
			prefixes = koreanBooleanPrefixes
		}
	}
	result := make([]promptTemplate, 0, templatesPerGroup)
	for prefixIndex, prefix := range prefixes {
		for actionIndex, action := range actions {
			templateText := prefix + action
			ordinal := prefixIndex*len(actions) + actionIndex
			id := opaqueID("template", lang+"\x00"+templateText)
			result = append(result, promptTemplate{ID: id, Language: lang, Label: op.Label, Ordinal: ordinal, Text: templateText})
		}
	}
	return result
}

func encodeRows(rows []row) ([]byte, map[string][]byte, error) {
	var all bytes.Buffer
	splits := map[string]*bytes.Buffer{
		"train":       {},
		"calibration": {},
		"test":        {},
	}
	for _, item := range rows {
		line, err := json.Marshal(item)
		if err != nil {
			return nil, nil, fmt.Errorf("encode row %s: %w", item.ID, err)
		}
		all.Write(line)
		all.WriteByte('\n')
		splitBuffer, ok := splits[item.Split]
		if !ok {
			return nil, nil, fmt.Errorf("unknown split %q", item.Split)
		}
		splitBuffer.Write(line)
		splitBuffer.WriteByte('\n')
	}
	result := make(map[string][]byte, len(splits))
	for name, buffer := range splits {
		result[name] = buffer.Bytes()
	}
	return all.Bytes(), result, nil
}

func validateRows(rows []row, templates []promptTemplate) error {
	if len(rows) != 2048 {
		return fmt.Errorf("generated %d rows, expected 2048", len(rows))
	}
	if len(templates) != len(operations)*len(languages)*templatesPerGroup {
		return fmt.Errorf("generated %d templates, expected %d", len(templates), len(operations)*len(languages)*templatesPerGroup)
	}
	ids := make(map[string]struct{}, len(rows))
	texts := make(map[string]struct{}, len(rows))
	splitByTemplate := make(map[string]string, len(templates))
	rowsByTemplate := make(map[string]int, len(templates))
	for _, item := range rows {
		if _, exists := ids[item.ID]; exists {
			return fmt.Errorf("duplicate example id %q", item.ID)
		}
		ids[item.ID] = struct{}{}
		if _, exists := texts[item.Text]; exists {
			return fmt.Errorf("duplicate prompt text %q", item.Text)
		}
		texts[item.Text] = struct{}{}
		key := item.Label + "/" + item.Language + "/" + item.TemplateID
		if prior, ok := splitByTemplate[key]; ok && prior != item.Split {
			return fmt.Errorf("template group %q crosses splits", key)
		}
		splitByTemplate[key] = item.Split
		rowsByTemplate[key]++
		if containsGoldLabel(item.Label, item.Text) {
			return fmt.Errorf("prompt %q spells its gold label", item.ID)
		}
	}
	for key, count := range rowsByTemplate {
		if count != rowsPerTemplate {
			return fmt.Errorf("template group %q has %d configurations, expected %d", key, count, rowsPerTemplate)
		}
	}
	for _, label := range operationOrder {
		for _, lang := range languages {
			counts := countRows(rows, func(item row) bool { return item.Label == label && item.Language == lang })
			if counts.Train != trainTemplates*rowsPerTemplate || counts.Calibration != calibrationTemplates*rowsPerTemplate || counts.Test != testTemplates*rowsPerTemplate {
				return fmt.Errorf("wrong split counts for %s/%s: %+v", label, lang, counts)
			}
		}
	}
	return nil
}

func countRows(rows []row, predicate func(row) bool) splitCounts {
	var result splitCounts
	for _, item := range rows {
		if !predicate(item) {
			continue
		}
		result.Total++
		switch item.Split {
		case "train":
			result.Train++
		case "calibration":
			result.Calibration++
		case "test":
			result.Test++
		}
	}
	return result
}

func containsGoldLabel(label, text string) bool {
	textTokens := tokenSet(text)
	parts := strings.Split(label, "_")
	for _, part := range parts {
		if _, exists := textTokens[part]; exists {
			return true
		}
	}
	return false
}

func tokenSet(text string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, token := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		result[token] = struct{}{}
	}
	return result
}

func opaqueID(prefix, value string) string {
	sum := sha256.Sum256([]byte(value))
	return prefix + "-" + hex.EncodeToString(sum[:8])
}

func hashHex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func generatorSourceHash() string {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		return ""
	}
	return hashHex(raw)
}

func writeNewOrIdentical(path string, contents []byte) error {
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, contents) {
			return nil
		}
		return fmt.Errorf("refusing to replace existing artifact with different bytes: %s", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check output %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dataset-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary output for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(contents); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary output for %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary output for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary output for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("publish output %s: %w", path, err)
	}
	return nil
}

func labelIndex(value string) int {
	for i, label := range operationOrder {
		if value == label {
			return i
		}
	}
	return len(operationOrder)
}

func languageIndex(value string) int {
	for i, lang := range languages {
		if value == lang {
			return i
		}
	}
	return len(languages)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
