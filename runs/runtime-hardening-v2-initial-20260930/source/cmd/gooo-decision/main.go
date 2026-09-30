package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const maxRequestBytes = 16 * 1024

type errorResponse struct {
	Schema string `json:"schema"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

func main() {
	flags := flag.NewFlagSet("gooo-decision", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	modelPath := flags.String("model", "", "path to strict model.json and sibling weights.bin")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 0 || *modelPath == "" {
		fmt.Fprintln(os.Stderr, "usage: gooo-decision --model model.json < request.json")
		os.Exit(2)
	}
	model, err := decision.Load(*modelPath)
	if err != nil {
		writeError(os.Stdout, err)
		os.Exit(2)
	}
	if err := execute(model, os.Stdin, os.Stdout); err != nil {
		writeError(os.Stdout, err)
		os.Exit(2)
	}
}

func execute(model *decision.Model, input io.Reader, output io.Writer) error {
	if model == nil {
		return errors.New("model is required")
	}
	raw, err := io.ReadAll(io.LimitReader(input, maxRequestBytes+1))
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	if len(raw) > maxRequestBytes {
		return fmt.Errorf("request exceeds %d bytes", maxRequestBytes)
	}
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return fmt.Errorf("validate request JSON: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request decision.DecisionRequest
	if err := decoder.Decode(&request); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request has trailing JSON content")
		}
		return fmt.Errorf("request has trailing content: %w", err)
	}
	var workspace decision.Workspace
	response, err := model.Decide(request, &workspace)
	if err != nil {
		return fmt.Errorf("decision rejected: %w", err)
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	return nil
}

func writeError(output io.Writer, err error) {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(errorResponse{
		Schema: decision.DecisionResponseSchema,
		Status: "rejected",
		Error:  err.Error(),
	})
}
