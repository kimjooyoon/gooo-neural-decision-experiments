package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const preparedRun = "runs/prepared-native-conditional-20261001"
const conditionalCohort = "studies/conditional-paths-v1/cohort"
const pairedRunner = "61f12318262e69520ab4af6663afb95004c494dc"
const oldCompiler = "9b8b900e32384b5397fb3d08dc545e22cc7530da"
const newCompiler = "e155148f5ce114ee4c55e720a0e90cc34cc458ce"

func preparedSources() map[string]string {
	files := directSources()
	for _, name := range []string{"report.json", "preexecution.json", "independent-go-tests.jsonl"} {
		files["native-prepared/"+name] = preparedRun + "/" + name
	}
	for _, name := range []string{"conditional-assignment.gooo.fixture", "en-budget-8.json", "en-budget-64.json", "ko-budget-8.json", "ko-budget-64.json", "structural-oracle.json", "holdout-cases.json", "manifest.json"} {
		files["conditional-cohort/"+name] = conditionalCohort + "/" + name
	}
	return files
}

type pairedProcess struct {
	Wall   int64   `json:"wall_ns"`
	User   int64   `json:"user_cpu_ns"`
	System int64   `json:"system_cpu_ns"`
	RSS    int64   `json:"lifetime_peak_rss_bytes"`
	Known  bool    `json:"rss_known"`
	CPU    float64 `json:"one_core_cpu_percent"`
}
type pairedCell struct {
	ID         string                `json:"id"`
	Pair       string                `json:"pair"`
	Version    string                `json:"version"`
	Arm        string                `json:"arm"`
	Language   string                `json:"language"`
	Budget     int                   `json:"budget"`
	Repeat     int                   `json:"repetition"`
	Position   int                   `json:"within_pair_position"`
	SourceSHA  string                `json:"generated_go_sha256"`
	ReplySHA   string                `json:"native_stdout_sha256"`
	Search     pathplan.SearchResult `json:"search"`
	Functional float64               `json:"finite_contract_percent"`
	Matched    int                   `json:"authored_structural_labels_matched"`
	Labels     int                   `json:"authored_structural_labels_total"`
	Stage      float64               `json:"native_total_stage_ms"`
	Process    pairedProcess         `json:"process"`
	Passed     int                   `json:"independent_arithmetic_passed"`
	Total      int                   `json:"independent_arithmetic_total"`
}
type pairedSummary struct {
	Version string  `json:"version"`
	Stage   float64 `json:"median_native_total_stage_ms"`
	Wall    float64 `json:"median_process_wall_ms"`
	CPU     float64 `json:"median_one_core_cpu_percent"`
	RSS     float64 `json:"median_lifetime_child_rss_bytes"`
	Passed  int     `json:"independent_arithmetic_passed"`
	Total   int     `json:"independent_arithmetic_total"`
	Matched int     `json:"selected_authored_structure_labels_matched"`
	Labels  int     `json:"selected_authored_structure_labels_total"`
}
type pairedReport struct {
	Schema       string          `json:"schema"`
	Runner       string          `json:"runner_revision"`
	Calls        int             `json:"native_calls"`
	Predictions  int             `json:"model_predictions"`
	External     int             `json:"external_calls"`
	Steps        int             `json:"optimizer_steps"`
	Tests        int             `json:"exact_go_arena_parity_tests_passed"`
	Observations int             `json:"arithmetic_observations"`
	Same         bool            `json:"all_80_pairs_same_source_and_search"`
	Saving       float64         `json:"median_paired_native_stage_saving_ms"`
	Summaries    []pairedSummary `json:"summaries"`
	Cells        []pairedCell    `json:"cells"`
}

func pairedMedian(values []float64) float64 {
	values = append([]float64(nil), values...)
	sort.Float64s(values)
	if len(values) == 0 {
		return 0
	}
	return (values[(len(values)-1)/2] + values[len(values)/2]) / 2
}
func equalNumber(a, b float64) bool {
	return !math.IsNaN(a) && !math.IsInf(a, 0) && math.Abs(a-b) <= 1e-9
}
func semanticSearch(value pathplan.SearchResult) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var owned pathplan.SearchResult
	if err = json.Unmarshal(raw, &owned); err != nil {
		return nil, err
	}
	for i := range owned.Selection.Receipts {
		owned.Selection.Receipts[i].PredictNS = 0
	}
	return json.Marshal(owned)
}
func auditPreparedRepository() error {
	return auditPreparedDirectory(preparedRun, conditionalCohort, preparedRun)
}

