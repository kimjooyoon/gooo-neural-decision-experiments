package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type resource struct {
	Activity   string  `json:"activity"`
	Form       string  `json:"form"`
	Mode       string  `json:"mode"`
	RealS      float64 `json:"real_seconds"`
	UserS      float64 `json:"user_seconds"`
	SystemS    float64 `json:"system_seconds"`
	CPUPercent float64 `json:"command_cpu_percent_one_core"`
	MaxRSS     int64   `json:"maximum_rss_bytes"`
	Footprint  int64   `json:"peak_memory_footprint_bytes"`
}

var (
	timeTotals = regexp.MustCompile(`([0-9.]+) real\s+([0-9.]+) user\s+([0-9.]+) sys`)
	maxRSS     = regexp.MustCompile(`([0-9]+)\s+maximum resident set size`)
	footprint  = regexp.MustCompile(`([0-9]+)\s+peak memory footprint`)
)

func writeResources(private, public string) error {
	var rows []resource
	for _, activity := range []string{"Qualified", "Clamp"} {
		for _, form := range []string{"source", "external"} {
			for _, mode := range []string{"model", "deterministic"} {
				stem := activity + "-" + form + "-" + mode
				raw, err := os.ReadFile(filepath.Join(private, stem+".log"))
				if err != nil {
					return err
				}
				row, err := parseResource(string(raw))
				if err != nil {
					return err
				}
				row.Activity, row.Form, row.Mode = activity, form, mode
				rows = append(rows, row)
			}
		}
	}
	return writeJSON(filepath.Join(public, "resources.json"), struct {
		Schema string     `json:"schema"`
		Scope  string     `json:"scope"`
		Rows   []resource `json:"rows"`
	}{"gooo/source-assembly-command-resources/v1",
		"macOS time -l totals for four-request CLI commands, including native compiler/children and loading; " +
			"CPU=(user+system)/real, one core; time output is rounded to 0.01s; " +
			"RSS/footprint are command resource observations, not model-only RAM; whole-host CPU is unobserved", rows})
}

func parseResource(text string) (resource, error) {
	var row resource
	totals := timeTotals.FindStringSubmatch(text)
	rss, memory := maxRSS.FindStringSubmatch(text), footprint.FindStringSubmatch(text)
	if len(totals) != 4 || len(rss) != 2 || len(memory) != 2 {
		return row, fmt.Errorf("missing command resource fields")
	}
	for i, target := range []*float64{&row.RealS, &row.UserS, &row.SystemS} {
		value, err := strconv.ParseFloat(totals[i+1], 64)
		if err != nil {
			return row, err
		}
		*target = value
	}
	var err error
	row.MaxRSS, err = strconv.ParseInt(rss[1], 10, 64)
	if err != nil {
		return row, err
	}
	row.Footprint, err = strconv.ParseInt(memory[1], 10, 64)
	if err != nil || row.RealS <= 0 {
		return row, fmt.Errorf("invalid command resource values")
	}
	row.CPUPercent = (row.UserS + row.SystemS) / row.RealS * 100
	return row, nil
}
