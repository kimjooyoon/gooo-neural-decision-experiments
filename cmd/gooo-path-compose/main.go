package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("gooo-path-compose", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	planPath := flags.String("plan", "", "typed structural plan JSON")
	modelPath := flags.String("model", "", "optional path-model metadata")
	seed := flags.String("seed", "", "optional reproducible probability sampling seed")
	testsPath := flags.String("tests", "", "optional bounded finite TDD test document")
	maxAttempts := flags.Int("max-attempts", 16, "TDD candidate budget, 1..64")
	timeout := flags.Duration("timeout", 2*time.Second, "TDD deadline, 1ms..30s")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *planPath == "" {
		return errors.New("supply --plan and optional --model/--seed/--tests")
	}
	info, err := os.Lstat(*planPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 128<<10 {
		return errors.New("path plan must be a bounded regular file")
	}
	raw, err := os.ReadFile(*planPath)
	if err != nil {
		return errors.New("path plan could not be read")
	}
	var plan pathplan.Plan
	if err := strictjson.Decode(raw, &plan); err != nil {
		return errors.New("path plan JSON is invalid")
	}
	if _, err := pathplan.Validate(plan); err != nil {
		return fmt.Errorf("path plan: %w", err)
	}
	var tests []pathplan.TestCase
	if *testsPath != "" {
		if *maxAttempts < 1 || *maxAttempts > 64 || *timeout < time.Millisecond || *timeout > 30*time.Second {
			return errors.New("TDD budget or deadline is invalid")
		}
		info, err := os.Lstat(*testsPath)
		if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 32<<10 {
			return errors.New("tests must be a bounded regular file")
		}
		raw, err := os.ReadFile(*testsPath)
		if err != nil {
			return errors.New("tests could not be read")
		}
		var document struct {
			Schema string              `json:"schema"`
			Cases  []pathplan.TestCase `json:"cases"`
		}
		if err := strictjson.Decode(raw, &document); err != nil || document.Schema != "gooo/typed-path-finite-tests/v1" || len(document.Cases) == 0 || len(document.Cases) > 128 {
			return errors.New("finite tests contract is invalid")
		}
		tests = document.Cases
	}
	var model *decision.Model
	if *modelPath != "" {
		model, err = decision.LoadPath(*modelPath)
		if err != nil {
			return errors.New("path model bundle is invalid")
		}
	}
	if *testsPath != "" {
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		result, program, err := pathplan.Search(ctx, plan, model, tests, *maxAttempts, *seed)
		if err != nil {
			return err
		}
		return json.NewEncoder(output).Encode(struct {
			Schema     string                `json:"schema"`
			Search     pathplan.SearchResult `json:"search"`
			GoooSource string                `json:"gooo_source"`
			GoSource   string                `json:"go_source"`
		}{"gooo/typed-path-tdd-compose-result/v1", result, program.GoooSource(), program.GoSource()})
	}
	selection, program, err := pathplan.Choose(plan, model, *seed)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(struct {
		Schema     string             `json:"schema"`
		Selection  pathplan.Selection `json:"selection"`
		GoooSource string             `json:"gooo_source"`
		GoSource   string             `json:"go_source"`
	}{"gooo/typed-path-compose-result/v1", selection, program.GoooSource(), program.GoSource()})
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "gooo-path-compose:", err)
		os.Exit(1)
	}
}