// Compact HF evidence carries the full report, preexecution bindings, cohort and
// independent Go events. GitHub retains the source/receipt/process captures;
// assembly additionally checks every raw capture. Verification never infers.
func auditPreparedDirectory(root, cohort, captures string) error {
	var report pairedReport
	raw, err := read(filepath.Join(root, "report.json"))
	if err != nil || json.Unmarshal(raw, &report) != nil || report.Schema != "gooo/prepared-native-comparison/v1" || report.Runner != pairedRunner || report.Calls != 160 || report.Predictions != 720 || report.External != 0 || report.Steps != 0 || report.Tests != 160 || report.Observations != 1440 || !report.Same || len(report.Cells) != 160 || len(report.Summaries) != 2 {
		return errors.New("invalid prepared comparison totals")
	}
	raw, err = read(filepath.Join(root, "preexecution.json"))
	var pre struct {
		Schema      string   `json:"schema"`
		Runner      string   `json:"runner_revision"`
		Old         string   `json:"baseline_revision"`
		New         string   `json:"prepared_revision"`
		OldSHA      string   `json:"baseline_binary_sha256"`
		NewSHA      string   `json:"prepared_binary_sha256"`
		GoSHA       string   `json:"go_binary_sha256"`
		CohortSHA   string   `json:"cohort_manifest_sha256"`
		Calls       int      `json:"planned_native_calls"`
		Predictions int      `json:"planned_model_predictions"`
		Steps       int      `json:"optimizer_steps"`
		External    int      `json:"external_calls"`
		Repeats     int      `json:"repetitions"`
		OldFirst    int      `json:"balanced_old_first_pairs"`
		NewFirst    int      `json:"balanced_new_first_pairs"`
		Order       []string `json:"execution_order"`
		Models      map[string]struct {
			Metadata string `json:"metadata_sha256"`
			Weights  string `json:"weights_sha256"`
		} `json:"models"`
	}
	if err != nil || json.Unmarshal(raw, &pre) != nil || pre.Schema != "gooo/prepared-native-comparison-preexecution/v1" || pre.Runner != pairedRunner || pre.Old != oldCompiler || pre.New != newCompiler || pre.OldSHA != "2545895533e9a1d4ab1e64c1b71f73595b9087c9a0d99c2fc0c78a841c112790" || pre.NewSHA != "01a3be2815f260dedbc4353ae477cef114ad5521cc5524fb32f47f036eecc910" || pre.GoSHA != "132b69336a1f809932a8a20b0201dbbb980e86e3a323ae32e893639d83d71598" || pre.Calls != 160 || pre.Predictions != 720 || pre.Steps != 0 || pre.External != 0 || pre.Repeats != 5 || pre.OldFirst != 40 || pre.NewFirst != 40 || len(pre.Order) != 160 || len(pre.Models) != 3 {
		return errors.New("invalid prepared preexecution bindings")
	}
	modelRoot := filepath.Join(filepath.Dir(root), "positioned-random", "models")
	if captures != "" {
		modelRoot = "runs/typed-path-positioned-random-20261001/models"
	}
	for _, arm := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		pin, exists := pre.Models[arm]
		metadata, e1 := read(filepath.Join(modelRoot, arm, "model.json"))
		weights, e2 := read(filepath.Join(modelRoot, arm, "weights.bin"))
		if !exists || e1 != nil || e2 != nil || digest(metadata) != pin.Metadata || digest(weights) != pin.Weights {
			return errors.New("paired preexecution model tensors differ")
		}
	}
	raw, err = read(filepath.Join(cohort, "manifest.json"))
	var manifest struct {
		Files        map[string]string `json:"fixture_files_sha256"`
		Decisions    int               `json:"decisions_per_plan"`
		Combinations int               `json:"declared_combinations"`
	}
	if err != nil || digest(raw) != pre.CohortSHA || json.Unmarshal(raw, &manifest) != nil || len(manifest.Files) != 7 || manifest.Decisions != 6 || manifest.Combinations != 64 {
		return errors.New("invalid prepared cohort")
	}
	for name, pin := range manifest.Files {
		if filepath.Base(name) != name {
			return errors.New("invalid cohort path")
		}
		data, err := read(filepath.Join(cohort, name))
		if err != nil || digest(data) != pin {
			return errors.New("prepared cohort hash mismatch")
		}
	}
	raw, err = read(filepath.Join(cohort, "structural-oracle.json"))
	var oracle map[string]string
	if err != nil || json.Unmarshal(raw, &oracle) != nil || len(oracle) != 6 {
		return errors.New("invalid structural oracle")
	}
	raw, err = read(filepath.Join(cohort, "holdout-cases.json"))
	var holdout []pathplan.TestCase
	if err != nil || json.Unmarshal(raw, &holdout) != nil || len(holdout) != 9 {
		return errors.New("invalid independent arithmetic cases")
	}
	inputs := map[int64]int64{}
	for _, item := range holdout {
		expected := pairedGold(item.Input)
		if _, exists := inputs[item.Input]; exists || expected != item.Expected {
			return errors.New("invalid arithmetic oracle")
		}
		inputs[item.Input] = expected
	}
	byID := map[string]pairedCell{}
	actuals := map[string]map[int64]int64{}
	pairs := map[string]pairedCell{}
	counts := map[string]int{}
	sums := map[string]int{}
	metrics := map[string]map[string][]float64{}
	predictions, oldFirst, newFirst, positive := 0, 0, 0, 0
	var savings []float64
	for index, row := range report.Cells {
		pair := fmt.Sprintf("r%d-%s-%s-b%d", row.Repeat, row.Language, row.Arm, row.Budget)
		if row.Repeat < 0 || row.Repeat >= 5 || row.Language != "en" && row.Language != "ko" || row.Arm != "offline" && row.Arm != "fp32" && row.Arm != "ptq_ternary" && row.Arm != "qat_ternary" || row.Budget != 8 && row.Budget != 64 || row.Version != "baseline" && row.Version != "prepared" || row.Pair != pair || row.ID != pair+"-"+row.Version || row.ID != pre.Order[index] || row.Position != index%2 {
			return errors.New("invalid balanced cell identity")
		}
		if _, exists := byID[row.ID]; exists {
			return errors.New("duplicate paired cell")
		}
		byID[row.ID] = row
		search := row.Search
		if search.DeclaredCombinations != 64 || len(search.Attempts) > row.Budget || search.Unattempted != 64-len(search.Attempts) || search.TrainingTotal != 7 || search.TypeRejected != 0 || len(search.Selection.Choices) != 6 || len(search.Selection.Receipts) != 6 || search.Selection.ExternalCalls != 0 || !search.Selection.ExternalCallsKnown || row.Labels != 6 || row.Total != 9 || row.Stage <= 0 || !equalNumber(row.Functional, 100*float64(search.SelectedTrainingPassed)/7) {
			return errors.New("invalid prepared search accounting")
		}
		calls := 0
		if row.Arm != "offline" {
			calls = 6
			pin, exists := pre.Models[row.Arm]
			if !exists || search.Selection.ModelVariant != row.Arm || search.Selection.MetadataSHA256 != pin.Metadata || search.Selection.WeightsSHA256 != pin.Weights {
				return errors.New("prepared model provenance mismatch")
			}
		}
		if search.Selection.ModelCalls != calls {
			return errors.New("prepared prediction count mismatch")
		}
		predictions += calls
		for _, receipt := range search.Selection.Receipts {
			if calls > 0 && receipt.PredictNS <= 0 || calls == 0 && receipt.PredictNS != 0 {
				return errors.New("lost or unexpected prediction timing")
			}
			if receipt.PredictNS > 0 {
				positive++
			}
		}
		matched := 0
		for key, want := range oracle {
			if search.Selection.Choices[key] == want {
				matched++
			}
		}
		if row.Matched != matched {
			return errors.New("structural agreement differs")
		}
		documentRaw, err := read(filepath.Join(cohort, fmt.Sprintf("%s-budget-%d.json", row.Language, row.Budget)))
		var document struct {
			Plan  pathplan.Plan       `json:"path_plan"`
			Cases []pathplan.TestCase `json:"test_cases"`
		}
		if err != nil || json.Unmarshal(documentRaw, &document) != nil || len(document.Cases) != 7 {
			return errors.New("invalid paired finite document")
		}
		program, err := pathplan.Compile(document.Plan, search.Selection.Choices)
		if err != nil {
			return errors.New("selected paired arena is invalid")
		}
		trainingPassed := 0
		for _, item := range document.Cases {
			value, err := program.Evaluate(item.Input)
			if err != nil {
				return err
			}
			if value.Int == item.Expected {
				trainingPassed++
			}
		}
		if trainingPassed != search.SelectedTrainingPassed {
			return errors.New("selected finite completeness differs")
		}
		actuals[row.ID] = map[int64]int64{}
		for input := range inputs {
			value, err := program.Evaluate(input)
			if err != nil {
				return err
			}
			actuals[row.ID][input] = value.Int
		}
		if row.Budget == 64 && (row.Passed != 9 || search.SelectedTrainingPassed != 7) {
			return errors.New("full-budget outcome differs")
		}
		if search.Status == "PARTIAL" {
			sums[row.Version+"/partial"]++
		}
		sums[row.Version+"/passed"] += row.Passed
		sums[row.Version+"/matched"] += matched
		counts[row.Version]++
		proc := row.Process
		if proc.Wall <= 0 || proc.User < 0 || proc.System < 0 || !proc.Known || proc.RSS <= 0 || !equalNumber(proc.CPU, 100*float64(proc.User+proc.System)/float64(proc.Wall)) {
			return errors.New("invalid child cost metrics")
		}
		if metrics[row.Version] == nil {
			metrics[row.Version] = map[string][]float64{}
		}
		for name, value := range map[string]float64{"stage": row.Stage, "wall": float64(proc.Wall) / 1e6, "cpu": proc.CPU, "rss": float64(proc.RSS)} {
			metrics[row.Version][name] = append(metrics[row.Version][name], value)
		}
		if first, exists := pairs[pair]; exists {
			if index%2 != 1 || report.Cells[index-1].Pair != pair || first.Version == row.Version || first.SourceSHA != row.SourceSHA || first.Functional != row.Functional {
				return errors.New("paired source or position differs")
			}
			before, e1 := semanticSearch(first.Search)
			after, e2 := semanticSearch(row.Search)
			if e1 != nil || e2 != nil || !bytes.Equal(before, after) {
				return errors.New("paired search semantics differ")
			}
			old, new := first, row
			if old.Version != "baseline" {
				old, new = new, old
			}
			savings = append(savings, old.Stage-new.Stage)
		} else {
			if index%2 != 0 {
				return errors.New("pair does not start first")
			}
			pairs[pair] = row
			if row.Version == "baseline" {
				oldFirst++
			} else {
				newFirst++
			}
		}
		if captures != "" {
			if err := auditPairedCapture(captures, row); err != nil {
				return err
			}
		}
	}
	if len(pairs) != 80 || predictions != 720 || positive != 720 || oldFirst != 40 || newFirst != 40 || !equalNumber(report.Saving, pairedMedian(savings)) {
		return errors.New("paired comparison arithmetic differs")
	}
	seenSummary := map[string]bool{}
	for _, summary := range report.Summaries {
		version := summary.Version
		if seenSummary[version] || counts[version] != 80 || sums[version+"/passed"] != 645 || sums[version+"/matched"] != 390 || sums[version+"/partial"] != 25 || summary.Passed != 645 || summary.Total != 720 || summary.Matched != 390 || summary.Labels != 480 {
			return errors.New("paired summary accounting differs")
		}
		seenSummary[version] = true
		for name, value := range map[string]float64{"stage": summary.Stage, "wall": summary.Wall, "cpu": summary.CPU, "rss": summary.RSS} {
			if !equalNumber(value, pairedMedian(metrics[version][name])) {
				return errors.New("paired median differs")
			}
		}
	}
	return auditPairedGo(root, byID, inputs, actuals)
}
func pairedGold(input int64) int64 {
	if input > 0 && input <= 5 {
		return 2*input - 20
	}
	return 2*input + 10
}

