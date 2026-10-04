package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type result struct {
	Mode              string  `json:"mode"`
	ExitCode          int     `json:"exit_code"`
	WallMS            float64 `json:"wall_ms"`
	UserMS            float64 `json:"user_ms"`
	SystemMS          float64 `json:"system_ms"`
	OneCoreCPUPercent float64 `json:"one_core_cpu_percent"`
	PeakRSSBytes      int64   `json:"peak_rss_bytes"`
	Decision          string  `json:"decision"`
	CandidateCount    int     `json:"evaluated_candidates"`
	TrainingPassed    int     `json:"training_passed"`
	TrainingCases     int     `json:"training_cases"`
	ModelCalls        int     `json:"model_calls"`
	ProposedMask      *int    `json:"proposed_mask,omitempty"`
	PredictionNS      int64   `json:"prediction_ns"`
	GeneratedDigest   string  `json:"generated_digest"`
}

type report struct {
	Decision  string `json:"decision"`
	Generated string `json:"generated_digest"`
	Paths     struct {
		Search struct {
			Evaluated int `json:"evaluated_candidates"`
			Passed    int `json:"selected_training_passed"`
			Cases     int `json:"training_cases"`
			Selection struct {
				Calls int `json:"local_model_predictions"`
				Three *struct {
					Mask      *int  `json:"proposed_mask"`
					PredictNS int64 `json:"predict_ns"`
				} `json:"three_choice_prediction"`
			} `json:"selection"`
		} `json:"search"`
	} `json:"body_paths"`
}

func main() {
	if len(os.Args) != 7 {
		panic("usage: measure-direct mode binary plan source model output-dir")
	}
	mode, bin, plan, source, model, out := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6]
	if mode != "model" && mode != "deterministic" {
		panic("invalid mode")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		panic(err)
	}
	args := []string{"body-codegen", "--json", "--activity", "Qualified", "--path-plan", plan}
	if mode == "model" {
		args = append(args, "--path-model", model)
	}
	args = append(args, source)
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	err := cmd.Run()
	wall := time.Since(start)
	if writeErr := os.WriteFile(filepath.Join(out, "stdout.json"), stdout.Bytes(), 0600); writeErr != nil {
		panic(writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(out, "stderr.txt"), stderr.Bytes(), 0600); writeErr != nil {
		panic(writeErr)
	}
	row := result{Mode: mode, ExitCode: 0, WallMS: float64(wall.Nanoseconds()) / 1e6}
	if err != nil {
		row.ExitCode = 1
	}
	if cmd.ProcessState != nil {
		row.ExitCode = cmd.ProcessState.ExitCode()
		row.UserMS = float64(cmd.ProcessState.UserTime().Nanoseconds()) / 1e6
		row.SystemMS = float64(cmd.ProcessState.SystemTime().Nanoseconds()) / 1e6
		if row.WallMS > 0 {
			row.OneCoreCPUPercent = 100 * (row.UserMS + row.SystemMS) / row.WallMS
		}
		if usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			row.PeakRSSBytes = usage.Maxrss
		}
	}
	var wrapped struct {
		Report report `json:"report"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &wrapped); err != nil {
		panic(err)
	}
	row.Decision = wrapped.Report.Decision
	row.GeneratedDigest = wrapped.Report.Generated
	row.CandidateCount = wrapped.Report.Paths.Search.Evaluated
	row.TrainingPassed = wrapped.Report.Paths.Search.Passed
	row.TrainingCases = wrapped.Report.Paths.Search.Cases
	row.ModelCalls = wrapped.Report.Paths.Search.Selection.Calls
	if p := wrapped.Report.Paths.Search.Selection.Three; p != nil {
		row.ProposedMask = p.Mask
		row.PredictionNS = p.PredictNS
	}
	encoded, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(out, "measurement.json"), append(encoded, '\n'), 0600); err != nil {
		panic(err)
	}
	os.Stdout.Write(encoded)
	os.Stdout.Write([]byte("\n"))
	if row.ExitCode != 0 {
		os.Exit(row.ExitCode)
	}
}
