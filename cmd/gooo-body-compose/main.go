// gooo-body-compose fills an authored typed body plan with closed operation
// decisions. Generated Gooo and Go are returned as data, never executed here.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodydecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("gooo-body-compose", flag.ContinueOnError)
	modelPath := flags.String("model", "", "explicit model.json; absent means declared deterministic fallbacks")
	seed := flags.String("sample-seed", "", "explicit reproducible sampling seed; requires model")
	trainingPath := flags.String("training-cases", "", "optional training-only case array for finite search")
	limit := flags.Int("attempt-limit", 64, "maximum training-tested candidates, 1..128")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("body plan is read only from stdin")
	}
	data, err := io.ReadAll(io.LimitReader(input, 65537))
	if err != nil || len(data) > 65536 {
		return errors.New("cannot read plan or plan exceeds 65536 bytes")
	}
	var plan bodyplan.Plan
	if err := strictjson.Decode(data, &plan); err != nil {
		return err
	}
	if _, err := bodydecision.Validate(plan); err != nil {
		return err
	}
	var model *decision.Model
	if *modelPath != "" {
		model, err = decision.Load(*modelPath)
		if err != nil {
			return err
		}
	}
	selected, err := bodydecision.Choose(plan, model, *seed)
	if err != nil {
		return err
	}
	choices := selected.Choices
	var search *bodydecision.SearchResult
	var trainingDigest string
	if *trainingPath != "" {
		file, err := os.Open(*trainingPath)
		if err != nil {
			return err
		}
		trainingData, readErr := io.ReadAll(io.LimitReader(file, 1048577))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || len(trainingData) > 1048576 {
			return errors.New("cannot read training cases or cases exceed 1 MiB")
		}
		var cases []bodydecision.Case
		if err := strictjson.Decode(trainingData, &cases); err != nil {
			return err
		}
		result, err := bodydecision.Search(plan, choices, cases, *limit)
		if err != nil {
			return err
		}
		search, choices = &result, result.Choices
		digest := sha256.Sum256(trainingData)
		trainingDigest = hex.EncodeToString(digest[:])
	}
	program, err := bodyplan.Compile(plan, choices)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	return json.NewEncoder(output).Encode(struct {
		Schema         string                     `json:"schema"`
		PlanSHA256     string                     `json:"plan_sha256"`
		Selection      bodydecision.Selection     `json:"initial_selection"`
		Search         *bodydecision.SearchResult `json:"training_search,omitempty"`
		TrainingSHA256 string                     `json:"training_cases_sha256,omitempty"`
		Choices        map[string]string          `json:"emitted_choices"`
		GoooSource     string                     `json:"gooo_source"`
		GoSource       string                     `json:"go_source"`
	}{"gooo/typed-body-composition-result/v1", hex.EncodeToString(digest[:]), selected, search, trainingDigest, choices, program.GoooSource(), program.GoSource()})
}