func auditPairedCapture(root string, row pairedCell) error {
	dir := filepath.Join(root, row.ID)
	raw, err := read(filepath.Join(dir, "native-stdout.json"))
	if err != nil || digest(raw) != row.ReplySHA {
		return errors.New("paired raw receipt hash mismatch")
	}
	var reply struct {
		Source string `json:"source"`
		Report struct {
			Decision  string `json:"decision"`
			Compiler  string `json:"compiler_source_sha"`
			Typecheck bool   `json:"typecheck_passed"`
			Replay    bool   `json:"deterministic_replay"`
			Writes    int    `json:"repository_writes"`
			Paths     struct {
				Matched    bool                  `json:"source_base_matched"`
				Search     pathplan.SearchResult `json:"search"`
				Functional float64               `json:"finite_functional_completeness_percent"`
				Timing     struct {
					Total   float64 `json:"total_ms"`
					Prepare float64 `json:"plan_prepare_ms"`
				} `json:"timing"`
			} `json:"body_paths"`
		} `json:"report"`
	}
	expected := oldCompiler
	if row.Version == "prepared" {
		expected = newCompiler
	}
	if json.Unmarshal(raw, &reply) != nil || reply.Report.Compiler != expected || reply.Report.Decision != "PASS" || !reply.Report.Typecheck || !reply.Report.Replay || reply.Report.Writes != 0 || !reply.Report.Paths.Matched || digest([]byte(reply.Source)) != row.SourceSHA || reply.Report.Paths.Timing.Total != row.Stage || reply.Report.Paths.Functional != row.Functional || row.Version == "prepared" && reply.Report.Paths.Timing.Prepare <= 0 {
		return errors.New("paired captured native contract differs")
	}
	a, _ := json.Marshal(row.Search)
	b, _ := json.Marshal(reply.Report.Paths.Search)
	if !bytes.Equal(a, b) {
		return errors.New("paired raw search differs")
	}
	raw, err = read(filepath.Join(dir, "generated.go.txt"))
	if err != nil || digest(raw) != row.SourceSHA || !bytes.Equal(raw, []byte(reply.Source)) {
		return errors.New("paired generated source differs")
	}
	raw, err = read(filepath.Join(dir, "process.json"))
	var proc pairedProcess
	if err != nil || json.Unmarshal(raw, &proc) != nil || proc != row.Process {
		return errors.New("paired raw cost differs")
	}
	return nil
}
func auditPairedGo(root string, cells map[string]pairedCell, inputs map[int64]int64, actuals map[string]map[int64]int64) error {
	raw, err := read(filepath.Join(root, "independent-go-tests.jsonl"))
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 8192), 1<<20)
	observed := map[string]map[int64]bool{}
	passed := map[string]int{}
	packages := map[string]bool{}
	tests := map[string]bool{}
	count := 0
	for scanner.Scan() {
		var event struct{ Action, Test, Output, Package string }
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			return errors.New("invalid paired Go event")
		}
		id := strings.TrimPrefix(event.Package, "gooo.prepared.native/")
		if _, exists := cells[id]; !exists {
			return errors.New("unbound paired Go package")
		}
		if event.Action == "fail" {
			return errors.New("failed paired Go event")
		}
		if event.Action == "pass" {
			if event.Test == "" {
				if packages[id] {
					return errors.New("duplicate package pass")
				}
				packages[id] = true
			} else if event.Test == "TestExactNativeObservation" {
				if tests[id] {
					return errors.New("duplicate test pass")
				}
				tests[id] = true
			}
		}
		index := strings.Index(event.Output, "GOOO_OBSERVATION ")
		if index < 0 {
			continue
		}
		var item struct {
			Cell     string `json:"cell"`
			Input    int64  `json:"input"`
			Expected int64  `json:"expected"`
			Actual   int64  `json:"actual"`
			Passed   bool   `json:"passed"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(event.Output[index+len("GOOO_OBSERVATION "):])), &item) != nil || item.Cell != id || event.Test != "TestExactNativeObservation" {
			return errors.New("invalid paired arithmetic event")
		}
		want, exists := inputs[item.Input]
		if !exists || item.Expected != want || item.Expected != pairedGold(item.Input) || item.Passed != (item.Actual == want) || actuals[id][item.Input] != item.Actual {
			return errors.New("paired arithmetic oracle differs")
		}
		if observed[id] == nil {
			observed[id] = map[int64]bool{}
		}
		if observed[id][item.Input] {
			return errors.New("duplicate paired arithmetic input")
		}
		observed[id][item.Input] = true
		count++
		if item.Passed {
			passed[id]++
		}
	}
	if scanner.Err() != nil || count != 1440 || len(packages) != 160 || len(tests) != 160 {
		return errors.New("paired Go event totals differ")
	}
	for id, row := range cells {
		if len(observed[id]) != 9 || passed[id] != row.Passed {
			return errors.New("paired per-cell arithmetic differs")
		}
	}
	return nil
}
