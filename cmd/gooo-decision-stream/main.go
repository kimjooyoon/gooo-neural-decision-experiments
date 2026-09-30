package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decisionstream"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}

func run(ctx context.Context, args []string, input io.ReadCloser, output io.WriteCloser, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("gooo-decision-stream", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	modelPath := flags.String("model", "", "path to strict model.json and sibling weights.bin")
	workers := flags.Int("workers", defaultWorkers(), "bounded worker count (1-8)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *modelPath == "" {
		fmt.Fprintln(diagnostics, "usage: gooo-decision-stream --model model.json [--workers 1-8]")
		return 2
	}
	if *workers < 1 || *workers > decisionstream.MaxWorkers {
		fmt.Fprintf(diagnostics, "workers must be between 1 and %d\n", decisionstream.MaxWorkers)
		return 2
	}
	model, err := decision.Load(*modelPath)
	if err != nil {
		fmt.Fprintf(diagnostics, "load model: %v\n", err)
		return 2
	}
	if err := decisionstream.Run(ctx, model, input, output, *workers); err != nil {
		fmt.Fprintf(diagnostics, "stream: %v\n", err)
		return 1
	}
	return 0
}

func defaultWorkers() int {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		return 1
	}
	if workers > decisionstream.MaxWorkers {
		return decisionstream.MaxWorkers
	}
	return workers
}
