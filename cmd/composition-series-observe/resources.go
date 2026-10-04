package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type resources struct {
	WallMS  float64 `json:"wall_ms"`
	UserS   float64 `json:"user_seconds"`
	SystemS float64 `json:"system_seconds"`
	MaxRSS  int64   `json:"maximum_resident_bytes"`
}

var cpuLine = regexp.MustCompile(`([0-9.]+) real\s+([0-9.]+) user\s+([0-9.]+) sys`)
var rssLine = regexp.MustCompile(`([0-9]+)\s+maximum resident set size`)

func invoke(compiler, public, private, stem string, args ...string) ([]byte, resources, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/time", append([]string{"-l", compiler}, args...)...)
	cmd.Dir = public
	var output, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &diagnostics
	started := time.Now()
	err := cmd.Run()
	r := resources{WallMS: float64(time.Since(started).Nanoseconds()) / 1e6}
	if writeErr := os.WriteFile(filepath.Join(private, stem+".log"), diagnostics.Bytes(), 0600); writeErr != nil {
		return nil, r, writeErr
	}
	if err != nil {
		return nil, r, fmt.Errorf("%s: %w", stem, err)
	}
	clock, rss := cpuLine.FindStringSubmatch(diagnostics.String()), rssLine.FindStringSubmatch(diagnostics.String())
	if len(clock) != 4 || len(rss) != 2 {
		return nil, r, fmt.Errorf("resource observations missing")
	}
	r.UserS, err = strconv.ParseFloat(clock[2], 64)
	if err != nil {
		return nil, r, err
	}
	r.SystemS, err = strconv.ParseFloat(clock[3], 64)
	if err != nil {
		return nil, r, err
	}
	r.MaxRSS, err = strconv.ParseInt(rss[1], 10, 64)
	return output.Bytes(), r, err
}
