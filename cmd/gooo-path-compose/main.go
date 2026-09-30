package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

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
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *planPath == "" {
		return errors.New("supply --plan and optional --model/--seed")
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
	var model *decision.Model
	if *modelPath != "" {
		model, err = decision.LoadPath(*modelPath)
		if err != nil {
			return errors.New("path model bundle is invalid")
		}
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
