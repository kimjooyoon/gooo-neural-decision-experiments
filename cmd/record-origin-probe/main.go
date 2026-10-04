package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

type observation struct {
	Form           string          `json:"form"`
	Feature        string          `json:"feature"`
	SourceSHA      string          `json:"source_sha256"`
	InputSHA       string          `json:"input_sha256"`
	FeaturesSHA    string          `json:"features_float32_le_sha256"`
	InputBytes     int             `json:"input_bytes"`
	WallNS         int64           `json:"command_wall_ns"`
	CPUNS          int64           `json:"command_cpu_ns"`
	CPUOneCore     float64         `json:"command_cpu_percent_one_core"`
	ModelCalls     int             `json:"model_predictions"`
	CandidateTests int             `json:"candidate_tests"`
	CompilerExport json.RawMessage `json:"compiler_export"`
	RawOutput      string          `json:"raw_output,omitempty"`
	FeatureValues  [768]float32    `json:"feature_values"`
	Error          string          `json:"error,omitempty"`
}

type export struct {
	SourceSHA string `json:"original_source_sha256"`
	Context   struct {
		Status, Text string
		SHA          string `json:"sha256"`
	}
	ModelCalls     int `json:"model_predictions"`
	CandidateTests int `json:"candidate_tests"`
}

func main() {
	compiler := flag.String("compiler", "", "Gooo executable with the origin contract")
	source := flag.String("source", "", "public record-field-updates.gooo.fixture")
	output := flag.String("out", "", "new output directory")
	flag.Parse()
	if *compiler == "" || *source == "" || *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "record-origin-probe requires --compiler --source --out")
		os.Exit(2)
	}
	if err := run(*compiler, *source, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(compiler, source, output string) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	forms, err := sourceForms(string(raw))
	if err != nil {
		return err
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	var rows []observation
	for i, name := range [4]string{"current", "saved", "renamed", "case_changed"} {
		file := filepath.Join(output, name+".gooo")
		if err = os.WriteFile(file, []byte(forms[i]), 0644); err != nil {
			return err
		}
		for _, feature := range [2]string{jointdecision.RecordSharedFeatureVersion, jointdecision.RecordOriginSharedFeatureVersion} {
			row, e := observe(compiler, file, name, feature)
			if e != nil {
				row.Error = e.Error()
			}
			rows = append(rows, row)
		}
	}
	checks := probeChecks(rows)
	report := struct {
		Schema       string          `json:"schema"`
		Scope        string          `json:"scope"`
		Checks       map[string]bool `json:"checks"`
		Observations []observation   `json:"observations"`
	}{"gooo/record-origin-representation-probe/v1",
		"four authored source forms; two explicit feature contracts; zero predictions, finite executions or training; CPU/wall is a child-command one-core ratio, not whole-host utilization; no RAM or GPU measurement", checks, rows}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(output, "observations.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	for name, passed := range checks {
		if !passed {
			return fmt.Errorf("representation check failed: %s; original observations retained", name)
		}
	}
	fmt.Println("4 source forms, 8 context exports, 6 representation checks; zero predictions and executions")
	return nil
}

func sourceForms(source string) ([4]string, error) {
	var forms [4]string
	start := strings.Index(source, "computes `")
	end := strings.Index(source, "` assembling {")
	if start < 0 || end <= start {
		return forms, fmt.Errorf("expected public record source assembly fixture")
	}
	start += len("computes `")
	body := "let copy = input0\nlet saved = copy\nlet old = copy.state\ncopy.title = \"draft\"\ncopy.state = \"wait\"\ncopy.reason = old\nreturn Candidate{title: copy.title, state: copy.state, reason: copy.reason + saved.reason + old}"
	forms[0] = source[:start] + body + source[end:]
	needle := `copy.title + \":\" + copy.state`
	if strings.Count(forms[0], needle) != 1 || strings.Count(forms[0], "한글") < 1 {
		return forms, fmt.Errorf("expected receiver alternative and public finite input")
	}
	forms[1] = strings.Replace(forms[0], needle, `saved.title + \":\" + saved.state`, 1)
	forms[2] = strings.ReplaceAll(forms[0], "copy", "working")
	forms[3] = strings.Replace(forms[0], "한글", "변경", 1)
	return forms, nil
}

func observe(compiler, source, form, feature string) (observation, error) {
	row := observation{Form: form, Feature: feature}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, compiler, "body-context", "--value-flow", "--activity", "Select", "--feature-version", feature, source)
	started := time.Now()
	raw, err := cmd.Output()
	row.WallNS = time.Since(started).Nanoseconds()
	if json.Valid(raw) {
		row.CompilerExport = raw
	} else {
		row.RawOutput = string(raw)
	}
	if err != nil {
		return row, fmt.Errorf("%s/%s context export: %w", form, feature, err)
	}
	row.CPUNS = (cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()).Nanoseconds()
	row.CPUOneCore = 100 * float64(row.CPUNS) / float64(row.WallNS)
	var result export
	if err = json.Unmarshal(raw, &result); err != nil || result.Context.Status != "ENCODED" {
		return row, fmt.Errorf("complete encoded compiler context required: %v", err)
	}
	row.SourceSHA, row.InputSHA, row.InputBytes = result.SourceSHA, result.Context.SHA, len(result.Context.Text)
	row.ModelCalls, row.CandidateTests, row.CompilerExport = result.ModelCalls, result.CandidateTests, raw
	if feature == jointdecision.RecordSharedFeatureVersion {
		err = jointdecision.FeaturesIntoRecordThree(result.Context.Text, &row.FeatureValues)
	} else {
		err = jointdecision.FeaturesIntoRecordOriginThree(result.Context.Text, &row.FeatureValues)
	}
	var bytes [768 * 4]byte
	for i, value := range row.FeatureValues {
		binary.LittleEndian.PutUint32(bytes[4*i:], math.Float32bits(value))
	}
	row.FeaturesSHA = fmt.Sprintf("%x", sha256.Sum256(bytes[:]))
	return row, err
}

func probeChecks(rows []observation) map[string]bool {
	if len(rows) != 8 {
		return map[string]bool{"eight_exports_required": false}
	}
	zero := len(rows) == 8
	for _, row := range rows {
		zero = zero && row.Error == "" && row.ModelCalls == 0 && row.CandidateTests == 0
	}
	return map[string]bool{
		"old_receiver_collision_retained":           rows[0].FeatureValues == rows[2].FeatureValues,
		"new_receiver_times_distinguished":          rows[1].FeatureValues != rows[3].FeatureValues,
		"new_rename_invariant":                      rows[1].FeatureValues == rows[5].FeatureValues,
		"new_case_input_invariant":                  rows[1].FeatureValues == rows[7].FeatureValues,
		"source_and_text_changes_retained":          rows[0].SourceSHA != rows[2].SourceSHA && rows[0].InputSHA != rows[2].InputSHA,
		"zero_predictions_and_candidate_executions": zero,
	}
}
